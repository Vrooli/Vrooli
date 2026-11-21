package trials

import (
	"context"
	"errors"
	"testing"
)

func oracleFixture() Fixture {
	return Fixture{Family: SuiteAddFeature, Oracle: []string{"bash", "check.sh"}, TargetDir: "/fx/target"}
}

func TestEvaluatorDeterministicPassFail(t *testing.T) {
	cases := []struct {
		name     string
		exit     int
		checkErr error
		want     Verdict
	}{
		{"oracle exit 0 → pass", 0, nil, VerdictPass},
		{"oracle exit 1 → fail", 1, nil, VerdictFail},
		{"oracle could-not-run → error", 0, errors.New("apply failed"), VerdictError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := func(_ context.Context, _ Fixture, _ string) (int, string, error) {
				return tc.exit, "out", tc.checkErr
			}
			ev := NewEvaluatorWithDeps(check, nil, nil)
			got := ev.Judge(context.Background(),
				TrialTask{ID: "g1", Suite: SuiteAddFeature}, oracleFixture(),
				RunResult{Verdict: VerdictUnspecified, Diff: "diff --git ..."})
			if got != tc.want {
				t.Fatalf("verdict = %v want %v", got, tc.want)
			}
		})
	}
}

func TestEvaluatorNegativeAbstention(t *testing.T) {
	ev := NewEvaluatorWithDeps(nil, nil, nil)
	neg := Fixture{Family: SuiteNegative, Negative: true}
	task := TrialTask{ID: "neg/x", Suite: SuiteNegative, Negative: true}

	// Correct abstention: no change, no diff → pass.
	if v := ev.Judge(context.Background(), task, neg, RunResult{ChangedFiles: 0, Diff: ""}); v != VerdictPass {
		t.Fatalf("abstention should pass, got %v", v)
	}
	// Fabricated change → fail.
	if v := ev.Judge(context.Background(), task, neg, RunResult{ChangedFiles: 2, Diff: "diff --git a/x b/x"}); v != VerdictFail {
		t.Fatalf("fabrication should fail, got %v", v)
	}
	// A diff with only whitespace still counts as abstention.
	if v := ev.Judge(context.Background(), task, neg, RunResult{ChangedFiles: 0, Diff: "   \n"}); v != VerdictPass {
		t.Fatalf("whitespace-only diff should pass as abstention, got %v", v)
	}
}

func TestEvaluatorAgentJudgeFallbackSelection(t *testing.T) {
	noOracle := Fixture{Family: SuiteResearch} // no deterministic oracle
	task := TrialTask{ID: "r1", Suite: SuiteResearch}

	// No oracle + no judge configured → honest error (never a guessed pass).
	ev := NewEvaluatorWithDeps(failingCheck(t), nil, nil)
	if v := ev.Judge(context.Background(), task, noOracle, RunResult{Diff: "x"}); v != VerdictError {
		t.Fatalf("oracle-less + no judge should error, got %v", v)
	}

	// No oracle + judge configured → judge decides (lower confidence).
	judged := false
	judge := func(_ context.Context, _ TrialTask, _ Fixture, _ RunResult) (Verdict, error) {
		judged = true
		return VerdictPass, nil
	}
	ev = NewEvaluatorWithDeps(failingCheck(t), judge, nil)
	if v := ev.Judge(context.Background(), task, noOracle, RunResult{Diff: "x"}); v != VerdictPass {
		t.Fatalf("judge verdict should be returned, got %v", v)
	}
	if !judged {
		t.Fatalf("agent-judge fallback was not selected for an oracle-less family")
	}

	// Judge unavailable → error, never a pass.
	ev = NewEvaluatorWithDeps(failingCheck(t), func(context.Context, TrialTask, Fixture, RunResult) (Verdict, error) {
		return VerdictUnspecified, errors.New("judge offline")
	}, nil)
	if v := ev.Judge(context.Background(), task, noOracle, RunResult{Diff: "x"}); v != VerdictError {
		t.Fatalf("judge error should be VerdictError, got %v", v)
	}
}

func TestEvaluatorOracleTakesPrecedenceOverJudge(t *testing.T) {
	// With a deterministic oracle present, the agent-judge must NOT be consulted.
	judge := func(_ context.Context, _ TrialTask, _ Fixture, _ RunResult) (Verdict, error) {
		t.Fatalf("agent-judge must not run when a deterministic oracle exists")
		return VerdictPass, nil
	}
	check := func(_ context.Context, _ Fixture, _ string) (int, string, error) { return 0, "", nil }
	ev := NewEvaluatorWithDeps(check, judge, nil)
	if v := ev.Judge(context.Background(), TrialTask{ID: "g1"}, oracleFixture(), RunResult{Diff: "x"}); v != VerdictPass {
		t.Fatalf("oracle pass expected, got %v", v)
	}
}

func TestEvaluatorNeverUpgradesRunnerError(t *testing.T) {
	ev := NewEvaluatorWithDeps(func(context.Context, Fixture, string) (int, string, error) {
		t.Fatalf("oracle must not run when the runner already errored")
		return 0, "", nil
	}, nil, nil)
	if v := ev.Judge(context.Background(), TrialTask{ID: "g1"}, oracleFixture(), RunResult{Verdict: VerdictError}); v != VerdictError {
		t.Fatalf("runner error must stay error, got %v", v)
	}
}

// failingCheck returns an oracleChecker that fails the test if invoked — used to
// assert the oracle path is NOT taken for oracle-less families.
func failingCheck(t *testing.T) oracleChecker {
	return func(context.Context, Fixture, string) (int, string, error) {
		t.Fatalf("oracle checker must not run for a family with no oracle")
		return 0, "", nil
	}
}
