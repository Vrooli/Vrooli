package main

import (
	"sort"
	"time"
)

// FilePriority represents a file with its calculated priority score
type FilePriority struct {
	Path           string
	Score          float64
	VisitCount     int
	LastVisited    time.Time
	LastModified   time.Time
	DaysSinceVisit float64
	DaysSinceMod   float64
	IsUnvisited    bool
}

// FilePrioritizer implements file prioritization logic (TM-SS-005, TM-SS-006, TM-SS-008)
type FilePrioritizer struct {
	maxVisitsPerFile int
}

// NewFilePrioritizer creates a FilePrioritizer with configuration
func NewFilePrioritizer(maxVisitsPerFile int) *FilePrioritizer {
	if maxVisitsPerFile <= 0 {
		maxVisitsPerFile = 3 // Default from PRD
	}
	return &FilePrioritizer{
		maxVisitsPerFile: maxVisitsPerFile,
	}
}

// PrioritizeFiles sorts files by staleness score using the PRD algorithm:
// score = (days_since_last_visit * 2) + days_since_last_modification - (total_visit_count * 0.5)
// Higher score = higher priority for next smart scan
// Implements TM-SS-005 (unvisited first), TM-SS-006 (least visited), TM-SS-008 (max visits filter)
func (fp *FilePrioritizer) PrioritizeFiles(files []TrackedFile, respectMaxVisits bool) []FilePriority {
	now := time.Now()
	priorities := make([]FilePriority, 0, len(files))

	for _, f := range files {
		// Skip deleted files
		if f.Deleted {
			continue
		}

		// TM-SS-008: Filter out files beyond max visits (unless force_rescan)
		if respectMaxVisits && f.VisitCount >= fp.maxVisitsPerFile {
			continue
		}

		// Calculate staleness score per PRD appendix
		daysSinceVisit := 0.0
		isUnvisited := f.VisitCount == 0

		if !isUnvisited && !f.LastVisited.IsZero() {
			daysSinceVisit = now.Sub(f.LastVisited).Hours() / 24.0
		} else if isUnvisited {
			// Unvisited files: use time since first_seen instead
			daysSinceVisit = now.Sub(f.FirstSeen).Hours() / 24.0
		}

		daysSinceMod := 0.0
		if !f.LastModified.IsZero() {
			daysSinceMod = now.Sub(f.LastModified).Hours() / 24.0
		}

		// Staleness algorithm from PRD
		score := (daysSinceVisit * 2.0) + daysSinceMod - (float64(f.VisitCount) * 0.5)

		// Bonus for unvisited files (TM-SS-005: unvisited first)
		if isUnvisited {
			score += 1000.0 // Large bonus ensures unvisited files appear first
		}

		priorities = append(priorities, FilePriority{
			Path:           f.FilePath,
			Score:          score,
			VisitCount:     f.VisitCount,
			LastVisited:    f.LastVisited,
			LastModified:   f.LastModified,
			DaysSinceVisit: daysSinceVisit,
			DaysSinceMod:   daysSinceMod,
			IsUnvisited:    isUnvisited,
		})
	}

	// Sort by score descending (highest priority first)
	// TM-SS-006: Among visited files, least-visited appears first (lower visit_count = higher score)
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].Score > priorities[j].Score
	})

	return priorities
}

// GetTopNFiles returns the top N highest-priority files
func (fp *FilePrioritizer) GetTopNFiles(files []TrackedFile, n int, respectMaxVisits bool) []string {
	priorities := fp.PrioritizeFiles(files, respectMaxVisits)

	if len(priorities) < n {
		n = len(priorities)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = priorities[i].Path
	}

	return result
}

// FilterByMaxVisits filters files that haven't exceeded max visits (TM-SS-008)
func (fp *FilePrioritizer) FilterByMaxVisits(files []TrackedFile) []TrackedFile {
	filtered := make([]TrackedFile, 0, len(files))
	for _, f := range files {
		if f.VisitCount < fp.maxVisitsPerFile && !f.Deleted {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
