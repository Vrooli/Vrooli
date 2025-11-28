package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

// Chat represents a conversation in the inbox
type Chat struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Preview     string    `json:"preview"`
	Model       string    `json:"model"`
	ViewMode    string    `json:"view_mode"` // "bubble" or "terminal"
	IsRead      bool      `json:"is_read"`
	IsArchived  bool      `json:"is_archived"`
	IsStarred   bool      `json:"is_starred"`
	LabelIDs    []string  `json:"label_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Message represents a single message in a chat
type Message struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"chat_id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	Model     string    `json:"model,omitempty"`
	TokenCount int      `json:"token_count,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Label represents a colored label for organizing chats
type Label struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	srv := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

	if err := srv.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS chats (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL DEFAULT 'New Chat',
		preview TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT 'claude-3-5-sonnet-20241022',
		view_mode TEXT NOT NULL DEFAULT 'bubble' CHECK (view_mode IN ('bubble', 'terminal')),
		is_read BOOLEAN NOT NULL DEFAULT false,
		is_archived BOOLEAN NOT NULL DEFAULT false,
		is_starred BOOLEAN NOT NULL DEFAULT false,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
		content TEXT NOT NULL,
		model TEXT,
		token_count INTEGER DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

	CREATE TABLE IF NOT EXISTS labels (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#6366f1',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS chat_labels (
		chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
		label_id UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
		PRIMARY KEY (chat_id, label_id)
	);

	CREATE INDEX IF NOT EXISTS idx_chat_labels_chat_id ON chat_labels(chat_id);
	CREATE INDEX IF NOT EXISTS idx_chat_labels_label_id ON chat_labels(label_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Chat endpoints
	s.router.HandleFunc("/api/v1/chats", s.handleListChats).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats", s.handleCreateChat).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}", s.handleGetChat).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}", s.handleUpdateChat).Methods("PATCH", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}", s.handleDeleteChat).Methods("DELETE", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}/messages", s.handleAddMessage).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}/read", s.handleToggleRead).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}/archive", s.handleToggleArchive).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{id}/star", s.handleToggleStar).Methods("POST", "OPTIONS")

	// Label endpoints
	s.router.HandleFunc("/api/v1/labels", s.handleListLabels).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/labels", s.handleCreateLabel).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/api/v1/labels/{id}", s.handleUpdateLabel).Methods("PATCH", "OPTIONS")
	s.router.HandleFunc("/api/v1/labels/{id}", s.handleDeleteLabel).Methods("DELETE", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{chatId}/labels/{labelId}", s.handleAssignLabel).Methods("PUT", "OPTIONS")
	s.router.HandleFunc("/api/v1/chats/{chatId}/labels/{labelId}", s.handleRemoveLabel).Methods("DELETE", "OPTIONS")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "agent-inbox-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Agent Inbox API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Chat handlers

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	// Query params for filtering
	archived := r.URL.Query().Get("archived") == "true"
	starred := r.URL.Query().Get("starred") == "true"

	query := `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.is_archived = $1
	`
	args := []interface{}{archived}

	if starred {
		query += " AND c.is_starred = true"
	}

	query += " GROUP BY c.id ORDER BY c.is_starred DESC, c.updated_at DESC"

	rows, err := s.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.jsonError(w, "Failed to list chats", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var c Chat
		var labelIDs []byte
		err := rows.Scan(&c.ID, &c.Name, &c.Preview, &c.Model, &c.ViewMode, &c.IsRead, &c.IsArchived, &c.IsStarred, &c.CreatedAt, &c.UpdatedAt, &labelIDs)
		if err != nil {
			continue
		}
		// Parse PostgreSQL array to []string
		c.LabelIDs = parsePostgresArray(string(labelIDs))
		chats = append(chats, c)
	}

	s.jsonResponse(w, chats, http.StatusOK)
}

func (s *Server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Model    string `json:"model"`
		ViewMode string `json:"view_mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Defaults
	if req.Name == "" {
		req.Name = "New Chat"
	}
	if req.Model == "" {
		req.Model = "claude-3-5-sonnet-20241022"
	}
	if req.ViewMode == "" {
		req.ViewMode = "bubble"
	}

	if req.ViewMode != "bubble" && req.ViewMode != "terminal" {
		s.jsonError(w, "view_mode must be 'bubble' or 'terminal'", http.StatusBadRequest)
		return
	}

	var chat Chat
	err := s.db.QueryRowContext(r.Context(), `
		INSERT INTO chats (name, model, view_mode)
		VALUES ($1, $2, $3)
		RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at
	`, req.Name, req.Model, req.ViewMode).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt,
	)
	if err != nil {
		s.jsonError(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	chat.LabelIDs = []string{}
	s.jsonResponse(w, chat, http.StatusCreated)
}

func (s *Server) handleGetChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var chat Chat
	var labelIDs []byte
	err := s.db.QueryRowContext(r.Context(), `
		SELECT c.id, c.name, c.preview, c.model, c.view_mode, c.is_read, c.is_archived, c.is_starred, c.created_at, c.updated_at,
			COALESCE(array_agg(cl.label_id) FILTER (WHERE cl.label_id IS NOT NULL), '{}') as label_ids
		FROM chats c
		LEFT JOIN chat_labels cl ON c.id = cl.chat_id
		WHERE c.id = $1
		GROUP BY c.id
	`, chatID).Scan(&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode, &chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt, &labelIDs)

	if err == sql.ErrNoRows {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.jsonError(w, "Failed to get chat", http.StatusInternalServerError)
		return
	}

	chat.LabelIDs = parsePostgresArray(string(labelIDs))

	// Get messages
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT id, chat_id, role, content, model, token_count, created_at
		FROM messages WHERE chat_id = $1 ORDER BY created_at ASC
	`, chatID)
	if err != nil {
		s.jsonError(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		var model sql.NullString
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &model, &m.TokenCount, &m.CreatedAt); err != nil {
			continue
		}
		if model.Valid {
			m.Model = model.String
		}
		messages = append(messages, m)
	}

	s.jsonResponse(w, map[string]interface{}{
		"chat":     chat,
		"messages": messages,
	}, http.StatusOK)
}

func (s *Server) handleUpdateChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Model *string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.Model != nil {
		updates = append(updates, fmt.Sprintf("model = $%d", argNum))
		args = append(args, *req.Model)
		argNum++
	}

	if len(updates) == 0 {
		s.jsonError(w, "No fields to update", http.StatusBadRequest)
		return
	}

	updates = append(updates, "updated_at = NOW()")
	args = append(args, chatID)

	query := fmt.Sprintf("UPDATE chats SET %s WHERE id = $%d RETURNING id, name, preview, model, view_mode, is_read, is_archived, is_starred, created_at, updated_at",
		strings.Join(updates, ", "), argNum)

	var chat Chat
	err := s.db.QueryRowContext(r.Context(), query, args...).Scan(
		&chat.ID, &chat.Name, &chat.Preview, &chat.Model, &chat.ViewMode,
		&chat.IsRead, &chat.IsArchived, &chat.IsStarred, &chat.CreatedAt, &chat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.jsonError(w, "Failed to update chat", http.StatusInternalServerError)
		return
	}

	// Get label IDs
	rows, err := s.db.QueryContext(r.Context(), "SELECT label_id FROM chat_labels WHERE chat_id = $1", chatID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var labelID string
			if rows.Scan(&labelID) == nil {
				chat.LabelIDs = append(chat.LabelIDs, labelID)
			}
		}
	}
	if chat.LabelIDs == nil {
		chat.LabelIDs = []string{}
	}

	s.jsonResponse(w, chat, http.StatusOK)
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	result, err := s.db.ExecContext(r.Context(), "DELETE FROM chats WHERE id = $1", chatID)
	if err != nil {
		s.jsonError(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAddMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		Model      string `json:"model"`
		TokenCount int    `json:"token_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role == "" || req.Content == "" {
		s.jsonError(w, "role and content are required", http.StatusBadRequest)
		return
	}

	if req.Role != "user" && req.Role != "assistant" && req.Role != "system" {
		s.jsonError(w, "role must be 'user', 'assistant', or 'system'", http.StatusBadRequest)
		return
	}

	// Verify chat exists
	var exists bool
	if err := s.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)", chatID).Scan(&exists); err != nil || !exists {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}

	var msg Message
	err := s.db.QueryRowContext(r.Context(), `
		INSERT INTO messages (chat_id, role, content, model, token_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, chat_id, role, content, model, token_count, created_at
	`, chatID, req.Role, req.Content, sql.NullString{String: req.Model, Valid: req.Model != ""}, req.TokenCount).Scan(
		&msg.ID, &msg.ChatID, &msg.Role, &msg.Content, &sql.NullString{}, &msg.TokenCount, &msg.CreatedAt,
	)
	if err != nil {
		s.jsonError(w, "Failed to add message", http.StatusInternalServerError)
		return
	}
	msg.Model = req.Model

	// Update chat preview and mark as unread if assistant message
	preview := req.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	updateQuery := "UPDATE chats SET preview = $1, updated_at = NOW()"
	if req.Role == "assistant" {
		updateQuery += ", is_read = false"
	}
	updateQuery += " WHERE id = $2"
	s.db.ExecContext(r.Context(), updateQuery, preview, chatID)

	s.jsonResponse(w, msg, http.StatusCreated)
}

func (s *Server) handleToggleRead(w http.ResponseWriter, r *http.Request) {
	s.toggleBool(w, r, "is_read")
}

func (s *Server) handleToggleArchive(w http.ResponseWriter, r *http.Request) {
	s.toggleBool(w, r, "is_archived")
}

func (s *Server) handleToggleStar(w http.ResponseWriter, r *http.Request) {
	s.toggleBool(w, r, "is_starred")
}

func (s *Server) toggleBool(w http.ResponseWriter, r *http.Request, field string) {
	vars := mux.Vars(r)
	chatID := vars["id"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Value *bool `json:"value"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var query string
	if req.Value != nil {
		query = fmt.Sprintf("UPDATE chats SET %s = $1, updated_at = NOW() WHERE id = $2 RETURNING %s", field, field)
		var newValue bool
		err := s.db.QueryRowContext(r.Context(), query, *req.Value, chatID).Scan(&newValue)
		if err == sql.ErrNoRows {
			s.jsonError(w, "Chat not found", http.StatusNotFound)
			return
		}
		if err != nil {
			s.jsonError(w, "Failed to update", http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]bool{field: newValue}, http.StatusOK)
	} else {
		query = fmt.Sprintf("UPDATE chats SET %s = NOT %s, updated_at = NOW() WHERE id = $1 RETURNING %s", field, field, field)
		var newValue bool
		err := s.db.QueryRowContext(r.Context(), query, chatID).Scan(&newValue)
		if err == sql.ErrNoRows {
			s.jsonError(w, "Chat not found", http.StatusNotFound)
			return
		}
		if err != nil {
			s.jsonError(w, "Failed to toggle", http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, map[string]bool{field: newValue}, http.StatusOK)
	}
}

// Label handlers

func (s *Server) handleListLabels(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), "SELECT id, name, color, created_at FROM labels ORDER BY name")
	if err != nil {
		s.jsonError(w, "Failed to list labels", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	labels := []Label{}
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			continue
		}
		labels = append(labels, l)
	}

	s.jsonResponse(w, labels, http.StatusOK)
}

func (s *Server) handleCreateLabel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		s.jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Color == "" {
		req.Color = "#6366f1"
	}

	var label Label
	err := s.db.QueryRowContext(r.Context(), `
		INSERT INTO labels (name, color)
		VALUES ($1, $2)
		RETURNING id, name, color, created_at
	`, req.Name, req.Color).Scan(&label.ID, &label.Name, &label.Color, &label.CreatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			s.jsonError(w, "Label with this name already exists", http.StatusConflict)
			return
		}
		s.jsonError(w, "Failed to create label", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, label, http.StatusCreated)
}

func (s *Server) handleUpdateLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	labelID := vars["id"]

	if _, err := uuid.Parse(labelID); err != nil {
		s.jsonError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updates := []string{}
	args := []interface{}{}
	argNum := 1

	if req.Name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argNum))
		args = append(args, *req.Color)
		argNum++
	}

	if len(updates) == 0 {
		s.jsonError(w, "No fields to update", http.StatusBadRequest)
		return
	}

	args = append(args, labelID)
	query := fmt.Sprintf("UPDATE labels SET %s WHERE id = $%d RETURNING id, name, color, created_at",
		strings.Join(updates, ", "), argNum)

	var label Label
	err := s.db.QueryRowContext(r.Context(), query, args...).Scan(&label.ID, &label.Name, &label.Color, &label.CreatedAt)

	if err == sql.ErrNoRows {
		s.jsonError(w, "Label not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.jsonError(w, "Failed to update label", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, label, http.StatusOK)
}

func (s *Server) handleDeleteLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	labelID := vars["id"]

	if _, err := uuid.Parse(labelID); err != nil {
		s.jsonError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	result, err := s.db.ExecContext(r.Context(), "DELETE FROM labels WHERE id = $1", labelID)
	if err != nil {
		s.jsonError(w, "Failed to delete label", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.jsonError(w, "Label not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAssignLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chatId"]
	labelID := vars["labelId"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(labelID); err != nil {
		s.jsonError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	// Verify both exist
	var chatExists, labelExists bool
	s.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM chats WHERE id = $1)", chatID).Scan(&chatExists)
	s.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM labels WHERE id = $1)", labelID).Scan(&labelExists)

	if !chatExists {
		s.jsonError(w, "Chat not found", http.StatusNotFound)
		return
	}
	if !labelExists {
		s.jsonError(w, "Label not found", http.StatusNotFound)
		return
	}

	_, err := s.db.ExecContext(r.Context(), `
		INSERT INTO chat_labels (chat_id, label_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, chatID, labelID)

	if err != nil {
		s.jsonError(w, "Failed to assign label", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"status": "assigned"}, http.StatusOK)
}

func (s *Server) handleRemoveLabel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID := vars["chatId"]
	labelID := vars["labelId"]

	if _, err := uuid.Parse(chatID); err != nil {
		s.jsonError(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(labelID); err != nil {
		s.jsonError(w, "Invalid label ID", http.StatusBadRequest)
		return
	}

	result, err := s.db.ExecContext(r.Context(), "DELETE FROM chat_labels WHERE chat_id = $1 AND label_id = $2", chatID, labelID)
	if err != nil {
		s.jsonError(w, "Failed to remove label", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.jsonError(w, "Label assignment not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, message string, status int) {
	s.jsonResponse(w, map[string]string{"error": message}, status)
}

func parsePostgresArray(arr string) []string {
	// Handle PostgreSQL array format: {uuid1,uuid2,...}
	arr = strings.Trim(arr, "{}")
	if arr == "" {
		return []string{}
	}
	return strings.Split(arr, ",")
}

// Middleware

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	// Get allowed origins from environment or use localhost defaults for development
	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOriginsStr == "" {
		// Default to localhost origins for development - lifecycle provides UI_PORT
		uiPort := os.Getenv("UI_PORT")
		if uiPort == "" {
			uiPort = "35000"
		}
		allowedOriginsStr = fmt.Sprintf("http://localhost:%s,http://127.0.0.1:%s", uiPort, uiPort)
	}
	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	allowedOriginSet := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginSet[strings.TrimSpace(origin)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowedOriginSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	log.Printf("%s | %v", msg, fields)
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start agent-inbox

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
