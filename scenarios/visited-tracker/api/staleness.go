package main

import (
	"math"
	"time"
)

// calculateStalenessScore computes a staleness score for a tracked file
// based on visit history and modification time
func calculateStalenessScore(file *TrackedFile) float64 {
	if file.LastVisited == nil {
		// Never visited files get high staleness based on age
		daysSinceModified := time.Since(file.LastModified).Hours() / 24.0
		return math.Min(100.0, daysSinceModified*2.0)
	}

	daysSinceVisit := time.Since(*file.LastVisited).Hours() / 24.0
	daysSinceModified := math.Abs(file.LastVisited.Sub(file.LastModified).Hours() / 24.0)

	// Estimate modifications (simplified: assume 1 mod per week if modified after visit)
	modificationsEstimate := 0.0
	if file.LastModified.After(*file.LastVisited) {
		modificationsEstimate = math.Max(1.0, math.Floor(daysSinceModified/7.0))
	}

	// Calculate staleness: (modifications × days_since_visit) / (visit_count + 1)
	staleness := (modificationsEstimate * daysSinceVisit) / float64(file.VisitCount+1)

	return math.Min(100.0, staleness)
}

// updateStalenessScores recalculates staleness scores for all files in a campaign
func updateStalenessScores(campaign *Campaign) {
	for i := range campaign.TrackedFiles {
		campaign.TrackedFiles[i].StalenessScore = calculateStalenessScore(&campaign.TrackedFiles[i])
	}
}
