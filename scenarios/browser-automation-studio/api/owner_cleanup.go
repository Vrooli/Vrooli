package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/retention"
)

type ownerCleanupEstimate struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	BlockedReason  string    `json:"blocked_reason,omitempty"`
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

type captureCleanupItem struct {
	ID         string
	Path       string
	Bytes      int64
	AgeSeconds int64
	Protected  bool
	ModifiedAt time.Time
}

type ownerCleanupPreview struct {
	ProviderID    string             `json:"provider_id"`
	Items         []ownerCleanupItem `json:"items"`
	BlockedReason string             `json:"blocked_reason,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
	MinAgeSeconds int64              `json:"min_age_seconds,omitempty"`
	KeepCount     int                `json:"keep_count,omitempty"`
	MaxBytes      int64              `json:"max_bytes,omitempty"`
}

type ownerCleanupApply struct {
	ProviderID     string              `json:"provider_id"`
	Preview        ownerCleanupPreview `json:"preview"`
	IdempotencyKey string              `json:"idempotency_key"`
	ApprovalMode   string              `json:"approval_mode"`
}

type ownerCleanupApplyResponse struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
}

type ownerCleanupService struct {
	repo         database.Repository
	root         string
	capturesRoot string
	log          *logrus.Logger
	mu           sync.Mutex
	done         map[string]ownerCleanupApplyResponse
}

func registerOwnerCleanupRoutes(r chi.Router, repo database.Repository, root, capturesRoot string, log *logrus.Logger) *ownerCleanupService {
	s := &ownerCleanupService{repo: repo, root: root, capturesRoot: capturesRoot, log: log, done: make(map[string]ownerCleanupApplyResponse)}
	r.Get("/api/v1/cleanup/estimate", s.estimate)
	r.Post("/api/v1/cleanup/preview", s.preview)
	r.Post("/api/v1/cleanup/apply", s.apply)
	return s
}

func (s *ownerCleanupService) sweep(r *http.Request, apply bool, ids []uuid.UUID, captureIDs map[string]struct{}) (*retention.Report, []captureCleanupItem, int64, int, int64, error) {
	seconds, _ := strconv.ParseInt(r.URL.Query().Get("min_age_seconds"), 10, 64)
	keep, _ := strconv.Atoi(r.URL.Query().Get("keep_count"))
	maxBytes := parseCleanupMaxBytes(r.URL.Query().Get("max_bytes"))
	return s.sweepWithOptions(r.Context(), apply, ids, captureIDs, seconds, keep, maxBytes)
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

func (s *ownerCleanupService) sweepWithOptions(ctx context.Context, apply bool, ids []uuid.UUID, captureIDs map[string]struct{}, seconds int64, keep int, maxBytes int64) (*retention.Report, []captureCleanupItem, int64, int, int64, error) {
	service := retention.NewService(s.repo, retention.OSFileSystem{}, s.root, s.log)
	report, err := service.Sweep(ctx, retention.Options{MaxAgeSeconds: seconds, KeepLatest: keep, MaxBytes: maxBytes, Apply: apply, ExecutionIDs: ids})
	if err != nil {
		return nil, nil, maxBytes, keep, seconds, err
	}
	captureMaxBytes := maxBytes
	if maxBytes > 0 {
		captureMaxBytes -= sumRemoved(report.Removed)
		if captureMaxBytes < 0 {
			captureMaxBytes = 0
		}
	}
	captures, err := s.captureCandidates(ctx, seconds, keep, captureMaxBytes, captureIDs)
	if err != nil {
		return nil, nil, maxBytes, keep, seconds, err
	}
	return report, captures, maxBytes, keep, seconds, nil
}

// StartAutomaticRetention makes the owner apply its own conservative evidence
// policy. Storage-manager still exposes preview/apply for operator-directed
// reclamation, but routine expired run evidence no longer waits for a human or
// a pressure signal. The interval is deliberately long and the first pass is
// delayed until one full interval after startup.
func (s *ownerCleanupService) StartAutomaticRetention(ctx context.Context) {
	enabled, seconds, keep, maxBytes := automaticRetentionPolicy()
	if !enabled {
		return
	}
	interval := 15 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_INTERVAL")); raw != "" {
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
				sweepCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
				if _, captures, _, _, _, err := s.sweepWithOptions(sweepCtx, true, nil, nil, seconds, keep, maxBytes); err != nil {
					if s.log != nil {
						s.log.WithError(err).Warn("automatic owner retention sweep failed")
					}
				} else {
					for _, item := range captures {
						if item.Protected {
							continue
						}
						if err := removeCapture(item.Path, s.capturesRoot); err != nil && s.log != nil {
							s.log.WithError(err).WithField("capture", item.ID).Warn("automatic capture retention removal failed")
						}
					}
				}
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func automaticRetentionPolicy() (bool, int64, int, int64) {
	enabled := true
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_ENABLED")); raw != "" {
		enabled = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
	}
	seconds := int64((7 * 24 * time.Hour) / time.Second)
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_MAX_AGE")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed >= 0 {
			seconds = int64(parsed / time.Second)
		}
	}
	keep := 0
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_KEEP_COUNT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			keep = parsed
		}
	}
	maxBytes := int64(0)
	if raw := strings.TrimSpace(os.Getenv("BAS_OWNER_RETENTION_MAX_BYTES")); raw != "" {
		if parsed, err := coreRetention.ParseBytes(raw); err == nil && parsed >= 0 {
			maxBytes = parsed
		}
	}
	return enabled, seconds, keep, maxBytes
}

func (s *ownerCleanupService) estimate(w http.ResponseWriter, r *http.Request) {
	report, captures, maxBytes, keep, seconds, err := s.sweep(r, false, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	bytes, count := cappedItems(report.Removed, maxBytes)
	for _, item := range captures {
		if maxBytes > 0 && bytes+item.Bytes > maxBytes {
			break
		}
		bytes += item.Bytes
		count++
	}
	writeOwnerCleanupJSON(w, ownerCleanupEstimate{ProviderID: "browser-automation-studio", EstimatedBytes: bytes, ItemCount: count, MinAgeSeconds: seconds, KeepCount: keep, MaxBytes: maxBytes, ObservedAt: time.Now().UTC()})
}

func (s *ownerCleanupService) preview(w http.ResponseWriter, r *http.Request) {
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
	report, captures, maxBytes, keep, seconds, err := s.sweep(r2, false, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	items := make([]ownerCleanupItem, 0, len(report.Removed))
	var total int64
	for _, item := range report.Removed {
		if maxBytes > 0 && total+item.EstimatedBytes > maxBytes {
			break
		}
		total += item.EstimatedBytes
		items = append(items, ownerCleanupItem{ID: item.ExecutionID.String(), Path: item.ArtifactDir, Bytes: item.EstimatedBytes, AgeSeconds: retentionItemAgeSeconds(item)})
	}
	for _, item := range captures {
		if maxBytes > 0 && total+item.Bytes > maxBytes {
			break
		}
		total += item.Bytes
		items = append(items, ownerCleanupItem{ID: item.ID, Path: item.Path, Bytes: item.Bytes, AgeSeconds: item.AgeSeconds, Protected: item.Protected})
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

func (s *ownerCleanupService) captureCandidates(ctx context.Context, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}) ([]captureCleanupItem, error) {
	root := filepath.Clean(strings.TrimSpace(s.capturesRoot))
	if root == "." || root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list captures root: %w", err)
	}
	now := time.Now()
	cutoff := now.Add(-time.Duration(minAgeSeconds) * time.Second)
	all := make([]captureCleanupItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		id := "capture:" + entry.Name()
		if wanted != nil {
			if _, ok := wanted[id]; !ok {
				continue
			}
		}
		path := filepath.Join(root, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, fmt.Errorf("stat capture %s: %w", entry.Name(), statErr)
		}
		if minAgeSeconds > 0 && info.ModTime().After(cutoff) {
			continue
		}
		bytes, sizeErr := directorySizeForCleanup(path)
		if sizeErr != nil {
			return nil, fmt.Errorf("size capture %s: %w", entry.Name(), sizeErr)
		}
		protected := s.captureProtected(ctx, entry.Name(), path)
		all = append(all, captureCleanupItem{ID: id, Path: path, Bytes: bytes, AgeSeconds: ageSeconds(now, info.ModTime()), Protected: protected, ModifiedAt: info.ModTime()})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.After(all[j].ModifiedAt) })
	for i := 0; i < len(all) && i < keepCount; i++ {
		all[i].Protected = true
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.Before(all[j].ModifiedAt) })
	selected := make([]captureCleanupItem, 0, len(all))
	var total int64
	for _, item := range all {
		if item.Protected && wanted == nil {
			continue
		}
		if maxBytes > 0 && wanted == nil && total+item.Bytes > maxBytes {
			break
		}
		total += item.Bytes
		selected = append(selected, item)
	}
	return selected, nil
}

func (s *ownerCleanupService) captureProtected(ctx context.Context, name, path string) bool {
	ids := []string{name}
	entries, err := os.ReadDir(path)
	if err == nil {
		for _, entry := range entries {
			ids = append(ids, entry.Name())
		}
	}
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil || s.repo == nil {
			continue
		}
		exec, getErr := s.repo.GetExecution(ctx, id)
		if getErr == nil && exec != nil && !database.IsTerminalStatus(exec.Status) {
			return true
		}
	}
	return false
}

func ageSeconds(now, modified time.Time) int64 {
	age := int64(now.Sub(modified).Seconds())
	if age < 0 {
		return 0
	}
	return age
}

func directorySizeForCleanup(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func removeCapture(path, root string) error {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	cleanPath := filepath.Clean(path)
	if cleanRoot == "." || cleanPath == "." {
		return errors.New("capture root is not configured")
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("capture path is outside the configured captures root")
	}
	return os.RemoveAll(cleanPath)
}

func (s *ownerCleanupService) apply(w http.ResponseWriter, r *http.Request) {
	var req ownerCleanupApply
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IdempotencyKey == "" {
		http.Error(w, "idempotency_key and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	if req.ApprovalMode != "owner" && req.ApprovalMode != "operator" {
		http.Error(w, "owner or operator approval is required", http.StatusForbidden)
		return
	}
	s.mu.Lock()
	if cached, ok := s.done[req.IdempotencyKey]; ok {
		s.mu.Unlock()
		writeOwnerCleanupJSON(w, cached)
		return
	}
	s.mu.Unlock()
	ids := make([]uuid.UUID, 0, len(req.Preview.Items))
	captureIDs := make(map[string]struct{})
	for _, item := range req.Preview.Items {
		if strings.HasPrefix(item.ID, "capture:") {
			captureIDs[item.ID] = struct{}{}
			continue
		}
		id, err := uuid.Parse(item.ID)
		if err != nil {
			http.Error(w, "preview contains an invalid item id", http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("min_age_seconds", strconv.FormatInt(req.Preview.MinAgeSeconds, 10))
	r2.URL.RawQuery = q.Encode()
	report, captures, _, _, _, err := s.sweep(r2, true, ids, captureIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	result := ownerCleanupApplyResponse{ReclaimedBytes: sumRemoved(report.Removed), RemovedItemIDs: make([]string, 0, len(report.Removed)), SkippedItemIDs: make([]string, 0, len(report.Skipped))}
	for _, item := range report.Removed {
		result.RemovedItemIDs = append(result.RemovedItemIDs, item.ExecutionID.String())
	}
	for _, item := range report.Skipped {
		result.SkippedItemIDs = append(result.SkippedItemIDs, item.ExecutionID.String())
	}
	for _, item := range captures {
		if item.Protected {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, "protected in-flight capture: "+item.ID)
			continue
		}
		if err := removeCapture(item.Path, s.capturesRoot); err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, fmt.Sprintf("capture %s: %v", item.ID, err))
			continue
		}
		result.ReclaimedBytes += item.Bytes
		result.RemovedItemIDs = append(result.RemovedItemIDs, item.ID)
	}
	s.mu.Lock()
	s.done[req.IdempotencyKey] = result
	s.mu.Unlock()
	writeOwnerCleanupJSON(w, result)
}

func cappedItems(items []retention.Item, max int64) (int64, int) {
	var total int64
	count := 0
	for _, item := range items {
		if max > 0 && total+item.EstimatedBytes > max {
			break
		}
		total += item.EstimatedBytes
		count++
	}
	return total, count
}

func sumRemoved(items []retention.Item) int64 {
	var total int64
	for _, item := range items {
		total += item.EstimatedBytes
	}
	return total
}

func writeOwnerCleanupJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
