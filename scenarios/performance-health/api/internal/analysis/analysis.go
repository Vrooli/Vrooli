// Package analysis turns a captured performance.json (CDP trace) + web-vitals
// into a deterministic, located findings set: it pairs ⚛ blink.user_timing
// begin/end marks by id2.local into a per-component count/avg/max table, derives
// long-task / paint / LCP deltas, and locates each hot component definition
// (file:line via symbol lookup). NO AI — findings are deterministic. A
// before/after comparison primitive diffs two traces of the same interaction.
//
// The real aggregation (which productizes the legacy hand-rolled perf-audit
// analysis step) lives in FileTraceLoader (parser.go) behind the TraceLoader
// seam; SourceLocator (locate.go) does deterministic component→file:line
// resolution.
package analysis

import (
	"context"
	"errors"
	"sort"
)

// ComponentTiming is one React component's commit profile across a trace.
type ComponentTiming struct {
	Component   string
	CommitCount int
	AvgMs       float64
	MaxMs       float64
	// Definition is the located file:line, empty if not located.
	Definition string
}

// Result is the outcome of analyzing one trace.
type Result struct {
	Scenario      string
	Components    []ComponentTiming
	LongTaskMs    int64
	LongTaskMaxMs float64
	LCPMs         int64
	FCPMs         int64
	FrameSummary  FrameSummary
	BrowserWork   []EventSummary
	InputEvents   []EventSummary
	Findings      []Finding
}

// Finding is one deterministic, located performance finding.
type Finding struct {
	Code       string
	Component  string
	Definition string
	Message    string
	Evidence   string
	Severity   string
}

// FrameSummary describes compositor frame health across a trace or marked
// interaction window. Counts are deliberately event-name based so small CDP
// fixtures can exercise the contract without full browser traces.
type FrameSummary struct {
	TraceDurationMs   float64
	BeginFrameCount   int
	DrawnFrameCount   int
	DroppedFrameCount int
	ApproxDrawnFPS    float64
	DroppedFrameRate  float64
}

// EventSummary is a deterministic rollup for expensive browser work or input
// dispatch by event name/type.
type EventSummary struct {
	Name    string
	Count   int
	TotalMs float64
	MaxMs   float64
	AvgMs   float64
}

// ComponentDelta is one component's commit-profile delta between two traces.
// Cardinality (count) and per-commit cost (avg/max) are tracked together so the
// "fix raised commit count but dropped per-commit cost" case is visible.
type ComponentDelta struct {
	Component      string
	BaselineCount  int
	CandidateCount int
	CountDelta     int
	BaselineAvgMs  float64
	CandidateAvgMs float64
	DeltaMs        float64
	BaselineMaxMs  float64
	CandidateMaxMs float64
	MaxDeltaMs     float64
}

// Comparison is the outcome of comparing two traces of the same interaction.
type Comparison struct {
	Scenario        string
	Components      []ComponentDelta
	LongTaskDeltaMs int64
	LCPDeltaMs      int64
	FrameDelta      FrameDelta
	BrowserWork     []EventDelta
	InputEvents     []EventDelta
}

// FrameDelta is candidate minus baseline for frame-health metrics.
type FrameDelta struct {
	TraceDurationDeltaMs   float64
	BeginFrameCountDelta   int
	DrawnFrameCountDelta   int
	DroppedFrameCountDelta int
	ApproxDrawnFPSDelta    float64
	DroppedFrameRateDelta  float64
}

// EventDelta is candidate minus baseline for one browser-work or input-event
// summary row.
type EventDelta struct {
	Name             string
	BaselineCount    int
	CandidateCount   int
	CountDelta       int
	BaselineTotalMs  float64
	CandidateTotalMs float64
	TotalDeltaMs     float64
	BaselineMaxMs    float64
	CandidateMaxMs   float64
	MaxDeltaMs       float64
	BaselineAvgMs    float64
	CandidateAvgMs   float64
	AvgDeltaMs       float64
}

// TraceLoader is the seam that reads a captured trace artifact into a parsed
// Result. The real implementation (FileTraceLoader) parses CDP JSON and locates
// components within the scenario's UI source; tests drive a fake. The scenario
// slug is threaded so the loader can locate component definitions.
type TraceLoader interface {
	Load(ctx context.Context, scenario, artifact string) (Result, error)
}

// Service is the analysis engine.
type Service struct {
	loader TraceLoader
}

// NewService wires an analysis Service over the trace-loader seam.
func NewService(loader TraceLoader) *Service { return &Service{loader: loader} }

// Analyze parses one trace into a located, quantified findings set.
func (s *Service) Analyze(ctx context.Context, scenario, artifact string) (Result, error) {
	if s == nil || s.loader == nil {
		return Result{}, errors.New("analysis: service not wired")
	}
	if scenario == "" {
		return Result{}, errors.New("analysis: scenario is required")
	}
	if artifact == "" {
		return Result{}, errors.New("analysis: trace artifact is required")
	}
	res, err := s.loader.Load(ctx, scenario, artifact)
	if err != nil {
		return Result{}, err
	}
	res.Scenario = scenario
	sortComponents(res.Components)
	return res, nil
}

// Compare diffs two traces of the same interaction.
func (s *Service) Compare(ctx context.Context, scenario, baseline, candidate string) (Comparison, error) {
	if s == nil || s.loader == nil {
		return Comparison{}, errors.New("analysis: service not wired")
	}
	if scenario == "" {
		return Comparison{}, errors.New("analysis: scenario is required")
	}
	if baseline == "" || candidate == "" {
		return Comparison{}, errors.New("analysis: baseline and candidate artifacts are required")
	}
	base, err := s.loader.Load(ctx, scenario, baseline)
	if err != nil {
		return Comparison{}, err
	}
	cand, err := s.loader.Load(ctx, scenario, candidate)
	if err != nil {
		return Comparison{}, err
	}
	return Comparison{
		Scenario:        scenario,
		Components:      diffComponents(base.Components, cand.Components),
		LongTaskDeltaMs: cand.LongTaskMs - base.LongTaskMs,
		LCPDeltaMs:      cand.LCPMs - base.LCPMs,
		FrameDelta:      diffFrames(base.FrameSummary, cand.FrameSummary),
		BrowserWork:     diffEvents(base.BrowserWork, cand.BrowserWork),
		InputEvents:     diffEvents(base.InputEvents, cand.InputEvents),
	}, nil
}

func sortComponents(c []ComponentTiming) {
	sort.Slice(c, func(i, j int) bool {
		if c[i].AvgMs != c[j].AvgMs {
			return c[i].AvgMs > c[j].AvgMs
		}
		return c[i].Component < c[j].Component
	})
}

func diffFrames(baseline, candidate FrameSummary) FrameDelta {
	return FrameDelta{
		TraceDurationDeltaMs:   round1(candidate.TraceDurationMs - baseline.TraceDurationMs),
		BeginFrameCountDelta:   candidate.BeginFrameCount - baseline.BeginFrameCount,
		DrawnFrameCountDelta:   candidate.DrawnFrameCount - baseline.DrawnFrameCount,
		DroppedFrameCountDelta: candidate.DroppedFrameCount - baseline.DroppedFrameCount,
		ApproxDrawnFPSDelta:    round1(candidate.ApproxDrawnFPS - baseline.ApproxDrawnFPS),
		DroppedFrameRateDelta:  round1(candidate.DroppedFrameRate - baseline.DroppedFrameRate),
	}
}

func diffEvents(baseline, candidate []EventSummary) []EventDelta {
	byName := map[string]EventSummary{}
	for _, s := range baseline {
		byName[s.Name] = s
	}
	seen := map[string]bool{}
	out := make([]EventDelta, 0, len(baseline)+len(candidate))
	for _, c := range candidate {
		seen[c.Name] = true
		b := byName[c.Name]
		out = append(out, eventDelta(b, c))
	}
	for name, b := range byName {
		if seen[name] {
			continue
		}
		out = append(out, eventDelta(b, EventSummary{Name: name}))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalDeltaMs != out[j].TotalDeltaMs {
			return out[i].TotalDeltaMs > out[j].TotalDeltaMs
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func eventDelta(baseline, candidate EventSummary) EventDelta {
	name := candidate.Name
	if name == "" {
		name = baseline.Name
	}
	return EventDelta{
		Name:             name,
		BaselineCount:    baseline.Count,
		CandidateCount:   candidate.Count,
		CountDelta:       candidate.Count - baseline.Count,
		BaselineTotalMs:  baseline.TotalMs,
		CandidateTotalMs: candidate.TotalMs,
		TotalDeltaMs:     round1(candidate.TotalMs - baseline.TotalMs),
		BaselineMaxMs:    baseline.MaxMs,
		CandidateMaxMs:   candidate.MaxMs,
		MaxDeltaMs:       round1(candidate.MaxMs - baseline.MaxMs),
		BaselineAvgMs:    baseline.AvgMs,
		CandidateAvgMs:   candidate.AvgMs,
		AvgDeltaMs:       round1(candidate.AvgMs - baseline.AvgMs),
	}
}

// diffComponents pairs components by name across two traces and computes the
// commit-count, average-commit-time, and max-commit-time deltas. Components
// present in only one trace contribute a one-sided delta. Sorted by largest
// per-commit-cost regression first.
func diffComponents(baseline, candidate []ComponentTiming) []ComponentDelta {
	byName := map[string]ComponentTiming{}
	for _, c := range baseline {
		byName[c.Component] = c
	}
	var out []ComponentDelta
	seen := map[string]bool{}
	for _, c := range candidate {
		seen[c.Component] = true
		b := byName[c.Component]
		out = append(out, ComponentDelta{
			Component:      c.Component,
			BaselineCount:  b.CommitCount,
			CandidateCount: c.CommitCount,
			CountDelta:     c.CommitCount - b.CommitCount,
			BaselineAvgMs:  b.AvgMs,
			CandidateAvgMs: c.AvgMs,
			DeltaMs:        round1(c.AvgMs - b.AvgMs),
			BaselineMaxMs:  b.MaxMs,
			CandidateMaxMs: c.MaxMs,
			MaxDeltaMs:     round1(c.MaxMs - b.MaxMs),
		})
	}
	for name, b := range byName {
		if seen[name] {
			continue
		}
		out = append(out, ComponentDelta{
			Component:     name,
			BaselineCount: b.CommitCount,
			CountDelta:    -b.CommitCount,
			BaselineAvgMs: b.AvgMs,
			DeltaMs:       round1(-b.AvgMs),
			BaselineMaxMs: b.MaxMs,
			MaxDeltaMs:    round1(-b.MaxMs),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DeltaMs != out[j].DeltaMs {
			return out[i].DeltaMs > out[j].DeltaMs
		}
		return out[i].Component < out[j].Component
	})
	return out
}
