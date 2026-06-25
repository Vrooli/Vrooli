// Package procsampler provides a single-pass process sampler that reads
// /proc/<pid>/{stat,cmdline,comm,cwd} once per cycle to yield pid, ppid,
// command line, working directory, CPU% (via inter-sample utime/stime delta),
// resident memory, and thread count.
//
// It replaces the per-metric `bash -c "ps | sort | head"` pipelines the
// collectors previously shelled out for: one /proc walk per cycle instead of
// ~10 forks. The Linux implementation lives in sampler_linux.go (build-tagged);
// non-Linux platforms return ErrUnsupported (see cross-platform-readiness).
package procsampler

// DOC: docs/internal/SEAMS.md#process-sampler

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrUnsupported is returned by Sample on platforms without a /proc filesystem.
// Callers treat it as "no samples this cycle", not a fatal error.
var ErrUnsupported = errors.New("procsampler: unsupported on this platform")

// ProcessSample is a single process observed during one sampling cycle. CPUPct
// is the share of one CPU consumed since the previous sample (so it can exceed
// 100% for multi-threaded processes), derived from the utime/stime delta.
type ProcessSample struct {
	PID     int
	PPID    int
	Comm    string  // short name from /proc/<pid>/comm
	Cmdline string  // full command line (nul-joined args rendered with spaces)
	Cwd     string  // resolved /proc/<pid>/cwd target ("" when unreadable)
	Owner   string  // attributed scenario name or "unknown" (filled by the attributor)
	CPUPct  float64 // CPU% over the inter-sample interval
	RSSKB   int64   // resident set size in kibibytes
	Threads int
	// UTime/STime are the raw cumulative CPU ticks read this cycle; retained so
	// the sampler can compute the next cycle's delta. Not persisted.
	utime uint64
	stime uint64
}

// Sampler walks the process table once and returns a sample per live process.
// Implementations tolerate races (a process that exits mid-walk is skipped, not
// an error) so a cycle is always best-effort.
type Sampler interface {
	// Sample returns the live process table. The returned slice is sorted by
	// descending CPUPct so callers can cheaply take a top-N slice.
	Sample(ctx context.Context) ([]ProcessSample, error)
}

// CachedSampler shares one underlying /proc walk across callers inside a short
// freshness window. It is intentionally tiny: the monitor cycle can call top-N
// helpers and attribution persistence back-to-back without re-walking /proc,
// while later cycles still refresh normally.
type CachedSampler struct {
	base Sampler
	ttl  time.Duration
	now  func() time.Time

	mu        sync.Mutex
	sampledAt time.Time
	samples   []ProcessSample
}

// NewCachedSampler wraps base with a short-lived in-memory cache. Non-positive
// ttl values disable caching while keeping the same interface.
func NewCachedSampler(base Sampler, ttl time.Duration) *CachedSampler {
	return &CachedSampler{base: base, ttl: ttl, now: time.Now}
}

// Sample returns the cached process table when it is still fresh, otherwise it
// delegates to the wrapped sampler and stores a defensive copy.
func (s *CachedSampler) Sample(ctx context.Context) ([]ProcessSample, error) {
	if s == nil || s.base == nil {
		return nil, ErrUnsupported
	}
	now := s.now()
	s.mu.Lock()
	if s.ttl > 0 && !s.sampledAt.IsZero() && now.Sub(s.sampledAt) < s.ttl {
		out := cloneSamples(s.samples)
		s.mu.Unlock()
		return out, nil
	}
	s.mu.Unlock()

	samples, err := s.base.Sample(ctx)
	if err != nil {
		return samples, err
	}

	s.mu.Lock()
	s.sampledAt = now
	s.samples = cloneSamples(samples)
	out := cloneSamples(s.samples)
	s.mu.Unlock()
	return out, nil
}

func cloneSamples(in []ProcessSample) []ProcessSample {
	if len(in) == 0 {
		return []ProcessSample{}
	}
	out := make([]ProcessSample, len(in))
	copy(out, in)
	return out
}

// clockTicksPerSec is the kernel USER_HZ used to convert /proc utime+stime
// (measured in clock ticks) into seconds. Linux is 100 on every supported
// architecture; the sampler does not depend on cgo to read sysconf(_SC_CLK_TCK).
const clockTicksPerSec = 100

// cpuDeltaTracker holds the previous cycle's cumulative CPU ticks per pid so the
// next cycle can compute a percentage. It is not safe for concurrent use; the
// sampler owns one and only touches it from Sample.
type cpuDeltaTracker struct {
	prev     map[int]cpuTicks
	prevTime time.Time
}

type cpuTicks struct {
	utime uint64
	stime uint64
}

func newCPUDeltaTracker() *cpuDeltaTracker {
	return &cpuDeltaTracker{prev: make(map[int]cpuTicks)}
}

// apply fills CPUPct on each sample from the delta since the previous cycle,
// then records the current ticks for the next cycle. The first cycle (or a
// newly-seen pid) reports 0% — there is no prior point to subtract.
func (t *cpuDeltaTracker) apply(samples []ProcessSample, now time.Time) {
	elapsed := now.Sub(t.prevTime).Seconds()
	next := make(map[int]cpuTicks, len(samples))
	for i := range samples {
		s := &samples[i]
		cur := cpuTicks{utime: s.utime, stime: s.stime}
		next[s.PID] = cur
		if elapsed <= 0 {
			continue
		}
		prior, ok := t.prev[s.PID]
		if !ok {
			continue
		}
		// Guard against pid reuse / counter resets: a negative delta means a
		// different process now holds this pid, so skip rather than report a
		// bogus spike.
		if cur.utime < prior.utime || cur.stime < prior.stime {
			continue
		}
		deltaTicks := float64((cur.utime - prior.utime) + (cur.stime - prior.stime))
		s.CPUPct = (deltaTicks / clockTicksPerSec) / elapsed * 100.0
	}
	t.prev = next
	t.prevTime = now
}

// sortByCPUDesc orders samples by descending CPU%, tie-broken by RSS so the
// output is deterministic for tests and stable top-N slices.
func sortByCPUDesc(samples []ProcessSample) {
	sort.SliceStable(samples, func(i, j int) bool {
		if samples[i].CPUPct != samples[j].CPUPct {
			return samples[i].CPUPct > samples[j].CPUPct
		}
		return samples[i].RSSKB > samples[j].RSSKB
	})
}
