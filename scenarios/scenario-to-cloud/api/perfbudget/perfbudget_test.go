package perfbudget

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const budgetsPath = "../../certification/budgets.json"

func loadBudgets(t *testing.T) *Budgets {
	t.Helper()
	b, err := Load(filepath.FromSlash(budgetsPath))
	if err != nil {
		t.Fatalf("load budgets: %v", err)
	}
	return b
}

// TestBudgetsLoadAndFreezeEveryRequiredBound [REQ:STC-P0-043] proves P23-O01
// and P23-O02: the checked-in budgets carry a numeric bound for every
// fixture profile, retry class, queue limit, retention family, the soak and
// the detection budget, and the phase-1 numbers are not duplicated with a
// different value.
func TestBudgetsLoadAndFreezeEveryRequiredBound(t *testing.T) {
	b := loadBudgets(t)
	if b.Phase23.FrozenAt == "" {
		t.Fatalf("phase_23.frozen_at missing")
	}
	for _, name := range RequiredFixtureProfiles() {
		fp, ok := b.Profile(name)
		if !ok {
			t.Fatalf("profile %s missing", name)
		}
		if fp.Latency.P95Max > b.Qualification.APIReadLatency.P95MsMax {
			t.Fatalf("profile %s p95 %.0f weaker than the phase-1 policy %.0f", name, fp.Latency.P95Max, b.Qualification.APIReadLatency.P95MsMax)
		}
	}
	if b.Phase23.ReaderConcurrency != b.Qualification.APIReadLatency.ConcurrentReaders {
		t.Fatalf("reader concurrency %d != policy %d", b.Phase23.ReaderConcurrency, b.Qualification.APIReadLatency.ConcurrentReaders)
	}
	for _, class := range RequiredRetryClasses() {
		rb, err := b.RetryFor(class)
		if err != nil {
			t.Fatal(err)
		}
		if rb.TotalBackoff() <= 0 && rb.MaxAttempts > 1 {
			t.Fatalf("class %s: multi-attempt class without backoff", class)
		}
	}
	if b.Phase23.DeploymentQueue.EffectfulOperationsPerDeploymentMax != 1 || b.Phase23.DeploymentQueue.EffectfulOperationsPerHostMax != 4 {
		t.Fatalf("queue limits = %+v", b.Phase23.DeploymentQueue)
	}
	if b.Phase23.Soak.DurationHoursMin != 24 || b.Phase23.Detection.AlertDetectionSecondsMax != 60 {
		t.Fatalf("soak/detection = %+v / %+v", b.Phase23.Soak, b.Phase23.Detection)
	}
	// Existing phase-1/2 keys are preserved (extend, never shrink).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b.Raw, &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"artifacts", "cleanup_ownership", "concurrency", "machine_profile", "qualification", "retention", "spend", "timeouts_seconds", "phase_23"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("budgets.json lost key %q", key)
		}
	}
}

// TestBudgetsValidationRefusesUnboundedOrDriftingBudgets [REQ:STC-P0-043]
// proves the validator refuses an unbounded retry class, a missing profile
// and a phase-23 number that drifts from its phase-1 owner.
func TestBudgetsValidationRefusesUnboundedOrDriftingBudgets(t *testing.T) {
	b := loadBudgets(t)
	mutate := func(t *testing.T, f func(m map[string]any)) error {
		t.Helper()
		var m map[string]any
		if err := json.Unmarshal(b.Raw, &m); err != nil {
			t.Fatal(err)
		}
		f(m)
		data, _ := json.Marshal(m)
		_, err := Parse(data)
		return err
	}
	p23 := func(m map[string]any) map[string]any { return m["phase_23"].(map[string]any) }

	cases := map[string]struct {
		f    func(m map[string]any)
		want string
	}{
		"unbounded retry": {func(m map[string]any) {
			p23(m)["retries"].(map[string]any)["safe_replay"].(map[string]any)["max_attempts"] = 0
		}, "max_attempts must be at least 1"},
		"absurd retry ceiling": {func(m map[string]any) {
			p23(m)["retries"].(map[string]any)["recover"].(map[string]any)["max_attempts"] = 1000
		}, "not a bounded ceiling"},
		"missing retry class": {func(m map[string]any) {
			delete(p23(m)["retries"].(map[string]any), "observe_then_replay")
		}, `retry class "observe_then_replay" has no budget`},
		"missing profile": {func(m map[string]any) {
			delete(p23(m)["fixture_profiles"].(map[string]any), "sql-uploads")
		}, `fixture profile "sql-uploads" missing`},
		"idle drift": {func(m map[string]any) {
			p23(m)["management_process"].(map[string]any)["idle"].(map[string]any)["rss_mib_max"] = 999
		}, "management_process.idle must equal qualification.idle_management_overhead"},
		"two writers": {func(m map[string]any) {
			p23(m)["deployment_queue"].(map[string]any)["effectful_operations_per_deployment_max"] = 2
		}, "must be 1"},
		"short soak": {func(m map[string]any) {
			p23(m)["soak"].(map[string]any)["duration_hours_min"] = 8
		}, "at least 24"},
		"rss over machine": {func(m map[string]any) {
			p23(m)["fixture_profiles"].(map[string]any)["stateless-web"].(map[string]any)["target_resources"].(map[string]any)["rss_mib_max"] = 8192
		}, "exceeds the machine profile"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := mutate(t, tc.f)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

// TestSmallSampleIsFlaggedAndNeverJudged [REQ:STC-P0-043] proves the
// small-sample rule: fewer than min_samples yields small_sample=true and
// Compare answers insufficient_samples for every latency finding even when
// the observed values are far inside (or far outside) the budget.
func TestSmallSampleIsFlaggedAndNeverJudged(t *testing.T) {
	b := loadBudgets(t)
	min := b.Phase23.MinSamplesForPercentiles
	small := make([]time.Duration, min-1)
	for i := range small {
		small[i] = 5 * time.Second // wildly over budget
	}
	s := Summarise(small, 0, min)
	if !s.SmallSample || s.Samples != min-1 || s.P95Ms != 5000 {
		t.Fatalf("summary = %+v", s)
	}
	m := syntheticMeasurement("stateless-web", s)
	v, err := Compare(m, b)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range v.Findings {
		if strings.HasPrefix(f.Metric, "latency_") || f.Metric == "error_rate" {
			if f.Standing != StandingInsufficientSamples {
				t.Fatalf("%s/%s standing %s on %d samples; a small sample must never be judged", f.Phase, f.Metric, f.Standing, f.Samples)
			}
			if strings.HasPrefix(f.Metric, "latency_") && f.Observed == 0 {
				t.Fatalf("%s/%s observed value dropped from the finding", f.Phase, f.Metric)
			}
		}
	}
	if v.Standing != StandingInsufficientSamples {
		t.Fatalf("verdict standing = %s", v.Standing)
	}
}

// TestExceededVerdictNamesTheMetric [REQ:STC-P0-043] proves an adequate
// sample over budget is exceeded and names phase and metric; the same
// sample inside budget is within.
func TestExceededVerdictNamesTheMetric(t *testing.T) {
	b := loadBudgets(t)
	min := b.Phase23.MinSamplesForPercentiles
	slow := make([]time.Duration, min)
	for i := range slow {
		slow[i] = 20 * time.Millisecond
	}
	slow[min-2], slow[min-1] = 3*time.Second, 3*time.Second // p99 tail over budget, p50/p95 fine
	s := Summarise(slow, 0, min)
	if s.SmallSample {
		t.Fatalf("%d samples flagged small", s.Samples)
	}
	v, err := Compare(syntheticMeasurement("headless-api", s), b)
	if err != nil {
		t.Fatal(err)
	}
	if v.Standing != StandingExceeded {
		t.Fatalf("standing = %s; findings %+v", v.Standing, v.Findings)
	}
	var exceeded []string
	for _, f := range v.Findings {
		if f.Standing == StandingExceeded {
			exceeded = append(exceeded, f.Phase+"/"+f.Metric)
		}
	}
	want := []string{"sustained/latency_p99_ms", "peak/latency_p99_ms"}
	if strings.Join(exceeded, ",") != strings.Join(want, ",") {
		t.Fatalf("exceeded = %v, want %v", exceeded, want)
	}

	fast := make([]time.Duration, min)
	for i := range fast {
		fast[i] = 20 * time.Millisecond
	}
	v, err = Compare(syntheticMeasurement("headless-api", Summarise(fast, 0, min)), b)
	if err != nil {
		t.Fatal(err)
	}
	if v.Standing != StandingWithin {
		t.Fatalf("within sample standing = %s; findings %+v", v.Standing, v.Findings)
	}
	// Error rate over budget on a full sample is exceeded on its own.
	v, err = Compare(syntheticMeasurement("headless-api", Summarise(fast, min/10, min)), b)
	if err != nil {
		t.Fatal(err)
	}
	if v.Standing != StandingExceeded {
		t.Fatalf("10%% errors standing = %s", v.Standing)
	}
}

// TestCompareRefusesIncompleteMeasurement proves a measurement missing the
// post_cleanup phase cannot be compared (P23-A02 needs the delta).
func TestCompareRefusesIncompleteMeasurement(t *testing.T) {
	b := loadBudgets(t)
	m := syntheticMeasurement("stateless-web", Summarise(nil, 0, 100))
	delete(m.Phases, PhasePostCleanup)
	if _, err := Compare(m, b); err == nil || !strings.Contains(err.Error(), "post_cleanup") {
		t.Fatalf("err = %v", err)
	}
	if _, err := Compare(syntheticMeasurement("no-such-profile", Summarise(nil, 0, 100)), b); err == nil {
		t.Fatalf("unknown profile accepted")
	}
}

// TestHarnessAgainstHandlerYieldsCompleteMeasurement [REQ:STC-P0-043]
// proves the in-process harness records all four phases, latency samples
// with counts, and management-process snapshots from runtime + /proc/self,
// and that the record round-trips as JSON evidence.
func TestHarnessAgainstHandlerYieldsCompleteMeasurement(t *testing.T) {
	b := loadBudgets(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/api/v1/broken" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	h := Harness{
		Profile:     "stateless-web",
		Paths:       []string{"/api/v1/deployments", "/api/v1/deployments/dep-1", "/api/v1/broken"},
		Concurrency: 4,
		Sustained:   150 * time.Millisecond,
		Peak:        150 * time.Millisecond,
	}
	m, err := h.Run(context.Background(), Target{Handler: handler}, b)
	if err != nil {
		t.Fatal(err)
	}
	if missing := m.Complete(); len(missing) > 0 {
		t.Fatalf("incomplete: %v", missing)
	}
	if m.Target.Kind != "handler" || m.Concurrency != 4 {
		t.Fatalf("target/concurrency = %+v / %d", m.Target, m.Concurrency)
	}
	for _, name := range []string{PhaseSustained, PhasePeak} {
		ph := m.Phases[name]
		if ph.Latency.Samples == 0 || ph.Latency.Errors == 0 {
			t.Fatalf("%s recorded no samples or no errors: %+v", name, ph.Latency)
		}
		if ph.Latency.MinSamples != b.Phase23.MinSamplesForPercentiles {
			t.Fatalf("%s min_samples not carried: %+v", name, ph.Latency)
		}
		if ph.Latency.P50Ms > ph.Latency.P95Ms || ph.Latency.P95Ms > ph.Latency.P99Ms || ph.Latency.P99Ms > ph.Latency.MaxMs {
			t.Fatalf("%s percentiles not monotonic: %+v", name, ph.Latency)
		}
		want := float64(ph.Latency.Errors) / float64(ph.Latency.Samples)
		if ph.Latency.ErrorRate != want {
			t.Fatalf("%s error rate %.4f != %.4f", name, ph.Latency.ErrorRate, want)
		}
		if ph.Process.Goroutines == 0 || ph.Process.HeapAllocBytes == 0 {
			t.Fatalf("%s process snapshot empty: %+v", name, ph.Process)
		}
	}
	base := m.Phases[PhaseBaseline]
	if base.Latency.Samples != 0 || !base.Latency.SmallSample {
		t.Fatalf("baseline is a quiet phase: %+v", base.Latency)
	}
	if base.Process.Available && base.Process.RSSBytes == 0 {
		t.Fatalf("/proc readable but RSS zero: %+v", base.Process)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back Measurement
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Complete()) != 0 || back.Phases[PhasePeak].Latency.Samples != m.Phases[PhasePeak].Latency.Samples {
		t.Fatalf("measurement did not round-trip")
	}
	// The measurement compares (small sample expected at this duration).
	v, err := Compare(m, b)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	for _, name := range Phases() {
		ph := m.Phases[name]
		t.Logf("%s: samples=%d errors=%d p50=%.2fms p95=%.2fms p99=%.2fms small_sample=%v goroutines=%d rss=%.1fMiB fds=%d proc=%v",
			name, ph.Latency.Samples, ph.Latency.Errors, ph.Latency.P50Ms, ph.Latency.P95Ms, ph.Latency.P99Ms, ph.Latency.SmallSample, ph.Process.Goroutines, ph.Process.RSSMiB(), ph.Process.OpenFDs, ph.Process.Available)
	}
	t.Logf("verdict standing=%s", v.Standing)
}

// TestHarnessRefusesAnImplicitURLTarget [REQ:STC-P0-043] proves the live
// pass cannot default to any host: an empty --target and a non-explicit
// url target are refused before a request is sent.
func TestHarnessRefusesAnImplicitURLTarget(t *testing.T) {
	b := loadBudgets(t)
	h := Harness{Profile: "stateless-web", Paths: []string{"/api/v1/deployments"}}
	if _, err := h.Run(context.Background(), Target{}, b); !errors.Is(err, ErrTargetRequired) {
		t.Fatalf("empty target err = %v", err)
	}
	if _, err := h.Run(context.Background(), Target{BaseURL: "http://127.0.0.1:1"}, b); !errors.Is(err, ErrTargetRequired) {
		t.Fatalf("non-explicit url err = %v", err)
	}
	if _, err := ParseArgs([]string{"--profile", "stateless-web"}); !errors.Is(err, ErrTargetRequired) {
		t.Fatalf("missing --target err = %v", err)
	}
	if _, err := ParseArgs([]string{"--target", "vrooli.com", "--profile", "stateless-web"}); err == nil {
		t.Fatalf("relative target accepted")
	}
	a, err := ParseArgs([]string{"--target", "http://127.0.0.1:15672/", "--profile", "headless-api", "--path", "/api/v1/deployments", "--path", "/api/v1/operations/op-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Target.Explicit || a.Target.BaseURL != "http://127.0.0.1:15672" || len(a.Paths) != 2 {
		t.Fatalf("args = %+v", a)
	}
	// A placeholder path is refused: the harness never guesses an id.
	h.Paths = []string{"/api/v1/deployments/{id}"}
	if _, err := h.Run(context.Background(), Target{Handler: http.NotFoundHandler()}, b); err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("placeholder err = %v", err)
	}
}

// syntheticMeasurement builds a complete measurement whose load phases carry
// the given latency summary and whose process snapshots are inside the idle
// budget with zero growth.
func syntheticMeasurement(profile string, lat LatencySummary) *Measurement {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	snap := ProcessSnapshot{TakenAt: now, Goroutines: 20, HeapAllocBytes: 8 << 20, RSSBytes: 64 << 20, OpenFDs: 12, Available: true}
	m := &Measurement{SchemaVersion: 1, Profile: profile, Target: TargetDescriptor{Kind: "handler", Explicit: true}, StartedAt: now, FinishedAt: now.Add(time.Minute), Concurrency: 20, Phases: map[string]*Phase{}}
	for _, name := range Phases() {
		ph := &Phase{Name: name, StartedAt: now, FinishedAt: now.Add(10 * time.Second), ProcessAt: snap, Process: snap}
		if name == PhaseSustained || name == PhasePeak {
			ph.Latency = lat
		} else {
			ph.Latency = Summarise(nil, 0, lat.MinSamples)
		}
		m.Phases[name] = ph
	}
	return m
}
