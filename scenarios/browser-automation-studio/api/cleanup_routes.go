package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/retention"
)

type ownerCleanupEstimate = retention.CleanupEstimate
type ownerCleanupItem = retention.CleanupItem
type ownerCleanupPreview = retention.CleanupPreview
type ownerCleanupApplyResponse = retention.CleanupApplyResponse

type ownerCleanupApply struct {
	ProviderID     string              `json:"provider_id"`
	Preview        ownerCleanupPreview `json:"preview"`
	IdempotencyKey string              `json:"idempotency_key"`
	ApprovalMode   string              `json:"approval_mode"`
}

type ownerCleanupRoutes struct {
	planner *retention.Service
}

func registerOwnerCleanupRoutes(r chi.Router, planner *retention.Service) {
	s := &ownerCleanupRoutes{planner: planner}
	r.Get("/api/v1/cleanup/estimate", s.estimate)
	r.Post("/api/v1/cleanup/preview", s.preview)
	r.Post("/api/v1/cleanup/apply", s.apply)
}

func (s *ownerCleanupRoutes) sweep(r *http.Request, ids []uuid.UUID, captureIDs, recordingIDs map[string]struct{}, estimatedBytes map[uuid.UUID]int64, recoveryOnly bool) (*retention.Report, []retention.CleanupItem, int64, int, int64, error) {
	if s.planner == nil {
		return nil, nil, 0, 0, 0, errors.New("retention planner is not configured")
	}
	seconds := retention.ParseCleanupAge(r.URL.Query().Get("min_age_seconds"))
	keep, _ := strconv.Atoi(r.URL.Query().Get("keep_count"))
	maxBytes := retention.ParseCleanupMaxBytes(r.URL.Query().Get("max_bytes"))
	return s.planner.CleanupSweep(r.Context(), false, ids, captureIDs, recordingIDs, estimatedBytes, seconds, keep, maxBytes, recoveryOnly)
}

func (s *ownerCleanupRoutes) estimate(w http.ResponseWriter, r *http.Request) {
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	report, captures, maxBytes, keep, seconds, err := s.sweep(r, nil, nil, nil, nil, recoveryOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	bytes, count := report.EstimatedBytes, report.RemovedCount
	for _, item := range captures {
		bytes += item.Bytes
		count++
	}
	writeOwnerCleanupJSON(w, ownerCleanupEstimate{ProviderID: "browser-automation-studio", EstimatedBytes: bytes, ItemCount: count, MinAgeSeconds: seconds, KeepCount: keep, MaxBytes: maxBytes, ObservedAt: time.Now().UTC()})
}

func (s *ownerCleanupRoutes) preview(w http.ResponseWriter, r *http.Request) {
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
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	report, captures, maxBytes, keep, seconds, err := s.sweep(r2, nil, nil, nil, nil, recoveryOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	items := make([]ownerCleanupItem, 0, len(report.Removed)+len(captures))
	for _, item := range report.Removed {
		items = append(items, ownerCleanupItem{ID: item.ExecutionID.String(), Path: item.ArtifactDir, Bytes: item.EstimatedBytes, AgeSeconds: retentionItemAgeSeconds(item)})
	}
	for _, item := range captures {
		items = append(items, item)
	}
	writeOwnerCleanupJSON(w, ownerCleanupPreview{ProviderID: "browser-automation-studio", Items: items, MinAgeSeconds: seconds, KeepCount: keep, MaxBytes: maxBytes})
}

func retentionItemAgeSeconds(item retention.Item) int64 {
	observed := item.StartedAt
	if item.CompletedAt != nil && !item.CompletedAt.IsZero() {
		observed = *item.CompletedAt
	}
	if observed.IsZero() {
		return 0
	}
	age := int64(time.Since(observed).Seconds())
	if age < 0 {
		return 0
	}
	return age
}

func (s *ownerCleanupRoutes) apply(w http.ResponseWriter, r *http.Request) {
	var req ownerCleanupApply
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IdempotencyKey == "" {
		http.Error(w, "idempotency_key and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	delegatedRecovery := r.Header.Get("X-Vrooli-Recovery-Lock") == "held-by-storage-manager"
	if !cleanupApprovalAllowed(req.ApprovalMode, delegatedRecovery) {
		http.Error(w, "owner or operator approval is required", http.StatusForbidden)
		return
	}
	if s.planner == nil {
		http.Error(w, "retention planner is not configured", http.StatusServiceUnavailable)
		return
	}
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	result, err := s.planner.ApplyCleanup(r.Context(), req.Preview, req.IdempotencyKey, recoveryOnly, delegatedRecovery)
	if err != nil {
		status := http.StatusServiceUnavailable
		http.Error(w, err.Error(), cleanupApplyStatus(err, status))
		return
	}
	writeOwnerCleanupJSON(w, result)
}

func cleanupApprovalAllowed(mode string, delegatedRecovery bool) bool {
	return mode == "owner" || mode == "operator" || delegatedRecovery && mode == "none"
}

func cleanupApplyStatus(err error, fallback int) int {
	if errors.Is(err, retention.ErrInvalidCleanupPreview) {
		return http.StatusBadRequest
	}
	if errors.Is(err, retention.ErrRecoveryLockHeld) {
		return http.StatusConflict
	}
	return fallback
}

func writeOwnerCleanupJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
