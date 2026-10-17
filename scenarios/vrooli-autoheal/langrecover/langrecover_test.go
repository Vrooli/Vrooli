package langrecover

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectGoSignature(t *testing.T) {
	cases := []struct {
		name string
		log  string
		want GoSignature
	}{
		{"empty", "", GoSignatureNone},
		{
			"missing-sum lowercase",
			"missing go.sum entry for module providing package modernc.org/sqlite",
			GoSignatureMissingSum,
		},
		{
			"missing-sum mixed-case",
			"Missing go.sum entry for module providing package golang.org/x/sys/unix",
			GoSignatureMissingSum,
		},
		{
			"missing-module",
			"no required module provides package github.com/vrooli/api-core; to add it: go get ...",
			GoSignatureMissingModule,
		},
		{
			"inconsistent-vendoring",
			"inconsistent vendoring in /home/x/scenario/api/vendor",
			GoSignatureMissingModule,
		},
		{
			"updates-to-go-mod-needed",
			"go: updates to go.mod needed; to update it:\n\tgo mod tidy",
			GoSignatureMissingModule,
		},
		{
			"unrelated",
			"build failed: syntax error in foo.go",
			GoSignatureNone,
		},
		{
			"prefer-missing-module-over-missing-sum",
			"missing go.sum entry; no required module provides package x",
			GoSignatureMissingModule,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectGoSignature(tc.log); got != tc.want {
				t.Fatalf("DetectGoSignature(%q) = %v, want %v", tc.log, got, tc.want)
			}
		})
	}
}

func TestDetectPnpmSignature(t *testing.T) {
	cases := []struct {
		name string
		log  string
		want PnpmSignature
	}{
		{"empty", "", PnpmSignatureNone},
		{
			"outdated-lockfile",
			"ERR_PNPM_OUTDATED_LOCKFILE pnpm-lock.yaml is out of date with package.json",
			PnpmSignatureOutdatedLockfile,
		},
		{
			"linking-failed",
			"ERR_PNPM_LINKING_FAILED could not link node_modules/.pnpm/foo",
			PnpmSignatureLinkingFailed,
		},
		{
			"enoent-node-modules",
			"ENOENT: no such file or directory, open '/x/ui/node_modules/.modules.yaml'",
			PnpmSignatureLinkingFailed,
		},
		{
			"enoent-unrelated",
			"ENOENT: no such file or directory, open '/x/src/missing.ts'",
			PnpmSignatureNone,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectPnpmSignature(tc.log); got != tc.want {
				t.Fatalf("DetectPnpmSignature(%q) = %v, want %v", tc.log, got, tc.want)
			}
		})
	}
}

func TestDecide(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x")
	mustWrite(t, filepath.Join(dir, "ui", "package.json"), "{}")

	t.Run("go signature wins over pnpm when both present", func(t *testing.T) {
		log := "missing go.sum entry for module ..."
		d := Decide(log, dir)
		if d.Kind != KindGo || d.GoSig != GoSignatureMissingSum {
			t.Fatalf("got %+v", d)
		}
		if !d.Has() {
			t.Fatalf("expected Has=true")
		}
	})
	t.Run("pnpm signature when no go signature", func(t *testing.T) {
		log := "ERR_PNPM_LINKING_FAILED ..."
		d := Decide(log, dir)
		if d.Kind != KindPnpm || d.PnpmSig != PnpmSignatureLinkingFailed {
			t.Fatalf("got %+v", d)
		}
	})
	t.Run("no scenario dir", func(t *testing.T) {
		if d := Decide("missing go.sum entry", ""); d.Has() {
			t.Fatalf("expected empty decision, got %+v", d)
		}
	})
	t.Run("no signature in log", func(t *testing.T) {
		if d := Decide("syntax error", dir); d.Has() {
			t.Fatalf("expected empty decision, got %+v", d)
		}
	})
}

func TestRecoverGo_MissingSum_RunsModDownloadAndDetectsSumChange(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(dir, "api", "go.sum"), "old\n")

	var called struct {
		dir  string
		name string
		args []string
	}
	runner := func(_ context.Context, runDir, name string, args ...string) ([]byte, error) {
		called.dir = runDir
		called.name = name
		called.args = args
		// Simulate `go mod download` rewriting go.sum.
		mustWrite(t, filepath.Join(runDir, "go.sum"), "new content\n")
		return []byte("downloaded\n"), nil
	}

	res, err := RecoverGo(context.Background(), runner, dir, GoSignatureMissingSum)
	if err != nil {
		t.Fatalf("RecoverGo error: %v", err)
	}
	if called.name != "go" || strings.Join(called.args, " ") != "mod download" {
		t.Fatalf("expected `go mod download`, got %s %v", called.name, called.args)
	}
	if !res.ModifiedTrackedFiles {
		t.Fatalf("expected ModifiedTrackedFiles=true")
	}
	if len(res.ModifiedPaths) != 1 || !strings.HasSuffix(res.ModifiedPaths[0], "go.sum") {
		t.Fatalf("expected go.sum modification, got %v", res.ModifiedPaths)
	}
	if res.Command != "go mod download" {
		t.Fatalf("unexpected Command: %q", res.Command)
	}
}

func TestRecoverGo_MissingSum_EscalatesToTidyWhenDownloadDoesNotChangeSum(t *testing.T) {
	// On real scenarios we observed `go mod download` exit 0 without adding
	// the missing h1 hash to go.sum. RecoverGo must escalate to `go mod tidy`
	// in that case so the scenario actually builds afterward.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(dir, "api", "go.sum"), "old\n")

	var commands [][]string
	runner := func(_ context.Context, runDir, name string, args ...string) ([]byte, error) {
		commands = append(commands, append([]string{name}, args...))
		// download is a no-op (does not change go.sum); tidy rewrites it.
		if strings.Join(args, " ") == "mod tidy" {
			mustWrite(t, filepath.Join(runDir, "go.sum"), "tidied\n")
			return []byte("tidied"), nil
		}
		return []byte("downloaded"), nil
	}

	res, err := RecoverGo(context.Background(), runner, dir, GoSignatureMissingSum)
	if err != nil {
		t.Fatalf("RecoverGo error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("expected 2 runner invocations (download then tidy), got %v", commands)
	}
	if strings.Join(commands[0][1:], " ") != "mod download" {
		t.Fatalf("first call should be `go mod download`, got %v", commands[0])
	}
	if strings.Join(commands[1][1:], " ") != "mod tidy" {
		t.Fatalf("second call should be `go mod tidy`, got %v", commands[1])
	}
	if !res.ModifiedTrackedFiles {
		t.Fatalf("ModifiedTrackedFiles should be true after tidy rewrites go.sum")
	}
	if !containsText(res.Command, "go mod tidy") {
		t.Fatalf("expected Command to record escalation, got %q", res.Command)
	}
}

func TestRecoverGo_MissingSum_DoesNotEscalateWhenDownloadSucceeds(t *testing.T) {
	// When `go mod download` already updates go.sum, RecoverGo must not run
	// the more expensive `go mod tidy` redundantly.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")
	mustWrite(t, filepath.Join(dir, "api", "go.sum"), "old\n")

	var calls int
	runner := func(_ context.Context, runDir, name string, args ...string) ([]byte, error) {
		calls++
		mustWrite(t, filepath.Join(runDir, "go.sum"), "rewritten\n")
		return nil, nil
	}
	res, err := RecoverGo(context.Background(), runner, dir, GoSignatureMissingSum)
	if err != nil {
		t.Fatalf("RecoverGo error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 runner call (download only), got %d", calls)
	}
	if res.Command != "go mod download" {
		t.Fatalf("Command should remain `go mod download` when no escalation, got %q", res.Command)
	}
}

func TestRecoverGo_MissingModule_RunsModTidy(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")

	var args []string
	runner := func(_ context.Context, _ string, _ string, a ...string) ([]byte, error) {
		args = a
		return nil, nil
	}
	_, err := RecoverGo(context.Background(), runner, dir, GoSignatureMissingModule)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(args, " ") != "mod tidy" {
		t.Fatalf("expected `go mod tidy`, got %v", args)
	}
}

func TestRecoverGo_NoSignatureIsNoop(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")
	called := false
	runner := func(context.Context, string, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}
	res, err := RecoverGo(context.Background(), runner, dir, GoSignatureNone)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatalf("runner should not have been called for GoSignatureNone")
	}
	if res.Kind != "" {
		t.Fatalf("expected empty result, got %+v", res)
	}
}

func TestRecoverGo_MissingGoModErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := RecoverGo(context.Background(), nil, dir, GoSignatureMissingSum); err == nil {
		t.Fatalf("expected error when go.mod absent")
	}
}

func TestRecoverGo_PropagatesRunnerError(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "api", "go.mod"), "module x\n")
	runner := func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte("failed"), errors.New("exit 1")
	}
	res, err := RecoverGo(context.Background(), runner, dir, GoSignatureMissingSum)
	if err != nil {
		t.Fatal(err)
	}
	if res.Err == nil || res.Err.Error() != "exit 1" {
		t.Fatalf("expected runner error to propagate via Result.Err, got %v", res.Err)
	}
}

func TestRecoverPnpm_OutdatedLockfile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "ui", "package.json"), "{}")
	mustWrite(t, filepath.Join(dir, "ui", "pnpm-lock.yaml"), "old\n")

	runner := func(_ context.Context, runDir, _ string, _ ...string) ([]byte, error) {
		mustWrite(t, filepath.Join(runDir, "pnpm-lock.yaml"), "new\n")
		return []byte("installed\n"), nil
	}
	res, err := RecoverPnpm(context.Background(), runner, dir, PnpmSignatureOutdatedLockfile)
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "pnpm install --ignore-workspace --no-frozen-lockfile" {
		t.Fatalf("unexpected command: %q", res.Command)
	}
	if !res.ModifiedTrackedFiles {
		t.Fatalf("expected ModifiedTrackedFiles=true")
	}
}

func TestRecoverPnpm_LinkingFailedRemovesNodeModules(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "ui", "package.json"), "{}")
	mustWrite(t, filepath.Join(dir, "ui", "node_modules", "marker"), "x")

	runner := func(context.Context, string, string, ...string) ([]byte, error) {
		return []byte("installed\n"), nil
	}
	res, err := RecoverPnpm(context.Background(), runner, dir, PnpmSignatureLinkingFailed)
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "ui", "node_modules")); !os.IsNotExist(statErr) {
		t.Fatalf("node_modules should have been removed, stat err = %v", statErr)
	}
	if res.Command != "pnpm install --ignore-workspace" {
		t.Fatalf("unexpected command: %q", res.Command)
	}
	if !containsText(res.Output, "rm -rf node_modules") {
		t.Fatalf("expected rm marker in output, got %q", res.Output)
	}
}

func TestDecideRepoRoot(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module vrooli\n")

	t.Run("go signature triggers repo-root decision", func(t *testing.T) {
		d := DecideRepoRoot("missing go.sum entry for module foo", dir)
		if d.Kind != KindRepoRoot || d.GoSig != GoSignatureMissingSum {
			t.Fatalf("got %+v", d)
		}
		if !d.Has() {
			t.Fatalf("expected Has=true")
		}
	})
	t.Run("no go.mod at root means no decision", func(t *testing.T) {
		empty := t.TempDir()
		if d := DecideRepoRoot("missing go.sum entry", empty); d.Has() {
			t.Fatalf("expected empty decision, got %+v", d)
		}
	})
	t.Run("no signature in log means no decision", func(t *testing.T) {
		if d := DecideRepoRoot("syntax error", dir); d.Has() {
			t.Fatalf("expected empty decision, got %+v", d)
		}
	})
}

func TestRecoverRepoRootGo_MissingSum_RunsModDownload(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module vrooli\n")
	mustWrite(t, filepath.Join(dir, "go.sum"), "old\n")

	var called struct {
		dir  string
		args []string
	}
	runner := func(_ context.Context, runDir, _ string, args ...string) ([]byte, error) {
		called.dir = runDir
		called.args = args
		mustWrite(t, filepath.Join(runDir, "go.sum"), "new\n")
		return []byte("downloaded\n"), nil
	}
	res, err := RecoverRepoRootGo(context.Background(), runner, dir, GoSignatureMissingSum)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if called.dir != dir {
		t.Fatalf("expected runner dir=%s, got %s", dir, called.dir)
	}
	if strings.Join(called.args, " ") != "mod download" {
		t.Fatalf("expected `go mod download`, got %v", called.args)
	}
	if res.Kind != KindRepoRoot {
		t.Fatalf("expected Kind=%q, got %q", KindRepoRoot, res.Kind)
	}
	if !res.ModifiedTrackedFiles || len(res.ModifiedPaths) != 1 || res.ModifiedPaths[0] != "go.sum" {
		t.Fatalf("expected go.sum mutation, got %+v", res.ModifiedPaths)
	}
}

func TestRecoverRepoRootGo_MissingModule_RunsModTidy(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module vrooli\n")
	var args []string
	runner := func(_ context.Context, _, _ string, a ...string) ([]byte, error) {
		args = a
		return nil, nil
	}
	if _, err := RecoverRepoRootGo(context.Background(), runner, dir, GoSignatureMissingModule); err != nil {
		t.Fatal(err)
	}
	if strings.Join(args, " ") != "mod tidy" {
		t.Fatalf("expected `go mod tidy`, got %v", args)
	}
}

func TestRecoverRepoRootGo_MissingGoModErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := RecoverRepoRootGo(context.Background(), nil, dir, GoSignatureMissingSum); err == nil {
		t.Fatalf("expected error when go.mod absent at repo root")
	}
}

func TestRecoverRepoRootGo_NoSignatureIsNoop(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module vrooli\n")
	called := false
	runner := func(context.Context, string, string, ...string) ([]byte, error) {
		called = true
		return nil, nil
	}
	res, err := RecoverRepoRootGo(context.Background(), runner, dir, GoSignatureNone)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("runner should not have been called")
	}
	if res.Kind != "" {
		t.Fatalf("expected empty result, got %+v", res)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
