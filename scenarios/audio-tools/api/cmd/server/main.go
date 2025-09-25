package main

import (
	"audio-tools/internal/handlers"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Server struct {
	router       *mux.Router
	db           *sql.DB
	audioHandler *handlers.AudioHandler
	config       *Config
}

type Config struct {
	Port        string
	DatabaseURL string
	WorkDir     string
	DataDir     string
}

func NewServer() (*Server, error) {
	config := &Config{
		Port:        getEnv("API_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/audio_tools"),
		WorkDir:     getEnv("WORK_DIR", "/tmp/audio-tools"),
		DataDir:     getEnv("DATA_DIR", "./data"),
	}

	// Create work and data directories
	os.MkdirAll(config.WorkDir, 0755)
	os.MkdirAll(config.DataDir, 0755)

	// Connect to database (optional - will work without it)
	var db *sql.DB
	dbURL := config.DatabaseURL
	if dbURL != "" {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Warning: Could not connect to database: %v", err)
			// Continue without database
		} else {
			if err := db.Ping(); err != nil {
				log.Printf("Warning: Database ping failed: %v", err)
				db = nil
			}
		}
	}

	server := &Server{
		router:       mux.NewRouter(),
		db:           db,
		config:       config,
		audioHandler: handlers.NewAudioHandler(db, config.WorkDir, config.DataDir),
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET", "OPTIONS")

	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Audio editing endpoints
	api.HandleFunc("/audio/edit", s.audioHandler.HandleEdit).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/convert", s.audioHandler.HandleConvert).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/metadata/{id}", s.audioHandler.HandleMetadata).Methods("GET", "OPTIONS")
	api.HandleFunc("/audio/enhance", s.audioHandler.HandleEnhance).Methods("POST", "OPTIONS")
	api.HandleFunc("/audio/analyze", s.audioHandler.HandleAnalyze).Methods("POST", "OPTIONS")

	// Status endpoint
	api.HandleFunc("/status", s.handleStatus).Methods("GET", "OPTIONS")

	// Documentation
	s.router.HandleFunc("/api/docs", s.handleDocs).Methods("GET")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "audio-tools",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}

	if s.db != nil {
		if err := s.db.Ping(); err != nil {
			health["database"] = "disconnected"
		} else {
			health["database"] = "connected"
		}
	} else {
		health["database"] = "not configured"
	}

	// Check if ffmpeg is available
	if _, err := os.Stat("/usr/bin/ffmpeg"); err == nil {
		health["ffmpeg"] = "available"
	} else {
		health["ffmpeg"] = "not found"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":     "audio-tools",
		"version":     "1.0.0",
		"uptime":      time.Since(startTime).Seconds(),
		"work_dir":    s.config.WorkDir,
		"data_dir":    s.config.DataDir,
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