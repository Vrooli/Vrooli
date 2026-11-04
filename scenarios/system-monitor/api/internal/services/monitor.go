package services

// DOC: docs/concepts/ARCHITECTURE.md#monitoring-service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/infrastructure"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

var cycleSequence atomic.Uint64

// MonitorService handles system monitoring operations
type MonitorService struct {
	logger         logx.Logger
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
	loopMu         sync.Mutex
	loopDone       chan struct{}
	loopRunning    bool
	loopStopping   bool

	// Per-process sampling seams (3b/3c/3d). sampler walks /proc once per cycle;
	// attributor maps each pid to its owning scenario; both are nil-safe so a
	// service constructed without them simply skips process sampling.
	snapshots         collectors.SnapshotProvider
	deviceGraphs      collectors.DeviceGraphProvider
	sampler           procsampler.Sampler
	attributor        *procsampler.Attributor
	procSampleEvery   time.Duration
	procSampleTopN    int
	procSampleOwners  []string
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

// WithMonitorLogger sets the structured logger used by monitoring loops.
func WithMonitorLogger(logger logx.Logger) MonitorOption {
	return func(s *MonitorService) {
		if logger != nil {
			s.logger = logger
		}
	}
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
		logger:                 logx.Default(),
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
		procSampleOwners:       append([]string(nil), cfg.Monitoring.ProcSampleOwners...),
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

	network := collectors.NewNetworkCollector(s.config.Monitoring.SocketAttributionThreshold)
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
	s.logger.Info("starting monitor service")
	s.loopMu.Lock()
	if s.loopStopping {
		s.loopMu.Unlock()
		return fmt.Errorf("monitor collection loop is still stopping")
	}
	if s.loopRunning {
		s.loopMu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.mu.Lock()
	s.ctx, s.cancel = ctx, cancel
	s.mu.Unlock()
	s.loopDone, s.loopRunning = done, true
	s.loopMu.Unlock()

	go func() {
		defer close(done)
		s.collectionLoop(ctx)
		s.loopMu.Lock()
		s.loopRunning = false
		s.loopMu.Unlock()
	}()

	s.logger.Info("monitor service started")
	return nil
}

// Stop gracefully shuts down the service
func (s *MonitorService) Stop() {
	_ = s.StopContext(context.Background())
}

// StopContext cancels collection and waits for the in-flight collector cycle
// to return. A timeout reports uncertainty and leaves Start fenced until the
// same loop's completion receipt arrives.
func (s *MonitorService) StopContext(ctx context.Context) error {
	s.logger.Info("stopping monitor service")
	s.loopMu.Lock()
	if !s.loopRunning && s.loopDone == nil {
		s.loopMu.Unlock()
		return nil
	}
	s.loopStopping = true
	done, cancel := s.loopDone, s.cancel
	s.loopMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.loopMu.Lock()
	s.loopStopping = false
	s.loopDone = nil
	s.loopMu.Unlock()
	s.logger.Info("monitor service stopped")
	return nil
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
	if collector, ok := s.collectors.Get("pressure"); ok {
		if pressure, ok := collector.(*collectors.PressureCollector); ok {
			pressure.SetMemoryPressureThresholds(MemoryPressureThresholdsFromSettings(settings))
		}
	}
}

// MemoryPressureThresholdsFromSettings maps settings onto the collector's
// level boundaries.
func MemoryPressureThresholdsFromSettings(settings Settings) collectors.MemoryPressureThresholds {
	return collectors.MemoryPressureThresholds{
		SomeAvg60ElevatedPercent:     settings.MemoryPressureSomeAvg60ElevatedPercent,
		FullAvg60CriticalPercent:     settings.MemoryPressureFullAvg60CriticalPercent,
		SwapInElevatedBytesPerSecond: settings.MemoryPressureSwapInElevatedBytesPerSecond,
		SwapInCriticalBytesPerSecond: settings.MemoryPressureSwapInCriticalBytesPerSecond,
	}
}

// EffectiveCollectionInterval returns the current baseline collection interval.
func (s *MonitorService) EffectiveCollectionInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricInterval
}

// collectionLoop continuously collects metrics. The tick interval is recomputed
// each cycle so live settings changes take effect without a restart.
func (s *MonitorService) collectionLoop(ctx context.Context) {
	for {
		timer := time.NewTimer(s.collectionTickInterval())
		select {
		case <-ctx.Done():
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
type collectionBatch struct {
	metrics []*collectors.MetricData
	errors  []error
	skipped int
}

func (s *MonitorService) collectMetrics() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	now := s.clock.Now()
	cycleID := fmt.Sprintf("cycle-%d-%d", now.UnixNano(), cycleSequence.Add(1))
	cycleStarted := time.Now()
	cycleForksBefore := collectors.CommandForkCount()
	batch := s.collectEnabledMetrics(ctx, now)

	for _, err := range batch.errors {
		s.logger.Warn("metric collection failed", "error", err)
	}
	s.persistMetricCycle(ctx, cycleID, now, batch.metrics)
	if s.shouldSampleProcesses(now) {
		s.sampleProcesses(ctx, now, gpuVRAMByPID(batch.metrics))
	}
	s.recordCycleSelfMetrics(time.Since(cycleStarted), collectors.CommandForkCount()-cycleForksBefore, now)
	s.recordCycleOutcome(len(batch.errors), batch.skipped)
}

func (s *MonitorService) collectEnabledMetrics(ctx context.Context, now time.Time) collectionBatch {
	batch := collectionBatch{}
	for _, collector := range s.collectors.GetEnabled() {
		name := collector.GetName()
		if !s.shouldCollect(name, collector.GetInterval(), now) {
			batch.skipped++
			continue
		}

		started := time.Now()
		forksBefore := collectors.CommandForkCount()
		data, err := collector.Collect(ctx)
		s.recordCollectorSelfMetrics(name, time.Since(started), collectors.CommandForkCount()-forksBefore)
		s.markCollected(name, now)
		if err != nil {
			batch.errors = append(batch.errors, err)
			batch.metrics = append(batch.metrics, &collectors.MetricData{
				CollectorName: name,
				Timestamp:     now,
				Values:        map[string]interface{}{"status": "failed", "reason": err.Error()},
			})
			continue
		}
		batch.metrics = append(batch.metrics, data)
	}
	return batch
}

func (s *MonitorService) persistMetricCycle(ctx context.Context, cycleID string, now time.Time, metrics []*collectors.MetricData) {
	observations := make([]repository.MetricObservation, 0, len(metrics))
	for _, data := range metrics {
		if data == nil {
			continue
		}
		observations = append(observations, repository.MetricObservation{CollectorName: data.CollectorName, Values: data.Values})
	}
	if len(observations) == 0 {
		return
	}

	s.mu.Lock()
	for _, data := range metrics {
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

	s.updateLatestSnapshot(cycleID, now, metrics)
	persistenceStarted := time.Now()
	if err := s.repo.SaveMetricCycle(ctx, cycleID, now, observations); err != nil {
		s.logger.Error("failed to persist metric cycle", "cycle_id", cycleID, "error", err)
	}
	s.mu.Lock()
	s.lastPersistenceDuration = time.Since(persistenceStarted)
	s.mu.Unlock()
}

func (s *MonitorService) recordCycleOutcome(failures, skipped int) {
	s.mu.Lock()
	s.lastCycleFailures = failures
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
			response.KernelMemory = repository.KernelMemoryFromValues(metric.Values, cycleID, metric.Timestamp)
			response.MemoryState = metricState(metric, "usage_percent", "memory collector unavailable")
			markObserved(&response.MemoryState)
			if value, ok := metric.Values["usage_percent"].(float64); ok {
				response.MemoryUsage = value
			}
		case "network":
			response.ConnectionsState = metricState(metric, "tcp_connections", "network collector unavailable")
			markObserved(&response.ConnectionsState)
			response.NetworkEstablishedRate = metricState(metric, "established_rate_per_second", "established rate has not been sampled")
			response.NetworkTimeWaitRate = metricState(metric, "time_wait_rate_per_second", "TIME_WAIT rate has not been sampled")
			response.NetworkCloseWaitRate = metricState(metric, "close_wait_rate_per_second", "CLOSE_WAIT rate has not been sampled")
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

// GetProcessTimelineInRange returns process attribution over the explicit
// half-open interval [start, end). The bounded range supports repeatable
// comparisons of externally timestamped work such as capture arms.
func (s *MonitorService) GetProcessTimelineInRange(ctx context.Context, start, end time.Time, owner string, top int, rank string) ([]repository.ProcessTimelineEntry, error) {
	if s.procRepo == nil {
		return nil, nil
	}
	if start.IsZero() || end.IsZero() || !end.After(start) || end.Sub(start) > 24*time.Hour {
		return nil, fmt.Errorf("process timeline range must be positive and no longer than 24 hours")
	}
	if rank != "rss" && rank != "gpu" && rank != "cpu_seconds" {
		rank = "cpu"
	}
	return s.procRepo.QueryProcessTimeline(ctx, repository.ProcessTimelineQuery{
		Start: start.UTC(),
		End:   end.UTC(),
		Owner: owner,
		Top:   top,
		Rank:  rank,
	})
}

const (
	DefaultProcessSampleQueryLimit = 1000
	MaxProcessSampleQueryLimit     = 5000
)

// GetProcessSamplesInRange returns owner-scoped raw observations from an exact
// half-open interval. One extra row detects truncation without an unbounded
// read. Raw samples are used intentionally; rolled-up rows cannot support
// per-arm attribution.
func (s *MonitorService) GetProcessSamplesInRange(ctx context.Context, start, end time.Time, owner string, limit int) ([]repository.ProcessSample, bool, error) {
	if s.procRepo == nil {
		return nil, false, nil
	}
	if start.IsZero() || end.IsZero() || !end.After(start) || end.Sub(start) > 24*time.Hour {
		return nil, false, fmt.Errorf("process sample range must be positive and no longer than 24 hours")
	}
	if strings.TrimSpace(owner) == "" {
		return nil, false, fmt.Errorf("process sample query requires an owner")
	}
	if limit <= 0 {
		limit = DefaultProcessSampleQueryLimit
	}
	if limit > MaxProcessSampleQueryLimit {
		return nil, false, fmt.Errorf("process sample query limit must be no greater than %d", MaxProcessSampleQueryLimit)
	}
	rows, err := s.procRepo.QueryProcessSamples(ctx, repository.ProcessSampleQuery{
		Start: start.UTC(), End: end.UTC(), Owner: owner, Limit: limit + 1,
	})
	if err != nil {
		return nil, false, err
	}
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}
	return rows, truncated, nil
}

// GetCurrentMetrics retrieves the current system metrics
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
	if memoryPressure, err := s.GetMemoryPressure(ctx); err == nil {
		detailed.MemoryDetails.MemoryPressure = memoryPressure
	}
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

// GetNetworkDiagnostic performs an explicitly requested, bounded attribution
// snapshot. It never advances the steady scheduler's collector state.
func (s *MonitorService) GetNetworkDiagnostic(ctx context.Context, topN int, maxDuration time.Duration, _ bool) (*models.NetworkDiagnosticSnapshot, error) {
	snapshot := &models.NetworkDiagnosticSnapshot{}
	latest := s.latestCollectorSnapshot()["network"]
	if latest != nil {
		if states, ok := latest.Values["tcp_states"].(map[string]int); ok {
			snapshot.InventoryConnections = states["established"]
		}
	}
	collector, ok := s.collectors.Get("network")
	if !ok || !collector.IsEnabled() {
		snapshot.FailureReason = "network collector is disabled"
		return snapshot, nil
	}
	network, ok := collector.(*collectors.NetworkCollector)
	if !ok {
		snapshot.FailureReason = "network collector does not support diagnostics"
		return snapshot, nil
	}
	result := network.CollectNetworkDiagnostic(ctx, collectors.NetworkDiagnosticRequest{
		Established: snapshot.InventoryConnections,
		TopN:        topN,
		MaxDuration: maxDuration,
	})
	owners := make([]models.NetworkOwner, len(result.Owners))
	for i, owner := range result.Owners {
		owners[i] = models.NetworkOwner{PID: owner.PID, ProcessName: owner.Comm, Connections: owner.Count}
	}
	endpoints := make([]models.NetworkEndpoint, len(result.Endpoints))
	for i, endpoint := range result.Endpoints {
		endpoints[i] = models.NetworkEndpoint{Scope: endpoint.Scope, Direction: endpoint.Direction, Port: endpoint.Port, Connections: endpoint.Connections}
	}
	coverage := 0.0
	if result.Total > 0 {
		coverage = float64(result.Attributed) * 100 / float64(result.Total)
		if coverage > 100 {
			// A socket inode can be referenced by more than one process (for
			// example after fork or descriptor sharing). Coverage describes
			// inventory records, not descriptor references, so it is bounded.
			coverage = 100
		}
	}
	snapshot.Ownership = &models.NetworkOwnership{
		Owners: owners, TotalConnections: result.Total, AttributedConnections: result.Attributed,
		AttributionCoveragePercent: coverage, Truncated: result.Truncated,
		Reason: result.Reason, Provenance: "native platform socket attribution",
	}
	snapshot.Endpoints = endpoints
	snapshot.InventoryConnections = result.Total
	snapshot.Truncated = result.Truncated
	snapshot.DurationMilliseconds = result.Duration.Milliseconds()
	snapshot.FailureReason = result.Reason
	return snapshot, nil
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
		detail.Notes = append(detail.Notes, "High disk pressure detected; storage-manager starts bounded autonomous recovery.")
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
