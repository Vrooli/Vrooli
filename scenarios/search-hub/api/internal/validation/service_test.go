package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	internaleval "search-hub/internal/eval"
)

func TestValidateScenarioCleanSearchDescriptor(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if report.Summary.Errors != 0 {
		t.Fatalf("errors = %d, findings = %#v", report.Summary.Errors, report.Findings)
	}
	if report.Summary.Providers != 1 {
		t.Fatalf("providers = %d, want 1", report.Summary.Providers)
	}
}

func TestValidateScenarioMissingSearchDescriptorFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireFinding(t, report, CodeConfigMissing)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioInvalidProviderFails(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "tests":{"cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireFinding(t, report, CodeProviderInvalid)
}

func TestValidateScenarioProviderGroupMismatchFailsDescriptor(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("other"))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireFinding(t, report, CodeProviderGroupMismatch)
	if finding.Location != ".vrooli/search.json:providers[0].provider_group" {
		t.Fatalf("location = %q, want provider_group path", finding.Location)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioMissingEvalCorpusFails(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireFinding(t, report, CodeEvalCorpusMissing)
}

func TestValidateScenarioEndpointWarningsRemainAdvisory(t *testing.T) {
	// A local_index provider that reindexes (required posture satisfied) but has
	// no status endpoint and no config write-back: both are advisory and must not
	// gate certification.
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_index",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "reindex_endpoint":{"http_json":{"scenario_id":"demo","path":"/reindex","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]},{"id":"neg","query":"zzqxwv nonsense","expect_no_strong_hit":true,"expect_max_score":0.2}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireFinding(t, report, CodeStatusEndpointMissing)
	requireFinding(t, report, CodeControlEndpointMissing)
	if report.Summary.Errors != 0 {
		t.Fatalf("errors = %d, findings = %#v", report.Summary.Errors, report.Findings)
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("status = %s, want passed", report.Summary.Status())
	}
}

func TestValidateScenarioActiveProviderRequiresClass(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireFinding(t, report, CodeProviderClassMissing)
	if finding.Severity != SeverityError {
		t.Fatalf("class-missing severity = %s, want error", finding.Severity)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIndexedProviderRequiresReindexEndpoint(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_index",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"demo","path":"/status","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireFinding(t, report, CodeReindexEndpointMissing)
	if finding.Severity != SeverityError {
		t.Fatalf("reindex-missing severity = %s, want error", finding.Severity)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioLocalLiveProviderNeedsNoControlPlane(t *testing.T) {
	// A computed-live provider with no reindex/config must pass: only a status
	// endpoint is advisory.
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.commands",
    "provider_group":"demo",
    "bucket":"BUCKET_DO",
    "type":"command",
    "description":"Commands",
    "scope":"SCOPE_PROJECT",
    "class":"local_live",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"demo","path":"/status","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_RAW"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"restart","expect_ids":["restart"]},{"id":"neg","query":"zzqxwv nonsense","expect_no_strong_hit":true,"expect_max_score":0.2}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeReindexEndpointMissing, CodeControlEndpointMissing, CodeProviderClassMissing} {
		if hasFinding(report, code) {
			t.Fatalf("local_live provider should not emit %s: %#v", code, report.Findings)
		}
	}
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0: %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
}

func TestValidateScenarioExternalProviderNeedsNoLocalPosture(t *testing.T) {
	// An external provider expects no local status/reindex/config endpoints.
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.web",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"web",
    "description":"Web results",
    "scope":"SCOPE_PROJECT",
    "class":"external",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_RAW"},
    "tests":{"description":"Primary corpus","cases":[{"id":"case","query":"news","expect_ids":["hit-1"]},{"id":"neg","query":"zzqxwv nonsense","expect_no_strong_hit":true,"expect_max_score":0.2}]}
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeStatusEndpointMissing, CodeReindexEndpointMissing, CodeControlEndpointMissing, CodeProviderClassMissing} {
		if hasFinding(report, code) {
			t.Fatalf("external provider should not emit %s: %#v", code, report.Findings)
		}
	}
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0: %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
}

func TestValidateScenarioCapabilityGapProviderIsExemptFromClassPosture(t *testing.T) {
	// A capability_gap stub is not routable and is exempt from class/control
	// posture checks.
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.future",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Planned corpus",
    "scope":"SCOPE_PROJECT",
    "state":"PROVIDER_STATE_CAPABILITY_GAP",
    "intended_home":"demo"
  }]
}`)

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeProviderClassMissing, CodeReindexEndpointMissing, CodeStatusEndpointMissing, CodeEvalCorpusMissing} {
		if hasFinding(report, code) {
			t.Fatalf("capability_gap stub should not emit %s: %#v", code, report.Findings)
		}
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("status = %s, want passed: %#v", report.Summary.Status(), report.Findings)
	}
}

func TestValidateScenarioDefaultDoesNotRequireEvalRunHistory(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	service := New(root)

	report, err := service.ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if report.Summary.Errors != 0 {
		t.Fatalf("errors = %d, findings = %#v", report.Summary.Errors, report.Findings)
	}
	if len(report.EvalEvidence) != 0 {
		t.Fatalf("default validation should not collect eval evidence: %#v", report.EvalEvidence)
	}
}

func TestValidateScenarioIncludeEvalsReportsUnavailableStoreSeparately(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))

	report, err := New(root).ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	finding := requireFinding(t, report, CodeEvalProviderUnavailable)
	if finding.Severity != SeverityError {
		t.Fatalf("severity = %s, want error", finding.Severity)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIncludeEvalsRequiresRunHistory(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	service := New(root)
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{
			"demo.docs.primary": evalSuite("demo.docs.primary", "demo.docs"),
		},
	}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodeEvalRunMissing)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIncludeEvalsReportsFreshPassingRun(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{
			"demo.docs.primary": evalSuite("demo.docs.primary", "demo.docs"),
		},
		runs: map[string][]*evalv1.EvalRun{
			"demo.docs.primary": {evalRun("run-fresh", "demo.docs.primary", now.Add(-time.Hour), "met")},
		},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 {
		t.Fatalf("unexpected findings: %#v", report.Findings)
	}
	if got := report.EvalEvidence[0].Freshness; got != "fresh" {
		t.Fatalf("freshness = %s, want fresh", got)
	}
}

func TestValidateScenarioIncludeEvalsFindsStaleAndRecallBelowTargetEvidence(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{
			"demo.docs.primary": evalSuite("demo.docs.primary", "demo.docs"),
		},
		runs: map[string][]*evalv1.EvalRun{
			// A stale run whose single reviewed positive missed → recall 0 < target.
			"demo.docs.primary": {evalRun("run-stale", "demo.docs.primary", now.Add(-48*time.Hour), "below_expectation")},
		},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodeEvalRecallBelowTarget)
	requireFinding(t, report, CodeEvalRunStale)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIncludeEvalsFailsOnJunkLeak(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	// Suite with one reviewed positive (met) and one negative that leaked.
	suite := &evalv1.EvalSuite{
		SuiteId: "demo.docs.primary", ProviderId: "demo.docs", State: "active",
		Cases: []*evalv1.EvalCase{
			{CaseId: "pos", Query: "docs", ExpectIds: []string{"doc-1"}, ExpectWithinTopK: 3},
			{CaseId: "neg", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
		},
	}
	run := &evalv1.EvalRun{
		RunId: "run-leak", SuiteId: "demo.docs.primary",
		CreatedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		Results: []*evalv1.CaseResult{
			{CaseId: "pos", Outcome: "met"},
			{CaseId: "neg", Outcome: "unexpected_hit"},
		},
	}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": suite},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodeEvalAssertFailed)
	// The positive met, so recall is 1.0 — no recall finding.
	if hasFinding(report, CodeEvalRecallBelowTarget) {
		t.Fatalf("recall should be 1.0 (positive met): %#v", report.Findings)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIncludeEvalsToleratesMissesWithinRecallTarget(t *testing.T) {
	// recall_target 0.5: one of two reviewed positives missing still certifies.
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", recallProviderConfig(`"scoring":{"recall_target":0.5},`))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	suite := &evalv1.EvalSuite{
		SuiteId: "demo.docs.primary", ProviderId: "demo.docs", State: "active",
		Cases: []*evalv1.EvalCase{
			{CaseId: "p1", Query: "q1", ExpectIds: []string{"a"}, ExpectWithinTopK: 3},
			{CaseId: "p2", Query: "q2", ExpectIds: []string{"b"}, ExpectWithinTopK: 3},
			{CaseId: "neg", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
		},
	}
	run := &evalv1.EvalRun{
		RunId: "run-half", SuiteId: "demo.docs.primary",
		CreatedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		Results: []*evalv1.CaseResult{
			{CaseId: "p1", Outcome: "met"},
			{CaseId: "p2", Outcome: "below_expectation"},
			{CaseId: "neg", Outcome: "met"},
		},
	}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": suite},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 2, Live: 2}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	if hasFinding(report, CodeEvalRecallBelowTarget) {
		t.Fatalf("recall 0.5 meets target 0.5, must not fail: %#v", report.Findings)
	}
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0: %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
	if got := report.EvalEvidence[0].Recall; got != 0.5 {
		t.Fatalf("evidence recall = %v, want 0.5", got)
	}
}

func TestValidateScenarioIncludeEvalsFailsOnTuningDrift(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	run := evalRun("run-drift", "demo.docs.primary", now.Add(-time.Hour), "met")
	// The run captured a different embed model than the declared (default) tuning.
	run.Config = &evalv1.ConfigSnapshot{EmbedModel: "some-other-model"}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": evalSuite("demo.docs.primary", "demo.docs")},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodeEvalRunOutdated)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioIncludeEvalsReportsStaleLiveLabels(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", cleanSearchConfig("demo"))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{
			"demo.docs.primary": evalSuite("demo.docs.primary", "demo.docs"),
		},
		runs: map[string][]*evalv1.EvalRun{
			"demo.docs.primary": {evalRun("run-fresh", "demo.docs.primary", now.Add(-time.Hour), "met")},
		},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 2, Live: 1, Stale: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	finding := requireFinding(t, report, CodeEvalLabelsStale)
	if finding.Severity != SeverityError {
		t.Fatalf("severity = %s, want error", finding.Severity)
	}
	if got := report.EvalEvidence[0].CorpusStatus; got != "stale" {
		t.Fatalf("corpus status = %s, want stale", got)
	}
}

func TestBuildMaturityAssessmentUsesSearchSpec(t *testing.T) {
	spec := mustLoadSpec(t)
	a, err := BuildMaturityAssessment("demo", []Finding{{
		Code:        CodeEvalCorpusMissing,
		Severity:    SeverityError,
		Title:       "Missing corpus",
		Message:     "No tests",
		Location:    ".vrooli/search.json",
		Remediation: "Add tests",
	}}, *spec)
	if err != nil {
		t.Fatalf("BuildMaturityAssessment: %v", err)
	}
	if a.GetProvider() != "search-hub" || a.GetPhase() != "search" {
		t.Fatalf("identity = %s/%s, want search-hub/search", a.GetProvider(), a.GetPhase())
	}
	if got := len(a.GetFindings()); got != 1 {
		t.Fatalf("findings = %d, want 1", got)
	}
	finding := a.GetFindings()[0]
	if finding.GetMaturity().GetCapabilityId() != "search_eval_performance" {
		t.Fatalf("capability = %q, want search_eval_performance", finding.GetMaturity().GetCapabilityId())
	}
	if finding.GetMaturity().GetCleanRequirement().String() != "CLEAN_REQUIREMENT_REQUIRED" {
		t.Fatalf("clean requirement = %s, want required", finding.GetMaturity().GetCleanRequirement())
	}
}

func TestPreviewFixesReportsMechanicalDescriptorRepairs(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tuning":{"rerank_enabled":true,"rerank_shortlist":999},
    "tests":{"suite_id":"external.primary","cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]},
    "unknown_field":"preserve me"
  }]
}`)

	scenario, candidates, err := New(root).PreviewFixes("demo", "", nil)
	if err != nil {
		t.Fatalf("PreviewFixes: %v", err)
	}
	if scenario != "demo" {
		t.Fatalf("scenario = %q, want demo", scenario)
	}
	if got, want := len(candidates), 3; got != want {
		t.Fatalf("candidates = %d, want %d: %#v", got, want, candidates)
	}
	last := candidates[len(candidates)-1].After
	for _, want := range []string{
		`"version": "1.0.0"`,
		`"suite_id": "demo.docs.primary"`,
		`"description": "Reviewed search eval corpus."`,
		`"rerank_shortlist": 250`,
		`"unknown_field": "preserve me"`,
	} {
		if !strings.Contains(last, want) {
			t.Fatalf("final preview missing %s:\n%s", want, last)
		}
	}
	readBack, err := os.ReadFile(filepath.Join(root, "scenarios", "demo", ".vrooli", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(readBack), `"1.0.0"`) {
		t.Fatal("PreviewFixes must not write descriptor changes")
	}
}

func TestApplyFixesIsIdempotentAndFilterable(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{
  "version":"",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tuning":{"rerank_enabled":true,"rerank_shortlist":999},
    "tests":{"cases":[{"id":"case","query":"docs","expect_ids":["doc-1"]}]}
  }]
}`)

	service := New(root)
	_, candidates, err := service.ApplyFixes("demo", "", []string{CodeEvalCorpusInvalid})
	if err != nil {
		t.Fatalf("ApplyFixes filtered: %v", err)
	}
	if got, want := len(candidates), 1; got != want {
		t.Fatalf("filtered candidates = %d, want %d: %#v", got, want, candidates)
	}
	applied, err := os.ReadFile(filepath.Join(root, "scenarios", "demo", ".vrooli", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(applied)
	if strings.Contains(text, `"version": "1.0.0"`) || strings.Contains(text, `"rerank_shortlist": 250`) {
		t.Fatalf("rule filter applied unrelated fixes:\n%s", text)
	}
	if !strings.Contains(text, `"suite_id": "demo.docs.primary"`) || !strings.Contains(text, `"description": "Reviewed search eval corpus."`) {
		t.Fatalf("filtered eval fix missing expected changes:\n%s", text)
	}

	_, again, err := service.ApplyFixes("demo", "", []string{CodeEvalCorpusInvalid})
	if err != nil {
		t.Fatalf("ApplyFixes second run: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second apply must be a no-op, got %#v", again)
	}
}

func TestPreviewFixesInvalidJSONHasNoCandidates(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", `{`)

	_, candidates, err := New(root).PreviewFixes("demo", "", nil)
	if err != nil {
		t.Fatalf("PreviewFixes invalid JSON: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("invalid JSON candidates = %#v, want none", candidates)
	}
}

func providerConfigWith(fields, testsJSON string) string {
	return `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_live",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"demo","path":"/status","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_RAW"},
    ` + fields + `
    "tests":` + testsJSON + `
  }]
}`
}

func TestValidateScenarioLatencyBudgetBreachIsAdvisory(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", providerConfigWith(`"performance":{"p95_ms":100},`,
		`{"description":"c","cases":[{"id":"p1","query":"q1","expect_ids":["a"]},{"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}]}`))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	suite := &evalv1.EvalSuite{
		SuiteId: "demo.docs.primary", ProviderId: "demo.docs", State: "active",
		Cases: []*evalv1.EvalCase{
			{CaseId: "p1", Query: "q1", ExpectIds: []string{"a"}, ExpectWithinTopK: 3},
			{CaseId: "neg", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
		},
	}
	run := &evalv1.EvalRun{
		RunId: "run-slow", SuiteId: "demo.docs.primary",
		CreatedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		Results: []*evalv1.CaseResult{
			{CaseId: "p1", Outcome: "met", Top: []*evalv1.ScoredHit{{Id: "a", Score: 0.9}}},
			{CaseId: "neg", Outcome: "met"},
		},
		Aggregate: &evalv1.EvalAggregate{Cases: 2, Met: 2, LatencyP95Ms: 500},
	}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": suite},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodePerfBudgetBreach)
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0 (breach is advisory): %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
	if report.EvalEvidence[0].LatencyP95Ms != 500 {
		t.Fatalf("evidence latency = %d, want 500", report.EvalEvidence[0].LatencyP95Ms)
	}
}

func TestValidateScenarioTelemetryRequiredFailsWithoutEvidence(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", providerConfigWith(`"performance":{"telemetry_required":true},`,
		`{"description":"c","cases":[{"id":"p1","query":"q1","expect_ids":["a"]},{"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}]}`))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	suite := &evalv1.EvalSuite{
		SuiteId: "demo.docs.primary", ProviderId: "demo.docs", State: "active",
		Cases: []*evalv1.EvalCase{
			{CaseId: "p1", Query: "q1", ExpectIds: []string{"a"}, ExpectWithinTopK: 3},
			{CaseId: "neg", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
		},
	}
	// Run with no aggregate ⇒ no latency evidence.
	run := &evalv1.EvalRun{
		RunId: "run-noagg", SuiteId: "demo.docs.primary",
		CreatedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		Results:   []*evalv1.CaseResult{{CaseId: "p1", Outcome: "met"}, {CaseId: "neg", Outcome: "met"}},
	}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": suite},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodePerfBudgetUnproven)
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioDegradationRateIsAdvisory(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", providerConfigWith(`"scoring":{"recall_target":0.5},"performance":{"degraded_rate_max":0.1},`,
		`{"description":"c","cases":[{"id":"p1","query":"q1","expect_ids":["a"]},{"id":"p2","query":"q2","expect_ids":["b"]},{"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}]}`))
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	service := New(root)
	service.Now = func() time.Time { return now }
	suite := &evalv1.EvalSuite{
		SuiteId: "demo.docs.primary", ProviderId: "demo.docs", State: "active",
		Cases: []*evalv1.EvalCase{
			{CaseId: "p1", Query: "q1", ExpectIds: []string{"a"}, ExpectWithinTopK: 3},
			{CaseId: "p2", Query: "q2", ExpectIds: []string{"b"}, ExpectWithinTopK: 3},
			{CaseId: "neg", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.2},
		},
	}
	run := &evalv1.EvalRun{
		RunId: "run-degraded", SuiteId: "demo.docs.primary",
		CreatedAt: now.Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		Results: []*evalv1.CaseResult{
			{CaseId: "p1", Outcome: "met", Top: []*evalv1.ScoredHit{{Id: "a", Score: 0.9}}},
			{CaseId: "p2", Outcome: "below_expectation"}, // empty Top ⇒ degraded
			{CaseId: "neg", Outcome: "met"},              // empty Top on a junk case is not degraded
		},
	}
	service.EvalStore = fakeEvalStore{
		suites: map[string]*evalv1.EvalSuite{"demo.docs.primary": suite},
		runs:   map[string][]*evalv1.EvalRun{"demo.docs.primary": {run}},
	}
	service.EvalValidator = fakeEvalValidator{rollup: &evalv1.CorpusValidationRollup{Positives: 2, Live: 2}}

	report, err := service.ValidateScenarioWithOptions(context.Background(), "demo", "", Options{IncludeEvals: true, EvalFreshnessWindow: 24 * time.Hour})
	if err != nil {
		t.Fatalf("ValidateScenarioWithOptions: %v", err)
	}
	requireFinding(t, report, CodePerfDegraded)
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0 (degraded is advisory): %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
}

func recallProviderConfig(scoringPrefix string) string {
	return `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_live",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"demo","path":"/status","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_RAW"},
    ` + scoringPrefix + `
    "tests":{"description":"c","cases":[
      {"id":"p1","query":"q1","expect_ids":["a"]},
      {"id":"p2","query":"q2","expect_ids":["b"]},
      {"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}
    ]}
  }]
}`
}

func liveProviderConfig(testsJSON string) string {
	return `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"demo.docs",
    "provider_group":"demo",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_live",
    "endpoint":{"http_json":{"scenario_id":"demo","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"demo","path":"/status","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_RAW"},
    "tests":` + testsJSON + `
  }]
}`
}

func TestValidateScenarioCorpusWithoutReviewedPositiveFails(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", liveProviderConfig(`{"description":"c","cases":[
    {"id":"c1","query":"generated q","status":"candidate","expect_ids":["a"]},
    {"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}
  ]}`))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireFinding(t, report, CodeEvalCorpusInadequate)
	if finding.Severity != SeverityError {
		t.Fatalf("severity = %s, want error", finding.Severity)
	}
	if !strings.Contains(finding.Message, "candidate/generated") {
		t.Fatalf("generated-only message = %q", finding.Message)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioCorpusWithoutNegativeFails(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", liveProviderConfig(`{"description":"c","cases":[
    {"id":"c1","query":"real q","expect_ids":["a"]}
  ]}`))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	finding := requireFinding(t, report, CodeEvalCorpusInadequate)
	if !strings.Contains(finding.Message, "junk-rejection") {
		t.Fatalf("no-negative message = %q", finding.Message)
	}
	if report.Summary.Status() != "failed" {
		t.Fatalf("status = %s, want failed", report.Summary.Status())
	}
}

func TestValidateScenarioDuplicateQueriesAreAdvisory(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", liveProviderConfig(`{"description":"c","cases":[
    {"id":"c1","query":"same query","expect_ids":["a"]},
    {"id":"c2","query":"same query","expect_ids":["b"]},
    {"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}
  ]}`))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	requireFinding(t, report, CodeEvalCorpusThin)
	if report.Summary.Errors != 0 || report.Summary.Status() != "passed" {
		t.Fatalf("status = %s errors = %d, want passed/0: %#v", report.Summary.Status(), report.Summary.Errors, report.Findings)
	}
}

func TestValidateScenarioDistinctScopeIsNotDuplicate(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", liveProviderConfig(`{"description":"c","cases":[
    {"id":"c1","query":"same query","scope":"scenario:a","expect_ids":["a"]},
    {"id":"c2","query":"same query","scope":"scenario:b","expect_ids":["b"]},
    {"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}
  ]}`))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if hasFinding(report, CodeEvalCorpusThin) {
		t.Fatalf("same query under distinct scopes must not be a duplicate: %#v", report.Findings)
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("status = %s, want passed", report.Summary.Status())
	}
}

func TestValidateScenarioAdequateCorpusPasses(t *testing.T) {
	root := t.TempDir()
	writeSearchConfig(t, root, "demo", liveProviderConfig(`{"description":"c","cases":[
    {"id":"c1","query":"real q","expect_ids":["a"]},
    {"id":"neg","query":"junk","expect_no_strong_hit":true,"expect_max_score":0.2}
  ]}`))

	report, err := New(root).ValidateScenario("demo", "")
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	for _, code := range []string{CodeEvalCorpusInadequate, CodeEvalCorpusThin} {
		if hasFinding(report, code) {
			t.Fatalf("adequate corpus should not emit %s: %#v", code, report.Findings)
		}
	}
	if report.Summary.Status() != "passed" {
		t.Fatalf("status = %s, want passed", report.Summary.Status())
	}
}

func hasFinding(report Report, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func requireFinding(t *testing.T, report Report, code string) Finding {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return finding
		}
	}
	t.Fatalf("missing finding %s in %#v", code, report.Findings)
	return Finding{}
}

type fakeEvalStore struct {
	suites map[string]*evalv1.EvalSuite
	runs   map[string][]*evalv1.EvalRun
	err    error
}

type fakeEvalValidator struct {
	rollup *evalv1.CorpusValidationRollup
	err    error
}

func (f fakeEvalValidator) ValidateCorpus(_ context.Context, suite *evalv1.EvalSuite, _ int32) (*evalv1.ValidateCorpusResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &evalv1.ValidateCorpusResponse{
		SuiteId:    suite.GetSuiteId(),
		ProviderId: suite.GetProviderId(),
		Rollup:     f.rollup,
	}, nil
}

func (f fakeEvalStore) GetSuite(_ context.Context, id string) (*evalv1.EvalSuite, error) {
	if f.err != nil {
		return nil, f.err
	}
	if suite, ok := f.suites[id]; ok {
		return suite, nil
	}
	return nil, internaleval.ErrSuiteNotFound{SuiteID: id}
}

func (f fakeEvalStore) ListRuns(_ context.Context, filter internaleval.ListRunsFilter) ([]*evalv1.EvalRun, error) {
	if f.err != nil {
		return nil, f.err
	}
	runs := append([]*evalv1.EvalRun(nil), f.runs[filter.SuiteID]...)
	if filter.Limit > 0 && len(runs) > filter.Limit {
		runs = runs[:filter.Limit]
	}
	return runs, nil
}

func evalSuite(suiteID, providerID string) *evalv1.EvalSuite {
	return &evalv1.EvalSuite{
		SuiteId:    suiteID,
		ProviderId: providerID,
		Name:       "Primary",
		State:      "active",
		Cases: []*evalv1.EvalCase{{
			CaseId:           "case",
			Query:            "docs",
			ExpectIds:        []string{"doc-1"},
			ExpectWithinTopK: 3,
		}},
	}
}

func evalRun(runID, suiteID string, createdAt time.Time, outcome string) *evalv1.EvalRun {
	return &evalv1.EvalRun{
		RunId:     runID,
		SuiteId:   suiteID,
		CreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
		Results: []*evalv1.CaseResult{{
			CaseId:  "case",
			Outcome: outcome,
		}},
	}
}

func writeSearchConfig(t *testing.T, root, scenario, content string) {
	t.Helper()
	dir := filepath.Join(root, "scenarios", scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "search.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cleanSearchConfig(scenario string) string {
	return `{
  "version":"1.0.0",
  "providers":[{
    "provider_id":"` + scenario + `.docs",
    "provider_group":"` + scenario + `",
    "bucket":"BUCKET_KNOW",
    "type":"doc",
    "description":"Docs",
    "scope":"SCOPE_PROJECT",
    "class":"local_index",
    "endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/search","method":"HTTP_METHOD_POST"}},
    "status_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/status","method":"HTTP_METHOD_POST"}},
    "reindex_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/reindex","method":"HTTP_METHOD_POST"}},
    "config_endpoint":{"http_json":{"scenario_id":"` + scenario + `","path":"/config","method":"HTTP_METHOD_POST"}},
    "result_mapping":{"results_path":"results","id_field":"id","title_field":"title","score_field":"score","score_scale":"SCORE_SCALE_COSINE_0_1"},
    "tuning":{"rerank_enabled":true,"rerank_shortlist":50},
    "tests":{"description":"Primary corpus","cases":[
      {"id":"case","query":"docs","expect_ids":["doc-1"]},
      {"id":"neg","query":"zzqxwv nonsense","expect_no_strong_hit":true,"expect_max_score":0.2}
    ]}
  }]
}`
}
