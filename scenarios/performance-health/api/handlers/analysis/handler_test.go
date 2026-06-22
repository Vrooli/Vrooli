package analysis

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalanalysis "performance-health/internal/analysis"
	"performance-health/internal/perfsample"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
)

// fakeLoader drives the analysis Service's lowest seam, returning a canned
// Result per artifact path so Compare can diff two distinct traces.
type fakeLoader struct {
	byArtifact map[string]internalanalysis.Result
	err        error
}

func (f *fakeLoader) Load(_ context.Context, scenario, artifact string) (internalanalysis.Result, error) {
	if f.err != nil {
		return internalanalysis.Result{}, f.err
	}
	res := f.byArtifact[artifact]
	res.Scenario = scenario
	return res, nil
}

type fakeSampleWriter struct {
	samples []perfsample.Sample
}

func (f *fakeSampleWriter) Insert(_ context.Context, s perfsample.Sample) error {
	f.samples = append(f.samples, s)
	return nil
}

// TestAnalyzeTraceMapsResultToProto builds the REAL analysis service over a
// fake loader, calls AnalyzeTrace, and asserts the web-vitals, per-component
// table, and located findings map correctly across the proto boundary. It also
// proves the LCP + slowest-component reading is persisted to the trend.
func TestAnalyzeTraceMapsResultToProto(t *testing.T) {
	loader := &fakeLoader{byArtifact: map[string]internalanalysis.Result{
		"trace.json": {
			LongTaskMs: 120,
			LCPMs:      2400,
			FCPMs:      800,
			Components: []internalanalysis.ComponentTiming{
				{Component: "Dashboard", CommitCount: 9, AvgMs: 18.5, MaxMs: 42.0, Definition: "src/Dashboard.tsx:12"},
				{Component: "Sidebar", CommitCount: 3, AvgMs: 4.0, MaxMs: 6.0},
			},
			Findings: []internalanalysis.Finding{{
				Code: "PERF_SLOW_COMPONENT", Component: "Dashboard", Definition: "src/Dashboard.tsx:12",
				Message: "avg commit 18.5ms", Evidence: "9 commits", Severity: "warning",
			}},
		},
	}}
	writer := &fakeSampleWriter{}
	h := NewHandler(internalanalysis.NewService(loader), writer, nil)

	resp, err := h.AnalyzeTrace(context.Background(), connect.NewRequest(&analysisv1.AnalyzeTraceRequest{
		Scenario: "demo", TraceArtifact: "trace.json",
	}))
	if err != nil {
		t.Fatalf("AnalyzeTrace: %v", err)
	}
	msg := resp.Msg
	if len(msg.GetComponents()) != 2 {
		t.Fatalf("components len = %d, want 2", len(msg.GetComponents()))
	}
	if len(msg.GetFindings()) != 1 {
		t.Fatalf("findings len = %d, want 1", len(msg.GetFindings()))
	}
	// The capture's LCP + slowest-component reading is persisted to the trend.
	if len(writer.samples) != 1 {
		t.Fatalf("expected 1 persisted sample, got %d", len(writer.samples))
	}
	top := msg.GetComponents()[0] // service sorts components by avg desc — Dashboard first.
	f := msg.GetFindings()[0]
	s := writer.samples[0]
	for _, c := range []struct {
		got, want any
		label     string
	}{
		{msg.GetScenario(), "demo", "scenario"},
		{msg.GetLongTaskMs(), int64(120), "long_task_ms"},
		{msg.GetLcpMs(), int64(2400), "lcp_ms"},
		{msg.GetFcpMs(), int64(800), "fcp_ms"},
		{top.GetComponent(), "Dashboard", "top.component"},
		{top.GetCommitCount(), int32(9), "top.commit_count"},
		{top.GetAvgMs(), 18.5, "top.avg_ms"},
		{top.GetMaxMs(), 42.0, "top.max_ms"},
		{top.GetDefinition(), "src/Dashboard.tsx:12", "top.definition"},
		{f.GetCode(), "PERF_SLOW_COMPONENT", "finding.code"},
		{f.GetComponent(), "Dashboard", "finding.component"},
		{f.GetSeverity(), "warning", "finding.severity"},
		{s.LCPMs, int64(2400), "sample.lcp_ms"},
		{s.SlowestComponent, "Dashboard", "sample.slowest_component"},
		{s.SlowestComponentAvgMs, 18.5, "sample.slowest_avg_ms"},
		{s.SlowestComponentMaxMs, 42.0, "sample.slowest_max_ms"},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.label, c.got, c.want)
		}
	}
}

// TestCompareTracesMapsDeltaToProto proves the CompareTraces RPC diffs two
// traces and maps the per-component + web-vitals deltas to the proto response.
func TestCompareTracesMapsDeltaToProto(t *testing.T) {
	loader := &fakeLoader{byArtifact: map[string]internalanalysis.Result{
		"base.json": {
			LongTaskMs: 100, LCPMs: 2000,
			Components: []internalanalysis.ComponentTiming{{Component: "Dashboard", CommitCount: 5, AvgMs: 10, MaxMs: 20}},
		},
		"cand.json": {
			LongTaskMs: 160, LCPMs: 2500,
			Components: []internalanalysis.ComponentTiming{{Component: "Dashboard", CommitCount: 8, AvgMs: 18, MaxMs: 30}},
		},
	}}
	h := NewHandler(internalanalysis.NewService(loader), nil, nil)

	resp, err := h.CompareTraces(context.Background(), connect.NewRequest(&analysisv1.CompareTracesRequest{
		Scenario: "demo", BaselineArtifact: "base.json", CandidateArtifact: "cand.json",
	}))
	if err != nil {
		t.Fatalf("CompareTraces: %v", err)
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" || msg.GetLongTaskDeltaMs() != 60 || msg.GetLcpDeltaMs() != 500 {
		t.Errorf("delta web-vitals mapped wrong: %+v", msg)
	}
	if len(msg.GetComponents()) != 1 {
		t.Fatalf("component deltas len = %d, want 1", len(msg.GetComponents()))
	}
	d := msg.GetComponents()[0]
	if d.GetComponent() != "Dashboard" || d.GetBaselineAvgMs() != 10 || d.GetCandidateAvgMs() != 18 || d.GetDeltaMs() != 8 {
		t.Errorf("avg delta mapped wrong: %+v", d)
	}
	if d.GetBaselineCount() != 5 || d.GetCandidateCount() != 8 || d.GetCountDelta() != 3 {
		t.Errorf("count delta mapped wrong: %+v", d)
	}
	if d.GetMaxDeltaMs() != 10 {
		t.Errorf("max delta = %v, want 10", d.GetMaxDeltaMs())
	}
}

// TestAnalyzeTraceRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestAnalyzeTraceRequiresScenario(t *testing.T) {
	h := NewHandler(internalanalysis.NewService(&fakeLoader{}), nil, nil)
	_, err := h.AnalyzeTrace(context.Background(), connect.NewRequest(&analysisv1.AnalyzeTraceRequest{TraceArtifact: "trace.json"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
