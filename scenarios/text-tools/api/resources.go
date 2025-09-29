package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	resourceCheckInterval = 60 * time.Second // Check every minute
	resourceTimeout       = 5 * time.Second
)

// ResourceManager manages optional resource connections
type ResourceManager struct {
	config       *Config
	mu           sync.RWMutex
	resources    map[string]*ResourceStatus
	stopChan     chan struct{}
	isMonitoring bool
}

// ResourceStatus tracks the status of a resource
type ResourceStatus struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Available    bool      `json:"available"`
	LastCheck    time.Time `json:"last_check"`
	LastError    string    `json:"last_error,omitempty"`
	CheckCount   int       `json:"check_count"`
	SuccessCount int       `json:"success_count"`
}

// NewResourceManager creates a new resource manager
func NewResourceManager(config *Config) *ResourceManager {
	rm := &ResourceManager{
		config:    config,
		resources: make(map[string]*ResourceStatus),
		stopChan:  make(chan struct{}),
	}

	// Initialize resource statuses
	if config.MinIOURL != "" {
		rm.resources["minio"] = &ResourceStatus{
			Name: "minio",
			URL:  config.MinIOURL + "/minio/health/live",
		}
	}

	if config.RedisURL != "" {
		rm.resources["redis"] = &ResourceStatus{
			Name: "redis",
			URL:  config.RedisURL + "/ping",
		}
	}

	if config.OllamaURL != "" {
		rm.resources["ollama"] = &ResourceStatus{
			Name: "ollama",
			URL:  config.OllamaURL + "/api/tags",
		}
	}

	if config.QdrantURL != "" {
		rm.resources["qdrant"] = &ResourceStatus{
			Name: "qdrant",
			URL:  config.QdrantURL + "/health",
		}
	}

	return rm
}

// Start begins resource monitoring
func (rm *ResourceManager) Start() {
	rm.mu.Lock()
	if rm.isMonitoring {
		rm.mu.Unlock()
		return
	}
	rm.isMonitoring = true
	rm.mu.Unlock()

	log.Println("Starting resource monitoring...")

	// Initial check
	rm.checkAllResources()

	// Start monitoring goroutine
	go rm.monitorResources()
}

// Stop stops resource monitoring
func (rm *ResourceManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if !rm.isMonitoring {
		return
	}

	log.Println("Stopping resource monitoring...")
	close(rm.stopChan)
	rm.isMonitoring = false
}

// monitorResources continuously monitors resource availability
func (rm *ResourceManager) monitorResources() {
	ticker := time.NewTicker(resourceCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.checkAllResources()
		case <-rm.stopChan:
			return
		}
	}
}

// checkAllResources checks the availability of all resources
func (rm *ResourceManager) checkAllResources() {
	rm.mu.RLock()
	resources := make([]*ResourceStatus, 0, len(rm.resources))
	for _, resource := range rm.resources {
		resources = append(resources, resource)
	}
	rm.mu.RUnlock()

	// Check resources concurrently
	var wg sync.WaitGroup
	for _, resource := range resources {
		wg.Add(1)
		go func(res *ResourceStatus) {
			defer wg.Done()
			rm.checkResource(res)
		}(resource)
	}
	wg.Wait()

	// Log availability changes
	rm.logAvailabilityChanges()
}

// checkResource checks the availability of a single resource
func (rm *ResourceManager) checkResource(resource *ResourceStatus) {
	// Special handling for Redis
	if resource.Name == "redis" {
		rm.checkRedis(resource)
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), resourceTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", resource.URL, nil)
	if err != nil {
		rm.updateResourceStatus(resource, false, err.Error())
		return
	}

	client := &http.Client{Timeout: resourceTimeout}
	resp, err := client.Do(req)
	if err != nil {
		rm.updateResourceStatus(resource, false, err.Error())
		return
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as available
	available := resp.StatusCode >= 200 && resp.StatusCode < 300
	errorMsg := ""
	if !available {
		errorMsg = resp.Status
	}

	rm.updateResourceStatus(resource, available, errorMsg)
}

// updateResourceStatus updates the status of a resource
func (rm *ResourceManager) updateResourceStatus(resource *ResourceStatus, available bool, errorMsg string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	previousAvailability := resource.Available

	resource.Available = available
	resource.LastCheck = time.Now()
	resource.LastError = errorMsg
	resource.CheckCount++

	if available {
		resource.SuccessCount++
		resource.LastError = "" // Clear error on success
	}

	// Log availability changes
	if previousAvailability != available {
		if available {
			log.Printf("Resource %s is now AVAILABLE", resource.Name)
			rm.handleResourceAvailable(resource.Name)
		} else {
			log.Printf("Resource %s is now UNAVAILABLE: %s", resource.Name, errorMsg)
			rm.handleResourceUnavailable(resource.Name)
		}
	}
}

// handleResourceAvailable handles when a resource becomes available
func (rm *ResourceManager) handleResourceAvailable(resourceName string) {
	switch resourceName {
	case "minio":
		log.Println("MinIO is available - file storage operations enabled")
		// Could initialize MinIO client here
	case "redis":
		log.Println("Redis is available - caching enabled")
		// Could initialize Redis client here
	case "ollama":
		log.Println("Ollama is available - AI features enabled")
		// Could test model availability here
	case "qdrant":
		log.Println("Qdrant is available - vector search enabled")
		// Could initialize Qdrant client here
	}
}

// handleResourceUnavailable handles when a resource becomes unavailable
func (rm *ResourceManager) handleResourceUnavailable(resourceName string) {
	switch resourceName {
	case "minio":
		log.Println("MinIO unavailable - falling back to local storage")
	case "redis":
		log.Println("Redis unavailable - running without cache")
	case "ollama":
		log.Println("Ollama unavailable - AI features disabled")
	case "qdrant":
		log.Println("Qdrant unavailable - vector search disabled")
	}
}

// logAvailabilityChanges logs current resource availability
func (rm *ResourceManager) logAvailabilityChanges() {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	availableCount := 0
	totalCount := len(rm.resources)

	for _, resource := range rm.resources {
		if resource.Available {
			availableCount++
		}
	}

	if totalCount > 0 {
		log.Printf("Resource status: %d/%d available", availableCount, totalCount)
	}
}

// GetResourceStatus returns the current status of all resources
func (rm *ResourceManager) GetResourceStatus() map[string]*ResourceStatus {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Create a copy to avoid race conditions
	status := make(map[string]*ResourceStatus)
	for name, resource := range rm.resources {
		status[name] = &ResourceStatus{
			Name:         resource.Name,
			URL:          resource.URL,
			Available:    resource.Available,
			LastCheck:    resource.LastCheck,
			LastError:    resource.LastError,
			CheckCount:   resource.CheckCount,
			SuccessCount: resource.SuccessCount,
		}
	}

	return status
}

// IsResourceAvailable checks if a specific resource is available
func (rm *ResourceManager) IsResourceAvailable(resourceName string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if resource, exists := rm.resources[resourceName]; exists {
		return resource.Available
	}
	return false
}

// GetResourceHealth returns health information for the health endpoint
func (rm *ResourceManager) GetResourceHealth() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	health := make(map[string]interface{})

	for name, resource := range rm.resources {
		if resource.Available {
			health[name] = "connected"
		} else {
			health[name] = map[string]interface{}{
				"status":     "disconnected",
				"last_error": resource.LastError,
				"last_check": resource.LastCheck.Unix(),
			}
		}
	}

	return health
}

// WaitForResource waits for a specific resource to become available
func (rm *ResourceManager) WaitForResource(resourceName string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if rm.IsResourceAvailable(resourceName) {
			return true
		}
		time.Sleep(1 * time.Second)
	}

	return false
}

// TryWithResource attempts to execute a function when a resource is available
func (rm *ResourceManager) TryWithResource(resourceName string, fn func() error) error {
	if !rm.IsResourceAvailable(resourceName) {
		return ResourceUnavailableError{Resource: resourceName}
	}

	return fn()
}

// checkRedis checks if Redis is available by attempting a connection
func (rm *ResourceManager) checkRedis(resource *ResourceStatus) {
	// For now, just assume Redis is available if the port is accessible
	// In a production system, we would use a proper Redis client
	// TODO: Implement proper Redis health check with go-redis client
	
	// Simple TCP connection check to Redis port (Vrooli uses 6380)
	conn, err := net.DialTimeout("tcp", "localhost:6380", resourceTimeout)
	if err != nil {
		rm.updateResourceStatus(resource, false, err.Error())
		return
	}
	conn.Close()
	rm.updateResourceStatus(resource, true, "")
}

// ResourceUnavailableError represents an error when a resource is unavailable
type ResourceUnavailableError struct {
	Resource string
}

func (e ResourceUnavailableError) Error() string {
	return "resource " + e.Resource + " is not available"
}

// GetResourceMetrics returns metrics about resource availability
func (rm *ResourceManager) GetResourceMetrics() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	metrics := make(map[string]interface{})

	for name, resource := range rm.resources {
		var availability float64
		if resource.CheckCount > 0 {
			availability = float64(resource.SuccessCount) / float64(resource.CheckCount)
		}

		metrics[name] = map[string]interface{}{
			"availability_percent": availability * 100,
			"total_checks":         resource.CheckCount,
			"successful_checks":    resource.SuccessCount,
			"currently_available":  resource.Available,
			"last_check":          resource.LastCheck.Unix(),
		}
	}

	return metrics
}