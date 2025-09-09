package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
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

// NewSystemHandler creates a new system handler
func NewSystemHandler(metricsService *services.MetricsService) *SystemHandler {
	return &SystemHandler{
		metricsService: metricsService,
		resourceCache:  &resourceCache{},
	}
}

// GetSystemInfo returns system information including orchestrator status
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	// Create context with timeout (5s for system info)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get orchestrator PID
	pidCmd := exec.CommandContext(ctx, "bash", "-c", "ps aux | grep 'enhanced_orchestrator.py' | grep -v grep | awk '{print $2}' | head -1")
	pidOutput, err := pidCmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get orchestrator process",
		})
		return
	}

	pid := strings.TrimSpace(string(pidOutput))
	if pid == "" {
		c.JSON(http.StatusOK, gin.H{
			"orchestrator_running": false,
			"uptime":               "00:00:00",
			"uptime_seconds":       0,
		})
		return
	}

	// Get uptime using the PID (reuse context)
	uptimeCmd := exec.CommandContext(ctx, "ps", "-p", pid, "-o", "etime=")
	uptimeOutput, err := uptimeCmd.Output()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get orchestrator uptime",
		})
		return
	}

	uptime := strings.TrimSpace(string(uptimeOutput))
	uptimeSeconds := parseUptimeToSeconds(uptime)

	// Get orchestrator status from API
	orchStatus := make(map[string]interface{})
	resp, err := http.Get("http://localhost:9500/status")
	if err == nil {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&orchStatus)
	}

	c.JSON(http.StatusOK, gin.H{
		"orchestrator_running": true,
		"orchestrator_pid":     pid,
		"uptime":               uptime,
		"uptime_seconds":       uptimeSeconds,
		"orchestrator_status":  orchStatus,
	})
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
		name := resource["Name"].(string)
		enabled := resource["Enabled"].(bool)
		running := resource["Running"].(bool)

		status := "offline"
		if running {
			status = "online"
		} else if enabled {
			status = "stopped"
		}

		transformedResources = append(transformedResources, map[string]interface{}{
			"id":      name,
			"name":    name,
			"type":    name,
			"status":  status,
			"enabled": enabled,
			"running": running,
		})
	}

	// Update cache
	h.resourceCache.mu.Lock()
	h.resourceCache.data = transformedResources
	h.resourceCache.timestamp = time.Now()
	h.resourceCache.mu.Unlock()

	c.JSON(http.StatusOK, transformedResources)
}

// Helper function to parse uptime string to seconds
func parseUptimeToSeconds(uptime string) int {
	uptime = strings.TrimSpace(uptime)
	parts := strings.Split(uptime, ":")

	switch len(parts) {
	case 2: // MM:SS
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		return minutes*60 + seconds
	case 3: // HH:MM:SS or DD-HH:MM
		if strings.Contains(parts[0], "-") {
			// DD-HH:MM format
			dayHour := strings.Split(parts[0], "-")
			days, _ := strconv.Atoi(dayHour[0])
			hours, _ := strconv.Atoi(dayHour[1])
			minutes, _ := strconv.Atoi(parts[1])
			seconds, _ := strconv.Atoi(parts[2])
			return days*86400 + hours*3600 + minutes*60 + seconds
		} else {
			// HH:MM:SS format
			hours, _ := strconv.Atoi(parts[0])
			minutes, _ := strconv.Atoi(parts[1])
			seconds, _ := strconv.Atoi(parts[2])
			return hours*3600 + minutes*60 + seconds
		}
	default:
		return 0
	}
}