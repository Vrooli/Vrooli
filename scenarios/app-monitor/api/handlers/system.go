package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

// SystemHandler handles system-related endpoints
type SystemHandler struct {
	metricsService *services.MetricsService
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(metricsService *services.MetricsService) *SystemHandler {
	return &SystemHandler{
		metricsService: metricsService,
	}
}

// GetSystemInfo returns system information including orchestrator status
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	// Get orchestrator PID
	pidCmd := exec.Command("bash", "-c", "ps aux | grep 'enhanced_orchestrator.py' | grep -v grep | awk '{print $2}' | head -1")
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

	// Get uptime using the PID
	uptimeCmd := exec.Command("ps", "-p", pid, "-o", "etime=")
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

// GetSystemMetrics returns current system metrics
func (h *SystemHandler) GetSystemMetrics(c *gin.Context) {
	metrics, err := h.metricsService.GetSystemMetrics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch system metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetResources returns the status of all resources
func (h *SystemHandler) GetResources(c *gin.Context) {
	// Execute vrooli resource status --json
	cmd := exec.Command("vrooli", "resource", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
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