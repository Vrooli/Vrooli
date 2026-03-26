// DOC: docs/concepts/ARCHITECTURE.md#system-overview
// DOC: docs/reference/configuration.md
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"brand-manager/config"
	"brand-manager/database"
	"brand-manager/handlers"
	"brand-manager/repository"

	"github.com/google/uuid"
	gorhandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "brand-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Load centralized configuration from environment with defaults
	cfg := config.Load()

	// Connect to SQLite with WAL mode and schema init [REQ:BM-REQ-STORE-INIT]
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Wire repositories (SQLite implementations)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	versionRepo := repository.NewSQLiteVersionRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	assetRepo := repository.NewSQLiteAssetRepository(db)

	// Wire handlers with config
	h := handlers.New(brandRepo, versionRepo, assignRepo).WithAssets(assetRepo).WithConfig(cfg)

	// Set up router
	router := mux.NewRouter()
	router.Use(requestIDMiddleware)
	router.Use(loggingMiddleware)

	// Health endpoint [REQ:BM-REQ-API-BRANDS]
	healthHandler := health.New().
		Version(cfg.APIVersion).
		Check(health.DB(db), health.Critical).
		Handler()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Register domain routes
	h.RegisterRoutes(router)

	handler := gorhandlers.RecoveryHandler()(router)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code for logging.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

// requestIDMiddleware assigns a unique request ID to each request.
// If the client sends X-Request-ID, it is reused; otherwise a new UUID is generated.
// The ID is set on the response header and available for structured logging.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", reqID)
		r.Header.Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		level := "info"
		if sw.code >= 500 {
			level = "error"
		} else if sw.code >= 400 {
			level = "warn"
		}
		reqID := w.Header().Get("X-Request-ID")
		log.Printf("[%s] %s %s %d %s req=%s", level, r.Method, r.RequestURI, sw.code, time.Since(start), reqID)
	})
}
