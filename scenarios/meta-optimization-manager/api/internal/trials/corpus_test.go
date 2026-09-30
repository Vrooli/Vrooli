package trials

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// committedFixturesRoot resolves the real trials/fixtures corpus from the
// scenario tree, skipping if the repo layout can't be located (e.g. an unusual
// CI checkout).
func committedFixturesRoot(t *testing.T) string {
	t.Helper()
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Skipf("repo root not locatable: %v", err)
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(repoRoot, fixtureScenario)
	if err != nil {
		t.Skipf("scenario path not resolvable: %v", err)
	}
	root := filepath.Join(scenarioRoot, "trials", "fixtures")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("fixtures corpus not present: %v", err)
	}
	return root
}

// TestCommittedCorpusResolves asserts every shipped family fixture parses,
// carries a prompt + revision, and (for positives) a runnable oracle.
func TestCommittedCorpusResolves(t *testing.T) {
	root := committedFixturesRoot(t)
	r := NewFixtureResolverWithRoot(root)

	families := []struct {
		suite    string
		task     TrialTask
		negative bool
	}{
		{SuiteAddFeature, TrialTask{ID: "trial/af", Suite: SuiteAddFeature}, false},
		{SuiteBugfix, TrialTask{ID: "trial/bf", Suite: SuiteBugfix}, false},
		{SuiteComprehend, TrialTask{ID: "trial/cp", Suite: SuiteComprehend}, false},
		{SuiteResearch, TrialTask{ID: "trial/rs", Suite: SuiteResearch}, false},
		{SuiteNegative, TrialTask{ID: "trial/negative/x", Suite: SuiteNegative, Negative: true}, true},
	}
	for _, f := range families {
		fx, ok, err := r.Resolve(context.Background(), f.task)
		if err != nil || !ok {
			t.Fatalf("%s: resolve ok=%v err=%v", f.suite, ok, err)
		}
		if fx.Prompt == "" || fx.Rev == "" {
			t.Fatalf("%s: missing prompt/rev: %+v", f.suite, fx)
		}
		if fx.Negative != f.negative {
			t.Fatalf("%s: negative=%v want %v", f.suite, fx.Negative, f.negative)
		}
		if !f.negative && len(fx.Oracle) == 0 {
			t.Fatalf("%s: positive fixture needs an oracle", f.suite)
		}
	}
}

// goldenDiffs are the reference "solved" diffs for each positive family. They
// confirm the committed oracles actually pass on a correct solution (and the
// negative-by-construction abstention is handled by the evaluator). Needs git.
func TestCommittedOraclesAcceptGoldenSolutions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := committedFixturesRoot(t)
	r := NewFixtureResolverWithRoot(root)

	cases := []struct {
		suite string
		diff  string
	}{
		{SuiteAddFeature, "diff --git a/sum.sh b/sum.sh\n--- a/sum.sh\n+++ b/sum.sh\n@@ -1,4 +1,4 @@\n #!/usr/bin/env bash\n # sum.sh A B — print the integer sum of A and B.\n # TODO: implement. Currently always prints 0.\n-echo 0\n+echo $(( $1 + $2 ))\n"},
		{SuiteBugfix, "diff --git a/parity.sh b/parity.sh\n--- a/parity.sh\n+++ b/parity.sh\n@@ -2,6 +2,6 @@\n n=\"$1\"\n if [ $(( n % 2 )) -eq 0 ]; then\n-  echo odd\n+  echo even\n else\n-  echo even\n+  echo odd\n fi\n"},
		{SuiteComprehend, "diff --git a/ANSWER.txt b/ANSWER.txt\nnew file mode 100644\n--- /dev/null\n+++ b/ANSWER.txt\n@@ -0,0 +1 @@\n+compute_checksum\n"},
		{SuiteResearch, "diff --git a/FOUND.txt b/FOUND.txt\nnew file mode 100644\n--- /dev/null\n+++ b/FOUND.txt\n@@ -0,0 +1 @@\n+z9x8c7v6\n"},
	}
	for _, c := range cases {
		fx, ok, err := r.Resolve(context.Background(), TrialTask{ID: "t", Suite: c.suite})
		if err != nil || !ok {
			t.Fatalf("%s: resolve: ok=%v err=%v", c.suite, ok, err)
		}
		exit, out, err := realOracleCheck(context.Background(), fx, c.diff)
		if err != nil {
			t.Fatalf("%s: oracle errored: %v\n%s", c.suite, err, out)
		}
		if exit != 0 {
			t.Fatalf("%s: golden solution did NOT pass the oracle (exit=%d):\n%s", c.suite, exit, out)
		}
	}
}
