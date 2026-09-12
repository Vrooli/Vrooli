package trials

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// writeFixture lays down a minimal fixture under root/<family>/.
func writeFixture(t *testing.T, root, family, spec, check, targetFile, targetBody string, oracle []string, negative bool) {
	t.Helper()
	dir := filepath.Join(root, family)
	if err := os.MkdirAll(filepath.Join(dir, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"family":"` + family + `","negative":` + boolJSON(negative) + `,"oracle":` + jsonStrings(oracle) + `}`
	mustWrite(t, filepath.Join(dir, "fixture.json"), manifest)
	mustWrite(t, filepath.Join(dir, "spec.md"), spec)
	if check != "" {
		mustWrite(t, filepath.Join(dir, "check.sh"), check)
	}
	if targetFile != "" {
		mustWrite(t, filepath.Join(dir, "target", targetFile), targetBody)
	}
}

func TestFixtureResolverReadsCorpusAndRevisions(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, SuiteAddFeature, "Add a greeting", "#!/usr/bin/env bash\nexit 0\n",
		"answer.txt", "TODO\n", []string{"bash", "check.sh"}, false)

	r := NewFixtureResolverWithRoot(root)
	fx, ok, err := r.Resolve(context.Background(), TrialTask{ID: "trial/g1", Suite: SuiteAddFeature})
	if err != nil || !ok {
		t.Fatalf("resolve: ok=%v err=%v", ok, err)
	}
	if fx.Family != SuiteAddFeature || fx.Prompt != "Add a greeting" || fx.Negative {
		t.Fatalf("fixture fields wrong: %+v", fx)
	}
	if len(fx.Oracle) != 2 || fx.Oracle[0] != "bash" {
		t.Fatalf("oracle wrong: %v", fx.Oracle)
	}
	if fx.Rev == "" {
		t.Fatalf("revision must be populated")
	}
	if filepath.Base(fx.TargetDir) != "target" {
		t.Fatalf("target dir wrong: %s", fx.TargetDir)
	}

	// Editing a target file changes the revision (invalidates idempotency).
	rev1 := fx.Rev
	mustWrite(t, filepath.Join(root, SuiteAddFeature, "target", "answer.txt"), "CHANGED\n")
	fx2, _, _ := r.Resolve(context.Background(), TrialTask{ID: "trial/g1", Suite: SuiteAddFeature})
	if fx2.Rev == rev1 {
		t.Fatalf("revision must change when a target file changes")
	}
}

func TestFixtureResolverNegativeAndMissing(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, SuiteNegative, "Refuse to answer", "", "", "", nil, true)

	r := NewFixtureResolverWithRoot(root)
	neg, ok, err := r.Resolve(context.Background(), TrialTask{ID: "trial/negative/x", Suite: SuiteNegative, Negative: true})
	if err != nil || !ok || !neg.Negative {
		t.Fatalf("negative fixture: ok=%v neg=%v err=%v", ok, neg.Negative, err)
	}

	// A family with no fixture dir resolves to ok=false (not an error).
	_, ok, err = r.Resolve(context.Background(), TrialTask{ID: "trial/b1", Suite: SuiteBugfix})
	if err != nil || ok {
		t.Fatalf("missing family should be ok=false, no error: ok=%v err=%v", ok, err)
	}
}

// TestRealOracleCheckRoundTrip exercises the production oracle path (copy target
// → apply the agent's diff → run the oracle) without a live model. It needs git;
// skipped if absent.
func TestRealOracleCheckRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	// Oracle passes iff answer.txt contains DONE.
	check := "#!/usr/bin/env bash\ngrep -q DONE answer.txt\n"
	writeFixture(t, root, SuiteAddFeature, "Change TODO to DONE", check,
		"answer.txt", "TODO\n", []string{"bash", "check.sh"}, false)

	r := NewFixtureResolverWithRoot(root)
	fx, _, err := r.Resolve(context.Background(), TrialTask{ID: "trial/g1", Suite: SuiteAddFeature})
	if err != nil {
		t.Fatal(err)
	}

	goodDiff := "diff --git a/answer.txt b/answer.txt\n--- a/answer.txt\n+++ b/answer.txt\n@@ -1 +1 @@\n-TODO\n+DONE\n"
	exit, _, err := realOracleCheck(context.Background(), fx, goodDiff)
	if err != nil {
		t.Fatalf("oracle check errored: %v", err)
	}
	if exit != 0 {
		t.Fatalf("correct diff should pass the oracle, exit=%d", exit)
	}

	// A diff that doesn't satisfy the oracle → non-zero exit (a clean FAIL).
	badDiff := "diff --git a/answer.txt b/answer.txt\n--- a/answer.txt\n+++ b/answer.txt\n@@ -1 +1 @@\n-TODO\n+NOPE\n"
	exit, _, err = realOracleCheck(context.Background(), fx, badDiff)
	if err != nil {
		t.Fatalf("oracle check errored: %v", err)
	}
	if exit == 0 {
		t.Fatalf("wrong diff should fail the oracle")
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func boolJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func jsonStrings(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	out := "["
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += `"` + s + `"`
	}
	return out + "]"
}
