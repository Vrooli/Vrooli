package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Global service instances (initialized in main)
var (
	treeService   *TreeService
	sectorService *SectorService
	stageService  *StageService
	graphService  *GraphService
)

// getTechTree retrieves the current tech tree based on request context.
func getTechTree(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tech tree not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve tech tree"})
		return
	}

	c.JSON(http.StatusOK, tree)
}

// listTechTrees retrieves all tech trees with optional filtering.
func listTechTrees(c *gin.Context) {
	filters := ListTreesFilters{
		TreeType:        strings.TrimSpace(c.Query("type")),
		Status:          strings.TrimSpace(c.Query("status")),
		IncludeArchived: strings.EqualFold(strings.TrimSpace(c.Query("include_archived")), "true"),
		TreeID:          strings.TrimSpace(c.Query(treeIDQueryParam)),
		Slug:            strings.TrimSpace(c.Query(treeSlugQueryParam)),
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	summaries, err := treeService.ListTrees(ctx, filters)
	if err != nil {
		log.Printf("ERROR listing tech trees: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tech trees"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"trees": summaries})
}

// createTechTreeHandler creates a new tech tree.
func createTechTreeHandler(c *gin.Context) {
	type requestBody struct {
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Description  string `json:"description"`
		TreeType     string `json:"tree_type"`
		Status       string `json:"status"`
		Version      string `json:"version"`
		ParentTreeID string `json:"parent_tree_id"`
		IsActive     *bool  `json:"is_active"`
	}

	var payload requestBody
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	params := CreateTreeParams{
		Name:         payload.Name,
		Slug:         payload.Slug,
		Description:  payload.Description,
		TreeType:     payload.TreeType,
		Status:       payload.Status,
		Version:      payload.Version,
		ParentTreeID: payload.ParentTreeID,
		IsActive:     payload.IsActive,
	}

	tree, err := treeService.CreateTree(ctx, params)
	if err != nil {
		log.Printf("ERROR: createTechTreeHandler failed: %v", err)
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create tech tree: %v", err)})
		return
	}

	writeTreeResponse(c, tree, http.StatusCreated)
}

// updateTechTreeHandler updates an existing tech tree.
func updateTechTreeHandler(c *gin.Context) {
	techTreeID := c.Param("id")
	if strings.TrimSpace(techTreeID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tech tree ID is required"})
		return
	}

	type requestBody struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		TreeType    *string `json:"tree_type"`
		Status      *string `json:"status"`
		Version     *string `json:"version"`
		Slug        *string `json:"slug"`
		IsActive    *bool   `json:"is_active"`
	}

	var payload requestBody
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	params := UpdateTreeParams{
		Name:        payload.Name,
		Description: payload.Description,
		TreeType:    payload.TreeType,
		Status:      payload.Status,
		Version:     payload.Version,
		Slug:        payload.Slug,
		IsActive:    payload.IsActive,
	}

	tree, err := treeService.UpdateTree(ctx, techTreeID, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tech tree not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Slug already exists"})
			return
		}
		if strings.Contains(err.Error(), "no updatable fields") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tech tree"})
		return
	}

	writeTreeResponse(c, tree, http.StatusOK)
}

// cloneTechTreeHandler creates a deep copy of an existing tech tree.
func cloneTechTreeHandler(c *gin.Context) {
	sourceTreeID := c.Param("id")
	if strings.TrimSpace(sourceTreeID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Source tech tree ID is required"})
		return
	}

	type requestBody struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		TreeType    string `json:"tree_type"`
		Status      string `json:"status"`
		IsActive    *bool  `json:"is_active"`
	}

	var payload requestBody
	if err := c.ShouldBindJSON(&payload); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	params := CloneTreeParams{
		Name:        payload.Name,
		Slug:        payload.Slug,
		Description: payload.Description,
		TreeType:    payload.TreeType,
		Status:      payload.Status,
		IsActive:    payload.IsActive,
	}

	tree, err := treeService.CloneTree(ctx, sourceTreeID, params)
	if err != nil {
		log.Printf("ERROR: cloneTechTreeHandler failed: %v", err)
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "source tree") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source tech tree not found"})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "Slug already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to clone tech tree: %v", err)})
		return
	}

	writeTreeResponse(c, tree, http.StatusCreated)
}
