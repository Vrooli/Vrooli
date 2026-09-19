package mutation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"unit-health/internal/executor"
	"unit-health/internal/testquality"
)

type pilotRunner struct{}

func (pilotRunner) Run(_ context.Context, command executor.Command) executor.Result {
	if !strings.Contains(strings.Join(command.Args, " "), "-run=") {
		return executor.Result{Status: executor.StatusError, FailureClass: executor.ClassMisconfiguration, FailureReason: "owning test was not selected"}
	}
	return executor.Result{Status: executor.StatusPassed}
}

func TestPilotUsesDisposableWorkspaceAndStableSummary(t *testing.T) {
	root := t.TempDir()
	packageRoot := filepath.Join(root, "pkg")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	source := `package pkg

func check(value int) bool {
	if value > 0 { return true }
	return false
}
`
	if err := os.WriteFile(filepath.Join(packageRoot, "source.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "source_test.go"), []byte(`package pkg
import "testing"
func TestCheck(t *testing.T) { if !check(1) { t.Fatal("bad") } }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	first, err := Run(context.Background(), PilotRequest{WorkspaceRoot: root, Package: "./pkg/...", Operators: []string{"boundary", "negate-condition", "return-constant"}, MaxMutants: 3, Seed: "same", CacheRoot: cache, Executor: pilotRunner{}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Run(context.Background(), PilotRequest{WorkspaceRoot: root, Package: "./pkg/...", Operators: []string{"boundary", "negate-condition", "return-constant"}, MaxMutants: 3, Seed: "same", CacheRoot: cache, Executor: pilotRunner{}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Summary != second.Summary || len(first.Receipts) != 3 || len(second.Receipts) != 3 {
		t.Fatalf("summaries are not deterministic: first=%+v second=%+v", first.Summary, second.Summary)
	}
	if string(mustRead(t, filepath.Join(packageRoot, "source.go"))) != source {
		t.Fatal("pilot changed the shared workspace")
	}
	for _, receipt := range first.Receipts {
		if receipt.Disposition != testquality.MutationSurvived {
			t.Fatalf("fake passing runner classified %s: %+v", receipt.Disposition, first.Receipts)
		}
	}
	entries, err := os.ReadDir(first.WorkspacePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("mutant copies remain in cache prefix: %v", entries)
	}
}

func TestDiscoverAndApplySupportsAllOperators(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(`package sample
func check(value int) bool { if value < 1 { return true }; return false }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err := Discover(root, root, []string{"boundary", "negate-condition", "return-constant"})
	if err != nil || len(candidates) < 3 {
		t.Fatalf("candidates=%v err=%v", candidates, err)
	}
	original := mustRead(t, filepath.Join(root, "sample.go"))
	seen := map[string]string{}
	for _, candidate := range candidates {
		if _, ok := seen[candidate.Operator]; ok {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, "sample.go"), original, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := Apply(filepath.Join(root, candidate.File), candidate); err != nil {
			t.Fatalf("apply %s: %v", candidate.Operator, err)
		}
		seen[candidate.Operator] = string(mustRead(t, filepath.Join(root, "sample.go")))
	}
	if !strings.Contains(seen[OperatorBoundary], "value <= 1") || !strings.Contains(seen[OperatorNegateCondition], "!(value < 1)") || strings.Count(seen[OperatorReturnConstant], "return false") < 2 {
		t.Fatalf("operators did not apply as expected: %v", seen)
	}
}

func TestMutationWorkspaceGuardsPathsAndClassifiesReceipts(t *testing.T) {
	root := t.TempDir()
	for _, packageArg := range []string{"", ".", "../escape", "/absolute"} {
		if _, err := safePackagePath(root, packageArg); err == nil {
			t.Fatalf("safePackagePath accepted %q", packageArg)
		}
	}
	if got, err := safePackagePath(root, "./pkg/..."); err != nil || !strings.HasSuffix(got, string(filepath.Separator)+"pkg") {
		t.Fatalf("safePackagePath = %q, %v", got, err)
	}
	if got := normalizeOperatorList([]string{"boundary", "boundary", "negate-condition"}); len(got) != 2 {
		t.Fatalf("normalized operators = %v", got)
	}

	cases := []struct {
		name       string
		result     executor.Result
		want       testquality.MutationDisposition
		wantDetail string
	}{
		{name: "passed", result: executor.Result{Status: executor.StatusPassed}, want: testquality.MutationSurvived, wantDetail: "owning test passed"},
		{name: "timeout", result: executor.Result{Status: executor.StatusTimeout, FailureReason: "timed out"}, want: testquality.MutationInfrastructureFailure, wantDetail: "timed out"},
		{name: "compile", result: executor.Result{Status: executor.StatusFailed, Stderr: "build failed"}, want: testquality.MutationInvalid, wantDetail: "mutated source did not compile"},
		{name: "failed", result: executor.Result{Status: executor.StatusFailed}, want: testquality.MutationKilled, wantDetail: "owning test failed"},
		{name: "unknown", result: executor.Result{Status: executor.StatusError}, want: testquality.MutationUnknown, wantDetail: "executor returned no classified evidence"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, detail := classifyResult(tc.result)
			if got != tc.want || detail != tc.wantDetail {
				t.Fatalf("classifyResult = %s/%q, want %s/%q", got, detail, tc.want, tc.wantDetail)
			}
		})
	}
	if got := summarize([]Receipt{{Disposition: testquality.MutationKilled}, {Disposition: testquality.MutationSurvived}, {Disposition: testquality.MutationInvalid}}); got.Generated != 3 || got.Killed != 1 || got.Survived != 1 || got.Invalid != 1 || got.KillRate != 0.5 {
		t.Fatalf("summary = %+v", got)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
