package retention

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/executionartifactlock"
	platform "github.com/vrooli/platform-go"
)

var (
	ErrRecoveryLockHeld      = errors.New("storage recovery lock is held")
	ErrInvalidCleanupPreview = errors.New("preview contains an invalid item id")
)

type cleanupApplySelection struct {
	executionIDs   []uuid.UUID
	captureIDs     map[string]struct{}
	recordingIDs   map[string]struct{}
	estimatedBytes map[uuid.UUID]int64
}

func (s *Service) cleanupRoot(id string) string {
	if strings.HasPrefix(id, "recording:") {
		return s.recordingsRoot
	}
	return s.capturesRoot
}

func (s *Service) invalidateOrphanRecoveryCache() {
	s.recoveryCacheMu.Lock()
	s.orphanCache = nil
	s.orphanCacheKey = ""
	s.orphanCacheAt = time.Time{}
	s.recoveryCacheMu.Unlock()
}

func (s *Service) ApplyCleanup(ctx context.Context, preview CleanupPreview, idempotencyKey string, recoveryOnly, recoveryLockHeld bool) (CleanupApplyResponse, error) {
	if strings.TrimSpace(idempotencyKey) == "" {
		return CleanupApplyResponse{}, errors.New("idempotency_key is required")
	}
	s.cleanupMu.Lock()
	if cached, ok := s.cleanupDone[idempotencyKey]; ok {
		s.cleanupMu.Unlock()
		return cached, nil
	}
	s.cleanupMu.Unlock()

	release := func() {}
	if !recoveryLockHeld {
		var err error
		release, err = s.acquireRecoveryLock()
		if err != nil {
			return CleanupApplyResponse{}, err
		}
	}
	defer release()

	selection, err := parseCleanupApplySelection(preview.Items)
	if err != nil {
		return CleanupApplyResponse{}, err
	}
	report, captures, _, _, _, err := s.CleanupSweep(ctx, true, selection.executionIDs, selection.captureIDs, selection.recordingIDs, selection.estimatedBytes, preview.MinAgeSeconds, 0, 0, recoveryOnly)
	if err != nil {
		return CleanupApplyResponse{}, err
	}
	result := CleanupApplyResponse{
		ReclaimedBytes: sumRemoved(report.Removed),
		RemovedItemIDs: make([]string, 0, len(report.Removed)+len(captures)),
		SkippedItemIDs: make([]string, 0, len(report.Skipped)+len(captures)),
	}
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
		root := s.cleanupRoot(item.ID)
		deleter, ok := s.fs.(containedDeleter)
		if !ok {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, fmt.Sprintf("capture %s: contained filesystem deletion unavailable", item.ID))
			continue
		}
		remove := func(lockCtx context.Context) error {
			if strings.HasPrefix(item.ID, "recording:") {
				id, parseErr := parseRecordingCleanupID(item.ID)
				if parseErr != nil {
					return ErrInvalidCleanupPreview
				}
				if s.store == nil {
					return errors.New("execution repository unavailable")
				}
				if _, rowErr := s.store.GetExecution(lockCtx, id); !errors.Is(rowErr, database.ErrNotFound) {
					if rowErr == nil {
						return errors.New("orphan recording became indexed during cleanup")
					}
					return rowErr
				}
				info, statErr := os.Lstat(item.Path)
				if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || !info.ModTime().Equal(item.ModifiedAt) {
					return errors.New("orphan recording changed since cleanup preview")
				}
				bytes, sizeErr := directorySizeForCleanup(item.Path)
				if sizeErr != nil || bytes != item.Bytes {
					return errors.New("orphan recording inventory changed since cleanup preview")
				}
			}
			return deleter.DeleteContained(lockCtx, root, item.Path)
		}
		var err error
		if strings.HasPrefix(item.ID, "recording:") {
			executionID, parseErr := parseRecordingCleanupID(item.ID)
			if parseErr != nil {
				err = ErrInvalidCleanupPreview
			} else {
				err = executionartifactlock.WithLock(ctx, s.recordingsRoot, executionID, remove)
			}
		} else {
			err = remove(ctx)
		}
		if err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, item.ID)
			result.Warnings = append(result.Warnings, fmt.Sprintf("capture %s: %v", item.ID, err))
			continue
		}
		result.ReclaimedBytes += item.Bytes
		result.RemovedItemIDs = append(result.RemovedItemIDs, item.ID)
	}
	if recoveryOnly {
		s.invalidateOrphanRecoveryCache()
	}
	s.cleanupMu.Lock()
	if s.cleanupDone == nil {
		s.cleanupDone = make(map[string]CleanupApplyResponse)
	}
	s.cleanupDone[idempotencyKey] = result
	s.cleanupMu.Unlock()
	return result, nil
}

func (s *Service) acquireRecoveryLock() (func(), error) {
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
			return nil, fmt.Errorf("%w", ErrRecoveryLockHeld)
		}
		return nil, fmt.Errorf("acquire shared recovery lock: %w", err)
	}
	return func() {
		release()
		_ = file.Close()
	}, nil
}

func parseCleanupApplySelection(items []CleanupItem) (cleanupApplySelection, error) {
	selection := cleanupApplySelection{
		executionIDs:   make([]uuid.UUID, 0, len(items)),
		captureIDs:     make(map[string]struct{}),
		recordingIDs:   make(map[string]struct{}),
		estimatedBytes: make(map[uuid.UUID]int64),
	}
	for _, item := range items {
		if strings.HasPrefix(item.ID, "capture:") {
			selection.captureIDs[item.ID] = struct{}{}
			continue
		}
		if strings.HasPrefix(item.ID, "recording:") {
			selection.recordingIDs[item.ID] = struct{}{}
			if id, err := parseRecordingCleanupID(item.ID); err == nil {
				selection.estimatedBytes[id] = item.Bytes
			}
			continue
		}
		id, err := uuid.Parse(item.ID)
		if err != nil {
			return cleanupApplySelection{}, ErrInvalidCleanupPreview
		}
		selection.executionIDs = append(selection.executionIDs, id)
		selection.estimatedBytes[id] = item.Bytes
	}
	return selection, nil
}

func sumRemoved(items []Item) int64 {
	var total int64
	for _, item := range items {
		total += item.EstimatedBytes
	}
	return total
}
