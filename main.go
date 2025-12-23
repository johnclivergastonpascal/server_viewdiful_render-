package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time" // Necesario para que el random cambie siempre

	"github.com/gorilla/mux"
)

// ... (Tus structs Segment y VideoInfo se mantienen igual)

type Segment struct {
	Parte    int `json:"parte"`
	Start    int `json:"inicio_seg"`
	Duration int `json:"duracion_seg"`
}

type VideoInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"titulo"`
	Duration  int       `json:"duracion_total_seg"`
	Segments  []Segment `json:"partes"`
	Thumbnail string    `json:"thumbnail"`
}

var videos []VideoInfo

func loadJSON() {
	file, err := os.ReadFile("videos.json")
	if err != nil {
		log.Fatalf("Error leyendo videos.json: %v", err)
	}

	err = json.Unmarshal(file, &videos)
	if err != nil {
		log.Fatalf("Error parseando videos.json: %v", err)
	}

	log.Println("videos.json cargado con éxito. Total videos:", len(videos))
}

// ---------------------------------------------------------
// NUEVA FUNCIÓN: Obtener videos mezclados
// ---------------------------------------------------------
func getShuffledVideos() []VideoInfo {
	// Creamos una copia para no alterar el orden original en memoria permanentemente
	shuffled := make([]VideoInfo, len(videos))
	copy(shuffled, videos)

	// Mezclamos la lista usando el tiempo actual como semilla
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	
	return shuffled
}

// ----------------------
// Endpoint: Videos paginados (AHORA ALEATORIOS)
// ----------------------
func getPaginatedVideos(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, errL := strconv.Atoi(limitStr)

	if page < 0 { page = 0 }
	if errL != nil || limit <= 0 { limit = 20 } // Aumenté el límite por defecto

	// IMPORTANTE: Mezclamos antes de paginar
	allShuffled := getShuffledVideos()

	startIdx := page * limit
	endIdx := startIdx + limit

	if startIdx >= len(allShuffled) {
		sendJSON(w, []VideoInfo{})
		return
	}

	if endIdx > len(allShuffled) {
		endIdx = len(allShuffled)
	}

	paginated := allShuffled[startIdx:endIdx]
	sendJSON(w, paginated)
}

// ----------------------
// Endpoint: Random (Ahora devuelve una lista grande)
// ----------------------
func getRandom(w http.ResponseWriter, r *http.Request) {
	if len(videos) == 0 {
		http.Error(w, "No hay videos", http.StatusInternalServerError)
		return
	}

	// Devuelve 30 videos al azar de toda la lista
	allShuffled := getShuffledVideos()
	max := 30
	if len(allShuffled) < max {
		max = len(allShuffled)
	}

	sendJSON(w, allShuffled[:max])
}

// ... (findVideoByID, getSingleVideo, searchVideos, getSitemap se mantienen igual)

func findVideoByID(id string) *VideoInfo {
	for _, v := range videos {
		if strings.ToLower(v.ID) == strings.ToLower(id) {
			return &v
		}
	}
	return nil
}

func getSingleVideo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	video := findVideoByID(id)
	if video == nil {
		http.Error(w, "Video no encontrado", http.StatusNotFound)
		return
	}
	sendJSON(w, video)
}

func searchVideos(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	var results []VideoInfo
	for _, v := range videos {
		if q != "" && strings.Contains(strings.ToLower(v.Title), q) {
			results = append(results, v)
		}
	}
	sendJSON(w, results)
}

func getSitemap(w http.ResponseWriter, r *http.Request) {
	baseURL := "https://viewdiful.vercel.app"
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	xml := `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
	for _, v := range videos {
		xml += `<url><loc>` + baseURL + `/video/` + v.ID + `</loc><priority>0.8</priority></url>`
	}
	xml += `</urlset>`
	w.Write([]byte(xml))
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Evitar que el navegador guarde el orden anterior
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(data)
}

func main() {
	loadJSON()
	r := mux.NewRouter()

	r.HandleFunc("/video/{id}", getSingleVideo).Methods("GET")
	r.HandleFunc("/videos", getPaginatedVideos).Methods("GET") // Devuelve lista mezclada
	r.HandleFunc("/search", searchVideos).Methods("GET")
	r.HandleFunc("/random", getRandom).Methods("GET") // Devuelve pool de videos aleatorios
	r.HandleFunc("/sitemap.xml", getSitemap).Methods("GET")

	port := "8080"
	log.Println("Servidor iniciado en puerto " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}