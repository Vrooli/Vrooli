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
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// MonitorService handles system monitoring operations
type MonitorService struct {
	config         *config.Config
	repo           repository.MetricsRepository
	procRepo       repository.ProcessSampleRepository
	collectors     *collectors.CollectorRegistry
	infra          infrastructure.Provider
	clock          Clock
	active         bool
	metricInterval time.Duration // live baseline collection interval (mu-protected)
	lastRun        map[string]time.Time
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc

	// Per-process sampling seams (3b/3c/3d). sampler walks /proc once per cycle;
	// attributor maps each pid to its owning scenario; both are nil-safe so a
	// service constructed without them simply skips process sampling.
	snapshots         collectors.SnapshotProvider
	sampler           procsampler.Sampler
	attributor        *procsampler.Attributor
	procSampleEvery   time.Duration
	procSampleTopN    int
	lastProcSampledAt time.Time

	lastCycleDuration       time.Duration
	lastCycleForks          uint64
	lastCollectorDurations  map[string]time.Duration
	lastCollectorForks      map[string]uint64
	lastProcSampleDuration  time.Duration
	lastCommandForkCount    uint64
	lastSelfMetricsRecorded time.Time
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

// WithProcessSampling injects the per-process sampling seams (tests/wiring).
func WithProcessSampling(repo repository.ProcessSampleRepository, sampler procsampler.Sampler, attributor *procsampler.Attributor) MonitorOption {
	return func(s *MonitorService) {
		s.procRepo = repo
		s.sampler = sampler
		s.attributor = attributor
	}
}

// NewMonitorService creates a new monitor service
func NewMonitorService(cfg *config.Config, repo repository.MetricsRepository, infra infrastructure.Provider, opts ...MonitorOption) *MonitorService {
	ctx, cancel := context.WithCancel(context.Background())

	baseInterval := cfg.Monitoring.MetricsInterval
	if baseInterval <= 0 {
		baseInterval = 20 * time.Second
	}

	procEvery := cfg.Monitoring.ProcSampleInterval
	if procEvery <= 0 {
		procEvery = 20 * time.Second
	}
	procTopN := cfg.Monitoring.ProcSampleTopN
	if procTopN <= 0 {
		procTopN = 50
	}

	svc := &MonitorService{
		config:                 cfg,
		repo:                   repo,
		collectors:             collectors.NewCollectorRegistry(),
		infra:                  infra,
		clock:                  RealClock{},
		active:                 true,
		metricInterval:         baseInterval,
		lastRun:                make(map[string]time.Time),
		lastCollectorDurations: make(map[string]time.Duration),
		lastCollectorForks:     make(map[string]uint64),
		ctx:                    ctx,
		cancel:                 cancel,
		snapshots:              collectors.NewCachedSnapshotProvider(0),
		procSampleEvery:        procEvery,
		procSampleTopN:         procTopN,
	}

	// The production repo implements ProcessSampleRepository too; capture it so
	// the sampler can persist. Tests that inject a metrics-only repo leave this
	// nil and process sampling is skipped.
	if pr, ok := repo.(repository.ProcessSampleRepository); ok {
		svc.procRepo = pr
	}
	svc.sampler = procsampler.NewCachedSampler(procsampler.NewSampler(), time.Second)
	collectors.SetTopProcessSampler(svc.sampler)
	// Primary path is the bare-host /proc heuristic; the docker fallback only
	// resolves genuinely containerized pids (none on a bare-host deployment).
	svc.attributor = procsampler.NewAttributor(NewDockerFallback())

	for _, opt := range opts {
		opt(svc)
	}

	// Register default collectors only if none were injected via options
	if len(svc.collectors.GetAll()) == 0 {
		svc.registerCollectors()
	}

	return svc
}

// registerCollectors registers all metric collectors, sharing one cached host
// snapshot provider across the cpu/memory/gpu collectors so a cycle probes the
// host once instead of three times.
func (s *MonitorService) registerCollectors() {
	cpu := collectors.NewCPUCollector()
	cpu.SetSnapshotProvider(s.snapshots)
	mem := collectors.NewMemoryCollector()
	mem.SetSnapshotProvider(s.snapshots)

	s.collectors.Register(cpu)
	s.collectors.Register(mem)
	s.collectors.Register(collectors.NewNetworkCollector())
	s.collectors.Register(collectors.NewDiskCollector())
	s.collectors.Register(collectors.NewProcessCollector())
	if gpuCollector := collectors.NewGPUCollector(); gpuCollector.IsEnabled() {
		gpuCollector.SetSnapshotProvider(s.snapshots)
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
		minInterval = 20 * time.Second
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
		interval = 20 * time.Second
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
	cycleStarted := time.Now()
	cycleForksBefore := collectors.CommandForkCount()
	var metricsData []*collectors.MetricData
	var errors []error

	for _, collector := range s.collectors.GetEnabled() {
		name := collector.GetName()
		interval := collector.GetInterval()
		if !s.shouldCollect(name, interval, now) {
			continue
		}

		started := time.Now()
		forksBefore := collectors.CommandForkCount()
		data, err := collector.Collect(ctx)
		s.recordCollectorSelfMetrics(name, time.Since(started), collectors.CommandForkCount()-forksBefore)
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

	// Per-process attribution sampling runs on its own (typically slower)
	// cadence; gate it so it doesn't fire on every fast metrics tick.
	if s.shouldSampleProcesses(now) {
		s.sampleProcesses(ctx, now)
	}
	s.recordCycleSelfMetrics(time.Since(cycleStarted), collectors.CommandForkCount()-cycleForksBefore, now)
}

// shouldSampleProcesses gates the /proc sampler to its configured interval.
func (s *MonitorService) shouldSampleProcesses(now time.Time) bool {
	if s.sampler == nil || s.procRepo == nil {
		return false
	}
	s.mu.RLock()
	last := s.lastProcSampledAt
	s.mu.RUnlock()
	if !last.IsZero() && now.Sub(last) < s.procSampleEvery {
		return false
	}
	return true
}

// sampleProcesses walks /proc once, attributes each pid to its owning scenario,
// caps to the configured top-N (logging what was dropped — no silent caps), and
// persists the cycle to the process_samples table.
func (s *MonitorService) sampleProcesses(ctx context.Context, now time.Time) {
	s.mu.Lock()
	s.lastProcSampledAt = now
	s.mu.Unlock()

	started := time.Now()
	samples, err := s.sampler.Sample(ctx)
	s.recordProcSampleDuration(time.Since(started))
	if err != nil {
		if err == procsampler.ErrUnsupported {
			return // non-Linux: aggregate collectors still work; nothing to persist
		}
		log.Printf("process sampler: %v", err)
		return
	}
	if len(samples) == 0 {
		return
	}

	if s.attributor != nil {
		s.attributor.Attribute(ctx, samples)
	}

	// samples arrive sorted by descending CPU%; cap to top-N and log the drop.
	dropped := 0
	if s.procSampleTopN > 0 && len(samples) > s.procSampleTopN {
		dropped = len(samples) - s.procSampleTopN
		samples = samples[:s.procSampleTopN]
	}
	if dropped > 0 {
		log.Printf("process sampler: capped to top %d by CPU, dropped %d lower-usage processes", s.procSampleTopN, dropped)
	}

	rows := make([]repository.ProcessSample, 0, len(samples))
	for _, ps := range samples {
		rows = append(rows, repository.ProcessSample{
			Timestamp: now.UTC(),
			PID:       ps.PID,
			PPID:      ps.PPID,
			Comm:      ps.Comm,
			Cmdline:   ps.Cmdline,
			Cwd:       ps.Cwd,
			Owner:     ps.Owner,
			CPUPct:    ps.CPUPct,
			RSSKB:     ps.RSSKB,
			Threads:   ps.Threads,
		})
	}
	if err := s.procRepo.SaveProcessSamples(ctx, rows); err != nil {
		log.Printf("process sampler: persist %d rows: %v", len(rows), err)
	}
}

func (s *MonitorService) recordCollectorSelfMetrics(name string, duration time.Duration, forks uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCollectorDurations[name] = duration
	s.lastCollectorForks[name] = forks
}

func (s *MonitorService) recordProcSampleDuration(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastProcSampleDuration = duration
}

func (s *MonitorService) recordCycleSelfMetrics(duration time.Duration, forks uint64, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCycleDuration = duration
	s.lastCycleForks = forks
	s.lastCommandForkCount = collectors.CommandForkCount()
	s.lastSelfMetricsRecorded = now
}

// SelfMetrics returns lightweight monitor overhead telemetry for health/status
// payloads. It intentionally reports only the latest completed cycle.
func (s *MonitorService) SelfMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	collectorDurations := make(map[string]float64, len(s.lastCollectorDurations))
	for name, duration := range s.lastCollectorDurations {
		collectorDurations[name] = float64(duration.Microseconds()) / 1000
	}
	collectorForks := make(map[string]uint64, len(s.lastCollectorForks))
	for name, forks := range s.lastCollectorForks {
		collectorForks[name] = forks
	}

	return map[string]interface{}{
		"last_cycle_duration_ms":       float64(s.lastCycleDuration.Microseconds()) / 1000,
		"last_cycle_forks":             s.lastCycleForks,
		"total_collector_forks":        s.lastCommandForkCount,
		"collector_duration_ms":        collectorDurations,
		"collector_forks":              collectorForks,
		"last_proc_sample_duration_ms": float64(s.lastProcSampleDuration.Microseconds()) / 1000,
		"recorded_at":                  s.lastSelfMetricsRecorded.UTC().Format(time.RFC3339),
	}
}

// GetProcessTimeline returns ranked process consumers over the window, grouped
// by owner/scenario. It is the standing replacement for the manual `ps`/`top`
// "top consumers by scenario" forensic.
func (s *MonitorService) GetProcessTimeline(ctx context.Context, window time.Duration, owner string, top int) ([]repository.ProcessTimelineEntry, error) {
	if s.procRepo == nil {
		return nil, nil
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	now := s.clock.Now()
	return s.procRepo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: now.Add(-window),
		// Pad the end slightly so a sample written at exactly "now" (common in
		// deterministic tests and same-tick reads) falls inside the half-open
		// [Start, End) window the repositories use.
		End:   now.Add(time.Second),
		Owner: owner,
		Top:   top,
	})
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

	populateCPUDetails(detailed, cpuData, topCPUProcs)
	populateMemoryDetails(detailed, memData, diskData, topMemProcs)
	populateNetworkDetails(detailed, netData)
	populateSystemDetails(detailed, diskData)
	populateGPUDetails(detailed, gpuData)

	detailed.SystemDetails.ServiceDependencies = s.infra.CheckServiceDependencies()

	return detailed, nil
}

// GetDiskDetail returns read-only disk usage detail plus cleanup-manager
// remediation guidance. It observes pressure and attribution only; cleanup
// execution remains owned by cleanup-manager policy/audit.
func (s *MonitorService) GetDiskDetail(_ context.Context) (*models.DiskDetailResponse, error) {
	partitions, err := collectors.GetDiskPartitions()
	if err != nil {
		return nil, err
	}

	detail := &models.DiskDetailResponse{
		Partitions:  diskPartitionsFromMaps(partitions),
		ActiveMount: "/",
		Depth:       2,
		Timestamp:   s.clock.Now(),
		Notes: []string{
			"Disk detail is observational only; broad cleanup is delegated to cleanup-manager plan/apply.",
			"Suggested handoff: cleanup-manager cleanup plan --profile conservative",
		},
	}
	if highestDiskPressure(detail.Partitions) >= 85 {
		detail.Notes = append(detail.Notes, "High disk pressure detected; request a cleanup-manager preview before applying remediation.")
	}

	topDirs, dirErr := collectors.GetLargestDirectories(detail.ActiveMount, detail.Depth, 8)
	if dirErr != nil {
		detail.Notes = append(detail.Notes, "Directory attribution unavailable: "+dirErr.Error())
	} else {
		detail.TopDirectories = topDirs
	}
	largestFiles, fileErr := collectors.GetLargestFiles(detail.ActiveMount, 8)
	if fileErr != nil {
		detail.Notes = append(detail.Notes, "Largest-file attribution unavailable: "+fileErr.Error())
	} else {
		detail.LargestFiles = largestFiles
	}

	return detail, nil
}

func diskPartitionsFromMaps(rows []map[string]interface{}) []models.DiskPartitionInfo {
	partitions := make([]models.DiskPartitionInfo, 0, len(rows))
	for _, row := range rows {
		partitions = append(partitions, models.DiskPartitionInfo{
			Device:         getStringValue(row, "device"),
			MountPoint:     getStringValue(row, "mount_point"),
			SizeBytes:      getInt64Value(row, "size_bytes"),
			SizeHuman:      getStringValue(row, "size_human"),
			UsedBytes:      getInt64Value(row, "used_bytes"),
			UsedHuman:      getStringValue(row, "used_human"),
			AvailableBytes: getInt64Value(row, "available_bytes"),
			AvailableHuman: getStringValue(row, "available_human"),
			UsePercent:     getFloat64Value(row, "use_percent"),
		})
	}
	return partitions
}

func highestDiskPressure(partitions []models.DiskPartitionInfo) float64 {
	var highest float64
	for _, partition := range partitions {
		if partition.UsePercent > highest {
			highest = partition.UsePercent
		}
	}
	return highest
}

// populateCPUDetails fills the CPU section of detailed from the cpu collector data.
func populateCPUDetails(detailed *models.DetailedMetrics, cpuData *collectors.MetricData, topCPUProcs []map[string]interface{}) {
	if cpuData == nil {
		return
	}

	detailed.CPUDetails = models.CPUMetrics{
		Usage:           getFloat64Value(cpuData.Values, "usage_percent"),
		LoadAverage:     getFloat64Slice(cpuData.Values, "load_average"),
		ContextSwitches: getInt64Value(cpuData.Values, "context_switches"),
		Goroutines:      getIntValue(cpuData.Values, "goroutines"),
	}

	for _, proc := range topCPUProcs {
		detailed.CPUDetails.TopProcesses = append(detailed.CPUDetails.TopProcesses, convertToProcessInfo(proc))
	}
}

// populateMemoryDetails fills the memory section of detailed (including disk
// usage, which is stored under MemoryMetrics in proto).
func populateMemoryDetails(detailed *models.DetailedMetrics, memData, diskData *collectors.MetricData, topMemProcs []map[string]interface{}) {
	if memData == nil {
		return
	}

	detailed.MemoryDetails = models.MemoryMetrics{
		Usage: getFloat64Value(memData.Values, "usage_percent"),
	}

	if swapInfo, ok := memData.Values["swap"].(map[string]interface{}); ok {
		detailed.MemoryDetails.SwapUsage = models.SwapInfo{
			Used:    getInt64Value(swapInfo, "used"),
			Total:   getInt64Value(swapInfo, "total"),
			Percent: getFloat64Value(swapInfo, "percent"),
		}
	}

	for _, proc := range topMemProcs {
		detailed.MemoryDetails.TopProcesses = append(detailed.MemoryDetails.TopProcesses, convertToProcessInfo(proc))
	}

	detailed.MemoryDetails.GrowthPatterns = nil

	if diskData == nil {
		return
	}
	if diskUsage, ok := diskData.Values["usage"].(map[string]interface{}); ok {
		detailed.MemoryDetails.DiskUsage = models.DiskInfo{
			Used:    getInt64Value(diskUsage, "used"),
			Total:   getInt64Value(diskUsage, "total"),
			Percent: getFloat64Value(diskUsage, "percent"),
		}
	}
}

// populateNetworkDetails fills the network section of detailed from the network collector data.
func populateNetworkDetails(detailed *models.DetailedMetrics, netData *collectors.MetricData) {
	if netData == nil {
		return
	}

	if states, ok := netData.Values["tcp_states"].(map[string]int); ok {
		detailed.NetworkDetails.TCPStates = models.TCPConnectionStates{
			Established: states["established"],
			TimeWait:    states["time_wait"],
			CloseWait:   states["close_wait"],
			Listen:      states["listen"],
			Total:       states["total"],
		}
	}

	if portUsage, ok := netData.Values["port_usage"].(map[string]int); ok {
		detailed.NetworkDetails.PortUsage = models.PortUsageInfo{
			Used:  portUsage["used"],
			Total: portUsage["total"],
		}
	}

	if bw, ok := netData.Values["bandwidth"].(map[string]interface{}); ok {
		detailed.NetworkDetails.NetworkStats = models.NetworkStatistics{
			BandwidthInMbps:  getFloat64Value(bw, "in_mbps"),
			BandwidthOutMbps: getFloat64Value(bw, "out_mbps"),
		}
	}
}

// populateSystemDetails fills the system section of detailed (file descriptors
// and inotify watchers) from the disk collector data.
func populateSystemDetails(detailed *models.DetailedMetrics, diskData *collectors.MetricData) {
	if diskData == nil {
		return
	}

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

// populateGPUDetails fills the GPU section of detailed from the gpu collector data.
func populateGPUDetails(detailed *models.DetailedMetrics, gpuData *collectors.MetricData) {
	if gpuData == nil {
		return
	}

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
