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
		name := stringValue(resource["Name"])
		if name == "" {
			continue
		}

		enabled, enabledKnown := parseBool(resource["Enabled"])
		running, _ := parseBool(resource["Running"])
		statusDetail := stringValue(resource["Status"])
		normalizedStatus := strings.ToLower(statusDetail)

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

		transformedResources = append(transformedResources, map[string]interface{}{
			"id":            name,
			"name":          name,
			"type":          name,
			"status":        status,
			"enabled":       enabled,
			"enabled_known": enabledKnown,
			"running":       running,
			"status_detail": statusDetail,
		})
	}

	// Update cache
	h.resourceCache.mu.Lock()
	h.resourceCache.data = transformedResources
	h.resourceCache.timestamp = time.Now()
	h.resourceCache.mu.Unlock()

	c.JSON(http.StatusOK, transformedResources)
}
