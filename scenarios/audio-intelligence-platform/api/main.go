package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	// API version
	apiVersion  = "1.0.0"
	serviceName = "audio-intelligence-platform"

	// Defaults
	defaultPort      = "8090"
	defaultWorkspace = "demo"

	// Timeouts
	httpTimeout    = 60 * time.Second
	discoveryDelay = 5 * time.Second

	// Database limits
	maxDBConnections   = 25
	maxIdleConnections = 5
	connMaxLifetime    = 5 * time.Minute

	// File limits
	maxFileSize = 500 * 1024 * 1024 // 500MB
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
}

// NewLogger creates a structured logger
func NewLogger() *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "[audio-intelligence-platform-api] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *Logger) Error(msg string, err error) {
	l.Printf("ERROR: %s: %v", msg, err)
}

func (l *Logger) Warn(msg string, err error) {
	l.Printf("WARN: %s: %v", msg, err)
}

func (l *Logger) Info(msg string) {
	l.Printf("INFO: %s", msg)
}

// HTTPError sends structured error response
func HTTPError(w http.ResponseWriter, message string, statusCode int, err error) {
	logger := NewLogger()
	logger.Error(fmt.Sprintf("HTTP %d: %s", statusCode, message), err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now().UTC(),
	}

	json.NewEncoder(w).Encode(errorResp)
}

// Transcription represents a transcription record
type Transcription struct {
	ID                uuid.UUID `json:"id"`
	Filename          string    `json:"filename"`
	FilePath          string    `json:"file_path"`
	TranscriptionText string    `json:"transcription_text"`
	DurationSeconds   float64   `json:"duration_seconds"`
	FileSizeBytes     int64     `json:"file_size_bytes"`
	WhisperModelUsed  string    `json:"whisper_model_used"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	EmbeddingStatus   string    `json:"embedding_status"`
}

// AIAnalysis represents an AI analysis result
type AIAnalysis struct {
	ID               uuid.UUID `json:"id"`
	TranscriptionID  uuid.UUID `json:"transcription_id"`
	AnalysisType     string    `json:"analysis_type"`
	PromptUsed       string    `json:"prompt_used"`
	ResultText       string    `json:"result_text"`
	CreatedAt        time.Time `json:"created_at"`
	ProcessingTimeMS int       `json:"processing_time_ms"`
}

// AudioService handles audio intelligence operations
type AudioService struct {
	db            *sql.DB
	n8nBaseURL    string
	windmillURL   string
	whisperURL    string
	ollamaURL     string
	minioEndpoint string
	qdrantURL     string
	httpClient    *http.Client
	logger        *Logger
}

// NewAudioService creates a new audio service
func NewAudioService(db *sql.DB, n8nURL, windmillURL, whisperURL, ollamaURL, minioEndpoint, qdrantURL string) *AudioService {
	return &AudioService{
		db:            db,
		n8nBaseURL:    n8nURL,
		windmillURL:   windmillURL,
		whisperURL:    whisperURL,
		ollamaURL:     ollamaURL,
		minioEndpoint: minioEndpoint,
		qdrantURL:     qdrantURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
		logger: NewLogger(),
	}
}

// ListTranscriptions returns all transcriptions
func (a *AudioService) ListTranscriptions(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`
		SELECT id, filename, file_path, transcription_text, duration_seconds, 
		       file_size_bytes, whisper_model_used, created_at, updated_at, embedding_status
		FROM transcriptions
		ORDER BY created_at DESC`)
	if err != nil {
		HTTPError(w, "Failed to query transcriptions", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var transcriptions []Transcription
	for rows.Next() {
		var t Transcription
		err := rows.Scan(&t.ID, &t.Filename, &t.FilePath, &t.TranscriptionText,
			&t.DurationSeconds, &t.FileSizeBytes, &t.WhisperModelUsed,
			&t.CreatedAt, &t.UpdatedAt, &t.EmbeddingStatus)
		if err != nil {
			continue
		}
		transcriptions = append(transcriptions, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transcriptions)
}

// GetTranscription returns a specific transcription
func (a *AudioService) GetTranscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptionID := vars["id"]

	var t Transcription
	err := a.db.QueryRow(`
		SELECT id, filename, file_path, transcription_text, duration_seconds, 
		       file_size_bytes, whisper_model_used, created_at, updated_at, embedding_status
		FROM transcriptions 
		WHERE id = $1`, transcriptionID).Scan(
		&t.ID, &t.Filename, &t.FilePath, &t.TranscriptionText,
		&t.DurationSeconds, &t.FileSizeBytes, &t.WhisperModelUsed,
		&t.CreatedAt, &t.UpdatedAt, &t.EmbeddingStatus)

	if err == sql.ErrNoRows {
		HTTPError(w, "Transcription not found", http.StatusNotFound, err)
		return
	}
	if err != nil {
		HTTPError(w, "Failed to query transcription", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

// UploadAudio handles audio file upload
func (a *AudioService) UploadAudio(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(maxFileSize)
	if err != nil {
		HTTPError(w, "Failed to parse form", http.StatusBadRequest, err)
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		HTTPError(w, "Failed to get audio file", http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	// Validate file type
	allowedTypes := []string{".mp3", ".wav", ".m4a", ".ogg", ".flac"}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	isAllowed := false
	for _, allowedType := range allowedTypes {
		if ext == allowedType {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		HTTPError(w, "Unsupported file type", http.StatusBadRequest, fmt.Errorf("file type %s not allowed", ext))
		return
	}

	// Trigger n8n transcription pipeline
	transcriptionID := uuid.New()
	payload := map[string]interface{}{
		"transcription_id": transcriptionID.String(),
		"filename":         header.Filename,
		"file_size":        header.Size,
		"timestamp":        time.Now().UTC(),
	}

	// Create a multipart request for n8n
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file
	fileWriter, err := writer.CreateFormFile("audio", header.Filename)
	if err != nil {
		HTTPError(w, "Failed to create form file", http.StatusInternalServerError, err)
		return
	}

	// Reset file pointer and copy
	file.Seek(0, 0)
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		HTTPError(w, "Failed to copy file", http.StatusInternalServerError, err)
		return
	}

	// Add metadata fields
	for key, value := range payload {
		writer.WriteField(key, fmt.Sprintf("%v", value))
	}

	writer.Close()

	// Send to n8n transcription pipeline
	webhookURL := fmt.Sprintf("%s/webhook/transcription-upload", a.n8nBaseURL)
	req, err := http.NewRequest("POST", webhookURL, &buf)
	if err != nil {
		HTTPError(w, "Failed to create request", http.StatusInternalServerError, err)
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		HTTPError(w, "Failed to send to transcription pipeline", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Transcription pipeline failed", resp.StatusCode, fmt.Errorf("n8n returned status %d", resp.StatusCode))
		return
	}

	// Return upload response
	response := map[string]interface{}{
		"transcription_id": transcriptionID.String(),
		"filename":         header.Filename,
		"file_size":        header.Size,
		"status":           "uploaded",
		"message":          "File uploaded successfully, transcription started",
		"timestamp":        time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// AnalyzeTranscription handles AI analysis requests
func (a *AudioService) AnalyzeTranscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptionID := vars["id"]

	var req struct {
		AnalysisType string `json:"analysis_type"`
		CustomPrompt string `json:"custom_prompt,omitempty"`
		Model        string `json:"model,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.AnalysisType == "" {
		HTTPError(w, "Missing required field: analysis_type", http.StatusBadRequest, nil)
		return
	}

	// Get transcription text
	var transcriptionText string
	err := a.db.QueryRow("SELECT transcription_text FROM transcriptions WHERE id = $1", transcriptionID).Scan(&transcriptionText)
	if err == sql.ErrNoRows {
		HTTPError(w, "Transcription not found", http.StatusNotFound, err)
		return
	}
	if err != nil {
		HTTPError(w, "Failed to query transcription", http.StatusInternalServerError, err)
		return
	}

	// Build analysis payload
	payload := map[string]interface{}{
		"transcription_id":   transcriptionID,
		"transcription_text": transcriptionText,
		"analysis_type":      req.AnalysisType,
		"custom_prompt":      req.CustomPrompt,
		"model":              req.Model,
		"timestamp":          time.Now().UTC(),
	}

	if req.Model == "" {
		payload["model"] = "llama3.1:8b"
	}

	// Send to n8n AI analysis workflow
	payloadBytes, _ := json.Marshal(payload)
	webhookURL := fmt.Sprintf("%s/webhook/ai-analysis", a.n8nBaseURL)

	startTime := time.Now()
	resp, err := a.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		HTTPError(w, "Failed to send to analysis pipeline", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Analysis pipeline failed", resp.StatusCode, fmt.Errorf("n8n returned status %d", resp.StatusCode))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		HTTPError(w, "Failed to decode analysis result", http.StatusInternalServerError, err)
		return
	}

	// Store analysis result in database
	analysisID := uuid.New()
	processingTime := int(time.Since(startTime).Milliseconds())

	prompt := req.CustomPrompt
	if prompt == "" {
		switch req.AnalysisType {
		case "summary":
			prompt = "Provide a concise summary of this transcription"
		case "insights":
			prompt = "Extract key insights and main points"
		default:
			prompt = fmt.Sprintf("Perform %s analysis", req.AnalysisType)
		}
	}

	resultText := ""
	if analysis, ok := result["analysis"].(string); ok {
		resultText = analysis
	} else if text, ok := result["result"].(string); ok {
		resultText = text
	} else {
		resultText = fmt.Sprintf("%v", result)
	}

	_, err = a.db.Exec(`
		INSERT INTO ai_analyses (id, transcription_id, analysis_type, prompt_used, result_text, processing_time_ms)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		analysisID, transcriptionID, req.AnalysisType, prompt, resultText, processingTime)

	if err != nil {
		a.logger.Error("Failed to store analysis result", err)
	}

	// Return analysis response
	response := map[string]interface{}{
		"analysis_id":        analysisID.String(),
		"transcription_id":   transcriptionID,
		"analysis_type":      req.AnalysisType,
		"result":             resultText,
		"processing_time_ms": processingTime,
		"model_used":         payload["model"],
		"timestamp":          time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SearchTranscriptions handles semantic search
func (a *AudioService) SearchTranscriptions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid JSON request body", http.StatusBadRequest, err)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	// Send to n8n semantic search workflow
	payload := map[string]interface{}{
		"query":     req.Query,
		"limit":     req.Limit,
		"timestamp": time.Now().UTC(),
	}

	payloadBytes, _ := json.Marshal(payload)
	webhookURL := fmt.Sprintf("%s/webhook/semantic-search", a.n8nBaseURL)

	resp, err := a.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		HTTPError(w, "Failed to send to search pipeline", http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		HTTPError(w, "Search pipeline failed", resp.StatusCode, fmt.Errorf("n8n returned status %d", resp.StatusCode))
		return
	}

	var searchResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		HTTPError(w, "Failed to decode search result", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResult)
}

// GetAnalyses returns analyses for a transcription
func (a *AudioService) GetAnalyses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transcriptionID := vars["id"]

	rows, err := a.db.Query(`
		SELECT id, transcription_id, analysis_type, prompt_used, result_text, created_at, processing_time_ms
		FROM ai_analyses
		WHERE transcription_id = $1
		ORDER BY created_at DESC`, transcriptionID)
	if err != nil {
		HTTPError(w, "Failed to query analyses", http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	var analyses []AIAnalysis
	for rows.Next() {
		var a AIAnalysis
		err := rows.Scan(&a.ID, &a.TranscriptionID, &a.AnalysisType,
			&a.PromptUsed, &a.ResultText, &a.CreatedAt, &a.ProcessingTimeMS)
		if err != nil {
			continue
		}
		analyses = append(analyses, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analyses)
}

// Health endpoint
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": serviceName,
		"version": apiVersion,
	})
}

// getResourcePort queries the port registry for a resource's port
func getResourcePort(resourceName string) string {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(
		"source ${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh && ports::get_resource_port %s",
		resourceName,
	))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to get port for %s, using default: %v", resourceName, err)
		// Fallback to defaults
		defaults := map[string]string{
			"n8n":      "5678",
			"windmill": "5681",
			"postgres": "5433",
			"whisper":  "8090",
			"ollama":   "11434",
			"minio":    "9000",
			"qdrant":   "6333",
		}
		if port, ok := defaults[resourceName]; ok {
			return port
		}
		return "8080" // Generic fallback
	}
	return strings.TrimSpace(string(output))
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start audio-intelligence-platform

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

	// Use port registry for resource ports
	n8nPort := getResourcePort("n8n")
	windmillPort := getResourcePort("windmill")
	whisperPort := getResourcePort("whisper")
	ollamaPort := getResourcePort("ollama")
	minioPort := getResourcePort("minio")
	qdrantPort := getResourcePort("qdrant")

	// Build URLs from environment or defaults
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = fmt.Sprintf("http://localhost:%s", n8nPort)
	}

	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	if windmillURL == "" {
		windmillURL = fmt.Sprintf("http://localhost:%s", windmillPort)
	}

	whisperURL := os.Getenv("WHISPER_BASE_URL")
	if whisperURL == "" {
		whisperURL = fmt.Sprintf("http://localhost:%s", whisperPort)
	}

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = fmt.Sprintf("http://localhost:%s", ollamaPort)
	}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = fmt.Sprintf("localhost:%s", minioPort)
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
	}

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

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger := NewLogger()
		logger.Error("Failed to connect to database", err)
		os.Exit(1)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(maxDBConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connMaxLifetime)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("🎵 Database URL configured")

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

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
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
		logger := NewLogger()
		logger.Error(fmt.Sprintf("❌ Database connection failed after %d attempts", maxRetries), pingErr)
		os.Exit(1)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize audio service
	audioService := NewAudioService(db, n8nURL, windmillURL, whisperURL, ollamaURL, minioEndpoint, qdrantURL)

	// Setup routes
	r := mux.NewRouter()

	// API endpoints
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/transcriptions", audioService.ListTranscriptions).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}", audioService.GetTranscription).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}/analyses", audioService.GetAnalyses).Methods("GET")
	r.HandleFunc("/api/transcriptions/{id}/analyze", audioService.AnalyzeTranscription).Methods("POST")
	r.HandleFunc("/api/upload", audioService.UploadAudio).Methods("POST")
	r.HandleFunc("/api/search", audioService.SearchTranscriptions).Methods("POST")

	// Start server
	log.Printf("Starting Audio Intelligence Platform API on port %s", port)
	log.Printf("  n8n URL: %s", n8nURL)
	log.Printf("  Windmill URL: %s", windmillURL)
	log.Printf("  Whisper URL: %s", whisperURL)
	log.Printf("  Ollama URL: %s", ollamaURL)
	log.Printf("  MinIO Endpoint: %s", minioEndpoint)
	log.Printf("  Qdrant URL: %s", qdrantURL)
	log.Printf("  Database: %s", dbURL)

	logger := NewLogger()
	logger.Info(fmt.Sprintf("Server starting on port %s", port))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("Server failed", err)
		os.Exit(1)
	}
}
