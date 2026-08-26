package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	coreRetention "github.com/vrooli/api-core/retention"
	"web-console/internal/sessionstore"
)

type ownerCleanupEstimate struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	MinAgeSeconds  int64     `json:"min_age_seconds,omitempty"`
	KeepCount      int       `json:"keep_count,omitempty"`
	MaxBytes       int64     `json:"max_bytes,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
}

type ownerCleanupItem struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	AgeSeconds int64  `json:"age_seconds"`
	Protected  bool   `json:"protected"`
}

type ownerCleanupPreview struct {
	ProviderID    string             `json:"provider_id"`
	Items         []ownerCleanupItem `json:"items"`
	MinAgeSeconds int64              `json:"min_age_seconds,omitempty"`
	KeepCount     int                `json:"keep_count,omitempty"`
	MaxBytes      int64              `json:"max_bytes,omitempty"`
}

type ownerCleanupApply struct {
	Preview        ownerCleanupPreview `json:"preview"`
	IdempotencyKey string              `json:"idempotency_key"`
	ApprovalMode   string              `json:"approval_mode"`
}

type ownerCleanupResult struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
}

type webConsoleCleanup struct {
	server *Server
	mu     sync.Mutex
	done   map[string]ownerCleanupResult
}

func (s *Server) registerOwnerCleanupRoutes() {
	h := &webConsoleCleanup{server: s, done: make(map[string]ownerCleanupResult)}
	s.router.HandleFunc("/api/v1/cleanup/estimate", h.estimate).Methods(http.MethodGet)
	s.router.HandleFunc("/api/v1/cleanup/preview", h.preview).Methods(http.MethodPost)
	s.router.HandleFunc("/api/v1/cleanup/apply", h.apply).Methods(http.MethodPost)
	h.startAutomaticRetention(context.Background())
}

func (h *webConsoleCleanup) candidates(r *http.Request) ([]sessionstore.Metadata, int64, int64, int) {
	seconds, _ := strconv.ParseInt(r.URL.Query().Get("min_age_seconds"), 10, 64)
	maxBytes := parseCleanupMaxBytes(r.URL.Query().Get("max_bytes"))
	keep, _ := strconv.Atoi(r.URL.Query().Get("keep_count"))
	rows := h.candidateRows(r.Context(), seconds, maxBytes, keep)
	return rows, seconds, maxBytes, keep
}

func parseCleanupMaxBytes(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if parsed, err := coreRetention.ParseBytes(raw); err == nil && parsed >= 0 {
		return parsed
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func (h *webConsoleCleanup) candidateRows(ctx context.Context, seconds, maxBytes int64, keep int) []sessionstore.Metadata {
	rows, err := h.server.sessionStore.ListRetentionCandidates(ctx)
	if err != nil {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second)
	filtered := make([]sessionstore.Metadata, 0, len(rows))
	for _, row := range rows {
		if seconds > 0 && sessionActivity(row).After(cutoff) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

// startAutomaticRetention runs the owner-local safe policy. It removes only
// archived, inactive evidence older than seven days; live sessions are still
// protected by isProtected and the session-store candidate query. The global
// storage-manager apply endpoint remains available for operator-directed
// reclamation and does not need to be used for routine expiry.
func (h *webConsoleCleanup) startAutomaticRetention(ctx context.Context) {
	enabled, seconds, keep, maxBytes := automaticRetentionPolicy()
	if !enabled {
		return
	}
	interval := 15 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("WC_OWNER_RETENTION_INTERVAL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				workCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				h.autoSweep(workCtx, seconds, keep, maxBytes)
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (h *webConsoleCleanup) autoSweep(ctx context.Context, seconds int64, keep int, maxBytes int64) {
	rows := h.candidateRows(ctx, seconds, maxBytes, keep)
	items := h.items(rows, maxBytes, keep)
	byID := make(map[string]sessionstore.Metadata, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	for _, item := range items {
		row, ok := byID[item.ID]
		if !ok || h.isProtected(row.ID) {
			continue
		}
		if _, err := pruneArchivedAgentHistory(row); err != nil {
			log.Printf("web-console automatic retention: prune %s: %v", row.ID, err)
			continue
		}
	}
}

func automaticRetentionPolicy() (bool, int64, int, int64) {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("WC_OWNER_RETENTION_ENABLED")); raw != "" {
		enabled = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
	}
	seconds := int64((30 * 24 * time.Hour) / time.Second)
	if raw := strings.TrimSpace(os.Getenv("WC_OWNER_RETENTION_MAX_AGE")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= 0 {
			seconds = int64(parsed / time.Second)
		}
	}
	keep := 0
	if raw := strings.TrimSpace(os.Getenv("WC_OWNER_RETENTION_KEEP_COUNT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			keep = parsed
		}
	}
	maxBytes := int64(0)
	if raw := strings.TrimSpace(os.Getenv("WC_OWNER_RETENTION_MAX_BYTES")); raw != "" {
		if parsed, err := coreRetention.ParseBytes(raw); err == nil && parsed >= 0 {
			maxBytes = parsed
		}
	}
	return enabled, seconds, keep, maxBytes
}

func (h *webConsoleCleanup) estimate(w http.ResponseWriter, r *http.Request) {
	rows, seconds, maxBytes, keep := h.candidates(r)
	items := h.items(rows, maxBytes, keep)
	writeCleanupJSON(w, ownerCleanupEstimate{ProviderID: "web-console-sessions", EstimatedBytes: itemsBytes(items), ItemCount: len(items), MinAgeSeconds: seconds, KeepCount: keep, MaxBytes: maxBytes, ObservedAt: time.Now().UTC()})
}

func (h *webConsoleCleanup) preview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Estimate ownerCleanupEstimate `json:"estimate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid cleanup preview", http.StatusBadRequest)
		return
	}
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("min_age_seconds", strconv.FormatInt(req.Estimate.MinAgeSeconds, 10))
	q.Set("keep_count", strconv.Itoa(req.Estimate.KeepCount))
	q.Set("max_bytes", strconv.FormatInt(req.Estimate.MaxBytes, 10))
	r2.URL.RawQuery = q.Encode()
	rows, seconds, maxBytes, keep := h.candidates(r2)
	writeCleanupJSON(w, ownerCleanupPreview{ProviderID: "web-console-sessions", Items: h.items(rows, maxBytes, keep), MinAgeSeconds: seconds, KeepCount: keep, MaxBytes: maxBytes})
}

func (h *webConsoleCleanup) apply(w http.ResponseWriter, r *http.Request) {
	var req ownerCleanupApply
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IdempotencyKey == "" {
		http.Error(w, "idempotency_key and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	if req.ApprovalMode != "owner" && req.ApprovalMode != "operator" {
		http.Error(w, "owner or operator approval is required", http.StatusForbidden)
		return
	}
	h.mu.Lock()
	if cached, ok := h.done[req.IdempotencyKey]; ok {
		h.mu.Unlock()
		writeCleanupJSON(w, cached)
		return
	}
	h.mu.Unlock()
	rows, _, _, _ := h.candidates(r)
	byID := make(map[string]sessionstore.Metadata, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	result := ownerCleanupResult{RemovedItemIDs: []string{}, SkippedItemIDs: []string{}}
	for _, item := range req.Preview.Items {
		row, ok := byID[item.ID]
		if !ok {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			continue
		}
		if h.isProtected(row.ID) || item.Protected {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, "protected live session: "+item.ID)
			continue
		}
		if h.server.conversations != nil && h.server.conversations.CountSessionEvents(r.Context(), row.ID) > 0 {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, "durable transcript protected: "+item.ID)
			continue
		}
		bytes, err := pruneArchivedAgentHistory(row)
		if err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		if err := h.server.sessionStore.Delete(r.Context(), row.ID); err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		result.ReclaimedBytes += bytes
		result.RemovedItemIDs = append(result.RemovedItemIDs, item.ID)
	}
	h.mu.Lock()
	h.done[req.IdempotencyKey] = result
	h.mu.Unlock()
	writeCleanupJSON(w, result)
}

func (h *webConsoleCleanup) items(rows []sessionstore.Metadata, maxBytes int64, keepCount int) []ownerCleanupItem {
	items := make([]ownerCleanupItem, 0, len(rows))
	var total int64
	for index, row := range rows {
		if keepCount > 0 && index >= len(rows)-keepCount {
			continue
		}
		if h.isProtected(row.ID) {
			continue
		}
		bytes, err := archivedAgentHistorySize(row)
		if err != nil || bytes == 0 {
			continue
		}
		if maxBytes > 0 && total+bytes > maxBytes {
			break
		}
		total += bytes
		path, _ := archivedAgentHistoryPath(row)
		items = append(items, ownerCleanupItem{ID: row.ID, Path: path, Bytes: bytes, AgeSeconds: sessionAgeSeconds(row)})
	}
	return items
}

func (h *webConsoleCleanup) isProtected(id string) bool {
	if h == nil || h.server == nil || h.server.sessions == nil {
		return false
	}
	_, ok := h.server.sessions.Get(id)
	return ok
}

func sessionActivity(row sessionstore.Metadata) time.Time {
	if !row.LastActivityAt.IsZero() {
		return row.LastActivityAt
	}
	if !row.ArchivedAt.IsZero() {
		return row.ArchivedAt
	}
	return row.Created
}

func sessionAgeSeconds(row sessionstore.Metadata) int64 {
	age := int64(time.Since(sessionActivity(row)).Seconds())
	if age < 0 {
		return 0
	}
	return age
}

func itemsBytes(items []ownerCleanupItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}

func writeCleanupJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
