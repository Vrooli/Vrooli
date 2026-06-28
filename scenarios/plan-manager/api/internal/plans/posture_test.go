package plans

import (
	"context"
	"strings"
	"testing"
)

// fakeMaturityReader returns a fixed maturity per scenario for deterministic
// posture-derivation tests.
type fakeMaturityReader struct {
	maturity map[string]string // scenario -> maturity ("" means key present but empty)
	known    map[string]bool   // scenario -> service.json located
	err      error
}

func (f fakeMaturityReader) Maturity(_ context.Context, scenario string) (string, bool, error) {
	if f.err != nil {
		return "", false, f.err
	}
	if f.known != nil && !f.known[scenario] {
		return "", false, nil
	}
	m, ok := f.maturity[scenario]
	if !ok {
		// Unknown scenario: no service.json located.
		return "", false, nil
	}
	return m, true, nil
}

func planWithScenario(scenario string) Plan {
	return Plan{
		Title:      "Test plan",
		References: []Reference{{Kind: ReferenceCode, Target: "scenarios/" + scenario + "/api/main.go"}},
	}
}

func TestResolvePosture_MaturityMatrix(t *testing.T) {
	reader := fakeMaturityReader{maturity: map[string]string{
		"green-svc": maturityGreenfield,
		"pilot-svc": maturityPilot,
		"prod-svc":  maturityProduction,
		"sun-svc":   maturitySunset,
		"weird-svc": "experimental",
		"empty-svc": "",
	}}

	cases := []struct {
		name         string
		scenario     string
		wantPosture  WorkPosture
		wantSource   WorkPostureSource
		detailSubstr string
	}{
		{"greenfield", "green-svc", WorkPostureGreenfield, WorkPostureSourceServiceMaturity, "greenfield"},
		{"pilot", "pilot-svc", WorkPostureBrownfield, WorkPostureSourceServiceMaturity, "pilot"},
		{"production", "prod-svc", WorkPostureBrownfield, WorkPostureSourceServiceMaturity, "production"},
		{"sunset", "sun-svc", WorkPostureBrownfield, WorkPostureSourceServiceMaturity, "sunset"},
		{"unrecognized", "weird-svc", WorkPostureGreenfield, WorkPostureSourceServiceMaturity, "unrecognized"},
		{"absent-key", "empty-svc", WorkPostureGreenfield, WorkPostureSourceServiceMaturity, "greenfield"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			posture, source, detail := ResolvePosture(context.Background(), planWithScenario(tc.scenario), reader)
			if posture != tc.wantPosture {
				t.Fatalf("posture = %q, want %q", posture, tc.wantPosture)
			}
			if source != tc.wantSource {
				t.Fatalf("source = %q, want %q", source, tc.wantSource)
			}
			if !strings.Contains(strings.ToLower(detail), tc.detailSubstr) {
				t.Fatalf("detail %q does not contain %q", detail, tc.detailSubstr)
			}
		})
	}
}

func TestResolvePosture_DefaultsGreenfield(t *testing.T) {
	reader := fakeMaturityReader{maturity: map[string]string{"x": maturityProduction}}

	// No associated scenario at all -> default greenfield.
	posture, source, _ := ResolvePosture(context.Background(), Plan{Title: "docs only"}, reader)
	if posture != WorkPostureGreenfield || source != WorkPostureSourceDefault {
		t.Fatalf("no-scenario => %q/%q, want greenfield/default", posture, source)
	}

	// Nil reader -> default greenfield even with a scenario signal.
	posture, source, _ = ResolvePosture(context.Background(), planWithScenario("x"), nil)
	if posture != WorkPostureGreenfield || source != WorkPostureSourceDefault {
		t.Fatalf("nil reader => %q/%q, want greenfield/default", posture, source)
	}

	// Scenario named but service.json not locatable -> default greenfield.
	posture, source, _ = ResolvePosture(context.Background(), planWithScenario("missing"), reader)
	if posture != WorkPostureGreenfield || source != WorkPostureSourceDefault {
		t.Fatalf("missing svc => %q/%q, want greenfield/default", posture, source)
	}
}

func TestResolvePosture_HonorsExplicitAndImport(t *testing.T) {
	reader := fakeMaturityReader{maturity: map[string]string{"prod-svc": maturityProduction}}

	override := planWithScenario("prod-svc")
	override.WorkPosture = WorkPostureGreenfield
	override.WorkPostureSource = WorkPostureSourceExplicitOverride
	override.WorkPostureDetail = "operator override"
	posture, source, detail := ResolvePosture(context.Background(), override, reader)
	if posture != WorkPostureGreenfield || source != WorkPostureSourceExplicitOverride || detail != "operator override" {
		t.Fatalf("explicit override not preserved: %q/%q/%q", posture, source, detail)
	}

	imported := planWithScenario("prod-svc")
	imported.WorkPosture = WorkPostureBrownfield
	imported.WorkPostureSource = WorkPostureSourceImportLegacy
	posture, source, _ = ResolvePosture(context.Background(), imported, reader)
	if posture != WorkPostureBrownfield || source != WorkPostureSourceImportLegacy {
		t.Fatalf("imported posture not preserved: %q/%q", posture, source)
	}
}

func TestScenarioForPlan_PrefersAnchorThenRefs(t *testing.T) {
	p := Plan{
		RegressionAnchor: RegressionAnchor{Scenario: "anchor-svc"},
		References:        []Reference{{Kind: ReferenceCode, Target: "scenarios/ref-svc/x.go"}},
	}
	if got := scenarioForPlan(p); got != "anchor-svc" {
		t.Fatalf("anchor should win: got %q", got)
	}

	p = Plan{Phases: []Phase{{References: []Reference{{Kind: ReferenceCode, Target: "scenarios/phase-svc/y.go"}}}}}
	if got := scenarioForPlan(p); got != "phase-svc" {
		t.Fatalf("phase ref should resolve: got %q", got)
	}

	if got := scenarioForPlan(Plan{}); got != "" {
		t.Fatalf("no signal should be empty: got %q", got)
	}
}

// TestPostureBlock_ExactGreenfieldSentence locks the exact rendered Greenfield
// block (with backticks preserved on the code-like tokens). This is a contract:
// the wizard must never author it and it must never drift.
func TestPostureBlock_ExactGreenfieldSentence(t *testing.T) {
	const want = "**This is greenfield work.** Do not include compatibility shims, " +
		"legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables."
	if got := PostureBlock(WorkPostureGreenfield); got != want {
		t.Fatalf("greenfield block drift:\n got: %q\nwant: %q", got, want)
	}
	if got := PostureBlock(WorkPostureUnspecified); got != want {
		t.Fatalf("unspecified posture must render greenfield block, got %q", got)
	}
	if got := PostureBlock(WorkPostureBrownfield); !strings.Contains(got, "Preserve external contracts") {
		t.Fatalf("brownfield block missing conservative note: %q", got)
	}
}

func TestParseMaturity(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want string
		ok   bool
	}{
		"greenfield":   {`{"maturity":"greenfield"}`, maturityGreenfield, true},
		"pilot":        {`{"maturity":"pilot"}`, maturityPilot, true},
		"absent":       {`{"name":"x"}`, "", true},
		"unrecognized": {`{"maturity":"weird"}`, "", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, ok, err := parseMaturity([]byte(tc.raw))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want || ok != tc.ok {
				t.Fatalf("parseMaturity(%s) = %q,%v want %q,%v", tc.raw, got, ok, tc.want, tc.ok)
			}
		})
	}
}
