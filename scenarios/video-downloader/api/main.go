package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Download struct {
	ID              int     `json:"id"`
	URL             string  `json:"url"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Platform        string  `json:"platform"`
	Duration        int     `json:"duration"`
	Format          string  `json:"format"`
	Quality         string  `json:"quality"`
	// Enhanced audio support
	AudioFormat     string  `json:"audio_format,omitempty"`
	AudioQuality    string  `json:"audio_quality,omitempty"`
	AudioPath       string  `json:"audio_path,omitempty"`
	AudioFileSize   int64   `json:"audio_file_size,omitempty"`
	FilePath        string  `json:"file_path"`
	FileSize        int64   `json:"file_size"`
	Status          string  `json:"status"`
	Progress        int     `json:"progress"`
	// Enhanced transcript support
	HasTranscript       bool   `json:"has_transcript"`
	TranscriptStatus    string `json:"transcript_status,omitempty"`
	TranscriptRequested bool   `json:"transcript_requested"`
	WhisperModel        string `json:"whisper_model,omitempty"`
	TargetLanguage      string `json:"target_language,omitempty"`
	Error               string `json:"error_message"`
	CreatedAt           string `json:"created_at"`
	CompletedAt         string `json:"completed_at,omitempty"`
	TranscriptStartedAt string `json:"transcript_started_at,omitempty"`
	TranscriptCompletedAt string `json:"transcript_completed_at,omitempty"`
}

type DownloadRequest struct {
	URL             string `json:"url"`
	Quality         string `json:"quality,omitempty"`
	Format          string `json:"format,omitempty"`
	// Enhanced audio options
	AudioFormat     string `json:"audio_format,omitempty"`
	AudioQuality    string `json:"audio_quality,omitempty"`
	AudioOnly       bool   `json:"audio_only,omitempty"`
	// Transcript options
	GenerateTranscript bool   `json:"generate_transcript,omitempty"`
	WhisperModel       string `json:"whisper_model,omitempty"`
	TargetLanguage     string `json:"target_language,omitempty"`
	UserID             string `json:"user_id,omitempty"`
}

// New transcript-related data structures
type Transcript struct {
	ID                    int                  `json:"id"`
	DownloadID            int                  `json:"download_id"`
	Language              string               `json:"language"`
	DetectedLanguage      string               `json:"detected_language,omitempty"`
	ConfidenceScore       float64              `json:"confidence_score"`
	ModelUsed             string               `json:"model_used"`
	FullText              string               `json:"full_text"`
	WordCount             int                  `json:"word_count"`
	ProcessingTimeMs      int                  `json:"processing_time_ms"`
	AudioDurationSeconds  float64              `json:"audio_duration_seconds"`
	WhisperVersion        string               `json:"whisper_version,omitempty"`
	Segments              []TranscriptSegment  `json:"segments,omitempty"`
	CreatedAt             string               `json:"created_at"`
	UpdatedAt             string               `json:"updated_at"`
}

type TranscriptSegment struct {
	ID              int                    `json:"id"`
	TranscriptID    int                    `json:"transcript_id"`
	StartTime       float64                `json:"start_time"`
	EndTime         float64                `json:"end_time"`
	Text            string                 `json:"text"`
	Confidence      float64                `json:"confidence"`
	SpeakerID       string                 `json:"speaker_id,omitempty"`
	WordTimestamps  map[string]interface{} `json:"word_timestamps,omitempty"`
	Sequence        int                    `json:"sequence"`
	CharacterStart  int                    `json:"character_start"`
	CharacterEnd    int                    `json:"character_end"`
}

type TranscriptSearchRequest struct {
	Query           string `json:"query"`
	Highlight       bool   `json:"highlight,omitempty"`
	ContextSeconds  int    `json:"context_seconds,omitempty"`
}

type TranscriptSearchMatch struct {
	SegmentID       int     `json:"segment_id"`
	Text            string  `json:"text"`
	StartTime       float64 `json:"start_time"`
	EndTime         float64 `json:"end_time"`
	RelevanceScore  float64 `json:"relevance_score"`
}

type TranscriptSearchResponse struct {
	Matches      []TranscriptSearchMatch `json:"matches"`
	TotalMatches int                     `json:"total_matches"`
}

type TranscriptExportRequest struct {
	Format             string `json:"format"` // srt, vtt, txt, json
	IncludeTimestamps  bool   `json:"include_timestamps,omitempty"`
	IncludeConfidence  bool   `json:"include_confidence,omitempty"`
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
	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}
	
	var err error
	db, err = sql.Open("postgres", connStr)
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
	log.Printf("📹 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
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

	// Validate required URL
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Set defaults for optional fields
	if req.Format == "" {
		req.Format = "mp4"
	}
	if req.Quality == "" {
		req.Quality = "best"
	}
	if req.AudioFormat == "" && req.AudioOnly {
		req.AudioFormat = "mp3"
	}
	if req.AudioQuality == "" && req.AudioOnly {
		req.AudioQuality = "192k"
	}
	if req.WhisperModel == "" && req.GenerateTranscript {
		req.WhisperModel = "base"
	}

	// Insert enhanced download record
	var downloadID int
	err := db.QueryRow(`
		INSERT INTO downloads (
			url, title, format, quality, 
			audio_format, audio_quality,
			status, transcript_requested, whisper_model, target_language,
			user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7, $8, $9, $10)
		RETURNING id`,
		req.URL, "Processing...", req.Format, req.Quality,
		req.AudioFormat, req.AudioQuality,
		req.GenerateTranscript, req.WhisperModel, req.TargetLanguage,
		req.UserID).Scan(&downloadID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add to queue with higher priority if transcript is requested
	priority := 1
	if req.GenerateTranscript {
		priority = 2 // Higher priority for transcript jobs
	}
	
	_, err = db.Exec(`
		INSERT INTO download_queue (download_id, position, priority)
		VALUES ($1, (SELECT COALESCE(MAX(position), 0) + 1 FROM download_queue), $2)`,
		downloadID, priority)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Estimate completion time based on transcript request
	estimatedMinutes := 5 // Base download time
	if req.GenerateTranscript {
		estimatedMinutes += 10 // Additional time for transcript
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"download_id": downloadID,
		"status": "queued",
		"estimated_completion_minutes": estimatedMinutes,
		"transcript_requested": req.GenerateTranscript,
		"audio_format": req.AudioFormat,
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

// Enhanced transcript handler functions
func getTranscriptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	downloadID, err := strconv.Atoi(vars["download_id"])
	if err != nil {
		http.Error(w, "Invalid download ID", http.StatusBadRequest)
		return
	}

	// Get query parameters
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	includeSegments := r.URL.Query().Get("include_segments") != "false"

	// Query transcript with optional segments
	var transcript Transcript
	err = db.QueryRow(`
		SELECT id, download_id, language, detected_language, confidence_score, 
			   model_used, full_text, word_count, processing_time_ms, 
			   audio_duration_seconds, whisper_version, created_at, updated_at
		FROM transcripts WHERE download_id = $1`, downloadID).Scan(
		&transcript.ID, &transcript.DownloadID, &transcript.Language, 
		&transcript.DetectedLanguage, &transcript.ConfidenceScore,
		&transcript.ModelUsed, &transcript.FullText, &transcript.WordCount,
		&transcript.ProcessingTimeMs, &transcript.AudioDurationSeconds,
		&transcript.WhisperVersion, &transcript.CreatedAt, &transcript.UpdatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Transcript not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Include segments if requested
	if includeSegments {
		rows, err := db.Query(`
			SELECT id, start_time, end_time, text, confidence, speaker_id,
				   word_timestamps, sequence, character_start, character_end
			FROM transcript_segments 
			WHERE transcript_id = $1 
			ORDER BY sequence`, transcript.ID)
		
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var segment TranscriptSegment
				var wordTimestampsJSON []byte
				err := rows.Scan(&segment.ID, &segment.StartTime, &segment.EndTime,
					&segment.Text, &segment.Confidence, &segment.SpeakerID,
					&wordTimestampsJSON, &segment.Sequence, 
					&segment.CharacterStart, &segment.CharacterEnd)
				if err == nil {
					segment.TranscriptID = transcript.ID
					if len(wordTimestampsJSON) > 0 {
						json.Unmarshal(wordTimestampsJSON, &segment.WordTimestamps)
					}
					transcript.Segments = append(transcript.Segments, segment)
				}
			}
		}
	}

	// Handle different export formats
	if format != "json" {
		exportPath, err := exportTranscript(transcript, format, true, false)
		if err != nil {
			http.Error(w, "Export failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"transcript": transcript,
			"export_url": "/api/transcript/export/" + exportPath,
			"format": format,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transcript)
}

func searchTranscriptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	downloadID, err := strconv.Atoi(vars["download_id"])
	if err != nil {
		http.Error(w, "Invalid download ID", http.StatusBadRequest)
		return
	}

	var req TranscriptSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	// Get transcript ID for this download
	var transcriptID int
	err = db.QueryRow("SELECT id FROM transcripts WHERE download_id = $1", downloadID).Scan(&transcriptID)
	if err == sql.ErrNoRows {
		http.Error(w, "Transcript not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Perform full-text search on transcript segments
	query := `
		SELECT ts.id, ts.start_time, ts.end_time, ts.text,
			   ts_rank(to_tsvector('english', ts.text), plainto_tsquery('english', $2)) as relevance
		FROM transcript_segments ts
		WHERE ts.transcript_id = $1 
		  AND to_tsvector('english', ts.text) @@ plainto_tsquery('english', $2)
		ORDER BY relevance DESC
		LIMIT 20`

	rows, err := db.Query(query, transcriptID, req.Query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var matches []TranscriptSearchMatch
	for rows.Next() {
		var match TranscriptSearchMatch
		err := rows.Scan(&match.SegmentID, &match.StartTime, &match.EndTime,
			&match.Text, &match.RelevanceScore)
		if err != nil {
			continue
		}
		matches = append(matches, match)
	}

	response := TranscriptSearchResponse{
		Matches:      matches,
		TotalMatches: len(matches),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateTranscriptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	downloadID, err := strconv.Atoi(vars["download_id"])
	if err != nil {
		http.Error(w, "Invalid download ID", http.StatusBadRequest)
		return
	}

	// Check if download exists and has audio
	var audioPath, whisperModel string
	var hasAudio bool
	err = db.QueryRow(`
		SELECT COALESCE(audio_path, '') != '', whisper_model 
		FROM downloads WHERE id = $1`, downloadID).Scan(&hasAudio, &whisperModel)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Download not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !hasAudio {
		http.Error(w, "No audio file available for transcription", http.StatusBadRequest)
		return
	}

	// Update transcript status to processing
	_, err = db.Exec(`
		UPDATE downloads 
		SET transcript_status = 'processing', transcript_started_at = NOW()
		WHERE id = $1`, downloadID)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Trigger n8n workflow for transcript generation
	// For now, return processing status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"download_id": downloadID,
		"status": "processing",
		"message": "Transcript generation started",
		"whisper_model": whisperModel,
	})
}

// Utility function to export transcripts in different formats
func exportTranscript(transcript Transcript, format string, includeTimestamps, includeConfidence bool) (string, error) {
	// TODO: Implement actual export functionality
	// This would generate SRT, VTT, or plain text files
	filename := fmt.Sprintf("transcript_%d.%s", transcript.ID, format)
	return filename, nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start video-downloader

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

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
	
	// Enhanced transcript API routes
	router.HandleFunc("/api/transcript/{download_id}", getTranscriptHandler).Methods("GET")
	router.HandleFunc("/api/transcript/{download_id}/search", searchTranscriptHandler).Methods("POST")
	router.HandleFunc("/api/transcript/{download_id}/generate", generateTranscriptHandler).Methods("POST")

	// Enable CORS with enhanced method support
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("Video Downloader API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

// getEnv function removed - no hardcoded defaults for configuration
