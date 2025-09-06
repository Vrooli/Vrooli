package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Campaign struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ContextPrompt string   `json:"context_prompt"`
	Color        string    `json:"color"`
	Icon         string    `json:"icon"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StreamEntry struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	Content    string    `json:"content"`
	Type       string    `json:"type"`
	Source     string    `json:"source"`
	Metadata   json.RawMessage `json:"metadata"`
	Processed  bool      `json:"processed"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrganizedNote struct {
	ID          string    `json:"id"`
	CampaignID  string    `json:"campaign_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Summary     string    `json:"summary"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Priority    int       `json:"priority"`
	EmbeddingID string    `json:"embedding_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Insight struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	NoteIDs    []string  `json:"note_ids"`
	Type       string    `json:"insight_type"`
	Content    string    `json:"content"`
	Confidence float64   `json:"confidence"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time `json:"created_at"`
}

var db *sql.DB
var n8nBaseURL string

func initDB() {
	var err error
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		log.Fatal("POSTGRES_URL environment variable is required")
	}
	
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("Database connected successfully")
}

func getCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, description, context_prompt, color, icon, active, created_at, updated_at
		FROM campaigns
		WHERE active = true
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ContextPrompt, 
			&c.Color, &c.Icon, &c.Active, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			continue
		}
		campaigns = append(campaigns, c)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaigns)
}

func createCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign Campaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	err := db.QueryRow(`
		INSERT INTO campaigns (name, description, context_prompt, color, icon)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, campaign.Name, campaign.Description, campaign.ContextPrompt, 
		campaign.Color, campaign.Icon).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	campaign.Active = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

func captureStream(w http.ResponseWriter, r *http.Request) {
	var entry StreamEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Save to database
	err := db.QueryRow(`
		INSERT INTO stream_entries (campaign_id, content, type, source, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, entry.CampaignID, entry.Content, entry.Type, entry.Source, entry.Metadata).Scan(&entry.ID, &entry.CreatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Trigger processing via n8n webhook
	go triggerProcessing(entry.ID, entry.CampaignID, entry.Content)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func triggerProcessing(entryID, campaignID, content string) {
	webhookURL := fmt.Sprintf("%s/webhook/process-stream", n8nBaseURL)
	
	payload := map[string]interface{}{
		"stream_entry_id": entryID,
		"campaign_id": campaignID,
		"content": content,
	}
	
	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to trigger processing: %v", err)
		return
	}
	defer resp.Body.Close()
	
	log.Printf("Processing triggered for entry %s", entryID)
}

func getNotes(w http.ResponseWriter, r *http.Request) {
	campaignID := r.URL.Query().Get("campaign_id")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}
	
	query := `
		SELECT id, campaign_id, title, content, summary, category, tags, 
			   priority, embedding_id, metadata, created_at, updated_at
		FROM organized_notes
	`
	
	var rows *sql.Rows
	var err error
	
	if campaignID != "" {
		query += " WHERE campaign_id = $1 ORDER BY created_at DESC LIMIT $2"
		rows, err = db.Query(query, campaignID, limit)
	} else {
		query += " ORDER BY created_at DESC LIMIT $1"
		rows, err = db.Query(query, limit)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var notes []OrganizedNote
	for rows.Next() {
		var n OrganizedNote
		var tags sql.NullString
		err := rows.Scan(&n.ID, &n.CampaignID, &n.Title, &n.Content, &n.Summary,
			&n.Category, &tags, &n.Priority, &n.EmbeddingID, &n.Metadata,
			&n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			continue
		}
		if tags.Valid {
			// Parse PostgreSQL array
			json.Unmarshal([]byte(tags.String), &n.Tags)
		}
		notes = append(notes, n)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func getInsights(w http.ResponseWriter, r *http.Request) {
	campaignID := r.URL.Query().Get("campaign_id")
	
	query := `
		SELECT id, campaign_id, note_ids, insight_type, content, 
			   confidence, metadata, created_at
		FROM insights
		WHERE confidence >= 0.6
	`
	
	var rows *sql.Rows
	var err error
	
	if campaignID != "" {
		query += " AND campaign_id = $1 ORDER BY created_at DESC LIMIT 20"
		rows, err = db.Query(query, campaignID)
	} else {
		query += " ORDER BY created_at DESC LIMIT 20"
		rows, err = db.Query(query)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var insights []Insight
	for rows.Next() {
		var i Insight
		var noteIDs sql.NullString
		err := rows.Scan(&i.ID, &i.CampaignID, &noteIDs, &i.Type,
			&i.Content, &i.Confidence, &i.Metadata, &i.CreatedAt)
		if err != nil {
			continue
		}
		if noteIDs.Valid {
			json.Unmarshal([]byte(noteIDs.String), &i.NoteIDs)
		}
		insights = append(insights, i)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

func searchNotes(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	campaignID := r.URL.Query().Get("campaign_id")
	
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}
	
	// For semantic search, we'd call the smart-semantic-search webhook
	// For now, we'll do a simple text search
	sqlQuery := `
		SELECT id, campaign_id, title, content, summary, category, tags,
			   priority, metadata, created_at, updated_at
		FROM organized_notes
		WHERE (title ILIKE $1 OR content ILIKE $1 OR summary ILIKE $1)
	`
	
	searchTerm := "%" + query + "%"
	var rows *sql.Rows
	var err error
	
	if campaignID != "" {
		sqlQuery += " AND campaign_id = $2 ORDER BY created_at DESC LIMIT 20"
		rows, err = db.Query(sqlQuery, searchTerm, campaignID)
	} else {
		sqlQuery += " ORDER BY created_at DESC LIMIT 20"
		rows, err = db.Query(sqlQuery, searchTerm)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var notes []OrganizedNote
	for rows.Next() {
		var n OrganizedNote
		var tags sql.NullString
		var embeddingID sql.NullString
		err := rows.Scan(&n.ID, &n.CampaignID, &n.Title, &n.Content, &n.Summary,
			&n.Category, &tags, &n.Priority, &n.Metadata,
			&n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			continue
		}
		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &n.Tags)
		}
		if embeddingID.Valid {
			n.EmbeddingID = embeddingID.String
		}
		notes = append(notes, n)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error": err.Error(),
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "stream-of-consciousness-analyzer",
	})
}

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))
	
	n8nBaseURL = os.Getenv("N8N_BASE_URL")
	if n8nBaseURL == "" {
		n8nBaseURL = "http://localhost:5678"
	}
	
	initDB()
	defer db.Close()
	
	router := mux.NewRouter()
	
	// API routes
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/campaigns", getCampaigns).Methods("GET")
	router.HandleFunc("/api/campaigns", createCampaign).Methods("POST")
	router.HandleFunc("/api/stream/capture", captureStream).Methods("POST")
	router.HandleFunc("/api/notes", getNotes).Methods("GET")
	router.HandleFunc("/api/insights", getInsights).Methods("GET")
	router.HandleFunc("/api/search", searchNotes).Methods("GET")
	
	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	log.Printf("Stream of Consciousness API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
