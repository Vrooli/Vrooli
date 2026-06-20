// Package metrics is a generic, reusable execution-metrics primitive: a
// Collector measures the cost of any unit of work (a provider validation, a
// build, a long-running async task) and produces a commonv1.ExecutionMetrics.
//
// It is environment-adaptive: it captures what the current machine/OS supports
// and marks the rest UNAVAILABLE via per-area Reliability, rather than reporting
// a misleading 0. The collector depends only on the generated commonv1 proto,
// the standard library, and golang.org/x/sys (transitively, via syscall) — it
// imports no maturity-go, no test-genie, no host-inventory source. Richer host
// facts (full memory, GPUs) enter by VALUE through WithEnvironment /
// WithGpuSampler so api-core gains no new dependency edge.
//
// Usage:
//
//	m := metrics.Start(metrics.WithEnvironment(env))
//	func() { defer m.Stage("compile").End() }()
//	out := m.Stop() // *commonv1.ExecutionMetrics
package metrics

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// activeCollectors counts collectors currently between Start and Stop. When more
// than one is active, process-wide rusage can no longer be cleanly attributed to
// a single unit of work, so CPU/RSS degrade to BEST_EFFORT.
var activeCollectors atomic.Int64

// GpuSampler returns a snapshot of per-device GPU usage for the current window.
// It is opt-in (it typically shells nvidia-smi/rocm-smi) and supplied via
// WithGpuSampler; absent, GPU usage is reported UNAVAILABLE.
type GpuSampler func(context.Context) []*commonv1.GpuUsage

// Option customizes a Collector at Start.
type Option func(*Collector)

// WithEnvironment overrides the stdlib baseline CaptureEnvironment with richer
// host facts (full memory total, present GPUs, host id). os/arch/num_cpu are
// backfilled from the runtime when the supplied value leaves them zero, so the
// baseline guarantee always holds.
func WithEnvironment(env *commonv1.CaptureEnvironment) Option {
	return func(c *Collector) {
		if env != nil {
			c.env = env
		}
	}
}

// WithGpuSampler wires an opt-in GPU snapshot the collector samples at stage and
// whole-op boundaries. Without it, GPU usage is UNAVAILABLE.
func WithGpuSampler(sampler GpuSampler) Option {
	return func(c *Collector) {
		c.gpuSampler = sampler
	}
}

// Collector accumulates timing, stages, gauges, and resource usage for one unit
// of work. It is intended for single-flight use within one logical operation;
// it is not safe for concurrent mutation from multiple goroutines.
type Collector struct {
	ctx        context.Context
	startedAt  time.Time
	baseRusage rusageSample
	env        *commonv1.CaptureEnvironment
	gpuSampler GpuSampler

	stages []*Stage
	gauges map[string]float64

	mu        sync.Mutex
	maxActive int64

	stopped bool
	result  *commonv1.ExecutionMetrics
}

// Start stamps started_at, samples the rusage baseline, and registers the
// collector as active.
func Start(opts ...Option) *Collector {
	c := &Collector{
		ctx:        context.Background(),
		startedAt:  time.Now(),
		baseRusage: sampleRusage(),
		gauges:     map[string]float64{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	c.observeActive(activeCollectors.Add(1))
	return c
}

// Gauge attaches a whole-operation domain number (io bytes, gc count, tokens,
// usd cost…) not first-classed by the schema.
func (c *Collector) Gauge(name string, value float64) {
	if c == nil || name == "" {
		return
	}
	c.gauges[name] = value
}

// Stage opens a measured scope of work. Close it with End; nest further with
// (*Stage).Stage. Flat usage stays trivial: defer c.Stage("compile").End().
func (c *Collector) Stage(name string) *Stage {
	if c == nil {
		return nil
	}
	s := newStage(c, name)
	c.stages = append(c.stages, s)
	return s
}

// Stop stamps completed_at, computes wall_clock_ms, fills the whole-op resource
// rollup and the CaptureEnvironment, and returns the assembled metrics. It is
// idempotent: repeated calls return the same result and only release the
// active-collector slot once.
func (c *Collector) Stop() *commonv1.ExecutionMetrics {
	if c == nil {
		return nil
	}
	if c.stopped {
		return c.result
	}
	c.stopped = true
	c.observeActive(activeCollectors.Load())
	completed := time.Now()
	end := sampleRusage()

	m := &commonv1.ExecutionMetrics{
		WallClockMs: completed.Sub(c.startedAt).Milliseconds(),
		StartedAt:   timestamppb.New(c.startedAt),
		CompletedAt: timestamppb.New(completed),
		Gauges:      copyGauges(c.gauges),
		Resources:   c.resourceUsage(c.baseRusage, end),
		Environment: c.buildEnvironment(),
	}
	for _, st := range c.stages {
		if st == nil {
			continue
		}
		if !st.ended {
			st.End()
		}
		m.Stages = append(m.Stages, st.proto)
	}

	activeCollectors.Add(-1)
	c.result = m
	return m
}

// Stage is one measured scope of work — the profiling-span / flamegraph model.
type Stage struct {
	c          *Collector
	name       string
	startedAt  time.Time
	baseRusage rusageSample
	gauges     map[string]float64
	children   []*Stage

	ended bool
	proto *commonv1.Stage
}

func newStage(c *Collector, name string) *Stage {
	return &Stage{
		c:          c,
		name:       name,
		startedAt:  time.Now(),
		baseRusage: sampleRusage(),
		gauges:     map[string]float64{},
	}
}

// Gauge attaches a per-stage domain number (tokens, rows…).
func (s *Stage) Gauge(name string, value float64) *Stage {
	if s == nil || name == "" {
		return s
	}
	s.gauges[name] = value
	return s
}

// Stage opens a nested child to narrow attribution further.
func (s *Stage) Stage(name string) *Stage {
	if s == nil {
		return nil
	}
	child := newStage(s.c, name)
	s.children = append(s.children, child)
	return child
}

// End closes the stage, recording duration_ms and a best-effort per-stage
// ResourceUsage. It is idempotent.
func (s *Stage) End() *Stage {
	if s == nil || s.ended {
		return s
	}
	s.ended = true
	duration := time.Since(s.startedAt)
	end := sampleRusage()

	s.proto = &commonv1.Stage{
		Name:       s.name,
		DurationMs: duration.Milliseconds(),
		Resources:  s.c.resourceUsage(s.baseRusage, end),
		Gauges:     copyGauges(s.gauges),
	}
	for _, child := range s.children {
		if child == nil {
			continue
		}
		if !child.ended {
			child.End()
		}
		s.proto.Children = append(s.proto.Children, child.proto)
	}
	return s
}

// resourceUsage builds a ResourceUsage from a start/end rusage pair, stamping
// per-area Reliability. CPU/RSS are RELIABLE when single-flight and measurable,
// BEST_EFFORT when another collector is concurrently active, UNAVAILABLE when
// the platform cannot sample rusage. GPU is sampled only when a sampler is wired.
func (c *Collector) resourceUsage(start, end rusageSample) *commonv1.ResourceUsage {
	usage := &commonv1.ResourceUsage{}
	rel := c.cpuMemReliability(start, end)
	if rel != commonv1.Reliability_RELIABILITY_UNAVAILABLE {
		usage.CpuUserMs = end.cpuUserMs - start.cpuUserMs
		usage.CpuSysMs = end.cpuSysMs - start.cpuSysMs
		usage.PeakRssBytes = end.maxRSSBytes
	}
	usage.Cpu = rel
	usage.Memory = rel
	usage.Gpu, usage.Gpus = c.gpuUsage()
	return usage
}

func (c *Collector) cpuMemReliability(start, end rusageSample) commonv1.Reliability {
	if !start.ok || !end.ok {
		return commonv1.Reliability_RELIABILITY_UNAVAILABLE
	}
	c.mu.Lock()
	concurrent := c.maxActive > 1
	c.mu.Unlock()
	if concurrent {
		return commonv1.Reliability_RELIABILITY_BEST_EFFORT
	}
	return commonv1.Reliability_RELIABILITY_RELIABLE
}

func (c *Collector) gpuUsage() (commonv1.Reliability, []*commonv1.GpuUsage) {
	if c.gpuSampler == nil {
		return commonv1.Reliability_RELIABILITY_UNAVAILABLE, nil
	}
	gpus := c.gpuSampler(c.ctx)
	if len(gpus) == 0 {
		return commonv1.Reliability_RELIABILITY_UNAVAILABLE, nil
	}
	return commonv1.Reliability_RELIABILITY_BEST_EFFORT, gpus
}

func (c *Collector) buildEnvironment() *commonv1.CaptureEnvironment {
	env := &commonv1.CaptureEnvironment{}
	if c.env != nil {
		env = proto.Clone(c.env).(*commonv1.CaptureEnvironment)
	}
	if env.GetOs() == "" {
		env.Os = runtime.GOOS
	}
	if env.GetArch() == "" {
		env.Arch = runtime.GOARCH
	}
	if env.GetNumCpu() == 0 {
		env.NumCpu = int32(runtime.NumCPU())
	}
	if env.GetRuntimeVersion() == "" {
		env.RuntimeVersion = runtime.Version()
	}
	return env
}

func (c *Collector) observeActive(n int64) {
	c.mu.Lock()
	if n > c.maxActive {
		c.maxActive = n
	}
	c.mu.Unlock()
}

func copyGauges(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
