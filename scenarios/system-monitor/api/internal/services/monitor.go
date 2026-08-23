package services

// DOC: docs/concepts/ARCHITECTURE.md#monitoring-service

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

var cycleSequence atomic.Uint64

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
	deviceGraphs      collectors.DeviceGraphProvider
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
	lastCycleHeadroomOK     bool
	lastCycleHeadroomReason string
	lastCycleFailures       int
	lastCycleSkipped        int
	lastStaleMetricCount    int
	lastPersistenceDuration time.Duration
	collectionProfile       CollectionProfile
	latestSnapshot          *models.MetricsResponse
	latestCollectorData     map[string]*collectors.MetricData
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
		lastCycleHeadroomOK:    true,
		collectionProfile:      CollectionProfileStandard,
		latestCollectorData:    make(map[string]*collectors.MetricData),
		ctx:                    ctx,
		cancel:                 cancel,
		snapshots:              collectors.NewCachedSnapshotProvider(0),
		deviceGraphs:           collectors.NewCachedDeviceGraphProvider(0),
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
		if snapshot, err := svc.snapshots.Snapshot(context.Background()); err == nil {
			svc.collectionProfile = CollectionProfileForHost(snapshot)
		}
		svc.registerCollectors()
	}

	return svc
}

// registerCollectors registers all metric collectors, sharing one cached host
// snapshot provider across the cpu/memory/gpu collectors so a cycle probes the
// host once instead of three times.
func (s *MonitorService) registerCollectors() {
	cpu := collectors.NewCPUCollector()
	configureCollectorProfile(cpu, s.collectionProfile, "cpu")
	cpu.SetSnapshotProvider(s.snapshots)
	cpu.SetDeviceGraphProvider(s.deviceGraphs)
	mem := collectors.NewMemoryCollector()
	configureCollectorProfile(mem, s.collectionProfile, "memory")
	mem.SetSnapshotProvider(s.snapshots)

	network := collectors.NewNetworkCollector()
	configureCollectorProfile(network, s.collectionProfile, "network")
	disk := collectors.NewDiskCollector()
	configureCollectorProfile(disk, s.collectionProfile, "disk")
	process := collectors.NewProcessCollector()
	configureCollectorProfile(process, s.collectionProfile, "process")
	pressure := collectors.NewPressureCollector()
	configureCollectorProfile(pressure, s.collectionProfile, "pressure")
	deviceGraph := collectors.NewDeviceGraphCollector()
	configureCollectorProfile(deviceGraph, s.collectionProfile, "device_graph")
	deviceGraph.SetDeviceGraphProvider(s.deviceGraphs)
	s.collectors.Register(cpu)
	s.collectors.Register(mem)
	s.collectors.Register(network)
	s.collectors.Register(disk)
	s.collectors.Register(process)
	s.collectors.Register(pressure)
	s.collectors.Register(deviceGraph)
	if gpuCollector := collectors.NewGPUCollector(); gpuCollector.IsEnabled() && s.collectionProfile != CollectionProfileLowPower {
		configureCollectorProfile(gpuCollector, s.collectionProfile, "gpu")
		gpuCollector.SetSnapshotProvider(s.snapshots)
		s.collectors.Register(gpuCollector)
	}
}

func configureCollectorProfile(collector interface {
	SetInterval(time.Duration)
}, profile CollectionProfile, name string,
) {
	if profile != CollectionProfileLowPower {
		return
	}
	intervals := map[string]time.Duration{
		"cpu": 30 * time.Second, "memory": 30 * time.Second,
		"network": 60 * time.Second, "disk": 120 * time.Second,
		"process": 60 * time.Second, "pressure": 30 * time.Second,
		"gpu": 60 * time.Second, "device_graph": 300 * time.Second,
	}
	if interval, ok := intervals[name]; ok {
		collector.SetInterval(interval)
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
// DeviceGraphs exposes the shared device-graph provider so the API can serve
// the graph from the same cached walk the collector uses. Handing out the
// existing provider rather than a fresh one is the point: two providers would
// mean two host walks and two answers about one machine.
func (s *MonitorService) DeviceGraphs() collectors.DeviceGraphProvider { return s.deviceGraphs }

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
	cycleID := fmt.Sprintf("cycle-%d-%d", now.UnixNano(), cycleSequence.Add(1))
	cycleStarted := time.Now()
	cycleForksBefore := collectors.CommandForkCount()
	var metricsData []*collectors.MetricData
	var errors []error
	skipped := 0

	for _, collector := range s.collectors.GetEnabled() {
		name := collector.GetName()
		interval := collector.GetInterval()
		if !s.shouldCollect(name, interval, now) {
			skipped++
			continue
		}

		started := time.Now()
		forksBefore := collectors.CommandForkCount()
		data, err := collector.Collect(ctx)
		s.recordCollectorSelfMetrics(name, time.Since(started), collectors.CommandForkCount()-forksBefore)
		s.markCollected(name, now)
		if err != nil {
			errors = append(errors, err)
			metricsData = append(metricsData, &collectors.MetricData{CollectorName: name, Timestamp: now, Values: map[string]interface{}{"status": "failed", "reason": err.Error()}})
			continue
		}
		metricsData = append(metricsData, data)
	}

	// Log any collection errors
	for _, err := range errors {
		log.Printf("Metric collection error: %v", err)
	}

	// Store metrics
	observations := make([]repository.MetricObservation, 0, len(metricsData))
	for _, data := range metricsData {
		observations = append(observations, repository.MetricObservation{CollectorName: data.CollectorName, Values: data.Values})
	}
	if len(observations) > 0 {
		s.mu.Lock()
		for _, data := range metricsData {
			if data == nil {
				continue
			}
			values := make(map[string]interface{}, len(data.Values))
			for key, value := range data.Values {
				values[key] = value
			}
			s.latestCollectorData[data.CollectorName] = &collectors.MetricData{
				CollectorName: data.CollectorName,
				Timestamp:     data.Timestamp,
				Type:          data.Type,
				Values:        values,
				Tags:          data.Tags,
			}
		}
		s.mu.Unlock()
		s.updateLatestSnapshot(cycleID, now, metricsData)
		persistenceStarted := time.Now()
		if err := s.repo.SaveMetricCycle(ctx, cycleID, now, observations); err != nil {
			log.Printf("Failed to store metric cycle %s: %v", cycleID, err)
		}
		s.mu.Lock()
		s.lastPersistenceDuration = time.Since(persistenceStarted)
		s.mu.Unlock()
	}

	// Per-process attribution sampling runs on its own (typically slower)
	// cadence; gate it so it doesn't fire on every fast metrics tick.
	if s.shouldSampleProcesses(now) {
		s.sampleProcesses(ctx, now, gpuVRAMByPID(metricsData))
	}
	s.recordCycleSelfMetrics(time.Since(cycleStarted), collectors.CommandForkCount()-cycleForksBefore, now)
	s.mu.Lock()
	s.lastCycleFailures = len(errors)
	s.lastCycleSkipped = skipped
	s.mu.Unlock()
}

// updateLatestSnapshot publishes the scheduler-owned view used by on-demand
// reads. Stateful collectors are never invoked from a request path.
func (s *MonitorService) updateLatestSnapshot(cycleID string, observedAt time.Time, data []*collectors.MetricData) {
	// Collectors have independent cadences (for example, CPU is frequent while
	// disk and GPU probes are deliberately slower). A cycle therefore contains
	// only the collectors whose cadence is due. Start from the last published
	// snapshot so a skipped collector keeps its last measured state instead of
	// being silently replaced by a zero-valued response field.
	s.mu.RLock()
	previous := s.latestSnapshot
	response := &models.MetricsResponse{CycleID: cycleID, Timestamp: observedAt}
	if previous != nil {
		copy := *previous
		response = &copy
		response.CycleID = cycleID
		response.Timestamp = observedAt
	}
	s.mu.RUnlock()
	markObserved := func(state *models.MetricState) {
		state.CycleID = cycleID
		state.ObservedAt = observedAt
	}
	for _, metric := range data {
		if metric == nil {
			continue
		}
		switch metric.CollectorName {
		case "cpu":
			response.CPUState = metricState(metric, "usage_percent", "CPU has not been sampled yet")
			markObserved(&response.CPUState)
			if value, ok := metric.Values["usage_percent"].(float64); ok {
				response.CPUUsage = value
			}
		case "memory":
			response.MemoryState = metricState(metric, "usage_percent", "memory collector unavailable")
			markObserved(&response.MemoryState)
			if value, ok := metric.Values["usage_percent"].(float64); ok {
				response.MemoryUsage = value
			}
		case "network":
			response.ConnectionsState = metricState(metric, "tcp_connections", "network collector unavailable")
			markObserved(&response.ConnectionsState)
			if value, ok := metric.Values["tcp_connections"].(int); ok {
				response.TCPConnections = value
			}
		case "gpu":
			response.GPUState = metricState(metric, "total_usage_percent", "GPU collector unavailable")
			markObserved(&response.GPUState)
			if value, ok := metric.Values["total_usage_percent"].(float64); ok {
				response.GPUUsage = &value
			}
		case "disk":
			response.DiskState = diskMetricState(metric)
			markObserved(&response.DiskState)
			if response.DiskState.Status == "measured" {
				response.DiskUsage = response.DiskState.Value
			}
		}
	}
	s.mu.Lock()
	s.latestSnapshot = response
	s.mu.Unlock()
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
func (s *MonitorService) sampleProcesses(ctx context.Context, now time.Time, gpuVRAMArgs ...map[int]float64) {
	var gpuVRAM map[int]float64
	if len(gpuVRAMArgs) > 0 {
		gpuVRAM = gpuVRAMArgs[0]
	}
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

	// Retain independent CPU and RSS leaders. A CPU-only cap loses the exact
	// low-CPU memory hog evidence needed after OOM; the bounded union is at
	// most 2*topN and still comes from this single /proc walk.
	samples, dropped := selectRankSamples(samples, s.procSampleTopN, gpuVRAM)
	if dropped > 0 {
		log.Printf("process sampler: retained CPU/RSS top-%d union (%d rows), dropped %d processes", s.procSampleTopN, len(samples), dropped)
	}

	rows := make([]repository.ProcessSample, 0, len(samples))
	for _, ps := range samples {
		rows = append(rows, repository.ProcessSample{
			Timestamp:            now.UTC(),
			PID:                  ps.PID,
			PPID:                 ps.PPID,
			Comm:                 ps.Comm,
			Cmdline:              ps.Cmdline,
			Cwd:                  ps.Cwd,
			Owner:                ps.Owner,
			CPUPct:               ps.CPUPct,
			CPUSeconds:           ps.CPUSeconds,
			CPUSecondsStatus:     ps.CPUSecondsStatus,
			CPUSecondsReason:     ps.CPUSecondsReason,
			RSSKB:                ps.RSSKB,
			SwapKB:               ps.SwapKB,
			MajorFaultsPerSecond: ps.MajorFaultsPerSecond,
			MetricsStatus:        ps.MetricsStatus,
			MetricsReason:        ps.MetricsReason,
			Threads:              ps.Threads,
			GPUVRAMMB:            gpuVRAM[ps.PID],
		})
	}
	if err := s.procRepo.SaveProcessSamples(ctx, rows); err != nil {
		log.Printf("process sampler: persist %d rows: %v", len(rows), err)
	}
}

func selectDualRankSamples(samples []procsampler.ProcessSample, top int) ([]procsampler.ProcessSample, int) {
	return selectRankSamples(samples, top, nil)
}

func selectRankSamples(samples []procsampler.ProcessSample, top int, gpuVRAM map[int]float64) ([]procsampler.ProcessSample, int) {
	if top <= 0 || len(samples) <= top {
		return samples, 0
	}
	cpu := append([]procsampler.ProcessSample(nil), samples...)
	rss := append([]procsampler.ProcessSample(nil), samples...)
	paging := append([]procsampler.ProcessSample(nil), samples...)
	sort.SliceStable(cpu, func(i, j int) bool { return cpu[i].CPUPct > cpu[j].CPUPct })
	sort.SliceStable(rss, func(i, j int) bool { return rss[i].RSSKB > rss[j].RSSKB })
	sort.SliceStable(paging, func(i, j int) bool {
		if paging[i].MajorFaultsPerSecond != paging[j].MajorFaultsPerSecond {
			return paging[i].MajorFaultsPerSecond > paging[j].MajorFaultsPerSecond
		}
		return paging[i].SwapKB > paging[j].SwapKB
	})
	keep := make(map[int]struct{}, top*2)
	for _, ranked := range [][]procsampler.ProcessSample{cpu, rss} {
		for i := 0; i < top && i < len(ranked); i++ {
			keep[ranked[i].PID] = struct{}{}
		}
	}
	for i := 0; i < top && i < len(paging); i++ {
		if paging[i].MajorFaultsPerSecond > 0 || paging[i].SwapKB > 0 {
			keep[paging[i].PID] = struct{}{}
		}
	}
	if len(gpuVRAM) > 0 {
		gpu := append([]procsampler.ProcessSample(nil), samples...)
		sort.SliceStable(gpu, func(i, j int) bool { return gpuVRAM[gpu[i].PID] > gpuVRAM[gpu[j].PID] })
		for i := 0; i < top && i < len(gpu); i++ {
			if gpuVRAM[gpu[i].PID] > 0 {
				keep[gpu[i].PID] = struct{}{}
			}
		}
	}
	selected := make([]procsampler.ProcessSample, 0, len(keep))
	for _, sample := range samples {
		if _, ok := keep[sample.PID]; ok {
			selected = append(selected, sample)
		}
	}
	return selected, len(samples) - len(selected)
}

func gpuVRAMByPID(metricsData []*collectors.MetricData) map[int]float64 {
	for _, data := range metricsData {
		if data.CollectorName != "gpu" {
			continue
		}
		devices, _ := data.Values["devices"].([]models.GPUDeviceMetrics)
		result := make(map[int]float64)
		for _, device := range devices {
			for _, process := range device.Processes {
				result[process.PID] += process.MemoryUsedMB
			}
		}
		return result
	}
	return nil
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
	s.lastCycleHeadroomOK = true
	s.lastCycleHeadroomReason = ""
	if duration > time.Duration(float64(s.metricInterval)*collectorCycleHeadroom) {
		s.lastCycleHeadroomOK = false
		s.lastCycleHeadroomReason = "collection cycle exceeded 50% of the configured interval"
	}
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
		"collection_profile":           string(s.collectionProfile),
		"last_cycle_duration_ms":       float64(s.lastCycleDuration.Microseconds()) / 1000,
		"last_cycle_forks":             s.lastCycleForks,
		"total_collector_forks":        s.lastCommandForkCount,
		"collector_duration_ms":        collectorDurations,
		"collector_forks":              collectorForks,
		"last_proc_sample_duration_ms": float64(s.lastProcSampleDuration.Microseconds()) / 1000,
		"headroom_ok":                  s.lastCycleHeadroomOK,
		"headroom_reason":              s.lastCycleHeadroomReason,
		"cycle_failures":               s.lastCycleFailures,
		"cycle_skipped":                s.lastCycleSkipped,
		"stale_metric_count":           s.lastStaleMetricCount,
		"last_persistence_duration_ms": float64(s.lastPersistenceDuration.Microseconds()) / 1000,
		"recorded_at":                  s.lastSelfMetricsRecorded.UTC().Format(time.RFC3339),
	}
}

// GetProcessTimeline returns ranked process consumers over the window, grouped
// by owner/scenario. It is the standing replacement for the manual `ps`/`top`
// "top consumers by scenario" forensic.
func (s *MonitorService) GetProcessTimeline(ctx context.Context, window time.Duration, owner string, top int) ([]repository.ProcessTimelineEntry, error) {
	return s.GetProcessTimelineRanked(ctx, window, owner, top, "cpu")
}

// GetProcessTimelineRanked returns bounded scenario attribution ranked by CPU,
// cumulative CPU seconds, RSS, or GPU. Invalid rank values fail closed to CPU.
func (s *MonitorService) GetProcessTimelineRanked(ctx context.Context, window time.Duration, owner string, top int, rank string) ([]repository.ProcessTimelineEntry, error) {
	if s.procRepo == nil {
		return nil, nil
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if rank != "rss" && rank != "gpu" && rank != "cpu_seconds" {
		rank = "cpu"
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
		Rank:  rank,
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
	s.markStale(metrics)
	return metrics, nil
}

func (s *MonitorService) markStale(metrics *models.MetricsResponse) {
	if metrics == nil {
		return
	}
	cutoff := s.clock.Now().Add(-2 * s.EffectiveCollectionInterval())
	stale := 0
	for _, state := range []*models.MetricState{&metrics.CPUState, &metrics.MemoryState, &metrics.ConnectionsState, &metrics.GPUState, &metrics.DiskState} {
		if state.Status == "measured" && !state.ObservedAt.IsZero() && state.ObservedAt.Before(cutoff) {
			state.Status = "stale"
			state.Reason = fmt.Sprintf("last observation is older than %s", 2*s.EffectiveCollectionInterval())
			stale++
		}
	}
	s.mu.Lock()
	s.lastStaleMetricCount = stale
	s.mu.Unlock()
}

// GetCurrentMetricsFresh performs on-demand metric collection using existing collectors.
func (s *MonitorService) GetCurrentMetricsFresh(ctx context.Context) (*models.MetricsResponse, error) {
	_ = ctx // retained for interface compatibility; sampling is scheduler-owned
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.latestSnapshot != nil {
		copy := *s.latestSnapshot
		return &copy, nil
	}
	now := s.clock.Now()
	return &models.MetricsResponse{
		Timestamp:        now,
		CPUState:         notYetSampledState("CPU has not been sampled yet", "percent", now),
		MemoryState:      notYetSampledState("memory has not been sampled yet", "percent", now),
		ConnectionsState: notYetSampledState("network has not been sampled yet", "count", now),
		GPUState:         notYetSampledState("GPU has not been sampled yet", "percent", now),
		DiskState:        notYetSampledState("disk has not been sampled yet", "percent", now),
	}, nil
}

func (s *MonitorService) collectFromRegistry(ctx context.Context, name string) (*collectors.MetricData, error) {
	collector, ok := s.collectors.Get(name)
	if !ok || !collector.IsEnabled() {
		return nil, nil
	}
	return collector.Collect(ctx)
}

// GetPressureSnapshot performs a bounded fresh read of the pressure collector.
// Unsupported/degraded hosts return an explicit unavailable snapshot instead of
// an error so callers can make a safe fail-closed recovery decision.
func (s *MonitorService) GetPressureSnapshot(ctx context.Context) (*models.PressureSnapshot, error) {
	data, err := s.collectFromRegistry(ctx, "pressure")
	if err != nil {
		return nil, err
	}
	snapshot := &models.PressureSnapshot{Timestamp: s.clock.Now(), Memory: map[string]map[string]float64{}}
	if data == nil {
		snapshot.DegradedReason = "pressure collector is disabled"
		return snapshot, nil
	}
	snapshot.Timestamp = data.Timestamp
	if available, ok := data.Values["available"].(bool); ok {
		snapshot.Available = available
	}
	if reason, ok := data.Values["degraded_reason"].(string); ok {
		snapshot.DegradedReason = reason
	}
	if memory, ok := data.Values["memory"].(map[string]map[string]float64); ok {
		snapshot.Memory = memory
	}
	snapshot.OOMKillCount = pressureInt64(data.Values["oom_kill_count"])
	snapshot.OOMCount = pressureInt64(data.Values["oom_count"])
	return snapshot, nil
}

// GetGPUHistory returns the persisted low-cardinality GPU summary timeline.
func (s *MonitorService) GetGPUHistory(ctx context.Context, window time.Duration) (*models.GPUHistory, error) {
	if window <= 0 {
		window = time.Hour
	}
	end := s.clock.Now()
	start := end.Add(-window)
	utilization, err := s.repo.GetHistoricalMetrics(ctx, "total_usage_percent", repository.TimeRange{StartTime: start, EndTime: end})
	if err != nil {
		return nil, err
	}
	vram, err := s.repo.GetHistoricalMetrics(ctx, "used_memory_mb", repository.TimeRange{StartTime: start, EndTime: end})
	if err != nil {
		return nil, err
	}
	return &models.GPUHistory{Start: start, End: end, Utilization: gpuHistoryPoints(utilization), VRAMUsedMB: gpuHistoryPoints(vram)}, nil
}

// GetPressureHistory returns retained PSI and OOM counter evidence over a
// bounded time window. It is read-only and does not collect or scan processes.
func (s *MonitorService) GetPressureHistory(ctx context.Context, window time.Duration) (*models.PressureHistory, error) {
	if window <= 0 {
		window = time.Hour
	}
	end := s.clock.Now()
	start := end.Add(-window)
	rangeFilter := repository.TimeRange{StartTime: start, EndTime: end}
	some, err := s.repo.GetHistoricalMetrics(ctx, "memory_psi_some_avg10", rangeFilter)
	if err != nil {
		return nil, err
	}
	full, err := s.repo.GetHistoricalMetrics(ctx, "memory_psi_full_avg10", rangeFilter)
	if err != nil {
		return nil, err
	}
	oomKills, err := s.repo.GetHistoricalMetrics(ctx, "oom_kill_count", rangeFilter)
	if err != nil {
		return nil, err
	}
	return &models.PressureHistory{Start: start, End: end, SomeAvg10: gpuHistoryPoints(some), FullAvg10: gpuHistoryPoints(full), OOMKillCount: gpuHistoryPoints(oomKills)}, nil
}

func gpuHistoryPoints(points []repository.MetricDataPoint) []models.GPUHistoryPoint {
	out := make([]models.GPUHistoryPoint, 0, len(points))
	for _, point := range points {
		out = append(out, models.GPUHistoryPoint{Timestamp: point.Timestamp, Value: point.Value})
	}
	return out
}

func pressureInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
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
			CycleID:                     m.CycleID,
			Timestamp:                   m.Timestamp,
			CPUUsage:                    m.CPUUsage,
			MemoryUsage:                 m.MemoryUsage,
			TCPConnections:              m.TCPConnections,
			GPUUsage:                    m.GPUUsage,
			SwapUsage:                   m.SwapUsage,
			CPUState:                    m.CPUState,
			CPUContextSwitchesPerSecond: m.CPUContextSwitchesPerSecond,
			CPUInterruptsPerSecond:      m.CPUInterruptsPerSecond,
			CPUNormalizedLoad1:          m.CPUNormalizedLoad1,
			CPUNormalizedLoad5:          m.CPUNormalizedLoad5,
			CPURunQueueDepth:            m.CPURunQueueDepth,
			CPUStallSomeAvg10:           m.CPUStallSomeAvg10,
			CPUStallFullAvg10:           m.CPUStallFullAvg10,
			CPUCoreImbalanceIndex:       m.CPUCoreImbalanceIndex,
			CPUModeIowait:               m.CPUModeIowait,
			CPUModeSteal:                m.CPUModeSteal,
			MemoryState:                 m.MemoryState,
			ConnectionsState:            m.ConnectionsState,
			GPUState:                    m.GPUState,
			SwapState:                   m.SwapState,
			SwapTrafficState:            m.SwapTrafficState,
			MajorFaultsState:            m.MajorFaultsState,
			FragmentationIndexState:     m.FragmentationIndexState,
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
	// Detailed reads are projections of scheduler-owned samples. Calling the
	// stateful CPU/GPU collectors here would advance their delta state between
	// scheduled cycles and produce false zero or unavailable readings.
	latest := s.latestCollectorSnapshot()
	cpuData := latest["cpu"]
	memData := latest["memory"]
	netData := latest["network"]
	diskData := latest["disk"]
	gpuData := latest["gpu"]
	pressureData := latest["pressure"]
	processData := latest["process"]

	// Get top processes
	topCPUProcs, _ := collectors.GetTopProcessesByCPU(5)
	topCPUSecondsProcs, _ := collectors.GetTopProcessesByCPUSeconds(5)
	topMemProcs, _ := collectors.GetTopProcessesByMemory(5)
	topPagingProcs, _ := collectors.GetTopProcessesByPaging(5)

	// Build detailed metrics response
	detailed := &models.DetailedMetrics{
		Timestamp: s.clock.Now(),
	}

	populateCPUDetails(detailed, cpuData, processData, topCPUProcs, topCPUSecondsProcs)
	populateMemoryDetails(detailed, memData, diskData, pressureData, topMemProcs, topPagingProcs)
	if pressureData != nil {
		detailed.CPUDetails.StallSomeAvg10 = pressureState(pressureData, "cpu_psi_some_avg10", "cpu_psi_status", "cpu_psi_reason")
		detailed.CPUDetails.StallFullAvg10 = pressureState(pressureData, "cpu_psi_full_avg10", "cpu_psi_status", "cpu_psi_reason")
	}
	populateNetworkDetails(detailed, netData)
	populateSystemDetails(detailed, diskData)
	populateGPUDetails(detailed, gpuData)

	detailed.SystemDetails.ServiceDependencies = s.infra.CheckServiceDependencies()

	return detailed, nil
}

// ReadCPUObservation returns the latest scheduler-owned CPU reading for
// threshold evaluation. It never advances collector state or invents a value.
func (s *MonitorService) ReadCPUObservation(ctx context.Context) (models.CPUMetrics, error) {
	detailed, err := s.GetDetailedMetrics(ctx)
	if err != nil {
		return models.CPUMetrics{}, err
	}
	return detailed.CPUDetails, nil
}

func (s *MonitorService) latestCollectorSnapshot() map[string]*collectors.MetricData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*collectors.MetricData, len(s.latestCollectorData))
	for name, data := range s.latestCollectorData {
		if data == nil {
			continue
		}
		values := make(map[string]interface{}, len(data.Values))
		for key, value := range data.Values {
			values[key] = value
		}
		result[name] = &collectors.MetricData{
			CollectorName: data.CollectorName,
			Timestamp:     data.Timestamp,
			Type:          data.Type,
			Values:        values,
			Tags:          data.Tags,
		}
	}
	return result
}

// GetDiskDetail returns read-only disk usage detail plus storage-manager
// remediation guidance. It observes pressure and attribution only; cleanup
// execution remains owned by storage-manager policy/audit.
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
			"Disk detail is observational only; broad cleanup is delegated to storage-manager plan/apply.",
			"Suggested handoff: storage-manager cleanup plan --profile conservative",
		},
	}
	if highestDiskPressure(detail.Partitions) >= 85 {
		detail.Notes = append(detail.Notes, "High disk pressure detected; request a storage-manager preview before applying remediation.")
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
func populateCPUDetails(detailed *models.DetailedMetrics, cpuData, processData *collectors.MetricData, topCPUProcs, topCPUSecondsProcs []map[string]interface{}) {
	if cpuData == nil {
		return
	}

	detailed.CPUDetails = models.CPUMetrics{
		Usage:                    getFloat64Value(cpuData.Values, "usage_percent"),
		LoadAverage:              getFloat64Slice(cpuData.Values, "load_average"),
		UsageState:               metricState(cpuData, "usage_percent", "CPU has not been measured"),
		ContextSwitchesPerSecond: metricState(cpuData, "context_switches_per_second", "context-switch counter rate has not been sampled"),
		InterruptsPerSecond:      metricState(cpuData, "interrupts_per_second", "interrupt counter rate has not been sampled"),
		NormalizedLoad1:          metricState(cpuData, "normalized_load_1", "normalized load is unavailable"),
		NormalizedLoad5:          metricState(cpuData, "normalized_load_5", "normalized load is unavailable"),
		RunQueueDepth:            metricState(cpuData, "run_queue_depth", "run-queue depth is unavailable"),
		ModeBreakdown:            metricStateMap(cpuData, "mode_breakdown", "CPU mode accounting is unavailable"),
		PerCoreUtilization:       metricStateMap(cpuData, "per_core_utilization", "per-core utilization is unavailable"),
		CoreImbalanceIndex:       metricState(cpuData, "core_imbalance_index", "core imbalance is unavailable"),
		QuotaThrottling:          metricState(cpuData, "quota_throttling", "CPU quota throttling is unavailable"),
		FrequencyDerateRatio:     metricState(cpuData, "frequency_derate_ratio", "CPU frequency backend is unavailable"),
		ThermalThrottleEvidence:  metricState(cpuData, "thermal_throttle_evidence", "thermal attribution is unavailable"),
		ThermalTripPointCelsius:  metricState(cpuData, "thermal_trip_point_celsius", "thermal trip point is unavailable"),
		ForkRate:                 metricState(processData, "fork_rate", "fork-rate counter is unavailable"),
	}
	detailed.CPUDetails.LoadAverageState = cpuLoadState(cpuData)

	for _, proc := range topCPUProcs {
		detailed.CPUDetails.TopProcesses = append(detailed.CPUDetails.TopProcesses, convertToProcessInfo(proc))
	}
	for _, proc := range topCPUSecondsProcs {
		detailed.CPUDetails.TopCPUSecondsProcesses = append(detailed.CPUDetails.TopCPUSecondsProcesses, convertToProcessInfo(proc))
	}
}

func metricStateMap(data *collectors.MetricData, key, reason string) map[string]models.MetricState {
	result := map[string]models.MetricState{}
	if data == nil {
		return result
	}
	if raw, ok := data.Values[key].(map[string]float64); ok {
		for name, value := range raw {
			result[name] = models.MetricState{Status: "measured", Value: value, Units: "percent", Provenance: data.Tags["source"], ObservedAt: data.Timestamp}
		}
	}
	if len(result) == 0 {
		result["_state"] = metricState(data, key, reason)
	}
	return result
}

func cpuLoadState(data *collectors.MetricData) models.MetricState {
	state := metricState(data, "load_average", "load average is unavailable")
	if data == nil {
		return state
	}
	if status, ok := data.Values["load_average_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := data.Values["load_average_reason"].(string); ok {
		state.Reason = reason
	}
	if provenance, ok := data.Values["load_average_provenance"].(string); ok && provenance != "" {
		state.Provenance = provenance
	}
	if loads, ok := data.Values["load_average"].([]float64); ok && len(loads) > 0 && state.Status == "measured" {
		state.Value = loads[0]
		state.Units = "load"
		state.Reason = ""
	}
	if state.Status != "measured" {
		state.Value = 0
	}
	return state
}

// populateMemoryDetails fills the memory section of detailed (including disk
// usage, which is stored under MemoryMetrics in proto).
func populateMemoryDetails(detailed *models.DetailedMetrics, memData, diskData, pressureData *collectors.MetricData, topMemProcs, topPagingProcs []map[string]interface{}) {
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
	for _, proc := range topPagingProcs {
		detailed.MemoryDetails.TopPagingProcesses = append(detailed.MemoryDetails.TopPagingProcesses, convertToProcessInfo(proc))
	}
	populatePressureDetails(&detailed.MemoryDetails, pressureData)

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

func populatePressureDetails(memory *models.MemoryMetrics, data *collectors.MetricData) {
	if data == nil {
		return
	}
	memory.Paging = models.PagingMetrics{
		SwapInPerSecond:           pressureState(data, "pswpin_per_second", "pswpin_rate_status", "pswpin_rate_reason"),
		SwapOutPerSecond:          pressureState(data, "pswpout_per_second", "pswpout_rate_status", "pswpout_rate_reason"),
		SwapTrafficPagesPerSecond: pressureState(data, "swap_traffic_pages_per_second", "swap_traffic_rate_status", "swap_traffic_rate_reason"),
		MajorFaultsPerSecond:      pressureState(data, "pgmajfault_per_second", "pgmajfault_rate_status", "pgmajfault_rate_reason"),
		PageFaultsPerSecond:       pressureState(data, "pgfault_per_second", "pgfault_rate_status", "pgfault_rate_reason"),
	}
	fragmentation := models.FragmentationMetrics{
		MaxFreeOrder:           pressureState(data, "fragmentation_max_free_order", "fragmentation_status", "fragmentation_reason"),
		LowOrderShare:          pressureState(data, "fragmentation_low_order_share", "fragmentation_status", "fragmentation_reason"),
		CompactionFailureRatio: pressureState(data, "compaction_failure_ratio", "compaction_failure_ratio_status", "compaction_failure_ratio_reason"),
		CompactionRates:        make(map[string]models.MetricState),
	}
	if histogram, ok := data.Values["buddyinfo"].(map[string]string); ok {
		fragmentation.Buddyinfo = histogram
	}
	for key := range data.Values {
		if strings.HasSuffix(key, "_per_second") && strings.HasPrefix(key, "compact_") {
			fragmentation.CompactionRates[strings.TrimSuffix(key, "_per_second")] = pressureState(data, key, strings.TrimSuffix(key, "_per_second")+"_rate_status", strings.TrimSuffix(key, "_per_second")+"_rate_reason")
		}
	}
	memory.Fragmentation = fragmentation
}

func pressureState(data *collectors.MetricData, valueKey, statusKey, reasonKey string) models.MetricState {
	units := "per second"
	if strings.Contains(valueKey, "psi_") {
		units = "percent"
	}
	state := models.MetricState{Status: "not_yet_sampled", Reason: "rate has not been sampled", Provenance: "system-monitor/pressure", Units: units, ObservedAt: data.Timestamp}
	if _, hasSignalStatus := data.Values[valueKey+"_status"]; hasSignalStatus {
		statusKey = valueKey + "_status"
		reasonKey = valueKey + "_reason"
	}
	if status, ok := data.Values[statusKey].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := data.Values[reasonKey].(string); ok && reason != "" {
		state.Reason = reason
	}
	if value, ok := data.Values[valueKey].(float64); ok {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	return state
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

	// Attribution is present only when the collector judged the connection count
	// alarming enough to walk /proc, so its absence is expected and not an error.
	if attribution, ok := netData.Values["socket_owners"].(collectors.SocketAttribution); ok {
		owners := make([]models.SocketOwnerInfo, 0, len(attribution.Owners))
		for _, owner := range attribution.Owners {
			owners = append(owners, models.SocketOwnerInfo{PID: owner.PID, Name: owner.Comm, Connections: owner.Count})
		}
		detailed.NetworkDetails.SocketOwners = &models.SocketOwnership{
			Owners:     owners,
			Attributed: attribution.Attributed,
			Total:      attribution.Total,
			Supported:  attribution.Supported,
			Reason:     attribution.Reason,
		}
	}
}

// populateSystemDetails fills the system section of detailed (file descriptors
// and inotify watchers) from the disk collector data.
func populateSystemDetails(detailed *models.DetailedMetrics, diskData *collectors.MetricData) {
	detailed.SystemDetails.APIProcessGoroutines = runtime.NumGoroutine()
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

	result.ProcessHealth.ForkRate = forkRateFromValues(data.Values)

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
		PID:                  getIntValue(proc, "pid"),
		Name:                 getStringValue(proc, "name"),
		CPUPercent:           getFloat64Value(proc, "cpu_percent"),
		MemoryMB:             getFloat64Value(proc, "memory_mb"),
		Threads:              getIntValue(proc, "threads"),
		FDs:                  getIntValue(proc, "fd_count"),
		Status:               getStringValue(proc, "status"),
		SwapKB:               getInt64Value(proc, "swap_kb"),
		MajorFaultsPerSecond: getFloat64Value(proc, "major_faults_per_second"),
		CPUSeconds:           getFloat64Value(proc, "cpu_seconds"),
		CPUSecondsState: models.MetricState{
			Status: getStringValue(proc, "cpu_seconds_status"),
			Value:  getFloat64Value(proc, "cpu_seconds"),
			Reason: getStringValue(proc, "cpu_seconds_reason"),
			Units:  "seconds",
		},
	}
}

// forkRateFromValues lifts the process collector's fork-rate keys into the typed
// model. It returns nil only when the collector emitted no fork-rate keys at
// all; an unsupported platform still yields a populated, explicitly unsupported
// value so the absence is visible rather than silently green.
func forkRateFromValues(values map[string]interface{}) *models.ForkRateInfo {
	status, ok := values["fork_rate_status"].(string)
	if !ok {
		return nil
	}
	if status != "measured" {
		reason, _ := values["fork_rate_reason"].(string)
		return &models.ForkRateInfo{Supported: false, Reason: reason}
	}
	info := &models.ForkRateInfo{Supported: true}
	if total, ok := values["forks_total"].(uint64); ok {
		info.ForksTotal = total
	}
	if rate, ok := values["forks_per_second"].(float64); ok {
		info.ForksPerSecond = rate
	}
	if pending, ok := values["fork_rate_pending"].(bool); ok {
		info.Pending = pending
	}
	if source, ok := values["fork_rate_source"].(string); ok {
		info.Source = source
	}
	return info
}
