package programs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeSuite(t *testing.T, suite AuthoringSuite) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "suite.json")
	raw, err := json.Marshal(suite)
	if err != nil {
		t.Fatalf("marshal suite: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write suite: %v", err)
	}
	return path
}

func twoCaseSuite() AuthoringSuite {
	return AuthoringSuite{
		Name:  "test",
		Floor: 1,
		Cases: []AuthoringCase{
			{ID: "a", Task: "one", Oracle: AuthoringOracle{Kind: "stdout_json_keys", Keys: []string{"rows"}}},
			{ID: "b", Task: "two", Oracle: AuthoringOracle{Kind: "stdout_contains", Text: "ok"}},
		},
	}
}

// TestAuthoringEvalMeasuresRatherThanStubbing is the regression guard for a
// handler that returned a fixed `unavailable` response without loading the
// corpus, resolving a route, or submitting anything. That response is
// indistinguishable from an honest degradation, and it passes a floor gate
// trivially because no floor comparison happens when nothing was measured.
func TestAuthoringEvalMeasuresRatherThanStubbing(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author: func(_ context.Context, instruction, task string) (string, string, error) {
			if instruction == "" {
				t.Fatal("the authoring brief must be supplied; without it the eval measures prompt omission")
			}
			return "print({'rows': 1})", "test-model", nil
		},
		RunCase: func(_ context.Context, source string) (string, string, int64, string, error) {
			if source == "" {
				t.Fatal("authored source must reach submission")
			}
			return "{'rows': 1} ok", "", 22, "", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.Status != "measured" {
		t.Fatalf("expected a measured run, got %q (%s)", result.Status, result.Reason)
	}
	if result.Met != 2 {
		t.Fatalf("expected 2 met, got met=%d missed=%d wrong=%d", result.Met, result.Missed, result.WrongResult)
	}
	if len(result.Cases) != 2 {
		t.Fatalf("expected per-case attribution, got %d cases", len(result.Cases))
	}
	if result.Cases[0].Model != "test-model" {
		t.Fatalf("a score must be attributable to the model that produced it, got %q", result.Cases[0].Model)
	}
}

func TestAuthoringEvalCountsWrongResultSeparatelyFromFailure(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author:    func(_ context.Context, _, _ string) (string, string, error) { return "print('nope')", "m", nil },
		RunCase: func(_ context.Context, _ string) (string, string, int64, string, error) {
			return "nope", "", 4, "", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.WrongResult != 2 || result.Met != 0 || result.Missed != 0 {
		t.Fatalf("a program that ran but answered wrongly is wrong_result, got met=%d missed=%d wrong=%d",
			result.Met, result.Missed, result.WrongResult)
	}
}

func TestAuthoringEvalCountsFailedProgramAsMissed(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author:    func(_ context.Context, _, _ string) (string, string, error) { return "print(nope)", "m", nil },
		RunCase: func(_ context.Context, _ string) (string, string, int64, string, error) {
			return "", "unresolved_name", 0, "name \"nope\" does not resolve", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.Missed != 2 || result.Met != 0 {
		t.Fatalf("expected 2 missed, got met=%d missed=%d", result.Met, result.Missed)
	}
	if result.Cases[0].Cause != "unresolved_name" {
		t.Fatalf("the failure cause must be retained per case, got %q", result.Cases[0].Cause)
	}
}

// TestAuthoringEvalReportsUnavailableOnlyForRealRouteLoss keeps the honest
// degradation path meaningful: it must be reachable, and it must not be the
// answer to every run.
func TestAuthoringEvalReportsUnavailableOnlyForRealRouteLoss(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author: func(_ context.Context, _, _ string) (string, string, error) {
			return "", "", errors.New("ai-gateway is unreachable: connection refused")
		},
		RunCase: func(_ context.Context, _ string) (string, string, int64, string, error) { return "", "", 0, "", nil },
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.Status != "unavailable" || result.Unavailable != 1 {
		t.Fatalf("a lost route must report unavailable, got %q", result.Status)
	}
	if result.Met != 0 {
		t.Fatalf("an unavailable run must not claim a score, got met=%d", result.Met)
	}
	if result.Reason == "" {
		t.Fatal("an unavailable run must name its missing dependency")
	}
}

func TestAuthoringEvalReportsMissingCorpus(t *testing.T) {
	result := RunAuthoringEval(context.Background(), AuthoringDeps{SuitePath: filepath.Join(t.TempDir(), "absent.json")})
	if result.Status != "unavailable" || result.Reason == "" {
		t.Fatalf("a missing corpus must be reported with a reason, got %q / %q", result.Status, result.Reason)
	}
}

func TestStripSourceFenceKeepsProgramBody(t *testing.T) {
	fenced := "```python\nprint(1)\n```"
	if got := stripSourceFence(fenced); got != "print(1)" {
		t.Fatalf("expected the fence removed, got %q", got)
	}
	if got := stripSourceFence("print(1)"); got != "print(1)" {
		t.Fatalf("unfenced source must be untouched, got %q", got)
	}
}
