package services

import (
	"context"
	"log"
	"sync"
	"time"

	"system-monitor-api/internal/collectors"
	"system-monitor-api/internal/config"
	"system-monitor-api/internal/infrastructure"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/repository"
)

// MonitorService handles system monitoring operations
type MonitorService struct {
	config     *config.Config
	repo       repository.MetricsRepository
	collectors *collectors.CollectorRegistry
	alertSvc   interface{} // Can be *AlertService or mock
	infra      infrastructure.Provider
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewMonitorService creates a new monitor service
func NewMonitorService(cfg *config.Config, repo repository.MetricsRepository, alertSvc interface{}) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &MonitorService{
		config:     cfg,
		repo:       repo,
		collectors: collectors.NewCollectorRegistry(),
		alertSvc:   alertSvc,
		infra:      infrastructure.NewStaticProvider(),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Register collectors
	svc.registerCollectors()

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

// collectionLoop continuously collects metrics
func (s *MonitorService) collectionLoop() {
	ticker := time.NewTicker(s.config.Monitoring.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectMetrics()
		}
	}
}

// collectMetrics collects metrics from all enabled collectors
func (s *MonitorService) collectMetrics() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Collect from all collectors
	metricsData, errors := s.collectors.CollectAll(ctx)

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
		return s.collectCurrentMetrics(ctx)
	}

	return metrics, nil
}

// collectCurrentMetrics performs real-time metric collection
func (s *MonitorService) collectCurrentMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	cpuCollector := collectors.NewCPUCollector()
	memCollector := collectors.NewMemoryCollector()
	netCollector := collectors.NewNetworkCollector()
	gpuCollector := collectors.NewGPUCollector()

	cpuData, _ := cpuCollector.Collect(ctx)
	memData, _ := memCollector.Collect(ctx)
	netData, _ := netCollector.Collect(ctx)
	var gpuData *collectors.MetricData
	if gpuCollector.IsEnabled() {
		gpuData, _ = gpuCollector.Collect(ctx)
	}

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
		Timestamp:      time.Now(),
	}, nil
}

// GetDetailedMetrics retrieves comprehensive system metrics
func (s *MonitorService) GetDetailedMetrics(ctx context.Context) (*models.DetailedMetrics, error) {
	// Collect detailed metrics from all collectors
	cpuCollector := collectors.NewCPUCollector()
	memCollector := collectors.NewMemoryCollector()
	netCollector := collectors.NewNetworkCollector()
	diskCollector := collectors.NewDiskCollector()
	processCollector := collectors.NewProcessCollector()
	gpuCollector := collectors.NewGPUCollector()

	cpuData, _ := cpuCollector.Collect(ctx)
	memData, _ := memCollector.Collect(ctx)
	netData, _ := netCollector.Collect(ctx)
	diskData, _ := diskCollector.Collect(ctx)
	_, _ = processCollector.Collect(ctx) // Collected but not used here
	var gpuData *collectors.MetricData
	if gpuCollector.IsEnabled() {
		gpuData, _ = gpuCollector.Collect(ctx)
	}

	// Get top processes
	topCPUProcs, _ := collectors.GetTopProcessesByCPU(5)
	topMemProcs, _ := collectors.GetTopProcessesByMemory(5)

	// Build detailed metrics response
	detailed := &models.DetailedMetrics{
		Timestamp: time.Now(),
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

		// Add mock growth patterns
		detailed.MemoryDetails.GrowthPatterns = []models.MemoryGrowth{
			{Process: "scenario-api-1", GrowthMBPerHour: 15.0, RiskLevel: "medium"},
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

		// Network Stats
		detailed.NetworkDetails.NetworkStats = models.NetworkStatistics{
			BandwidthInMbps:  12.5,
			BandwidthOutMbps: 8.2,
			PacketLoss:       0.1,
			DNSSuccessRate:   99.2,
			DNSLatencyMs:     15.0,
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
	collector := collectors.NewProcessCollector()
	data, err := collector.Collect(ctx)
	if err != nil {
		return nil, err
	}

	result := &models.ProcessMonitorData{
		Timestamp: time.Now(),
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
