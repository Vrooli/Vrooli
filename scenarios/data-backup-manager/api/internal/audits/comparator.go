package audits

import (
	"fmt"
	"sort"
	"time"
)

// compareInventories produces the generic live-vs-snapshot verdict. It compares
// only generic signals — counts, bytes, the path-list hash, the tree content
// hash (when both sides computed it), and per-SQLite-file integrity/schema —
// and lists each specific mismatch. live_newer_than_snapshot is set when the
// live tree's newest modification time is after the snapshot timestamp, which
// explains a mismatch as expected drift rather than corruption.
//
// snapshot is the restored artifact (the proof of recoverability); live is the
// freshly captured live artifact. snapshotTime is the snapshot's own start time
// (zero if unknown).
func compareInventories(live, snapshot InventorySummary, snapshotTime time.Time) AuditComparison {
	var mismatches []string

	addCount := func(name string, liveN, snapN int64) {
		if liveN != snapN {
			mismatches = append(mismatches, fmt.Sprintf("%s: live=%d snapshot=%d", name, liveN, snapN))
		}
	}
	addCount("file count", live.Files, snapshot.Files)
	addCount("directory count", live.Directories, snapshot.Directories)
	addCount("symlink count", live.Symlinks, snapshot.Symlinks)
	addCount("other-entry count", live.Other, snapshot.Other)
	addCount("regular bytes", live.RegularBytes, snapshot.RegularBytes)

	if live.PathListSHA256 != snapshot.PathListSHA256 {
		mismatches = append(mismatches, "path-list hash differs")
	}
	// Content hash is only meaningful when both sides computed it.
	if live.TreeContentSHA != "" && snapshot.TreeContentSHA != "" && live.TreeContentSHA != snapshot.TreeContentSHA {
		mismatches = append(mismatches, "tree content hash differs")
	}

	mismatches = append(mismatches, compareSQLite(live.SQLite, snapshot.SQLite)...)

	sort.Strings(mismatches)
	return AuditComparison{
		Matches:               len(mismatches) == 0,
		Mismatches:            mismatches,
		LiveNewerThanSnapshot: !snapshotTime.IsZero() && live.MaxModTime.After(snapshotTime),
	}
}

// compareSQLite flags per-file SQLite differences: a failed integrity check on
// either side, a file present on one side only, or a divergent schema hash. The
// comparison is keyed by relative path and stays generic — schema *hashes* are
// compared, never table semantics.
func compareSQLite(live, snapshot []SqliteInventory) []string {
	var out []string
	liveByPath := indexSQLite(live)
	snapByPath := indexSQLite(snapshot)

	for path, s := range snapByPath {
		if s.IntegrityStatus == "failed" {
			out = append(out, fmt.Sprintf("sqlite %s: snapshot integrity check failed", path))
		}
		l, ok := liveByPath[path]
		if !ok {
			out = append(out, fmt.Sprintf("sqlite %s: present in snapshot, absent in live", path))
			continue
		}
		if l.SchemaSHA256 != s.SchemaSHA256 {
			out = append(out, fmt.Sprintf("sqlite %s: schema hash differs", path))
		}
	}
	for path, l := range liveByPath {
		if l.IntegrityStatus == "failed" {
			out = append(out, fmt.Sprintf("sqlite %s: live integrity check failed", path))
		}
		if _, ok := snapByPath[path]; !ok {
			out = append(out, fmt.Sprintf("sqlite %s: present in live, absent in snapshot", path))
		}
	}
	return out
}

func indexSQLite(items []SqliteInventory) map[string]SqliteInventory {
	m := make(map[string]SqliteInventory, len(items))
	for _, it := range items {
		m[it.Path] = it
	}
	return m
}
