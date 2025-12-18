package main

import (
	"github.com/vrooli/api-core/preflight"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Structured logger for better observability
type Logger struct {
	prefix string
}

func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.log("FATAL", msg, fields...)
	os.Exit(1)
}

func (l *Logger) log(level, msg string, fields ...interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"service":   "smartnotes-api",
		"message":   msg,
	}

	// Add prefix if provided
	if l.prefix != "" {
		entry["component"] = l.prefix
	}

	// Add fields as key-value pairs
	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			entry[key] = fields[i+1]
		}
	}

	jsonBytes, _ := json.Marshal(entry)
	log.Println(string(jsonBytes))
}

type Note struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	FolderID     *string   `json:"folder_id,omitempty"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	ContentType  string    `json:"content_type"`
	Summary      *string   `json:"summary,omitempty"`
	IsPinned     bool      `json:"is_pinned"`
	IsArchived   bool      `json:"is_archived"`
	IsFavorite   bool      `json:"is_favorite"`
	WordCount    int       `json:"word_count"`
	ReadingTime  int       `json:"reading_time_minutes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastAccessed time.Time `json:"last_accessed"`
	Tags         []string  `json:"tags,omitempty"`
}

type Folder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Tag struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	UsageCount int    `json:"usage_count"`
}

type Template struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	UsageCount  int       `json:"usage_count"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB
var defaultUserID string
var logger *Logger

func getDefaultUserID() string {
	if defaultUserID == "" {
		// Fetch the default user from the database
		err := db.QueryRow("SELECT id FROM users WHERE username = 'default' LIMIT 1").Scan(&defaultUserID)
		if err != nil {
			// If no default user exists, create one
			err = db.QueryRow(`
				INSERT INTO users (username, email, password_hash, preferences) 
				VALUES ('default', 'user@smartnotes.local', 'placeholder_hash', '{"theme": "light"}')
				ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
				RETURNING id
			`).Scan(&defaultUserID)
			if err != nil {
				logger.Warn("Could not get or create default user", "error", err.Error())
				// Use a fallback UUID
				defaultUserID = "00000000-0000-0000-0000-000000000001"
			}
		}
	}
	return defaultUserID
}

func normalizeUserID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "default") || strings.EqualFold(trimmed, "default_user") || strings.EqualFold(trimmed, "default-user") {
		return getDefaultUserID()
	}

	if _, err := uuid.Parse(trimmed); err != nil {
		if logger != nil {
			logger.Warn("Invalid user ID provided, falling back to default user", "provided_user_id", trimmed, "error", err.Error())
		} else {
			log.Printf("[WARN] Invalid user ID provided (%s), falling back to default user: %v", trimmed, err)
		}
		return getDefaultUserID()
	}

	return trimmed
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "notes",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logger
	logger = NewLogger("main")
	// Database configuration - support both POSTGRES_URL and individual components
	// SmartNotes uses its own database named 'notes'
	dbURL := os.Getenv("NOTES_DB_URL")
	if dbURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		// Always use 'notes' database for SmartNotes scenario
		dbName := "notes"

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" {
			logger.Fatal("Database configuration missing. Provide NOTES_DB_URL or all required POSTGRES_* environment variables",
				"required_vars", []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD"})
		}

		// Build connection string without logging sensitive info
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
		logger.Info("Database connection string configured", "database", dbName, "host", dbHost, "port", dbPort)
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("Failed to open database connection", "error", err.Error())
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	logger.Info("Attempting database connection with exponential backoff", "max_retries", maxRetries, "base_delay", baseDelay.String())

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Info("Database connected successfully", "attempt", attempt+1, "max_retries", maxRetries)
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

		logger.Warn("Database connection attempt failed",
			"attempt", attempt+1,
			"max_retries", maxRetries,
			"error", pingErr.Error(),
			"retry_delay", actualDelay.String())

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Info("Retry progress update",
				"attempts_made", attempt+1,
				"max_retries", maxRetries,
				"current_delay", delay.String(),
				"jitter", jitter.String())
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		logger.Fatal("Database connection failed after all retries",
			"max_retries", maxRetries,
			"error", pingErr.Error())
	}

	logger.Info("Database connection pool established successfully",
		"max_open_conns", 25,
		"max_idle_conns", 5,
		"conn_max_lifetime", "5m")

	// Setup routes
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Notes endpoints
	router.HandleFunc("/api/notes", getNotesHandler).Methods("GET")
	router.HandleFunc("/api/notes", createNoteHandler).Methods("POST")
	router.HandleFunc("/api/notes/{id}", getNoteHandler).Methods("GET")
	router.HandleFunc("/api/notes/{id}", updateNoteHandler).Methods("PUT")
	router.HandleFunc("/api/notes/{id}", deleteNoteHandler).Methods("DELETE")

	// Folders endpoints
	router.HandleFunc("/api/folders", getFoldersHandler).Methods("GET")
	router.HandleFunc("/api/folders", createFolderHandler).Methods("POST")
	router.HandleFunc("/api/folders/{id}", updateFolderHandler).Methods("PUT")
	router.HandleFunc("/api/folders/{id}", deleteFolderHandler).Methods("DELETE")

	// Tags endpoints
	router.HandleFunc("/api/tags", getTagsHandler).Methods("GET")
	router.HandleFunc("/api/tags", createTagHandler).Methods("POST")

	// Templates endpoints
	router.HandleFunc("/api/templates", getTemplatesHandler).Methods("GET")
	router.HandleFunc("/api/templates", createTemplateHandler).Methods("POST")

	// Search endpoints
	router.HandleFunc("/api/search", searchHandler).Methods("POST")
	router.HandleFunc("/api/search/semantic", semanticSearchHandler).Methods("POST")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("API_PORT environment variable is required")
	}

	logger.Info("SmartNotes API starting", "port", port, "endpoints", map[string]int{
		"notes":     5,
		"folders":   4,
		"tags":      2,
		"templates": 2,
		"search":    2,
		"health":    1,
	})

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		logger.Fatal("Server failed to start", "port", port, "error", err.Error())
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbConnected := false
	var dbLatency float64
	var dbError map[string]interface{}

	if db != nil {
		start := time.Now()
		err := db.Ping()
		dbLatency = float64(time.Since(start).Milliseconds())

		if err == nil {
			dbConnected = true
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   "Database ping failed",
				"category":  "resource",
				"retryable": true,
			}
		}
	} else {
		dbError = map[string]interface{}{
			"code":      "NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
			"retryable": false,
		}
	}

	// Determine overall health status
	status := "healthy"
	readiness := true
	if !dbConnected {
		status = "degraded"
		readiness = false
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "smartnotes-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": readiness,
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getNotesHandler(w http.ResponseWriter, r *http.Request) {
	userID := normalizeUserID(r.URL.Query().Get("user_id"))

	query := `
		SELECT n.id, n.title, n.content, n.content_type, n.summary,
		       n.is_pinned, n.is_archived, n.is_favorite,
		       n.word_count, n.reading_time_minutes,
		       n.created_at, n.updated_at, n.last_accessed,
		       n.folder_id,
		       array_agg(DISTINCT t.name) as tags
		FROM notes n
		LEFT JOIN note_tags nt ON n.id = nt.note_id
		LEFT JOIN tags t ON nt.tag_id = t.id
		WHERE n.user_id = $1 AND n.is_archived = false
		GROUP BY n.id
		ORDER BY n.is_pinned DESC, n.updated_at DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	notes := []Note{} // Initialize as empty slice instead of nil
	for rows.Next() {
		var n Note
		var tags sql.NullString
		err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.ContentType, &n.Summary,
			&n.IsPinned, &n.IsArchived, &n.IsFavorite,
			&n.WordCount, &n.ReadingTime,
			&n.CreatedAt, &n.UpdatedAt, &n.LastAccessed,
			&n.FolderID, &tags)
		if err != nil {
			continue
		}
		n.UserID = userID
		if tags.Valid && tags.String != "" {
			json.Unmarshal([]byte(tags.String), &n.Tags)
		}
		notes = append(notes, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func createNoteHandler(w http.ResponseWriter, r *http.Request) {
	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	note.UserID = normalizeUserID(note.UserID)
	if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
		note.UserID = normalizeUserID(queryUserID)
	}

	// Set defaults if not provided
	if note.ContentType == "" {
		note.ContentType = "markdown"
	}

	query := `
		INSERT INTO notes (user_id, title, content, content_type, folder_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, last_accessed`

	err := db.QueryRow(query, note.UserID, note.Title, note.Content,
		note.ContentType, note.FolderID).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt, &note.LastAccessed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate word count and reading time
	note.WordCount = len(note.Content) / 5          // Rough estimate
	note.ReadingTime = (note.WordCount + 199) / 200 // Assuming 200 words per minute

	// Index in Qdrant for semantic search
	go indexNoteInQdrant(note)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func getNoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var note Note
	query := `
		SELECT n.id, n.user_id, n.title, n.content, n.content_type, n.summary,
		       n.is_pinned, n.is_archived, n.is_favorite,
		       n.word_count, n.reading_time_minutes,
		       n.created_at, n.updated_at, n.last_accessed, n.folder_id
		FROM notes n
		WHERE n.id = $1`

	err := db.QueryRow(query, id).Scan(&note.ID, &note.UserID, &note.Title,
		&note.Content, &note.ContentType, &note.Summary,
		&note.IsPinned, &note.IsArchived, &note.IsFavorite,
		&note.WordCount, &note.ReadingTime,
		&note.CreatedAt, &note.UpdatedAt, &note.LastAccessed, &note.FolderID)

	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	// Update last accessed
	db.Exec("UPDATE notes SET last_accessed = CURRENT_TIMESTAMP WHERE id = $1", id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func updateNoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE notes 
		SET title = $2, content = $3, folder_id = $4,
		    is_pinned = $5, is_favorite = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING user_id, updated_at`

	err := db.QueryRow(query, id, note.Title, note.Content, note.FolderID,
		note.IsPinned, note.IsFavorite).Scan(&note.UserID, &note.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	note.ID = id
	note.UserID = normalizeUserID(note.UserID)

	// Re-index in Qdrant with updated content
	go indexNoteInQdrant(note)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("DELETE FROM notes WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from Qdrant as well
	go deleteNoteFromQdrant(id)

	w.WriteHeader(http.StatusNoContent)
}

func getFoldersHandler(w http.ResponseWriter, r *http.Request) {
	userID := normalizeUserID(r.URL.Query().Get("user_id"))

	query := `
		SELECT id, user_id, parent_id, name, icon, color, position, created_at, updated_at
		FROM folders
		WHERE user_id = $1
		ORDER BY position, name`

	rows, err := db.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.Icon,
			&f.Color, &f.Position, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		folders = append(folders, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func createFolderHandler(w http.ResponseWriter, r *http.Request) {
	var folder Folder
	if err := json.NewDecoder(r.Body).Decode(&folder); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	folder.UserID = normalizeUserID(folder.UserID)
	if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
		folder.UserID = normalizeUserID(queryUserID)
	}
	if folder.Icon == "" {
		folder.Icon = "📁"
	}
	if folder.Color == "" {
		folder.Color = "#6366f1"
	}

	query := `
		INSERT INTO folders (user_id, parent_id, name, icon, color, position)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := db.QueryRow(query, folder.UserID, folder.ParentID, folder.Name,
		folder.Icon, folder.Color, folder.Position).Scan(&folder.ID, &folder.CreatedAt, &folder.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func updateFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var folder Folder
	if err := json.NewDecoder(r.Body).Decode(&folder); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE folders 
		SET name = $2, icon = $3, color = $4, position = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at`

	err := db.QueryRow(query, id, folder.Name, folder.Icon, folder.Color,
		folder.Position).Scan(&folder.UpdatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	folder.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folder)
}

func deleteFolderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("DELETE FROM folders WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getTagsHandler(w http.ResponseWriter, r *http.Request) {
	userID := normalizeUserID(r.URL.Query().Get("user_id"))

	query := `
		SELECT id, name, color, usage_count
		FROM tags
		WHERE user_id = $1
		ORDER BY usage_count DESC, name`

	rows, err := db.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.UsageCount)
		if err != nil {
			continue
		}
		t.UserID = userID
		tags = append(tags, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func createTagHandler(w http.ResponseWriter, r *http.Request) {
	var tag Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userID := normalizeUserID(tag.UserID)
	if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
		userID = normalizeUserID(queryUserID)
	}
	tag.UserID = userID

	if tag.Color == "" {
		tag.Color = "#10b981"
	}

	query := `
		INSERT INTO tags (user_id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, name) DO UPDATE
		SET usage_count = tags.usage_count + 1
		RETURNING id, usage_count`

	err := db.QueryRow(query, userID, tag.Name, tag.Color).Scan(&tag.ID, &tag.UsageCount)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	userID := normalizeUserID(r.URL.Query().Get("user_id"))

	query := `
		SELECT id, user_id, name, description, content, category, usage_count, is_public, created_at
		FROM templates
		WHERE user_id = $1 OR is_public = true
		ORDER BY usage_count DESC, name`

	rows, err := db.Query(query, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Description, &t.Content,
			&t.Category, &t.UsageCount, &t.IsPublic, &t.CreatedAt)
		if err != nil {
			continue
		}
		templates = append(templates, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func createTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var template Template
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	template.UserID = normalizeUserID(template.UserID)
	if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
		template.UserID = normalizeUserID(queryUserID)
	}

	query := `
		INSERT INTO templates (user_id, name, description, content, category, is_public)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := db.QueryRow(query, template.UserID, template.Name, template.Description,
		template.Content, template.Category, template.IsPublic).Scan(&template.ID, &template.CreatedAt)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	var searchReq struct {
		Query  string `json:"query"`
		UserID string `json:"user_id"`
		Limit  int    `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	searchReq.UserID = normalizeUserID(searchReq.UserID)
	if queryUserID := r.URL.Query().Get("user_id"); queryUserID != "" {
		searchReq.UserID = normalizeUserID(queryUserID)
	}
	if searchReq.Limit == 0 {
		searchReq.Limit = 20
	}

	// Simple text search - in production, this would call the n8n workflow
	query := `
		SELECT n.id, n.title, n.content, n.summary, n.created_at, n.updated_at
		FROM notes n
		WHERE n.user_id = $1
		  AND (n.title ILIKE $2 OR n.content ILIKE $2)
		  AND n.is_archived = false
		ORDER BY n.updated_at DESC
		LIMIT $3`

	searchPattern := fmt.Sprintf("%%%s%%", searchReq.Query)
	rows, err := db.Query(query, searchReq.UserID, searchPattern, searchReq.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Note
	for rows.Next() {
		var n Note
		err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Summary, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			continue
		}
		n.UserID = searchReq.UserID
		results = append(results, n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"query":   searchReq.Query,
		"count":   len(results),
	})
}
