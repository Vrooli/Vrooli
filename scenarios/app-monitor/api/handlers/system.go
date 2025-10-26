package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
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

type ResourceStatusSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Status       string `json:"status"`
	Enabled      bool   `json:"enabled"`
	EnabledKnown bool   `json:"enabledKnown"`
	Running      bool   `json:"running"`
	StatusDetail string `json:"statusDetail"`
}

type ResourcePaths struct {
	ServiceConfig string `json:"serviceConfig,omitempty"`
	RuntimeConfig string `json:"runtimeConfig,omitempty"`
	Capabilities  string `json:"capabilities,omitempty"`
	Schema        string `json:"schema,omitempty"`
}

type ResourceDetailResponse struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Category           string                 `json:"category,omitempty"`
	Description        string                 `json:"description,omitempty"`
	Summary            ResourceStatusSummary  `json:"summary"`
	CLIStatus          map[string]interface{} `json:"cliStatus,omitempty"`
	ServiceConfig      map[string]interface{} `json:"serviceConfig,omitempty"`
	RuntimeConfig      map[string]interface{} `json:"runtimeConfig,omitempty"`
	CapabilityMetadata map[string]interface{} `json:"capabilityMetadata,omitempty"`
	Schema             map[string]interface{} `json:"schema,omitempty"`
	Paths              ResourcePaths          `json:"paths"`
}

func findRepoRoot() (string, error) {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root, nil
	}
	if root := os.Getenv("APP_ROOT"); root != "" {
		return root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if dir == "" || dir == string(filepath.Separator) {
			break
		}
		if _, err := os.Stat(filepath.Join(dir, ".vrooli")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repository root not found from %s", wd)
}

func toRelativePath(root, target string) string {
	if root == "" || target == "" {
		return target
	}
	if rel, err := filepath.Rel(root, target); err == nil {
		return rel
	}
	return target
}

func readJSONFile(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func readYAMLFile(path string) (map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func loadServiceResourceConfig(root, name string) (map[string]interface{}, string, error) {
	if root == "" {
		return nil, "", fmt.Errorf("repository root not available")
	}
	path := filepath.Join(root, ".vrooli", "service.json")
	data, err := readJSONFile(path)
	if err != nil {
		return nil, path, err
	}
	resourcesRaw, ok := data["resources"].(map[string]interface{})
	if !ok {
		return nil, path, fmt.Errorf("service.json missing resources section")
	}
	entryRaw, exists := resourcesRaw[name]
	if !exists {
		return nil, path, fs.ErrNotExist
	}
	entry, ok := entryRaw.(map[string]interface{})
	if !ok {
		return nil, path, fmt.Errorf("resource %s entry is not an object", name)
	}
	return entry, path, nil
}

func loadJSONIfExists(path string) (map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	return readJSONFile(path)
}

func loadYAMLIfExists(path string) (map[string]interface{}, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	return readYAMLFile(path)
}

func buildSummaryFromMap(data map[string]interface{}) ResourceStatusSummary {
	summary := ResourceStatusSummary{
		ID:           stringValue(data["id"]),
		Name:         stringValue(data["name"]),
		Type:         stringValue(data["type"]),
		Status:       stringValue(data["status"]),
		StatusDetail: stringValue(data["status_detail"]),
	}
	if value, known := parseBool(data["enabled"]); known {
		summary.Enabled = value
	}
	if enabledKnownRaw, ok := data["enabled_known"]; ok {
		if value, known := parseBool(enabledKnownRaw); known {
			summary.EnabledKnown = value
		} else if boolVal, ok := enabledKnownRaw.(bool); ok {
			summary.EnabledKnown = boolVal
		}
	}
	if value, known := parseBool(data["running"]); known {
		summary.Running = value
	}
	if summary.Status == "" {
		summary.Status = stringValue(data["status"])
	}
	if summary.Type == "" {
		summary.Type = stringValue(data["name"])
	}
	return summary
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

func (h *SystemHandler) removeFromResourceCache(id string) {
	h.resourceCache.mu.Lock()
	defer h.resourceCache.mu.Unlock()

	if len(h.resourceCache.data) == 0 {
		return
	}

	filtered := make([]map[string]interface{}, 0, len(h.resourceCache.data))
	for _, resource := range h.resourceCache.data {
		if stringValue(resource["id"]) != id {
			filtered = append(filtered, resource)
		}
	}
	h.resourceCache.data = filtered
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
func (h *SystemHandler) fetchResourceDetail(ctx context.Context, name string) (map[string]interface{}, map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "resource", "status", name, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	var parsed interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, nil, err
	}

	var candidate map[string]interface{}
	switch value := parsed.(type) {
	case []interface{}:
		for _, item := range value {
			typed, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			resourceName := stringValue(lookupValue(typed, "Name"))
			if resourceName == "" {
				resourceName = stringValue(lookupValue(typed, "name"))
			}
			if resourceName != "" && strings.EqualFold(resourceName, name) {
				candidate = typed
				break
			}
		}
		if candidate == nil {
			for _, item := range value {
				typed, ok := item.(map[string]interface{})
				if ok {
					candidate = typed
					break
				}
			}
		}
	case map[string]interface{}:
		candidate = value
	default:
		return nil, nil, fmt.Errorf("unexpected response for resource %s", name)
	}

	if candidate == nil {
		return nil, nil, fmt.Errorf("resource %s not found", name)
	}

	if transformed, valid := h.transformResource(candidate); valid {
		return candidate, transformed, nil
	}

	return nil, nil, fmt.Errorf("resource %s response missing required fields", name)
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

// GetSystemStatus returns lightweight system status for quick polling
// This endpoint is optimized to avoid expensive full data fetches
func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get counts from orchestrator cache (fast, uses existing cache)
	appCount, resourceCount, uptime, status := getQuickSystemStatus(ctx)

	c.JSON(http.StatusOK, gin.H{
		"status":          status,
		"app_count":       appCount,
		"resource_count":  resourceCount,
		"uptime_seconds":  uptime,
		"checked_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

// getQuickSystemStatus retrieves system status without expensive operations
// Uses single unified CLI call for optimal performance
func getQuickSystemStatus(ctx context.Context) (appCount int, resourceCount int, uptimeSeconds *float64, status string) {
	status = "degraded" // Default to degraded if command fails

	// Single unified CLI call - much faster than separate scenario/resource list commands
	// Timeout set to 15s to accommodate resource health checks (with 118 resources, this can take 10-15s)
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "vrooli", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		// Log error for debugging but don't expose details to API response
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "vrooli status command failed: %s\n", string(exitErr.Stderr))
		}
		// If status command fails, return degraded status with zero counts
		return 0, 0, nil, "degraded"
	}

	// Parse unified status response
	var statusData struct {
		ScenariosTotal   int    `json:"scenarios_total"`
		ResourcesEnabled int    `json:"resources_enabled"`
		HealthStatus     string `json:"health_status"`
	}

	if err := json.Unmarshal(output, &statusData); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse vrooli status response: %v\n", err)
		return 0, 0, nil, "degraded"
	}

	appCount = statusData.ScenariosTotal
	resourceCount = statusData.ResourcesEnabled
	status = statusData.HealthStatus

	// Normalize status value to expected format
	if status == "" {
		status = "healthy"
	}

	// Get orchestrator uptime separately (quick operation with 500ms timeout)
	quickCtx, quickCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer quickCancel()

	if uptime, err := getOrchestratorUptime(quickCtx); err == nil {
		uptimeSeconds = &uptime
	}
	// Note: uptime failure doesn't change status - status comes from vrooli status command

	return appCount, resourceCount, uptimeSeconds, status
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

	// Fetch fresh status for this specific resource and update cache
	statusCtx, statusCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer statusCancel()

	resource, fetchErr := h.fetchResourceStatus(statusCtx, name)
	if fetchErr == nil {
		// Update only this resource in cache (preserves other resources)
		h.upsertResourceCache(resource)
	} else {
		// Remove from cache on error so it gets re-fetched next time
		h.removeFromResourceCache(name)
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

func (h *SystemHandler) GetResourceDetails(c *gin.Context) {
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

	rawStatus, summaryMap, err := h.fetchResourceDetail(ctx, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to fetch resource %s details: %v", name, err),
		})
		return
	}

	summary := buildSummaryFromMap(summaryMap)
	h.upsertResourceCache(summaryMap)

	detail := ResourceDetailResponse{
		ID:          summary.ID,
		Name:        summary.Name,
		Category:    stringValue(lookupValue(rawStatus, "Category")),
		Description: stringValue(lookupValue(rawStatus, "Description")),
		Summary:     summary,
		CLIStatus:   rawStatus,
		Paths:       ResourcePaths{},
	}

	if detail.Category == "" {
		detail.Category = summary.Type
	}

	if root, rootErr := findRepoRoot(); rootErr == nil {
		if serviceCfg, servicePath, err := loadServiceResourceConfig(root, name); err == nil {
			detail.ServiceConfig = serviceCfg
			detail.Paths.ServiceConfig = toRelativePath(root, servicePath)
			if detail.Description == "" {
				detail.Description = stringValue(serviceCfg["description"])
			}
			if detail.Category == "" {
				if category := stringValue(serviceCfg["category"]); category != "" {
					detail.Category = category
				}
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			detail.Paths.ServiceConfig = toRelativePath(root, filepath.Join(root, ".vrooli", "service.json"))
		}

		resourceRoot := filepath.Join(root, "resources", name)

		runtimePath := filepath.Join(resourceRoot, "config", "runtime.json")
		if runtimeCfg, err := loadJSONIfExists(runtimePath); err == nil {
			detail.RuntimeConfig = runtimeCfg
			detail.Paths.RuntimeConfig = toRelativePath(root, runtimePath)
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			detail.Paths.RuntimeConfig = toRelativePath(root, runtimePath)
		}

		capabilitiesPath := filepath.Join(resourceRoot, "config", "capabilities.yaml")
		if capabilityData, err := loadYAMLIfExists(capabilitiesPath); err == nil {
			detail.CapabilityMetadata = capabilityData
			detail.Paths.Capabilities = toRelativePath(root, capabilitiesPath)
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			detail.Paths.Capabilities = toRelativePath(root, capabilitiesPath)
		}

		schemaPath := filepath.Join(resourceRoot, "config", "schema.json")
		if schemaData, err := loadJSONIfExists(schemaPath); err == nil {
			detail.Schema = schemaData
			detail.Paths.Schema = toRelativePath(root, schemaPath)
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			detail.Paths.Schema = toRelativePath(root, schemaPath)
		}
	}

	if detail.Description == "" && summary.StatusDetail != "" {
		detail.Description = summary.StatusDetail
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    detail,
	})
}
