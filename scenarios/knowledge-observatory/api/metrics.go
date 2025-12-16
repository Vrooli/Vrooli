package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	maxPairSamplesPerVector       = 100
	freshnessHalfLife             = 30 * 24 * time.Hour
	redundancySimilarityThreshold = 0.95
	defaultCoverageScore          = 0.70
)

// QualityMetrics represents aggregated knowledge health scores
type QualityMetrics struct {
	Coherence  *float64 `json:"coherence,omitempty"`
	Freshness  *float64 `json:"freshness,omitempty"`
	Redundancy *float64 `json:"redundancy,omitempty"`
	Coverage   *float64 `json:"coverage,omitempty"`
}

// CollectionHealth represents health metrics for a single collection
type CollectionHealth struct {
	Name    string         `json:"name"`
	Size    *int           `json:"size,omitempty"`
	Metrics *QualityMetrics `json:"metrics,omitempty"`
}

// HealthResponse represents the full health check response [REQ:KO-QM-004]
type HealthResponse struct {
	TotalEntries   *int               `json:"total_entries,omitempty"`
	Collections    []CollectionHealth `json:"collections"`
	OverallHealth  string             `json:"overall_health"`
	OverallMetrics *QualityMetrics    `json:"overall_metrics,omitempty"`
	Timestamp      time.Time          `json:"timestamp"`
}

func forEachSampledVectorPair(vectors [][]float64, maxPairsPerVector int, fn func(a, b []float64)) {
	if maxPairsPerVector <= 0 {
		maxPairsPerVector = maxPairSamplesPerVector
	}
	for i := 0; i < len(vectors); i++ {
		maxJ := len(vectors)
		if i+maxPairsPerVector+1 < maxJ {
			maxJ = i + maxPairsPerVector + 1
		}
		for j := i + 1; j < maxJ; j++ {
			fn(vectors[i], vectors[j])
		}
	}
}

// calculateCoherence computes semantic coherence score [REQ:KO-QM-001]
// Higher coherence = more topically consistent knowledge
func calculateCoherence(vectors [][]float64) float64 {
	if len(vectors) < 2 {
		return 1.0 // Single vector is perfectly coherent
	}

	var totalSimilarity float64
	var pairCount int

	forEachSampledVectorPair(vectors, maxPairSamplesPerVector, func(a, b []float64) {
		totalSimilarity += cosineSimilarity(a, b)
		pairCount++
	})
	if pairCount == 0 {
		return 0.5
	}

	avgSimilarity := totalSimilarity / float64(pairCount)

	// Normalize to 0-1 range (cosine similarity is -1..1).
	return (avgSimilarity + 1.0) / 2.0
}

// calculateFreshness computes time-based freshness score [REQ:KO-QM-002]
// Higher freshness = more recent knowledge
func calculateFreshness(timestamps []time.Time) float64 {
	if len(timestamps) == 0 {
		return 0.0
	}

	now := time.Now()
	var totalScore float64

	// Exponential decay: score = e^(-age/halfLife)
	// halfLife of 30 days means score drops to 0.5 after 30 days.
	decayRate := math.Log(2) / float64(freshnessHalfLife)
	for _, ts := range timestamps {
		age := now.Sub(ts)
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

	forEachSampledVectorPair(vectors, maxPairSamplesPerVector, func(a, b []float64) {
		similarity := cosineSimilarity(a, b)
		if similarity > threshold {
			duplicateCount++
		}
		totalPairs++
	})
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
	// For now, return basic metrics.
	// We can fetch the real vector count from Qdrant, but quality metrics are still placeholders.

	var size *int
	if base := strings.TrimSpace(s.qdrantURL()); base != "" {
		if count, err := s.getCollectionPointCount(ctx, base, collection); err == nil {
			size = &count
		} else {
			s.log("failed to retrieve collection point count", map[string]interface{}{
				"collection": collection,
				"error":      err.Error(),
			})
		}
	}

	health := &CollectionHealth{
		Name: collection,
		Size: size,
		// Metrics are intentionally omitted unless computed from real underlying data.
		Metrics: nil,
	}

	return health, nil
}

type qdrantCountRequest struct {
	Exact bool `json:"exact"`
}

type qdrantCountResponse struct {
	Result struct {
		Count int `json:"count"`
	} `json:"result"`
}

func (s *Server) getCollectionPointCount(ctx context.Context, qdrantBase string, collection string) (int, error) {
	baseURL, err := url.Parse(strings.TrimRight(qdrantBase, "/"))
	if err != nil {
		return 0, fmt.Errorf("invalid qdrant url: %w", err)
	}
	baseURL.Path = fmt.Sprintf("%s/collections/%s/points/count", strings.TrimRight(baseURL.Path, "/"), collection)

	body, err := json.Marshal(qdrantCountRequest{Exact: true})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal count request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL.String(), bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create count request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("qdrant count request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("qdrant count returned status %d", resp.StatusCode)
	}

	var decoded qdrantCountResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}
	return decoded.Result.Count, nil
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

	collectionHealths := make([]CollectionHealth, 0, len(collections))
	var totalEntries int
	totalEntriesKnown := true

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
		if health.Size == nil {
			totalEntriesKnown = false
		} else {
			totalEntries += *health.Size
		}
	}

	overallMetrics := averageCollectionMetrics(collectionHealths)
	healthScore := knowledgeHealthScore(overallMetrics)

	var totalEntriesPtr *int
	if totalEntriesKnown {
		totalEntriesPtr = &totalEntries
	}

	response := HealthResponse{
		TotalEntries:   totalEntriesPtr,
		Collections:    collectionHealths,
		OverallHealth:  formatHealthStatus(healthScore),
		OverallMetrics: overallMetrics,
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log("failed to encode health response", map[string]interface{}{"error": err.Error()})
	}
}

func averageCollectionMetrics(collectionHealths []CollectionHealth) *QualityMetrics {
	if len(collectionHealths) == 0 {
		return nil
	}

	var (
		sumCoherence, sumFreshness, sumRedundancy, sumCoverage float64
		count                                                 int
	)

	for _, health := range collectionHealths {
		if health.Metrics == nil ||
			health.Metrics.Coherence == nil ||
			health.Metrics.Freshness == nil ||
			health.Metrics.Redundancy == nil ||
			health.Metrics.Coverage == nil {
			continue
		}
		sumCoherence += *health.Metrics.Coherence
		sumFreshness += *health.Metrics.Freshness
		sumRedundancy += *health.Metrics.Redundancy
		sumCoverage += *health.Metrics.Coverage
		count++
	}

	if count == 0 {
		return nil
	}

	c := float64(count)
	coherence := sumCoherence / c
	freshness := sumFreshness / c
	redundancy := sumRedundancy / c
	coverage := sumCoverage / c
	return &QualityMetrics{
		Coherence:  &coherence,
		Freshness:  &freshness,
		Redundancy: &redundancy,
		Coverage:   &coverage,
	}
}

func knowledgeHealthScore(metrics *QualityMetrics) *float64 {
	if metrics == nil || metrics.Coherence == nil || metrics.Freshness == nil || metrics.Coverage == nil || metrics.Redundancy == nil {
		return nil
	}
	score := (*metrics.Coherence + *metrics.Freshness + *metrics.Coverage - *metrics.Redundancy) / 3.0
	return &score
}

// calculateQualityMetrics is a helper for testing [REQ:KO-QM-001,KO-QM-002,KO-QM-003]
func calculateQualityMetrics(vectors [][]float64, timestamps []time.Time) QualityMetrics {
	coherence := calculateCoherence(vectors)
	freshness := calculateFreshness(timestamps)
	redundancy := detectRedundancy(vectors, redundancySimilarityThreshold)

	// Coverage would require domain analysis.
	// In tests we keep a deterministic score to validate the metric wiring and ranges.
	coverage := defaultCoverageScore

	return QualityMetrics{
		Coherence:  &coherence,
		Freshness:  &freshness,
		Redundancy: &redundancy,
		Coverage:   &coverage,
	}
}

// formatHealthStatus converts numeric score to status string
func formatHealthStatus(score *float64) string {
	if score == nil {
		return "unknown"
	}
	switch {
	case *score >= 0.8:
		return "excellent"
	case *score >= 0.6:
		return "good"
	case *score >= 0.4:
		return "fair"
	default:
		return "poor"
	}
}
