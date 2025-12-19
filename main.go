package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

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

// ----------------------
// Cargar JSON a memoria
// ----------------------
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

// --- Buscar video por ID ---
func findVideoByID(id string) *VideoInfo {
	for _, v := range videos {
		if strings.EqualFold(v.ID, id) {
			return &v
		}
	}
	return nil
}

// ----------------------
// Endpoint: Video único
// ----------------------
func getSingleVideo(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	video := findVideoByID(id)
	if video == nil {
		http.Error(w, "Video no encontrado", http.StatusNotFound)
		return
	}

	sendJSON(w, video)
}

// ----------------------
// Endpoint: Videos PAGINADOS + RANDOM
// ----------------------
func getPaginatedVideos(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 0 {
		page = 0
	}
	if limit <= 0 {
		limit = 10
	}

	// 👉 COPIA para no romper el orden original
	shuffled := make([]VideoInfo, len(videos))
	copy(shuffled, videos)

	// 👉 RANDOM REAL
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	startIdx := page * limit
	endIdx := startIdx + limit

	if startIdx >= len(shuffled) {
		sendJSON(w, []VideoInfo{})
		return
	}

	if endIdx > len(shuffled) {
		endIdx = len(shuffled)
	}

	sendJSON(w, shuffled[startIdx:endIdx])
}

// ----------------------
// Endpoint: Búsqueda
// ----------------------
func searchVideos(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	id := strings.ToLower(r.URL.Query().Get("id"))

	var results []VideoInfo

	for _, v := range videos {
		if id != "" && strings.ToLower(v.ID) == id {
			results = append(results, v)
			break
		}
		if q != "" && strings.Contains(strings.ToLower(v.Title), q) {
			results = append(results, v)
		}
	}

	sendJSON(w, results)
}

// ----------------------
// Endpoint: Random único
// ----------------------
func getRandom(w http.ResponseWriter, r *http.Request) {
	if len(videos) == 0 {
		http.Error(w, "No hay videos", http.StatusInternalServerError)
		return
	}

	randomVideo := videos[rand.Intn(len(videos))]
	sendJSON(w, randomVideo)
}

// ----------------------
// Endpoint: Sitemap
// ----------------------
func getSitemap(w http.ResponseWriter, r *http.Request) {
	baseURL := "https://viewdiful.vercel.app"

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`

	for _, v := range videos {
		xml += `
	<url>
		<loc>` + baseURL + `/video/` + v.ID + `</loc>
		<changefreq>daily</changefreq>
		<priority>0.8</priority>
	</url>`
	}

	xml += `
</urlset>`

	w.Write([]byte(xml))
}

// ----------------------
// Helper JSON
// ----------------------
func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

// ----------------------
// MAIN
// ----------------------
func main() {
	rand.Seed(time.Now().UnixNano()) // 🎲 RANDOM REAL

	loadJSON()

	r := mux.NewRouter()
	r.HandleFunc("/video/{id}", getSingleVideo).Methods("GET")
	r.HandleFunc("/videos", getPaginatedVideos).Methods("GET")
	r.HandleFunc("/search", searchVideos).Methods("GET")
	r.HandleFunc("/random", getRandom).Methods("GET")
	r.HandleFunc("/sitemap.xml", getSitemap).Methods("GET")

	port := "8080"
	log.Println("Servidor escuchando en http://localhost:" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
