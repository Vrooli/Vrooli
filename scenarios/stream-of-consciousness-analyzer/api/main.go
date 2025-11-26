package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

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
	// Database configuration - support both POSTGRES_URL and individual components
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
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
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start stream-of-consciousness-analyzer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	initDB()
	defer db.Close()

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
