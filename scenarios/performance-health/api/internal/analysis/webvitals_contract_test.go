package analysis

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// testdata/web-vitals.json is a VERBATIM performance.web-vitals.json from a real
// BAS capture (secrets-manager, 2026-08-21). It is the producer contract in
// fixture form: the payload shape is decided by
// scenarios/browser-automation-studio/playwright-driver/src/tracing/web-vitals-script.ts,
// which stores raw PerformanceEntry fields.
//
// It exists because the consumer struct drifted from that producer without
// anything failing: it read `start` where the producer writes `startTime`, and
// `lcp.start` where the producer writes `lcp.value`. LCP and FCP silently
// decoded as 0 on every capture — and LCP flows through perfsample into the
// trend table, which is what `budget set --lcp-max-ms` compares against, so
// every LCP budget was measuring against zero.
const webVitalsFixture = "testdata/web-vitals.json"

// TestWebVitalsFixtureMatchesProducerFieldNames pins the exact JSON keys the
// capture script writes. If the producer renames one, this fails here rather
// than silently zeroing a metric downstream.
func TestWebVitalsFixtureMatchesProducerFieldNames(t *testing.T) {
	raw, err := os.ReadFile(webVitalsFixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("fixture is not valid JSON: %v", err)
	}
	for _, key := range []string{"paint", "lcp", "cls", "longTasks", "fcp", "navigation"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("fixture lost top-level key %q — the producer contract changed", key)
		}
	}

	var shape struct {
		Paint []map[string]any `json:"paint"`
		LCP   map[string]any   `json:"lcp"`
		CLS   map[string]any   `json:"cls"`
		Long  []map[string]any `json:"longTasks"`
		Nav   map[string]any   `json:"navigation"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		t.Fatalf("unmarshal shape: %v", err)
	}
	// Timings are PerformanceEntry fields: startTime, never start.
	if len(shape.Paint) == 0 {
		t.Fatal("fixture has no paint entries")
	}
	if _, ok := shape.Paint[0]["startTime"]; !ok {
		t.Error("paint entries must carry startTime")
	}
	if _, ok := shape.Paint[0]["start"]; ok {
		t.Error("paint entries must NOT carry start — that was the bug")
	}
	if _, ok := shape.LCP["value"]; !ok {
		t.Error("lcp must carry value")
	}
	if _, ok := shape.CLS["value"]; !ok {
		t.Error("cls must carry value")
	}
	if len(shape.Long) == 0 {
		t.Fatal("fixture has no longTasks")
	}
	for _, key := range []string{"startTime", "duration"} {
		if _, ok := shape.Long[0][key]; !ok {
			t.Errorf("longTasks entries must carry %q", key)
		}
	}
	// The navigation block flattens PerformanceNavigationTiming under the
	// producer's own names, which differ from the spec's field names
	// (domContentLoaded here is domContentLoadedEventEnd on the entry).
	for _, key := range []string{"domContentLoaded", "loadEventEnd", "responseEnd", "domInteractive", "type"} {
		if _, ok := shape.Nav[key]; !ok {
			t.Errorf("navigation must carry %q", key)
		}
	}
}

// TestLoadDecodesRealWebVitals: the parser must produce the fixture's real
// numbers, not zeros. The assertions are exact because the fixture is fixed.
func TestLoadDecodesRealWebVitals(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "performance.json")
	// A Tier-0 trace (no ⚛ marks) is enough: web-vitals parsing is independent
	// of component marks, and this keeps the fixture focused on the bug.
	if err := os.WriteFile(trace, []byte(`{"traceEvents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(webVitalsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "performance.web-vitals.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := FileTraceLoader{}.Load(context.Background(), "secrets-manager", trace)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if res.LCPMs != 248 {
		t.Errorf("LCPMs = %d, want 248 (0 means the lcp field name regressed)", res.LCPMs)
	}
	if res.FCPMs != 248 {
		t.Errorf("FCPMs = %d, want 248 (0 means the paint/fcp field name regressed)", res.FCPMs)
	}
	if res.LongTaskMs != 209 {
		t.Errorf("LongTaskMs = %d, want 209", res.LongTaskMs)
	}
	if res.CLS < 0.029 || res.CLS > 0.030 {
		t.Errorf("CLS = %v, want ~0.0294 (0 means cls is being dropped again)", res.CLS)
	}
}

// TestFCPFallsBackToPaintEntry: a payload carrying only the paint entry (no
// top-level fcp) still reports a real FCP.
func TestFCPFallsBackToPaintEntry(t *testing.T) {
	var wv webVitals
	body := `{"paint":[{"name":"first-contentful-paint","startTime":312}]}`
	if err := json.Unmarshal([]byte(body), &wv); err != nil {
		t.Fatal(err)
	}
	if got := fcpMs(wv); got != 312 {
		t.Errorf("fcpMs = %d, want 312", got)
	}
}

// TestVitalsAbsentIsZeroNotError: a Tier-0 capture with no sidecar must parse
// cleanly with zeroed vitals rather than failing analysis.
func TestVitalsAbsentIsZeroNotError(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "performance.json")
	if err := os.WriteFile(trace, []byte(`{"traceEvents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := FileTraceLoader{}.Load(context.Background(), "demo", trace)
	if err != nil {
		t.Fatalf("Load must not error without a sidecar: %v", err)
	}
	if res.LCPMs != 0 || res.FCPMs != 0 || res.CLS != 0 {
		t.Errorf("expected zeroed vitals, got LCP=%d FCP=%d CLS=%v", res.LCPMs, res.FCPMs, res.CLS)
	}
}

// TestLoadDecodesNavigationTiming: the four PerformanceNavigationTiming phases
// and the navigation type come through from the verbatim fixture. The phases
// are monotonic, which is the cheapest check that the fields are not
// cross-wired — a swap between responseEnd and loadEventEnd would still yield
// four plausible-looking numbers.
func TestLoadDecodesNavigationTiming(t *testing.T) {
	res := loadFixtureResult(t)

	if res.ResponseEndMs != 4 {
		t.Errorf("ResponseEndMs = %d, want 4", res.ResponseEndMs)
	}
	if res.DOMInteractiveMs != 14 {
		t.Errorf("DOMInteractiveMs = %d, want 14", res.DOMInteractiveMs)
	}
	if res.DOMContentLoadedMs != 101 {
		t.Errorf("DOMContentLoadedMs = %d, want 101", res.DOMContentLoadedMs)
	}
	if res.LoadEventEndMs != 202 {
		t.Errorf("LoadEventEndMs = %d, want 202", res.LoadEventEndMs)
	}
	if res.NavigationType != "navigate" {
		t.Errorf("NavigationType = %q, want navigate", res.NavigationType)
	}

	if !(res.ResponseEndMs <= res.DOMInteractiveMs &&
		res.DOMInteractiveMs <= res.DOMContentLoadedMs &&
		res.DOMContentLoadedMs <= res.LoadEventEndMs) {
		t.Errorf("navigation phases are not monotonic — fields are cross-wired: response=%d interactive=%d dcl=%d load=%d",
			res.ResponseEndMs, res.DOMInteractiveMs, res.DOMContentLoadedMs, res.LoadEventEndMs)
	}
}

// TestNavigationAbsentIsZeroNotError: a capture whose observer never fired
// (entry type unsupported) parses cleanly with zeroed navigation.
func TestNavigationAbsentIsZeroNotError(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "performance.json")
	if err := os.WriteFile(trace, []byte(`{"traceEvents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "performance.web-vitals.json"),
		[]byte(`{"fcp":100,"lcp":{"value":200}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := FileTraceLoader{}.Load(context.Background(), "demo", trace)
	if err != nil {
		t.Fatalf("Load must not error without a navigation block: %v", err)
	}
	if res.LoadEventEndMs != 0 || res.NavigationType != "" {
		t.Errorf("expected zeroed navigation, got load=%d type=%q", res.LoadEventEndMs, res.NavigationType)
	}
	if res.LCPMs != 200 {
		t.Errorf("the rest of the payload must still parse, LCPMs = %d", res.LCPMs)
	}
}

// loadFixtureResult parses the verbatim capture fixture through the real loader.
func loadFixtureResult(t *testing.T) Result {
	t.Helper()
	dir := t.TempDir()
	trace := filepath.Join(dir, "performance.json")
	if err := os.WriteFile(trace, []byte(`{"traceEvents":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(webVitalsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "performance.web-vitals.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := FileTraceLoader{}.Load(context.Background(), "secrets-manager", trace)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return res
}
