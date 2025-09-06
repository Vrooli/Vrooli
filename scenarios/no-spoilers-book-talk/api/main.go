package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const (
	serviceName    = "no-spoilers-book-talk-api"
	apiVersion     = "1.0.0"
	maxFileSize    = 50 * 1024 * 1024 // 50MB max file size
	maxChatHistory = 50                // Maximum chat history to return
)

// Book represents a book in the system
type Book struct {
	ID               uuid.UUID              `json:"id"`
	Title            string                 `json:"title"`
	Author           string                 `json:"author,omitempty"`
	FilePath         string                 `json:"file_path"`
	FileType         string                 `json:"file_type"`
	FileSizeBytes    int64                  `json:"file_size_bytes,omitempty"`
	TotalChunks      int                    `json:"total_chunks"`
	TotalWords       int                    `json:"total_words"`
	TotalCharacters  int                    `json:"total_characters,omitempty"`
	Chapters         []Chapter              `json:"chapters,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	ProcessingStatus string                 `json:"processing_status"`
	ProcessingError  string                 `json:"processing_error,omitempty"`
	ProcessedAt      *time.Time             `json:"processed_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Chapter represents a chapter in a book
type Chapter struct {
	Title         string `json:"title"`
	StartPosition int    `json:"start_position"`
	EndPosition   int    `json:"end_position"`
	WordCount     int    `json:"word_count,omitempty"`
}

// UserProgress represents a user's reading progress
type UserProgress struct {
	ID                    uuid.UUID `json:"id"`
	BookID                uuid.UUID `json:"book_id"`
	UserID                string    `json:"user_id"`
	CurrentPosition       int       `json:"current_position"`
	PositionType          string    `json:"position_type"`
	PositionValue         float64   `json:"position_value"`
	LastReadChunkID       *int      `json:"last_read_chunk_id,omitempty"`
	ReadingSessionCount   int       `json:"reading_session_count"`
	TotalReadingTimeMin   int       `json:"total_reading_time_minutes"`
	Notes                 string    `json:"notes,omitempty"`
	FavoriteQuotes        []string  `json:"favorite_quotes,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
	CreatedAt             time.Time `json:"created_at"`
	PercentageComplete    float64   `json:"percentage_complete"`
}

// Conversation represents a chat interaction
type Conversation struct {
	ID                       uuid.UUID `json:"id"`
	BookID                   uuid.UUID `json:"book_id"`
	UserID                   string    `json:"user_id"`
	UserMessage              string    `json:"user_message"`
	AIResponse               string    `json:"ai_response"`
	UserPosition             int       `json:"user_position"`
	PositionType             string    `json:"position_type"`
	ContextChunksUsed        []int     `json:"context_chunks_used,omitempty"`
	SourcesReferenced        []string  `json:"sources_referenced,omitempty"`
	PositionBoundaryRespected bool      `json:"position_boundary_respected"`
	ResponseQualityScore     *float64  `json:"response_quality_score,omitempty"`
	ProcessingTimeMs         int       `json:"processing_time_ms,omitempty"`
	CreatedAt                time.Time `json:"created_at"`
}

// BookTalkService handles the main API operations
type BookTalkService struct {
	db         *sql.DB
	n8nBaseURL string
	qdrantURL  string
	dataDir    string
	logger     *log.Logger
}

// NewBookTalkService creates a new service instance
func NewBookTalkService(db *sql.DB, n8nURL, qdrantURL, dataDir string) *BookTalkService {
	return &BookTalkService{
		db:         db,
		n8nBaseURL: n8nURL,
		qdrantURL:  qdrantURL,
		dataDir:    dataDir,
		logger:     log.New(os.Stdout, "[book-talk-api] ", log.LstdFlags|log.Lshortfile),
	}
}

// Health endpoint
func (s *BookTalkService) Health(w http.ResponseWriter, r *http.Request) {
	// Test database connection
	if err := s.db.Ping(); err != nil {
		s.httpError(w, "Database connection failed", http.StatusServiceUnavailable, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "healthy",
		"service":        serviceName,
		"version":        apiVersion,
		"timestamp":      time.Now().UTC(),
		"data_directory": s.dataDir,
		"n8n_url":        s.n8nBaseURL,
		"qdrant_url":     s.qdrantURL,
	})
}

// UploadBook handles book file uploads and processing
func (s *BookTalkService) UploadBook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		s.httpError(w, "File too large or invalid multipart form", http.StatusBadRequest, err)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.httpError(w, "No file provided", http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	// Validate file type
	fileType := strings.ToLower(filepath.Ext(header.Filename))
	if fileType == "" {
		fileType = "txt" // Default to txt if no extension
	} else {
		fileType = fileType[1:] // Remove the dot
	}

	supportedTypes := []string{"txt", "epub", "pdf"}
	if !contains(supportedTypes, fileType) {
		s.httpError(w, fmt.Sprintf("Unsupported file type: %s. Supported: %v", fileType, supportedTypes), http.StatusBadRequest, nil)
		return
	}

	// Get optional parameters
	title := r.FormValue("title")
	if title == "" {
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}
	author := r.FormValue("author")
	userID := r.FormValue("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	// Save uploaded file
	bookID := uuid.New()
	uploadsDir := filepath.Join(s.dataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		s.httpError(w, "Failed to create upload directory", http.StatusInternalServerError, err)
		return
	}

	filename := fmt.Sprintf("%s_%s", bookID.String(), header.Filename)
	filePath := filepath.Join(uploadsDir, filename)
	
	dst, err := os.Create(filePath)
	if err != nil {
		s.httpError(w, "Failed to save uploaded file", http.StatusInternalServerError, err)
		return
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		s.httpError(w, "Failed to save file content", http.StatusInternalServerError, err)
		return
	}

	// Create book record in database
	_, err = s.db.Exec(`
		INSERT INTO books (id, title, author, file_path, file_type, file_size_bytes, processing_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		bookID, title, author, filePath, fileType, fileSize, "pending")
	
	if err != nil {
		s.httpError(w, "Failed to create book record", http.StatusInternalServerError, err)
		return
	}

	s.logger.Printf("Book uploaded: %s (%s) by %s", title, bookID, author)

	// Trigger async processing
	go s.processBookAsync(bookID, filePath, fileType)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"book_id":                  bookID,
		"title":                    title,
		"author":                   author,
		"file_type":                fileType,
		"file_size_bytes":          fileSize,
		"processing_status":        "pending",
		"estimated_processing_time": s.estimateProcessingTime(fileSize),
		"message":                  "Book uploaded successfully, processing started",
		"timestamp":                time.Now().UTC(),
	})
}

// GetBooks lists all books for a user
func (s *BookTalkService) GetBooks(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	
	query := `
		SELECT b.id, b.title, b.author, b.file_path, b.file_type, b.file_size_bytes,
		       b.total_chunks, b.total_words, b.processing_status, b.processed_at,
		       b.created_at, b.updated_at,
		       up.current_position, up.position_type, up.position_value,
		       CASE 
		           WHEN b.total_chunks > 0 THEN 
		               ROUND((COALESCE(up.current_position, 0)::NUMERIC / b.total_chunks::NUMERIC) * 100, 2)
		           ELSE 0 
		       END as percentage_complete
		FROM books b
		LEFT JOIN user_progress up ON b.id = up.book_id AND ($1 = '' OR up.user_id = $1)
		ORDER BY b.created_at DESC`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		s.httpError(w, "Failed to query books", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var books []map[string]interface{}
	for rows.Next() {
		var book Book
		var progress UserProgress
		var fileSizeBytes sql.NullInt64
		var processedAt sql.NullTime
		var currentPos, posValue sql.NullInt64
		var posType sql.NullString
		var percentComplete sql.NullFloat64

		err := rows.Scan(&book.ID, &book.Title, &book.Author, &book.FilePath, 
			&book.FileType, &fileSizeBytes, &book.TotalChunks, &book.TotalWords,
			&book.ProcessingStatus, &processedAt, &book.CreatedAt, &book.UpdatedAt,
			&currentPos, &posType, &posValue, &percentComplete)
		
		if err != nil {
			continue
		}

		bookData := map[string]interface{}{
			"id":                book.ID,
			"title":             book.Title,
			"author":            book.Author,
			"file_type":         book.FileType,
			"total_chunks":      book.TotalChunks,
			"total_words":       book.TotalWords,
			"processing_status": book.ProcessingStatus,
			"created_at":        book.CreatedAt,
			"updated_at":        book.UpdatedAt,
		}

		if fileSizeBytes.Valid {
			bookData["file_size_bytes"] = fileSizeBytes.Int64
		}
		if processedAt.Valid {
			bookData["processed_at"] = processedAt.Time
		}

		// Include progress if available
		if currentPos.Valid && userID != "" {
			bookData["user_progress"] = map[string]interface{}{
				"current_position":    currentPos.Int64,
				"position_type":       posType.String,
				"position_value":      posValue.Int64,
				"percentage_complete": percentComplete.Float64,
			}
		}

		books = append(books, bookData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// GetBook gets a specific book with user progress
func (s *BookTalkService) GetBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookIDStr := vars["book_id"]
	userID := r.URL.Query().Get("user_id")

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		s.httpError(w, "Invalid book ID", http.StatusBadRequest, err)
		return
	}

	// Get book details with progress
	var book Book
	var progress *UserProgress
	
	bookQuery := `
		SELECT id, title, author, file_path, file_type, file_size_bytes,
		       total_chunks, total_words, total_characters, processing_status, 
		       processing_error, processed_at, created_at, updated_at, metadata
		FROM books WHERE id = $1`
	
	var fileSizeBytes sql.NullInt64
	var totalChars sql.NullInt64
	var processedAt sql.NullTime
	var processingError sql.NullString
	var metadataJSON sql.NullString
	
	err = s.db.QueryRow(bookQuery, bookID).Scan(
		&book.ID, &book.Title, &book.Author, &book.FilePath, &book.FileType,
		&fileSizeBytes, &book.TotalChunks, &book.TotalWords, &totalChars,
		&book.ProcessingStatus, &processingError, &processedAt,
		&book.CreatedAt, &book.UpdatedAt, &metadataJSON)
	
	if err == sql.ErrNoRows {
		s.httpError(w, "Book not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		s.httpError(w, "Failed to get book", http.StatusInternalServerError, err)
		return
	}

	if fileSizeBytes.Valid {
		book.FileSizeBytes = fileSizeBytes.Int64
	}
	if totalChars.Valid {
		book.TotalCharacters = int(totalChars.Int64)
	}
	if processedAt.Valid {
		book.ProcessedAt = &processedAt.Time
	}
	if processingError.Valid {
		book.ProcessingError = processingError.String
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &book.Metadata)
	}

	// Get user progress if user ID provided
	if userID != "" {
		progressQuery := `
			SELECT id, current_position, position_type, position_value,
			       reading_session_count, total_reading_time_minutes, notes,
			       updated_at, created_at,
			       CASE 
			           WHEN $2 > 0 THEN 
			               ROUND((current_position::NUMERIC / $2::NUMERIC) * 100, 2)
			           ELSE 0 
			       END as percentage_complete
			FROM user_progress WHERE book_id = $1 AND user_id = $3`
		
		var prog UserProgress
		var notes sql.NullString
		
		err = s.db.QueryRow(progressQuery, bookID, book.TotalChunks, userID).Scan(
			&prog.ID, &prog.CurrentPosition, &prog.PositionType, &prog.PositionValue,
			&prog.ReadingSessionCount, &prog.TotalReadingTimeMin, &notes,
			&prog.UpdatedAt, &prog.CreatedAt, &prog.PercentageComplete)
		
		if err == nil {
			prog.BookID = bookID
			prog.UserID = userID
			if notes.Valid {
				prog.Notes = notes.String
			}
			progress = &prog
		}
	}

	response := map[string]interface{}{
		"book": book,
	}
	if progress != nil {
		response["user_progress"] = progress
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ChatWithBook handles spoiler-free chat about book content
func (s *BookTalkService) ChatWithBook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookIDStr := vars["book_id"]

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		s.httpError(w, "Invalid book ID", http.StatusBadRequest, err)
		return
	}

	var req struct {
		Message         string  `json:"message"`
		UserID          string  `json:"user_id"`
		CurrentPosition int     `json:"current_position"`
		PositionType    string  `json:"position_type"`
		Temperature     float64 `json:"temperature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.httpError(w, "Invalid JSON request", http.StatusBadRequest, err)
		return
	}

	if req.Message == "" || req.UserID == "" {
		s.httpError(w, "Missing required fields: message, user_id", http.StatusBadRequest, nil)
		return
	}

	if req.PositionType == "" {
		req.PositionType = "chunk"
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	startTime := time.Now()

	// Verify book exists and is processed
	var book Book
	err = s.db.QueryRow("SELECT id, title, author, total_chunks, processing_status FROM books WHERE id = $1", bookID).
		Scan(&book.ID, &book.Title, &book.Author, &book.TotalChunks, &book.ProcessingStatus)
	
	if err == sql.ErrNoRows {
		s.httpError(w, "Book not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		s.httpError(w, "Failed to get book", http.StatusInternalServerError, err)
		return
	}

	if book.ProcessingStatus != "completed" {
		s.httpError(w, fmt.Sprintf("Book processing not complete (status: %s)", book.ProcessingStatus), http.StatusConflict, nil)
		return
	}

	// Get safe context chunks based on position
	contextChunks, err := s.getSafeContext(bookID, req.CurrentPosition, req.Message)
	if err != nil {
		s.httpError(w, "Failed to retrieve safe context", http.StatusInternalServerError, err)
		return
	}

	// Generate AI response using Ollama
	aiResponse, sources, err := s.generateChatResponse(req.Message, contextChunks, book, req.Temperature)
	if err != nil {
		s.httpError(w, "Failed to generate response", http.StatusInternalServerError, err)
		return
	}

	processingTime := int(time.Since(startTime).Milliseconds())

	// Store conversation
	conversationID := uuid.New()
	contextChunkIDs := make([]int, len(contextChunks))
	for i, chunk := range contextChunks {
		contextChunkIDs[i] = chunk.ChunkNumber
	}

	_, err = s.db.Exec(`
		INSERT INTO conversations (id, book_id, user_id, user_message, ai_response, 
		                          user_position, position_type, context_chunks_used, 
		                          sources_referenced, position_boundary_respected, processing_time_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		conversationID, bookID, req.UserID, req.Message, aiResponse,
		req.CurrentPosition, req.PositionType, 
		toJSON(contextChunkIDs), toJSON(sources), true, processingTime)

	if err != nil {
		s.logger.Printf("Failed to store conversation: %v", err)
		// Continue anyway - response is more important than storage
	}

	s.logger.Printf("Chat response generated for book %s, user %s in %dms", book.Title, req.UserID, processingTime)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id":             conversationID,
		"response":                    aiResponse,
		"sources_used":                sources,
		"context_chunks_count":        len(contextChunks),
		"position_boundary_respected": true,
		"processing_time_ms":          processingTime,
		"user_position":               req.CurrentPosition,
		"position_type":               req.PositionType,
		"timestamp":                   time.Now().UTC(),
	})
}

// UpdateProgress updates user reading progress
func (s *BookTalkService) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookIDStr := vars["book_id"]

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		s.httpError(w, "Invalid book ID", http.StatusBadRequest, err)
		return
	}

	var req struct {
		UserID          string  `json:"user_id"`
		CurrentPosition int     `json:"current_position"`
		PositionType    string  `json:"position_type"`
		PositionValue   float64 `json:"position_value"`
		Notes           string  `json:"notes"`
		ReadingTimeMin  int     `json:"reading_time_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.httpError(w, "Invalid JSON request", http.StatusBadRequest, err)
		return
	}

	if req.UserID == "" {
		s.httpError(w, "Missing required field: user_id", http.StatusBadRequest, nil)
		return
	}

	if req.PositionType == "" {
		req.PositionType = "chunk"
	}

	// Verify book exists
	var totalChunks int
	err = s.db.QueryRow("SELECT total_chunks FROM books WHERE id = $1", bookID).Scan(&totalChunks)
	if err == sql.ErrNoRows {
		s.httpError(w, "Book not found", http.StatusNotFound, nil)
		return
	} else if err != nil {
		s.httpError(w, "Failed to verify book", http.StatusInternalServerError, err)
		return
	}

	// Upsert progress
	progressID := uuid.New()
	query := `
		INSERT INTO user_progress (id, book_id, user_id, current_position, position_type, 
		                          position_value, notes, reading_session_count, total_reading_time_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, $8)
		ON CONFLICT (book_id, user_id) 
		DO UPDATE SET 
		    current_position = EXCLUDED.current_position,
		    position_type = EXCLUDED.position_type,
		    position_value = EXCLUDED.position_value,
		    notes = EXCLUDED.notes,
		    reading_session_count = user_progress.reading_session_count + 1,
		    total_reading_time_minutes = user_progress.total_reading_time_minutes + EXCLUDED.total_reading_time_minutes,
		    updated_at = NOW()
		RETURNING id`

	err = s.db.QueryRow(query, progressID, bookID, req.UserID, req.CurrentPosition,
		req.PositionType, req.PositionValue, req.Notes, req.ReadingTimeMin).Scan(&progressID)

	if err != nil {
		s.httpError(w, "Failed to update progress", http.StatusInternalServerError, err)
		return
	}

	percentComplete := float64(0)
	if totalChunks > 0 {
		percentComplete = (float64(req.CurrentPosition) / float64(totalChunks)) * 100
	}

	s.logger.Printf("Progress updated for user %s on book %s: position %d", req.UserID, bookID, req.CurrentPosition)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"progress_id":         progressID,
		"book_id":             bookID,
		"user_id":             req.UserID,
		"current_position":    req.CurrentPosition,
		"position_type":       req.PositionType,
		"position_value":      req.PositionValue,
		"percentage_complete": percentComplete,
		"available_chunks":    req.CurrentPosition + 1, // Chunks available for safe discussion
		"total_chunks":        totalChunks,
		"updated_at":          time.Now().UTC(),
	})
}

// GetConversations gets chat history for a book and user
func (s *BookTalkService) GetConversations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bookIDStr := vars["book_id"]
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		s.httpError(w, "Missing required parameter: user_id", http.StatusBadRequest, nil)
		return
	}

	bookID, err := uuid.Parse(bookIDStr)
	if err != nil {
		s.httpError(w, "Invalid book ID", http.StatusBadRequest, err)
		return
	}

	limit := maxChatHistory
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= maxChatHistory {
			limit = l
		}
	}

	query := `
		SELECT id, user_message, ai_response, user_position, position_type,
		       context_chunks_used, sources_referenced, position_boundary_respected,
		       processing_time_ms, created_at
		FROM conversations 
		WHERE book_id = $1 AND user_id = $2 
		ORDER BY created_at DESC 
		LIMIT $3`

	rows, err := s.db.Query(query, bookID, userID, limit)
	if err != nil {
		s.httpError(w, "Failed to get conversations", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		var contextChunksJSON, sourcesJSON sql.NullString
		var processingTime sql.NullInt64

		err := rows.Scan(&conv.ID, &conv.UserMessage, &conv.AIResponse,
			&conv.UserPosition, &conv.PositionType, &contextChunksJSON,
			&sourcesJSON, &conv.PositionBoundaryRespected, &processingTime, &conv.CreatedAt)
		
		if err != nil {
			continue
		}

		conv.BookID = bookID
		conv.UserID = userID
		
		if processingTime.Valid {
			conv.ProcessingTimeMs = int(processingTime.Int64)
		}

		if contextChunksJSON.Valid {
			json.Unmarshal([]byte(contextChunksJSON.String), &conv.ContextChunksUsed)
		}
		if sourcesJSON.Valid {
			json.Unmarshal([]byte(sourcesJSON.String), &conv.SourcesReferenced)
		}

		conversations = append(conversations, conv)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversations": conversations,
		"book_id":       bookID,
		"user_id":       userID,
		"count":         len(conversations),
		"limit":         limit,
	})
}

// Helper functions

func (s *BookTalkService) httpError(w http.ResponseWriter, message string, statusCode int, err error) {
	if err != nil {
		s.logger.Printf("ERROR: %s: %v", message, err)
	} else {
		s.logger.Printf("ERROR: %s", message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(errorResp)
}

func (s *BookTalkService) estimateProcessingTime(fileSize int64) int {
	// Rough estimate: 1MB = 10 seconds
	seconds := int(fileSize / (1024 * 1024) * 10)
	if seconds < 5 {
		seconds = 5
	}
	return seconds
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// processBookAsync handles async book processing
func (s *BookTalkService) processBookAsync(bookID uuid.UUID, filePath, fileType string) {
	s.logger.Printf("Starting async processing for book %s", bookID)
	
	// Update status to processing
	s.db.Exec("UPDATE books SET processing_status = 'processing' WHERE id = $1", bookID)
	
	// TODO: Implement actual text extraction and chunking
	// For now, simulate processing
	time.Sleep(5 * time.Second)
	
	// Update as completed (in real implementation, this would process the file)
	s.db.Exec(`
		UPDATE books SET 
		    processing_status = 'completed',
		    processed_at = NOW(),
		    total_chunks = 100,
		    total_words = 50000
		WHERE id = $1`, bookID)
	
	s.logger.Printf("Completed processing for book %s", bookID)
}

// BookChunk represents a chunk of book content
type BookChunk struct {
	ChunkNumber int    `json:"chunk_number"`
	Content     string `json:"content"`
	Chapter     int    `json:"chapter,omitempty"`
}

// getSafeContext retrieves book chunks that are safe (within reading position)
func (s *BookTalkService) getSafeContext(bookID uuid.UUID, currentPosition int, query string) ([]BookChunk, error) {
	// TODO: Implement vector search with position filtering using Qdrant
	// For now, return mock chunks within the position boundary
	
	var chunks []BookChunk
	
	// Simulate retrieving relevant chunks up to current position
	maxChunks := min(5, currentPosition+1) // Get up to 5 relevant chunks within reading boundary
	
	for i := 0; i < maxChunks; i++ {
		chunks = append(chunks, BookChunk{
			ChunkNumber: i,
			Content:     fmt.Sprintf("Sample content from chunk %d (within reading boundary)", i),
			Chapter:     (i / 10) + 1,
		})
	}
	
	return chunks, nil
}

// generateChatResponse generates AI response using context chunks
func (s *BookTalkService) generateChatResponse(message string, contextChunks []BookChunk, book Book, temperature float64) (string, []string, error) {
	// TODO: Implement actual Ollama integration
	// For now, return a mock response
	
	response := fmt.Sprintf("Based on what you've read so far in \"%s\" by %s, I can discuss the content up to your current position. This is a simulated response that would normally be generated by analyzing the provided context chunks while ensuring no spoilers from future content.", book.Title, book.Author)
	
	sources := []string{
		"Chapter 1, opening passage",
		"Character introduction section",
	}
	
	return response, sources, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	// Configuration
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "20300"
		}
	}

	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/vrooli?sslmode=disable"
	}

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize service
	service := NewBookTalkService(db, n8nURL, qdrantURL, dataDir)

	// Setup routes
	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", service.Health).Methods("GET")

	// Book management
	r.HandleFunc("/api/v1/books", service.GetBooks).Methods("GET")
	r.HandleFunc("/api/v1/books/upload", service.UploadBook).Methods("POST")
	r.HandleFunc("/api/v1/books/{book_id}", service.GetBook).Methods("GET")

	// Chat and progress
	r.HandleFunc("/api/v1/books/{book_id}/chat", service.ChatWithBook).Methods("POST")
	r.HandleFunc("/api/v1/books/{book_id}/progress", service.UpdateProgress).Methods("PUT")
	r.HandleFunc("/api/v1/books/{book_id}/conversations", service.GetConversations).Methods("GET")

	// Start server
	log.Printf("Starting No Spoilers Book Talk API on port %s", port)
	log.Printf("  Database: %s", dbURL)
	log.Printf("  n8n: %s", n8nURL) 
	log.Printf("  Qdrant: %s", qdrantURL)
	log.Printf("  Data directory: %s", dataDir)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}