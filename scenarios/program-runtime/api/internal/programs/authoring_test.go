package programs

import (
	"context"
	"encoding/json"
	"errors"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/encoding/protojson"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestAuthoringEvalRecordsComparableTimingAndIdentity(t *testing.T) {
	suitePath := writeSuite(t, AuthoringSuite{
		Name: "timed",
		Cases: []AuthoringCase{{
			ID: "case-1", Task: "return rows", Oracle: AuthoringOracle{Kind: "stdout_contains", Text: "rows"},
		}},
	})
	corpus, err := os.ReadFile(suitePath)
	if err != nil {
		t.Fatal(err)
	}
	var instruction string
	source := "print({'rows': 1})"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	result := RunAuthoringEval(ctx, AuthoringDeps{
		SuitePath: suitePath,
		Author: func(_ context.Context, brief, _ string) (string, string, error) {
			instruction = brief
			return "```python\n" + source + "\n```", "test-model", nil
		},
		RunCase: func(_ context.Context, _, got string) (string, string, int64, string, error) {
			if got != source {
				t.Fatalf("source fence not stripped before execution: %q", got)
			}
			return "rows", "", 4, "", nil
		},
	})
	if len(result.Cases) != 1 {
		t.Fatalf("cases=%d, want 1", len(result.Cases))
	}
	item := result.Cases[0]
	if result.ReceiptVersion != AuthoringReceiptVersion || result.CorpusSHA256 != sha256Hex(corpus) {
		t.Fatalf("receipt/corpus identity=%q/%q", result.ReceiptVersion, result.CorpusSHA256)
	}
	if result.InstructionSHA256 != sha256Hex([]byte(instruction)) || item.TaskSHA256 != sha256Hex([]byte("return rows")) || item.SourceSHA256 != sha256Hex([]byte(source)) {
		t.Fatalf("instruction/task/source digests are not over the consumed inputs: %+v", result)
	}
	if result.AuthorRole != AuthoringRole || result.AuthorSchemaSHA256 != sha256Hex([]byte(AuthoringSchemaJSON)) || result.RoutePolicyRevision != unknownRevision {
		t.Fatalf("authoring route metadata=%+v", result)
	}
	if result.StartedAt == "" || result.RuntimeRevision == "" || result.ElapsedMS < 0 || result.TotalBudgetMS <= 0 || result.PerCaseReserveMS != perCaseReserve.Milliseconds() {
		t.Fatalf("run timing/revision metadata=%+v", result)
	}
	if item.GenerationElapsedMS < 0 || item.ExecutionElapsedMS < 0 || item.OracleElapsedMS < 0 || item.CaseStartOffsetMS < 0 || item.ElapsedToFirstWorkingMS == nil || *item.ElapsedToFirstWorkingMS < 0 {
		t.Fatalf("case timing fields=%+v", item)
	}
	if result.RoutePolicyRevision != "unknown" {
		t.Fatalf("unavailable route revision must be explicit, got %q", result.RoutePolicyRevision)
	}
}

func TestAuthoringEvalLeavesFirstWorkingTimeAbsentAfterFailure(t *testing.T) {
	result := RunAuthoringEval(context.Background(), AuthoringDeps{
		SuitePath: writeSuite(t, AuthoringSuite{Cases: []AuthoringCase{{
			ID: "failed", Task: "fail", Oracle: AuthoringOracle{Kind: "stdout_contains", Text: "ok"},
		}}}),
		Author: func(context.Context, string, string) (string, string, error) {
			return "raise ValueError()", "test-model", nil
		},
		RunCase: func(context.Context, string, string) (string, string, int64, string, error) {
			return "", "kernel_runtime", 0, "ValueError: failure", nil
		},
	})
	if len(result.Cases) != 1 {
		t.Fatalf("cases=%d, want 1", len(result.Cases))
	}
	if result.Cases[0].ElapsedToFirstWorkingMS != nil || result.Cases[0].OracleElapsedMS != 0 {
		t.Fatalf("failed execution must not claim oracle/working time: %+v", result.Cases[0])
	}
	if result.Cases[0].ExecutionElapsedMS < 0 || result.Cases[0].GenerationElapsedMS < 0 {
		t.Fatalf("completed phases must have nonnegative timing: %+v", result.Cases[0])
	}
}

func TestAuthoringRequestSerializesLocalOnlyProfile(t *testing.T) {
	raw, err := json.Marshal(AuthoringInferenceArgs("brief", "task"))
	if err != nil {
		t.Fatal(err)
	}
	var request inferencev1.RunRequest
	if err := protojson.Unmarshal(raw, &request); err != nil {
		t.Fatalf("decode serialized authoring request: %v", err)
	}
	if request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY || request.GetRole() != AuthoringRole || request.GetSchemaJson() != AuthoringSchemaJSON {
		t.Fatalf("serialized authoring request lost its local-only typed contract: %+v", &request)
	}
	if request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL || request.GetPrivacyProvenance() != "program_runtime.code_authoring.user_task_and_repository_context" {
		t.Fatalf("serialized authoring request lost its owner-derived privacy contract: %s from %q", request.GetPrivacyClass(), request.GetPrivacyProvenance())
	}
}

func TestAuthoringRouteRequiresKnownLocalProviderAndModel(t *testing.T) {
	for _, tc := range []struct {
		provider, model string
		wantErr         bool
	}{
		{"ollama", "local-model:latest", false},
		{"", "model", true},
		{"openrouter", "hosted-model", true},
		{"future-provider", "model", true},
		{"ollama", "", true},
	} {
		err := ValidateAuthoringRoute(tc.provider, tc.model)
		if (err != nil) != tc.wantErr {
			t.Errorf("provider=%q model=%q error=%v wantErr=%v", tc.provider, tc.model, err, tc.wantErr)
		}
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

func TestJSONAssertionOracleChecksNestedValuesAndCalls(t *testing.T) {
	oracle := AuthoringOracle{Kind: "json_assertions", Assertions: []JSONAssertion{
		{Path: "signals.top.0.provider_id", Op: "eq", Value: "alpha"},
		{Path: "signals.top", Op: "length_eq", Value: 2},
		{Path: "signals.missing", Op: "missing"},
	}, CallCounts: []FixtureCallCount{{BindingID: "search-hub/query/query", Count: 1, Args: map[string]any{"query": "program runtime"}, Rows: "ranked"}}, MaxOutputBytes: 100}
	stdout := `{"signals":{"top":[{"provider_id":"alpha"},{"provider_id":"beta"}]}}`
	execution := AuthoringCaseExecution{FixtureCalls: map[string]int32{"search-hub/query/query": 1}, FixtureRequests: []FixtureCallTrace{{BindingID: "search-hub/query/query", Args: map[string]any{"query": "program runtime"}, Rows: "ranked"}}}
	if !fixtureOracleSatisfied(oracle, stdout, execution) {
		t.Fatal("nested value and exact call-count assertions should pass")
	}
	wrongCount := execution
	wrongCount.FixtureCalls = map[string]int32{"search-hub/query/query": 2}
	if fixtureOracleSatisfied(oracle, stdout, wrongCount) {
		t.Fatal("the oracle must reject duplicate binding calls")
	}
	if fixtureOracleSatisfied(oracle, stdout+" "+stdout, execution) {
		t.Fatal("the oracle must enforce the output bound")
	}
	if fixtureOracleSatisfied(oracle, stdout+` {"extra":true}`, execution) {
		t.Fatal("the oracle must reject trailing JSON values")
	}
	wrongArgs := execution
	wrongArgs.FixtureRequests = []FixtureCallTrace{{BindingID: "search-hub/query/query", Args: map[string]any{"query": "wrong query"}, Rows: "ranked"}}
	if fixtureOracleSatisfied(oracle, stdout, wrongArgs) {
		t.Fatal("the oracle must reject calls with mismatched task arguments")
	}
}

func TestJSONAssertionOperatorsHavePositiveAndNearMissCases(t *testing.T) {
	tests := []struct {
		name        string
		actual      any
		exists      bool
		check       JSONAssertion
		other       any
		otherExists bool
		want        bool
	}{
		{"eq", "alpha", true, JSONAssertion{Op: "eq", Value: "alpha"}, nil, false, true},
		{"eq-near-miss", "beta", true, JSONAssertion{Op: "eq", Value: "alpha"}, nil, false, false},
		{"ne", "alpha", true, JSONAssertion{Op: "ne", Value: "beta"}, nil, false, true},
		{"ne-near-miss", "alpha", true, JSONAssertion{Op: "ne", Value: "alpha"}, nil, false, false},
		{"gt", json.Number("3"), true, JSONAssertion{Op: "gt", Value: json.Number("2")}, nil, false, true},
		{"gt-near-miss", json.Number("2"), true, JSONAssertion{Op: "gt", Value: json.Number("2")}, nil, false, false},
		{"gte", json.Number("2"), true, JSONAssertion{Op: "gte", Value: json.Number("2")}, nil, false, true},
		{"gte-near-miss", json.Number("1"), true, JSONAssertion{Op: "gte", Value: json.Number("2")}, nil, false, false},
		{"lt", json.Number("1"), true, JSONAssertion{Op: "lt", Value: json.Number("2")}, nil, false, true},
		{"lt-near-miss", json.Number("2"), true, JSONAssertion{Op: "lt", Value: json.Number("2")}, nil, false, false},
		{"lte", json.Number("2"), true, JSONAssertion{Op: "lte", Value: json.Number("2")}, nil, false, true},
		{"lte-near-miss", json.Number("3"), true, JSONAssertion{Op: "lte", Value: json.Number("2")}, nil, false, false},
		{"contains-string", "alpha-beta", true, JSONAssertion{Op: "contains", Value: "beta"}, nil, false, true},
		{"contains-array", []any{"alpha", "beta"}, true, JSONAssertion{Op: "contains", Value: "beta"}, nil, false, true},
		{"contains-near-miss", []any{"alpha"}, true, JSONAssertion{Op: "contains", Value: "beta"}, nil, false, false},
		{"exists", nil, true, JSONAssertion{Op: "exists"}, nil, false, true},
		{"exists-near-miss", nil, false, JSONAssertion{Op: "exists"}, nil, false, false},
		{"missing", nil, false, JSONAssertion{Op: "missing"}, nil, false, true},
		{"missing-near-miss", nil, true, JSONAssertion{Op: "missing"}, nil, false, false},
		{"length-eq", []any{1, 2}, true, JSONAssertion{Op: "length_eq", Value: 2}, nil, false, true},
		{"length-eq-near-miss", []any{1}, true, JSONAssertion{Op: "length_eq", Value: 2}, nil, false, false},
		{"length-lte", "ab", true, JSONAssertion{Op: "length_lte", Value: 2}, nil, false, true},
		{"length-lte-near-miss", "abc", true, JSONAssertion{Op: "length_lte", Value: 2}, nil, false, false},
		{"equal-path", "alpha", true, JSONAssertion{Op: "equal_path"}, "alpha", true, true},
		{"equal-path-near-miss", "alpha", true, JSONAssertion{Op: "equal_path"}, "beta", true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := assertionMatches(test.actual, test.exists, test.check, test.other, test.otherExists); got != test.want {
				t.Fatalf("assertion result=%v want %v", got, test.want)
			}
		})
	}
}

func TestJSONAssertionOracleRejectsMalformedAndNonObjectOutput(t *testing.T) {
	oracle := AuthoringOracle{Kind: "json_assertions", Assertions: []JSONAssertion{{Path: "status", Op: "eq", Value: "ok"}}}
	for _, output := range []string{`{"status":`, `[{"status":"ok"}]`, `{"status":"ok"} trailing`, `{"status":"ok"} {"extra":true}`} {
		if fixtureOracleSatisfied(oracle, output, AuthoringCaseExecution{}) {
			t.Errorf("malformed/non-object output passed: %q", output)
		}
	}
	if fixtureOracleSatisfied(AuthoringOracle{Kind: "json_assertions"}, `{"status":"ok"}`, AuthoringCaseExecution{}) {
		t.Fatal("an empty oracle must fail closed")
	}
}

func TestAuthoringEvalReturnsSuiteOrderedRowsForSkippedCases(t *testing.T) {
	suite := AuthoringSuite{Name: "partial", Cases: []AuthoringCase{
		{ID: "second", Task: "first", Oracle: AuthoringOracle{Kind: "stdout_non_empty"}},
		{ID: "first", Task: "second", Oracle: AuthoringOracle{Kind: "stdout_non_empty"}},
	}}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(perCaseReserve/2))
	defer cancel()
	result := RunAuthoringEval(ctx, AuthoringDeps{SuitePath: writeSuite(t, suite), Author: func(context.Context, string, string) (string, string, error) {
		t.Fatal("deadline should prevent authoring")
		return "", "", nil
	}, RunCase: func(context.Context, string, string) (string, string, int64, string, error) {
		return "", "", 0, "", nil
	}})
	if len(result.Cases) != 2 || result.Cases[0].CaseID != "second" || result.Cases[1].CaseID != "first" {
		t.Fatalf("one ordered row per task is required: %#v", result.Cases)
	}
	for _, item := range result.Cases {
		if item.CaseStatus != "not_attempted" || item.AttemptCount != 0 {
			t.Fatalf("incorrect skipped row: %+v", item)
		}
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
