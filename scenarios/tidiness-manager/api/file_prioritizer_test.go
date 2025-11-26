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
