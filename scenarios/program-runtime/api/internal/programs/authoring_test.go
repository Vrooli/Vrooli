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
		RunCase: func(_ context.Context, _, source string) (string, string, int64, string, error) {
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

func TestAuthoringEvalPassesExplicitSetupBeforeAuthoredProgram(t *testing.T) {
	setupSeen := false
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, AuthoringSuite{
			Name: "session-reuse",
			Cases: []AuthoringCase{{
				ID:     "reuse",
				Task:   "Reuse prior_result and print its value.",
				Setup:  "prior_result = {'value': 'reused'}",
				Oracle: AuthoringOracle{Kind: "stdout_contains", Text: "reused"},
			}},
		}),
		Author: func(context.Context, string, string) (string, string, error) {
			return "print(prior_result['value'])", "test-model", nil
		},
		RunCase: func(_ context.Context, setup, source string) (string, string, int64, string, error) {
			if setup != "prior_result = {'value': 'reused'}" {
				t.Fatalf("expected explicit session setup, got %q", setup)
			}
			if source == "" {
				t.Fatal("authored source must reach submission")
			}
			setupSeen = true
			return "reused", "", 7, "", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if !setupSeen || result.Met != 1 || result.Status != "measured" {
		t.Fatalf("expected setup-backed measured case, got setup=%v status=%q met=%d", setupSeen, result.Status, result.Met)
	}
}

func TestAuthoringOraclesUseDeclaredFields(t *testing.T) {
	if satisfiesOracle(AuthoringOracle{Kind: "stdout_contains", Value: "reused"}, "something else") {
		t.Fatal("stdout_contains must use the declared value, not merely non-empty output")
	}
	if !satisfiesOracle(AuthoringOracle{Kind: "row_count_relation", Relation: "gt", Value: 1}, "2") {
		t.Fatal("row_count_relation should compare the observed count")
	}
	if satisfiesOracle(AuthoringOracle{Kind: "row_count_relation", Relation: "gt", Value: 1}, "1") {
		t.Fatal("row_count_relation must reject an equal count for gt")
	}
	if satisfiesOracle(AuthoringOracle{Kind: "handle_metadata", Key: "role"}, "{'other': 'value'}") {
		t.Fatal("handle_metadata must require its declared key")
	}
	if satisfiesOracle(AuthoringOracle{Kind: "null_verdict"}, "binding-id") {
		t.Fatal("null_verdict must not accept an arbitrary successful binding")
	}
	if !satisfiesOracle(AuthoringOracle{Kind: "null_verdict"}, "no capability") {
		t.Fatal("null_verdict must accept the exact safe-stop answer requested by the corpus")
	}
}

func TestAuthoringEvalCountsWrongResultSeparatelyFromFailure(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author:    func(_ context.Context, _, _ string) (string, string, error) { return "print('nope')", "m", nil },
		RunCase: func(_ context.Context, _, _ string) (string, string, int64, string, error) {
			return "nope", "", 4, "", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.WrongResult != 2 || result.Met != 0 || result.Missed != 0 {
		t.Fatalf("a program that ran but answered wrongly is wrong_result, got met=%d missed=%d wrong=%d",
			result.Met, result.Missed, result.WrongResult)
	}
	if len(result.RuleMisses) != 1 || result.RuleMisses[0].RuleID != "unattributed" || result.RuleMisses[0].Count != 2 {
		t.Fatalf("wrong results must be explicitly counted, got %#v", result.RuleMisses)
	}
	for _, item := range result.Cases {
		if item.RuleID != "unattributed" || item.FailureDetail == "" {
			t.Fatalf("wrong result must retain explicit attribution evidence, got %#v", item)
		}
	}
}

func TestAuthoringEvalCountsFailedProgramAsMissed(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, twoCaseSuite()),
		Author:    func(_ context.Context, _, _ string) (string, string, error) { return "print(nope)", "m", nil },
		RunCase: func(_ context.Context, _, _ string) (string, string, int64, string, error) {
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
	if result.Cases[0].RuleID != "unattributed" {
		t.Fatalf("an unmatched failure must not be guessed, got %q", result.Cases[0].RuleID)
	}
	if len(result.RuleMisses) != 1 || result.RuleMisses[0].Count != 2 {
		t.Fatalf("per-rule counts must sum to missed cases, got %#v", result.RuleMisses)
	}
}

func TestAuthoringEvalAttributesRecordedHarnessFailure(t *testing.T) {
	deps := AuthoringDeps{
		SuitePath: writeSuite(t, AuthoringSuite{Name: "attribution", Cases: []AuthoringCase{{
			ID: "ambiguous", Task: "call", Oracle: AuthoringOracle{Kind: "stdout_contains", Text: "ok"},
		}}}),
		Author: func(_ context.Context, _, _ string) (string, string, error) { return "call()", "m", nil },
		RunCase: func(_ context.Context, _, _ string) (string, string, int64, string, error) {
			return "", "ambiguous_response", 0, "binding search-hub/query/query has no determinable primary response field", nil
		},
	}
	result := RunAuthoringEval(context.Background(), deps)
	if result.Missed != 1 || result.Cases[0].RuleID != "rows-field-for-ambiguous" {
		t.Fatalf("expected recorded failure to resolve to rows-field rule, got %#v", result)
	}
	if len(result.RuleMisses) != 1 || result.RuleMisses[0].RuleID != "rows-field-for-ambiguous" || result.RuleMisses[0].Count != 1 {
		t.Fatalf("expected one attributed miss, got %#v", result.RuleMisses)
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
		RunCase: func(_ context.Context, _, _ string) (string, string, int64, string, error) { return "", "", 0, "", nil },
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
