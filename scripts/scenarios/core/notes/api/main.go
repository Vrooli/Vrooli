package main

import (
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

type Note struct {
	ID                 string    `json:"id"`
	UserID            string    `json:"user_id"`
	FolderID          *string   `json:"folder_id,omitempty"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	ContentType       string    `json:"content_type"`
	Summary           *string   `json:"summary,omitempty"`
	IsPinned          bool      `json:"is_pinned"`
	IsArchived        bool      `json:"is_archived"`
	IsFavorite        bool      `json:"is_favorite"`
	WordCount         int       `json:"word_count"`
	ReadingTime       int       `json:"reading_time_minutes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	LastAccessed      time.Time `json:"last_accessed"`
	Tags              []string  `json:"tags,omitempty"`
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

func main() {
	// Initialize database
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/notes?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

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
	
	// Search endpoint
	router.HandleFunc("/api/search", searchHandler).Methods("POST")
	
	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	
	handler := c.Handler(router)
	
	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8950"
	}
	
	log.Printf("SmartNotes API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "smartnotes-api"})
}

func getNotesHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	
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
	
	var notes []Note
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
	
	if note.UserID == "" {
		note.UserID = "default-user"
	}
	
	query := `
		INSERT INTO notes (user_id, title, content, content_type, folder_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	
	err := db.QueryRow(query, note.UserID, note.Title, note.Content, 
		note.ContentType, note.FolderID).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
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
		RETURNING updated_at`
	
	err := db.QueryRow(query, id, note.Title, note.Content, note.FolderID,
		note.IsPinned, note.IsFavorite).Scan(&note.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	note.ID = id
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
	
	w.WriteHeader(http.StatusNoContent)
}

func getFoldersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	
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
	
	if folder.UserID == "" {
		folder.UserID = "default-user"
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	
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
	
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	
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
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default-user"
	}
	
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
	
	if template.UserID == "" {
		template.UserID = "default-user"
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
	
	if searchReq.UserID == "" {
		searchReq.UserID = "default-user"
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