package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

// QualityMetrics represents aggregated knowledge health scores
type QualityMetrics struct {
	Coherence  float64 `json:"coherence"`
	Freshness  float64 `json:"freshness"`
	Redundancy float64 `json:"redundancy"`
	Coverage   float64 `json:"coverage"`
}

// CollectionHealth represents health metrics for a single collection
type CollectionHealth struct {
	Name    string         `json:"name"`
	Size    int            `json:"size"`
	Metrics QualityMetrics `json:"metrics"`
}

// HealthResponse represents the full health check response [REQ:KO-QM-004]
type HealthResponse struct {
	TotalEntries   int                `json:"total_entries"`
	Collections    []CollectionHealth `json:"collections"`
	OverallHealth  string             `json:"overall_health"`
	OverallMetrics QualityMetrics     `json:"overall_metrics"`
	Timestamp      time.Time          `json:"timestamp"`
}

// calculateCoherence computes semantic coherence score [REQ:KO-QM-001]
// Higher coherence = more topically consistent knowledge
func calculateCoherence(vectors [][]float64) float64 {
	if len(vectors) < 2 {
		return 1.0 // Single vector is perfectly coherent
	}

	// Calculate mean pairwise cosine similarity
	var totalSimilarity float64
	var pairCount int

	for i := 0; i < len(vectors); i++ {
		for j := i + 1; j < len(vectors) && j < i+100; j++ { // Sample up to 100 pairs per vector
			similarity := cosineSimilarity(vectors[i], vectors[j])
			totalSimilarity += similarity
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.5
	}

	avgSimilarity := totalSimilarity / float64(pairCount)

	// Normalize to 0-1 range (cosine similarity is already -1 to 1, shift to 0-1)
	coherence := (avgSimilarity + 1.0) / 2.0

	return coherence
}

// calculateFreshness computes time-based freshness score [REQ:KO-QM-002]
// Higher freshness = more recent knowledge
func calculateFreshness(timestamps []time.Time) float64 {
	if len(timestamps) == 0 {
		return 0.0
	}

	now := time.Now()
	var totalScore float64

	for _, ts := range timestamps {
		age := now.Sub(ts)

		// Exponential decay: score = e^(-age/halfLife)
		// halfLife of 30 days means score drops to 0.5 after 30 days
		halfLife := 30 * 24 * time.Hour
		decayRate := math.Log(2) / float64(halfLife)
		score := math.Exp(-float64(age) * decayRate)

		totalScore += score
	}

	avgFreshness := totalScore / float64(len(timestamps))
	return avgFreshness
}

// detectRedundancy identifies duplicate/near-duplicate entries [REQ:KO-QM-003]
// Higher redundancy = more duplicates (bad)
func detectRedundancy(vectors [][]float64, threshold float64) float64 {
	if len(vectors) < 2 {
		return 0.0 // No duplicates possible
	}

	duplicateCount := 0
	totalPairs := 0

	for i := 0; i < len(vectors); i++ {
		for j := i + 1; j < len(vectors) && j < i+100; j++ { // Sample pairs
			similarity := cosineSimilarity(vectors[i], vectors[j])
			if similarity > threshold { // Default threshold is 0.95
				duplicateCount++
			}
			totalPairs++
		}
	}

	if totalPairs == 0 {
		return 0.0
	}

	redundancy := float64(duplicateCount) / float64(totalPairs)
	return redundancy
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}

// getCollectionHealth calculates health metrics for a collection
func (s *Server) getCollectionHealth(ctx context.Context, collection string) (*CollectionHealth, error) {
	// For now, return basic metrics
	// In production, this would query Qdrant for actual vectors and metadata

	health := &CollectionHealth{
		Name: collection,
		Size: 0, // Would be populated from Qdrant
		Metrics: QualityMetrics{
			Coherence:  0.75, // Placeholder - would calculate from actual vectors
			Freshness:  0.80, // Placeholder - would calculate from timestamps
			Redundancy: 0.05, // Placeholder - would detect duplicates
			Coverage:   0.70, // Placeholder - would analyze topic coverage
		},
	}

	return health, nil
}

// handleHealthEndpoint returns knowledge system health [REQ:KO-QM-004]
func (s *Server) handleHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get list of collections
	collections, err := s.getCollections(ctx)
	if err != nil {
		s.log("failed to list collections", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to retrieve collections")
		return
	}

	var collectionHealths []CollectionHealth
	var totalEntries int
	var overallCoherence, overallFreshness, overallRedundancy, overallCoverage float64

	// Calculate health for each collection
	for _, coll := range collections {
		health, err := s.getCollectionHealth(ctx, coll)
		if err != nil {
			s.log("failed to get collection health", map[string]interface{}{
				"collection": coll,
				"error":      err.Error(),
			})
			continue
		}

		collectionHealths = append(collectionHealths, *health)
		totalEntries += health.Size
		overallCoherence += health.Metrics.Coherence
		overallFreshness += health.Metrics.Freshness
		overallRedundancy += health.Metrics.Redundancy
		overallCoverage += health.Metrics.Coverage
	}

	// Calculate average metrics
	count := float64(len(collectionHealths))
	if count == 0 {
		count = 1 // Avoid division by zero
	}

	overallMetrics := QualityMetrics{
		Coherence:  overallCoherence / count,
		Freshness:  overallFreshness / count,
		Redundancy: overallRedundancy / count,
		Coverage:   overallCoverage / count,
	}

	// Determine overall health status
	avgScore := (overallMetrics.Coherence + overallMetrics.Freshness + overallMetrics.Coverage - overallMetrics.Redundancy) / 3.0
	var healthStatus string
	switch {
	case avgScore >= 0.8:
		healthStatus = "excellent"
	case avgScore >= 0.6:
		healthStatus = "good"
	case avgScore >= 0.4:
		healthStatus = "fair"
	default:
		healthStatus = "poor"
	}

	response := HealthResponse{
		TotalEntries:   totalEntries,
		Collections:    collectionHealths,
		OverallHealth:  healthStatus,
		OverallMetrics: overallMetrics,
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log("failed to encode health response", map[string]interface{}{"error": err.Error()})
	}
}

// calculateQualityMetrics is a helper for testing [REQ:KO-QM-001,KO-QM-002,KO-QM-003]
func calculateQualityMetrics(vectors [][]float64, timestamps []time.Time) QualityMetrics {
	coherence := calculateCoherence(vectors)
	freshness := calculateFreshness(timestamps)
	redundancy := detectRedundancy(vectors, 0.95)

	// Coverage would require domain analysis - placeholder for now
	coverage := 0.70

	return QualityMetrics{
		Coherence:  coherence,
		Freshness:  freshness,
		Redundancy: redundancy,
		Coverage:   coverage,
	}
}

// formatHealthStatus converts numeric score to status string
func formatHealthStatus(score float64) string {
	switch {
	case score >= 0.8:
		return "excellent"
	case score >= 0.6:
		return "good"
	case score >= 0.4:
		return "fair"
	default:
		return "poor"
	}
}
