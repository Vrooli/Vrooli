package main

import (
	"testing"
	"time"
)

// [REQ:TM-SS-005] Test unvisited files are prioritized first
func TestFilePrioritizer_UnvisitedFirst(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{
			FilePath:     "visited.go",
			VisitCount:   2,
			LastVisited:  now.Add(-48 * time.Hour), // 2 days ago
			LastModified: now.Add(-72 * time.Hour), // 3 days ago
			FirstSeen:    now.Add(-100 * time.Hour),
		},
		{
			FilePath:     "unvisited.go",
			VisitCount:   0,                        // Never visited
			LastModified: now.Add(-24 * time.Hour), // 1 day ago
			FirstSeen:    now.Add(-24 * time.Hour),
		},
		{
			FilePath:     "recently-visited.go",
			VisitCount:   1,
			LastVisited:  now.Add(-12 * time.Hour), // 0.5 days ago
			LastModified: now.Add(-48 * time.Hour), // 2 days ago
			FirstSeen:    now.Add(-50 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	if len(priorities) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(priorities))
	}

	// First file should be unvisited (gets +1000 bonus)
	if !priorities[0].IsUnvisited {
		t.Errorf("Expected first file to be unvisited, got %s (score: %.2f, unvisited: %v)",
			priorities[0].Path, priorities[0].Score, priorities[0].IsUnvisited)
	}

	// Verify it's the unvisited.go file
	if priorities[0].Path != "unvisited.go" {
		t.Errorf("Expected unvisited.go first, got %s", priorities[0].Path)
	}
}

// [REQ:TM-SS-006] Test least-visited files are prioritized among visited files
func TestFilePrioritizer_LeastVisited(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{
			FilePath:     "visited-twice.go",
			VisitCount:   2,
			LastVisited:  now.Add(-48 * time.Hour),
			LastModified: now.Add(-48 * time.Hour),
			FirstSeen:    now.Add(-100 * time.Hour),
		},
		{
			FilePath:     "visited-once.go",
			VisitCount:   1,
			LastVisited:  now.Add(-48 * time.Hour), // Same staleness
			LastModified: now.Add(-48 * time.Hour),
			FirstSeen:    now.Add(-100 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	// visited-once should rank higher (lower visit count = higher score)
	if priorities[0].Path != "visited-once.go" {
		t.Errorf("Expected visited-once.go first (least visited), got %s", priorities[0].Path)
	}

	// Verify score calculation
	// visited-once: (2 * 2) + 2 - (1 * 0.5) = 4 + 2 - 0.5 = 5.5
	// visited-twice: (2 * 2) + 2 - (2 * 0.5) = 4 + 2 - 1.0 = 5.0
	if priorities[0].Score <= priorities[1].Score {
		t.Errorf("Expected visited-once.go to have higher score: %.2f vs %.2f",
			priorities[0].Score, priorities[1].Score)
	}
}

// [REQ:TM-SS-008] Test max visits filter
func TestFilePrioritizer_MaxVisitsFilter(t *testing.T) {
	fp := NewFilePrioritizer(3) // Max 3 visits
	now := time.Now()

	files := []TrackedFile{
		{
			FilePath:     "at-limit.go",
			VisitCount:   3, // At max visits
			LastVisited:  now.Add(-24 * time.Hour),
			LastModified: now.Add(-24 * time.Hour),
			FirstSeen:    now.Add(-100 * time.Hour),
		},
		{
			FilePath:     "below-limit.go",
			VisitCount:   2, // Below max visits
			LastVisited:  now.Add(-24 * time.Hour),
			LastModified: now.Add(-24 * time.Hour),
			FirstSeen:    now.Add(-100 * time.Hour),
		},
		{
			FilePath:     "over-limit.go",
			VisitCount:   5, // Exceeded max visits
			LastVisited:  now.Add(-24 * time.Hour),
			LastModified: now.Add(-24 * time.Hour),
			FirstSeen:    now.Add(-100 * time.Hour),
		},
	}

	// Test WITH max visits respect
	priorities := fp.PrioritizeFiles(files, true)
	if len(priorities) != 1 {
		t.Errorf("Expected 1 file (below-limit), got %d", len(priorities))
	}
	if len(priorities) > 0 && priorities[0].Path != "below-limit.go" {
		t.Errorf("Expected below-limit.go, got %s", priorities[0].Path)
	}

	// Test WITHOUT max visits respect (force rescan)
	priorities = fp.PrioritizeFiles(files, false)
	if len(priorities) != 3 {
		t.Errorf("Expected 3 files when not respecting max visits, got %d", len(priorities))
	}
}

// [REQ:TM-SS-005] [REQ:TM-SS-006] Test staleness scoring algorithm
func TestFilePrioritizer_StalenessAlgorithm(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	// Test file: visited 2 days ago, modified 3 days ago, visit count 1
	files := []TrackedFile{
		{
			FilePath:     "test.go",
			VisitCount:   1,
			LastVisited:  now.Add(-48 * time.Hour), // 2 days
			LastModified: now.Add(-72 * time.Hour), // 3 days
			FirstSeen:    now.Add(-100 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	// Expected score: (2 * 2) + 3 - (1 * 0.5) = 4 + 3 - 0.5 = 6.5
	expectedScore := (2.0 * 2.0) + 3.0 - (1.0 * 0.5)
	if priorities[0].Score < expectedScore-0.1 || priorities[0].Score > expectedScore+0.1 {
		t.Errorf("Expected score ~%.2f, got %.2f (days_since_visit: %.2f, days_since_mod: %.2f)",
			expectedScore, priorities[0].Score, priorities[0].DaysSinceVisit, priorities[0].DaysSinceMod)
	}
}

// [REQ:TM-SS-008] Test FilterByMaxVisits
func TestFilePrioritizer_FilterByMaxVisits(t *testing.T) {
	fp := NewFilePrioritizer(2) // Max 2 visits

	files := []TrackedFile{
		{FilePath: "file1.go", VisitCount: 0, Deleted: false},
		{FilePath: "file2.go", VisitCount: 1, Deleted: false},
		{FilePath: "file3.go", VisitCount: 2, Deleted: false}, // At limit
		{FilePath: "file4.go", VisitCount: 3, Deleted: false}, // Over limit
		{FilePath: "file5.go", VisitCount: 0, Deleted: true},  // Deleted
	}

	filtered := fp.FilterByMaxVisits(files)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 files (visit count < 2, not deleted), got %d", len(filtered))
	}

	for _, f := range filtered {
		if f.VisitCount >= 2 {
			t.Errorf("File %s has visit count %d (>= max 2)", f.FilePath, f.VisitCount)
		}
		if f.Deleted {
			t.Errorf("File %s is deleted", f.FilePath)
		}
	}
}

// [REQ:TM-SS-005] Test GetTopNFiles
func TestFilePrioritizer_GetTopNFiles(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{FilePath: "file1.go", VisitCount: 0, FirstSeen: now.Add(-24 * time.Hour)}, // Unvisited
		{FilePath: "file2.go", VisitCount: 1, LastVisited: now.Add(-48 * time.Hour), LastModified: now.Add(-48 * time.Hour)},
		{FilePath: "file3.go", VisitCount: 2, LastVisited: now.Add(-24 * time.Hour), LastModified: now.Add(-24 * time.Hour)},
		{FilePath: "file4.go", VisitCount: 0, FirstSeen: now.Add(-12 * time.Hour)}, // Unvisited
	}

	topFiles := fp.GetTopNFiles(files, 2, false)

	if len(topFiles) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(topFiles))
	}

	// Both should be unvisited (highest priority)
	for _, path := range topFiles {
		if path != "file1.go" && path != "file4.go" {
			t.Errorf("Expected unvisited files in top 2, got %s", path)
		}
	}
}

// [REQ:TM-SS-005] [REQ:TM-SS-006] Test edge case: empty file list
func TestFilePrioritizer_EmptyFileList(t *testing.T) {
	fp := NewFilePrioritizer(3)

	files := []TrackedFile{}
	priorities := fp.PrioritizeFiles(files, false)

	if len(priorities) != 0 {
		t.Errorf("Expected empty result for empty input, got %d files", len(priorities))
	}

	topFiles := fp.GetTopNFiles(files, 5, false)
	if len(topFiles) != 0 {
		t.Errorf("Expected empty result from GetTopNFiles for empty input, got %d files", len(topFiles))
	}
}

// [REQ:TM-SS-006] Test edge case: all files deleted
func TestFilePrioritizer_AllFilesDeleted(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{FilePath: "deleted1.go", VisitCount: 0, FirstSeen: now.Add(-24 * time.Hour), Deleted: true},
		{FilePath: "deleted2.go", VisitCount: 1, LastVisited: now.Add(-48 * time.Hour), Deleted: true},
	}

	priorities := fp.PrioritizeFiles(files, false)
	if len(priorities) != 0 {
		t.Errorf("Expected no files when all are deleted, got %d", len(priorities))
	}

	filtered := fp.FilterByMaxVisits(files)
	if len(filtered) != 0 {
		t.Errorf("Expected no files when all are deleted, got %d", len(filtered))
	}
}

// [REQ:TM-SS-008] Test edge case: request more files than available
func TestFilePrioritizer_RequestMoreThanAvailable(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{FilePath: "file1.go", VisitCount: 0, FirstSeen: now.Add(-24 * time.Hour)},
		{FilePath: "file2.go", VisitCount: 1, LastVisited: now.Add(-48 * time.Hour), LastModified: now.Add(-48 * time.Hour)},
	}

	// Request 10 files but only 2 available
	topFiles := fp.GetTopNFiles(files, 10, false)
	if len(topFiles) != 2 {
		t.Errorf("Expected 2 files (all available), got %d", len(topFiles))
	}
}

// [REQ:TM-SS-005] Test edge case: zero timestamps
func TestFilePrioritizer_ZeroTimestamps(t *testing.T) {
	fp := NewFilePrioritizer(3)

	files := []TrackedFile{
		{
			FilePath:     "zero-times.go",
			VisitCount:   1,
			LastVisited:  time.Time{}, // Zero time
			LastModified: time.Time{}, // Zero time
			FirstSeen:    time.Time{},
		},
	}

	priorities := fp.PrioritizeFiles(files, false)
	if len(priorities) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(priorities))
	}

	// Should handle zero times gracefully (score should be based only on visit count)
	// Expected: 0 + 0 - (1 * 0.5) = -0.5
	expectedScore := -0.5
	if priorities[0].Score < expectedScore-0.1 || priorities[0].Score > expectedScore+0.1 {
		t.Errorf("Expected score ~%.2f for zero timestamps, got %.2f", expectedScore, priorities[0].Score)
	}
}

// [REQ:TM-SS-008] Test edge case: maxVisitsPerFile zero or negative
func TestFilePrioritizer_InvalidMaxVisits(t *testing.T) {
	// Zero should default to 3
	fp1 := NewFilePrioritizer(0)
	if fp1.maxVisitsPerFile != 3 {
		t.Errorf("Expected maxVisitsPerFile=3 for zero input, got %d", fp1.maxVisitsPerFile)
	}

	// Negative should default to 3
	fp2 := NewFilePrioritizer(-5)
	if fp2.maxVisitsPerFile != 3 {
		t.Errorf("Expected maxVisitsPerFile=3 for negative input, got %d", fp2.maxVisitsPerFile)
	}
}

// [REQ:TM-SS-005] [REQ:TM-SS-006] Test multiple unvisited files are ordered by staleness
func TestFilePrioritizer_MultipleUnvisitedOrdering(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{
			FilePath:     "old-unvisited.go",
			VisitCount:   0,
			FirstSeen:    now.Add(-100 * time.Hour), // Older
			LastModified: now.Add(-100 * time.Hour),
		},
		{
			FilePath:     "new-unvisited.go",
			VisitCount:   0,
			FirstSeen:    now.Add(-10 * time.Hour), // Newer
			LastModified: now.Add(-10 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	// Both unvisited, but older file should have higher score due to staleness
	if priorities[0].Path != "old-unvisited.go" {
		t.Errorf("Expected old-unvisited.go first (more stale), got %s", priorities[0].Path)
	}

	// Verify both have unvisited bonus
	if !priorities[0].IsUnvisited || !priorities[1].IsUnvisited {
		t.Error("Expected both files to be marked as unvisited")
	}
}

// [REQ:TM-SS-006] Test staleness with same visit count but different visit times
func TestFilePrioritizer_StalenessTiebreaker(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	files := []TrackedFile{
		{
			FilePath:     "stale.go",
			VisitCount:   1,
			LastVisited:  now.Add(-100 * time.Hour), // Not visited in long time
			LastModified: now.Add(-50 * time.Hour),
			FirstSeen:    now.Add(-200 * time.Hour),
		},
		{
			FilePath:     "fresh.go",
			VisitCount:   1,
			LastVisited:  now.Add(-10 * time.Hour), // Recently visited
			LastModified: now.Add(-50 * time.Hour), // Same modification time
			FirstSeen:    now.Add(-200 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	// Stale file should rank higher (longer since last visit)
	if priorities[0].Path != "stale.go" {
		t.Errorf("Expected stale.go first (longer since visit), got %s", priorities[0].Path)
	}

	// Verify scores reflect staleness difference
	if priorities[0].Score <= priorities[1].Score {
		t.Errorf("Expected stale.go to have higher score (%.2f) than fresh.go (%.2f)",
			priorities[0].Score, priorities[1].Score)
	}
}

// [REQ:TM-SS-008] Test FilterByMaxVisits with mixed deleted and non-deleted files
func TestFilePrioritizer_FilterMixedDeletedFiles(t *testing.T) {
	fp := NewFilePrioritizer(2)

	files := []TrackedFile{
		{FilePath: "valid1.go", VisitCount: 0, Deleted: false},
		{FilePath: "deleted.go", VisitCount: 0, Deleted: true},
		{FilePath: "valid2.go", VisitCount: 1, Deleted: false},
		{FilePath: "overvisited.go", VisitCount: 3, Deleted: false},
	}

	filtered := fp.FilterByMaxVisits(files)

	// Should return only valid1.go and valid2.go (not deleted, visit count < 2)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 valid files, got %d", len(filtered))
	}

	validPaths := map[string]bool{"valid1.go": true, "valid2.go": true}
	for _, f := range filtered {
		if !validPaths[f.FilePath] {
			t.Errorf("Unexpected file in filtered results: %s", f.FilePath)
		}
	}
}

// [REQ:TM-SS-005] [REQ:TM-SS-006] Test score calculation precision
func TestFilePrioritizer_ScorePrecision(t *testing.T) {
	fp := NewFilePrioritizer(3)
	now := time.Now()

	// Create file with specific staleness values for precise score verification
	files := []TrackedFile{
		{
			FilePath:     "precise.go",
			VisitCount:   3,
			LastVisited:  now.Add(-36 * time.Hour),  // 1.5 days
			LastModified: now.Add(-120 * time.Hour), // 5 days
			FirstSeen:    now.Add(-200 * time.Hour),
		},
	}

	priorities := fp.PrioritizeFiles(files, false)

	// Expected: (1.5 * 2) + 5 - (3 * 0.5) = 3 + 5 - 1.5 = 6.5
	expectedScore := 6.5
	tolerance := 0.01 // Very tight tolerance for precision test

	if priorities[0].Score < expectedScore-tolerance || priorities[0].Score > expectedScore+tolerance {
		t.Errorf("Score calculation imprecise: expected %.2f, got %.2f (days_since_visit: %.2f, days_since_mod: %.2f, visit_count: %d)",
			expectedScore, priorities[0].Score, priorities[0].DaysSinceVisit, priorities[0].DaysSinceMod, priorities[0].VisitCount)
	}
}
