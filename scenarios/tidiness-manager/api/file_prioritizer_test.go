package main

import (
	"testing"
	"time"
)

// FileInfo represents file metadata for prioritization
type FileInfo struct {
	Path         string
	LastVisit    time.Time
	LastModified time.Time
	VisitCount   int
	LineCount    int
}

// [REQ:TM-SS-005] File prioritization - unvisited first
func TestFilePrioritizer_UnvisitedFirst(t *testing.T) {
	now := time.Now()
	files := []FileInfo{
		{
			Path:         "visited.go",
			LastVisit:    now.Add(-24 * time.Hour),
			LastModified: now.Add(-48 * time.Hour),
			VisitCount:   2,
			LineCount:    100,
		},
		{
			Path:         "unvisited.go",
			LastVisit:    time.Time{}, // Zero value = never visited
			LastModified: now.Add(-24 * time.Hour),
			VisitCount:   0,
			LineCount:    150,
		},
		{
			Path:         "another_unvisited.go",
			LastVisit:    time.Time{},
			LastModified: now.Add(-12 * time.Hour),
			VisitCount:   0,
			LineCount:    200,
		},
	}

	prioritized := PrioritizeFiles(files)

	// Unvisited files should come first
	if prioritized[0].VisitCount != 0 {
		t.Errorf("First file should be unvisited, got VisitCount=%d", prioritized[0].VisitCount)
	}
	if prioritized[1].VisitCount != 0 {
		t.Errorf("Second file should be unvisited, got VisitCount=%d", prioritized[1].VisitCount)
	}
	if prioritized[2].VisitCount == 0 {
		t.Errorf("Third file should be visited, got VisitCount=%d", prioritized[2].VisitCount)
	}
}

// [REQ:TM-SS-006] File prioritization - least visited
func TestFilePrioritizer_LeastVisited(t *testing.T) {
	now := time.Now()
	files := []FileInfo{
		{
			Path:         "heavily_visited.go",
			LastVisit:    now.Add(-1 * time.Hour),
			LastModified: now.Add(-48 * time.Hour),
			VisitCount:   10,
			LineCount:    100,
		},
		{
			Path:         "lightly_visited.go",
			LastVisit:    now.Add(-24 * time.Hour),
			LastModified: now.Add(-24 * time.Hour),
			VisitCount:   2,
			LineCount:    150,
		},
		{
			Path:         "moderately_visited.go",
			LastVisit:    now.Add(-12 * time.Hour),
			LastModified: now.Add(-12 * time.Hour),
			VisitCount:   5,
			LineCount:    200,
		},
	}

	prioritized := PrioritizeFiles(files)

	// Among visited files, least visited should come first
	if prioritized[0].VisitCount >= prioritized[1].VisitCount {
		t.Errorf("Files not sorted by visit count: %d >= %d", prioritized[0].VisitCount, prioritized[1].VisitCount)
	}
	if prioritized[1].VisitCount >= prioritized[2].VisitCount {
		t.Errorf("Files not sorted by visit count: %d >= %d", prioritized[1].VisitCount, prioritized[2].VisitCount)
	}
}

// [REQ:TM-SS-005,TM-SS-006] Test staleness algorithm from PRD
func TestFilePrioritizer_StalenessAlgorithm(t *testing.T) {
	now := time.Now()

	// Test case: file never visited (unvisited)
	unvisited := FileInfo{
		Path:         "unvisited.go",
		LastVisit:    time.Time{},
		LastModified: now.Add(-10 * 24 * time.Hour), // 10 days old
		VisitCount:   0,
		LineCount:    100,
	}

	// Test case: file visited once, 30 days ago
	stale := FileInfo{
		Path:         "stale.go",
		LastVisit:    now.Add(-30 * 24 * time.Hour),
		LastModified: now.Add(-35 * 24 * time.Hour),
		VisitCount:   1,
		LineCount:    100,
	}

	// Test case: recently visited multiple times
	recent := FileInfo{
		Path:         "recent.go",
		LastVisit:    now.Add(-1 * 24 * time.Hour),
		LastModified: now.Add(-2 * 24 * time.Hour),
		VisitCount:   5,
		LineCount:    100,
	}

	unvisitedScore := calculateStalenessScore(unvisited, now)
	staleScore := calculateStalenessScore(stale, now)
	recentScore := calculateStalenessScore(recent, now)

	// Unvisited should have highest priority (highest score)
	if unvisitedScore <= staleScore {
		t.Errorf("Unvisited score (%f) should be > stale score (%f)", unvisitedScore, staleScore)
	}

	// Stale should have higher priority than recent
	if staleScore <= recentScore {
		t.Errorf("Stale score (%f) should be > recent score (%f)", staleScore, recentScore)
	}
}

// PrioritizeFiles sorts files by staleness score (highest first)
// Algorithm: score = (days_since_last_visit * 2) + (days_since_last_modification) - (total_visit_count * 0.5)
func PrioritizeFiles(files []FileInfo) []FileInfo {
	now := time.Now()
	type scoredFile struct {
		file  FileInfo
		score float64
	}

	scored := make([]scoredFile, len(files))
	for i, f := range files {
		scored[i] = scoredFile{
			file:  f,
			score: calculateStalenessScore(f, now),
		}
	}

	// Sort by score descending (highest score = highest priority)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := make([]FileInfo, len(files))
	for i, sf := range scored {
		result[i] = sf.file
	}
	return result
}

// calculateStalenessScore computes priority score for a file
func calculateStalenessScore(f FileInfo, now time.Time) float64 {
	var daysSinceLastVisit float64
	if f.LastVisit.IsZero() {
		// Never visited - assign very high days count
		daysSinceLastVisit = 10000
	} else {
		daysSinceLastVisit = now.Sub(f.LastVisit).Hours() / 24
	}

	daysSinceModified := now.Sub(f.LastModified).Hours() / 24

	// PRD formula: (days_since_last_visit * 2) + (days_since_last_modification) - (total_visit_count * 0.5)
	score := (daysSinceLastVisit * 2) + daysSinceModified - (float64(f.VisitCount) * 0.5)
	return score
}

// [REQ:TM-SS-005] Test unvisited files always win regardless of other factors
func TestFilePrioritizer_UnvisitedAlwaysFirst(t *testing.T) {
	now := time.Now()
	files := []FileInfo{
		{
			// Visited file with huge line count (trying to game the system)
			Path:         "huge_visited.go",
			LastVisit:    now.Add(-1 * time.Hour),
			LastModified: now.Add(-2 * time.Hour),
			VisitCount:   1,
			LineCount:    10000,
		},
		{
			// Tiny unvisited file
			Path:         "tiny_unvisited.go",
			LastVisit:    time.Time{},
			LastModified: now.Add(-1 * 24 * time.Hour),
			VisitCount:   0,
			LineCount:    10,
		},
	}

	prioritized := PrioritizeFiles(files)

	// Unvisited should always come first
	if prioritized[0].Path != "tiny_unvisited.go" {
		t.Errorf("Unvisited file should be first, got %s", prioritized[0].Path)
	}
}
