package report

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// renderCompleteness drives a full Print with the given ScoreRunner and
// returns the rendered report. A plain *bytes.Buffer is not a TTY, so color
// is disabled and assertions match raw substrings.
func renderCompleteness(t *testing.T, runner ScoreRunner) string {
	t.Helper()
	resp := baseResp()
	var buf bytes.Buffer
	pr := New(&buf, "demo-scenario", "", nil, nil, false, nil, nil)
	pr.SetScoreRunner(runner)
	pr.Print(resp)
	return buf.String()
}

const sampleScoreJSON = `{
  "scenario": "demo-scenario",
  "category": "utility",
  "maturity": {"workingRung": "R1 Safe & standards-clean", "satisfiedThrough": "R0 Runnable & green", "buildPassing": true},
  "composite": {"score": 82, "classification": "mostly_complete", "classificationLabel": "Mostly complete"},
  "trend": {"previousScore": 76, "previousCalculatedAt": "2026-06-08T12:00:00Z", "delta": 6},
  "freshness": {
    "currentDigest": "td:abc",
    "phases": [
      {"phase": "unit", "verdict": "fresh"},
      {"phase": "smoke", "verdict": "stale"},
      {"phase": "structure", "verdict": "unknown"}
    ],
    "suggestedCommand": "vrooli scenario test demo-scenario --phases smoke,structure"
  },
  "recommendations": [
    {"priority": "high", "description": "Fix the 2 standards errors blocking R1.", "impactPoints": 6},
    {"priority": "medium", "description": "Add UI feature tests.", "impactPoints": 3},
    {"priority": "low", "description": "Tidy docs.", "impactPoints": 1},
    {"priority": "low", "description": "A fourth rec that must be capped."}
  ]
}`

func TestCompletenessSupplement_RendersScorePayload(t *testing.T) {
	var gotScenario string
	out := renderCompleteness(t, func(_ context.Context, scenario string) ([]byte, error) {
		gotScenario = scenario
		return []byte(sampleScoreJSON), nil
	})

	if gotScenario != "demo-scenario" {
		t.Fatalf("runner called with scenario %q, want demo-scenario", gotScenario)
	}
	for _, want := range []string{
		"COMPLETENESS (scenario-completeness-scoring):",
		"82/100 (mostly_complete)",
		"working rung: R1 Safe & standards-clean",
		"Trend: ↑6 since 2026-06-08 (previous 76/100)",
		"1. [high] Fix the 2 standards errors blocking R1. (+6 pts)",
		"Stale evidence: smoke, structure",
		"refresh: vrooli scenario test demo-scenario --phases smoke,structure",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "A fourth rec that must be capped.") {
		t.Errorf("recommendations not capped at %d:\n%s", maxCompletenessRecommendations, out)
	}
}

func TestCompletenessSupplement_RendersWithoutTrend(t *testing.T) {
	out := renderCompleteness(t, func(context.Context, string) ([]byte, error) {
		return []byte(`{
		  "maturity": {"workingRung": "R1 Safe & standards-clean"},
		  "composite": {"score": 82, "classification": "mostly_complete"},
		  "freshness": {"phases": [{"phase": "unit", "verdict": "fresh"}]}
		}`), nil
	})
	if strings.Contains(out, "Trend:") {
		t.Errorf("trend line must be absent when payload omits trend:\n%s", out)
	}
	if !strings.Contains(out, "82/100 (mostly_complete)") {
		t.Errorf("score line missing:\n%s", out)
	}
}

func TestCompletenessSupplement_SilentWhenRunnerErrors(t *testing.T) {
	out := renderCompleteness(t, func(context.Context, string) ([]byte, error) {
		return nil, errors.New("exec: \"scenario-completeness-scoring\": executable file not found in $PATH")
	})
	if strings.Contains(out, "COMPLETENESS") {
		t.Errorf("supplement must be silently absent on runner error:\n%s", out)
	}
}

func TestCompletenessSupplement_SilentWhenRunnerNil(t *testing.T) {
	out := renderCompleteness(t, nil)
	if strings.Contains(out, "COMPLETENESS") {
		t.Errorf("supplement must be absent with a nil runner:\n%s", out)
	}
}

func TestCompletenessSupplement_BudgetEnforcedOnSlowRunner(t *testing.T) {
	start := time.Now()
	out := renderCompleteness(t, func(ctx context.Context, _ string) ([]byte, error) {
		// A well-behaved runner (exec.CommandContext) returns when the
		// context expires; simulate that instead of sleeping past it.
		<-ctx.Done()
		return nil, ctx.Err()
	})
	elapsed := time.Since(start)

	if strings.Contains(out, "COMPLETENESS") {
		t.Errorf("supplement must be silently absent on timeout:\n%s", out)
	}
	if elapsed < completenessBudget {
		t.Fatalf("runner context expired after %v, before the %v budget", elapsed, completenessBudget)
	}
	if elapsed > completenessBudget+time.Second {
		t.Fatalf("budget not enforced: render blocked for %v", elapsed)
	}
}

func TestCompletenessSupplement_SilentOnMalformedJSON(t *testing.T) {
	out := renderCompleteness(t, func(context.Context, string) ([]byte, error) {
		return []byte(`{"composite": {"score": `), nil
	})
	if strings.Contains(out, "COMPLETENESS") {
		t.Errorf("supplement must be silently absent on malformed JSON:\n%s", out)
	}
}

func TestCompletenessSupplement_SilentOnEmptyPayload(t *testing.T) {
	out := renderCompleteness(t, func(context.Context, string) ([]byte, error) {
		return []byte(`{}`), nil
	})
	if strings.Contains(out, "COMPLETENESS") {
		t.Errorf("supplement must be silently absent on an empty payload:\n%s", out)
	}
}

func TestCompletenessSupplement_LadderCleanHeadline(t *testing.T) {
	out := renderCompleteness(t, func(context.Context, string) ([]byte, error) {
		return []byte(`{
		  "maturity": {"ladderClean": true},
		  "composite": {"score": 97, "classification": "production_ready"},
		  "freshness": {"phases": [{"phase": "unit", "verdict": "fresh"}]}
		}`), nil
	})
	if !strings.Contains(out, "working rung: ladder clean through R4") {
		t.Errorf("ladder-clean headline missing:\n%s", out)
	}
	if strings.Contains(out, "Stale evidence") {
		t.Errorf("no stale hint expected when everything is fresh:\n%s", out)
	}
}
