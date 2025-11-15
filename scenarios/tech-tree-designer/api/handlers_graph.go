package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Delegation function for backward compatibility
func fetchDependencies(ctx context.Context, treeID string) ([]DependencyPayload, error) {
	return graphService.FetchDependencies(ctx, treeID)
}

// Get dependencies
func getDependencies(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	dependencies, err := fetchDependencies(c.Request.Context(), tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dependencies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dependencies": dependencies, "tree": tree})
}

func updateGraph(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var request GraphUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Use request directly as it already matches the GraphUpdateRequest type
	sectors, dependencies, err := graphService.UpdateGraph(c.Request.Context(), tree.ID, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Graph updated successfully",
		"tree":         tree,
		"sectors":      sectors,
		"dependencies": dependencies,
	})
}

// Get cross-sector connections
func getCrossSectorConnections(c *gin.Context) {
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
		SELECT sc.id, sc.source_sector_id, sc.target_sector_id,
			   sc.connection_type, sc.strength, sc.description, sc.examples,
			   s1.name as source_name, s2.name as target_name
		FROM sector_connections sc
		JOIN sectors s1 ON sc.source_sector_id = s1.id
		JOIN sectors s2 ON sc.target_sector_id = s2.id
		WHERE s1.tree_id = $1 AND s2.tree_id = $1
		ORDER BY sc.strength DESC
	`, tree.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connections"})
		return
	}
	defer rows.Close()

	var connections []gin.H
	for rows.Next() {
		var conn struct {
			ID             string          `json:"id"`
			SourceSectorID string          `json:"source_sector_id"`
			TargetSectorID string          `json:"target_sector_id"`
			ConnectionType string          `json:"connection_type"`
			Strength       float64         `json:"strength"`
			Description    string          `json:"description"`
			Examples       json.RawMessage `json:"examples"`
		}
		var sourceName, targetName string

		err := rows.Scan(&conn.ID, &conn.SourceSectorID, &conn.TargetSectorID,
			&conn.ConnectionType, &conn.Strength, &conn.Description, &conn.Examples,
			&sourceName, &targetName)
		if err != nil {
			continue
		}

		connections = append(connections, gin.H{
			"connection":  conn,
			"source_name": sourceName,
			"target_name": targetName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"connections": connections, "tree": tree})
}

// Add new scenario mapping

func exportGraphAsDOT(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
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

	dotOutput := graphService.ExportGraphAsDOT(ctx, tree, sectors, dependencies)
	c.Data(http.StatusOK, "text/vnd.graphviz", []byte(dotOutput))
}
