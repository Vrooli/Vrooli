// Package cleanup exposes the music-tools owner-side retention contract.
// Storage-manager may estimate, preview, and apply cleanup, but only this
// scenario deletes its managed generated audio.
package cleanup

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"music-tools/internal/module"
	"music-tools/internal/storage"

	"github.com/gorilla/mux"
)

const ProviderID = "music-tools-job-blobs"

type Deps struct {
	Store *storage.Store
	Now   func() time.Time
}

type owner struct {
	store *storage.Store
	now   func() time.Time
	mu    sync.Mutex
	done  map[string]ApplyResponse
}

type Item struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	AgeSeconds int64  `json:"age_seconds"`
	Protected  bool   `json:"protected"`
}

type EstimateResponse struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	ObservedAt     time.Time `json:"observed_at"`
	MinAgeSeconds  int64     `json:"min_age_seconds"`
}

type PreviewResponse struct {
	ProviderID    string   `json:"provider_id"`
	Items         []Item   `json:"items"`
	BlockedReason string   `json:"blocked_reason,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	MinAgeSeconds int64    `json:"min_age_seconds"`
}

type ApplyRequest struct {
	ProviderID     string          `json:"provider_id"`
	Preview        PreviewResponse `json:"preview"`
	IdempotencyKey string          `json:"idempotency_key"`
	ApprovalMode   string          `json:"approval_mode"`
}

type ApplyResponse struct {
	ProviderID     string   `json:"provider_id"`
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
	AlreadyDone    bool     `json:"already_done,omitempty"`
}

func Module(deps Deps) module.Module {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	h := &owner{store: deps.Store, now: deps.Now, done: map[string]ApplyResponse{}}
	return module.Module{Name: "cleanup", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/cleanup/estimate", h.estimate).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/cleanup/preview", h.preview).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/cleanup/apply", h.apply).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func (h *owner) candidates(minAge time.Duration) []Item {
	if minAge <= 0 {
		minAge = 7 * 24 * time.Hour
	}
	cutoff := h.now().Add(-minAge)
	var out []Item
	for _, entry := range h.store.Entries() {
		if !entry.Regenerable || entry.Protected || entry.LastAccess.After(cutoff) {
			continue
		}
		out = append(out, Item{ID: entry.Key, Path: entry.Key, Bytes: int64(entry.Size), AgeSeconds: int64(h.now().Sub(entry.LastAccess) / time.Second), Protected: entry.Protected})
	}
	return out
}

func (h *owner) estimate(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) {
		http.Error(w, "cleanup is controller-owned", http.StatusForbidden)
		return
	}
	items := h.candidates(parseAge(r.URL.Query().Get("min_age_seconds")))
	writeJSON(w, EstimateResponse{ProviderID: ProviderID, EstimatedBytes: itemsBytes(items), ItemCount: len(items), ObservedAt: h.now(), MinAgeSeconds: int64(parseAge(r.URL.Query().Get("min_age_seconds")) / time.Second)})
}

func (h *owner) preview(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) {
		http.Error(w, "cleanup is controller-owned", http.StatusForbidden)
		return
	}
	var req struct {
		Estimate EstimateResponse `json:"estimate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid cleanup preview", http.StatusBadRequest)
		return
	}
	items := h.candidates(time.Duration(req.Estimate.MinAgeSeconds) * time.Second)
	warnings := []string{}
	if req.Estimate.ProviderID != "" && req.Estimate.ProviderID != ProviderID {
		warnings = append(warnings, "estimate provider_id did not match music-tools owner")
	}
	writeJSON(w, PreviewResponse{ProviderID: ProviderID, Items: items, Warnings: warnings, MinAgeSeconds: req.Estimate.MinAgeSeconds})
}

func (h *owner) apply(w http.ResponseWriter, r *http.Request) {
	if !recoveryRequest(r) || r.Header.Get("X-Vrooli-Recovery-Lock") != "held-by-storage-manager" {
		http.Error(w, "storage-manager recovery authorization is required", http.StatusForbidden)
		return
	}
	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProviderID != ProviderID || strings.TrimSpace(req.IdempotencyKey) == "" {
		http.Error(w, "provider_id, idempotency_key, and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	if req.ApprovalMode != "owner" && req.ApprovalMode != "operator" {
		http.Error(w, "owner or operator approval is required", http.StatusForbidden)
		return
	}
	h.mu.Lock()
	if prior, ok := h.done[req.IdempotencyKey]; ok {
		prior.AlreadyDone = true
		h.mu.Unlock()
		writeJSON(w, prior)
		return
	}
	h.mu.Unlock()
	current := make(map[string]Item)
	for _, item := range h.candidates(time.Duration(req.Preview.MinAgeSeconds) * time.Second) {
		current[item.ID] = item
	}
	result := ApplyResponse{ProviderID: ProviderID, RemovedItemIDs: []string{}, SkippedItemIDs: []string{}}
	for _, reviewed := range req.Preview.Items {
		item, ok := current[reviewed.ID]
		if !ok || item.Path != reviewed.Path || item.Bytes != reviewed.Bytes || reviewed.Protected {
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

func parseAge(raw string) time.Duration {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds <= 0 {
		seconds = int64((7 * 24 * time.Hour) / time.Second)
	}
	return time.Duration(seconds) * time.Second
}

func recoveryRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Vrooli-Recovery-Only")), "true")
}
func itemsBytes(items []Item) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "cleanup_estimate", Path: "/api/v1/cleanup/estimate", Method: http.MethodGet, Summary: "Estimate music-tools output cleanup", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "cleanup_preview", Path: "/api/v1/cleanup/preview", Method: http.MethodPost, Summary: "Preview music-tools output cleanup", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
	{ID: "cleanup_apply", Path: "/api/v1/cleanup/apply", Method: http.MethodPost, Summary: "Apply music-tools output cleanup", Category: "cleanup", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe}},
}
