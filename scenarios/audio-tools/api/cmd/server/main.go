package main

import (
	"audio-tools/internal/handlers"
	"audio-tools/internal/storage"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
)

type Server struct {
	router       *mux.Router
	db           *sql.DB
	audioHandler *handlers.AudioHandler
	storage      *storage.MinIOStorage
	config       *Config
}

type Config struct {
	Port            string
	DatabaseURL     string
	WorkDir         string
	DataDir         string
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucketName string
	MinIOUseSSL     bool
}

func initializeDatabase(db *sql.DB) error {
	// Create the audio_processing_jobs table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS audio_processing_jobs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		audio_asset_id UUID,
		operation_type VARCHAR(50) NOT NULL,
		parameters JSONB DEFAULT '{}',
		status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
		progress_percentage INTEGER DEFAULT 0,
		start_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		end_time TIMESTAMP WITH TIME ZONE,
		output_files JSONB,
		error_message TEXT,
		processing_node VARCHAR(255),
		resource_usage JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS audio_assets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		file_path TEXT NOT NULL,
		format VARCHAR(20),
		duration_seconds DECIMAL(10,3),
		sample_rate INTEGER,
		bit_depth INTEGER,
		channels INTEGER,
		bitrate INTEGER,
		file_size_bytes BIGINT,
		metadata JSONB DEFAULT '{}',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_processed TIMESTAMP WITH TIME ZONE,
		quality_score DECIMAL(3,2),
		noise_level DECIMAL(5,2),
		speech_detected BOOLEAN DEFAULT FALSE,
		language VARCHAR(10),
		tags TEXT[]
	);
	`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Create indexes
	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_audio_assets_format ON audio_assets(format);
	CREATE INDEX IF NOT EXISTS idx_audio_assets_created ON audio_assets(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audio_processing_jobs_asset ON audio_processing_jobs(audio_asset_id);
	CREATE INDEX IF NOT EXISTS idx_audio_processing_jobs_status ON audio_processing_jobs(status);
	CREATE INDEX IF NOT EXISTS idx_audio_processing_jobs_operation ON audio_processing_jobs(operation_type);
	`

	_, err = db.Exec(indexSQL)
	if err != nil {
		// Indexes are optional, just log the error
		log.Printf("Warning: Could not create indexes: %v", err)
	}

	return nil
}

func NewServer() (*Server, error) {
	config := &Config{
		Port:            getEnv("API_PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/audio_tools"),
		WorkDir:         getEnv("WORK_DIR", "/tmp/audio-tools"),
		DataDir:         getEnv("DATA_DIR", "./data"),
		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucketName: getEnv("MINIO_BUCKET_NAME", "audio-files"),
		MinIOUseSSL:     getEnv("MINIO_USE_SSL", "false") == "true",
	}

	// Create work and data directories
	os.MkdirAll(config.WorkDir, 0755)
	os.MkdirAll(config.DataDir, 0755)

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	// Continue without database if not available.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Printf("Warning: Could not connect to database: %v", err)
		db = nil
	} else {
		// Set connection pool settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Initialize database schema if needed
		if err := initializeDatabase(db); err != nil {
			log.Printf("Warning: Could not initialize database schema: %v", err)
		}
	}

	// Initialize MinIO storage (optional)
	var minioStorage *storage.MinIOStorage
	if config.MinIOEndpoint != "" {
		var err error
		minioStorage, err = storage.NewMinIOStorage(
			config.MinIOEndpoint,
			config.MinIOAccessKey,
			config.MinIOSecretKey,
			config.MinIOBucketName,
			config.MinIOUseSSL,
		)
		if err != nil {
			log.Printf("Warning: Could not connect to MinIO: %v", err)
			// Continue without MinIO - will use filesystem
		} else {
			log.Printf("Successfully connected to MinIO at %s", config.MinIOEndpoint)
		}
	}

	server := &Server{
		router:       mux.NewRouter(),
		db:           db,
		config:       config,
		storage:      minioStorage,
		audioHandler: handlers.NewAudioHandler(db, config.WorkDir, config.DataDir),
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health check - use api-core/health for standardized response
	// DB is optional for this scenario (can run without database)
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/health", healthHandler).Methods("GET", "OPTIONS")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Audio editing endpoints
	api.HandleFunc("/audio/edit", s.audioHandler.HandleEdit).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/convert", s.audioHandler.HandleConvert).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/metadata/{id}", s.audioHandler.HandleMetadata).Methods("GET", "OPTIONS")
	api.HandleFunc("/audio/metadata", s.audioHandler.HandleMetadata).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/enhance", s.audioHandler.HandleEnhance).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/analyze", s.audioHandler.HandleAnalyze).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/vad", s.audioHandler.HandleVAD).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/remove-silence", s.audioHandler.HandleRemoveSilence).Methods("POST", "OPTIONS")

	// Status endpoint
	api.HandleFunc("/status", s.handleStatus).Methods("GET", "OPTIONS")

	// Documentation
	s.router.HandleFunc("/api/docs", s.handleDocs).Methods("GET")
}


func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":  "audio-tools",
		"version":  "1.0.0",
		"uptime":   time.Since(startTime).Seconds(),
		"work_dir": s.config.WorkDir,
		"data_dir": s.config.DataDir,
		"capabilities": []string{
			"edit", "convert", "metadata", "enhance", "analyze",
			"trim", "merge", "split", "fade", "volume", "normalize",
			"noise_reduction", "speed", "pitch", "equalizer",
		},
		"supported_formats": []string{
			"mp3", "wav", "flac", "aac", "ogg", "m4a", "wma", "opus",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	docs := map[string]interface{}{
		"name":        "Audio Tools API",
		"version":     "1.0.0",
		"description": "Comprehensive audio processing and analysis platform",
		"endpoints": []map[string]string{
			{
				"method":      "GET",
				"path":        "/health",
				"description": "Health check",
			},
			{
				"method":      "GET",
				"path":        "/api/v1/status",
				"description": "Service status and capabilities",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/edit",
				"description": "Edit audio files (trim, merge, split, fade, volume, normalize)",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/convert",
				"description": "Convert audio between formats",
			},
			{
				"method":      "GET",
				"path":        "/api/v1/audio/metadata/{id}",
				"description": "Extract audio metadata",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/enhance",
				"description": "Enhance audio quality (noise reduction, EQ, compression)",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/analyze",
				"description": "Analyze audio content",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/vad",
				"description": "Voice activity detection - identify speech segments and silence",
			},
			{
				"method":      "POST",
				"path":        "/api/v1/audio/remove-silence",
				"description": "Remove silence from audio, keeping only speech segments",
			},
		},
		"supported_operations": []string{
			"trim", "merge", "split", "fade_in", "fade_out",
			"volume_adjustment", "normalize", "noise_reduction",
			"speed_change", "pitch_shift", "equalizer",
		},
		"supported_formats": []string{
			"mp3", "wav", "flac", "aac", "ogg", "m4a",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[%s] %s %s - %v", r.Method, r.RequestURI, r.RemoteAddr, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use specific origin for production, wildcard only for development
		allowedOrigin := os.Getenv("CORS_ORIGIN")
		if allowedOrigin == "" {
			// Default to localhost for development
			allowedOrigin = "http://localhost:*"
		}
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

var startTime = time.Now()

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		if s.db != nil {
			s.db.Close()
		}

		// Clean up work directory
		files, _ := filepath.Glob(filepath.Join(s.config.WorkDir, "*"))
		for _, file := range files {
			os.Remove(file)
		}
	}()

	log.Printf("Audio Tools API starting on port %s", s.config.Port)
	log.Printf("Work directory: %s", s.config.WorkDir)
	log.Printf("API documentation: http://localhost:%s/api/docs", s.config.Port)

	return srv.ListenAndServe()
}

func main() {
	// Check if running through lifecycle (optional - will work either way)
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		log.Println("Note: Running outside Vrooli lifecycle system")
	}

	log.Println("Starting Audio Tools API...")

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	if err := server.Run(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
