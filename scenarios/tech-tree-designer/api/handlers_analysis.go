package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func analyzeStrategicPath(c *gin.Context) {
	var request AnalysisRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tree, err := resolveTreeContext(c)
	if err != nil {
		log.Printf("analysis proceeding without tree context: %v", err)
	}

	// Generate strategic recommendations based on current state
	recommendations := generateStrategicRecommendations(request)

	// Calculate projected timeline using persisted milestones when available
	timeline := calculateProjectedTimeline(tree, request)

	// Identify bottlenecks
	bottlenecks := identifyBottlenecks()

	// Analyze cross-sector impacts
	impacts := analyzeCrossSectorImpacts()

	response := AnalysisResponse{
		Recommendations:    recommendations,
		ProjectedTimeline:  timeline,
		BottleneckAnalysis: bottlenecks,
		CrossSectorImpacts: impacts,
	}

	body := gin.H{
		"recommendations":      response.Recommendations,
		"projected_timeline":   response.ProjectedTimeline,
		"bottleneck_analysis":  response.BottleneckAnalysis,
		"cross_sector_impacts": response.CrossSectorImpacts,
	}
	if tree != nil {
		body["tree"] = tree
	}

	c.JSON(http.StatusOK, body)
}

// Generate strategic recommendations
func generateStrategicRecommendations(request AnalysisRequest) []StrategicRecommendation {
	// Simple heuristic-based recommendations (in production, this would use AI)
	recommendations := []StrategicRecommendation{
		{
			Scenario:         "graph-studio",
			PriorityScore:    9.5,
			ImpactMultiplier: 3.2,
			Reasoning:        "Graph visualization capabilities are fundamental for all tech tree analysis and strategic planning interfaces",
		},
		{
			Scenario:         "research-assistant",
			PriorityScore:    9.0,
			ImpactMultiplier: 2.8,
			Reasoning:        "Research capabilities accelerate discovery of new technology pathways and sector connections",
		},
		{
			Scenario:         "data-structurer",
			PriorityScore:    8.5,
			ImpactMultiplier: 2.5,
			Reasoning:        "Data organization capabilities are required for all sector foundation stages",
		},
	}

	return recommendations
}

// Calculate projected timeline for milestones
func calculateProjectedTimeline(tree *TechTree, request AnalysisRequest) ProjectedTimeline {
	ctx := context.Background()
	if tree == nil {
		if fallbackTree, err := fetchDefaultTechTree(ctx); err == nil {
			tree = fallbackTree
		}
	}

	if db == nil || tree == nil {
		return defaultProjectedTimeline()
	}

	rows, err := db.QueryContext(ctx, `
		SELECT name, estimated_completion_date, completion_percentage, confidence_level,
		       business_value_estimate
		FROM strategic_milestones
		WHERE tree_id = $1
	`, tree.ID)
	if err != nil {
		log.Printf("calculateProjectedTimeline fallback due to query error: %v", err)
		return defaultProjectedTimeline()
	}
	defer rows.Close()

	now := time.Now()
	var projections []MilestoneProjection
	for rows.Next() {
		var (
			name                 string
			estimatedDate        sql.NullTime
			completionPercentage sql.NullFloat64
			confidenceLevel      sql.NullFloat64
		)

		if err := rows.Scan(&name, &estimatedDate, &completionPercentage, &confidenceLevel); err != nil {
			continue
		}

		estimated := estimatedDate.Time
		if !estimatedDate.Valid {
			estimated = inferCompletionDate(now, request.TimeHorizon, completionPercentage.Float64, len(projections))
		}
		if estimated.Before(now) {
			estimated = now.Add(24 * time.Hour)
		}

		confidence := confidenceLevel.Float64
		if !confidenceLevel.Valid {
			confidence = 0.75 - float64(len(projections))*0.1
		}
		confidence = math.Max(0.05, math.Min(0.95, confidence))

		estimated = estimated.Add(time.Duration(len(projections)) * 6 * time.Hour)

		projections = append(projections, MilestoneProjection{
			Name:                name,
			EstimatedCompletion: estimated,
			Confidence:          confidence,
		})
	}

	if len(projections) == 0 {
		return defaultProjectedTimeline()
	}

	sort.SliceStable(projections, func(i, j int) bool {
		return projections[i].EstimatedCompletion.Before(projections[j].EstimatedCompletion)
	})

	return ProjectedTimeline{Milestones: projections}
}

func inferCompletionDate(now time.Time, horizonMonths int, completionPercentage float64, index int) time.Time {
	months := horizonMonths
	if months <= 0 {
		months = 12
	}
	completion := completionPercentage
	if completion < 0 {
		completion = 0
	}
	if completion > 100 {
		completion = 100
	}
	remainingFraction := 1 - (completion / 100)
	if remainingFraction < 0.05 {
		remainingFraction = 0.05
	}
	projectedMonths := float64(months) * remainingFraction
	if projectedMonths < 1 {
		projectedMonths = 1
	}
	projectedMonths += float64(index) * 0.5
	return now.AddDate(0, int(math.Ceil(projectedMonths)), 0)
}

func defaultProjectedTimeline() ProjectedTimeline {
	now := time.Now()
	milestones := []MilestoneProjection{
		{
			Name:                "Personal Productivity Mastery",
			EstimatedCompletion: now.AddDate(0, 6, 0),
			Confidence:          0.8,
		},
		{
			Name:                "Core Sector Digital Twins",
			EstimatedCompletion: now.AddDate(2, 0, 0),
			Confidence:          0.6,
		},
		{
			Name:                "Civilization Digital Twin",
			EstimatedCompletion: now.AddDate(5, 0, 0),
			Confidence:          0.4,
		},
	}

	return ProjectedTimeline{Milestones: milestones}
}

// Identify bottlenecks in the tech tree
func identifyBottlenecks() []string {
	return []string{
		"Software engineering foundation stages need more scenario coverage",
		"Cross-sector integration patterns are underdeveloped",
		"AI analysis capabilities require more sophisticated models",
		"Real-time progress tracking needs automated scenario status detection",
	}
}

// Analyze cross-sector impacts
func analyzeCrossSectorImpacts() []CrossSectorImpact {
	return []CrossSectorImpact{
		{
			SourceSector: "Software Engineering",
			TargetSector: "Manufacturing",
			ImpactScore:  0.9,
			Description:  "Software development capabilities directly enable all manufacturing system development",
		},
		{
			SourceSector: "Healthcare",
			TargetSector: "Manufacturing",
			ImpactScore:  0.6,
			Description:  "Healthcare insights drive medical device and biotech manufacturing requirements",
		},
	}
}

// Get strategic milestones
func getStrategicMilestones(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	rows, err := db.Query(`
		SELECT id, tree_id, name, description, milestone_type, required_sectors,
			   required_stages, completion_percentage, estimated_completion_date,
			   confidence_level, business_value_estimate, created_at, updated_at
		FROM strategic_milestones
		WHERE tree_id = $1
		ORDER BY estimated_completion_date NULLS LAST,
		         business_value_estimate DESC,
		         created_at ASC
	`, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch milestones"})
		return
	}
	defer rows.Close()

	var milestones []StrategicMilestone
	for rows.Next() {
		var milestone StrategicMilestone
		err := rows.Scan(&milestone.ID, &milestone.TreeID, &milestone.Name,
			&milestone.Description, &milestone.MilestoneType, &milestone.RequiredSectors,
			&milestone.RequiredStages, &milestone.CompletionPercentage,
			&milestone.EstimatedCompletionDate, &milestone.ConfidenceLevel,
			&milestone.BusinessValueEstimate, &milestone.CreatedAt, &milestone.UpdatedAt)
		if err != nil {
			continue
		}
		milestones = append(milestones, milestone)
	}

	c.JSON(http.StatusOK, gin.H{"milestones": milestones, "tree": tree})
}

// Get recommendations (simplified version)
func getRecommendations(c *gin.Context) {
	resources := 5 // Default resource level
	if r := c.Query("resources"); r != "" {
		if parsed, err := strconv.Atoi(r); err == nil {
			resources = parsed
		}
	}

	request := AnalysisRequest{
		CurrentResources: resources,
		TimeHorizon:      12, // 12 months
		PrioritySectors:  []string{"software", "manufacturing", "healthcare"},
	}

	tree, err := resolveTreeContext(c)
	if err != nil {
		log.Printf("recommendations proceeding without tree context: %v", err)
	}

	recommendations := generateStrategicRecommendations(request)
	payload := gin.H{"recommendations": recommendations}
	if tree != nil {
		payload["tree"] = tree
	}
	c.JSON(http.StatusOK, payload)
}
