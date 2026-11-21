package trials

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// The hidden golden.diff artifacts confirm the committed oracles accept one
// known solution for each positive family. Needs git.
func TestCommittedOraclesAcceptGoldenSolutions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := committedFixturesRoot(t)
	r := NewFixtureResolverWithRoot(root)

	for _, family := range []string{SuiteAddFeature, SuiteBugfix, SuiteComprehend, SuiteResearch} {
		fx, ok, err := r.Resolve(context.Background(), TrialTask{ID: "t", Suite: family})
		if err != nil || !ok {
			t.Fatalf("%s: resolve: ok=%v err=%v", family, ok, err)
		}
		if strings.TrimSpace(fx.GoldenSolution) == "" {
			t.Fatalf("%s: golden reference solution is missing", family)
		}
		exit, out, err := realOracleCheck(context.Background(), fx, fx.GoldenSolution)
		if err != nil {
			t.Fatalf("%s: oracle errored: %v\n%s", family, err, out)
		}
		if exit != 0 {
			t.Fatalf("%s: golden solution did NOT pass the oracle (exit=%d):\n%s", family, exit, out)
		}
	}
}
