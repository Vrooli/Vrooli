// Package captures provides HTTP handlers for quick-capture operations.
//
// Captures are raw, unclassified thoughts entered by the user. They are
// stored as folders under {rootDir}/captures/{id}/ with a capture.json file.
// An AI classification agent automatically analyzes the text and suggests
// one or more backlog items for user confirmation.
//
// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md
package captures

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
)

type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

// BacklogItemCreator abstracts backlog item creation for the create-item endpoint.
type BacklogItemCreator interface {
	ItemDir(kind string, name string) string
	SaveItem(kind, name, title, description string, tags []string) error
}

// EventDispatcher emits graph invalidation events for graph projections.
type EventDispatcher interface {
	DispatchInvalidate(lenses ...string)
}

// EventLogger records view events for analytics.
type EventLogger interface {
	EmitCaptureViewed(captureID string)
}

// Handler provides HTTP handlers for capture operations.
type Handler struct {
	rootDir         string
	agentService    AgentSpawner
	promptClient    promptmanager.Client
	backlogCreator  BacklogItemCreator
	eventDispatcher EventDispatcher
	eventLogger     EventLogger
}

// NewHandler creates a new captures handler.
func NewHandler(rootDir string, agentService AgentSpawner, promptClient promptmanager.Client) *Handler {
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

// SetBacklogCreator sets the backlog item creator for the create-item endpoint.
func (h *Handler) SetBacklogCreator(creator BacklogItemCreator) {
	h.backlogCreator = creator
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (h *Handler) SetEventLogger(l EventLogger) {
	h.eventLogger = l
}

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (h *Handler) SetEventDispatcher(d EventDispatcher) {
	h.eventDispatcher = d
}

func (h *Handler) invalidateTopologyGraph() {
	if h.eventDispatcher == nil {
		return
	}
	h.eventDispatcher.DispatchInvalidate("topology")
}

func (h *Handler) invalidateAllGraphLenses() {
	if h.eventDispatcher == nil {
		return
	}
	h.eventDispatcher.DispatchInvalidate("topology", "flow", "operations")
}

// RegisterRoutes registers capture endpoints on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/captures", h.List).Methods("GET")
	r.HandleFunc("/api/v1/captures", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/captures/{id}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/captures/{id}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/captures/{id}/classify", h.Classify).Methods("POST")
	r.HandleFunc("/api/v1/captures/{id}/create-item", h.CreateItem).Methods("POST")
}

// capture represents the on-disk capture state.
type capture struct {
	ID             string          `json:"id"`
	Text           string          `json:"text"`
	Attachments    []string        `json:"attachments"`
	Created        string          `json:"created"`
	Status         string          `json:"status"`
	Classification *classification `json:"classification,omitempty"`
}

// classification represents the AI-generated classification result.
type classification struct {
	Items        []classificationItem `json:"items"`
	ClassifiedAt string               `json:"classified_at"`
}

// classificationItem represents one suggested backlog item.
type classificationItem struct {
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Confidence  float64  `json:"confidence"`
}

func (h *Handler) capturesDir() string {
	return filepath.Join(h.rootDir, "captures")
}

func (h *Handler) captureDir(id string) string {
	return filepath.Join(h.capturesDir(), id)
}

func (h *Handler) captureSpecPath(id string) string {
	return filepath.Join(h.captureDir(id), "capture.json")
}

func (h *Handler) classificationPath(id string) string {
	return filepath.Join(h.captureDir(id), "classification.json")
}

// List returns all captures, newest first.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	capturesRoot := h.capturesDir()
	entries, err := os.ReadDir(capturesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			_ = httputil.JSON(w, map[string]any{"captures": []any{}})
			return
		}
		apierr.MapError(w, "[captures] list", apierr.Internal("failed to read captures directory"))
		return
	}

	var caps []capture
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cap, err := h.loadCapture(entry.Name())
		if err != nil {
			log.Printf("[captures] list: skipping %s: %v", entry.Name(), err)
			continue
		}
		caps = append(caps, *cap)
	}

	// Sort by created time descending (newest first).
	sort.Slice(caps, func(i, j int) bool {
		return caps[i].Created > caps[j].Created
	})

	if caps == nil {
		caps = []capture{}
	}
	_ = httputil.JSON(w, map[string]any{"captures": caps})
}

// allowedImageTypes lists Content-Types accepted for capture attachments.
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// Create creates a new capture from a multipart form (text + optional image files).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		apierr.MapError(w, "[captures] create", apierr.BadRequest("invalid multipart form"))
		return
	}

	text := strings.TrimSpace(r.FormValue("text"))
	if text == "" {
		apierr.MapError(w, "[captures] create", apierr.BadRequest("text is required"))
		return
	}

	id := fmt.Sprintf("cap-%s", idgen.Generate())
	now := time.Now().UTC().Format(time.RFC3339)

	cap := capture{
		ID:          id,
		Text:        text,
		Attachments: []string{},
		Created:     now,
		Status:      "classifying",
	}

	dir := h.captureDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to create capture directory"))
		return
	}

	// Save attached image files.
	files := r.MultipartForm.File["files"]
	for _, fh := range files {
		mediaType, _, _ := mime.ParseMediaType(fh.Header.Get("Content-Type"))
		if !allowedImageTypes[mediaType] {
			// Clean up the capture directory on rejection.
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.BadRequest("unsupported file type: %s", mediaType))
			return
		}

		attDir := filepath.Join(dir, "attachments")
		if err := os.MkdirAll(attDir, 0o755); err != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to create attachments directory"))
			return
		}

		safeName := sanitizeFilename(fh.Filename)
		destPath := filepath.Join(attDir, safeName)

		src, err := fh.Open()
		if err != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to read uploaded file"))
			return
		}

		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to save uploaded file"))
			return
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			_ = os.RemoveAll(dir)
			apierr.MapError(w, "[captures] create", apierr.Internal("failed to write uploaded file"))
			return
		}

		cap.Attachments = append(cap.Attachments, filepath.Join("attachments", safeName))
	}

	if err := h.writeCapture(&cap); err != nil {
		apierr.MapError(w, "[captures] create", apierr.Internal("failed to write capture"))
		return
	}

	// Auto-trigger classification agent.
	resp := map[string]any{"capture": cap}
	runResult, err := h.spawnClassifyAgent(r, &cap)
	if err != nil {
		// Classification failed to start, but capture was created. Mark as failed.
		log.Printf("[captures] create: classification spawn failed: %v", err)
		cap.Status = "failed"
		_ = h.writeCapture(&cap)
		resp["capture"] = cap
	} else {
		resp["task_id"] = runResult.TaskID
		resp["run_id"] = runResult.RunID
		resp["base_url"] = runResult.BaseURL
	}

	h.invalidateTopologyGraph()
	_ = httputil.JSONWithStatus(w, http.StatusCreated, resp)
}

// sanitizeFilename removes path separators and dangerous characters from a filename.
func sanitizeFilename(name string) string {
	// Use only the base name (strip any directory components).
	name = filepath.Base(name)
	// Replace characters that could cause issues.
	replacer := strings.NewReplacer(
		"..", "_",
		"/", "_",
		"\\", "_",
		"\x00", "_",
	)
	name = replacer.Replace(name)
	if name == "" || name == "." {
		name = "unnamed"
	}
	return name
}

// Get returns a single capture by ID.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] get", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] get", apierr.Internal("failed to load capture"))
		return
	}
	_ = httputil.JSON(w, map[string]any{"capture": cap})
	if h.eventLogger != nil {
		h.eventLogger.EmitCaptureViewed(id)
	}
}

// Delete removes a capture and its folder.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dir := h.captureDir(id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Idempotent: already deleted.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		apierr.MapError(w, "[captures] delete", apierr.Internal("failed to delete capture"))
		return
	}
	h.invalidateTopologyGraph()
	w.WriteHeader(http.StatusNoContent)
}

// Classify spawns a classification agent for a capture (for retry).
func (h *Handler) Classify(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] classify", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to load capture"))
		return
	}

	// Update status to classifying.
	cap.Status = "classifying"
	cap.Classification = nil
	if err := h.writeCapture(cap); err != nil {
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to update capture"))
		return
	}

	// Remove stale classification file if present.
	_ = os.Remove(h.classificationPath(id))

	runResult, err := h.spawnClassifyAgent(r, cap)
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[captures] classify", apierr.Unavailable("agent-manager is not available"))
			return
		}
		apierr.MapError(w, "[captures] classify", apierr.Internal("failed to spawn classification agent"))
		return
	}

	h.invalidateTopologyGraph()
	_ = httputil.JSON(w, map[string]any{
		"task_id":  runResult.TaskID,
		"run_id":   runResult.RunID,
		"base_url": runResult.BaseURL,
		"created":  time.Now().UTC().Format(time.RFC3339),
	})
}

// createItemRequest is the JSON body for CreateItem.
type createItemRequest struct {
	Kind string `json:"kind"`
}

// CreateItem creates a backlog item from a classified capture.
// It pre-fills the item with text from the capture and tags from the classification.
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	if h.backlogCreator == nil {
		apierr.MapError(w, "[captures] create-item", apierr.Internal("backlog creator not configured"))
		return
	}

	id := mux.Vars(r)["id"]
	cap, err := h.loadCapture(id)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[captures] create-item", apierr.NotFound("capture not found"))
			return
		}
		apierr.MapError(w, "[captures] create-item", apierr.Internal("failed to load capture"))
		return
	}

	// Parse optional kind override from request body.
	kind := "execute"
	var req createItemRequest
	if r.Body != nil {
		if decErr := json.NewDecoder(r.Body).Decode(&req); decErr == nil && req.Kind != "" {
			kind = strings.ToLower(strings.TrimSpace(req.Kind))
		}
	}

	// Build item title from capture text (truncated).
	title := truncate(cap.Text, 80)

	// Collect tags from classification items.
	var tags []string
	if cap.Classification != nil {
		for _, ci := range cap.Classification.Items {
			tags = append(tags, ci.Tags...)
		}
	}
	// Deduplicate tags.
	seen := make(map[string]bool, len(tags))
	dedupTags := make([]string, 0, len(tags))
	for _, t := range tags {
		if !seen[t] {
			seen[t] = true
			dedupTags = append(dedupTags, t)
		}
	}

	// Generate a name from the title.
	name := sanitizeCaptureItemName(title)
	if name == "" {
		name = "capture-item-" + id
	}

	if err := h.backlogCreator.SaveItem(kind, name, title, cap.Text, dedupTags); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			apierr.MapError(w, "[captures] create-item", apierr.Conflict("backlog item already exists"))
			return
		}
		log.Printf("[captures] create-item: %v", err)
		apierr.MapError(w, "[captures] create-item", apierr.Internal("failed to create backlog item"))
		return
	}

	// Mark capture as classified.
	cap.Status = "classified"
	_ = h.writeCapture(cap)

	log.Printf("[captures] create-item: created %s/%s from capture %s", kind, name, id)
	h.invalidateAllGraphLenses()
	_ = httputil.JSONWithStatus(w, http.StatusCreated, map[string]any{
		"kind": kind,
		"name": name,
	})
}

// sanitizeCaptureItemName converts a title to a folder-safe name.
func sanitizeCaptureItemName(title string) string {
	name := strings.ToLower(title)
	name = strings.ReplaceAll(name, " ", "-")
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	s := result.String()
	// Truncate to a reasonable length for folder names.
	if len(s) > 60 {
		s = s[:60]
	}
	return strings.TrimRight(s, "-")
}

// loadCapture reads a capture from disk, merging classification.json if present.
func (h *Handler) loadCapture(id string) (*capture, error) {
	data, err := os.ReadFile(h.captureSpecPath(id))
	if err != nil {
		return nil, err
	}
	var cap capture
	if err := json.Unmarshal(data, &cap); err != nil {
		return nil, fmt.Errorf("unmarshal capture %s: %w", id, err)
	}

	// Merge classification.json if it exists.
	classData, err := os.ReadFile(h.classificationPath(id))
	if err == nil {
		var cls classification
		if err := json.Unmarshal(classData, &cls); err == nil {
			cap.Classification = &cls
			if cap.Status == "classifying" {
				cap.Status = "classified"
			}
		}
	}

	// Auto-fail captures stuck in classifying for > 2 minutes.
	if cap.Status == "classifying" {
		created, parseErr := time.Parse(time.RFC3339, cap.Created)
		if parseErr == nil && time.Since(created) > 2*time.Minute {
			cap.Status = "failed"
			_ = h.writeCapture(&cap)
		}
	}

	return &cap, nil
}

// writeCapture writes a capture to disk.
func (h *Handler) writeCapture(cap *capture) error {
	data, err := json.MarshalIndent(cap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal capture: %w", err)
	}
	return os.WriteFile(h.captureSpecPath(cap.ID), data, 0o644)
}

// spawnClassifyAgent spawns the classification agent for a capture.
func (h *Handler) spawnClassifyAgent(r *http.Request, cap *capture) (*agentmanager.RunResult, error) {
	service := h.agentService
	if service == nil {
		return nil, agentmanager.ErrNotAvailable
	}
	if !service.IsEnabled() {
		return nil, agentmanager.ErrNotAvailable
	}

	entry, ok := promptcatalog.ResolveCaptureSkill()
	if !ok {
		return nil, fmt.Errorf("capture prompt catalog entry missing")
	}
	skillID := entry.SkillID
	variables := map[string]string{
		"CAPTURE_TEXT": cap.Text,
		"CAPTURE_ID":   cap.ID,
	}

	prompt, err := h.promptClient.ReadSkill(r.Context(), skillID, variables, false)
	if err != nil {
		return nil, fmt.Errorf("fetch classify prompt: %w", err)
	}

	activityCtx := agentactivity.WithSpec(r.Context(), agentactivity.Spec{
		OwnerType:   agentactivity.OwnerCapture,
		OwnerName:   cap.ID,
		OwnerTitle:  truncate(cap.Text, 120),
		Purpose:     agentactivity.PurposeClassify,
		RequestedBy: "swarm-manager",
		Metadata: map[string]string{
			"entrypoint": "captures.classify",
			"skill_id":   skillID,
		},
	})

	result, err := service.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        "capture",
		Name:        cap.ID,
		Title:       "Classify capture: " + truncate(cap.Text, 60),
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   h.captureDir(cap.ID),
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "classify",
		Environment: map[string]string{"VROOLI_SPAWN_SOURCE": "capture/" + cap.ID},
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// EnsureCapturesDir creates the captures directory if it doesn't exist.
func (h *Handler) EnsureCapturesDir() error {
	return os.MkdirAll(h.capturesDir(), 0o755)
}
