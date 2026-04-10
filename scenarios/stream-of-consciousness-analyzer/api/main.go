// DOC: docs/concepts/ARCHITECTURE.md#api-layer
// DOC: docs/reference/api-endpoints.md
package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Server wires the HTTP router and database connection
type Server struct {
	db          *sql.DB
	router      *mux.Router
	schemes     SchemeStore
	information InformationStore
	thoughts    ThoughtStore
	export      ExportStore
	suggestions SuggestionProvider
}

// NewServer initializes database, schema, services, and routes
func NewServer(db *sql.DB) *Server {
	srv := &Server{
		db:          db,
		router:      mux.NewRouter(),
		schemes:     NewSchemeService(db),
		information: NewInformationService(db),
		thoughts:    NewThoughtService(db),
		export:      NewExportService(db),
		suggestions: NewSuggestionService(db),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(requestTimeoutMiddleware)
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().
		Version(AppVersion).
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Scheme CRUD
	s.router.HandleFunc("/api/v1/schemes", handleListSchemes(s.schemes)).Methods("GET")
	s.router.HandleFunc("/api/v1/schemes", handleCreateScheme(s.schemes)).Methods("POST")
	s.router.HandleFunc("/api/v1/schemes/{id}", handleGetScheme(s.schemes)).Methods("GET")
	s.router.HandleFunc("/api/v1/schemes/{id}", handleUpdateScheme(s.schemes)).Methods("PUT")
	s.router.HandleFunc("/api/v1/schemes/{id}", handleDeleteScheme(s.schemes)).Methods("DELETE")

	// Information CRUD (nested under schemes)
	s.router.HandleFunc("/api/v1/schemes/{schemeId}/information", handleListInformation(s.information)).Methods("GET")
	s.router.HandleFunc("/api/v1/schemes/{schemeId}/information", handleCreateInformation(s.information)).Methods("POST")
	s.router.HandleFunc("/api/v1/schemes/{schemeId}/information/{infoId}", handleUpdateInformation(s.information)).Methods("PUT")
	s.router.HandleFunc("/api/v1/schemes/{schemeId}/information/{infoId}", handleDeleteInformation(s.information)).Methods("DELETE")

	// Thought CRUD
	s.router.HandleFunc("/api/v1/thoughts", handleListThoughts(s.thoughts)).Methods("GET")
	s.router.HandleFunc("/api/v1/thoughts", handleCreateThought(s.thoughts)).Methods("POST")
	s.router.HandleFunc("/api/v1/thoughts/{id}", handleGetThought(s.thoughts)).Methods("GET")
	s.router.HandleFunc("/api/v1/thoughts/{id}", handleUpdateThought(s.thoughts)).Methods("PUT")
	s.router.HandleFunc("/api/v1/thoughts/{id}", handleDeleteThought(s.thoughts)).Methods("DELETE")

	// Thought edges
	s.router.HandleFunc("/api/v1/thoughts/{id}/edges", handleListEdges(s.thoughts)).Methods("GET")
	s.router.HandleFunc("/api/v1/thoughts/{id}/edges", handleCreateEdge(s.thoughts)).Methods("POST")
	s.router.HandleFunc("/api/v1/thoughts/{id}/edges/{edgeId}", handleDeleteEdge(s.thoughts)).Methods("DELETE")

	// Export
	s.router.HandleFunc("/api/v1/schemes/{id}/export", handleExportScheme(s.export)).Methods("GET")

	// Suggestions / LLM providers
	s.router.HandleFunc("/api/v1/providers", handleGetProviders(s.suggestions)).Methods("GET")
	s.router.HandleFunc("/api/v1/schemes/{id}/suggestions", handleGenerateSuggestions(s.suggestions)).Methods("POST")
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// statusWriter wraps ResponseWriter to capture the HTTP status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// requestTimeoutMiddleware attaches a deadline to every request context so that
// no single handler can run indefinitely (e.g. slow DB or LLM calls).
func requestTimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs each request with method, path, status, and duration.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		level := "INFO"
		if sw.status >= 500 {
			level = "ERROR"
		} else if sw.status >= 400 {
			level = "WARN"
		}
		log.Printf("[%s] %s %s %d %s", level, r.Method, r.RequestURI, sw.status, time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "stream-of-consciousness-analyzer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Connect to database with automatic retry and backoff
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	// Apply schema
	if err := ensureSchema(db); err != nil {
		log.Fatalf("Schema migration failed: %v", err)
	}

	srv := NewServer(db)

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
