// Package analysis turns a captured performance.json (CDP trace) + web-vitals
// into a deterministic, located findings set: it pairs ⚛ blink.user_timing
// begin/end marks by id2.local into a per-component count/avg/max table, derives
// long-task / paint / LCP deltas, and locates each hot component definition
// (file:line via symbol lookup). NO AI — findings are deterministic. A
// before/after comparison primitive diffs two traces of the same interaction.
//
// The real aggregation (ported from the scenario-performance-audit skill's
// Phase 5) lives in FileTraceLoader (parser.go) behind the TraceLoader seam;
// SourceLocator (locate.go) does deterministic component→file:line resolution.
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
	Scenario   string
	Components []ComponentTiming
	LongTaskMs int64
	LCPMs      int64
	FCPMs      int64
	Findings   []Finding
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
