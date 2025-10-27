package services

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"app-monitor-api/logger"
)

// SystemMetrics represents system-wide metrics
type SystemMetrics struct {
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
	Disk      float64   `json:"disk"`
	Network   float64   `json:"network"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricsService handles system metrics collection with caching
type MetricsService struct {
	cache       *SystemMetrics
	cacheMutex  sync.RWMutex
	cacheExpiry time.Time
	cacheTTL    time.Duration
}

// NewMetricsService creates a new metrics service with caching
func NewMetricsService() *MetricsService {
	return &MetricsService{
		cacheTTL: 5 * time.Second, // Cache metrics for 5 seconds
	}
}

// GetSystemMetrics retrieves current system metrics with caching
func (s *MetricsService) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	// Check if cache is valid
	s.cacheMutex.RLock()
	if s.cache != nil && time.Now().Before(s.cacheExpiry) {
		cachedMetrics := *s.cache
		s.cacheMutex.RUnlock()
		return &cachedMetrics, nil
	}
	s.cacheMutex.RUnlock()

	// Collect fresh metrics
	metrics, err := s.collectSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}

	// Update cache
	s.cacheMutex.Lock()
	s.cache = metrics
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.cacheMutex.Unlock()

	return metrics, nil
}

// collectSystemMetrics gathers system metrics from various sources
func (s *MetricsService) collectSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect metrics in parallel for better performance
	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var collectionErrors []error

	// CPU Usage
	wg.Add(1)
	go func() {
		defer wg.Done()
		cpu, err := s.getCPUUsage(ctx)
		if err != nil {
			errMutex.Lock()
			collectionErrors = append(collectionErrors, fmt.Errorf("cpu: %w", err))
			errMutex.Unlock()
		} else {
			metrics.CPU = cpu
		}
	}()

	// Memory Usage
	wg.Add(1)
	go func() {
		defer wg.Done()
		memory, err := s.getMemoryUsage(ctx)
		if err != nil {
			errMutex.Lock()
			collectionErrors = append(collectionErrors, fmt.Errorf("memory: %w", err))
			errMutex.Unlock()
		} else {
			metrics.Memory = memory
		}
	}()

	// Disk Usage
	wg.Add(1)
	go func() {
		defer wg.Done()
		disk, err := s.getDiskUsage(ctx)
		if err != nil {
			errMutex.Lock()
			collectionErrors = append(collectionErrors, fmt.Errorf("disk: %w", err))
			errMutex.Unlock()
		} else {
			metrics.Disk = disk
		}
	}()

	// Network Usage
	wg.Add(1)
	go func() {
		defer wg.Done()
		network, err := s.getNetworkUsage(ctx)
		if err != nil {
			errMutex.Lock()
			collectionErrors = append(collectionErrors, fmt.Errorf("network: %w", err))
			errMutex.Unlock()
		} else {
			metrics.Network = network
		}
	}()

	wg.Wait()

	// Return partial results even if some metrics failed
	if len(collectionErrors) > 0 {
		logger.Warn(fmt.Sprintf("some metrics collection failed: %v", collectionErrors))
	}

	return metrics, nil
}

// getCPUUsage retrieves current CPU usage percentage
func (s *MetricsService) getCPUUsage(ctx context.Context) (float64, error) {
	// Add timeout to prevent hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try using /proc/stat for more reliable CPU calculation
	cmd := exec.CommandContext(ctxWithTimeout, "bash", "-c", `
		grep 'cpu ' /proc/stat | head -1 | awk '{
			idle=$5; 
			total=0; 
			for(i=2;i<=NF;i++){total+=$i}; 
			print 100*(1-idle/total)
		}'
	`)

	output, err := cmd.Output()
	if err != nil {
		// Fallback to top command (reuse timeout context)
		cmd = exec.CommandContext(ctxWithTimeout, "bash", "-c", "top -b -n1 | grep '%Cpu' | awk '{print 100-$8}'")
		output, err = cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("failed to get CPU usage: %w", err)
		}
	}

	cpuStr := strings.TrimSpace(string(output))
	cpuStr = strings.TrimSuffix(cpuStr, "%")
	cpu, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU usage: %w", err)
	}

	// Ensure value is within valid range
	if cpu < 0 {
		cpu = 0
	} else if cpu > 100 {
		cpu = 100
	}

	return cpu, nil
}

// getMemoryUsage retrieves current memory usage percentage
func (s *MetricsService) getMemoryUsage(ctx context.Context) (float64, error) {
	// Add timeout to prevent hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Use /proc/meminfo for more accurate memory calculation
	cmd := exec.CommandContext(ctxWithTimeout, "bash", "-c", `
		awk '/MemTotal:/ {total=$2} /MemAvailable:/ {available=$2} END {
			if (total > 0) printf "%.1f", 100*(1-available/total)
		}' /proc/meminfo
	`)

	output, err := cmd.Output()
	if err != nil {
		// Fallback to free command (reuse timeout context)
		cmd = exec.CommandContext(ctxWithTimeout, "bash", "-c", "free -m | awk 'NR==2{printf \"%.1f\", $3*100/$2}'")
		output, err = cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("failed to get memory usage: %w", err)
		}
	}

	memStr := strings.TrimSpace(string(output))
	mem, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory usage: %w", err)
	}

	// Ensure value is within valid range
	if mem < 0 {
		mem = 0
	} else if mem > 100 {
		mem = 100
	}

	return mem, nil
}

// getDiskUsage retrieves current disk usage percentage for root filesystem
func (s *MetricsService) getDiskUsage(ctx context.Context) (float64, error) {
	// Add timeout to prevent hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "bash", "-c", "df / | awk 'NR==2 {print $5}' | sed 's/%//'")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get disk usage: %w", err)
	}

	diskStr := strings.TrimSpace(string(output))
	disk, err := strconv.ParseFloat(diskStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse disk usage: %w", err)
	}

	// Ensure value is within valid range
	if disk < 0 {
		disk = 0
	} else if disk > 100 {
		disk = 100
	}

	return disk, nil
}

// getNetworkUsage retrieves current network I/O rate in KB/s
func (s *MetricsService) getNetworkUsage(ctx context.Context) (float64, error) {
	// Add timeout to prevent hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Try to get network stats from /proc/net/dev
	cmd := exec.CommandContext(ctxWithTimeout, "bash", "-c", `
		awk '/:/ {
			rx+=$2; tx+=$10
		} END {
			print (rx+tx)/1024/1024
		}' /proc/net/dev
	`)

	output, err := cmd.Output()
	if err != nil {
		// Default to 0 if we can't get network stats
		return 0, nil
	}

	netStr := strings.TrimSpace(string(output))
	net, err := strconv.ParseFloat(netStr, 64)
	if err != nil {
		return 0, nil
	}

	// Cap at reasonable maximum (10 GB/s)
	if net > 10000 {
		net = 10000
	}

	return net, nil
}

// GetMetricsHistory retrieves historical metrics (placeholder for future implementation)
func (s *MetricsService) GetMetricsHistory(ctx context.Context, duration time.Duration) ([]*SystemMetrics, error) {
	// For now, just return current metrics
	// In the future, this could query a time-series database
	current, err := s.GetSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}
	return []*SystemMetrics{current}, nil
}
