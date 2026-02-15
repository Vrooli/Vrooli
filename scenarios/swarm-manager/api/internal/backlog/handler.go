// Package backlog provides HTTP handlers for backlog management.
//
// Backlog items are stored as git-tracked folders with spec.json files in
// scenarios/swarm-manager/{ideas|research|fix|execute}/. This handler provides
// CRUD operations, file access, agent spawning, and conversion between backlog kinds.
//
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md
//
// Related PRD targets: OT-P0-001, OT-P0-002
package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
)

// BacklogStatus represents the lifecycle state of a backlog item.
type BacklogStatus string

const (
	StatusBacklog     BacklogStatus = "backlog"
	StatusResearching BacklogStatus = "researching"
	StatusReady       BacklogStatus = "ready"
	StatusQueued      BacklogStatus = "queued"
	StatusInProgress  BacklogStatus = "in_progress"
	StatusCompleted   BacklogStatus = "completed"
	StatusArchived    BacklogStatus = "archived"
)

// BacklogKind represents a category of backlog work.
type BacklogKind string

const (
	KindIdea     BacklogKind = "idea"
	KindResearch BacklogKind = "research"
	KindFix      BacklogKind = "fix"
	KindExecute  BacklogKind = "execute"
)

var backlogKindDirs = map[BacklogKind]string{
	KindIdea:     "ideas",
	KindResearch: "research",
	KindFix:      "fix",
	KindExecute:  "execute",
}

// BacklogItem represents a unit of work stored on disk.
type BacklogItem struct {
	Name           string        `json:"name"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	Status         BacklogStatus `json:"status"`
	Priority       int           `json:"priority"`
	Tags           []string      `json:"tags"`
	Created        string        `json:"created"`
	Updated        string        `json:"updated"`
	Kind           BacklogKind   `json:"kind"`
	ResearchTarget string        `json:"research_target,omitempty"`
}

// BacklogFile represents a file or directory within a backlog item folder.
type BacklogFile struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	Type     string        `json:"type"` // "file" or "directory"
	Size     int64         `json:"size,omitempty,string"`
	Children []BacklogFile `json:"children,omitempty"`
}

// Handler provides HTTP handlers for backlog operations.
type Handler struct {
	rootDir      string
	agentService agentmanager.Service
	promptClient promptmanager.Client
}

// researchSkillIDs maps (ResearchMode, BacklogKind) to prompt-manager skill IDs.
var researchSkillIDs = map[ResearchMode]map[BacklogKind]string{
	ResearchModeClarify: {KindIdea: "swarm-manager-clarify-idea"},
	ResearchModeSuggest: {KindIdea: "swarm-manager-suggest-idea"},
	ResearchModeEnhance: {KindIdea: "swarm-manager-enhance-idea"},
	ResearchModeResearch: {
		KindIdea:     "swarm-manager-research-idea",
		KindFix:      "swarm-manager-research-fix",
		KindExecute:  "swarm-manager-research-general",
		KindResearch: "swarm-manager-research-general",
	},
}

// processingSkillIDs maps BacklogKind to prompt-manager skill IDs.
var processingSkillIDs = map[BacklogKind]string{
	KindIdea:     "swarm-manager-process-idea",
	KindFix:      "swarm-manager-process-fix",
	KindExecute:  "swarm-manager-process-execute",
	KindResearch: "swarm-manager-process-execute",
}

// NewHandler creates a new backlog handler.
// If rootDir is empty, it defaults to the scenario root directory.
func NewHandler(rootDir string) *Handler {
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	return &Handler{
		rootDir:      rootDir,
		agentService: nil, // Uses default discovery-backed service
		promptClient: promptmanager.NewHTTPClient(),
	}
}

// NewHandlerWithClients creates a new backlog handler with custom dependencies.
func NewHandlerWithClients(rootDir string, agentService agentmanager.Service, promptClient promptmanager.Client) *Handler {
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	h := &Handler{
		rootDir:      rootDir,
		agentService: agentService,
		promptClient: promptClient,
	}
	if h.promptClient == nil {
		h.promptClient = promptmanager.NewHTTPClient()
	}
	return h
}

func validateBacklogStatus(status string) bool {
	switch status {
	case "backlog", "researching", "ready", "queued", "in_progress", "completed", "archived":
		return true
	default:
		return false
	}
}

func parseBacklogKind(raw string) (BacklogKind, error) {
	candidate := BacklogKind(strings.ToLower(strings.TrimSpace(raw)))
	if _, ok := backlogKindDirs[candidate]; ok {
		return candidate, nil
	}
	return "", fmt.Errorf("invalid backlog kind: %s", raw)
}

func validateCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.Kind) == "" {
		return "kind is required"
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	return ""
}

func normalizeCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.TrimSpace(req.Title)
	}
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	if req.ResearchTarget != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.ResearchTarget))
		if normalized == "" {
			req.ResearchTarget = nil
		} else {
			req.ResearchTarget = &normalized
		}
	}
}

func validateUpdateBacklogItemRequest(req *apipb.UpdateBacklogItemRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if !validateBacklogStatus(req.Status) {
		return "status must be a valid backlog status"
	}
	if req.Priority < 1 || req.Priority > 10 {
		return "priority must be between 1 and 10"
	}
	return ""
}

func normalizeResearchTarget(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "", nil
	}
	switch value {
	case "idea", "fix", "execute", "unspecified":
		return value, nil
	default:
		return "", fmt.Errorf("research_target must be idea, fix, execute, or unspecified")
	}
}

func backlogToProto(item BacklogItem) *domainpb.BacklogItem {
	result := &domainpb.BacklogItem{
		Name:        item.Name,
		Title:       item.Title,
		Description: item.Description,
		Status:      string(item.Status),
		Priority:    int32(item.Priority),
		Tags:        item.Tags,
		Created:     item.Created,
		Updated:     item.Updated,
		Kind:        string(item.Kind),
	}
	if strings.TrimSpace(item.ResearchTarget) != "" {
		result.ResearchTarget = &item.ResearchTarget
	}
	return result
}

func backlogFilesToProto(files []BacklogFile) []*domainpb.BacklogFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]*domainpb.BacklogFile, 0, len(files))
	for _, file := range files {
		result = append(result, backlogFileToProto(file))
	}
	return result
}

func backlogFileToProto(file BacklogFile) *domainpb.BacklogFile {
	children := backlogFilesToProto(file.Children)
	var size *int64
	if file.Type == "file" {
		size = &file.Size
	}
	return &domainpb.BacklogFile{
		Name:     file.Name,
		Path:     file.Path,
		Type:     file.Type,
		Size:     size,
		Children: children,
	}
}

// RegisterRoutes registers the backlog API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/backlog", h.List).Methods("GET")
	r.HandleFunc("/api/v1/backlog", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Update).Methods("PUT")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files", h.UploadFile).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files/{filepath:.*}", h.GetFileContent).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/queue", h.Queue).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/research", h.Research).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/convert", h.Convert).Methods("POST")
}

// ResearchMode describes the intent for idea agent work.
type ResearchMode string

const (
	ResearchModeClarify  ResearchMode = "clarify"
	ResearchModeSuggest  ResearchMode = "suggest"
	ResearchModeEnhance  ResearchMode = "enhance"
	ResearchModeResearch ResearchMode = "research"
)

func parseResearchMode(raw string) ResearchMode {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	switch candidate {
	case "clarify":
		return ResearchModeClarify
	case "suggest":
		return ResearchModeSuggest
	case "enhance":
		return ResearchModeEnhance
	case "research", "", "explore", "investigate":
		return ResearchModeResearch
	default:
		return ResearchModeResearch
	}
}

func validateResearchModeForKind(kind BacklogKind, mode ResearchMode) error {
	if kind == KindIdea {
		switch mode {
		case ResearchModeClarify, ResearchModeSuggest, ResearchModeEnhance:
			return nil
		default:
			return fmt.Errorf("mode must be clarify, suggest, or enhance")
		}
	}
	return nil
}

func normalizeResearchRequest(req *apipb.BacklogResearchRequest) {
	if req == nil {
		return
	}
	if req.Mode != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*req.Mode))
		if trimmed == "" {
			req.Mode = nil
		} else {
			req.Mode = &trimmed
		}
	}
	if req.TargetKind != nil {
		trimmed := strings.ToLower(strings.TrimSpace(*req.TargetKind))
		if trimmed == "" {
			req.TargetKind = nil
		} else {
			req.TargetKind = &trimmed
		}
	}
}

func readOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// List returns all backlog items.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	kinds, err := parseKindsQuery(r)
	if err != nil {
		httputil.BadRequest(w, "[backlog] list", err.Error())
		return
	}

	items, err := h.loadAllItems(kinds)
	if err != nil {
		httputil.InternalError(w, "[backlog] list", err.Error())
		return
	}

	// Sort by priority (ascending) then by updated (descending)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].Updated > items[j].Updated
	})

	protoItems := make([]*domainpb.BacklogItem, 0, len(items))
	for _, item := range items {
		protoItems = append(protoItems, backlogToProto(item))
	}

	resp := &apipb.ListBacklogItemsResponse{Items: protoItems}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] list", "failed to encode response")
	}
}

// Get returns a single backlog item by name.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get")
	if !ok {
		return
	}

	item, err := h.loadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] get", err.Error())
		return
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] get", "failed to encode response")
	}
}

// Create creates a new backlog item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateBacklogItemRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] create", "invalid request body")
		return
	}
	normalizeCreateBacklogItemRequest(&req)
	if !httputil.ValidateProtoRequest(w, "[backlog] create", "invalid request body", &req) {
		return
	}
	if validationErr := validateCreateBacklogItemRequest(&req); validationErr != "" {
		httputil.BadRequest(w, "[backlog] create", validationErr)
		return
	}

	kind, err := parseBacklogKind(req.Kind)
	if err != nil {
		httputil.BadRequest(w, "[backlog] create", err.Error())
		return
	}

	// Sanitize name (folder-safe). Allow title fallback.
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Title
	}
	name = sanitizeName(name)
	if name == "" {
		httputil.BadRequest(w, "[backlog] create", "name is required")
		return
	}

	itemDir := h.itemDir(kind, name)
	if _, err := os.Stat(itemDir); err == nil {
		httputil.Conflict(w, "[backlog] create", "backlog item already exists")
		return
	}

	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		log.Printf("[backlog] create: failed to create directory for %q: %v", name, err)
		httputil.InternalError(w, "", "failed to create backlog directory")
		return
	}

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

	researchTarget := ""
	if req.ResearchTarget != nil {
		normalized, err := normalizeResearchTarget(*req.ResearchTarget)
		if err != nil {
			httputil.BadRequest(w, "[backlog] create", err.Error())
			return
		}
		researchTarget = normalized
	}
	if kind != KindResearch {
		researchTarget = ""
	}

	item := BacklogItem{
		Name:           name,
		Title:          req.Title,
		Description:    description,
		Status:         StatusBacklog,
		Priority:       priority,
		Tags:           tags,
		Created:        now,
		Updated:        now,
		Kind:           kind,
		ResearchTarget: researchTarget,
	}

	if err := h.saveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		log.Printf("[backlog] create: failed to save %q: %v", name, err)
		httputil.InternalError(w, "", "failed to save backlog item")
		return
	}

	log.Printf("[backlog] created: %q (kind=%s, priority=%d, status=%s)", name, kind, priority, StatusBacklog)
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] create", "failed to encode response")
	}
}

// Update updates an existing backlog item.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update")
	if !ok {
		return
	}

	existing, err := h.loadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] update", "backlog item not found")
			return
		}
		log.Printf("[backlog] update: failed to load %q: %v", name, err)
		httputil.InternalError(w, "", err.Error())
		return
	}

	var update apipb.UpdateBacklogItemRequest
	if err := httputil.DecodeProtoJSON(r, &update); err != nil {
		httputil.BadRequest(w, "[backlog] update", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] update", "invalid request body", &update) {
		return
	}
	if validationErr := validateUpdateBacklogItemRequest(&update); validationErr != "" {
		httputil.BadRequest(w, "[backlog] update", validationErr)
		return
	}

	oldStatus := existing.Status
	oldPriority := existing.Priority

	existing.Title = update.Title
	existing.Description = update.Description
	existing.Status = BacklogStatus(update.Status)
	existing.Priority = int(update.Priority)
	existing.Tags = update.Tags
	existing.Updated = time.Now().UTC().Format(time.RFC3339)
	if existing.Kind == KindResearch && update.ResearchTarget != nil {
		normalized, err := normalizeResearchTarget(*update.ResearchTarget)
		if err != nil {
			httputil.BadRequest(w, "[backlog] update", err.Error())
			return
		}
		existing.ResearchTarget = normalized
	}
	if existing.Kind != KindResearch {
		existing.ResearchTarget = ""
	}

	if err := h.saveItem(existing); err != nil {
		log.Printf("[backlog] update: failed to save %q: %v", name, err)
		httputil.InternalError(w, "", "failed to save backlog item")
		return
	}

	if oldStatus != existing.Status || oldPriority != existing.Priority {
		log.Printf("[backlog] updated: %q (status=%s→%s, priority=%d→%d)", name, oldStatus, existing.Status, oldPriority, existing.Priority)
	} else {
		log.Printf("[backlog] updated: %q", name)
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(existing)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] update", "failed to encode response")
	}
}

// Delete deletes a backlog item by name.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete")
	if !ok {
		return
	}

	itemDir := h.itemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := os.RemoveAll(itemDir); err != nil {
		log.Printf("[backlog] delete: failed to delete %q: %v", name, err)
		httputil.InternalError(w, "", "failed to delete backlog item")
		return
	}

	log.Printf("[backlog] deleted: %q (kind=%s)", name, kind)
	w.WriteHeader(http.StatusNoContent)
}

// ListFiles returns the file tree for a backlog item.
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "files")
	if !ok {
		return
	}

	itemDir := h.itemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	files, err := h.buildFileTree(itemDir, "")
	if err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to read file tree")
		return
	}

	resp := &apipb.BacklogFilesResponse{Files: backlogFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] list files", "failed to encode response")
	}
}

func (h *Handler) buildFileTree(baseDir, relativePath string) ([]BacklogFile, error) {
	dirPath := filepath.Join(baseDir, relativePath)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]BacklogFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(relativePath, name)
		file := BacklogFile{
			Name: name,
			Path: path,
		}

		if entry.IsDir() {
			file.Type = "directory"
			children, err := h.buildFileTree(baseDir, path)
			if err == nil {
				file.Children = children
			}
		} else {
			file.Type = "file"
			if info, err := entry.Info(); err == nil {
				file.Size = info.Size()
			}
		}

		files = append(files, file)
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []BacklogFile{}
	}
	return files, nil
}

func (h *Handler) loadAllItems(kinds []BacklogKind) ([]BacklogItem, error) {
	var items []BacklogItem

	if len(kinds) == 0 {
		kinds = []BacklogKind{KindIdea, KindResearch, KindFix, KindExecute}
	}

	for _, kind := range kinds {
		kindDir := h.kindDir(kind)
		err := filepath.WalkDir(kindDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) && path == kindDir {
					return nil
				}
				return err
			}

			if d.IsDir() && path != kindDir {
				specPath := filepath.Join(path, "spec.json")
				if _, err := os.Stat(specPath); err == nil {
					item, err := h.loadItemFromPath(kind, specPath)
					if err == nil {
						items = append(items, item)
					}
				}
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if items == nil {
		items = []BacklogItem{}
	}
	return items, nil
}

func (h *Handler) loadItem(kind BacklogKind, name string) (BacklogItem, error) {
	specPath := filepath.Join(h.itemDir(kind, name), "spec.json")
	return h.loadItemFromPath(kind, specPath)
}

func (h *Handler) loadItemFromPath(kind BacklogKind, specPath string) (BacklogItem, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return BacklogItem{}, err
	}

	var item BacklogItem
	if err := json.Unmarshal(data, &item); err != nil {
		return BacklogItem{}, err
	}

	item.Name = filepath.Base(filepath.Dir(specPath))
	item.Kind = kind
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.ResearchTarget != "" && item.Kind != KindResearch {
		item.ResearchTarget = ""
	}
	// Normalize status to valid proto values. On-disk data may contain
	// legacy values (e.g. "done") that are not in the proto enum.
	if !validateBacklogStatus(string(item.Status)) {
		switch string(item.Status) {
		case "done", "complete", "finished":
			item.Status = StatusCompleted
		default:
			item.Status = StatusBacklog
		}
	}
	// Backfill missing created timestamp from updated or file mtime.
	if strings.TrimSpace(item.Created) == "" {
		if strings.TrimSpace(item.Updated) != "" {
			item.Created = item.Updated
		} else if info, statErr := os.Stat(specPath); statErr == nil {
			item.Created = info.ModTime().UTC().Format(time.RFC3339)
		} else {
			item.Created = time.Now().UTC().Format(time.RFC3339)
		}
	}
	// Ensure priority is within valid range (1-10).
	if item.Priority < 1 {
		item.Priority = 5
	} else if item.Priority > 10 {
		item.Priority = 10
	}
	return item, nil
}

func (h *Handler) saveItem(item BacklogItem) error {
	if item.Kind == "" {
		return fmt.Errorf("backlog kind is required")
	}
	specPath := filepath.Join(h.itemDir(item.Kind, item.Name), "spec.json")
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(specPath, data, 0o644)
}

// GetFileContent returns the content of a specific file within a backlog item.
func (h *Handler) GetFileContent(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get file")
	if !ok {
		return
	}
	filePath := mux.Vars(r)["filepath"]

	itemDir := h.itemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	fullPath, valid := httputil.SafeFilePath(itemDir, filePath)
	if !valid {
		httputil.BadRequest(w, "[backlog] get file", "invalid file path")
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "file not found")
			return
		}
		log.Printf("[backlog] get file content: failed to read %s/%s: %v", name, filePath, err)
		httputil.InternalError(w, "[backlog] get file content", "failed to read file")
		return
	}

	contentType := getContentType(filepath.Ext(fullPath))
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

// UploadFile handles file uploads to a backlog item folder.
func (h *Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "upload file")
	if !ok {
		return
	}

	itemDir := h.itemDir(kind, name)
	if _, err := os.Stat(itemDir); os.IsNotExist(err) {
		httputil.NotFound(w, "", "backlog item not found")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httputil.BadRequest(w, "[backlog] upload file", "failed to parse upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.BadRequest(w, "[backlog] upload file", "file is required")
		return
	}
	defer file.Close()

	targetPath := header.Filename
	if path := r.FormValue("path"); strings.TrimSpace(path) != "" {
		targetPath = filepath.Join(path, targetPath)
	}

	fullPath, valid := httputil.SafeFilePath(itemDir, targetPath)
	if !valid {
		httputil.BadRequest(w, "[backlog] upload file", "invalid file path")
		return
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		log.Printf("[backlog] upload file: failed to create directory %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to create directory")
		return
	}

	out, err := os.Create(fullPath)
	if err != nil {
		log.Printf("[backlog] upload file: failed to create file %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to save file")
		return
	}
	defer out.Close()

	written, err := out.ReadFrom(file)
	if err != nil {
		log.Printf("[backlog] upload file: failed to write file %s: %v", fullPath, err)
		httputil.InternalError(w, "[backlog] upload file", "failed to save file")
		return
	}

	log.Printf("[backlog] uploaded: %s/%s (%d bytes)", name, targetPath, written)

	fileNode := BacklogFile{
		Name: header.Filename,
		Path: targetPath,
		Type: "file",
		Size: written,
	}

	resp := &apipb.BacklogFileResponse{File: backlogFileToProto(fileNode)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] upload file", "failed to encode response")
	}
}

// Queue queues a backlog item for processing via agent-manager.
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "queue")
	if !ok {
		return
	}

	item, err := h.loadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] queue", "backlog item not found")
			return
		}
		log.Printf("[backlog] queue: failed to load %q: %v", name, err)
		httputil.InternalError(w, "", err.Error())
		return
	}

	if !isQueueableStatus(item.Status) {
		httputil.BadRequest(w, "[backlog] queue", "backlog item cannot be queued from current status: "+string(item.Status))
		return
	}

	var pbReq apipb.QueueBacklogItemRequest
	if r.Body != nil {
		if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
			// Tolerate empty bodies (all fields optional).
			if !errors.Is(err, io.EOF) && r.ContentLength != 0 {
				httputil.BadRequest(w, "[backlog] queue", "invalid request body")
				return
			}
		}
		if !httputil.ValidateProtoRequest(w, "[backlog] queue", "invalid queue request", &pbReq) {
			return
		}
	}
	operation := "generator"
	if pbReq.GetOperation() != "" {
		operation = strings.ToLower(strings.TrimSpace(pbReq.GetOperation()))
	}
	mode := execution.ModeYOLO
	if pbReq.GetMode() != "" {
		mode = execution.Mode(strings.ToLower(strings.TrimSpace(pbReq.GetMode())))
	}
	startedBy := strings.TrimSpace(pbReq.GetStartedBy())
	if startedBy == "" {
		startedBy = "swarm-manager"
	}

	if kind == KindResearch {
		httputil.BadRequest(w, "[backlog] queue", "research items must be converted before processing")
		return
	}

	executionService := execution.NewService(execution.ServiceConfig{
		RootDir:      h.rootDir,
		StorePath:    filepath.Join(h.rootDir, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(h.rootDir, ".vrooli", "execution-policy.json"),
		AgentService: h.agentService,
	})
	record, err := executionService.QueueBacklog(r.Context(), execution.CreateRequest{
		BacklogKind:  string(kind),
		BacklogName:  name,
		Mode:         mode,
		DelaySeconds: pbReq.GetDelaySeconds(),
		StartedBy:    startedBy,
		Operation:    operation,
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[backlog] queue", "agent-manager is not available")
			return
		}
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] queue", "backlog item not found")
			return
		}
		if strings.Contains(err.Error(), "cannot be queued") {
			httputil.BadRequest(w, "[backlog] queue", err.Error())
			return
		}
		httputil.InternalError(w, "[backlog] queue", "failed to queue execution")
		return
	}

	item, err = h.loadItem(kind, name)
	if err != nil {
		log.Printf("[backlog] queue: failed to reload %q after queue: %v", name, err)
		httputil.InternalError(w, "[backlog] queue", "failed to load updated backlog item")
		return
	}

	log.Printf("[backlog] queued: %q (kind=%s, status=%s, taskId=%s, executionId=%s)", name, kind, item.Status, record.TaskID, record.ExecutionID)

	resp := &apipb.QueueBacklogItemResponse{
		Item:    backlogToProto(item),
		TaskId:  record.TaskID,
		RunId:   record.RunID,
		BaseUrl: "",
		Created: record.CreatedAt,
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		httputil.InternalError(w, "[backlog] queue", "failed to encode response")
	}
}

// Research spawns a research agent via agent-manager for the specified backlog item.
func (h *Handler) Research(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "research")
	if !ok {
		return
	}

	item, err := h.loadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] research", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] research", "failed to load backlog item")
		return
	}

	var req apipb.BacklogResearchRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[backlog] research", "invalid request body")
			return
		}
		normalizeResearchRequest(&req)
		if !httputil.ValidateProtoRequest(w, "[backlog] research", "invalid request body", &req) {
			return
		}
	}

	mode := parseResearchMode(readOptionalString(req.Mode))
	if err := validateResearchModeForKind(kind, mode); err != nil {
		httputil.BadRequest(w, "[backlog] research", err.Error())
		return
	}

	if kind == KindResearch && req.TargetKind != nil {
		normalized, err := normalizeResearchTarget(*req.TargetKind)
		if err != nil {
			httputil.BadRequest(w, "[backlog] research", err.Error())
			return
		}
		item.ResearchTarget = normalized
		item.Updated = time.Now().UTC().Format(time.RFC3339)
		if err := h.saveItem(item); err != nil {
			log.Printf("[backlog] research: failed to update research target for %q: %v", name, err)
		}
	}

	scopePath := strings.TrimSpace(readOptionalString(req.ScopePath))
	if scopePath == "" {
		scopePath = h.itemDir(kind, item.Name)
	}
	projectRoot := strings.TrimSpace(readOptionalString(req.ProjectRoot))
	if projectRoot == "" {
		projectRoot = "."
	}

	service := h.agentService
	if service == nil {
		h.agentService = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		service = h.agentService
	}

	prompt, promptErr := h.fetchResearchPrompt(r.Context(), item, mode)
	if promptErr != nil {
		log.Printf("[backlog] research: prompt fetch failed: %v", promptErr)
		prompt = "Use the backlog item folder as context and perform the requested research."
	}
	if strings.TrimSpace(readOptionalString(req.Prompt)) != "" {
		prompt = prompt + "\n\nAdditional context from user:\n" + strings.TrimSpace(readOptionalString(req.Prompt))
	}

	runResult, err := service.SpawnBacklog(r.Context(), agentmanager.BacklogSpawnRequest{
		Kind:        string(kind),
		Name:        item.Name,
		Title:       buildResearchTitle(item, mode),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		CreatedBy:   "swarm-manager",
		Purpose:     "research",
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[backlog] research", "agent-manager is not available")
			return
		}
		httputil.InternalError(w, "[backlog] research", "failed to spawn research agent")
		return
	}

	resp := &apipb.BacklogResearchResponse{
		TaskId:  runResult.TaskID,
		RunId:   runResult.RunID,
		BaseUrl: runResult.BaseURL,
		Created: runResult.CreatedAt,
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] research", "failed to encode response")
	}
}

// Convert moves a backlog item to another kind.
func (h *Handler) Convert(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "convert")
	if !ok {
		return
	}

	var req apipb.ConvertBacklogItemRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] convert", "invalid request body")
		return
	}
	req.TargetKind = strings.ToLower(strings.TrimSpace(req.TargetKind))
	if strings.TrimSpace(req.TargetKind) == "" {
		httputil.BadRequest(w, "[backlog] convert", "target_kind is required")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[backlog] convert", "invalid request body", &req) {
		return
	}

	targetKind, err := parseBacklogKind(req.TargetKind)
	if err != nil {
		httputil.BadRequest(w, "[backlog] convert", err.Error())
		return
	}

	targetName := name
	if req.TargetName != nil {
		candidate := strings.TrimSpace(*req.TargetName)
		if candidate == "" {
			httputil.BadRequest(w, "[backlog] convert", "target_name is invalid")
			return
		}
		targetName = candidate
	}
	targetName = sanitizeName(targetName)
	if targetName == "" {
		httputil.BadRequest(w, "[backlog] convert", "target_name is invalid")
		return
	}

	sourceDir := h.itemDir(kind, name)
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		httputil.NotFound(w, "[backlog] convert", "backlog item not found")
		return
	}

	targetDir := h.itemDir(targetKind, targetName)
	if _, err := os.Stat(targetDir); err == nil {
		httputil.Conflict(w, "[backlog] convert", "target backlog item already exists")
		return
	}

	if err := os.MkdirAll(h.kindDir(targetKind), 0o755); err != nil {
		log.Printf("[backlog] convert: failed to create target dir %s: %v", targetDir, err)
		httputil.InternalError(w, "[backlog] convert", "failed to create target backlog directory")
		return
	}

	if err := os.Rename(sourceDir, targetDir); err != nil {
		log.Printf("[backlog] convert: failed to move %s to %s: %v", sourceDir, targetDir, err)
		httputil.InternalError(w, "[backlog] convert", "failed to move backlog item")
		return
	}

	item, err := h.loadItem(targetKind, targetName)
	if err != nil {
		httputil.InternalError(w, "[backlog] convert", "failed to load moved backlog item")
		return
	}
	item.Name = targetName
	item.Kind = targetKind
	item.Updated = time.Now().UTC().Format(time.RFC3339)
	if item.Kind != KindResearch {
		item.ResearchTarget = ""
	}

	if err := h.saveItem(item); err != nil {
		log.Printf("[backlog] convert: failed to save %q: %v", targetName, err)
		httputil.InternalError(w, "[backlog] convert", "failed to update backlog item")
		return
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] convert", "failed to encode response")
	}
}

func (h *Handler) spawnProcessingAgent(ctx context.Context, item BacklogItem, operation string) (agentmanager.RunResult, error) {
	service := h.agentService
	if service == nil {
		h.agentService = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		service = h.agentService
	}

	scopePath := h.itemDir(item.Kind, item.Name)
	prompt, promptErr := h.fetchProcessingPrompt(ctx, item, operation)
	if promptErr != nil {
		log.Printf("[backlog] process: prompt fetch failed: %v", promptErr)
		prompt = "Use the backlog item folder as context and complete the requested work."
	}

	return service.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        string(item.Kind),
		Name:        item.Name,
		Title:       buildProcessingTitle(item, operation),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   scopePath,
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "process",
	})
}

// isQueueableStatus checks if an item can be queued from its current status.
func isQueueableStatus(status BacklogStatus) bool {
	switch status {
	case StatusBacklog, StatusResearching, StatusReady:
		return true
	default:
		return false
	}
}

func (h *Handler) parseKindAndName(w http.ResponseWriter, r *http.Request, action string) (BacklogKind, string, bool) {
	vars := mux.Vars(r)
	kindRaw := vars["kind"]
	name := vars["name"]
	kind, err := parseBacklogKind(kindRaw)
	if err != nil {
		httputil.BadRequest(w, "[backlog] "+action, "invalid kind")
		return "", "", false
	}
	if strings.TrimSpace(name) == "" {
		httputil.BadRequest(w, "[backlog] "+action, "name is required")
		return "", "", false
	}
	return kind, name, true
}

func parseKindsQuery(r *http.Request) ([]BacklogKind, error) {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("kinds"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("kind"))
	}
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	kinds := make([]BacklogKind, 0, len(parts))
	for _, part := range parts {
		kind, err := parseBacklogKind(part)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

func (h *Handler) kindDir(kind BacklogKind) string {
	return filepath.Join(h.rootDir, backlogKindDirs[kind])
}

func (h *Handler) itemDir(kind BacklogKind, name string) string {
	return filepath.Join(h.kindDir(kind), name)
}

// sanitizeName converts a name to a folder-safe format.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func buildResearchTitle(item BacklogItem, mode ResearchMode) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch mode {
	case ResearchModeClarify:
		return "Clarify idea: " + label
	case ResearchModeSuggest:
		return "Suggest improvements: " + label
	case ResearchModeEnhance:
		return "Enhance idea: " + label
	default:
		return "Research: " + label
	}
}

// researchSkillID returns the prompt-manager skill ID for a research mode and item kind.
func researchSkillID(mode ResearchMode, kind BacklogKind) string {
	if kindMap, ok := researchSkillIDs[mode]; ok {
		if id, ok := kindMap[kind]; ok {
			return id
		}
		// Workflow skills (clarify/suggest/enhance) only have KindIdea.
		// For other kinds in workflow modes, fall back to idea.
		if id, ok := kindMap[KindIdea]; ok {
			return id
		}
	}
	return "swarm-manager-research-general"
}

// fetchResearchPrompt loads a research prompt from prompt-manager.
func (h *Handler) fetchResearchPrompt(ctx context.Context, item BacklogItem, mode ResearchMode) (string, error) {
	skillID := researchSkillID(mode, item.Kind)
	withScope := mode == ResearchModeResearch
	return h.promptClient.ReadSkill(ctx, skillID, buildVariableMap(item, h.itemDir(item.Kind, item.Name)), withScope)
}

// fetchProcessingPrompt loads a processing prompt from prompt-manager.
func (h *Handler) fetchProcessingPrompt(ctx context.Context, item BacklogItem, operation string) (string, error) {
	skillID := processingSkillIDs[item.Kind]
	if skillID == "" {
		skillID = "swarm-manager-process-execute"
	}
	prompt, err := h.promptClient.ReadSkill(ctx, skillID, buildVariableMap(item, h.itemDir(item.Kind, item.Name)), true)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(operation) == "improver" {
		prompt = prompt + "\n\nOperation hint: improver (focus on improving an existing scenario).\n"
	}
	return prompt, nil
}

// buildVariableMap creates the template variable map for prompt-manager skill rendering.
func buildVariableMap(item BacklogItem, itemFolder string) map[string]string {
	return map[string]string{
		"ITEM_NAME":        item.Name,
		"ITEM_TITLE":       item.Title,
		"ITEM_DESCRIPTION": item.Description,
		"ITEM_KIND":        string(item.Kind),
		"ITEM_STATUS":      string(item.Status),
		"ITEM_PRIORITY":    fmt.Sprintf("%d", item.Priority),
		"ITEM_TAGS":        strings.Join(item.Tags, ", "),
		"ITEM_FOLDER":      itemFolder,
		"RESEARCH_TARGET":  item.ResearchTarget,
	}
}

func buildProcessingTitle(item BacklogItem, operation string) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch item.Kind {
	case KindFix:
		return "Apply fix: " + label
	case KindExecute:
		return "Execute task: " + label
	default:
		if strings.TrimSpace(operation) == "improver" {
			return "Improve scenario: " + label
		}
		return "Generate scenario: " + label
	}
}

func getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".md", ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".js", ".jsx", ".ts", ".tsx":
		return "text/javascript"
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
