package main

import (
	"log"
	"os"
	"path/filepath"
)

// cleanupOrphanedWorkflowCaptures removes the legacy per-repo workflow-captures
// data trees (data/<repoID>/workflow-captures/...) left behind by the deleted
// workflow-capture stack. Plan B replaced that store with test-genie run
// pointers (no data migration — Decision 5).
//
// It runs once at startup and is idempotent: when no orphan directories remain
// it is a silent no-op, so subsequent boots skip cleanly without a migrations
// registry.
func cleanupOrphanedWorkflowCaptures(store *VisualCaptureStorage) {
	if store == nil {
		return
	}
	root, err := store.dataRoot()
	if err != nil {
		return
	}
	repoEntries, err := os.ReadDir(root)
	if err != nil {
		return // data root not created yet — nothing to clean
	}

	var removed int
	var freed int64
	for _, repo := range repoEntries {
		if !repo.IsDir() {
			continue
		}
		wcDir := filepath.Join(root, repo.Name(), "workflow-captures")
		info, statErr := os.Stat(wcDir)
		if statErr != nil || !info.IsDir() {
			continue
		}
		freed += dirSize(wcDir)
		if rmErr := os.RemoveAll(wcDir); rmErr != nil {
			log.Printf("[workflow-captures-cleanup] failed to remove %s: %v", wcDir, rmErr)
			continue
		}
		removed++
	}
	if removed > 0 {
		log.Printf("[workflow-captures-cleanup] removed %d orphaned workflow-captures dir(s), freed %d bytes", removed, freed)
	}
}
