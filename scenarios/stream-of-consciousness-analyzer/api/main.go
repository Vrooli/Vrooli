package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Campaign struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ContextPrompt string    `json:"context_prompt"`
	Color         string    `json:"color"`
	Icon          string    `json:"icon"`
	Active        bool      `json:"active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StreamEntry struct {
	ID         string          `json:"id"`
	CampaignID string          `json:"campaign_id"`
	Content    string          `json:"content"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	Metadata   json.RawMessage `json:"metadata"`
	Processed  bool            `json:"processed"`
	CreatedAt  time.Time       `json:"created_at"`
}

type OrganizedNote struct {
	ID          string          `json:"id"`
	CampaignID  string          `json:"campaign_id"`
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	Summary     string          `json:"summary"`
	Category    string          `json:"category"`
	Tags        []string        `json:"tags"`
	Priority    int             `json:"priority"`
	EmbeddingID string          `json:"embedding_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Insight struct {
	ID         string          `json:"id"`
	CampaignID string          `json:"campaign_id"`
	NoteIDs    []string        `json:"note_ids"`
	Type       string          `json:"insight_type"`
	Content    string          `json:"content"`
	Confidence float64         `json:"confidence"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
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

	// Trigger in-process enrichment instead of external workflow engine
	go processStreamEntry(entry.ID, entry.CampaignID, entry.Content)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

// processStream provides a synchronous API to store a stream entry and immediately create an organized note.
func processStream(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Text     string `json:"text"`
		Campaign string `json:"campaign"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Text) == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	campaignID := payload.Campaign
	if campaignID == "" {
		campaignID = "default"
	}

	// Store stream entry
	var entry StreamEntry
	err := db.QueryRow(`
		INSERT INTO stream_entries (campaign_id, content, type, source, metadata, processed)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id, created_at
	`, campaignID, payload.Text, "text", "api", json.RawMessage(`{"source":"process-stream"}`)).
		Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		http.Error(w, "failed to store stream entry: "+err.Error(), http.StatusInternalServerError)
		return
	}
	entry.CampaignID = campaignID
	entry.Content = payload.Text
	entry.Type = "text"
	entry.Source = "api"
	entry.Processed = true

	note, err := insertNote(campaignID, generateTitle(payload.Text), payload.Text, generateSummary(payload.Text), "stream", []string{"stream", "api"}, json.RawMessage(`{"source":"process-stream"}`))
	if err != nil {
		http.Error(w, "failed to create note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"stream_entry": entry,
		"note":         note,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// organizeThoughts consolidates a set of thoughts into a single organized note.
func organizeThoughts(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Thoughts []string `json:"thoughts"`
		Campaign string   `json:"campaign"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(payload.Thoughts) == 0 {
		http.Error(w, "thoughts are required", http.StatusBadRequest)
		return
	}

	campaignID := payload.Campaign
	if campaignID == "" {
		campaignID = "default"
	}

	combined := strings.Join(payload.Thoughts, "\n")
	summary := generateSummary(combined)
	title := generateTitle(summary)
	tags := []string{"organized", "thoughts"}

	note, err := insertNote(campaignID, title, combined, summary, "organized", tags, json.RawMessage(`{"source":"organize-thoughts"}`))
	if err != nil {
		http.Error(w, "failed to create organized note: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

// extractInsights generates a lightweight insight linking notes together.
func extractInsights(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		NoteIDs  []string `json:"noteIds"`
		Campaign string   `json:"campaign"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(payload.NoteIDs) == 0 {
		http.Error(w, "noteIds are required", http.StatusBadRequest)
		return
	}

	campaignID := payload.Campaign
	if campaignID == "" {
		campaignID = "default"
	}

	insightContent := fmt.Sprintf("Combined insight for %d notes.", len(payload.NoteIDs))
	insight, err := insertInsight(campaignID, payload.NoteIDs, insightContent)
	if err != nil {
		http.Error(w, "failed to create insight: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insight)
}

// processStreamEntry performs lightweight summarization/tagging and stores a note locally.
func processStreamEntry(entryID, campaignID, content string) {
	title := generateTitle(content)
	summary := generateSummary(content)
	tags := []string{"stream", "auto"}
	category := "stream"

	_, err := insertNote(campaignID, title, content, summary, category, tags, json.RawMessage(`{"source":"stream-processor"}`))
	if err != nil {
		log.Printf("Failed to create organized note for stream entry %s: %v", entryID, err)
		return
	}

	if entryID != "" {
		_, _ = db.Exec(`UPDATE stream_entries SET processed = true WHERE id = $1`, entryID)
	}

	log.Printf("Processed stream entry %s into organized note", entryID)
}

func generateTitle(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "Untitled thought"
	}
	words := strings.Fields(trimmed)
	maxWords := 8
	if len(words) < maxWords {
		return strings.Title(strings.Join(words, " "))
	}
	return strings.Title(strings.Join(words[:maxWords], " ")) + "..."
}

func generateSummary(content string) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) <= 180 {
		return trimmed
	}
	return trimmed[:180] + "..."
}

func insertNote(campaignID, title, content, summary, category string, tags []string, metadata json.RawMessage) (*OrganizedNote, error) {
	if metadata == nil {
		metadata = json.RawMessage(`{}`)
	}

	tagsJSON, _ := json.Marshal(tags)

	var note OrganizedNote
	var tagsRaw sql.NullString
	var metadataRaw json.RawMessage
	err := db.QueryRow(`
		INSERT INTO organized_notes (campaign_id, title, content, summary, category, tags, priority, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, campaign_id, title, content, summary, category, tags, priority, embedding_id, metadata, created_at, updated_at
	`, campaignID, title, content, summary, category, string(tagsJSON), 1, metadata).
		Scan(&note.ID, &note.CampaignID, &note.Title, &note.Content, &note.Summary, &note.Category, &tagsRaw, &note.Priority, &note.EmbeddingID, &metadataRaw, &note.CreatedAt, &note.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if tagsRaw.Valid {
		_ = json.Unmarshal([]byte(tagsRaw.String), &note.Tags)
	}
	if metadataRaw != nil {
		note.Metadata = metadataRaw
	}

	return &note, nil
}

func insertInsight(campaignID string, noteIDs []string, content string) (*Insight, error) {
	if len(noteIDs) == 0 {
		return nil, fmt.Errorf("noteIDs required")
	}

	noteIDsJSON, _ := json.Marshal(noteIDs)
	metadata := json.RawMessage(`{"source":"insight-generator"}`)
	confidence := 0.7

	var insight Insight
	var noteIDsRaw []byte
	var metadataRaw []byte
	err := db.QueryRow(`
		INSERT INTO insights (campaign_id, note_ids, insight_type, content, confidence, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, campaign_id, note_ids, insight_type, content, confidence, metadata, created_at
	`, campaignID, string(noteIDsJSON), "summary", content, confidence, metadata).
		Scan(&insight.ID, &insight.CampaignID, &noteIDsRaw, &insight.Type, &insight.Content, &insight.Confidence, &metadataRaw, &insight.CreatedAt)

	if err != nil {
		return nil, err
	}

	if noteIDsRaw != nil {
		_ = json.Unmarshal(noteIDsRaw, &insight.NoteIDs)
	}
	if metadataRaw != nil {
		insight.Metadata = json.RawMessage(metadataRaw)
	}

	return &insight, nil
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
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "stream-of-consciousness-analyzer",
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "stream-of-consciousness-analyzer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Port configuration - check both API_PORT and PORT for compatibility
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}

	initDB()

	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/campaigns", getCampaigns).Methods("GET")
	router.HandleFunc("/api/campaigns", createCampaign).Methods("POST")
	router.HandleFunc("/api/stream/capture", captureStream).Methods("POST")
	router.HandleFunc("/api/process-stream", processStream).Methods("POST")
	router.HandleFunc("/api/organize-thoughts", organizeThoughts).Methods("POST")
	router.HandleFunc("/api/extract-insights", extractInsights).Methods("POST")
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

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: handler,
		Port:    port,
		Cleanup: func(ctx context.Context) error {
			if db != nil {
				return db.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
