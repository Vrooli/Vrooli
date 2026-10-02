package retention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	coreRetention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/browser-automation-studio/database"
)

const ownerCleanupBatchCap = 200
const recoveryProtectionCacheTTL = 20 * time.Minute
const orphanRecordingEstimateCap = 20

// ParseCleanupAge preserves the conservative default for omitted or invalid age values.
func ParseCleanupAge(raw string) int64 {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds <= 0 {
		_, seconds, _, _ = AutomaticRetentionPolicy()
	}
	return seconds
}

// ParseCleanupMaxBytes accepts the established human-readable and raw-byte forms.
func ParseCleanupMaxBytes(raw string) int64 {
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

func (s *Service) CleanupSweep(ctx context.Context, apply bool, ids []uuid.UUID, captureIDs, recordingIDs map[string]struct{}, estimatedBytes map[uuid.UUID]int64, seconds int64, keep int, maxBytes int64, recoveryOnly bool) (*Report, []CleanupItem, int64, int, int64, error) {
	maxItems := 0
	if recoveryOnly {
		// Recovery runs already enforce a 2 GiB outer cap, so allow a 200-item batch.
		maxItems = 200
	} else if len(ids) == 0 {
		// Bound broad estimates and sweeps so a large artifact root stays tractable.
		maxItems = 200
	}
	var report *Report
	// A non-nil empty ID list is an explicit empty apply; nil is reserved for broad sweeps.
	if apply && ids != nil && len(ids) == 0 {
		report = &Report{}
	} else {
		var err error
		report, err = s.Sweep(ctx, Options{MaxAgeSeconds: seconds, KeepLatest: keep, MaxBytes: maxBytes, MaxItems: maxItems, Apply: apply, ExecutionIDs: ids, EstimatedBytes: estimatedBytes})
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
func (s *Service) protectedRecordingIDs(ctx context.Context, wanted map[string]struct{}, cache bool) (map[string]struct{}, error) {
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
			entry, getErr := s.store.GetExecution(ctx, id)
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
	entries, _, err := s.store.ListExecutions(ctx, database.ExecutionQuery{})
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

func selectUnprotectedCleanupItems(all []CleanupItem, keepCount int, maxBytes int64) []CleanupItem {
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.After(all[j].ModifiedAt) })
	for i := 0; i < len(all) && i < keepCount; i++ {
		all[i].Protected = true
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.Before(all[j].ModifiedAt) })
	selected := make([]CleanupItem, 0, len(all))
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
	return selected
}

func (s *Service) captureCandidates(ctx context.Context, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}) ([]CleanupItem, error) {
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
	// Size only the oldest batch so large capture roots remain bounded across ticks.
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].info.ModTime().Before(candidates[j].info.ModTime()) })
	if wanted == nil && len(candidates) > ownerCleanupBatchCap {
		candidates = candidates[:ownerCleanupBatchCap]
	}
	all := make([]CleanupItem, 0, len(candidates))
	for _, candidate := range candidates {
		bytes, sizeErr := directorySizeForCleanup(candidate.path)
		if sizeErr != nil {
			return nil, fmt.Errorf("size capture %s: %w", candidate.entry.Name(), sizeErr)
		}
		protected := s.captureProtected(ctx, candidate.entry.Name(), candidate.path)
		all = append(all, CleanupItem{ID: candidate.id, Path: candidate.path, Bytes: bytes, AgeSeconds: ageSeconds(now, candidate.info.ModTime()), Protected: protected, ModifiedAt: candidate.info.ModTime()})
	}
	if wanted != nil {
		// Keep requested protected captures visible with their newest-first markers.
		sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.After(all[j].ModifiedAt) })
		for i := 0; i < len(all) && i < keepCount; i++ {
			all[i].Protected = true
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].ModifiedAt.Before(all[j].ModifiedAt) })
		return all, nil
	}
	return selectUnprotectedCleanupItems(all, keepCount, maxBytes), nil
}

func orphanRecordingWantedCandidates(root string, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64) ([]CleanupItem, error) {
	now := time.Now()
	cutoff := now.Add(-time.Duration(minAgeSeconds) * time.Second)
	all := make([]CleanupItem, 0, len(wanted))
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
		all = append(all, CleanupItem{ID: raw, Path: path, Bytes: bytes, AgeSeconds: ageSeconds(now, info.ModTime()), ModifiedAt: info.ModTime()})
	}
	return selectUnprotectedCleanupItems(all, keepCount, maxBytes), nil
}

func (s *Service) captureProtected(ctx context.Context, name, path string) bool {
	eligible, err := s.captureEligible(ctx, name, path)
	return err != nil || !eligible
}

func (s *Service) orphanRecordingCandidates(ctx context.Context, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64, cache bool) ([]CleanupItem, error) {
	if wanted == nil && cache {
		key := fmt.Sprintf("%d/%d/%d", minAgeSeconds, keepCount, maxBytes)
		s.recoveryCacheMu.Lock()
		if s.orphanCache != nil && s.orphanCacheKey == key && time.Since(s.orphanCacheAt) < 30*time.Second {
			items := append([]CleanupItem(nil), s.orphanCache...)
			s.recoveryCacheMu.Unlock()
			return items, nil
		}
		s.recoveryCacheMu.Unlock()
		items, err := orphanRecordingCandidates(ctx, s.recordingsRoot, protected, minAgeSeconds, keepCount, maxBytes, nil, estimatedBytes)
		if err != nil {
			return nil, err
		}
		s.recoveryCacheMu.Lock()
		s.orphanCache = append([]CleanupItem(nil), items...)
		s.orphanCacheKey = key
		s.orphanCacheAt = time.Now()
		s.recoveryCacheMu.Unlock()
		return items, nil
	}
	return orphanRecordingCandidates(ctx, s.recordingsRoot, protected, minAgeSeconds, keepCount, maxBytes, wanted, estimatedBytes)
}

func orphanRecordingCandidates(ctx context.Context, root string, protected map[string]struct{}, minAgeSeconds int64, keepCount int, maxBytes int64, wanted map[string]struct{}, estimatedBytes map[uuid.UUID]int64) ([]CleanupItem, error) {
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
	candidates := make([]CleanupItem, 0)
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
		path := filepath.Join(root, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			return nil, fmt.Errorf("stat recording %s: %w", entry.Name(), statErr)
		}
		if minAgeSeconds > 0 && info.ModTime().After(cutoff) {
			continue
		}
		candidates = append(candidates, CleanupItem{ID: itemID, Path: path, AgeSeconds: ageSeconds(time.Now(), info.ModTime()), ModifiedAt: info.ModTime()})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].ModifiedAt.Before(candidates[j].ModifiedAt) })
	if wanted == nil && len(candidates) > orphanRecordingEstimateCap {
		candidates = candidates[:orphanRecordingEstimateCap]
	}
	all := make([]CleanupItem, 0, len(candidates))
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
	return selectUnprotectedCleanupItems(all, keepCount, maxBytes), nil
}
