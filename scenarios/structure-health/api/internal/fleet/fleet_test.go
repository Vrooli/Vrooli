package fleet

import (
	"context"
	"errors"
	"testing"

	"structure-health/internal/validation"
)

type fakeEngine struct {
	responses map[string]validation.Response
	errs      map[string]error
}

func (f fakeEngine) Validate(_ context.Context, req validation.Request) (validation.Response, error) {
	if err, ok := f.errs[req.Scenario]; ok {
		return validation.Response{}, err
	}
	return f.responses[req.Scenario], nil
}

type fakeLister struct{ names []string }

func (f fakeLister) Scenarios() ([]string, error) { return f.names, nil }

type fakeTargetLister struct{ targets []Target }

func (f fakeTargetLister) Scenarios() ([]string, error) { return nil, nil }
func (f fakeTargetLister) Targets() ([]Target, error)   { return f.targets, nil }

func resp(profile string, recognized bool, findings ...validation.Finding) validation.Response {
	return validation.Response{
		Profile:  validation.DetectedProfile{ID: profile, Recognized: recognized},
		Findings: findings,
		Surfaces: []validation.SurfaceReconcile{{Surface: "api", Declared: true}, {Surface: "ui", Declared: false}},
	}
}

// [REQ:SH-FLEET-001]
func TestScanAggregatesRollupAndDistributions(t *testing.T) {
	engine := fakeEngine{responses: map[string]validation.Response{
		"alpha": resp("react-vite-go", true,
			validation.Finding{Code: "FRESHNESS_CHECK_MISSING", Severity: "error", AutofixAvailable: true},
			validation.Finding{Code: "PROFILE_ENV_VALIDATION", Severity: "warning"},
		),
		"beta": resp("react-vite-go", true), // clean
		"gamma": resp("python-only", false,
			validation.Finding{Code: "PROFILE_ENV_VALIDATION", Severity: "warning", AutofixAvailable: true},
		),
	}}
	s := New(engine, fakeLister{names: []string{"gamma", "alpha", "beta"}})

	result, err := s.Scan(context.Background(), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if result.ScenarioCount != 3 {
		t.Fatalf("scenario count = %d, want 3", result.ScenarioCount)
	}
	// alpha has an error → not passing; beta + gamma pass (no errors).
	if result.PassingCount != 2 {
		t.Fatalf("passing count = %d, want 2", result.PassingCount)
	}
	if result.MissingFreshness != 1 {
		t.Fatalf("missing freshness = %d, want 1", result.MissingFreshness)
	}
	if result.AutofixableTotal != 2 {
		t.Fatalf("autofixable total = %d, want 2", result.AutofixableTotal)
	}
	// Entries are sorted alphabetically.
	if result.Entries[0].Scenario != "alpha" || result.Entries[2].Scenario != "gamma" {
		t.Fatalf("entries not sorted: %+v", result.Entries)
	}
	if !result.Entries[1].Passed { // beta
		t.Fatalf("beta should pass")
	}
	if got := result.Entries[0].Surfaces; len(got) != 1 || got[0] != "api" {
		t.Fatalf("alpha declared surfaces = %v, want [api]", got)
	}

	// Rule conformance: PROFILE_ENV_VALIDATION offends 2 scenarios, FRESHNESS 1.
	if len(result.RuleConformance) != 2 {
		t.Fatalf("rule conformance count = %d, want 2", len(result.RuleConformance))
	}
	top := result.RuleConformance[0]
	if top.Code != "PROFILE_ENV_VALIDATION" || top.OffendingScenarios != 2 || top.TotalFindings != 2 {
		t.Fatalf("unexpected top rule conformance: %+v", top)
	}
	if top.Autofixable != 1 {
		t.Fatalf("PROFILE_ENV_VALIDATION autofixable = %d, want 1", top.Autofixable)
	}

	// Profile distribution: react-vite-go=2, python-only=1.
	if result.ProfileDistribution[0].ProfileID != "react-vite-go" || result.ProfileDistribution[0].ScenarioCount != 2 {
		t.Fatalf("unexpected profile distribution: %+v", result.ProfileDistribution)
	}
	if result.ProfileDistribution[1].Recognized {
		t.Fatalf("python-only should be unrecognized")
	}
}

// [REQ:SH-FLEET-002]
func TestScanRecordsErrorsAndContinues(t *testing.T) {
	engine := fakeEngine{
		responses: map[string]validation.Response{"good": resp("react-vite-go", true)},
		errs:      map[string]error{"bad": errors.New("code-facts down")},
	}
	s := New(engine, nil)
	result, err := s.Scan(context.Background(), []string{"bad", "good"})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.ScenarioCount != 1 || len(result.Entries) != 1 {
		t.Fatalf("expected 1 graded scenario, got %d", result.ScenarioCount)
	}
	if len(result.Errors) != 1 || result.Errors[0].Scenario != "bad" {
		t.Fatalf("expected 1 scan error for bad, got %+v", result.Errors)
	}
}

// [REQ:SH-FLEET-001]
func TestScanRequiresEngine(t *testing.T) {
	var s *Scanner
	if _, err := s.Scan(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil scanner")
	}
	s2 := New(fakeEngine{}, nil)
	if _, err := s2.Scan(context.Background(), nil); err == nil {
		t.Fatal("expected error when no lister and no explicit scenarios")
	}
}

func TestScanEnumeratesAllTargetKinds(t *testing.T) {
	engine := targetEngine{}
	targets := []Target{
		{Kind: "project", ID: "repo", Root: "/repo"},
		{Kind: "scenario", ID: "demo", Root: "/repo/scenarios/demo"},
		{Kind: "resource", ID: "postgres", Root: "/repo/resources/postgres"},
		{Kind: "tool", ID: "ffmpeg", Root: "/repo/internal/tools/ffmpeg"},
		{Kind: "safeguard", ID: "tcp-tuning", Root: "/repo/internal/safeguards/tcp-tuning"},
		{Kind: "package", ID: "api-core", Root: "/repo/packages/api-core"},
		{Kind: "control-plane", ID: "internal", Root: "/repo/internal"},
		{Kind: "docs", ID: "docs", Root: "/repo/docs"},
		{Kind: "team", ID: "infra-health", Root: "/repo/docs/infra-health"},
	}
	s := New(engine, fakeTargetLister{targets: targets})
	result, err := s.Scan(context.Background(), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.TargetCount != len(targets) || result.ScenarioCount != len(targets) {
		t.Fatalf("target count = %d/%d, want %d", result.TargetCount, result.ScenarioCount, len(targets))
	}
	if len(result.Entries) != len(targets) {
		t.Fatalf("entries = %d, want %d", len(result.Entries), len(targets))
	}
	if result.Entries[0].TargetKind != "control-plane" || result.Entries[len(result.Entries)-1].TargetKind != "tool" {
		t.Fatalf("entries not sorted by target kind: %+v", result.Entries)
	}
}

type targetEngine struct{}

func (targetEngine) Validate(_ context.Context, req validation.Request) (validation.Response, error) {
	return resp("target", true), nil
}
