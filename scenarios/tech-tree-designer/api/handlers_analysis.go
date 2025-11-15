package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
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

	// Generate strategic recommendations based on current state
	recommendations := generateStrategicRecommendations(request)

	// Calculate projected timeline
	timeline := calculateProjectedTimeline(request)

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

	tree, err := resolveTreeContext(c)
	if err != nil {
		log.Printf("analysis proceeding without tree context: %v", err)
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
func calculateProjectedTimeline(request AnalysisRequest) ProjectedTimeline {
	milestones := []MilestoneProjection{
		{
			Name:                "Personal Productivity Mastery",
			EstimatedCompletion: time.Now().AddDate(0, 6, 0),
			Confidence:          0.8,
		},
		{
			Name:                "Core Sector Digital Twins",
			EstimatedCompletion: time.Now().AddDate(2, 0, 0),
			Confidence:          0.6,
		},
		{
			Name:                "Civilization Digital Twin",
			EstimatedCompletion: time.Now().AddDate(5, 0, 0),
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
		ORDER BY business_value_estimate DESC
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
