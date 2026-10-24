package execution

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// PruneOrphanedPending drops pending records whose backlog spec.json no
// longer exists on disk. These records can never be started (loadBacklogItem
// would fail) and would otherwise consume the queue-depth budget forever.
// Safe to call at startup and on every queue attempt.
func (s *Service) PruneOrphanedPending() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return 0, err
	}
	filtered, pruned := pruneOrphanedPendingRecords(records, s.itemDir)
	if pruned == 0 {
		return 0, nil
	}
	if err := s.store.Save(filtered); err != nil {
		return 0, err
	}
	return pruned, nil
}

// pruneOrphanedPendingRecords returns the input with orphaned pending
// records removed. A pending record is orphaned when its backlog item's
// spec.json cannot be stat'd (e.g., the item was deleted, renamed, or the
// record leaked in from a test run targeting the wrong state file).
func pruneOrphanedPendingRecords(records []Record, itemDir func(kind, name string) string) ([]Record, int) {
	filtered := make([]Record, 0, len(records))
	pruned := 0
	for _, r := range records {
		if r.Status == StatusPending {
			specPath := filepath.Join(itemDir(r.BacklogKind, r.BacklogName), "spec.json")
			if _, err := os.Stat(specPath); errors.Is(err, os.ErrNotExist) {
				pruned++
				slog.Info("pruned orphaned pending execution",
					"execution_id", r.ExecutionID,
					"kind", r.BacklogKind,
					"name", r.BacklogName,
					"created_at", r.CreatedAt)
				continue
			}
		}
		filtered = append(filtered, r)
	}
	return filtered, pruned
}
