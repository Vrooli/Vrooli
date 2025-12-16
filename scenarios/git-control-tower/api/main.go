package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	git    GitRunner
	audit  AuditLogger
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
		return nil, fmt.Errorf("failed to initialize database handle: %w", err)
	}

	// Initialize audit logger with graceful degradation
	var auditLogger AuditLogger
	if db != nil {
		auditLogger = NewPostgresAuditLogger(db)
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}

	srv := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
		git:    &ExecGitRunner{GitPath: "git"},
		audit:  auditLogger,
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/status", s.handleRepoStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/diff", s.handleDiff).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/stage", s.handleStage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/unstage", s.handleUnstage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/commit", s.handleCommit).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/sync-status", s.handleSyncStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/discard", s.handleDiscard).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/push", s.handlePush).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/pull", s.handlePull).Methods("POST")
	s.router.HandleFunc("/api/v1/audit", s.handleAuditQuery).Methods("GET")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "git-control-tower-api",
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
	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(r.Context())

	checks := NewHealthChecks(HealthCheckDeps{
		DB:      NewSQLDBChecker(s.db),
		Git:     s.git,
		RepoDir: repoDir,
	})
	result := checks.Run(r.Context())

	if !result.Readiness {
		resp.ServiceUnavailable(result)
		return
	}
	resp.OK(result)
}

func (s *Server) handleRepoStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	status, err := GetRepoStatus(ctx, RepoStatusDeps{
		Git:     s.git,
		RepoDir: repoDir,
	})
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(status)
}

// [REQ:GCT-OT-P0-003] File diff endpoint
func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	diff, err := GetDiff(ctx, DiffDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, DiffRequest{
		Path:   query.Get("path"),
		Staged: query.Get("staged") == "true",
		Base:   query.Get("base"),
	})
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(diff)
}

// [REQ:GCT-OT-P0-004] Stage/unstage operations
func (s *Server) handleStage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req StageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := StageFiles(ctx, StagingDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for stage operation
	auditEntry := AuditEntry{
		Operation: AuditOpStage,
		RepoDir:   repoDir,
		Paths:     req.Paths,
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil {
		auditEntry.Paths = result.Staged
	}
	// Log asynchronously to avoid blocking the response
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

func (s *Server) handleUnstage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req UnstageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := UnstageFiles(ctx, StagingDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for unstage operation
	auditEntry := AuditEntry{
		Operation: AuditOpUnstage,
		RepoDir:   repoDir,
		Paths:     req.Paths,
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil {
		auditEntry.Paths = result.Unstaged
	}
	// Log asynchronously to avoid blocking the response
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// [REQ:GCT-OT-P0-005] Commit composition API
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req CommitRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := CreateCommit(ctx, CommitDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for commit operation
	auditEntry := AuditEntry{
		Operation:     AuditOpCommit,
		RepoDir:       repoDir,
		CommitMessage: req.Message,
		Success:       result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil {
		if result.Success {
			auditEntry.CommitHash = result.Hash
		} else {
			auditEntry.Error = result.Error
			if len(result.ValidationErrors) > 0 {
				auditEntry.Error = strings.Join(result.ValidationErrors, "; ")
			}
		}
	}
	// Log asynchronously to avoid blocking the response
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// [REQ:GCT-OT-P0-006] Push/pull status
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	req := SyncStatusRequest{
		Fetch:  query.Get("fetch") == "true",
		Remote: query.Get("remote"),
	}

	result, err := GetSyncStatus(ctx, SyncStatusDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(result)
}

// handleDiscard handles POST /api/v1/repo/discard
func (s *Server) handleDiscard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req DiscardRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if len(req.Paths) == 0 {
		resp.BadRequest("paths are required")
		return
	}

	result, err := DiscardFiles(ctx, DiscardDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// Audit logging for discard operation
	auditEntry := AuditEntry{
		Operation: AuditOpDiscard,
		RepoDir:   repoDir,
		Paths:     req.Paths,
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil {
		auditEntry.Paths = result.Discarded
	}
	// Log asynchronously
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// handlePush handles POST /api/v1/repo/push
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req PushRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PushToRemote(ctx, PushPullDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// Audit logging for push operation
	auditEntry := AuditEntry{
		Operation: AuditOpPush,
		RepoDir:   repoDir,
		Success:   result != nil && result.Success,
		Metadata: map[string]interface{}{
			"remote": result.Remote,
			"branch": result.Branch,
		},
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = result.Error
	}
	// Log asynchronously
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// handlePull handles POST /api/v1/repo/pull
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req PullRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PullFromRemote(ctx, PushPullDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	// Audit logging for pull operation
	auditEntry := AuditEntry{
		Operation: AuditOpPull,
		RepoDir:   repoDir,
		Success:   result != nil && result.Success,
		Metadata: map[string]interface{}{
			"remote":        result.Remote,
			"branch":        result.Branch,
			"has_conflicts": result.HasConflicts,
		},
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = result.Error
	}
	// Log asynchronously
	go func() {
		logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer logCancel()
		_ = s.audit.Log(logCtx, auditEntry)
	}()

	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// [REQ:GCT-OT-P0-007] Audit log query endpoint
func (s *Server) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)

	// Parse query parameters
	query := r.URL.Query()
	req := AuditQueryRequest{
		Operation: AuditOperation(query.Get("operation")),
		Branch:    query.Get("branch"),
		Limit:     50, // Default limit
	}

	// Parse optional parameters
	if limitStr := query.Get("limit"); limitStr != "" {
		if _, err := fmt.Sscanf(limitStr, "%d", &req.Limit); err != nil {
			resp.BadRequest("invalid limit parameter")
			return
		}
		if req.Limit > 1000 {
			req.Limit = 1000 // Cap at 1000
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if _, err := fmt.Sscanf(offsetStr, "%d", &req.Offset); err != nil {
			resp.BadRequest("invalid offset parameter")
			return
		}
	}

	result, err := s.audit.Query(ctx, req)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(result)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
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
   vrooli scenario start git-control-tower

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
