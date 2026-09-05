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
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/retention"
	platform "github.com/vrooli/platform-go"
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

const ownerCleanupBatchCap = 200

// A recovery run is bounded by storage-manager's recovery window. Keep the
// protected execution snapshot for that window so each owner batch does not
// rebuild the full execution index while the controller is advancing through
// the same recording root. Exact-ID apply requests still re-check protection
// against the database before deleting.
const recoveryProtectionCacheTTL = 20 * time.Minute

// Orphan recording estimates must remain cheap because storage-manager asks
// for a fresh preview before every recovery batch. The apply request is
// already capped by storage-manager and may carry the requested IDs directly.
const orphanRecordingEstimateCap = 20

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
	repo             database.Repository
	root             string
	capturesRoot     string
	recoveryLockPath string
	log              *logrus.Logger
	mu               sync.Mutex
	done             map[string]ownerCleanupApplyResponse
	recoveryCacheMu  sync.Mutex
	protectedCacheAt time.Time
	protectedCache   map[string]struct{}
	orphanCacheAt    time.Time
	orphanCacheKey   string
	orphanCache      []captureCleanupItem
}

func registerOwnerCleanupRoutes(r chi.Router, repo database.Repository, root, capturesRoot string, log *logrus.Logger) *ownerCleanupService {
	s := &ownerCleanupService{repo: repo, root: root, capturesRoot: capturesRoot, recoveryLockPath: sharedRecoveryLockPath(), log: log, done: make(map[string]ownerCleanupApplyResponse)}
	r.Get("/api/v1/cleanup/estimate", s.estimate)
	r.Post("/api/v1/cleanup/preview", s.preview)
	r.Post("/api/v1/cleanup/apply", s.apply)
	return s
}

func sharedRecoveryLockPath() string {
	if resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto}); err == nil {
		if paths, resolveErr := resolver.Resolve(storage.Options{}); resolveErr == nil && paths.StateDir != "" {
			return filepath.Join(paths.StateDir, "recovery.lock")
		}
	}
	base := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if base == "" {
		base, _ = os.UserHomeDir()
		base = filepath.Join(base, ".vrooli")
	}
	return filepath.Join(base, "state", "storage-manager", "recovery.lock")
}

func (s *ownerCleanupService) acquireRecoveryLock() (func(), error) {
	path := filepath.Clean(strings.TrimSpace(s.recoveryLockPath))
	if path == "." || path == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create shared recovery lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open shared recovery lock: %w", err)
	}
	release, err := platform.LockFile(file, true)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, platform.ErrLockUnavailable) {
			return nil, fmt.Errorf("storage recovery lock is held")
		}
		return nil, fmt.Errorf("acquire shared recovery lock: %w", err)
	}
	return func() {
		release()
		_ = file.Close()
	}, nil
}

func (s *ownerCleanupService) sweep(r *http.Request, apply bool, ids []uuid.UUID, captureIDs, recordingIDs map[string]struct{}, estimatedBytes map[uuid.UUID]int64, recoveryOnly bool) (*retention.Report, []captureCleanupItem, int64, int, int64, error) {
	seconds := parseCleanupAge(r.URL.Query().Get("min_age_seconds"))
	keep, _ := strconv.Atoi(r.URL.Query().Get("keep_count"))
	maxBytes := parseCleanupMaxBytes(r.URL.Query().Get("max_bytes"))
	return s.sweepWithOptions(r.Context(), apply, ids, captureIDs, recordingIDs, estimatedBytes, seconds, keep, maxBytes, recoveryOnly)
}

func parseCleanupAge(raw string) int64 {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds <= 0 {
		_, seconds, _, _ = automaticRetentionPolicy()
	}
	return seconds
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

func (s *ownerCleanupService) sweepWithOptions(ctx context.Context, apply bool, ids []uuid.UUID, captureIDs, recordingIDs map[string]struct{}, estimatedBytes map[uuid.UUID]int64, seconds int64, keep int, maxBytes int64, recoveryOnly bool) (*retention.Report, []captureCleanupItem, int64, int, int64, error) {
	service := retention.NewService(s.repo, retention.OSFileSystem{}, s.root, s.log)
	maxItems := 0
	if recoveryOnly {
		// Recovery batches are deliberately larger than ordinary retention
		// probes. Storage-manager owns the outer 2 GiB byte cap; limiting the
		// owner to 20 directories made a large, already-budgeted evidence root
		// appear almost empty and forced many slow recovery iterations.
		maxItems = 200
	} else if len(ids) == 0 {
		// Both estimates and scheduled sweeps must be bounded. An owner with
		// thousands of artifact directories must not turn a health/estimate
		// request into an unbounded recursive filesystem scan.
		maxItems = 200
	}
	var report *retention.Report
	// An operator apply is constrained to the IDs in its signed preview. An
	// empty, non-nil ID slice therefore means “no execution items”, not “scan
	// the database for eligible items”. Automatic retention passes nil and is
	// the only broad sweep path.
	if apply && ids != nil && len(ids) == 0 {
		report = &retention.Report{}
	} else {
		var err error
		report, err = service.Sweep(ctx, retention.Options{MaxAgeSeconds: seconds, KeepLatest: keep, MaxBytes: maxBytes, MaxItems: maxItems, Apply: apply, ExecutionIDs: ids, EstimatedBytes: estimatedBytes})
		if err != nil {
			return nil, nil, maxBytes, keep, seconds, err
		}
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
	recordingMaxBytes := maxBytes
	if recordingMaxBytes > 0 {
		for _, item := range captures {
			recordingMaxBytes -= item.Bytes
			if recordingMaxBytes <= 0 {
				recordingMaxBytes = 0
				break
			}
		}
	}
	protected, err := s.protectedRecordingIDs(ctx, recordingIDs, recoveryOnly)
	if err != nil {
		return nil, nil, maxBytes, keep, seconds, err
	}
	recordings, err := s.orphanRecordingCandidates(ctx, protected, seconds, keep, recordingMaxBytes, recordingIDs, estimatedBytes, recoveryOnly)
	if err != nil {
		return nil, nil, maxBytes, keep, seconds, err
	}
	captures = append(captures, recordings...)
	return report, captures, maxBytes, keep, seconds, nil
}

// protectedRecordingIDs returns execution directories that are still
// referenced by BAS. Operator apply requests validate only their requested
// recording IDs, avoiding a full index read for every recovery batch. A
// preview/estimate validates the complete index before selecting orphans.
func (s *ownerCleanupService) protectedRecordingIDs(ctx context.Context, wanted map[string]struct{}, cache bool) (map[string]struct{}, error) {
	if wanted == nil && cache {
		s.recoveryCacheMu.Lock()
		if s.protectedCache != nil && time.Since(s.protectedCacheAt) < recoveryProtectionCacheTTL {
			protected := s.protectedCache
			s.recoveryCacheMu.Unlock()
			return protected, nil
		}
		s.recoveryCacheMu.Unlock()
	}
	protected := make(map[string]struct{})
	if wanted != nil {
		for raw := range wanted {
			if !strings.HasPrefix(raw, "recording:") {
				continue
			}
			id, err := uuid.Parse(strings.TrimPrefix(raw, "recording:"))
			if err != nil {
				continue
			}
			entry, getErr := s.repo.GetExecution(ctx, id)
			if getErr != nil {
				if errors.Is(getErr, database.ErrNotFound) {
					continue
				}
				return nil, fmt.Errorf("check recording protection %s: %w", id, getErr)
			}
			if entry != nil {
				protected[id.String()] = struct{}{}
			}
		}
		return protected, nil
	}
	entries, err := s.repo.ListExecutions(ctx, nil, nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("list executions for orphan recording protection: %w", err)
	}
	protected = make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry != nil && entry.ID != uuid.Nil {
			protected[entry.ID.String()] = struct{}{}
		}
	}
	if cache {
		s.recoveryCacheMu.Lock()
		s.protectedCache = protected
		s.protectedCacheAt = time.Now()
		s.recoveryCacheMu.Unlock()
	}
	return protected, nil
}

func (s *ownerCleanupService) orphanRecordingCandidates(ctx context.Context, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64, cache bool) ([]captureCleanupItem, error) {
	if wanted == nil && cache {
		key := fmt.Sprintf("%d/%d/%d", minAgeSeconds, keepCount, maxBytes)
		s.recoveryCacheMu.Lock()
		if s.orphanCache != nil && s.orphanCacheKey == key && time.Since(s.orphanCacheAt) < 30*time.Second {
			items := append([]captureCleanupItem(nil), s.orphanCache...)
			s.recoveryCacheMu.Unlock()
			return items, nil
		}
		s.recoveryCacheMu.Unlock()
		items, err := orphanRecordingCandidates(ctx, s.root, protected, minAgeSeconds, keepCount, maxBytes, nil, estimatedBytes)
		if err != nil {
			return nil, err
		}
		s.recoveryCacheMu.Lock()
		s.orphanCache = append([]captureCleanupItem(nil), items...)
		s.orphanCacheKey = key
		s.orphanCacheAt = time.Now()
		s.recoveryCacheMu.Unlock()
		return items, nil
	}
	return orphanRecordingCandidates(ctx, s.root, protected, minAgeSeconds, keepCount, maxBytes, wanted, estimatedBytes)
}

func (s *ownerCleanupService) invalidateOrphanRecoveryCache() {
	s.recoveryCacheMu.Lock()
	s.orphanCache = nil
	s.orphanCacheKey = ""
	s.orphanCacheAt = time.Time{}
	s.recoveryCacheMu.Unlock()
}

// StartAutomaticRetention makes the owner apply its own conservative evidence
// policy. Storage-manager still exposes preview/apply for operator-directed
// reclamation, but routine expired run evidence no longer waits for a human or
// a pressure signal. The interval is deliberately long and the first pass is
// delayed until one full interval after startup.
func (s *ownerCleanupService) StartAutomaticRetention(ctx context.Context) {
	enabled, _, keep, _ := automaticRetentionPolicy()
	if !enabled {
		return
	}
	budgets, err := evidenceBudgets()
	if err != nil {
		if s.log != nil {
			s.log.WithError(err).Error("browser evidence retention configuration invalid")
		}
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
				release, lockErr := s.acquireRecoveryLock()
				if lockErr != nil {
					cancel()
					if s.log != nil {
						s.log.WithError(lockErr).Debug("automatic owner retention deferred while storage recovery is active")
					}
					continue
				}
				if err := s.enforceEvidenceBudgets(sweepCtx, budgets, keep); err != nil && s.log != nil {
					s.log.WithError(err).Warn("automatic owner retention sweep failed")
				}
				release()
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *ownerCleanupService) cleanupRoot(id string) string {
	if strings.HasPrefix(id, "recording:") {
		return s.root
	}
	return s.capturesRoot
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
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	report, captures, maxBytes, keep, seconds, err := s.sweep(r, false, nil, nil, nil, nil, recoveryOnly)
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
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	report, captures, maxBytes, keep, seconds, err := s.sweep(r2, false, nil, nil, nil, nil, recoveryOnly)
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
	type candidate struct {
		entry os.DirEntry
		info  os.FileInfo
		path  string
		id    string
	}
	candidates := make([]candidate, 0, len(entries))
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
		candidates = append(candidates, candidate{entry: entry, info: info, path: path, id: id})
	}
	// Select the oldest bounded batch before recursively sizing directories. A
	// large capture root must not turn an estimate or scheduled tick into a
	// full tree walk; the next tick advances through the remaining entries.
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].info.ModTime().Before(candidates[j].info.ModTime()) })
	if wanted == nil && len(candidates) > ownerCleanupBatchCap {
		candidates = candidates[:ownerCleanupBatchCap]
	}
	all := make([]captureCleanupItem, 0, len(candidates))
	for _, candidate := range candidates {
		bytes, sizeErr := directorySizeForCleanup(candidate.path)
		if sizeErr != nil {
			return nil, fmt.Errorf("size capture %s: %w", candidate.entry.Name(), sizeErr)
		}
		protected := s.captureProtected(ctx, candidate.entry.Name(), candidate.path)
		all = append(all, captureCleanupItem{ID: candidate.id, Path: candidate.path, Bytes: bytes, AgeSeconds: ageSeconds(now, candidate.info.ModTime()), Protected: protected, ModifiedAt: candidate.info.ModTime()})
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

// orphanRecordingCandidates selects bounded, expired recording directories
// that have no execution-index owner. The recordings root is dedicated to
// execution artifacts, but UUID validation and the protected index set keep
// this from becoming a general-purpose directory walker.
func orphanRecordingCandidates(ctx context.Context, root string, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64) ([]captureCleanupItem, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, nil
	}
	if wanted != nil {
		return orphanRecordingWantedCandidates(root, protected, minAgeSeconds, keepCount, maxBytes, wanted, estimatedBytes)
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list recordings root: %w", err)
	}
	cutoff := time.Now().Add(-time.Duration(minAgeSeconds) * time.Second)
	candidates := make([]captureCleanupItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		id, parseErr := uuid.Parse(entry.Name())
		if parseErr != nil {
			continue
		}
		if _, exists := protected[id.String()]; exists {
			continue
		}
		itemID := "recording:" + id.String()
		if wanted != nil {
			if _, exists := wanted[itemID]; !exists {
				continue
			}
		}
		path := filepath.Join(root, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, fmt.Errorf("stat recording %s: %w", entry.Name(), statErr)
		}
		if minAgeSeconds > 0 && info.ModTime().After(cutoff) {
			continue
		}
		candidates = append(candidates, captureCleanupItem{ID: itemID, Path: path, AgeSeconds: ageSeconds(time.Now(), info.ModTime()), ModifiedAt: info.ModTime()})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].ModifiedAt.Before(candidates[j].ModifiedAt) })
	if wanted == nil && len(candidates) > orphanRecordingEstimateCap {
		candidates = candidates[:orphanRecordingEstimateCap]
	}
	all := make([]captureCleanupItem, 0, len(candidates))
	for _, candidate := range candidates {
		id, _ := uuid.Parse(strings.TrimPrefix(candidate.ID, "recording:"))
		bytes, ok := estimatedBytes[id]
		if !ok {
			var sizeErr error
			bytes, sizeErr = directorySizeForCleanup(candidate.Path)
			if sizeErr != nil {
				return nil, fmt.Errorf("size recording %s: %w", candidate.ID, sizeErr)
			}
		}
		candidate.Bytes = bytes
		all = append(all, candidate)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.After(all[j].ModifiedAt) })
	for i := 0; i < len(all) && i < keepCount; i++ {
		all[i].Protected = true
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.Before(all[j].ModifiedAt) })
	selected := make([]captureCleanupItem, 0, len(all))
	var total int64
	for _, item := range all {
		if item.Protected {
			continue
		}
		if maxBytes > 0 && total+item.Bytes > maxBytes {
			break
		}
		total += item.Bytes
		selected = append(selected, item)
	}
	return selected, nil
}

// orphanRecordingWantedCandidates validates only IDs supplied by an existing
// recovery preview. It deliberately avoids re-reading a large recordings root
// for every apply batch; containment is still enforced by removeCapture.
func orphanRecordingWantedCandidates(root string, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64) ([]captureCleanupItem, error) {
	now := time.Now()
	cutoff := now.Add(-time.Duration(minAgeSeconds) * time.Second)
	all := make([]captureCleanupItem, 0, len(wanted))
	for raw := range wanted {
		if !strings.HasPrefix(raw, "recording:") {
			continue
		}
		id, err := uuid.Parse(strings.TrimPrefix(raw, "recording:"))
		if err != nil {
			continue
		}
		if _, exists := protected[id.String()]; exists {
			continue
		}
		path := filepath.Join(root, id.String())
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return nil, fmt.Errorf("stat recording %s: %w", id, statErr)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || (minAgeSeconds > 0 && info.ModTime().After(cutoff)) {
			continue
		}
		bytes, ok := estimatedBytes[id]
		if !ok {
			bytes, err = directorySizeForCleanup(path)
			if err != nil {
				return nil, fmt.Errorf("size recording %s: %w", id, err)
			}
		}
		all = append(all, captureCleanupItem{ID: raw, Path: path, Bytes: bytes, AgeSeconds: ageSeconds(now, info.ModTime()), ModifiedAt: info.ModTime()})
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.After(all[j].ModifiedAt) })
	for i := 0; i < len(all) && i < keepCount; i++ {
		all[i].Protected = true
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.Before(all[j].ModifiedAt) })
	selected := make([]captureCleanupItem, 0, len(all))
	var total int64
	for _, item := range all {
		if item.Protected {
			continue
		}
		if maxBytes > 0 && total+item.Bytes > maxBytes {
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
	if err := coreRetention.DeleteContained(context.Background(), cleanRoot, cleanPath, nil); err != nil {
		return err
	}
	return nil
}

func (s *ownerCleanupService) apply(w http.ResponseWriter, r *http.Request) {
	var req ownerCleanupApply
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IdempotencyKey == "" {
		http.Error(w, "idempotency_key and valid cleanup preview are required", http.StatusBadRequest)
		return
	}
	delegatedRecovery := r.Header.Get("X-Vrooli-Recovery-Lock") == "held-by-storage-manager"
	if req.ApprovalMode != "owner" && req.ApprovalMode != "operator" && !(delegatedRecovery && req.ApprovalMode == "none") {
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
	release := func() {}
	if r.Header.Get("X-Vrooli-Recovery-Lock") != "held-by-storage-manager" {
		var lockErr error
		release, lockErr = s.acquireRecoveryLock()
		if lockErr != nil {
			http.Error(w, lockErr.Error(), http.StatusConflict)
			return
		}
	}
	defer release()
	ids := make([]uuid.UUID, 0, len(req.Preview.Items))
	captureIDs := make(map[string]struct{})
	recordingIDs := make(map[string]struct{})
	estimatedBytes := make(map[uuid.UUID]int64)
	for _, item := range req.Preview.Items {
		if strings.HasPrefix(item.ID, "capture:") {
			captureIDs[item.ID] = struct{}{}
			continue
		}
		if strings.HasPrefix(item.ID, "recording:") {
			recordingIDs[item.ID] = struct{}{}
			if id, parseErr := uuid.Parse(strings.TrimPrefix(item.ID, "recording:")); parseErr == nil {
				estimatedBytes[id] = item.Bytes
			}
			continue
		}
		id, err := uuid.Parse(item.ID)
		if err != nil {
			http.Error(w, "preview contains an invalid item id", http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
		estimatedBytes[id] = item.Bytes
	}
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("min_age_seconds", strconv.FormatInt(req.Preview.MinAgeSeconds, 10))
	r2.URL.RawQuery = q.Encode()
	recoveryOnly := r.Header.Get("X-Vrooli-Recovery-Only") == "true"
	report, captures, _, _, _, err := s.sweep(r2, true, ids, captureIDs, recordingIDs, estimatedBytes, recoveryOnly)
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
		if err := removeCapture(item.Path, s.cleanupRoot(item.ID)); err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, fmt.Sprintf("capture %s: %v", item.ID, err))
			continue
		}
		result.ReclaimedBytes += item.Bytes
		result.RemovedItemIDs = append(result.RemovedItemIDs, item.ID)
	}
	if recoveryOnly {
		// Advance the orphan candidate cache after this batch, but retain the
		// protected execution index. Rebuilding that full index for every batch
		// caused recovery-time CPU spikes on large BAS databases.
		s.invalidateOrphanRecoveryCache()
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
