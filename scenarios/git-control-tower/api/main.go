package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver (CGO-free, enables static builds)

	"git-control-tower/ssh"
)

// Config holds minimal runtime configuration
type Config struct {
	Port string
}

// Server wires the HTTP router and database connection
type Server struct {
	config          *Config
	db              *sql.DB
	router          *mux.Router
	git             GitRunner
	audit           AuditLogger
	sandbox         *WorkspaceSandboxClient
	capabilities    *CapabilityRegistry
	sshDeps         ssh.SSHDeps
	repos           *RepoService
	credStore       *CredentialsStore
	storageResolver *storage.Resolver
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port: requireEnv("API_PORT"),
	}

	// Connect to SQLite with automatic retry and backoff.
	dsn, err := sqliteDSN()
	if err != nil {
		return nil, fmt.Errorf("sqlite configuration failed: %w", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       "sqlite", // modernc.org/sqlite registers as "sqlite" (not "sqlite3")
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err := ensureAuditSchema(db); err != nil {
		return nil, fmt.Errorf("audit schema initialization failed: %w", err)
	}
	if err := ensureRepoSchema(db); err != nil {
		return nil, fmt.Errorf("repo schema initialization failed: %w", err)
	}

	// Initialize audit logger with graceful degradation
	var auditLogger AuditLogger
	if db != nil {
		auditLogger = NewSQLiteAuditLogger(db)
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
		capabilities: NewCapabilityRegistry(knownCapabilities, map[string]StatusChecker{
			"workspace-sandbox": &ScenarioChecker{
				Slug:   "workspace-sandbox",
				Client: &http.Client{Timeout: 3 * time.Second},
			},
		}, 30*time.Second),
		sshDeps: ssh.SSHDeps{Platform: ssh.DefaultPlatform()},
	}
	srv.repos = NewRepoService(NewSQLiteRepoStore(db), srv.git)

	credStore, err := NewCredentialsStore("")
	if err != nil {
		log.Printf("WARNING: credentials store initialization failed: %v (SSH/HTTPS auth for git operations will be unavailable)", err)
	} else {
		srv.credStore = credStore
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return nil, fmt.Errorf("storage resolver init failed: %w", err)
	}
	srv.storageResolver = resolver

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/status", s.handleRepoStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/diff", s.handleDiff).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/history", s.handleRepoHistory).Methods("GET")
	// Repo registry endpoints
	s.router.HandleFunc("/api/v1/repos", s.handleRepoList).Methods("GET")
	s.router.HandleFunc("/api/v1/repos/active", s.handleRepoActive).Methods("GET")
	s.router.HandleFunc("/api/v1/repos/active", s.handleRepoSetActive).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/open", s.handleRepoOpen).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/clone", s.handleRepoClone).Methods("POST")
	s.router.HandleFunc("/api/v1/repos/{id}", s.handleRepoRemove).Methods("DELETE")
	s.router.HandleFunc("/api/v1/repo/stage", s.handleStage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/unstage", s.handleUnstage).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/commit", s.handleCommit).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/approved-changes", s.handleApprovedChanges).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/approved-changes/preview", s.handleApprovedChangesPreview).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/sync-status", s.handleSyncStatus).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/discard", s.handleDiscard).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/ignore", s.handleIgnore).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/grouping-rules", s.handleGetGroupingRules).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/grouping-rules", s.handleSaveGroupingRules).Methods("PUT")
	s.router.HandleFunc("/api/v1/repo/gitignore/health", s.handleGitignoreHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/gitignore/move", s.handleGitignoreMove).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/push", s.handlePush).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/pull", s.handlePull).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/upstream-action", s.handleUpstreamAction).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branches", s.handleRepoBranches).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/branch/create", s.handleBranchCreate).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branch/switch", s.handleBranchSwitch).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/branch/publish", s.handleBranchPublish).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/files", s.handleFiles).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/files/dir", s.handleDirectoryList).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/files/content", s.handleSaveFileContent).Methods("PUT")
	s.router.HandleFunc("/api/v1/repo/files/delete", s.handleDeletePath).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/related", s.handleRelatedFiles).Methods("GET")
	s.router.HandleFunc("/api/v1/repo/search/content", s.handleContentSearch).Methods("GET")
	s.router.HandleFunc("/api/v1/capabilities", s.handleCapabilities).Methods("GET")
	s.router.HandleFunc("/api/v1/audit", s.handleAuditQuery).Methods("GET")

	// Credentials management endpoints
	s.router.HandleFunc("/api/v1/credentials", s.handleListCredentials).Methods("GET")
	s.router.HandleFunc("/api/v1/credentials", s.handleSaveCredential).Methods("POST")
	s.router.HandleFunc("/api/v1/credentials/{id}", s.handleDeleteCredential).Methods("DELETE")
	s.router.HandleFunc("/api/v1/credentials/test", s.handleTestCredential).Methods("POST")
	s.router.HandleFunc("/api/v1/repo/remote/url", s.handleUpdateRemoteURL).Methods("POST")

	// SSH key management endpoints
	s.router.HandleFunc("/api/v1/ssh/keys", ssh.HandleListKeys(s.sshDeps)).Methods("GET")
	s.router.HandleFunc("/api/v1/ssh/keys/generate", ssh.HandleGenerateKey(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys/public", ssh.HandleGetPublicKey(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys/test", ssh.HandleTestConnection(s.sshDeps)).Methods("POST")
	s.router.HandleFunc("/api/v1/ssh/keys", ssh.HandleDeleteKey(s.sshDeps)).Methods("DELETE")
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return gorillahandlers.RecoveryHandler()(s.router)
}

// NOTE: The old handleHealth with custom HealthChecks has been replaced by
// api-core/health for standardized responses. The DB check is now handled
// by the health package with Critical priority.

func (s *Server) handleRepoStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	status, err := GetRepoStatus(hctx.Ctx, RepoStatusDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(status)
}

func (s *Server) handleRepoHistory(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	limit := 30
	includeParam := strings.TrimSpace(r.URL.Query().Get("include"))
	includeFiles := includeParam == "files" || includeParam == "details"
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			hctx.Resp.BadRequest("limit must be a positive integer")
			return
		}
		if parsed > 200 {
			parsed = 200
		}
		limit = parsed
	}

	history, err := GetRepoHistory(hctx.Ctx, RepoHistoryDeps{
		Git:          hctx.Git,
		RepoDir:      hctx.RepoDir,
		Limit:        limit,
		IncludeFiles: includeFiles,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(history)
}

// [REQ:GCT-OT-P0-003] File diff endpoint
func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	query := r.URL.Query()

	// Parse and validate view mode
	modeStr := query.Get("mode")
	var mode ViewMode
	switch modeStr {
	case "full_diff":
		mode = ViewModeFullDiff
	case "source":
		mode = ViewModeSource
	default:
		mode = ViewModeDiff
	}

	diff, err := GetDiff(hctx.Ctx, DiffDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, DiffRequest{
		Path:      query.Get("path"),
		Staged:    query.Get("staged") == "true",
		Untracked: query.Get("untracked") == "true",
		Base:      query.Get("base"),
		Commit:    query.Get("commit"),
		Mode:      mode,
		Any:       query.Get("any") == "true",
	})
	if err != nil {
		var tooLarge *FileTooLargeError
		if errors.As(err, &tooLarge) {
			hctx.Resp.PayloadTooLarge(tooLarge.Error())
			return
		}
		var unsupported *UnsupportedBinaryError
		if errors.As(err, &unsupported) {
			hctx.Resp.UnsupportedMediaType(unsupported.Error())
			return
		}
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(diff)
}

// [REQ:GCT-OT-P0-004] Stage/unstage operations
func (s *Server) handleStage(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req StageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := StageFiles(hctx.Ctx, StagingDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for stage operation
	auditEntry := AuditEntry{
		Operation: AuditOpStage,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

func (s *Server) handleUnstage(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req UnstageRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if !ValidateStagingRequest(w, req) {
		return
	}

	result, err := UnstageFiles(hctx.Ctx, StagingDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for unstage operation
	auditEntry := AuditEntry{
		Operation: AuditOpUnstage,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// [REQ:GCT-OT-P0-005] Commit composition API
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CommitRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := CreateCommit(hctx.Ctx, CommitDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	// [REQ:GCT-OT-P0-007] Audit logging for commit operation
	commitMessage := req.Message
	if result != nil && strings.TrimSpace(result.Message) != "" {
		commitMessage = result.Message
	}
	auditEntry := AuditEntry{
		Operation:     AuditOpCommit,
		RepoDir:       hctx.RepoDir,
		CommitMessage: commitMessage,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// [REQ:GCT-OT-P0-006] Push/pull status
func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	// Parse query parameters
	query := r.URL.Query()
	req := SyncStatusRequest{
		Fetch:  query.Get("fetch") == "true",
		Remote: query.Get("remote"),
	}

	result, err := GetSyncStatus(hctx.Ctx, SyncStatusDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleDiscard handles POST /api/v1/repo/discard
func (s *Server) handleDiscard(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req DiscardRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if len(req.Paths) == 0 {
		hctx.Resp.BadRequest("paths are required")
		return
	}

	result, err := DiscardFiles(hctx.Ctx, DiscardDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)

	// Audit logging for discard operation
	auditEntry := AuditEntry{
		Operation: AuditOpDiscard,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handleIgnore handles POST /api/v1/repo/ignore
func (s *Server) handleIgnore(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req IgnoreRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if strings.TrimSpace(req.Path) == "" {
		hctx.Resp.BadRequest("path is required")
		return
	}

	result, err := IgnorePath(hctx.Ctx, IgnoreDeps{
		Git:     hctx.Git,
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
	}, req)

	auditEntry := AuditEntry{
		Operation: AuditOpIgnore,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handlePush handles POST /api/v1/repo/push
func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req PushRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PushToRemote(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)

	// Audit logging for push operation
	auditEntry := AuditEntry{
		Operation: AuditOpPush,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handlePull handles POST /api/v1/repo/pull
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req PullRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := PullFromRemote(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)

	// Audit logging for pull operation
	auditEntry := AuditEntry{
		Operation: AuditOpPull,
		RepoDir:   hctx.RepoDir,
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
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handleUpstreamAction handles POST /api/v1/repo/upstream-action
func (s *Server) handleUpstreamAction(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 60*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req UpstreamActionRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := RunUpstreamAction(hctx.Ctx, PushPullDeps{
		Git:       hctx.Git,
		RepoDir:   hctx.RepoDir,
		CredStore: s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}
	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
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

// handleListCredentials handles GET /api/v1/credentials
func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	result, err := ListCredentials(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleSaveCredential handles POST /api/v1/credentials
func (s *Server) handleSaveCredential(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CredentialSaveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := SaveCredential(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handleDeleteCredential handles DELETE /api/v1/credentials/{id}
func (s *Server) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		resp.BadRequest("credential ID is required")
		return
	}

	result, err := DeleteCredential(ctx, CredentialsDeps{}, CredentialDeleteRequest{ID: id})
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

// handleTestCredential handles POST /api/v1/credentials/test
func (s *Server) handleTestCredential(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CredentialTestRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := TestCredential(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleUpdateRemoteURL handles POST /api/v1/repo/remote/url
func (s *Server) handleUpdateRemoteURL(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req RemoteURLUpdateRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := UpdateRemoteURL(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// Keep this in sync with initialization/sqlite/schema.sql.
const auditSchemaSQL = `
CREATE TABLE IF NOT EXISTS git_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,
    repo_dir TEXT NOT NULL,
    branch TEXT,
    paths TEXT,
    commit_hash TEXT,
    commit_message TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON git_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON git_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_branch ON git_audit_log(branch);
CREATE INDEX IF NOT EXISTS idx_audit_log_op_created ON git_audit_log(operation, created_at DESC);
`

const repoSchemaSQL = `
CREATE TABLE IF NOT EXISTS git_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    remote_url TEXT,
    added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_opened_at TEXT,
    is_favorite INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_git_repos_last_opened ON git_repos(last_opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_git_repos_added_at ON git_repos(added_at DESC);

CREATE TABLE IF NOT EXISTS git_repo_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

func ensureAuditSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, statement := range strings.Split(auditSchemaSQL, ";") {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}

func ensureRepoSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, statement := range strings.Split(repoSchemaSQL, ";") {
		stmt := strings.TrimSpace(statement)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}

	return nil
}

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("GCT_SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(dsn, "file:") {
		return dsn, nil
	}

	dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
	if dataRoot == "" {
		dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
	}
	if dataRoot == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
	}

	return sqliteFileDSN(filepath.Join(dataRoot, "git-control-tower.db"))
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=page_size(4096)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=mmap_size(268435456)",
		path,
	), nil
}

func (s *Server) groupingConfigPath(repoID int64) (string, error) {
	return s.storageResolver.Path(
		storage.Options{ScenarioID: "git-control-tower"},
		storage.ClassConfig,
		fmt.Sprintf("%d/grouping-rules.json", repoID),
	)
}

func (s *Server) handleGetGroupingRules(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	cfg, err := LoadGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(cfg)
}

func (s *Server) handleSaveGroupingRules(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var cfg GroupingRulesConfig
	if !ParseJSONBody(w, r, &cfg) {
		return
	}

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if err := SaveGroupingRules(GroupingDeps{FS: OSFileIO{}, ConfigPath: configPath}, cfg); err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(cfg)
}

func (s *Server) handleGitignoreHealth(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	result, err := AnalyzeGitignoreHealth(HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
		GroupingDeps: GroupingDeps{
			FS:         OSFileIO{},
			ConfigPath: configPath,
		},
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

func (s *Server) handleGitignoreMove(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req GitignoreMoveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	configPath, err := s.groupingConfigPath(hctx.RepoID)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	result, err := MoveGitignoreEntry(HealthDeps{
		FS:      OSFileIO{},
		RepoDir: hctx.RepoDir,
		GroupingDeps: GroupingDeps{
			FS:         OSFileIO{},
			ConfigPath: configPath,
		},
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
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

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error { return srv.db.Close() },
	}); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
