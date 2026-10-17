// Package cleanup exposes image-tools' owner-side retention contract. The
// controller may ask for estimates and previews, but image-tools remains the
// only component allowed to delete its managed result blobs.
package cleanup

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	internaljobs "image-tools/internal/jobs"
	"image-tools/internal/module"
	"image-tools/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/schedule"
)

const (
	ProviderID = "image-tools-job-blobs"
	batchCap   = 200
	defaultAge = 7 * 24 * time.Hour
)

type jobSource interface {
	ListRetentionCandidates(context.Context, time.Time, int) ([]internaljobs.Job, error)
}

type recentJobSource interface {
	List(context.Context, int) ([]internaljobs.Job, error)
}

type Deps struct {
	Jobs  jobSource
	Store *storage.Store
	Now   func() time.Time
}

type ownerCleanup struct {
	jobs  jobSource
	store *storage.Store
	now   func() time.Time

	mu      sync.Mutex
	done    map[string]ownerApplyResponse
	applyMu sync.Mutex
}

type ownerEstimateResponse struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	BlockedReason  string    `json:"blocked_reason,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
	MinAgeSeconds  int64     `json:"min_age_seconds,omitempty"`
	KeepCount      int       `json:"keep_count,omitempty"`
	MaxBytes       int64     `json:"max_bytes,omitempty"`
}

type ownerItem struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	AgeSeconds int64  `json:"age_seconds"`
	Protected  bool   `json:"protected"`
}

type ownerPreviewResponse struct {
	ProviderID    string      `json:"provider_id"`
	Items         []ownerItem `json:"items"`
	BlockedReason string      `json:"blocked_reason,omitempty"`
	Warnings      []string    `json:"warnings,omitempty"`
	MinAgeSeconds int64       `json:"min_age_seconds,omitempty"`
	KeepCount     int         `json:"keep_count,omitempty"`
	MaxBytes      int64       `json:"max_bytes,omitempty"`
}

type ownerApplyRequest struct {
	ProviderID     string               `json:"provider_id"`
	Preview        ownerPreviewResponse `json:"preview"`
	IdempotencyKey string               `json:"idempotency_key"`
	ApprovalMode   string               `json:"approval_mode"`
}

type ownerApplyResponse struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
	AlreadyDone    bool     `json:"already_done,omitempty"`
}

// Module mounts the shared storage-manager owner contract.
func Module(cfg Deps) module.Module {
	if cfg.Now == nil {
		cfg.Now = schedule.System().Now
	}
	h := &ownerCleanup{jobs: cfg.Jobs, store: cfg.Store, now: cfg.Now, done: make(map[string]ownerApplyResponse)}
	return module.Module{
		Name: "cleanup",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/cleanup/estimate", h.estimate).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/cleanup/preview", h.preview).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/cleanup/apply", h.apply).Methods(http.MethodPost)
		},
		Endpoints: Endpoints,
	}
}

func (h *ownerCleanup) candidates(ctx context.Context, age time.Duration, keep int, maxBytes int64) ([]ownerItem, error) {
	if age <= 0 {
		age = defaultAge
	}
	if keep < 0 {
		keep = 0
	}
	if maxBytes < 0 {
		maxBytes = 0
	}
	jobs, err := h.jobs.ListRetentionCandidates(ctx, h.now().Add(-age), batchCap)
	if err != nil {
		return nil, err
	}
	referenced := make(map[string]struct{})
	if source, ok := h.jobs.(recentJobSource); ok {
		all, listErr := source.List(ctx, 1000)
		if listErr != nil {
			return nil, listErr
		}
		for _, other := range all {
			if other.State.Terminal() {
				continue
			}
			for _, candidate := range jobs {
				if other.ID != candidate.ID && bytes.Contains(other.Payload, []byte(candidate.ResultRef)) {
					referenced[candidate.ResultRef] = struct{}{}
				}
			}
		}
	}
	// The query is oldest-first. KeepCount protects the newest eligible items;
	// because the query is bounded, this remains deterministic and cheap.
	if keep > 0 && keep < len(jobs) {
		jobs = jobs[:len(jobs)-keep]
	} else if keep >= len(jobs) {
		jobs = nil
	}
	items := make([]ownerItem, 0, len(jobs))
	var total int64
	for _, job := range jobs {
		if jobPinned(job) {
			continue
		}
		if _, ok := referenced[job.ResultRef]; ok {
			continue
		}
		path, pathErr := h.store.PathForKey(job.ResultRef)
		if pathErr != nil || !strings.HasPrefix(job.ResultRef, "out/") {
			continue
		}
		info, statErr := os.Stat(path)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		bytes := info.Size()
		if maxBytes > 0 && total+bytes > maxBytes {
			break
		}
		total += bytes
		items = append(items, ownerItem{ID: job.ID, Path: job.ResultRef, Bytes: bytes, AgeSeconds: nonNegativeAge(h.now().Sub(finishedAtOrCreated(job)))})
	}
	return items, nil
}

func (h *ownerCleanup) estimate(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) {
		http.Error(w, "cleanup is controller-owned", http.StatusForbidden)
		return
	}
	age, keep, maxBytes := parsePolicy(r.URL.Query().Get("min_age_seconds"), r.URL.Query().Get("keep_count"), r.URL.Query().Get("max_bytes"))
	items, err := h.candidates(r.Context(), age, keep, maxBytes)
	if err != nil {
		http.Error(w, "unable to estimate cleanup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ownerEstimateResponse{ProviderID: ProviderID, EstimatedBytes: itemsBytes(items), ItemCount: len(items), ObservedAt: h.now(), MinAgeSeconds: int64(age / time.Second), KeepCount: keep, MaxBytes: maxBytes})
}

func (h *ownerCleanup) preview(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) {
		http.Error(w, "cleanup is controller-owned", http.StatusForbidden)
		return
	}
	var req struct {
		Estimate ownerEstimateResponse `json:"estimate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid cleanup preview", http.StatusBadRequest)
		return
	}
	age := time.Duration(req.Estimate.MinAgeSeconds) * time.Second
	items, err := h.candidates(r.Context(), age, req.Estimate.KeepCount, req.Estimate.MaxBytes)
	if err != nil {
		http.Error(w, "unable to preview cleanup: "+err.Error(), http.StatusInternalServerError)
		return
	}
	warnings := []string{}
	if req.Estimate.ProviderID != "" && req.Estimate.ProviderID != ProviderID {
		warnings = append(warnings, "estimate provider_id did not match image-tools owner")
	}
	writeJSON(w, ownerPreviewResponse{ProviderID: ProviderID, Items: items, Warnings: warnings, MinAgeSeconds: int64(age / time.Second), KeepCount: req.Estimate.KeepCount, MaxBytes: req.Estimate.MaxBytes})
}

func (h *ownerCleanup) apply(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) || r.Header.Get("X-Vrooli-Recovery-Lock") != "held-by-storage-manager" {
		http.Error(w, "storage-manager recovery authorization is required", http.StatusForbidden)
		return
	}
	var req ownerApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.IdempotencyKey) == "" || req.ProviderID != ProviderID {
		http.Error(w, "provider_id, idempotency_key, and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	if req.ApprovalMode != "owner" && req.ApprovalMode != "operator" {
		http.Error(w, "owner or operator approval is required", http.StatusForbidden)
		return
	}
	h.applyMu.Lock()
	defer h.applyMu.Unlock()
	h.mu.Lock()
	if prior, ok := h.done[req.IdempotencyKey]; ok {
		prior.AlreadyDone = true
		h.mu.Unlock()
		writeJSON(w, prior)
		return
	}
	h.mu.Unlock()
	// Rebuild the current eligibility index and require every requested item to
	// still match the reviewed key and byte count. This prevents stale previews
	// from deleting a newly-created output or a replaced blob.
	age := time.Duration(req.Preview.MinAgeSeconds) * time.Second
	current, err := h.candidates(r.Context(), age, req.Preview.KeepCount, req.Preview.MaxBytes)
	if err != nil {
		http.Error(w, "unable to validate cleanup preview: "+err.Error(), http.StatusInternalServerError)
		return
	}
	byID := make(map[string]ownerItem, len(current))
	for _, item := range current {
		byID[item.ID] = item
	}
	result := ownerApplyResponse{RemovedItemIDs: []string{}, SkippedItemIDs: []string{}}
	for _, reviewed := range req.Preview.Items {
		item, ok := byID[reviewed.ID]
		if !ok || reviewed.Path != item.Path || reviewed.Bytes != item.Bytes || reviewed.Protected {
			result.SkippedItemIDs = append(result.SkippedItemIDs, reviewed.ID)
			result.Warnings = append(result.Warnings, "stale, changed, or protected output: "+reviewed.ID)
			continue
		}
		if err := h.store.Delete(r.Context(), item.Path); err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, reviewed.ID)
			result.Warnings = append(result.Warnings, "delete failed for "+reviewed.ID+": "+err.Error())
			continue
		}
		result.ReclaimedBytes += item.Bytes
		result.RemovedItemIDs = append(result.RemovedItemIDs, reviewed.ID)
	}
	h.mu.Lock()
	h.done[req.IdempotencyKey] = result
	h.mu.Unlock()
	writeJSON(w, result)
}

func parsePolicy(ageRaw, keepRaw, maxRaw string) (time.Duration, int, int64) {
	seconds, err := strconv.ParseInt(strings.TrimSpace(ageRaw), 10, 64)
	if err != nil || seconds <= 0 {
		seconds = int64(defaultAge / time.Second)
	}
	keep, _ := strconv.Atoi(strings.TrimSpace(keepRaw))
	maxBytes, _ := strconv.ParseInt(strings.TrimSpace(maxRaw), 10, 64)
	return time.Duration(seconds) * time.Second, keep, maxBytes
}

func recoveryRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Vrooli-Recovery-Only")), "true")
}

func jobPinned(job internaljobs.Job) bool {
	return strings.EqualFold(strings.TrimSpace(job.Meta["retention_pinned"]), "true") || strings.EqualFold(strings.TrimSpace(job.Meta["pinned"]), "true")
}

func finishedAtOrCreated(j internaljobs.Job) time.Time {
	if j.FinishedAt != nil {
		return *j.FinishedAt
	}
	return j.CreatedAt
}

func nonNegativeAge(d time.Duration) int64 {
	if d < 0 {
		return 0
	}
	return int64(d / time.Second)
}

func itemsBytes(items []ownerItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "cleanup_estimate", Path: "/api/v1/cleanup/estimate", Method: "GET", Summary: "Estimate image-tools output cleanup", Description: "Returns a bounded estimate of terminal managed job-output blobs older than the requested age.", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Controller-owned cleanup probe; binary deletion remains behind the owner contract."}},
	{ID: "cleanup_preview", Path: "/api/v1/cleanup/preview", Method: "POST", Summary: "Preview image-tools output cleanup", Description: "Recomputes the bounded owner cleanup candidate set before approval.", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Controller-owned cleanup preview."}},
	{ID: "cleanup_apply", Path: "/api/v1/cleanup/apply", Method: "POST", Summary: "Apply image-tools output cleanup", Description: "Deletes only reviewed terminal job-output blobs through image-tools' storage owner.", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Controller-owned cleanup apply with idempotency and shared recovery locking."}},
}
