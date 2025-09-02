package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Download struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
	Duration    int    `json:"duration"`
	Format      string `json:"format"`
	Quality     string `json:"quality"`
	FilePath    string `json:"file_path"`
	FileSize    int64  `json:"file_size"`
	Status      string `json:"status"`
	Progress    int    `json:"progress"`
	Error       string `json:"error_message"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
}

type DownloadRequest struct {
	URL      string `json:"url"`
	Quality  string `json:"quality"`
	Format   string `json:"format"`
	AudioOnly bool  `json:"audio_only"`
	UserID   string `json:"user_id"`
}

type QueueItem struct {
	ID         int `json:"id"`
	DownloadID int `json:"download_id"`
	Position   int `json:"position"`
	Priority   int `json:"priority"`
	RetryCount int `json:"retry_count"`
}

var db *sql.DB

func initDB() {
	var err error
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "video_downloader"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	log.Println("Connected to database")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func createDownloadHandler(w http.ResponseWriter, r *http.Request) {
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Insert download record
	var downloadID int
	err := db.QueryRow(`
		INSERT INTO downloads (url, title, format, quality, status, user_id)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id`,
		req.URL, "Processing...", req.Format, req.Quality, req.UserID).Scan(&downloadID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add to queue
	_, err = db.Exec(`
		INSERT INTO download_queue (download_id, position, priority)
		VALUES ($1, (SELECT COALESCE(MAX(position), 0) + 1 FROM download_queue), 1)`,
		downloadID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": downloadID,
		"status": "queued",
	})
}

func getQueueHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT d.id, d.url, d.title, d.format, d.quality, d.status, d.progress
		FROM downloads d
		JOIN download_queue dq ON d.id = dq.download_id
		WHERE d.status IN ('pending', 'processing')
		ORDER BY dq.priority DESC, dq.position ASC`)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var queue []Download
	for rows.Next() {
		var d Download
		err := rows.Scan(&d.ID, &d.URL, &d.Title, &d.Format, &d.Quality, &d.Status, &d.Progress)
		if err != nil {
			continue
		}
		queue = append(queue, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queue)
}

func getHistoryHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, url, title, format, quality, status, created_at
		FROM downloads
		WHERE status = 'completed'
		ORDER BY created_at DESC
		LIMIT 100`)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []Download
	for rows.Next() {
		var d Download
		err := rows.Scan(&d.ID, &d.URL, &d.Title, &d.Format, &d.Quality, &d.Status, &d.CreatedAt)
		if err != nil {
			continue
		}
		history = append(history, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func deleteDownloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Update status to cancelled
	_, err = db.Exec("UPDATE downloads SET status = 'cancelled' WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from queue
	_, err = db.Exec("DELETE FROM download_queue WHERE download_id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cancelled"})
}

func analyzeURLHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Call n8n workflow to analyze URL
	// For now, return mock data
	response := map[string]interface{}{
		"url": req.URL,
		"title": "Sample Video Title",
		"description": "This is a sample video description",
		"platform": "YouTube",
		"duration": 300,
		"thumbnail": "https://via.placeholder.com/320x180",
		"availableQualities": []string{"1080p", "720p", "480p", "360p"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func processQueueHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Trigger n8n workflow to process queue
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
}

func main() {
	initDB()
	defer db.Close()

	router := mux.NewRouter()
	
	// API routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/download", createDownloadHandler).Methods("POST")
	router.HandleFunc("/api/queue", getQueueHandler).Methods("GET")
	router.HandleFunc("/api/queue/process", processQueueHandler).Methods("POST")
	router.HandleFunc("/api/history", getHistoryHandler).Methods("GET")
	router.HandleFunc("/api/download/{id}", deleteDownloadHandler).Methods("DELETE")
	router.HandleFunc("/api/analyze", analyzeURLHandler).Methods("POST")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8850"
	}

	log.Printf("Video Downloader API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
