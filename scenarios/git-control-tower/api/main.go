package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port string
}

// Server wires the HTTP router and database connection
type Server struct {
	config  *Config
	db      *sql.DB
	router  *mux.Router
	git     GitRunner
	audit   AuditLogger
	sandbox *WorkspaceSandboxClient
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port: requireEnv("API_PORT"),
	}

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
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
		config:  cfg,
		db:      db,
		router:  mux.NewRouter(),
		git:     &ExecGitRunner{GitPath: "git"},
		audit:   auditLogger,
		sandbox: NewWorkspaceSandboxClient(5 * time.Second),
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
	s.router.HandleFunc("/api/v1/repo/history", s.handleRepoHistory).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/stage", s.handleStage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/unstage", s.handleUnstage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/commit", s.handleCommit).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/approved-changes", s.handleApprovedChanges).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/approved-changes/preview", s.handleApprovedChangesPreview).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/sync-status", s.handleSyncStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/discard", s.handleDiscard).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/ignore", s.handleIgnore).Methods("POST")
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

func (s *Server) handleRepoHistory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	limit := 30
	includeParam := strings.TrimSpace(r.URL.Query().Get("include"))
	includeFiles := includeParam == "files" || includeParam == "details"
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 200 {
			parsed = 200
		}
		limit = parsed
	}

	history, err := GetRepoHistory(ctx, RepoHistoryDeps{
		Git:          s.git,
		RepoDir:      repoDir,
		Limit:        limit,
		IncludeFiles: includeFiles,
	})
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(history)
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
		Path:      query.Get("path"),
		Staged:    query.Get("staged") == "true",
		Untracked: query.Get("untracked") == "true",
		Base:      query.Get("base"),
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

// handleIgnore handles POST /api/v1/repo/ignore
func (s *Server) handleIgnore(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req IgnoreRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Path) == "" {
		resp.BadRequest("path is required")
		return
	}

	result, err := IgnorePath(ctx, IgnoreDeps{
		Git:     s.git,
		RepoDir: repoDir,
	}, req)

	auditEntry := AuditEntry{
		Operation: AuditOpIgnore,
		RepoDir:   repoDir,
		Paths:     []string{req.Path},
		Success:   result != nil && result.Success,
	}
	if err != nil {
		auditEntry.Error = err.Error()
	} else if result != nil && !result.Success {
		auditEntry.Error = strings.Join(result.Errors, "; ")
	}
	if result != nil && len(result.Ignored) > 0 {
		auditEntry.Paths = result.Ignored
	}
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

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "git-control-tower",
	}) {
		return // Process was re-exec'd after rebuild
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
