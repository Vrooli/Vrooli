package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    string                 `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Dependencies map[string]interface{} `json:"dependencies"`
}

// RepositoryStatus represents the current git repository status
type RepositoryStatus struct {
	Branch    string        `json:"branch"`
	Tracking  TrackingInfo  `json:"tracking"`
	Staged    []ChangedFile `json:"staged"`
	Unstaged  []ChangedFile `json:"unstaged"`
	Untracked []string      `json:"untracked"`
	Conflicts []string      `json:"conflicts"`
}

// TrackingInfo represents branch tracking status
type TrackingInfo struct {
	Remote string `json:"remote"`
	Branch string `json:"branch"`
	Ahead  int    `json:"ahead"`
	Behind int    `json:"behind"`
}

// ChangedFile represents a file with changes
type ChangedFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Scope     string `json:"scope,omitempty"`
}

// StageRequest represents a request to stage files
type StageRequest struct {
	Files []string `json:"files"`
	Scope string   `json:"scope,omitempty"` // Optional: stage by scope (e.g., "scenario:git-control-tower")
}

// StageResponse represents the response from stage/unstage operations
type StageResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

// CommitRequest represents a request to create a commit
type CommitRequest struct {
	Message string   `json:"message"`
	Files   []string `json:"files,omitempty"` // Optional: specific files (default: all staged)
}

// CommitResponse represents the response from commit operation
type CommitResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	CommitHash string `json:"commit_hash,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

// PreviewResponse represents the change preview and impact analysis
type PreviewResponse struct {
	Summary         ChangesSummary `json:"summary"`
	ImpactAnalysis  ImpactAnalysis `json:"impact_analysis"`
	StagedFiles     []ChangedFile  `json:"staged_files"`
	EstimatedCommit string         `json:"estimated_commit_size"`
	Recommendations []string       `json:"recommendations"`
}

// ChangesSummary provides high-level statistics about changes
type ChangesSummary struct {
	TotalFiles     int            `json:"total_files"`
	FilesModified  int            `json:"files_modified"`
	FilesAdded     int            `json:"files_added"`
	FilesDeleted   int            `json:"files_deleted"`
	TotalAdditions int            `json:"total_additions"`
	TotalDeletions int            `json:"total_deletions"`
	ByScope        map[string]int `json:"by_scope"`
	ByFileType     map[string]int `json:"by_file_type"`
}

// ImpactAnalysis assesses the deployment impact of changes
type ImpactAnalysis struct {
	Severity          string   `json:"severity"` // low, medium, high
	AffectedScenarios []string `json:"affected_scenarios"`
	AffectedResources []string `json:"affected_resources"`
	RequiresRestart   bool     `json:"requires_restart"`
	BreakingChange    bool     `json:"breaking_change"`
	TestingRequired   bool     `json:"testing_required"`
	DeploymentRisk    string   `json:"deployment_risk"` // low, medium, high
	EstimatedDowntime string   `json:"estimated_downtime"`
}

func main() {
	// Lifecycle protection check
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start git-control-tower

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable must be set")
	}

	// Initialize database
	if err := initDB(); err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
	}

	router := mux.NewRouter()

	// CORS middleware
	router.Use(corsMiddleware)

	// Health check (standard endpoint for ecosystem interoperability)
	router.HandleFunc("/health", handleHealth).Methods("GET")
	router.HandleFunc("/api/v1/health", handleHealth).Methods("GET")

	// Git status endpoints
	router.HandleFunc("/api/v1/status", handleGetStatus).Methods("GET")
	router.HandleFunc("/api/v1/diff/{path:.*}", handleGetDiff).Methods("GET")

	// Git mutation endpoints
	router.HandleFunc("/api/v1/stage", handleStage).Methods("POST")
	router.HandleFunc("/api/v1/unstage", handleUnstage).Methods("POST")
	router.HandleFunc("/api/v1/commit", handleCommit).Methods("POST")

	// Branch operations
	router.HandleFunc("/api/v1/branches", handleListBranches).Methods("GET")
	router.HandleFunc("/api/v1/branches", handleCreateBranch).Methods("POST")
	router.HandleFunc("/api/v1/checkout", handleCheckout).Methods("POST")

	// AI-assisted commit messages
	router.HandleFunc("/api/v1/commit/suggest", handleGenerateCommitMessage).Methods("POST")

	// Conflict detection
	router.HandleFunc("/api/v1/conflicts", handleDetectConflicts).Methods("GET")

	// Change preview and impact analysis
	router.HandleFunc("/api/v1/preview", handleChangePreview).Methods("GET")

	// UI routes
	router.HandleFunc("/", handleHome).Methods("GET", "HEAD")
	router.HandleFunc("/index.html", handleHome).Methods("GET", "HEAD")

	log.Printf("Git Control Tower API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func initDB() error {
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		return fmt.Errorf("POSTGRES_PORT environment variable must be set")
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "vrooli"
	}
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	if dbPass == "" {
		return fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}

	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		dbHost, dbPort, dbName, dbUser, dbPass)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return err
	}

	log.Println("Database connection established")
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get allowed origins from environment or default to localhost
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "http://localhost:36400"
		}

		// Set specific allowed origin
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

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	deps := make(map[string]interface{})

	// Check database
	dbStatus := "unhealthy"
	if db != nil && db.Ping() == nil {
		dbStatus = "healthy"
	}
	deps["database"] = map[string]string{"status": dbStatus}

	// Check git binary availability
	gitAvailable := false
	if _, err := exec.LookPath("git"); err == nil {
		gitAvailable = true
	}
	deps["git_binary"] = map[string]bool{"available": gitAvailable}

	// Check repository access
	repoAccessible := false
	vrooliRoot := getVrooliRoot()
	if vrooliRoot != "" {
		if _, err := os.Stat(filepath.Join(vrooliRoot, ".git")); err == nil {
			repoAccessible = true
		}
	}
	deps["repository_access"] = map[string]bool{"accessible": repoAccessible}

	readiness := dbStatus == "healthy" && gitAvailable && repoAccessible

	response := HealthResponse{
		Status:       "healthy",
		Service:      "git-control-tower",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Readiness:    readiness,
		Dependencies: deps,
	}

	if !readiness {
		response.Status = "degraded"
	}

	json.NewEncoder(w).Encode(response)
}

func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	status, err := getGitStatus(vrooliRoot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get git status: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(status)
}

func getGitStatus(vrooliRoot string) (*RepositoryStatus, error) {
	// Get current branch
	branch, err := runGitCommand(vrooliRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %v", err)
	}

	// Get tracking info
	tracking := getTrackingInfo(vrooliRoot, branch)

	// Get staged files
	staged := getStagedFiles(vrooliRoot)

	// Get unstaged files
	unstaged := getUnstagedFiles(vrooliRoot)

	// Get untracked files
	untracked := getUntrackedFiles(vrooliRoot)

	// Get conflicts
	conflicts := getConflicts(vrooliRoot)

	return &RepositoryStatus{
		Branch:    branch,
		Tracking:  tracking,
		Staged:    staged,
		Unstaged:  unstaged,
		Untracked: untracked,
		Conflicts: conflicts,
	}, nil
}

func handleGetDiff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	vars := mux.Vars(r)
	filePath := vars["path"]

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	// Get diff for file
	diff, err := runGitCommand(vrooliRoot, "diff", "HEAD", "--", filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get diff: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(diff))
}

// Helper functions

func getVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "Vrooli")
	}
	return ""
}

func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %v: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func getTrackingInfo(dir, branch string) TrackingInfo {
	tracking := TrackingInfo{}

	// Get remote tracking branch
	remoteBranch, err := runGitCommand(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return tracking
	}

	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) == 2 {
		tracking.Remote = parts[0]
		tracking.Branch = parts[1]
	}

	// Get ahead/behind counts
	counts, err := runGitCommand(dir, "rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err == nil {
		var ahead, behind int
		fmt.Sscanf(counts, "%d\t%d", &ahead, &behind)
		tracking.Ahead = ahead
		tracking.Behind = behind
	}

	return tracking
}

func getStagedFiles(dir string) []ChangedFile {
	output, err := runGitCommand(dir, "diff", "--cached", "--name-status")
	if err != nil || output == "" {
		return []ChangedFile{}
	}

	return parseChangedFiles(output)
}

func getUnstagedFiles(dir string) []ChangedFile {
	output, err := runGitCommand(dir, "diff", "--name-status")
	if err != nil || output == "" {
		return []ChangedFile{}
	}

	return parseChangedFiles(output)
}

func getUntrackedFiles(dir string) []string {
	output, err := runGitCommand(dir, "ls-files", "--others", "--exclude-standard")
	if err != nil || output == "" {
		return []string{}
	}

	return strings.Split(output, "\n")
}

func getConflicts(dir string) []string {
	output, err := runGitCommand(dir, "diff", "--name-only", "--diff-filter=U")
	if err != nil || output == "" {
		return []string{}
	}

	return strings.Split(output, "\n")
}

func parseChangedFiles(output string) []ChangedFile {
	files := []ChangedFile{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := mapGitStatus(parts[0])
		path := parts[1]
		scope := detectScope(path)

		files = append(files, ChangedFile{
			Path:   path,
			Status: status,
			Scope:  scope,
		})
	}

	return files
}

func mapGitStatus(status string) string {
	switch status {
	case "A":
		return "added"
	case "M":
		return "modified"
	case "D":
		return "deleted"
	case "R":
		return "renamed"
	case "C":
		return "copied"
	default:
		return "unknown"
	}
}

func detectScope(path string) string {
	if strings.HasPrefix(path, "scenarios/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return "scenario:" + parts[1]
		}
	} else if strings.HasPrefix(path, "resources/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 {
			return "resource:" + parts[1]
		}
	}
	return ""
}

func handleStage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req StageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	var filesToStage []string

	// If scope is specified, stage all files matching that scope
	if req.Scope != "" {
		status := RepositoryStatus{}
		status.Unstaged = getUnstagedFiles(vrooliRoot)
		status.Untracked = getUntrackedFiles(vrooliRoot)

		for _, f := range status.Unstaged {
			if f.Scope == req.Scope {
				filesToStage = append(filesToStage, f.Path)
			}
		}
		for _, f := range status.Untracked {
			if detectScope(f) == req.Scope {
				filesToStage = append(filesToStage, f)
			}
		}
	} else {
		filesToStage = req.Files
	}

	if len(filesToStage) == 0 {
		json.NewEncoder(w).Encode(StageResponse{
			Success: false,
			Message: "No files to stage",
			Files:   []string{},
		})
		return
	}

	// Stage files using git add
	args := append([]string{"add"}, filesToStage...)
	if _, err := runGitCommand(vrooliRoot, args...); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stage files: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit entry
	logAudit("stage", filesToStage)

	json.NewEncoder(w).Encode(StageResponse{
		Success: true,
		Message: fmt.Sprintf("Staged %d file(s)", len(filesToStage)),
		Files:   filesToStage,
	})
}

func handleUnstage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req StageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	var filesToUnstage []string

	// If scope is specified, unstage all files matching that scope
	if req.Scope != "" {
		status := RepositoryStatus{}
		status.Staged = getStagedFiles(vrooliRoot)

		for _, f := range status.Staged {
			if f.Scope == req.Scope {
				filesToUnstage = append(filesToUnstage, f.Path)
			}
		}
	} else {
		filesToUnstage = req.Files
	}

	if len(filesToUnstage) == 0 {
		json.NewEncoder(w).Encode(StageResponse{
			Success: false,
			Message: "No files to unstage",
			Files:   []string{},
		})
		return
	}

	// Unstage files using git reset
	args := append([]string{"reset", "HEAD"}, filesToUnstage...)
	if _, err := runGitCommand(vrooliRoot, args...); err != nil {
		http.Error(w, fmt.Sprintf("Failed to unstage files: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit entry
	logAudit("unstage", filesToUnstage)

	json.NewEncoder(w).Encode(StageResponse{
		Success: true,
		Message: fmt.Sprintf("Unstaged %d file(s)", len(filesToUnstage)),
		Files:   filesToUnstage,
	})
}

func handleCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Commit message is required", http.StatusBadRequest)
		return
	}

	// Validate commit message format (conventional commits)
	if !validateCommitMessage(req.Message) {
		http.Error(w, "Commit message does not follow conventional commit format. Expected: type(scope): description", http.StatusBadRequest)
		return
	}

	// If specific files are provided, stage them first
	if len(req.Files) > 0 {
		args := append([]string{"add"}, req.Files...)
		if _, err := runGitCommand(vrooliRoot, args...); err != nil {
			http.Error(w, fmt.Sprintf("Failed to stage files: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Create commit
	output, err := runGitCommand(vrooliRoot, "commit", "-m", req.Message)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create commit: %v", err), http.StatusInternalServerError)
		return
	}

	// Get commit hash
	hash, err := runGitCommand(vrooliRoot, "rev-parse", "HEAD")
	if err != nil {
		log.Printf("Warning: Failed to get commit hash: %v", err)
		hash = ""
	}

	// Log audit entry
	logAudit("commit", []string{hash})

	json.NewEncoder(w).Encode(CommitResponse{
		Success:    true,
		Message:    "Commit created successfully",
		CommitHash: hash,
		Summary:    output,
	})
}

func validateCommitMessage(message string) bool {
	// Conventional commit format: type(scope): description
	// Examples: feat(api): add new endpoint, fix: resolve bug, docs: update readme
	// Also accept simple format without scope: type: description

	// Split on first colon
	parts := strings.SplitN(message, ":", 2)
	if len(parts) < 2 {
		return false
	}

	firstPart := strings.TrimSpace(parts[0])

	// Check for type(scope) or just type
	if strings.Contains(firstPart, "(") {
		// Has scope - validate format
		if !strings.HasSuffix(firstPart, ")") {
			return false
		}
	}

	// Valid types
	validTypes := map[string]bool{
		"feat":     true,
		"fix":      true,
		"docs":     true,
		"style":    true,
		"refactor": true,
		"perf":     true,
		"test":     true,
		"chore":    true,
		"ci":       true,
		"build":    true,
		"revert":   true,
	}

	// Extract type
	typeStr := strings.Split(firstPart, "(")[0]
	typeStr = strings.TrimSpace(typeStr)

	return validTypes[typeStr]
}

// Branch operations

// BranchInfo represents information about a git branch
type BranchInfo struct {
	Name       string `json:"name"`
	Current    bool   `json:"current"`
	CommitHash string `json:"commit_hash,omitempty"`
}

// BranchListResponse represents the response from listing branches
type BranchListResponse struct {
	Branches []BranchInfo `json:"branches"`
}

// BranchCreateRequest represents a request to create a branch
type BranchCreateRequest struct {
	Name       string `json:"name"`
	StartPoint string `json:"start_point,omitempty"` // Optional: branch/commit to start from
}

// BranchCreateResponse represents the response from creating a branch
type BranchCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Branch  string `json:"branch,omitempty"`
}

// BranchCheckoutRequest represents a request to checkout a branch
type BranchCheckoutRequest struct {
	Branch string `json:"branch"`
}

// BranchCheckoutResponse represents the response from checking out a branch
type BranchCheckoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Branch  string `json:"branch,omitempty"`
}

// CommitMessageRequest represents a request to generate a commit message
type CommitMessageRequest struct {
	Context   string   `json:"context,omitempty"`    // Optional context for AI
	Files     []string `json:"files,omitempty"`      // Optional specific files
	MaxLength int      `json:"max_length,omitempty"` // Optional max length (default: 72)
}

// CommitMessageResponse represents the response from commit message generation
type CommitMessageResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions"`
	Model       string   `json:"model,omitempty"`
}

// ConflictInfo represents detailed conflict information
type ConflictInfo struct {
	File          string `json:"file"`
	ConflictType  string `json:"conflict_type"` // merge, rebase, cherry-pick
	OurChanges    string `json:"our_changes,omitempty"`
	TheirChanges  string `json:"their_changes,omitempty"`
	ConflictLines int    `json:"conflict_lines"`
}

// ConflictDetectionResponse represents the response from conflict detection
type ConflictDetectionResponse struct {
	Success           bool           `json:"success"`
	HasConflicts      bool           `json:"has_conflicts"`
	ConflictCount     int            `json:"conflict_count"`
	Conflicts         []ConflictInfo `json:"conflicts"`
	PotentialConflict bool           `json:"potential_conflict"` // Indicates local changes vs remote
	Message           string         `json:"message"`
}

func handleListBranches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	// Get all branches
	output, err := runGitCommand(vrooliRoot, "branch", "-a", "--format=%(refname:short)|%(HEAD)|%(objectname:short)")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list branches: %v", err), http.StatusInternalServerError)
		return
	}

	branches := []BranchInfo{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		branches = append(branches, BranchInfo{
			Name:       parts[0],
			Current:    parts[1] == "*",
			CommitHash: parts[2],
		})
	}

	json.NewEncoder(w).Encode(BranchListResponse{
		Branches: branches,
	})
}

func handleCreateBranch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req BranchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Branch name is required", http.StatusBadRequest)
		return
	}

	// Validate branch name (no spaces, special characters)
	if strings.ContainsAny(req.Name, " \t\n\r") {
		http.Error(w, "Branch name cannot contain whitespace", http.StatusBadRequest)
		return
	}

	// Create branch
	var err error
	if req.StartPoint != "" {
		_, err = runGitCommand(vrooliRoot, "branch", req.Name, req.StartPoint)
	} else {
		_, err = runGitCommand(vrooliRoot, "branch", req.Name)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create branch: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit entry
	logAudit("branch_create", map[string]string{"branch": req.Name})

	json.NewEncoder(w).Encode(BranchCreateResponse{
		Success: true,
		Message: fmt.Sprintf("Branch '%s' created successfully", req.Name),
		Branch:  req.Name,
	})
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req BranchCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Branch == "" {
		http.Error(w, "Branch name is required", http.StatusBadRequest)
		return
	}

	// Check for uncommitted changes
	status := RepositoryStatus{}
	status.Staged = getStagedFiles(vrooliRoot)
	status.Unstaged = getUnstagedFiles(vrooliRoot)

	if len(status.Staged) > 0 || len(status.Unstaged) > 0 {
		http.Error(w, "Cannot checkout with uncommitted changes. Commit or stash changes first.", http.StatusConflict)
		return
	}

	// Checkout branch
	_, err := runGitCommand(vrooliRoot, "checkout", req.Branch)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to checkout branch: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit entry
	logAudit("branch_checkout", map[string]string{"branch": req.Branch})

	json.NewEncoder(w).Encode(BranchCheckoutResponse{
		Success: true,
		Message: fmt.Sprintf("Checked out branch '%s'", req.Branch),
		Branch:  req.Branch,
	})
}

func handleGenerateCommitMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	var req CommitMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - use defaults
		req = CommitMessageRequest{}
	}

	// Set defaults
	if req.MaxLength == 0 {
		req.MaxLength = 72
	}

	// Get current changes to analyze
	status, err := getGitStatus(vrooliRoot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get git status: %v", err), http.StatusInternalServerError)
		return
	}

	// Build diff summary for AI
	diffSummary := buildDiffSummary(vrooliRoot, status, req.Files)

	// Try to generate commit message with AI
	suggestions, model, err := generateCommitMessageWithAI(diffSummary, req.Context, req.MaxLength)

	if err != nil {
		// Fallback: generate simple message based on file patterns
		suggestions = generateFallbackMessages(status, req.Files)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CommitMessageResponse{
			Success:     true,
			Message:     "Generated fallback suggestions (AI unavailable)",
			Suggestions: suggestions,
			Model:       "fallback",
		})
		return
	}

	json.NewEncoder(w).Encode(CommitMessageResponse{
		Success:     true,
		Message:     "Generated commit message suggestions",
		Suggestions: suggestions,
		Model:       model,
	})
}

func buildDiffSummary(vrooliRoot string, status *RepositoryStatus, specificFiles []string) string {
	var summary strings.Builder

	// Determine which files to analyze
	filesToAnalyze := specificFiles
	if len(filesToAnalyze) == 0 {
		// Use staged files if no specific files requested
		for _, f := range status.Staged {
			filesToAnalyze = append(filesToAnalyze, f.Path)
		}
	}

	// Limit to first 10 files to avoid token overload
	if len(filesToAnalyze) > 10 {
		filesToAnalyze = filesToAnalyze[:10]
	}

	summary.WriteString("Changes summary:\n")
	for _, file := range filesToAnalyze {
		// Get diff stats
		diff, err := runGitCommand(vrooliRoot, "diff", "--stat", "HEAD", file)
		if err == nil && diff != "" {
			summary.WriteString(fmt.Sprintf("- %s\n", diff))
		}
	}

	return summary.String()
}

func generateCommitMessageWithAI(diffSummary, context string, maxLength int) ([]string, string, error) {
	// Check for Ollama first (preferred for local)
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Try Ollama
	suggestions, err := queryOllama(ollamaURL, diffSummary, context, maxLength)
	if err == nil {
		return suggestions, "ollama", nil
	}

	// Try OpenRouter as fallback
	openRouterKey := os.Getenv("OPENROUTER_API_KEY")
	if openRouterKey != "" {
		suggestions, err := queryOpenRouter(openRouterKey, diffSummary, context, maxLength)
		if err == nil {
			return suggestions, "openrouter", nil
		}
	}

	return nil, "", fmt.Errorf("no AI service available")
}

func queryOllama(baseURL, diffSummary, context string, maxLength int) ([]string, error) {
	// Build prompt
	prompt := fmt.Sprintf(`You are an expert at writing conventional commit messages. Given the following code changes, generate 3 commit message suggestions following conventional commit format (type(scope): description).

Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert

Changes:
%s

%s

Provide exactly 3 commit message suggestions, one per line, each under %d characters. Be specific and descriptive.`,
		diffSummary,
		context,
		maxLength)

	// Query Ollama API
	reqBody := map[string]interface{}{
		"model":  "llama3:8b", // Default small model
		"prompt": prompt,
		"stream": false,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(baseURL+"/api/generate", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	response, ok := result["response"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Parse response into suggestions
	suggestions := parseAISuggestions(response)
	return suggestions, nil
}

func queryOpenRouter(apiKey, diffSummary, context string, maxLength int) ([]string, error) {
	// Similar implementation for OpenRouter
	// Placeholder for now - would use OpenRouter API
	return nil, fmt.Errorf("openrouter not yet implemented")
}

func parseAISuggestions(response string) []string {
	lines := strings.Split(response, "\n")
	suggestions := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove numbering (1., 2., etc.) and quotes
		line = strings.TrimPrefix(line, "1. ")
		line = strings.TrimPrefix(line, "2. ")
		line = strings.TrimPrefix(line, "3. ")
		line = strings.Trim(line, "\"'")

		// Validate format
		if validateCommitMessage(line) {
			suggestions = append(suggestions, line)
		}

		if len(suggestions) >= 3 {
			break
		}
	}

	return suggestions
}

func generateFallbackMessages(status *RepositoryStatus, specificFiles []string) []string {
	// Simple rule-based fallback when AI is unavailable
	suggestions := []string{}

	// Detect patterns in changed files
	hasAPI := false
	hasUI := false
	hasDocs := false
	hasTests := false

	files := specificFiles
	if len(files) == 0 {
		for _, f := range status.Staged {
			files = append(files, f.Path)
		}
	}

	for _, file := range files {
		if strings.Contains(file, "/api/") || strings.HasSuffix(file, ".go") {
			hasAPI = true
		}
		if strings.Contains(file, "/ui/") || strings.Contains(file, ".tsx") || strings.Contains(file, ".jsx") {
			hasUI = true
		}
		if strings.HasSuffix(file, ".md") || strings.Contains(file, "/docs/") {
			hasDocs = true
		}
		if strings.Contains(file, "_test.") || strings.Contains(file, "/test/") {
			hasTests = true
		}
	}

	// Generate suggestions based on patterns
	if hasTests {
		suggestions = append(suggestions, "test: add test coverage for core functionality")
	}
	if hasAPI {
		suggestions = append(suggestions, "feat(api): implement new endpoint")
	}
	if hasUI {
		suggestions = append(suggestions, "feat(ui): update user interface")
	}
	if hasDocs {
		suggestions = append(suggestions, "docs: update documentation")
	}

	// Generic fallback
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "chore: update codebase")
	}

	// Pad to 3 suggestions
	for len(suggestions) < 3 {
		suggestions = append(suggestions, fmt.Sprintf("feat: update files (%d changed)", len(files)))
	}

	return suggestions
}

func handleDetectConflicts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vrooliRoot := getVrooliRoot()
	if vrooliRoot == "" {
		http.Error(w, "VROOLI_ROOT not set", http.StatusInternalServerError)
		return
	}

	// Get current status
	status, err := getGitStatus(vrooliRoot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get git status: %v", err), http.StatusInternalServerError)
		return
	}

	// Check for active merge conflicts
	conflicts := []ConflictInfo{}
	for _, file := range status.Conflicts {
		conflictInfo := analyzeConflict(vrooliRoot, file)
		conflicts = append(conflicts, conflictInfo)
	}

	// Check for potential conflicts with remote
	potentialConflict := false
	message := "No conflicts detected"

	if len(conflicts) > 0 {
		message = fmt.Sprintf("Found %d file(s) with active merge conflicts", len(conflicts))
	} else if status.Tracking.Behind > 0 && (len(status.Staged) > 0 || len(status.Unstaged) > 0) {
		potentialConflict = true
		message = fmt.Sprintf("Potential conflicts: %d commit(s) behind remote with local changes", status.Tracking.Behind)
	} else if status.Tracking.Behind > 0 {
		message = fmt.Sprintf("Repository is %d commit(s) behind remote (no local changes)", status.Tracking.Behind)
	} else {
		message = "No conflicts or potential conflicts detected"
	}

	json.NewEncoder(w).Encode(ConflictDetectionResponse{
		Success:           true,
		HasConflicts:      len(conflicts) > 0,
		ConflictCount:     len(conflicts),
		Conflicts:         conflicts,
		PotentialConflict: potentialConflict,
		Message:           message,
	})
}

func analyzeConflict(vrooliRoot, file string) ConflictInfo {
	info := ConflictInfo{
		File:         file,
		ConflictType: "merge",
	}

	// Try to read the file and count conflict markers
	fullPath := filepath.Join(vrooliRoot, file)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return info
	}

	// Count conflict markers
	lines := strings.Split(string(content), "\n")
	conflictLines := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "<<<<<<<") || strings.HasPrefix(line, "=======") || strings.HasPrefix(line, ">>>>>>>") {
			conflictLines++
		}
	}

	info.ConflictLines = conflictLines / 3 // Each conflict has 3 markers

	return info
}

func handleChangePreview(w http.ResponseWriter, r *http.Request) {
	vrooliRoot := getVrooliRoot()

	// Get current staged files
	cmd := exec.Command("git", "-C", vrooliRoot, "diff", "--cached", "--numstat")
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, "Failed to get staged changes", http.StatusInternalServerError)
		return
	}

	// Parse the numstat output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	summary := ChangesSummary{
		ByScope:    make(map[string]int),
		ByFileType: make(map[string]int),
	}

	var stagedFiles []ChangedFile
	affectedScenarios := make(map[string]bool)
	affectedResources := make(map[string]bool)
	requiresRestart := false
	breakingChange := false

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		additions := 0
		deletions := 0
		if parts[0] != "-" {
			fmt.Sscanf(parts[0], "%d", &additions)
		}
		if parts[1] != "-" {
			fmt.Sscanf(parts[1], "%d", &deletions)
		}

		filePath := parts[2]
		scope := detectScope(filePath)

		// Determine status
		status := "modified"
		if additions > 0 && deletions == 0 {
			status = "added"
			summary.FilesAdded++
		} else if additions == 0 && deletions > 0 {
			status = "deleted"
			summary.FilesDeleted++
		} else {
			summary.FilesModified++
		}

		stagedFiles = append(stagedFiles, ChangedFile{
			Path:      filePath,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Scope:     scope,
		})

		summary.TotalAdditions += additions
		summary.TotalDeletions += deletions
		summary.TotalFiles++

		// Track by scope
		if scope != "" {
			summary.ByScope[scope]++

			// Extract scenario/resource name
			if strings.HasPrefix(scope, "scenario:") {
				affectedScenarios[strings.TrimPrefix(scope, "scenario:")] = true
			} else if strings.HasPrefix(scope, "resource:") {
				affectedResources[strings.TrimPrefix(scope, "resource:")] = true
			}
		}

		// Track by file type
		ext := filepath.Ext(filePath)
		if ext == "" {
			ext = "no-extension"
		}
		summary.ByFileType[ext]++

		// Check if changes require restart
		if strings.Contains(filePath, "service.json") ||
			strings.Contains(filePath, "main.go") ||
			strings.Contains(filePath, "api/") {
			requiresRestart = true
		}

		// Check for breaking changes
		if strings.Contains(filePath, "schema.sql") ||
			strings.Contains(filePath, "/api/") && status == "deleted" {
			breakingChange = true
		}
	}

	// Build impact analysis
	severity := "low"
	if summary.TotalFiles > 10 || breakingChange {
		severity = "high"
	} else if summary.TotalFiles > 5 || requiresRestart {
		severity = "medium"
	}

	deploymentRisk := "low"
	if breakingChange {
		deploymentRisk = "high"
	} else if requiresRestart {
		deploymentRisk = "medium"
	}

	estimatedDowntime := "0s"
	if requiresRestart {
		estimatedDowntime = "10-30s"
	}
	if breakingChange {
		estimatedDowntime = "1-5m"
	}

	scenarios := make([]string, 0, len(affectedScenarios))
	for s := range affectedScenarios {
		scenarios = append(scenarios, s)
	}

	resources := make([]string, 0, len(affectedResources))
	for r := range affectedResources {
		resources = append(resources, r)
	}

	impact := ImpactAnalysis{
		Severity:          severity,
		AffectedScenarios: scenarios,
		AffectedResources: resources,
		RequiresRestart:   requiresRestart,
		BreakingChange:    breakingChange,
		TestingRequired:   summary.TotalFiles > 0,
		DeploymentRisk:    deploymentRisk,
		EstimatedDowntime: estimatedDowntime,
	}

	// Generate recommendations
	recommendations := []string{}
	if summary.TotalFiles == 0 {
		recommendations = append(recommendations, "No staged changes - nothing to commit")
	}
	if breakingChange {
		recommendations = append(recommendations, "Breaking changes detected - ensure backward compatibility or coordinate with dependents")
	}
	if requiresRestart {
		recommendations = append(recommendations, "Changes require service restart - plan deployment window")
	}
	if len(scenarios) > 3 {
		recommendations = append(recommendations, "Multiple scenarios affected - consider splitting into separate commits")
	}
	if summary.TotalFiles > 20 {
		recommendations = append(recommendations, "Large changeset detected - consider breaking into smaller, focused commits")
	}
	if summary.TotalAdditions+summary.TotalDeletions > 500 {
		recommendations = append(recommendations, "High LOC delta - ensure adequate test coverage")
	}

	// Estimate commit size
	commitSize := "small"
	totalChanges := summary.TotalAdditions + summary.TotalDeletions
	if totalChanges > 100 {
		commitSize = "medium"
	}
	if totalChanges > 500 {
		commitSize = "large"
	}

	response := PreviewResponse{
		Summary:         summary,
		ImpactAnalysis:  impact,
		StagedFiles:     stagedFiles,
		EstimatedCommit: commitSize,
		Recommendations: recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func logAudit(operation string, details interface{}) {
	if db == nil {
		log.Printf("Audit log (db unavailable): operation=%s details=%v", operation, details)
		return
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Printf("Failed to marshal audit details: %v", err)
		return
	}

	_, err = db.Exec(`
		INSERT INTO git_audit_logs (operation, user, details)
		VALUES ($1, $2, $3)
	`, operation, "agent", detailsJSON)

	if err != nil {
		log.Printf("Failed to insert audit log: %v", err)
	}
}
