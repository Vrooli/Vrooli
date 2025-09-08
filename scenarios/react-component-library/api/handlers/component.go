package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vrooli/scenarios/react-component-library/models"
	"github.com/vrooli/scenarios/react-component-library/services"
)

type ComponentHandler struct {
	componentService *services.ComponentService
	testingService   *services.TestingService
	aiService        *services.AIService
	searchService    *services.SearchService
}

func NewComponentHandler(
	componentService *services.ComponentService,
	testingService *services.TestingService,
	aiService *services.AIService,
	searchService *services.SearchService,
) *ComponentHandler {
	return &ComponentHandler{
		componentService: componentService,
		testingService:   testingService,
		aiService:        aiService,
		searchService:    searchService,
	}
}

// ListComponents handles GET /api/v1/components
func (h *ComponentHandler) ListComponents(c *gin.Context) {
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	components, total, err := h.componentService.ListComponents(category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve components",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"components": components,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// CreateComponent handles POST /api/v1/components
func (h *ComponentHandler) CreateComponent(c *gin.Context) {
	var component models.Component
	if err := c.ShouldBindJSON(&component); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component data",
			"details": err.Error(),
		})
		return
	}

	// Generate UUID and set timestamps
	component.ID = uuid.New()
	component.CreatedAt = time.Now()
	component.UpdatedAt = time.Now()
	component.Version = "1.0.0"
	component.IsActive = true
	
	// Set author from context (if available)
	if author, exists := c.Get("user"); exists {
		component.Author = author.(string)
	} else {
		component.Author = "anonymous"
	}

	// Validate component code (basic syntax check)
	if err := h.componentService.ValidateComponentCode(component.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component code",
			"details": err.Error(),
		})
		return
	}

	// Create component
	if err := h.componentService.CreateComponent(&component); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create component",
			"details": err.Error(),
		})
		return
	}

	// Run initial tests asynchronously
	go func() {
		testRequest := models.ComponentTestRequest{
			TestTypes: []models.TestType{
				models.TestTypeAccessibility,
				models.TestTypePerformance,
				models.TestTypeLinting,
			},
		}
		h.testingService.RunTests(component.ID, testRequest)
	}()

	// Index in search service
	go func() {
		h.searchService.IndexComponent(&component)
	}()

	c.JSON(http.StatusCreated, component)
}

// GetComponent handles GET /api/v1/components/:id
func (h *ComponentHandler) GetComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	component, err := h.componentService.GetComponent(id)
	if err != nil {
		if err == services.ErrComponentNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Component not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve component",
			"details": err.Error(),
		})
		return
	}

	// Record usage analytics
	go func() {
		h.componentService.RecordUsage(id, c.GetHeader("User-Agent"), "api-view")
	}()

	c.JSON(http.StatusOK, component)
}

// UpdateComponent handles PUT /api/v1/components/:id
func (h *ComponentHandler) UpdateComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	var updateData models.Component
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid update data",
			"details": err.Error(),
		})
		return
	}

	// Validate component code if provided
	if updateData.Code != "" {
		if err := h.componentService.ValidateComponentCode(updateData.Code); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid component code",
				"details": err.Error(),
			})
			return
		}
	}

	updateData.ID = id
	updateData.UpdatedAt = time.Now()

	component, err := h.componentService.UpdateComponent(&updateData)
	if err != nil {
		if err == services.ErrComponentNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Component not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update component",
			"details": err.Error(),
		})
		return
	}

	// Re-run tests if code was updated
	if updateData.Code != "" {
		go func() {
			testRequest := models.ComponentTestRequest{
				TestTypes: []models.TestType{
					models.TestTypeAccessibility,
					models.TestTypePerformance,
					models.TestTypeLinting,
				},
			}
			h.testingService.RunTests(id, testRequest)
		}()
	}

	// Update search index
	go func() {
		h.searchService.UpdateComponentIndex(component)
	}()

	c.JSON(http.StatusOK, component)
}

// DeleteComponent handles DELETE /api/v1/components/:id
func (h *ComponentHandler) DeleteComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	if err := h.componentService.DeleteComponent(id); err != nil {
		if err == services.ErrComponentNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Component not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete component",
			"details": err.Error(),
		})
		return
	}

	// Remove from search index
	go func() {
		h.searchService.RemoveFromIndex(id)
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Component deleted successfully",
	})
}

// SearchComponents handles GET /api/v1/components/search
func (h *ComponentHandler) SearchComponents(c *gin.Context) {
	var searchReq models.ComponentSearchRequest
	if err := c.ShouldBindQuery(&searchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid search parameters",
			"details": err.Error(),
		})
		return
	}

	if searchReq.Limit == 0 {
		searchReq.Limit = 20
	}

	startTime := time.Now()
	response, err := h.searchService.SearchComponents(searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search failed",
			"details": err.Error(),
		})
		return
	}

	response.SearchTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, response)
}

// GenerateComponent handles POST /api/v1/components/generate
func (h *ComponentHandler) GenerateComponent(c *gin.Context) {
	var genReq models.ComponentGenerationRequest
	if err := c.ShouldBindJSON(&genReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid generation request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.aiService.GenerateComponent(genReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Component generation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// TestComponent handles POST /api/v1/components/:id/test
func (h *ComponentHandler) TestComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	var testReq models.ComponentTestRequest
	if err := c.ShouldBindJSON(&testReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid test request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.testingService.RunTests(id, testReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Testing failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// BenchmarkComponent handles GET /api/v1/components/:id/benchmark
func (h *ComponentHandler) BenchmarkComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	benchmark, err := h.testingService.BenchmarkComponent(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Benchmarking failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, benchmark)
}

// ExportComponent handles POST /api/v1/components/:id/export
func (h *ComponentHandler) ExportComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	var exportReq models.ComponentExportRequest
	if err := c.ShouldBindJSON(&exportReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid export request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.componentService.ExportComponent(id, exportReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Export failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ImproveComponent handles POST /api/v1/components/:id/improve
func (h *ComponentHandler) ImproveComponent(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	var improveReq models.ComponentImprovementRequest
	if err := c.ShouldBindJSON(&improveReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid improvement request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.aiService.ImproveComponent(id, improveReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Improvement analysis failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetComponentVersions handles GET /api/v1/components/:id/versions
func (h *ComponentHandler) GetComponentVersions(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid component ID",
		})
		return
	}

	versions, err := h.componentService.GetComponentVersions(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve versions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
	})
}

// GetUsageAnalytics handles GET /api/v1/analytics/usage
func (h *ComponentHandler) GetUsageAnalytics(c *gin.Context) {
	componentID := c.Query("component_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	analytics, err := h.componentService.GetUsageAnalytics(componentID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve analytics",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// GetPopularComponents handles GET /api/v1/analytics/popular
func (h *ComponentHandler) GetPopularComponents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	timeframe := c.DefaultQuery("timeframe", "30d")

	popular, err := h.componentService.GetPopularComponents(limit, timeframe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve popular components",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"popular_components": popular,
		"timeframe":         timeframe,
	})
}