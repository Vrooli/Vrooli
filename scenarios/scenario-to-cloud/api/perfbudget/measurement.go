package perfbudget

import (
	"bufio"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Phase names of one measurement. Every measurement carries all four so a
// report can never omit the post-cleanup comparison silently.
const (
	PhaseBaseline    = "baseline"
	PhaseSustained   = "sustained"
	PhasePeak        = "peak"
	PhasePostCleanup = "post_cleanup"
)

// Phases in report order.
func Phases() []string {
	return []string{PhaseBaseline, PhaseSustained, PhasePeak, PhasePostCleanup}
}

// Measurement is one harness run against one fixture profile. Schema version
// 1. It is an evidence record: it stores what was observed, including sample
// counts, and never a derived pass/fail.
type Measurement struct {
	SchemaVersion int               `json:"schema_version"`
	Profile       string            `json:"profile"`
	Target        TargetDescriptor  `json:"target"`
	StartedAt     time.Time         `json:"started_at"`
	FinishedAt    time.Time         `json:"finished_at"`
	Concurrency   int               `json:"concurrency"`
	Phases        map[string]*Phase `json:"phases"`
	Notes         []string          `json:"notes,omitempty"`
}

// TargetDescriptor says what was measured. Kind is "handler" for an
// in-process http.Handler (tests) or "url" for an explicit base URL.
type TargetDescriptor struct {
	Kind    string `json:"kind"`
	BaseURL string `json:"base_url,omitempty"`
	// Explicit is true only when the operator named the target. A url
	// measurement without it is refused before any request is sent.
	Explicit bool `json:"explicit"`
}

// Phase is one measured window.
type Phase struct {
	Name       string          `json:"name"`
	StartedAt  time.Time       `json:"started_at"`
	FinishedAt time.Time       `json:"finished_at"`
	Latency    LatencySummary  `json:"latency"`
	Process    ProcessSnapshot `json:"process_end"`
	ProcessAt  ProcessSnapshot `json:"process_start"`
}

// LatencySummary summarises request latencies of one phase. SmallSample is
// set when Samples is below the budget's min_samples_for_percentiles; the
// percentiles are still reported (as observed values) but Compare refuses to
// turn them into a verdict.
type LatencySummary struct {
	Samples     int     `json:"samples"`
	Errors      int     `json:"errors"`
	ErrorRate   float64 `json:"error_rate"`
	MinMs       float64 `json:"min_ms"`
	P50Ms       float64 `json:"p50_ms"`
	P95Ms       float64 `json:"p95_ms"`
	P99Ms       float64 `json:"p99_ms"`
	MaxMs       float64 `json:"max_ms"`
	SmallSample bool    `json:"small_sample"`
	MinSamples  int     `json:"min_samples"`
}

// ProcessSnapshot is a sample of the management process taken by the
// harness itself: Go runtime metrics plus /proc/self on Linux. Available is
// false when /proc could not be read; RSS and FDs are then zero and Compare
// reports insufficient evidence for them rather than "within".
type ProcessSnapshot struct {
	TakenAt        time.Time `json:"taken_at"`
	Goroutines     int       `json:"goroutines"`
	HeapAllocBytes uint64    `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64    `json:"heap_sys_bytes"`
	RSSBytes       uint64    `json:"rss_bytes"`
	OpenFDs        int       `json:"open_fds"`
	Available      bool      `json:"proc_available"`
}

// RSSMiB is the resident set in MiB.
func (p ProcessSnapshot) RSSMiB() float64 { return float64(p.RSSBytes) / (1024 * 1024) }

// Summarise computes a LatencySummary from raw sample durations. Errors are
// counted separately from latencies (an errored request still has a latency
// and is included in the percentiles so a fast failure cannot flatter them).
func Summarise(latencies []time.Duration, errors, minSamples int) LatencySummary {
	s := LatencySummary{Samples: len(latencies), Errors: errors, MinSamples: minSamples}
	if s.Samples == 0 {
		s.SmallSample = true
		return s
	}
	s.ErrorRate = float64(errors) / float64(s.Samples)
	ms := make([]float64, len(latencies))
	for i, d := range latencies {
		ms[i] = float64(d) / float64(time.Millisecond)
	}
	sort.Float64s(ms)
	s.MinMs = ms[0]
	s.MaxMs = ms[len(ms)-1]
	s.P50Ms = percentile(ms, 50)
	s.P95Ms = percentile(ms, 95)
	s.P99Ms = percentile(ms, 99)
	s.SmallSample = s.Samples < minSamples
	return s
}

// percentile uses the nearest-rank method over sorted values.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}

// SnapshotProcess samples the current process. It never fails: on a host
// without /proc the Linux-only fields are zero and Available is false.
func SnapshotProcess() ProcessSnapshot {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	snap := ProcessSnapshot{
		TakenAt:        time.Now().UTC(),
		Goroutines:     runtime.NumGoroutine(),
		HeapAllocBytes: ms.HeapAlloc,
		HeapSysBytes:   ms.HeapSys,
	}
	if rss, ok := readVmRSS("/proc/self/status"); ok {
		snap.RSSBytes = rss
		snap.Available = true
	}
	if n, ok := countFDs("/proc/self/fd"); ok {
		snap.OpenFDs = n
	} else {
		snap.Available = false
	}
	return snap
}

func readVmRSS(path string) (uint64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "VmRSS:"))
		if len(fields) == 0 {
			return 0, false
		}
		kb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

func countFDs(dir string) (int, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, false
	}
	return len(entries), true
}

// Complete reports whether every phase is present with a start and end
// process snapshot; a measurement missing a phase is not a full run.
func (m *Measurement) Complete() []string {
	var missing []string
	for _, name := range Phases() {
		ph, ok := m.Phases[name]
		if !ok || ph == nil {
			missing = append(missing, name)
			continue
		}
		if ph.ProcessAt.TakenAt.IsZero() || ph.Process.TakenAt.IsZero() {
			missing = append(missing, name+":process")
		}
	}
	return missing
}
