package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

// resourceCache caches resource status to prevent excessive command execution
type resourceCache struct {
	data       []map[string]interface{}
	timestamp  time.Time
	mu         sync.RWMutex
	fetching   bool
	fetchMutex sync.Mutex
}

// SystemHandler handles system-related endpoints
type SystemHandler struct {
	metricsService *services.MetricsService
	resourceCache  *resourceCache
}

func lookupValue(raw map[string]interface{}, key string) interface{} {
	if raw == nil {
		return nil
	}
	for k, v := range raw {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

func (h *SystemHandler) invalidateResourceCache() {
	h.resourceCache.mu.Lock()
	defer h.resourceCache.mu.Unlock()
	h.resourceCache.data = nil
	h.resourceCache.timestamp = time.Time{}
}

func (h *SystemHandler) upsertResourceCache(resource map[string]interface{}) {
	if resource == nil {
		return
	}

	h.resourceCache.mu.Lock()
	defer h.resourceCache.mu.Unlock()

	if len(h.resourceCache.data) == 0 {
		h.resourceCache.data = []map[string]interface{}{resource}
		h.resourceCache.timestamp = time.Now()
		return
	}

	replaced := false
	for i, existing := range h.resourceCache.data {
		if stringValue(existing["id"]) == stringValue(resource["id"]) {
			h.resourceCache.data[i] = resource
			replaced = true
			break
		}
	}

	if !replaced {
		h.resourceCache.data = append(h.resourceCache.data, resource)
	}

	h.resourceCache.timestamp = time.Now()
}

func (h *SystemHandler) transformResource(raw map[string]interface{}) (map[string]interface{}, bool) {
	name := stringValue(lookupValue(raw, "Name"))
	if name == "" {
		return nil, false
	}

	enabled, enabledKnown := parseBool(lookupValue(raw, "Enabled"))
	running, _ := parseBool(lookupValue(raw, "Running"))
	statusDetail := stringValue(lookupValue(raw, "Status"))
	normalizedStatus := strings.ToLower(statusDetail)
	typeValue := stringValue(lookupValue(raw, "Type"))
	if typeValue == "" {
		typeValue = name
	}

	status := "offline"
	switch {
	case strings.Contains(normalizedStatus, "unregistered"):
		status = "unregistered"
	case strings.Contains(normalizedStatus, "error") || strings.Contains(normalizedStatus, "failed"):
		status = "error"
	case running:
		status = "online"
	case enabled && enabledKnown:
		status = "stopped"
	case !enabledKnown && !running:
		status = "unknown"
	}

	return map[string]interface{}{
		"id":            name,
		"name":          name,
		"type":          typeValue,
		"status":        status,
		"enabled":       enabled,
		"enabled_known": enabledKnown,
		"running":       running,
		"status_detail": statusDetail,
	}, true
}

func (h *SystemHandler) fetchResourceStatus(ctx context.Context, name string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "resource", "status", name, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var parsed interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, err
	}

	switch value := parsed.(type) {
	case []interface{}:
		for _, item := range value {
			if typed, ok := item.(map[string]interface{}); ok {
				if transformed, valid := h.transformResource(typed); valid {
					return transformed, nil
				}
			}
		}
	case map[string]interface{}:
		if transformed, valid := h.transformResource(value); valid {
			return transformed, nil
		}
	}

	return nil, fmt.Errorf("resource %s not found", name)
}

func parseBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		normalized := strings.TrimSpace(strings.ToLower(v))
		switch normalized {
		case "true", "yes", "y", "1", "online", "running", "enabled":
			return true, true
		case "false", "no", "n", "0", "offline", "disabled", "stopped":
			return false, true
		case "", "n/a", "na", "unknown":
			return false, false
		default:
			return false, false
		}
	case float64:
		return v != 0, true
	default:
		return false, false
	}
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(metricsService *services.MetricsService) *SystemHandler {
	return &SystemHandler{
		metricsService: metricsService,
		resourceCache:  &resourceCache{},
	}
}

// GetSystemMetrics placeholder - metrics are now handled by system-monitor iframe
func (h *SystemHandler) GetSystemMetrics(c *gin.Context) {
	// Return empty metrics - actual metrics shown via system-monitor iframe
	c.JSON(http.StatusOK, gin.H{
		"message": "Metrics are displayed via system-monitor iframe",
	})
}

// GetResources returns the status of all resources with caching
func (h *SystemHandler) GetResources(c *gin.Context) {
	// Check cache first (cache valid for 20 seconds)
	h.resourceCache.mu.RLock()
	if time.Since(h.resourceCache.timestamp) < 20*time.Second && len(h.resourceCache.data) > 0 {
		cachedData := h.resourceCache.data
		h.resourceCache.mu.RUnlock()
		c.JSON(http.StatusOK, cachedData)
		return
	}
	h.resourceCache.mu.RUnlock()

	// Prevent concurrent fetches
	h.resourceCache.fetchMutex.Lock()
	defer h.resourceCache.fetchMutex.Unlock()

	// Check cache again after acquiring lock (another request might have updated it)
	h.resourceCache.mu.RLock()
	if time.Since(h.resourceCache.timestamp) < 20*time.Second && len(h.resourceCache.data) > 0 {
		cachedData := h.resourceCache.data
		h.resourceCache.mu.RUnlock()
		c.JSON(http.StatusOK, cachedData)
		return
	}
	h.resourceCache.mu.RUnlock()

	// Mark as fetching
	h.resourceCache.mu.Lock()
	h.resourceCache.fetching = true
	h.resourceCache.mu.Unlock()

	defer func() {
		h.resourceCache.mu.Lock()
		h.resourceCache.fetching = false
		h.resourceCache.mu.Unlock()
	}()

	// Execute vrooli resource status --json with timeout (use --fast to skip health checks)
	// 30 second timeout to account for slow resources that need attention
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "resource", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Return cached data if available on error
		h.resourceCache.mu.RLock()
		if len(h.resourceCache.data) > 0 {
			cachedData := h.resourceCache.data
			h.resourceCache.mu.RUnlock()
			c.JSON(http.StatusOK, cachedData)
			return
		}
		h.resourceCache.mu.RUnlock()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch resources",
		})
		return
	}

	// Parse the JSON response
	var resources []map[string]interface{}
	if err := json.Unmarshal(output, &resources); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse resource response",
		})
		return
	}

	// Transform resource data for frontend
	transformedResources := make([]map[string]interface{}, 0, len(resources))
	for _, resource := range resources {
		if transformed, valid := h.transformResource(resource); valid {
			transformedResources = append(transformedResources, transformed)
		}
	}

	// Update cache
	h.resourceCache.mu.Lock()
	h.resourceCache.data = transformedResources
	h.resourceCache.timestamp = time.Now()
	h.resourceCache.mu.Unlock()

	c.JSON(http.StatusOK, transformedResources)
}

func (h *SystemHandler) handleResourceAction(c *gin.Context, action string) {
	name := strings.TrimSpace(c.Param("id"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Resource name is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "vrooli", "resource", action, name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to %s resource %s: %s", action, name, errMsg),
		})
		return
	}

	// Invalidate cache so subsequent requests fetch fresh data
	h.invalidateResourceCache()

	statusCtx, statusCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer statusCancel()

	resource, fetchErr := h.fetchResourceStatus(statusCtx, name)
	if fetchErr == nil {
		h.upsertResourceCache(resource)
	}

	response := gin.H{
		"success": true,
		"message": fmt.Sprintf("Resource %s command '%s' executed successfully", name, action),
	}
	if resource != nil {
		response["data"] = resource
	}

	if fetchErr != nil {
		response["warning"] = fmt.Sprintf("Command succeeded but failed to refresh status: %v", fetchErr)
	}

	c.JSON(http.StatusOK, response)
}

// StartResource starts a specific resource via the Vrooli CLI.
func (h *SystemHandler) StartResource(c *gin.Context) {
	h.handleResourceAction(c, "start")
}

// StopResource stops a specific resource via the Vrooli CLI.
func (h *SystemHandler) StopResource(c *gin.Context) {
	h.handleResourceAction(c, "stop")
}

// GetResourceStatus refreshes the status for a single resource.
func (h *SystemHandler) GetResourceStatus(c *gin.Context) {
	name := strings.TrimSpace(c.Param("id"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Resource name is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resource, err := h.fetchResourceStatus(ctx, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch resource %s status: %v", name, err),
		})
		return
	}

	h.upsertResourceCache(resource)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resource,
	})
}
