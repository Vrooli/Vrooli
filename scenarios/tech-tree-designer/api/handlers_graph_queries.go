package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Get neighborhood of a stage (stages within N hops via dependencies)
func getStageNeighborhood(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	stageID := c.Query("stage_id")
	if stageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stage_id query parameter required"})
		return
	}

	depth, _ := strconv.Atoi(c.Query("depth"))
	if depth == 0 {
		depth = 2 // Default to 2 hops
	}

	direction := c.Query("direction")
	if direction == "" {
		direction = "both"
	}

	includeHierarchy := c.Query("include_hierarchy") == "true"
	includeScenarios := c.Query("include_scenarios") == "true"

	maxResults, _ := strconv.Atoi(c.Query("max_results"))
	if maxResults == 0 {
		maxResults = 100
	}

	opts := NeighborhoodOptions{
		StageID:          stageID,
		Depth:            depth,
		Direction:        direction,
		IncludeHierarchy: includeHierarchy,
		IncludeScenarios: includeScenarios,
		MaxResults:       maxResults,
	}

	result, err := graphQueryService.GetNeighborhood(c.Request.Context(), tree.ID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tree":         tree,
		"neighborhood": result,
	})
}

// Get shortest path between two stages
func getShortestPath(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	fromStageID := c.Query("from")
	toStageID := c.Query("to")

	if fromStageID == "" || toStageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both 'from' and 'to' query parameters required"})
		return
	}

	maxDepth, _ := strconv.Atoi(c.Query("max_depth"))
	if maxDepth == 0 {
		maxDepth = 10
	}

	opts := PathOptions{
		FromStageID: fromStageID,
		ToStageID:   toStageID,
		MaxDepth:    maxDepth,
	}

	result, err := graphQueryService.GetShortestPath(c.Request.Context(), tree.ID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tree": tree,
		"path": result,
	})
}

// Get hierarchical ancestor chain
func getStageAncestors(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	stageID := c.Query("stage_id")
	if stageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stage_id query parameter required"})
		return
	}

	depth, _ := strconv.Atoi(c.Query("depth"))
	if depth == 0 {
		depth = 10 // Default to full chain
	}

	includeChildren := c.Query("include_children") == "true"

	opts := AncestorOptions{
		StageID:         stageID,
		Depth:           depth,
		IncludeChildren: includeChildren,
	}

	result, err := graphQueryService.GetAncestors(c.Request.Context(), tree.ID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tree":      tree,
		"ancestors": result,
	})
}

// Export graph view as text for LLM consumption
func exportGraphViewAsText(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	// Get optional filters
	stageID := c.Query("stage_id")
	depth, _ := strconv.Atoi(c.Query("depth"))
	format := c.Query("format") // "text" or "json"
	if format == "" {
		format = "text"
	}

	ctx := c.Request.Context()

	// If stage_id provided, get neighborhood; otherwise get full tree
	if stageID != "" {
		if depth == 0 {
			depth = 2
		}

		opts := NeighborhoodOptions{
			StageID:          stageID,
			Depth:            depth,
			Direction:        "both",
			IncludeHierarchy: true,
			IncludeScenarios: true,
			MaxResults:       200,
		}

		result, err := graphQueryService.GetNeighborhood(ctx, tree.ID, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if format == "json" {
			c.JSON(http.StatusOK, gin.H{
				"tree":         tree,
				"neighborhood": result,
				"query_type":   "neighborhood",
			})
			return
		}

		// Generate text format
		text := generateTextExport(tree, result)
		c.Data(http.StatusOK, "text/plain", []byte(text))
		return
	}

	// Full tree export
	sectors, err := fetchSectorsWithStages(ctx, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sectors"})
		return
	}

	dependencies, err := fetchDependencies(ctx, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load dependencies"})
		return
	}

	if format == "json" {
		c.JSON(http.StatusOK, gin.H{
			"tree":         tree,
			"sectors":      sectors,
			"dependencies": dependencies,
			"query_type":   "full_tree",
		})
		return
	}

	// Generate text format for full tree
	text := generateFullTreeText(tree, sectors, dependencies)
	c.Data(http.StatusOK, "text/plain", []byte(text))
}

// Helper: Generate text export for neighborhood
func generateTextExport(tree *TechTree, neighborhood *NeighborhoodResponse) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Tech Tree: %s\n", tree.Name))
	builder.WriteString(fmt.Sprintf("Tree Type: %s\n", tree.TreeType))
	builder.WriteString(fmt.Sprintf("Query Type: Neighborhood\n\n"))

	builder.WriteString(fmt.Sprintf("## Origin Stage: %s\n", neighborhood.Origin.Stage.Name))
	builder.WriteString(fmt.Sprintf("Sector: %s (%s)\n", neighborhood.Origin.Sector.Name, neighborhood.Origin.Sector.Category))
	builder.WriteString(fmt.Sprintf("Type: %s\n", neighborhood.Origin.Stage.StageType))
	builder.WriteString(fmt.Sprintf("Progress: %.1f%%\n", neighborhood.Origin.Stage.ProgressPercentage))
	if neighborhood.Origin.Stage.Description != "" {
		builder.WriteString(fmt.Sprintf("Description: %s\n", neighborhood.Origin.Stage.Description))
	}
	builder.WriteString("\n")

	if len(neighborhood.Origin.ScenarioMappings) > 0 {
		builder.WriteString("### Linked Scenarios:\n")
		for _, mapping := range neighborhood.Origin.ScenarioMappings {
			builder.WriteString(fmt.Sprintf("- %s (%s, priority %d)\n",
				mapping.ScenarioName, mapping.CompletionStatus, mapping.Priority))
		}
		builder.WriteString("\n")
	}

	if len(neighborhood.Stages) > 0 {
		builder.WriteString(fmt.Sprintf("## Connected Stages (within %d hops):\n\n", neighborhood.MaxDepth))
		for _, stage := range neighborhood.Stages {
			builder.WriteString(fmt.Sprintf("### %s (distance: %d)\n", stage.Stage.Name, stage.Distance))
			builder.WriteString(fmt.Sprintf("Sector: %s\n", stage.Sector.Name))
			builder.WriteString(fmt.Sprintf("Type: %s\n", stage.Stage.StageType))
			builder.WriteString(fmt.Sprintf("Progress: %.1f%%\n", stage.Stage.ProgressPercentage))

			if len(stage.ScenarioMappings) > 0 {
				builder.WriteString("Scenarios: ")
				for i, mapping := range stage.ScenarioMappings {
					if i > 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(mapping.ScenarioName)
				}
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// Helper: Generate text export for full tree
func generateFullTreeText(tree *TechTree, sectors []Sector, dependencies []DependencyPayload) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Tech Tree: %s\n", tree.Name))
	builder.WriteString(fmt.Sprintf("Tree Type: %s\n", tree.TreeType))
	builder.WriteString(fmt.Sprintf("Version: %s\n\n", tree.Version))

	for _, sector := range sectors {
		builder.WriteString(fmt.Sprintf("## Sector: %s\n", sector.Name))
		builder.WriteString(fmt.Sprintf("Category: %s\n", sector.Category))
		builder.WriteString(fmt.Sprintf("Progress: %.1f%%\n", sector.ProgressPercentage))
		if sector.Description != "" {
			builder.WriteString(fmt.Sprintf("%s\n", sector.Description))
		}
		builder.WriteString("\n")

		if len(sector.Stages) > 0 {
			builder.WriteString("### Stages:\n")
			for _, stage := range sector.Stages {
				builder.WriteString(fmt.Sprintf("- **%s** (%s) - %.1f%%\n",
					stage.Name, stage.StageType, stage.ProgressPercentage))

				if len(stage.ScenarioMappings) > 0 {
					builder.WriteString("  Scenarios: ")
					for i, mapping := range stage.ScenarioMappings {
						if i > 0 {
							builder.WriteString(", ")
						}
						builder.WriteString(fmt.Sprintf("%s (%s)", mapping.ScenarioName, mapping.CompletionStatus))
					}
					builder.WriteString("\n")
				}
			}
			builder.WriteString("\n")
		}
	}

	if len(dependencies) > 0 {
		builder.WriteString("## Dependencies\n\n")
		for _, dep := range dependencies {
			builder.WriteString(fmt.Sprintf("- %s → %s (%s, strength: %.2f)\n",
				dep.PrerequisiteName, dep.DependentName,
				dep.Dependency.DependencyType, dep.Dependency.DependencyStrength))
		}
	}

	return builder.String()
}
