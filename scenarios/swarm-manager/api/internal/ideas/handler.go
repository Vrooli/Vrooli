// Package ideas provides HTTP handlers for idea management.
//
// Ideas are stored as git-tracked folders with spec.json files in the
// scenarios/swarm-manager/ideas/ directory. This handler provides CRUD
// operations for managing ideas.
//
// Related PRD targets: OT-P0-001, OT-P0-002
package ideas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/ecosystem"
	"swarm-manager/internal/httputil"
)

// IdeaStatus represents the lifecycle state of an idea.
type IdeaStatus string

const (
	StatusBacklog     IdeaStatus = "backlog"
	StatusResearching IdeaStatus = "researching"
	StatusReady       IdeaStatus = "ready"
	StatusQueued      IdeaStatus = "queued"
	StatusInProgress  IdeaStatus = "in_progress"
	StatusCompleted   IdeaStatus = "completed"
	StatusArchived    IdeaStatus = "archived"
)

// Idea represents a proposal for a new scenario.
// [REQ:REQ-P0-001] Idea data structure matching spec.json schema
type Idea struct {
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      IdeaStatus `json:"status"`
	Priority    int        `json:"priority"`
	Tags        []string   `json:"tags"`
	Created     string     `json:"created"`
	Updated     string     `json:"updated"`
}

// IdeaFile represents a file or directory within an idea folder.
// [REQ:REQ-P0-004] File tree data structure for idea details page
type IdeaFile struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	Size     int64      `json:"size,omitempty,string"`
	Children []IdeaFile `json:"children,omitempty"`
}

// Handler provides HTTP handlers for idea operations.
type Handler struct {
	ideasDir        string
	ecosystemClient ecosystem.Client
	agentService    agentmanager.Service
}

// NewHandler creates a new ideas handler.
// If ideasDir is empty, it defaults to the scenario's ideas directory.
// If ecosystemClient is nil, a default HTTP client is used.
func NewHandler(ideasDir string) *Handler {
	if ideasDir == "" {
		// Default to the scenario's ideas directory
		ideasDir = filepath.Join("scenarios", "swarm-manager", "ideas")
	}
	return &Handler{
		ideasDir:        ideasDir,
		ecosystemClient: nil, // Uses default discovery-backed client
		agentService:    nil, // Uses default discovery-backed service
	}
}

// NewHandlerWithClient creates a new ideas handler with a custom ecosystem client.
// This is useful for testing without needing to mock HTTP.
func NewHandlerWithClient(ideasDir string, client ecosystem.Client) *Handler {
	h := NewHandler(ideasDir)
	h.ecosystemClient = client
	return h
}

// NewHandlerWithClients creates a new ideas handler with custom ecosystem and agent services.
func NewHandlerWithClients(ideasDir string, ecosystemClient ecosystem.Client, agentService agentmanager.Service) *Handler {
	h := NewHandler(ideasDir)
	h.ecosystemClient = ecosystemClient
	h.agentService = agentService
	return h
}

func validateIdeaStatus(status string) bool {
	switch status {
	case "backlog", "researching", "ready", "queued", "in_progress", "completed", "archived":
		return true
	default:
		return false
	}
}

func validateCreateIdeaRequest(req *apipb.CreateIdeaRequest) string {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Title) == "" {
		return "name and title are required"
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	return ""
}

func validateUpdateIdeaRequest(req *apipb.UpdateIdeaRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if !validateIdeaStatus(req.Status) {
		return "status must be a valid idea status"
	}
	if req.Priority < 1 || req.Priority > 10 {
		return "priority must be between 1 and 10"
	}
	return ""
}

func validateQueueIdeaRequest(req *apipb.QueueIdeaRequest) string {
	if req.Operation == nil || strings.TrimSpace(*req.Operation) == "" {
		return ""
	}
	switch *req.Operation {
	case "generator", "improver":
		return ""
	default:
		return "operation must be 'generator' or 'improver'"
	}
}

func ideaToProto(idea Idea) *domainpb.Idea {
	return &domainpb.Idea{
		Name:        idea.Name,
		Title:       idea.Title,
		Description: idea.Description,
		Status:      string(idea.Status),
		Priority:    int32(idea.Priority),
		Tags:        idea.Tags,
		Created:     idea.Created,
		Updated:     idea.Updated,
	}
}

func ideaFilesToProto(files []IdeaFile) []*domainpb.IdeaFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]*domainpb.IdeaFile, 0, len(files))
	for _, file := range files {
		result = append(result, ideaFileToProto(file))
	}
	return result
}

func ideaFileToProto(file IdeaFile) *domainpb.IdeaFile {
	children := ideaFilesToProto(file.Children)
	var size *int64
	if file.Type == "file" {
		size = &file.Size
	}
	return &domainpb.IdeaFile{
		Name:     file.Name,
		Path:     file.Path,
		Type:     file.Type,
		Size:     size,
		Children: children,
	}
}

// RegisterRoutes registers the ideas API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/ideas", h.List).Methods("GET")
	r.HandleFunc("/api/v1/ideas", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/ideas/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/ideas/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/ideas/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/ideas/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/ideas/{name}/files", h.UploadFile).Methods("POST")
	r.HandleFunc("/api/v1/ideas/{name}/files/{filepath:.*}", h.GetFileContent).Methods("GET")
	r.HandleFunc("/api/v1/ideas/{name}/queue", h.Queue).Methods("POST")
	r.HandleFunc("/api/v1/ideas/{name}/research", h.Research).Methods("POST")
}

// ResearchRequest captures optional fields for spawning a research agent.
type ResearchRequest struct {
	Prompt      string `json:"prompt,omitempty"`
	ScopePath   string `json:"scopePath,omitempty"`
	ProjectRoot string `json:"projectRoot,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

// ResearchMode describes the intent for idea agent work.
type ResearchMode string

const (
	ResearchModeClarify ResearchMode = "clarify"
	ResearchModeSuggest ResearchMode = "suggest"
	ResearchModeEnhance ResearchMode = "enhance"
)

func parseResearchMode(raw string) (ResearchMode, error) {
	mode := ResearchMode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return ResearchModeClarify, nil
	}
	switch mode {
	case "research":
		return ResearchModeClarify, nil
	case ResearchModeClarify, ResearchModeSuggest, ResearchModeEnhance:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid mode")
	}
}

// ResearchResponse returns agent-manager identifiers.
type ResearchResponse struct {
	TaskID  string `json:"taskId"`
	RunID   string `json:"runId"`
	BaseURL string `json:"baseUrl"`
	Created string `json:"created"`
}

// List returns all ideas.
// [REQ:REQ-P0-002] GET /api/v1/ideas endpoint
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ideas, err := h.loadAllIdeas()
	if err != nil {
		httputil.InternalError(w, "[ideas] list", err.Error())
		return
	}

	// Sort by priority (ascending) then by updated (descending)
	sort.Slice(ideas, func(i, j int) bool {
		if ideas[i].Priority != ideas[j].Priority {
			return ideas[i].Priority < ideas[j].Priority
		}
		return ideas[i].Updated > ideas[j].Updated
	})

	protoIdeas := make([]*domainpb.Idea, 0, len(ideas))
	for _, idea := range ideas {
		protoIdeas = append(protoIdeas, ideaToProto(idea))
	}

	resp := &apipb.ListIdeasResponse{Ideas: protoIdeas}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[ideas] list", "failed to encode response")
	}
}

// Get returns a single idea by name.
// [REQ:REQ-P0-002] GET /api/v1/ideas/{name} endpoint
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	idea, err := h.loadIdea(name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "idea not found")
			return
		}
		httputil.InternalError(w, "[ideas] get", err.Error())
		return
	}

	resp := &apipb.IdeaResponse{Idea: ideaToProto(idea)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[ideas] get", "failed to encode response")
	}
}

// Create creates a new idea.
// [REQ:REQ-P0-002] POST /api/v1/ideas endpoint
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateIdeaRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[ideas] create", "invalid request body")
		return
	}
	if validationErr := validateCreateIdeaRequest(&req); validationErr != "" {
		httputil.BadRequest(w, "[ideas] create", validationErr)
		return
	}

	// Sanitize name (folder-safe)
	name := sanitizeName(req.Name)

	// Check if idea already exists
	ideaDir := filepath.Join(h.ideasDir, name)
	if _, err := os.Stat(ideaDir); err == nil {
		httputil.Conflict(w, "[ideas] create", "idea already exists")
		return
	}

	// Create idea directory
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		log.Printf("[ideas] create: failed to create directory for %q: %v", name, err)
		httputil.InternalError(w, "", "failed to create idea directory")
		return
	}

	// Set defaults
	now := time.Now().UTC().Format(time.RFC3339)
	priority := 5
	if req.Priority != nil {
		priority = int(*req.Priority)
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	idea := Idea{
		Name:        name,
		Title:       req.Title,
		Description: description,
		Status:      StatusBacklog,
		Priority:    priority,
		Tags:        tags,
		Created:     now,
		Updated:     now,
	}

	// Write spec.json
	if err := h.saveIdea(idea); err != nil {
		// Clean up on failure
		os.RemoveAll(ideaDir)
		log.Printf("[ideas] create: failed to save %q: %v", name, err)
		httputil.InternalError(w, "", "failed to save idea")
		return
	}

	log.Printf("[ideas] created: %q (priority=%d, status=%s)", name, priority, StatusBacklog)
	resp := &apipb.IdeaResponse{Idea: ideaToProto(idea)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[ideas] create", "failed to encode response")
	}
}

// Update updates an existing idea.
// [REQ:REQ-P0-002] PUT /api/v1/ideas/{name} endpoint
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Load existing idea
	existing, err := h.loadIdea(name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[ideas] update", "idea not found")
			return
		}
		log.Printf("[ideas] update: failed to load %q: %v", name, err)
		httputil.InternalError(w, "", err.Error())
		return
	}

	// Decode update payload
	var update apipb.UpdateIdeaRequest
	if err := httputil.DecodeProtoJSON(r, &update); err != nil {
		httputil.BadRequest(w, "[ideas] update", "invalid request body")
		return
	}
	if validationErr := validateUpdateIdeaRequest(&update); validationErr != "" {
		httputil.BadRequest(w, "[ideas] update", validationErr)
		return
	}

	// Track what changed for logging
	oldStatus := existing.Status
	oldPriority := existing.Priority

	// Apply updates (preserve immutable fields)
	existing.Title = update.Title
	existing.Description = update.Description
	existing.Status = IdeaStatus(update.Status)
	existing.Priority = int(update.Priority)
	existing.Tags = update.Tags
	existing.Updated = time.Now().UTC().Format(time.RFC3339)

	// Save
	if err := h.saveIdea(existing); err != nil {
		log.Printf("[ideas] update: failed to save %q: %v", name, err)
		httputil.InternalError(w, "", "failed to save idea")
		return
	}

	// Log meaningful state transitions
	if oldStatus != existing.Status || oldPriority != existing.Priority {
		log.Printf("[ideas] updated: %q (status=%s→%s, priority=%d→%d)", name, oldStatus, existing.Status, oldPriority, existing.Priority)
	} else {
		log.Printf("[ideas] updated: %q", name)
	}

	resp := &apipb.IdeaResponse{Idea: ideaToProto(existing)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[ideas] update", "failed to encode response")
	}
}

// Delete removes an idea.
// [REQ:REQ-P0-002] DELETE /api/v1/ideas/{name} endpoint
//
// This operation is idempotent: calling DELETE on a non-existent resource
// returns 204 (success), not 404. This makes the operation replay-safe
// for retries and network failures.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	ideaDir := filepath.Join(h.ideasDir, name)

	// Check if idea exists - if not, still return success (idempotent delete)
	if _, err := os.Stat(ideaDir); os.IsNotExist(err) {
		// Resource already doesn't exist - operation is complete (idempotent)
		log.Printf("[ideas] delete: %q (already gone, no-op)", name)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := os.RemoveAll(ideaDir); err != nil {
		httputil.InternalError(w, "[ideas] delete", "failed to delete idea")
		return
	}

	log.Printf("[ideas] deleted: %q", name)
	w.WriteHeader(http.StatusNoContent)
}

// ListFiles returns the file tree for an idea.
// [REQ:REQ-P0-004] GET /api/v1/ideas/{name}/files endpoint
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	ideaDir := filepath.Join(h.ideasDir, name)

	// Check if idea exists
	if _, err := os.Stat(ideaDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "idea not found")
		return
	}

	files, err := h.buildFileTree(ideaDir, "")
	if err != nil {
		httputil.InternalError(w, "[ideas] list files", "failed to read idea files")
		return
	}

	log.Printf("[ideas] list files: %q (%d items)", name, len(files))
	resp := &apipb.IdeaFilesResponse{Files: ideaFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[ideas] list files", "failed to encode response")
	}
}

// buildFileTree recursively builds a file tree structure.
func (h *Handler) buildFileTree(baseDir, relativePath string) ([]IdeaFile, error) {
	currentDir := baseDir
	if relativePath != "" {
		currentDir = filepath.Join(baseDir, relativePath)
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return nil, err
	}

	var files []IdeaFile
	for _, entry := range entries {
		entryPath := entry.Name()
		if relativePath != "" {
			entryPath = filepath.Join(relativePath, entry.Name())
		}

		file := IdeaFile{
			Name: entry.Name(),
			Path: entryPath,
		}

		if entry.IsDir() {
			file.Type = "directory"
			children, err := h.buildFileTree(baseDir, entryPath)
			if err != nil {
				return nil, err
			}
			file.Children = children
		} else {
			file.Type = "file"
			info, err := entry.Info()
			if err == nil {
				file.Size = info.Size()
			}
		}

		files = append(files, file)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []IdeaFile{}
	}
	return files, nil
}

// loadAllIdeas reads all ideas from the ideas directory.
func (h *Handler) loadAllIdeas() ([]Idea, error) {
	var ideas []Idea

	err := filepath.WalkDir(h.ideasDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// If ideas dir doesn't exist, return empty list
			if os.IsNotExist(err) && path == h.ideasDir {
				return nil
			}
			return err
		}

		// Only look at spec.json files in immediate subdirectories
		if d.IsDir() && path != h.ideasDir {
			specPath := filepath.Join(path, "spec.json")
			if _, err := os.Stat(specPath); err == nil {
				idea, err := h.loadIdeaFromPath(specPath)
				if err == nil {
					ideas = append(ideas, idea)
				}
			}
			return fs.SkipDir // Don't descend further
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if ideas == nil {
		ideas = []Idea{}
	}
	return ideas, nil
}

// loadIdea reads a single idea by name.
func (h *Handler) loadIdea(name string) (Idea, error) {
	specPath := filepath.Join(h.ideasDir, name, "spec.json")
	return h.loadIdeaFromPath(specPath)
}

// loadIdeaFromPath reads an idea from a spec.json file.
func (h *Handler) loadIdeaFromPath(specPath string) (Idea, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return Idea{}, err
	}

	var idea Idea
	if err := json.Unmarshal(data, &idea); err != nil {
		return Idea{}, err
	}

	// Ensure name matches directory
	idea.Name = filepath.Base(filepath.Dir(specPath))
	return idea, nil
}

// saveIdea writes an idea to its spec.json file.
func (h *Handler) saveIdea(idea Idea) error {
	specPath := filepath.Join(h.ideasDir, idea.Name, "spec.json")
	data, err := json.MarshalIndent(idea, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(specPath, data, 0o644)
}

// GetFileContent returns the content of a specific file within an idea.
// [REQ:REQ-P0-004] GET /api/v1/ideas/{name}/files/{filepath} endpoint for file preview
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	filePath := vars["filepath"]

	ideaDir := filepath.Join(h.ideasDir, name)

	// Check if idea exists
	if _, err := os.Stat(ideaDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "idea not found")
		return
	}

	// Validate path is within idea directory (prevents path traversal)
	fullPath, valid := httputil.SafeFilePath(ideaDir, filePath)
	if !valid {
		httputil.BadRequest(w, "", "invalid file path")
		return
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		httputil.NotFound(w, "", "file not found")
		return
	}
	if err != nil {
		httputil.InternalError(w, "[ideas] get file content", "failed to read file")
		return
	}

	// Don't allow reading directories
	if info.IsDir() {
		httputil.BadRequest(w, "", "cannot read directory content")
		return
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		httputil.InternalError(w, "[ideas] get file content", "failed to read file")
		return
	}

	// Determine content type based on extension
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := getContentType(ext)

	log.Printf("[ideas] get file content: %s/%s (%d bytes)", name, filePath, len(content))
	w.Header().Set("Content-Type", contentType)
	if _, err := w.Write(content); err != nil {
		log.Printf("[ideas] get file content: failed to write response: %v", err)
	}
}

// UploadFile handles file uploads to an idea folder.
// [REQ:REQ-P0-004] POST /api/v1/ideas/{name}/files endpoint for file upload
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	ideaDir := filepath.Join(h.ideasDir, name)

	// Check if idea exists
	if _, err := os.Stat(ideaDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "idea not found")
		return
	}

	// Parse multipart form (max 32MB)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httputil.BadRequest(w, "[ideas] upload file", "failed to parse upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.BadRequest(w, "[ideas] upload file", "file is required")
		return
	}
	defer file.Close()

	// Get optional path parameter for subdirectory
	targetPath := r.FormValue("path")
	if targetPath == "" {
		targetPath = header.Filename
	} else {
		targetPath = filepath.Join(targetPath, header.Filename)
	}

	// Validate path is within idea directory (prevents path traversal)
	fullPath, valid := httputil.SafeFilePath(ideaDir, targetPath)
	if !valid {
		httputil.BadRequest(w, "", "invalid file path")
		return
	}

	// Create parent directories if needed
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		httputil.InternalError(w, "[ideas] upload file", "failed to create directory")
		return
	}

	// Create the destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		httputil.InternalError(w, "[ideas] upload file", "failed to save file")
		return
	}
	defer dst.Close()

	// Copy the uploaded file
	written, err := dst.ReadFrom(file)
	if err != nil {
		httputil.InternalError(w, "[ideas] upload file", "failed to save file")
		return
	}

	log.Printf("[ideas] uploaded: %s/%s (%d bytes)", name, targetPath, written)

	// Return the uploaded file info
	response := IdeaFile{
		Name: header.Filename,
		Path: targetPath,
		Type: "file",
		Size: written,
	}

	resp := &apipb.IdeaFileResponse{File: ideaFileToProto(response)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[ideas] upload file", "failed to encode response")
	}
}

// getContentType returns the MIME type for a file extension.
func getContentType(ext string) string {
	switch ext {
	case ".md", ".markdown":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".js", ".mjs":
		return "application/javascript"
	case ".ts", ".tsx":
		return "text/typescript"
	case ".go":
		return "text/x-go"
	case ".py":
		return "text/x-python"
	case ".rs":
		return "text/x-rust"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".xml":
		return "application/xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/plain"
	}
}

// Queue queues an idea for processing via ecosystem-manager.
// [REQ:REQ-P0-005] POST /api/v1/ideas/{name}/queue endpoint
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Load the idea
	idea, err := h.loadIdea(name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[ideas] queue", "idea not found")
			return
		}
		log.Printf("[ideas] queue: failed to load %q: %v", name, err)
		httputil.InternalError(w, "", err.Error())
		return
	}

	// Validate status - only allow queueing from certain states
	if !isQueueableStatus(idea.Status) {
		httputil.BadRequest(w, "[ideas] queue", "idea cannot be queued from current status: "+string(idea.Status))
		return
	}

	// Parse optional request body for operation type
	var req apipb.QueueIdeaRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[ideas] queue", "invalid request body")
			return
		}
		if validationErr := validateQueueIdeaRequest(&req); validationErr != "" {
			httputil.BadRequest(w, "[ideas] queue", validationErr)
			return
		}
	}
	operation := "generator"
	if req.Operation != nil && *req.Operation != "" {
		operation = *req.Operation
	}

	// Create task in ecosystem-manager
	taskID, err := h.createEcosystemTask(r.Context(), idea, operation)
	if err != nil {
		log.Printf("[ideas] queue: failed to create ecosystem task for %q: %v", name, err)
		httputil.InternalError(w, "", "failed to create ecosystem task: "+err.Error())
		return
	}

	// Update idea status to queued
	oldStatus := idea.Status
	idea.Status = StatusQueued
	idea.Updated = time.Now().UTC().Format(time.RFC3339)
	if err := h.saveIdea(idea); err != nil {
		log.Printf("[ideas] queue: failed to update status for %q: %v", name, err)
		httputil.InternalError(w, "", "failed to update idea status")
		return
	}

	log.Printf("[ideas] queued: %q (status=%s→%s, taskId=%s, operation=%s)", name, oldStatus, StatusQueued, taskID, operation)

	response := &apipb.QueueIdeaResponse{
		Idea:   ideaToProto(idea),
		TaskId: taskID,
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, response); err != nil {
		httputil.InternalError(w, "[ideas] queue", "failed to encode response")
	}
}

// Research spawns a research agent via agent-manager for the specified idea.
// [REQ:REQ-P1-004-API] Research agent spawn API
func (h *Handler) Research(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	idea, err := h.loadIdea(name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[ideas] research", "idea not found")
			return
		}
		httputil.InternalError(w, "[ideas] research", "failed to load idea")
		return
	}

	var req ResearchRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.BadRequest(w, "[ideas] research", "invalid request body")
			return
		}
	}

	mode, modeErr := parseResearchMode(req.Mode)
	if modeErr != nil {
		httputil.BadRequest(w, "[ideas] research", "mode must be clarify, suggest, or enhance")
		return
	}

	scopePath := strings.TrimSpace(req.ScopePath)
	if scopePath == "" {
		scopePath = filepath.Join(h.ideasDir, idea.Name)
	}
	projectRoot := strings.TrimSpace(req.ProjectRoot)
	if projectRoot == "" {
		projectRoot = "."
	}

	service := h.agentService
	if service == nil {
		h.agentService = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		service = h.agentService
	}

	response, err := service.SpawnResearch(r.Context(), agentmanager.ResearchSpawnRequest{
		IdeaName:    idea.Name,
		Mode:        string(mode),
		Title:       buildResearchTitle(idea, mode),
		Description: buildResearchDescription(idea, mode),
		Prompt:      strings.TrimSpace(req.Prompt),
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		CreatedBy:   "swarm-manager",
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[ideas] research", "agent-manager is not available")
			return
		}
		httputil.InternalError(w, "[ideas] research", "failed to spawn research agent")
		return
	}

	if err := httputil.JSONWithStatus(w, http.StatusCreated, ResearchResponse{
		TaskID:  response.TaskID,
		RunID:   response.RunID,
		BaseURL: response.BaseURL,
		Created: response.CreatedAt,
	}); err != nil {
		httputil.InternalError(w, "[ideas] research", "failed to encode response")
	}
}

// isQueueableStatus checks if an idea can be queued from its current status.
func isQueueableStatus(status IdeaStatus) bool {
	switch status {
	case StatusBacklog, StatusResearching, StatusReady:
		return true
	default:
		return false
	}
}

// createEcosystemTask creates a task in the ecosystem-manager queue.
// This method uses the seam pattern - if an ecosystem client is injected,
// it uses that; otherwise it uses the default discovery-backed client.
func (h *Handler) createEcosystemTask(ctx context.Context, idea Idea, operation string) (string, error) {
	// Build category from tags
	category := "uncategorized"
	if len(idea.Tags) > 0 {
		category = idea.Tags[0]
	}

	// Create the task request
	req := ecosystem.CreateTaskRequest{
		Title:     "Generate scenario from idea: " + idea.Title,
		Operation: operation,
		Priority:  idea.Priority,
		Category:  category,
	}

	// Use injected client if available (seam for testing)
	client := h.ecosystemClient
	if client == nil {
		client = ecosystem.NewHTTPClient()
	}

	return client.CreateTask(ctx, req)
}

// sanitizeName converts a name to a folder-safe format.
func sanitizeName(name string) string {
	// Convert to lowercase, replace spaces with hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func buildResearchTitle(idea Idea, mode ResearchMode) string {
	label := strings.TrimSpace(idea.Title)
	if label == "" {
		label = strings.TrimSpace(idea.Name)
	}
	if label == "" {
		label = "idea"
	}
	switch mode {
	case ResearchModeClarify:
		return "Clarify idea: " + label
	case ResearchModeSuggest:
		return "Suggest improvements: " + label
	case ResearchModeEnhance:
		return "Enhance idea: " + label
	default:
		return "Research idea: " + label
	}
}

func buildResearchDescription(idea Idea, mode ResearchMode) string {
	var builder strings.Builder
	builder.WriteString("Research this scenario idea and summarize actionable next steps.\n\n")
	builder.WriteString("Title: ")
	builder.WriteString(idea.Title)
	builder.WriteString("\n")
	builder.WriteString("Description: ")
	if strings.TrimSpace(idea.Description) == "" {
		builder.WriteString("No description provided.\n")
	} else {
		builder.WriteString(idea.Description)
		builder.WriteString("\n")
	}
	builder.WriteString("Status: ")
	builder.WriteString(string(idea.Status))
	builder.WriteString("\n")
	builder.WriteString("Priority: ")
	builder.WriteString(fmt.Sprintf("%d", idea.Priority))
	builder.WriteString("\n")
	if len(idea.Tags) > 0 {
		builder.WriteString("Tags: ")
		builder.WriteString(strings.Join(idea.Tags, ", "))
		builder.WriteString("\n")
	}

	switch mode {
	case ResearchModeClarify:
		builder.WriteString("\nGoal: generate the most important clarifying questions about scope, requirements, constraints, and implementation details.\n")
		builder.WriteString("Write results to clarify/questions.json with schema:\n")
		builder.WriteString("{\"questions\":[{\"id\":\"q1\",\"question\":\"...\",\"answer\":\"\"}]}\n")
		builder.WriteString("Preserve existing questions and answers; append new questions if needed.\n")
	case ResearchModeSuggest:
		builder.WriteString("\nGoal: propose improvements or alternative approaches for this idea.\n")
		builder.WriteString("Write results to suggest/suggestions.json with schema:\n")
		builder.WriteString("{\"suggestions\":[{\"id\":\"s1\",\"suggestion\":\"...\",\"details\":\"...\",\"status\":\"pending\"}]}\n")
		builder.WriteString("Preserve existing suggestions and decisions; append new suggestions if needed.\n")
	case ResearchModeEnhance:
		builder.WriteString("\nGoal: produce a refined plan based on clarifications and accepted suggestions.\n")
		builder.WriteString("Read clarify/questions.json and suggest/suggestions.json if present. Apply accepted suggestions, ignore rejected ones.\n")
		builder.WriteString("Write the enhancement summary to enhance/summary.md and update spec.json if necessary.\n")
	default:
		builder.WriteString("\nProvide a concise research summary, risks, and recommended next actions. ")
		builder.WriteString("If applicable, suggest updates to notes or spec.json fields.\n")
	}

	return builder.String()
}
