package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// Provider isolates infrastructure-level telemetry.
type Provider interface {
	GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error)
	CheckServiceDependencies() []models.ServiceHealth
}

// StaticProvider returns placeholder infrastructure data.
type StaticProvider struct{}

func NewStaticProvider() *StaticProvider {
	return &StaticProvider{}
}

func (p *StaticProvider) GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error) {
	result := &models.InfrastructureMonitorData{
		Timestamp: time.Now(),
	}

	result.DatabasePools = []models.ConnectionPool{
		{
			Name:     "postgres-main",
			Active:   8,
			Idle:     2,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}

	result.HTTPClientPools = []models.ConnectionPool{
		{
			Name:     "scenario-api-1->ollama",
			Active:   3,
			Idle:     7,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}

	result.MessageQueues = models.MessageQueueInfo{
		RedisPubSub: models.RedisPubSubInfo{
			Subscribers: 12,
			Channels:    5,
		},
		BackgroundJobs: models.BackgroundJobsInfo{
			Pending: 3,
			Active:  1,
			Failed:  0,
		},
	}

	result.StorageIO = models.StorageIOInfo{
		QueueDepthState: models.MetricState{Status: "unsupported", Reason: "Disk queue depth is not collected", Provenance: "system-monitor/infrastructure/storage_io", Units: "requests"},
		IOWaitState:     models.MetricState{Status: "unsupported", Reason: "Disk I/O wait is not collected", Provenance: "system-monitor/infrastructure/storage_io", Units: "%"},
	}

	return result, nil
}

func (p *StaticProvider) CheckServiceDependencies() []models.ServiceHealth {
	return []models.ServiceHealth{
		{
			Name:      "postgres",
			Status:    "healthy",
			LatencyMs: 2.5,
			LastCheck: time.Now(),
			Endpoint:  "localhost:5432",
		},
		{
			Name:      "redis",
			Status:    "healthy",
			LatencyMs: 0.8,
			LastCheck: time.Now(),
			Endpoint:  "localhost:6379",
		},
		{
			Name:      "qdrant",
			Status:    "healthy",
			LatencyMs: 4.1,
			LastCheck: time.Now(),
			Endpoint:  "localhost:6333",
		},
		{
			Name:      "ollama",
			Status:    "healthy",
			LatencyMs: 15.2,
			LastCheck: time.Now(),
			Endpoint:  "localhost:11434",
		},
	}
}

// MeasuredProvider preserves the existing infrastructure workspace while
// replacing placeholder disk signals with native physical-device counters.
// Throughput rates require consecutive samples; queue depth is instantaneous.
type MeasuredProvider struct {
	*StaticProvider
	mu         sync.Mutex
	read       func() (diskCounters, error)
	now        func() time.Time
	previous   *diskCounters
	previousAt time.Time
	lastRead   models.MetricState
	lastWrite  models.MetricState
	lastQueue  models.MetricState
}

type (
	diskCounter  struct{ readSectors, writeSectors, inFlight uint64 }
	diskCounters map[string]diskCounter
)

var errDiskUnsupported = errors.New("physical disk counters unavailable")

func NewMeasuredProvider() *MeasuredProvider {
	return &MeasuredProvider{StaticProvider: NewStaticProvider(), read: readPhysicalDiskCounters, now: time.Now}
}

func (p *MeasuredProvider) GetInfrastructureMonitorData(ctx context.Context) (*models.InfrastructureMonitorData, error) {
	result, err := p.StaticProvider.GetInfrastructureMonitorData(ctx)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	// Concurrent readers share one observation instead of producing noisy
	// near-zero-duration rates from each other's requests.
	if p.previous != nil && !now.Before(p.previousAt) && now.Sub(p.previousAt) < time.Second {
		result.StorageIO.ReadRate, result.StorageIO.WriteRate = p.lastRead, p.lastWrite
		result.StorageIO.QueueDepthState = p.lastQueue
		result.StorageIO.ReadMBPerSec, result.StorageIO.WriteMBPerSec = p.lastRead.Value, p.lastWrite.Value
		result.StorageIO.DiskQueueDepth = p.lastQueue.Value
		return result, nil
	}
	counters, err := p.read()
	state := models.MetricState{Status: "not_yet_sampled", Reason: "A second disk counter sample is required", Provenance: "Linux physical devices /proc/diskstats", Units: "MB/s", ObservedAt: now}
	read, write := state, state
	queue := models.MetricState{Status: "not_yet_sampled", Reason: "Disk queue depth is not yet sampled", Provenance: "Linux physical devices /proc/diskstats", Units: "requests", ObservedAt: now}
	if err != nil {
		state.Status = "failed"
		if errors.Is(err, errDiskUnsupported) {
			state.Status = "unsupported"
		}
		state.Reason = err.Error()
		read, write = state, state
		queue.Status, queue.Reason = state.Status, err.Error()
		p.previous = nil
	} else if p.previous != nil && now.After(p.previousAt) {
		readDelta, writeDelta, valid := diskCounterDeltas(*p.previous, counters)
		if valid {
			seconds := now.Sub(p.previousAt).Seconds()
			read.Status, write.Status = "measured", "measured"
			read.Reason, write.Reason = "", ""
			read.Value = float64(readDelta) * 512 / seconds / 1_000_000
			write.Value = float64(writeDelta) * 512 / seconds / 1_000_000
			read.Provenance = fmt.Sprintf("Linux physical devices /proc/diskstats, averaged over %.2f seconds", seconds)
			write.Provenance = read.Provenance
		} else {
			read.Reason, write.Reason = "Disk counters reset or inventory changed", "Disk counters reset or inventory changed"
		}
	}
	if err == nil {
		if inFlight, ok := diskQueueDepth(counters); ok {
			queue.Status, queue.Reason, queue.Value = "measured", "", inFlight
		}
		p.previous = &counters
		p.previousAt = now
	}
	result.Timestamp = now
	result.StorageIO.ReadRate, result.StorageIO.WriteRate = read, write
	result.StorageIO.QueueDepthState = queue
	p.lastRead, p.lastWrite = read, write
	p.lastQueue = queue
	result.StorageIO.ReadMBPerSec, result.StorageIO.WriteMBPerSec = read.Value, write.Value
	result.StorageIO.DiskQueueDepth = queue.Value
	return result, nil
}

// Linux diskstats sectors are fixed 512-byte units. Enumerate whole physical
// sysfs devices only, excluding partitions and virtual/stacked devices so the
// same I/O is not counted twice. Refuse a partial inventory rather than report
// a silently incomplete aggregate.
func readPhysicalDiskCounters() (diskCounters, error) {
	if runtime.GOOS != "linux" {
		return diskCounters{}, errDiskUnsupported
	}
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return diskCounters{}, fmt.Errorf("read disk inventory: %w", err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		path, err := filepath.EvalSymlinks(filepath.Join("/sys/block", entry.Name()))
		if err != nil {
			return diskCounters{}, fmt.Errorf("resolve disk inventory: %w", err)
		}
		if !strings.Contains(path, "/virtual/") {
			names[entry.Name()] = true
		}
	}
	if len(names) == 0 {
		return diskCounters{}, errDiskUnsupported
	}
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return diskCounters{}, fmt.Errorf("read disk counters: %w", err)
	}
	return parsePhysicalDiskCounters(string(data), names)
}

func parsePhysicalDiskCounters(data string, names map[string]bool) (diskCounters, error) {
	result := diskCounters{}
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 12 || !names[fields[2]] {
			continue
		}
		read, err1 := strconv.ParseUint(fields[5], 10, 64)
		write, err2 := strconv.ParseUint(fields[9], 10, 64)
		inFlight, err3 := strconv.ParseUint(fields[11], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			return diskCounters{}, fmt.Errorf("invalid disk counters for %s", fields[2])
		}
		result[fields[2]] = diskCounter{readSectors: read, writeSectors: write, inFlight: inFlight}
	}
	if len(result) != len(names) {
		return nil, errors.New("disk counter inventory incomplete")
	}

	return result, nil
}

func diskQueueDepth(counters diskCounters) (float64, bool) {
	if len(counters) == 0 {
		return 0, false
	}
	var total uint64
	for _, counter := range counters {
		if ^uint64(0)-total < counter.inFlight {
			return 0, false
		}
		total += counter.inFlight
	}
	return float64(total), true
}

func diskCounterDeltas(previous, current diskCounters) (uint64, uint64, bool) {
	if len(previous) != len(current) || len(current) == 0 {
		return 0, 0, false
	}
	var readTotal, writeTotal uint64
	for name, next := range current {
		prior, ok := previous[name]
		if !ok || next.readSectors < prior.readSectors || next.writeSectors < prior.writeSectors {
			return 0, 0, false
		}
		read, write := next.readSectors-prior.readSectors, next.writeSectors-prior.writeSectors
		if ^uint64(0)-readTotal < read || ^uint64(0)-writeTotal < write {
			return 0, 0, false
		}
		readTotal += read
		writeTotal += write
	}
	return readTotal, writeTotal, true
}
