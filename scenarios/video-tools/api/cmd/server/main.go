package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/video-tools/internal/video"
)

// Logger provides structured logging
type Logger struct {
	prefix string
}

// Info logs an informational message with structured fields
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log("INFO", msg, fields)
}

// Error logs an error message with structured fields
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log("ERROR", msg, fields)
}

// Warn logs a warning message with structured fields
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log("WARN", msg, fields)
}

func (l *Logger) log(level, msg string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}
	if l.prefix != "" {
		entry["service"] = l.prefix
	}
	for k, v := range fields {
		entry[k] = v
	}
	data, _ := json.Marshal(entry)
	log.Println(string(data))
}

func newLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

var logger = newLogger("video-tools-api")

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	WorkDir     string
	APIToken    string
}

// Server holds server dependencies
type Server struct {
	config    *Config
	db        *sql.DB
	router    *mux.Router
	processor *video.Processor
}

// Response is a generic API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// VideoUploadRequest represents video upload payload
type VideoUploadRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	AutoAnalyze bool     `json:"auto_analyze"`
	URL         string   `json:"url,omitempty"`
}

// VideoEditRequest represents video editing operations
type VideoEditRequest struct {
	Operations   []video.EditOperation `json:"operations"`
	OutputFormat string                `json:"output_format,omitempty"`
	Quality      string                `json:"quality,omitempty"`
}

// VideoConvertRequest represents format conversion request
type VideoConvertRequest struct {
	TargetFormat string `json:"target_format"`
	Resolution   string `json:"resolution,omitempty"`
	Quality      string `json:"quality,omitempty"`
	Preset       string `json:"preset,omitempty"`
	Compression  struct {
		CRF    int    `json:"crf,omitempty"`
		Preset string `json:"preset,omitempty"`
	} `json:"compression,omitempty"`
	AudioSettings struct {
		Codec   string `json:"codec,omitempty"`
		Bitrate string `json:"bitrate_kbps,omitempty"`
	} `json:"audio_settings,omitempty"`
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	config := &Config{
		Port:        getEnv("API_PORT", "18125"),
		DatabaseURL: getEnv("DATABASE_URL", fmt.Sprintf("postgres://vrooli:%s@localhost:5433/video_tools?sslmode=disable", getEnv("POSTGRES_PASSWORD", ""))),
		WorkDir:     getEnv("WORK_DIR", "/tmp/video-tools"),
		APIToken:    getEnv("API_TOKEN", "video-tools-secret-token"),
	}

	// Create work directory
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create video processor
	processor, err := video.NewProcessor(filepath.Join(config.WorkDir, "processing"))
	if err != nil {
		return nil, fmt.Errorf("failed to create video processor: %w", err)
	}

	server := &Server{
		config:    config,
		db:        db,
		router:    mux.NewRouter(),
		processor: processor,
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
	s.router.Use(s.authMiddleware)

	// Health check (no auth)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Video management
	api.HandleFunc("/video/upload", s.handleVideoUpload).Methods("POST")
	api.HandleFunc("/video/{id}", s.handleGetVideo).Methods("GET")
	api.HandleFunc("/video/{id}/info", s.handleGetVideoInfo).Methods("GET")
	api.HandleFunc("/video/{id}/edit", s.handleVideoEdit).Methods("POST")
	api.HandleFunc("/video/{id}/convert", s.handleVideoConvert).Methods("POST")
	api.HandleFunc("/video/{id}/frames", s.handleExtractFrames).Methods("GET", "POST")
	api.HandleFunc("/video/{id}/thumbnail", s.handleGenerateThumbnail).Methods("POST")
	api.HandleFunc("/video/{id}/audio", s.handleExtractAudio).Methods("POST")
	api.HandleFunc("/video/{id}/subtitles", s.handleAddSubtitles).Methods("POST")
	api.HandleFunc("/video/{id}/compress", s.handleCompress).Methods("POST")
	api.HandleFunc("/video/{id}/analyze", s.handleAnalyze).Methods("POST")

	// Processing jobs
	api.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.handleGetJob).Methods("GET")
	api.HandleFunc("/jobs/{id}/cancel", s.handleCancelJob).Methods("POST")

	// Streaming
	api.HandleFunc("/stream/create", s.handleCreateStream).Methods("POST")
	api.HandleFunc("/stream/{id}/start", s.handleStartStream).Methods("POST")
	api.HandleFunc("/stream/{id}/stop", s.handleStopStream).Methods("POST")
	api.HandleFunc("/streams", s.handleListStreams).Methods("GET")

	// Documentation
	s.router.HandleFunc("/docs", s.handleDocs).Methods("GET")
	s.router.HandleFunc("/api/status", s.handleStatus).Methods("GET")
}

// Middleware functions
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http_request", map[string]interface{}{
			"method":      r.Method,
			"uri":         r.RequestURI,
			"remote_addr": r.RemoteAddr,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment, default to localhost for development
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:35000,http://localhost:3000"
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			// Check if origin is in allowed list
			for _, allowed := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(allowed) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and docs
		if r.URL.Path == "/health" || r.URL.Path == "/api/health" || r.URL.Path == "/docs" || r.URL.Path == "/api/status" {
			next.ServeHTTP(w, r)
			return
		}

		// Check authorization header
		token := r.Header.Get("Authorization")
		if token == "" || token != "Bearer "+s.config.APIToken {
			s.sendError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler functions
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "video-tools API",
		"version":   "1.0.0",
	}

	// Check database connection
	if err := s.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "disconnected"
	} else {
		health["database"] = "connected"
	}

	// Check ffmpeg availability
	if _, err := os.Stat("/usr/bin/ffmpeg"); err != nil {
		health["ffmpeg"] = "not available"
	} else {
		health["ffmpeg"] = "available"
	}

	s.sendJSON(w, http.StatusOK, health)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	// Get video statistics
	var stats struct {
		TotalVideos   int `json:"total_videos"`
		TotalJobs     int `json:"total_jobs"`
		ActiveStreams int `json:"active_streams"`
	}

	s.db.QueryRow(`
		SELECT 
			COUNT(DISTINCT va.id) as total_videos,
			COUNT(DISTINCT pj.id) as total_jobs,
			COUNT(DISTINCT ss.id) FILTER (WHERE ss.is_active = true) as active_streams
		FROM video_assets va
		LEFT JOIN processing_jobs pj ON va.id = pj.video_id
		LEFT JOIN streaming_sessions ss ON va.id = ss.video_id
	`).Scan(&stats.TotalVideos, &stats.TotalJobs, &stats.ActiveStreams)

	status := map[string]interface{}{
		"status":     "operational",
		"version":    "1.0.0",
		"uptime":     time.Now().Unix(),
		"statistics": stats,
		"capabilities": []string{
			"video-upload",
			"format-conversion",
			"frame-extraction",
			"thumbnail-generation",
			"audio-extraction",
			"subtitle-support",
			"compression",
			"streaming",
		},
	}

	s.sendJSON(w, http.StatusOK, status)
}

func (s *Server) handleVideoUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		s.sendError(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()

	// Generate unique ID
	videoID := uuid.New().String()

	// Save file temporarily
	tempPath := filepath.Join(s.config.WorkDir, "uploads", videoID+filepath.Ext(header.Filename))
	os.MkdirAll(filepath.Dir(tempPath), 0755)

	dst, err := os.Create(tempPath)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	// Get video info
	info, err := s.processor.GetVideoInfo(tempPath)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to analyze video")
		return
	}

	// Insert into database
	query := `INSERT INTO video_assets (
		id, name, description, format, duration_seconds, 
		resolution_width, resolution_height, frame_rate, 
		file_size_bytes, codec, bitrate_kbps, has_audio, 
		audio_channels, status
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'ready')`

	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}
	description := r.FormValue("description")

	_, err = s.db.Exec(query,
		videoID, name, description, info.Format, info.Duration,
		info.Width, info.Height, info.FrameRate,
		info.FileSize, info.Codec, info.BitRate, info.HasAudio,
		info.AudioChannels,
	)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to save video metadata")
		return
	}

	// Generate thumbnail
	thumbnailPath := filepath.Join(s.config.WorkDir, "thumbnails", videoID+".jpg")
	os.MkdirAll(filepath.Dir(thumbnailPath), 0755)
	s.processor.GenerateThumbnail(tempPath, thumbnailPath, info.Duration/2, 320)

	// Return response
	s.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"video_id":      videoID,
		"upload_status": "completed",
		"metadata": map[string]interface{}{
			"duration":   info.Duration,
			"resolution": fmt.Sprintf("%dx%d", info.Width, info.Height),
			"format":     info.Format,
			"size_bytes": info.FileSize,
		},
	})
}

func (s *Server) handleGetVideo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	query := `SELECT id, name, description, format, duration_seconds, 
		resolution_width, resolution_height, frame_rate, file_size_bytes, 
		codec, bitrate_kbps, has_audio, audio_channels, status, created_at
		FROM video_assets WHERE id = $1`

	var video struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		Description   *string   `json:"description"`
		Format        string    `json:"format"`
		Duration      float64   `json:"duration_seconds"`
		Width         int       `json:"resolution_width"`
		Height        int       `json:"resolution_height"`
		FrameRate     float64   `json:"frame_rate"`
		FileSize      int64     `json:"file_size_bytes"`
		Codec         string    `json:"codec"`
		Bitrate       int       `json:"bitrate_kbps"`
		HasAudio      bool      `json:"has_audio"`
		AudioChannels *int      `json:"audio_channels"`
		Status        string    `json:"status"`
		CreatedAt     time.Time `json:"created_at"`
	}

	err := s.db.QueryRow(query, videoID).Scan(
		&video.ID, &video.Name, &video.Description, &video.Format, &video.Duration,
		&video.Width, &video.Height, &video.FrameRate, &video.FileSize,
		&video.Codec, &video.Bitrate, &video.HasAudio, &video.AudioChannels,
		&video.Status, &video.CreatedAt,
	)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "video not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query video")
		return
	}

	s.sendJSON(w, http.StatusOK, video)
}

func (s *Server) handleGetVideoInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	// Get video file path
	videoPath := filepath.Join(s.config.WorkDir, "uploads", videoID+".*")
	matches, _ := filepath.Glob(videoPath)
	if len(matches) == 0 {
		s.sendError(w, http.StatusNotFound, "video file not found")
		return
	}

	info, err := s.processor.GetVideoInfo(matches[0])
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to get video info")
		return
	}

	s.sendJSON(w, http.StatusOK, info)
}

func (s *Server) handleVideoEdit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	var req VideoEditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create processing job
	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, parameters, status) 
		VALUES ($1, $2, 'edit', $3, 'pending')`

	params, _ := json.Marshal(req)
	_, err := s.db.Exec(query, jobID, videoID, params)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	// TODO: Process edits asynchronously
	// For now, return job ID
	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id":                jobID,
		"estimated_duration_ms": 5000,
		"status":                "pending",
	})
}

func (s *Server) handleVideoConvert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	var req VideoConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create processing job
	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, parameters, status) 
		VALUES ($1, $2, 'convert', $3, 'pending')`

	params, _ := json.Marshal(req)
	_, err := s.db.Exec(query, jobID, videoID, params)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	})
}

func (s *Server) handleExtractFrames(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	// Get video file path
	videoPath := filepath.Join(s.config.WorkDir, "uploads", videoID+".*")
	matches, _ := filepath.Glob(videoPath)
	if len(matches) == 0 {
		s.sendError(w, http.StatusNotFound, "video file not found")
		return
	}

	// Parse timestamps from query params
	timestamps := r.URL.Query().Get("timestamps")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "jpg"
	}

	// Extract frames
	outputDir := filepath.Join(s.config.WorkDir, "frames", videoID)
	options := video.FrameExtractionOptions{
		Format: format,
	}

	// TODO: Parse timestamps
	if timestamps != "" {
		// Parse comma-separated timestamps
		options.Timestamps = []float64{10, 20, 30} // Example
	}

	frames, err := s.processor.ExtractFrames(matches[0], outputDir, options)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to extract frames")
		return
	}

	// Build response
	var frameData []map[string]interface{}
	for i, frame := range frames {
		frameData = append(frameData, map[string]interface{}{
			"timestamp": options.Timestamps[i],
			"path":      frame,
			"format":    format,
		})
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"frames": frameData,
	})
}

func (s *Server) handleGenerateThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	// Get video file path
	videoPath := filepath.Join(s.config.WorkDir, "uploads", videoID+".*")
	matches, _ := filepath.Glob(videoPath)
	if len(matches) == 0 {
		s.sendError(w, http.StatusNotFound, "video file not found")
		return
	}

	// Generate thumbnail
	thumbnailPath := filepath.Join(s.config.WorkDir, "thumbnails", videoID+".jpg")
	os.MkdirAll(filepath.Dir(thumbnailPath), 0755)

	// Get video info for duration
	info, _ := s.processor.GetVideoInfo(matches[0])
	timestamp := info.Duration / 2 // Middle of video

	err := s.processor.GenerateThumbnail(matches[0], thumbnailPath, timestamp, 320)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to generate thumbnail")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"thumbnail_path": thumbnailPath,
		"timestamp":      timestamp,
	})
}

func (s *Server) handleExtractAudio(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	// Get video file path
	videoPath := filepath.Join(s.config.WorkDir, "uploads", videoID+".*")
	matches, _ := filepath.Glob(videoPath)
	if len(matches) == 0 {
		s.sendError(w, http.StatusNotFound, "video file not found")
		return
	}

	// Extract audio
	audioPath := filepath.Join(s.config.WorkDir, "audio", videoID+".mp3")
	os.MkdirAll(filepath.Dir(audioPath), 0755)

	err := s.processor.ExtractAudio(matches[0], audioPath)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to extract audio")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"audio_path": audioPath,
		"format":     "mp3",
	})
}

func (s *Server) handleAddSubtitles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	var req struct {
		SubtitlePath string `json:"subtitle_path"`
		BurnIn       bool   `json:"burn_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create processing job
	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, parameters, status) 
		VALUES ($1, $2, 'edit', $3, 'pending')`

	params, _ := json.Marshal(req)
	_, err := s.db.Exec(query, jobID, videoID, params)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	})
}

func (s *Server) handleCompress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	var req struct {
		TargetSizeMB int `json:"target_size_mb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create processing job
	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, parameters, status) 
		VALUES ($1, $2, 'compress', $3, 'pending')`

	params, _ := json.Marshal(req)
	_, err := s.db.Exec(query, jobID, videoID, params)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id": jobID,
		"status": "pending",
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["id"]

	var req struct {
		AnalysisTypes []string               `json:"analysis_types"`
		Options       map[string]interface{} `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create processing job
	jobID := uuid.New().String()
	query := `INSERT INTO processing_jobs (id, video_id, job_type, parameters, status) 
		VALUES ($1, $2, 'analyze', $3, 'pending')`

	params, _ := json.Marshal(req)
	_, err := s.db.Exec(query, jobID, videoID, params)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	s.sendJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id":         jobID,
		"status":         "pending",
		"analysis_types": req.AnalysisTypes,
	})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, video_id, job_type, status, progress_percentage, 
		created_at, started_at, completed_at 
		FROM processing_jobs 
		ORDER BY created_at DESC 
		LIMIT 100`

	rows, err := s.db.Query(query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query jobs")
		return
	}
	defer rows.Close()

	var jobs []map[string]interface{}
	for rows.Next() {
		var id, videoID, jobType, status string
		var progress int
		var createdAt time.Time
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&id, &videoID, &jobType, &status, &progress,
			&createdAt, &startedAt, &completedAt); err != nil {
			continue
		}

		job := map[string]interface{}{
			"id":                  id,
			"video_id":            videoID,
			"job_type":            jobType,
			"status":              status,
			"progress_percentage": progress,
			"created_at":          createdAt,
		}

		if startedAt.Valid {
			job["started_at"] = startedAt.Time
		}
		if completedAt.Valid {
			job["completed_at"] = completedAt.Time
		}

		jobs = append(jobs, job)
	}

	s.sendJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	query := `SELECT id, video_id, job_type, parameters, status, progress_percentage, 
		output_path, error_message, created_at, started_at, completed_at 
		FROM processing_jobs WHERE id = $1`

	var job struct {
		ID           string          `json:"id"`
		VideoID      string          `json:"video_id"`
		JobType      string          `json:"job_type"`
		Parameters   json.RawMessage `json:"parameters"`
		Status       string          `json:"status"`
		Progress     int             `json:"progress_percentage"`
		OutputPath   *string         `json:"output_path"`
		ErrorMessage *string         `json:"error_message"`
		CreatedAt    time.Time       `json:"created_at"`
		StartedAt    *time.Time      `json:"started_at"`
		CompletedAt  *time.Time      `json:"completed_at"`
	}

	err := s.db.QueryRow(query, jobID).Scan(
		&job.ID, &job.VideoID, &job.JobType, &job.Parameters, &job.Status,
		&job.Progress, &job.OutputPath, &job.ErrorMessage,
		&job.CreatedAt, &job.StartedAt, &job.CompletedAt,
	)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query job")
		return
	}

	s.sendJSON(w, http.StatusOK, job)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	query := `UPDATE processing_jobs SET status = 'cancelled' WHERE id = $1 AND status IN ('pending', 'processing')`
	result, err := s.db.Exec(query, jobID)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to cancel job")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusBadRequest, "job cannot be cancelled")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"job_id": jobID,
		"status": "cancelled",
	})
}

func (s *Server) handleCreateStream(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string                   `json:"name"`
		InputSource   map[string]interface{}   `json:"input_source"`
		OutputTargets []map[string]interface{} `json:"output_targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Generate stream key
	sessionID := uuid.New().String()
	streamKey := uuid.New().String()

	// Insert into database
	query := `INSERT INTO streaming_sessions (id, name, input_source, output_targets, stream_key) 
		VALUES ($1, $2, $3, $4, $5)`

	outputJSON, _ := json.Marshal(req.OutputTargets)

	_, err := s.db.Exec(query, sessionID, req.Name, "rtmp", outputJSON, streamKey)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to create stream")
		return
	}

	s.sendJSON(w, http.StatusCreated, map[string]interface{}{
		"session_id":  sessionID,
		"stream_key":  streamKey,
		"rtmp_url":    fmt.Sprintf("rtmp://localhost:1935/live/%s", streamKey),
		"preview_url": fmt.Sprintf("http://localhost:%s/stream/%s", s.config.Port, sessionID),
	})
}

func (s *Server) handleStartStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	query := `UPDATE streaming_sessions SET is_active = true, start_time = CURRENT_TIMESTAMP 
		WHERE id = $1`
	_, err := s.db.Exec(query, sessionID)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to start stream")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"status":     "streaming",
	})
}

func (s *Server) handleStopStream(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	query := `UPDATE streaming_sessions SET is_active = false, end_time = CURRENT_TIMESTAMP 
		WHERE id = $1`
	_, err := s.db.Exec(query, sessionID)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to stop stream")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"session_id": sessionID,
		"status":     "stopped",
	})
}

func (s *Server) handleListStreams(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, is_active, viewer_count, start_time, end_time 
		FROM streaming_sessions 
		ORDER BY created_at DESC 
		LIMIT 100`

	rows, err := s.db.Query(query)
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query streams")
		return
	}
	defer rows.Close()

	var streams []map[string]interface{}
	for rows.Next() {
		var id, name string
		var isActive bool
		var viewerCount int
		var startTime, endTime sql.NullTime

		if err := rows.Scan(&id, &name, &isActive, &viewerCount, &startTime, &endTime); err != nil {
			continue
		}

		stream := map[string]interface{}{
			"id":           id,
			"name":         name,
			"is_active":    isActive,
			"viewer_count": viewerCount,
		}

		if startTime.Valid {
			stream["start_time"] = startTime.Time
		}
		if endTime.Valid {
			stream["end_time"] = endTime.Time
		}

		streams = append(streams, stream)
	}

	s.sendJSON(w, http.StatusOK, streams)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"name":        "video-tools API",
		"version":     "1.0.0",
		"description": "Video processing, editing, and conversion utilities",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "description": "Health check"},
			{"method": "GET", "path": "/api/status", "description": "Service status and statistics"},
			{"method": "POST", "path": "/api/v1/video/upload", "description": "Upload video file"},
			{"method": "GET", "path": "/api/v1/video/{id}", "description": "Get video metadata"},
			{"method": "GET", "path": "/api/v1/video/{id}/info", "description": "Get detailed video info"},
			{"method": "POST", "path": "/api/v1/video/{id}/edit", "description": "Edit video"},
			{"method": "POST", "path": "/api/v1/video/{id}/convert", "description": "Convert video format"},
			{"method": "GET", "path": "/api/v1/video/{id}/frames", "description": "Extract frames"},
			{"method": "POST", "path": "/api/v1/video/{id}/thumbnail", "description": "Generate thumbnail"},
			{"method": "POST", "path": "/api/v1/video/{id}/audio", "description": "Extract audio"},
			{"method": "POST", "path": "/api/v1/video/{id}/subtitles", "description": "Add subtitles"},
			{"method": "POST", "path": "/api/v1/video/{id}/compress", "description": "Compress video"},
			{"method": "POST", "path": "/api/v1/video/{id}/analyze", "description": "Analyze video with AI"},
			{"method": "GET", "path": "/api/v1/jobs", "description": "List processing jobs"},
			{"method": "GET", "path": "/api/v1/jobs/{id}", "description": "Get job details"},
			{"method": "POST", "path": "/api/v1/stream/create", "description": "Create streaming session"},
			{"method": "GET", "path": "/api/v1/streams", "description": "List streaming sessions"},
		},
	}

	s.sendJSON(w, http.StatusOK, docs)
}

// Helper functions
func (s *Server) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: status < 400,
		Data:    data,
	})
}

func (s *Server) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Run starts the server
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting_down_server", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server_shutdown_error", map[string]interface{}{"error": err.Error()})
		}

		s.db.Close()
	}()

	logger.Info("server_starting", map[string]interface{}{
		"port":     s.config.Port,
		"api_url":  fmt.Sprintf("http://localhost:%s", s.config.Port),
		"docs_url": fmt.Sprintf("http://localhost:%s/docs", s.config.Port),
	})

	return srv.ListenAndServe()
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start video-tools

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	logger.Info("starting_api", map[string]interface{}{"version": "1.0.0"})

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
