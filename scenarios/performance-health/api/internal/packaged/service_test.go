package packaged

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	packagedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/packaged"
	_ "modernc.org/sqlite"
)

func TestPackagedExperimentV1ValidatesCompleteAttributableMatrix(t *testing.T) {
	manifest := validExperimentManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	manifest.Subjects[0].Cells = manifest.Subjects[0].Cells[:5]
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "six cells") {
		t.Fatalf("incomplete matrix error=%v", err)
	}
}

func TestPackagedExperimentV1RejectsInvalidDimensionsActionably(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ExperimentManifest)
		want string
	}{
		{"platform", func(m *ExperimentManifest) { m.Platform = "windows-x64" }, "platform must be"},
		{"sample count", func(m *ExperimentManifest) { m.SampleCount = 9 }, "sample_count must be 10"},
		{"modes", func(m *ExperimentManifest) { m.Modes = []string{"cold", "cold"} }, "modes must contain"},
		{"subject count", func(m *ExperimentManifest) { m.Subjects = m.Subjects[:3] }, "all four reference scenarios"},
		{"missing journey", func(m *ExperimentManifest) { m.Subjects[0].JourneyID = " " }, "requires journey_id"},
		{"duplicate cell", func(m *ExperimentManifest) { m.Subjects[0].Cells[5] = m.Subjects[0].Cells[4] }, "duplicates cell"},
		{"missing artifact identity", func(m *ExperimentManifest) { m.Subjects[0].Cells[0].ArtifactDigest = "bad" }, "sha256 digest"},
		{"artifact without digest", func(m *ExperimentManifest) { m.Subjects[0].Cells[0].ArtifactDigest = "" }, "requires an artifact_path and sha256 digest together"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validExperimentManifest()
			test.edit(&manifest)
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error=%v, want actionable %q", err, test.want)
			}
		})
	}
}

func TestExperimentManifestAllowsRampBuiltArtifactsToBeUnresolved(t *testing.T) {
	manifest := validExperimentManifest()
	for i := range manifest.Subjects {
		for j := range manifest.Subjects[i].Cells {
			manifest.Subjects[i].Cells[j].ArtifactPath = ""
			manifest.Subjects[i].Cells[j].ArtifactDigest = ""
		}
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("matrix requiring ramp builds rejected before artifacts exist: %v", err)
	}
}

func validExperimentManifest() ExperimentManifest {
	digest := "sha256:" + strings.Repeat("a", 64)
	cells := []ExperimentCell{
		{Strategy: "electron-sidecar", ArtifactPath: "/artifacts/sidecar.AppImage", ArtifactDigest: digest},
		{Strategy: "electron-optimized", Toggles: OptimizationToggles{true, true, true, true}, ArtifactPath: "/artifacts/all.AppImage", ArtifactDigest: digest},
		{Strategy: "electron-optimized", Toggles: OptimizationToggles{SharedRuntimeReadiness: true}, ArtifactPath: "/artifacts/readiness.AppImage", ArtifactDigest: digest},
		{Strategy: "electron-optimized", Toggles: OptimizationToggles{SecureCustomSchemeUI: true}, ArtifactPath: "/artifacts/scheme.AppImage", ArtifactDigest: digest},
		{Strategy: "electron-optimized", Toggles: OptimizationToggles{V8SnapshotCodeCache: true}, ArtifactPath: "/artifacts/cache.AppImage", ArtifactDigest: digest},
		{Strategy: "electron-optimized", Toggles: OptimizationToggles{ElectronFuses: true}, ArtifactPath: "/artifacts/fuses.AppImage", ArtifactDigest: digest},
	}
	return ExperimentManifest{SchemaVersion: ExperimentSchemaVersion, ExperimentID: "e3-linux-first", Platform: "linux-x64", SampleCount: 10, Modes: []string{"cold", "warm"}, Environment: EnvironmentMetadata{HostFingerprint: "host:abc", Display: "xvfb", ProfilerMode: "none", PowerState: "ac"}, Subjects: []ExperimentSubject{
		{Scenario: "hello-desktop", JourneyID: "hello-reference", Cells: cells},
		{Scenario: "system-monitor", JourneyID: "system-monitor-reference", Cells: cells},
		{Scenario: "web-console", JourneyID: "web-console-reference", Cells: cells},
		{Scenario: "landing-page-business-suite", JourneyID: "lpbs-reference", Cells: cells},
	}}
}

func TestPackagedExperimentV1UsesCurrentDesktopReferenceSubjects(t *testing.T) {
	manifest := validExperimentManifest()
	got := make([]string, 0, len(manifest.Subjects))
	for _, subject := range manifest.Subjects {
		got = append(got, subject.Scenario)
	}
	want := []string{"hello-desktop", "system-monitor", "web-console", "landing-page-business-suite"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("reference subjects=%v, want %v", got, want)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("current TARGETS.md subject set rejected: %v", err)
	}
	for _, retired := range []string{"secrets-manager", "git-control-tower"} {
		t.Run("rejects retired "+retired, func(t *testing.T) {
			candidate := validExperimentManifest()
			candidate.Subjects[1].Scenario = retired
			if err := candidate.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported subject") {
				t.Fatalf("Validate() error=%v, want retired reference rejected", err)
			}
		})
	}
}

func TestPackagedExperimentV1RejectsUnattributableInput(t *testing.T) {
	manifest := ExperimentManifest{SchemaVersion: ExperimentSchemaVersion, ExperimentID: "e", Platform: "linux-x64", SampleCount: 10, Modes: []string{"cold", "warm"}, Environment: EnvironmentMetadata{HostFingerprint: "h", Display: "x", ProfilerMode: "none", PowerState: "ac"}, Subjects: []ExperimentSubject{{Scenario: "unknown", JourneyID: "j"}, {}, {}, {}}}
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "unsupported subject") {
		t.Fatalf("subject error=%v", err)
	}
}

func TestExperimentValidateHandlerReturnsActionableInputError(t *testing.T) {
	handler := NewHandler(NewService(nil, nil, ""))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/packaged/experiment/validate", strings.NewReader(`{"schema_version":"wrong"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "schema_version must be packaged-experiment.v1") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestMarkdownComparisonLabelsUnavailableEvidence(t *testing.T) {
	report := Report{SchemaVersion: "packaged-comparison.v1", Cells: []ReportCell{{Scenario: "demo", Strategy: "electron-sidecar", Platform: "linux-x64", Condition: "cold", Samples: 10, JourneyParity: "unavailable: no journey replay receipt", Reliability: ReliabilityEvidence{Attempts: 10, Successful: 10, Status: "partial: failed launch attempts are not retained"}, ArtifactDigest: "sha256:" + strings.Repeat("a", 64), AttributionStatus: "known: electron-sidecar baseline disables E3 optimization toggles", Environment: EnvironmentMetadata{HostFingerprint: "host:sample", Display: "xvfb", ProfilerMode: "none", PowerState: "ac"}, Metrics: map[string]MetricSummary{}}}}
	markdown := RenderMarkdown(report)
	for _, expected := range []string{"unavailable: no journey replay receipt", "partial: failed launch attempts are not retained", "sha256:", "host:sample", "known: electron-sidecar"} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("markdown omits %q: %s", expected, markdown)
		}
	}
}

func TestHTTPLauncherResolvesScenarioToDesktopPort(t *testing.T) {
	want := Sample{SchemaVersion: SchemaVersion, SampleID: "remote", Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-sidecar", Platform: "linux", CapturedAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC), ExperimentID: "exp", HostFingerprint: "host", ArtifactDigest: "sha256:abc", Condition: "cold", Display: "xvfb", ProfilerMode: "disabled", PowerState: "unknown", Marks: []Mark{{Name: "app_interactive", ElapsedNs: 1_000_000, Availability: Availability{State: "available", Reason: "observed"}}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/measured-launch" {
			t.Errorf("request path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	launcher := &HTTPLauncher{Port: func() string { return port }}
	got, err := launcher.Launch(context.Background(), CaptureRequest{Scenario: "hello-desktop", Condition: "cold"})
	if err != nil || got.SampleID != want.SampleID {
		t.Fatalf("Launch() = %+v, %v", got, err)
	}
}

type fakeLauncher struct{ calls int }

func (f *fakeLauncher) Launch(_ context.Context, req CaptureRequest) (Sample, error) {
	f.calls++
	return Sample{SchemaVersion: SchemaVersion, SampleID: fmt.Sprintf("sample-%d", f.calls), Scenario: req.Scenario, Ramp: req.Ramp, Strategy: req.Strategy, Platform: "linux", CapturedAt: time.Date(2026, 10, 7, 12, 0, f.calls, 0, time.UTC), ExperimentID: req.Experiment, HostFingerprint: "host", ArtifactDigest: "sha256:abc", Condition: req.Condition, Display: "xvfb", ProfilerMode: "disabled", PowerState: "unknown", Marks: []Mark{{Name: "app_interactive", ElapsedNs: int64(f.calls) * 1_000_000, Availability: Availability{State: "available", Reason: "observed"}}}, Metrics: []Metric{{Name: "tree_pss_mb", Value: float64(f.calls), Unit: "MB", Availability: Availability{State: "available", Reason: "observed"}}}}, nil
}

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return NewStore(db)
}

func TestCaptureDiscardsWarmupAndPersistsOnlyRequestedRuns(t *testing.T) {
	store := testStore(t)
	launcher := &fakeLauncher{}
	service := NewService(store, launcher, "")
	request := CaptureRequest{Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-sidecar", Runs: 2, Condition: "cold", Artifact: "/tmp/app.AppImage", Experiment: "exp-a"}
	samples, err := service.Capture(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if launcher.calls != 3 || len(samples) != 2 {
		t.Fatalf("calls=%d samples=%d", launcher.calls, len(samples))
	}
	stored, err := store.List(context.Background(), "hello-desktop", "desktop", "electron-sidecar", "cold", "exp-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].SampleID == "sample-1" {
		t.Fatalf("stored samples include warmup or wrong count: %+v", stored)
	}
	attempts, err := store.ListAttempts(context.Background(), "hello-desktop", "desktop", "electron-sidecar", "cold", "exp-a")
	if err != nil || len(attempts) != 2 || !attempts[0].Succeeded || !attempts[1].Succeeded {
		t.Fatalf("attempt evidence = %+v, err=%v", attempts, err)
	}
}

func TestReportKeepsDifferentArtifactsInSeparateComparableGroups(t *testing.T) {
	store := testStore(t)
	for i, digest := range []string{"sha256:a", "sha256:b"} {
		sample := Sample{SchemaVersion: SchemaVersion, SampleID: fmt.Sprintf("s%d", i), Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-sidecar", Platform: "linux", CapturedAt: time.Date(2026, 10, 7, 12, 0, i, 0, time.UTC), ExperimentID: "exp", HostFingerprint: "host", ArtifactDigest: digest, Condition: "cold", Display: "xvfb", ProfilerMode: "disabled", PowerState: "unknown", Marks: []Mark{{Name: "app_interactive", ElapsedNs: int64(i+1) * 1_000_000, Availability: Availability{State: "available", Reason: "observed"}}}}
		if err := store.Insert(context.Background(), sample); err != nil {
			t.Fatal(err)
		}
	}
	report, err := NewService(store, &fakeLauncher{}, "").Report(context.Background(), "hello-desktop", "desktop", "electron-sidecar", "cold", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cells) != 2 || !report.Cells[0].Comparable || !report.Cells[1].Comparable {
		t.Fatalf("report merged non-comparable artifacts: %+v", report)
	}
}

func TestReportJSONUsesCellCollection(t *testing.T) {
	encoded, err := json.Marshal(Report{SchemaVersion: "packaged-comparison.v1", Cells: []ReportCell{{Scenario: "hello-desktop", Condition: "cold"}}})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["cells"]; !ok {
		t.Fatalf("report JSON missing cells: %s", encoded)
	}
	if _, ok := payload["groups"]; ok {
		t.Fatalf("report JSON still exposes groups: %s", encoded)
	}
}

func TestReportSeparatesEachOptimizationToggleCell(t *testing.T) {
	store := testStore(t)
	profiles := []OptimizationToggles{{SharedRuntimeReadiness: true}, {SecureCustomSchemeUI: true}}
	for i, profile := range profiles {
		sample := Sample{SchemaVersion: SchemaVersion, SampleID: fmt.Sprintf("toggle-%d", i), Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-optimized", ToggleAttribution: profile, Platform: "linux-x64", CapturedAt: time.Date(2026, 10, 7, 12, 0, i, 0, time.UTC), ExperimentID: "exp-toggle", HostFingerprint: "host", ArtifactDigest: "sha256:" + strings.Repeat("a", 64), Condition: "cold", Display: "xvfb", ProfilerMode: "disabled", PowerState: "ac", Marks: []Mark{{Name: "app_interactive", ElapsedNs: int64(i+1) * 1_000_000, Availability: Availability{State: "available", Reason: "observed"}}}}
		if err := store.Insert(context.Background(), sample); err != nil {
			t.Fatal(err)
		}
	}
	report, err := NewService(store, &fakeLauncher{}, "").Report(context.Background(), "hello-desktop", "desktop", "electron-optimized", "cold", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cells) != 2 || report.Cells[0].ToggleAttribution == report.Cells[1].ToggleAttribution {
		t.Fatalf("toggle profiles were merged into comparison groups: %+v", report.Cells)
	}
}

func TestNearestRankUsesCeilingRank(t *testing.T) {
	values := []float64{10, 20, 30, 40}
	if got := nearest(values, .95); got != 40 {
		t.Fatalf("p95=%v, want nearest rank max", got)
	}
}

func TestReportCarriesJourneyReplayReceiptsAndDisposition(t *testing.T) {
	store := testStore(t)
	sample := Sample{SchemaVersion: SchemaVersion, SampleID: "journey-sample", Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-sidecar", Platform: "linux-x64", CapturedAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC), ExperimentID: "exp-journey", HostFingerprint: "host", ArtifactDigest: "sha256:" + strings.Repeat("b", 64), Condition: "cold", Display: "xvfb", ProfilerMode: "disabled", PowerState: "ac", JourneyEvidence: &JourneyEvidence{JourneyID: "journey-1", CaptureID: "capture-1", Disposition: "pass"}}
	if err := store.Insert(context.Background(), sample); err != nil {
		t.Fatal(err)
	}
	report, err := NewService(store, &fakeLauncher{}, "").Report(context.Background(), "hello-desktop", "desktop", "electron-sidecar", "cold", "exp-journey")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cells) != 1 || report.Cells[0].JourneyParity != "passed: 1 of 1 journey replays" || len(report.Cells[0].JourneyReceipts) != 1 || report.Cells[0].JourneyReceipts[0].CaptureID != "capture-1" {
		t.Fatalf("journey evidence was not reported: %+v", report.Cells)
	}
}

func TestReportRPCReturnsJSONAndMarkdownWithCellEvidence(t *testing.T) {
	store := testStore(t)
	toggles := OptimizationToggles{SecureCustomSchemeUI: true}
	sample := Sample{
		SchemaVersion: SchemaVersion, SampleID: "rpc-report-sample", Scenario: "hello-desktop", Ramp: "desktop",
		Strategy: "electron-optimized", ToggleAttribution: toggles, Platform: "linux-x64",
		CapturedAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC), ExperimentID: "rpc-report-exp",
		HostFingerprint: "host:fingerprint", ArtifactDigest: "sha256:" + strings.Repeat("c", 64),
		Condition: "cold", Display: "xvfb", ProfilerMode: "none", PowerState: "ac",
		Marks:           []Mark{{Name: "app_interactive", ElapsedNs: 25_000_000, Availability: Availability{State: "available", Reason: "observed"}}},
		JourneyEvidence: &JourneyEvidence{JourneyID: "hello-journey", CaptureID: "capture-1", Disposition: "pass"},
	}
	if err := store.Insert(context.Background(), sample); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordAttempt(context.Background(), LaunchAttempt{Scenario: sample.Scenario, Ramp: sample.Ramp, Strategy: sample.Strategy, Experiment: sample.ExperimentID, Condition: sample.Condition, Toggles: toggles, Attempted: sample.CapturedAt, Succeeded: true}); err != nil {
		t.Fatal(err)
	}

	handler := &RPCHandler{service: NewService(store, &fakeLauncher{}, "")}
	response, err := handler.Report(context.Background(), connect.NewRequest(&packagedv1.ReportRequest{
		QueryJson: `{"experiment_id":"rpc-report-exp"}`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal([]byte(response.Msg.GetReportJson()), &report); err != nil {
		t.Fatalf("invalid report JSON: %v", err)
	}
	if len(report.Cells) != 1 {
		t.Fatalf("report groups=%d, want one: %+v", len(report.Cells), report.Cells)
	}
	group := report.Cells[0]
	if group.Metrics["app_interactive_ms"].P50 == nil || *group.Metrics["app_interactive_ms"].P50 != 25 {
		t.Fatalf("p50 missing or incorrect: %+v", group.Metrics["app_interactive_ms"])
	}
	if !group.ToggleAttribution.SecureCustomSchemeUI || group.ArtifactDigest != sample.ArtifactDigest || group.Environment.HostFingerprint != sample.HostFingerprint {
		t.Fatalf("cell attribution/provenance missing: %+v", group)
	}
	if group.JourneyParity != "passed: 1 of 1 journey replays" || group.Reliability.Status != "recorded: 1 of 1 launches succeeded" {
		t.Fatalf("parity/reliability missing: %+v", group)
	}
	for _, expected := range []string{"false / true / false / false", "25.000 ms", "passed: 1 of 1 journey replays", "recorded: 1 of 1 launches succeeded", sample.ArtifactDigest, sample.HostFingerprint} {
		if !strings.Contains(response.Msg.GetMarkdown(), expected) {
			t.Fatalf("Markdown omits %q: %s", expected, response.Msg.GetMarkdown())
		}
	}
}

func TestReportJSONPreservesMissingDataReasons(t *testing.T) {
	store := testStore(t)
	sample := Sample{
		SchemaVersion: SchemaVersion, SampleID: "missing-evidence", Scenario: "hello-desktop", Ramp: "desktop",
		Strategy: "electron-sidecar", Platform: "linux-x64", CapturedAt: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		ExperimentID: "missing-evidence-exp", Condition: "cold",
	}
	if err := store.Insert(context.Background(), sample); err != nil {
		t.Fatal(err)
	}
	report, err := NewService(store, &fakeLauncher{}, "").Report(context.Background(), "", "", "", "", sample.ExperimentID)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"missing_data"`) || !strings.Contains(string(encoded), "journey parity evidence unavailable") || !strings.Contains(string(encoded), "incomplete environment or artifact provenance") {
		t.Fatalf("report JSON obscured missing evidence: %s", encoded)
	}
}

type failMeasurementLauncher struct{ calls int }

func (f *failMeasurementLauncher) Launch(_ context.Context, request CaptureRequest) (Sample, error) {
	f.calls++
	if f.calls == 2 {
		return Sample{}, fmt.Errorf("simulated measured launch failure")
	}
	return (&fakeLauncher{calls: f.calls - 1}).Launch(context.Background(), request)
}

func TestReportPreservesFailedLaunchAttemptWithoutSample(t *testing.T) {
	store := testStore(t)
	service := NewService(store, &failMeasurementLauncher{}, "")
	_, err := service.Capture(context.Background(), CaptureRequest{Scenario: "hello-desktop", Ramp: "desktop", Strategy: "electron-sidecar", Runs: 2, Condition: "cold", Artifact: "/tmp/app.AppImage", Experiment: "exp-fail"})
	if err == nil {
		t.Fatal("Capture unexpectedly succeeded")
	}
	report, err := service.Report(context.Background(), "hello-desktop", "desktop", "electron-sidecar", "cold", "exp-fail")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Cells) != 1 {
		t.Fatalf("failed-only cell missing from report: %+v", report.Cells)
	}
	group := report.Cells[0]
	if group.Samples != 0 || group.Reliability.Attempts != 1 || group.Reliability.Failed != 1 || len(group.Reliability.Failures) != 1 || !strings.Contains(group.Reliability.Failures[0], "simulated") {
		t.Fatalf("failed launch evidence incomplete: %+v", group)
	}
}

func TestJourneySummaryKeepsMissingEvidenceExplicit(t *testing.T) {
	status, receipts := summarizeJourneyEvidence([]Sample{{}, {JourneyEvidence: &JourneyEvidence{JourneyID: "journey", CaptureID: "capture", Disposition: "pass"}}})
	if status != "incomplete: 1 of 2 launches lack journey evidence" || len(receipts) != 1 {
		t.Fatalf("status=%q receipts=%+v", status, receipts)
	}
}
