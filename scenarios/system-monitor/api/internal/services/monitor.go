package services

// DOC: docs/concepts/ARCHITECTURE.md#monitoring-service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// MonitorService handles system monitoring operations
type MonitorService struct {
	config         *config.Config
	repo           repository.MetricsRepository
	collectors     *collectors.CollectorRegistry
	infra          infrastructure.Provider
	clock          Clock
	active         bool
	metricInterval time.Duration // live baseline collection interval (mu-protected)
	lastRun        map[string]time.Time
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// MonitorOption configures a MonitorService.
type MonitorOption func(*MonitorService)

// WithMonitorClock sets the clock used by the monitor service.
func WithMonitorClock(c Clock) MonitorOption {
	return func(s *MonitorService) { s.clock = c }
}

// WithCollectors injects collectors, skipping the default registerCollectors call.
func WithCollectors(cs ...collectors.Collector) MonitorOption {
	return func(s *MonitorService) {
		for _, c := range cs {
			s.collectors.Register(c)
		}
	}
}

// NewMonitorService creates a new monitor service
func NewMonitorService(cfg *config.Config, repo repository.MetricsRepository, infra infrastructure.Provider, opts ...MonitorOption) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	baseInterval := cfg.Monitoring.MetricsInterval
	if baseInterval <= 0 {
		baseInterval = 10 * time.Second
	}

	svc := &MonitorService{
		config:         cfg,
		repo:           repo,
		collectors:     collectors.NewCollectorRegistry(),
		infra:          infra,
		clock:          RealClock{},
		active:         true,
		metricInterval: baseInterval,
		lastRun:        make(map[string]time.Time),
		ctx:            ctx,
		cancel:         cancel,
	}

	for _, opt := range opts {
		opt(svc)
	}

	// Register default collectors only if none were injected via options
	if len(svc.collectors.GetAll()) == 0 {
		svc.registerCollectors()
	}

	return svc
}

// registerCollectors registers all metric collectors
func (s *MonitorService) registerCollectors() {
	s.collectors.Register(collectors.NewCPUCollector())
	s.collectors.Register(collectors.NewMemoryCollector())
	s.collectors.Register(collectors.NewNetworkCollector())
	s.collectors.Register(collectors.NewDiskCollector())
	s.collectors.Register(collectors.NewProcessCollector())
	if gpuCollector := collectors.NewGPUCollector(); gpuCollector.IsEnabled() {
		s.collectors.Register(gpuCollector)
	}
}

// Start begins the monitoring service
func (s *MonitorService) Start() error {
	log.Println("Starting monitor service...")

	// Start collection loop
	go s.collectionLoop()

	log.Println("Monitor service started")
	return nil
}

// Stop gracefully shuts down the service
func (s *MonitorService) Stop() {
	log.Println("Stopping monitor service...")
	s.cancel()
	log.Println("Monitor service stopped")
}

// SetActive toggles metric collection without shutting down the service.
func (s *MonitorService) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = active
}

// IsActive returns whether metric collection is active.
func (s *MonitorService) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// ApplySettings applies live settings to the collection cadence. The collection
// loop re-reads the interval on each cycle, so changes take effect on the next
// tick without restarting the service. Non-positive intervals are ignored.
func (s *MonitorService) ApplySettings(settings Settings) {
	if settings.MetricCollectionInterval <= 0 {
		return
	}
	s.mu.Lock()
	s.metricInterval = time.Duration(settings.MetricCollectionInterval) * time.Second
	s.mu.Unlock()
}

// EffectiveCollectionInterval returns the current baseline collection interval.
func (s *MonitorService) EffectiveCollectionInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricInterval
}

// collectionLoop continuously collects metrics. The tick interval is recomputed
// each cycle so live settings changes take effect without a restart.
func (s *MonitorService) collectionLoop() {
	for {
		timer := time.NewTimer(s.collectionTickInterval())
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if !s.IsActive() {
				continue
			}
			s.collectMetrics()
		}
	}
}

func (s *MonitorService) collectionTickInterval() time.Duration {
	minInterval := s.EffectiveCollectionInterval()
	if minInterval <= 0 {
		minInterval = 10 * time.Second
	}

	for _, collector := range s.collectors.GetEnabled() {
		interval := collector.GetInterval()
		if interval > 0 && interval < minInterval {
			minInterval = interval
		}
	}

	if minInterval < time.Second {
		minInterval = time.Second
	}

	return minInterval
}

func (s *MonitorService) shouldCollect(name string, interval time.Duration, now time.Time) bool {
	if interval <= 0 {
		interval = s.EffectiveCollectionInterval()
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	s.mu.RLock()
	lastRun, exists := s.lastRun[name]
	s.mu.RUnlock()

	if exists && now.Sub(lastRun) < interval {
		return false
	}
	return true
}

func (s *MonitorService) markCollected(name string, now time.Time) {
	s.mu.Lock()
	s.lastRun[name] = now
	s.mu.Unlock()
}

// collectMetrics collects metrics from all enabled collectors
func (s *MonitorService) collectMetrics() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	now := s.clock.Now()
	var metricsData []*collectors.MetricData
	var errors []error

	for _, collector := range s.collectors.GetEnabled() {
		name := collector.GetName()
		interval := collector.GetInterval()
		if !s.shouldCollect(name, interval, now) {
			continue
		}

		data, err := collector.Collect(ctx)
		s.markCollected(name, now)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		metricsData = append(metricsData, data)
	}

	// Log any collection errors
	for _, err := range errors {
		log.Printf("Metric collection error: %v", err)
	}

	// Store metrics
	for _, data := range metricsData {
		if err := s.repo.SaveMetrics(ctx, data.CollectorName, data.Values); err != nil {
			log.Printf("Failed to store metrics from %s: %v", data.CollectorName, err)
		}
	}
}

// GetCurrentMetrics retrieves the current system metrics
func (s *MonitorService) GetCurrentMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	// Get latest metrics from repository
	metrics, err := s.repo.GetLatestMetrics(ctx)
	if err != nil {
		// Fallback to real-time collection
		return s.GetCurrentMetricsFresh(ctx)
	}

	return metrics, nil
}

// GetCurrentMetricsFresh performs on-demand metric collection using existing collectors.
func (s *MonitorService) GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error) {
	cpuData, _ := s.collectFromRegistry(ctx, "cpu")
	memData, _ := s.collectFromRegistry(ctx, "memory")
	netData, _ := s.collectFromRegistry(ctx, "network")
	gpuData, _ := s.collectFromRegistry(ctx, "gpu")

	cpuUsage := 0.0
	if cpuData != nil {
		if val, ok := cpuData.Values["usage_percent"].(float64); ok {
			cpuUsage = val
		}
	}

	memUsage := 0.0
	if memData != nil {
		if val, ok := memData.Values["usage_percent"].(float64); ok {
			memUsage = val
		}
	}

	tcpConnections := 0
	if netData != nil {
		if val, ok := netData.Values["tcp_connections"].(int); ok {
			tcpConnections = val
		}
	}

	var gpuUsagePtr *float64
	if gpuData != nil {
		if val, ok := gpuData.Values["total_usage_percent"].(float64); ok {
			usage := val
			gpuUsagePtr = &usage
		}
	}

	return &models.MetricsResponse{
		CPUUsage:       cpuUsage,
		MemoryUsage:    memUsage,
		TCPConnections: tcpConnections,
		GPUUsage:       gpuUsagePtr,
		Timestamp:      s.clock.Now(),
	}, nil
}

func (s *MonitorService) collectFromRegistry(ctx context.Context, name string) (*collectors.MetricData, error) {
	collector, ok := s.collectors.Get(name)
	if !ok || !collector.IsEnabled() {
		return nil, nil
	}
	return collector.Collect(ctx)
}

// GetMetricsTimeline retrieves a windowed timeline of metric samples.
func (s *MonitorService) GetMetricsTimeline(ctx context.Context, windowSeconds, sampleIntervalSeconds int) (*models.MetricsTimelineResponse, error) {
	if windowSeconds <= 0 {
		windowSeconds = 120
	}
	if sampleIntervalSeconds <= 0 {
		sampleIntervalSeconds = 5
	}

	now := s.clock.Now()
	start := now.Add(-time.Duration(windowSeconds) * time.Second)

	results, err := s.repo.GetMetrics(ctx, repository.MetricsFilter{
		TimeRange: repository.TimeRange{
			StartTime: start,
			EndTime:   now,
		},
	})
	if err != nil {
		return nil, err
	}

	samples := make([]models.MetricTimelineSample, 0, len(results))
	for _, m := range results {
		samples = append(samples, models.MetricTimelineSample{
			Timestamp:      m.Timestamp,
			CPUUsage:       m.CPUUsage,
			MemoryUsage:    m.MemoryUsage,
			TCPConnections: m.TCPConnections,
			GPUUsage:       m.GPUUsage,
		})
	}

	return &models.MetricsTimelineResponse{
		WindowSeconds:         windowSeconds,
		SampleIntervalSeconds: sampleIntervalSeconds,
		Samples:               samples,
	}, nil
}

// GetDetailedMetrics retrieves comprehensive system metrics
func (s *MonitorService) GetDetailedMetrics(ctx context.Context) (*models.DetailedMetrics, error) {
	// Collect detailed metrics from registered collectors
	cpuData, _ := s.collectFromRegistry(ctx, "cpu")
	memData, _ := s.collectFromRegistry(ctx, "memory")
	netData, _ := s.collectFromRegistry(ctx, "network")
	diskData, _ := s.collectFromRegistry(ctx, "disk")
	gpuData, _ := s.collectFromRegistry(ctx, "gpu")

	// Get top processes
	topCPUProcs, _ := collectors.GetTopProcessesByCPU(5)
	topMemProcs, _ := collectors.GetTopProcessesByMemory(5)

	// Build detailed metrics response
	detailed := &models.DetailedMetrics{
		Timestamp: s.clock.Now(),
	}

	// CPU Details
	if cpuData != nil {
		detailed.CPUDetails = models.CPUMetrics{
			Usage:           getFloat64Value(cpuData.Values, "usage_percent"),
			LoadAverage:     getFloat64Slice(cpuData.Values, "load_average"),
			ContextSwitches: getInt64Value(cpuData.Values, "context_switches"),
			Goroutines:      getIntValue(cpuData.Values, "goroutines"),
		}

		// Convert top processes
		for _, proc := range topCPUProcs {
			detailed.CPUDetails.TopProcesses = append(detailed.CPUDetails.TopProcesses, convertToProcessInfo(proc))
		}
	}

	// Memory Details
	if memData != nil {
		detailed.MemoryDetails = models.MemoryMetrics{
			Usage: getFloat64Value(memData.Values, "usage_percent"),
		}

		// Add swap info
		if swapInfo, ok := memData.Values["swap"].(map[string]interface{}); ok {
			detailed.MemoryDetails.SwapUsage = models.SwapInfo{
				Used:    getInt64Value(swapInfo, "used"),
				Total:   getInt64Value(swapInfo, "total"),
				Percent: getFloat64Value(swapInfo, "percent"),
			}
		}

		// Add top memory processes
		for _, proc := range topMemProcs {
			detailed.MemoryDetails.TopProcesses = append(detailed.MemoryDetails.TopProcesses, convertToProcessInfo(proc))
		}

		detailed.MemoryDetails.GrowthPatterns = nil

		// Add disk usage from disk collector (stored under MemoryMetrics in proto)
		if diskData != nil {
			if diskUsage, ok := diskData.Values["usage"].(map[string]interface{}); ok {
				detailed.MemoryDetails.DiskUsage = models.DiskInfo{
					Used:    getInt64Value(diskUsage, "used"),
					Total:   getInt64Value(diskUsage, "total"),
					Percent: getFloat64Value(diskUsage, "percent"),
				}
			}
		}
	}

	// Network Details
	if netData != nil {
		// TCP States
		if states, ok := netData.Values["tcp_states"].(map[string]int); ok {
			detailed.NetworkDetails.TCPStates = models.TCPConnectionStates{
				Established: states["established"],
				TimeWait:    states["time_wait"],
				CloseWait:   states["close_wait"],
				Listen:      states["listen"],
				Total:       states["total"],
			}
		}

		// Port Usage
		if portUsage, ok := netData.Values["port_usage"].(map[string]int); ok {
			detailed.NetworkDetails.PortUsage = models.PortUsageInfo{
				Used:  portUsage["used"],
				Total: portUsage["total"],
			}
		}

		// Network Stats - populated from collector data when available
		if bw, ok := netData.Values["bandwidth"].(map[string]interface{}); ok {
			detailed.NetworkDetails.NetworkStats = models.NetworkStatistics{
				BandwidthInMbps:  getFloat64Value(bw, "in_mbps"),
				BandwidthOutMbps: getFloat64Value(bw, "out_mbps"),
			}
		}
	}

	// System Details
	if diskData != nil {
		if fdInfo, ok := diskData.Values["file_descriptors"].(map[string]interface{}); ok {
			detailed.SystemDetails.FileDescriptors = models.FileDescriptorInfo{
				Used:    getIntValue(fdInfo, "used"),
				Max:     getIntValue(fdInfo, "max"),
				Percent: getFloat64Value(fdInfo, "percent"),
			}
		}

		if inotifyInfo, ok := diskData.Values["inotify_watchers"].(map[string]interface{}); ok {
			info := models.InotifyWatcherInfo{
				Supported:        getBoolValue(inotifyInfo, "supported"),
				WatchesUsed:      getIntValue(inotifyInfo, "watches_used"),
				WatchesMax:       getIntValue(inotifyInfo, "watches_max"),
				WatchesPercent:   getFloat64Value(inotifyInfo, "watches_percent"),
				InstancesUsed:    getIntValue(inotifyInfo, "instances_used"),
				InstancesMax:     getIntValue(inotifyInfo, "instances_max"),
				InstancesPercent: getFloat64Value(inotifyInfo, "instances_percent"),
			}
			detailed.SystemDetails.InotifyWatchers = &info
		}
	}

	// GPU Details
	if gpuData != nil {
		metrics := models.GPUMetrics{}
		if summary, ok := gpuData.Values["summary"].(models.GPUSummary); ok {
			metrics.Summary = summary
		}
		if devices, ok := gpuData.Values["devices"].([]models.GPUDeviceMetrics); ok {
			metrics.Devices = devices
		}
		if driver, ok := gpuData.Values["driver_version"].(string); ok {
			metrics.Driver = driver
		}
		if model, ok := gpuData.Values["primary_model"].(string); ok {
			metrics.Model = model
		}
		if warnings, ok := gpuData.Values["warnings"].([]string); ok {
			metrics.Errors = warnings
		}
		detailed.GPUDetails = &metrics
	}

	detailed.SystemDetails.ServiceDependencies = s.infra.CheckServiceDependencies()

	return detailed, nil
}

// GetProcessMonitorData retrieves process monitoring information
func (s *MonitorService) GetProcessMonitorData(ctx context.Context) (*models.ProcessMonitorData, error) {
	data, err := s.collectFromRegistry(ctx, "process")
	if err != nil {
		return nil, err
	}

	result := &models.ProcessMonitorData{
		Timestamp: s.clock.Now(),
	}

	if data == nil {
		return result, nil
	}

	// Extract process health information
	if healthData, ok := data.Values["process_health"].(map[string]interface{}); ok {
		result.ProcessHealth = models.ProcessHealthInfo{
			TotalProcesses: getIntValue(healthData, "total_count"),
		}

		// Add zombie processes
		if zombies, ok := data.Values["zombie_processes"].([]map[string]interface{}); ok {
			for _, zombie := range zombies {
				result.ProcessHealth.ZombieProcesses = append(result.ProcessHealth.ZombieProcesses, convertToProcessInfo(zombie))
			}
		}

		// Add high thread count processes
		if highThread, ok := data.Values["high_thread_count"].([]map[string]interface{}); ok {
			for _, proc := range highThread {
				result.ProcessHealth.HighThreadCount = append(result.ProcessHealth.HighThreadCount, convertToProcessInfo(proc))
			}
		}
	}

	// Add top processes as resource matrix
	if topProcs, ok := data.Values["top_by_cpu"].([]map[string]interface{}); ok {
		for _, proc := range topProcs {
			result.ResourceMatrix = append(result.ResourceMatrix, convertToProcessInfo(proc))
		}
	}

	return result, nil
}

// GetInfrastructureMonitorData retrieves infrastructure monitoring data
func (s *MonitorService) GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error) {
	return s.infra.GetInfrastructureMonitorData(ctx)
}

func convertToProcessInfo(proc map[string]interface{}) models.ProcessInfo {
	return models.ProcessInfo{
		PID:        getIntValue(proc, "pid"),
		Name:       getStringValue(proc, "name"),
		CPUPercent: getFloat64Value(proc, "cpu_percent"),
		MemoryMB:   getFloat64Value(proc, "memory_mb"),
		Threads:    getIntValue(proc, "threads"),
		FDs:        getIntValue(proc, "fd_count"),
		Status:     getStringValue(proc, "status"),
	}
}
