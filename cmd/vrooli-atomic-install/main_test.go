package main

import (
	"bytes"
	"flag"
	"os"
	osuser "os/user"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const snapshotCommandMarkerEnv = "VROOLI_ATOMIC_INSTALL_TEST_SNAPSHOT_COMMAND_MARKER"

func TestSnapshotCommandChildExitHelper(t *testing.T) {
	if flag.Lookup("test.run").Value.String() != "^TestSnapshotCommandChildExitHelper$" {
		return
	}
	marker := os.Getenv(snapshotCommandMarkerEnv)
	if marker == "" {
		return
	}
	if err := os.WriteFile(marker, []byte("child ran"), 0600); err != nil {
		t.Fatal(err)
	}
	os.Exit(23)
}

func TestSnapshotLockExecDispatchPreservesChildExitCode(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "child-ran")
	t.Setenv(snapshotCommandMarkerEnv, "")
	t.Setenv(snapshotCommandMarkerEnv, marker)
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	args := []string{"installer", "--snapshot-lock-exec", dir, "--", executable, "-test.run=^TestSnapshotCommandChildExitHelper$"}
	if code := run(args, &stderr); code != 23 {
		t.Fatalf("run() exit code = %d, want child exit code 23; stderr=%q", code, stderr.String())
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "child ran" {
		t.Fatalf("snapshot command child marker = %q, error = %v", got, err)
	}
}

func TestParseInvocationAcceptedShapesSelectOneMode(t *testing.T) {
	tests := []struct {
		name string
		args []string
		mode invocationMode
	}{
		{"snapshot root", []string{"installer", "--snapshot-root"}, modeSnapshotRoot},
		{"abort bundle", []string{"installer", "--abort-bundle", "bin"}, modeAbortBundle},
		{"resume bundle", []string{"installer", "--resume-bundle", "bin"}, modeResumeBundle},
		{"companions", []string{"installer", "--companions", "src", "bin", "hash"}, modeCompanions},
		{"prepare bundle", []string{"installer", "--prepare-bundle", "src", "out"}, modePrepareBundle},
		{"prepare hub bundle", []string{"installer", "--prepare-hub-bundle", "src", "out", "pointer"}, modePrepareHubBundle},
		{"verify bundle", []string{"installer", "--verify-bundle", "src", "hash"}, modeVerifyBundle},
		{"prepared companions", []string{"installer", "--prepared-companions", "src", "bin", "root-hash", "manifest-hash"}, modePreparedCompanions},
		{"install bundle", []string{"installer", "--install-bundle", "bundle", "bin", "manifest"}, modeInstallBundle},
		{"install bundle with source", []string{"installer", "--install-bundle", "bundle", "bin", "manifest", "source"}, modeInstallBundle},
		{"publish source", []string{"installer", "--publish-source", "bundle", "bin", "manifest", "source", "pointer"}, modePublishSource},
		{"verify runtime paths", []string{"installer", "--verify-runtime-paths", "bin", "pointer"}, modeVerifyRuntimePaths},
		{"install project bundle", []string{"installer", "--install-project-bundle", "source", "bin", "pointer", "checkout"}, modeInstallProjectBundle},
		{"owned bundle", []string{"installer", "--install-bundle-owned", "bundle", "manifest", "source"}, modeInstallBundleOwned},
		{"owned publication", []string{"installer", "--publish-source-owned", "bundle", "manifest", "source"}, modePublishSourceOwned},
		{"snapshot lock exec", []string{"installer", "--snapshot-lock-exec", "root", "--", "cmd"}, modeSnapshotLockExec},
		{"snapshot lock exec args", []string{"installer", "--snapshot-lock-exec", "root", "--", "cmd", "arg"}, modeSnapshotLockExec},
		{"snapshot lease exec", []string{"installer", "--snapshot-lease-exec", "root", "snapshot", "--", "cmd"}, modeSnapshotLeaseExec},
		{"try snapshot lease exec", []string{"installer", "--try-snapshot-lease-exec", "root", "snapshot", "--", "cmd"}, modeTrySnapshotLeaseExec},
		{"prune install snapshots", []string{"installer", "--prune-install-snapshots", "root"}, modePruneInstallSnapshots},
		{"prune hub snapshots", []string{"installer", "--prune-hub-snapshots", "root", "current", "legacy", "--", "retire"}, modePruneHubSnapshots},
		{"prune hub snapshots args", []string{"installer", "--prune-hub-snapshots", "root", "current", "legacy", "--", "retire", "arg"}, modePruneHubSnapshots},
		{"positional install", []string{"installer", "source", "dest"}, modeAtomicInstall},
		{"positional install with current hash", []string{"installer", "source", "dest", "hash"}, modeAtomicInstallExpected},
		{"legacy three argument fallback", []string{"installer", "--unknown", "dest"}, modeAtomicInstall},
		{"legacy four argument fallback", []string{"installer", "--unknown", "dest", "hash"}, modeAtomicInstallExpected},
		{"legacy snapshot root fallback", []string{"installer", "--snapshot-root", "extra"}, modeAtomicInstall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			call, err := parseInvocation(tt.args)
			if err != nil {
				t.Fatalf("parseInvocation() error = %v", err)
			}
			if call.mode != tt.mode {
				t.Fatalf("parseInvocation() mode = %d, want %d", call.mode, tt.mode)
			}
			if call.mode == modeInvalid || len(call.args) == 0 {
				t.Fatalf("parsed invocation is incomplete: %+v", call)
			}
		})
	}
}

func TestParseInvocationRejectsUnsupportedShapes(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no mode", []string{"installer"}},
		{"unknown two argument form", []string{"installer", "--unknown"}},
		{"unmatched five argument form", []string{"installer", "--unknown", "a", "b", "c"}},
		{"unmatched six argument form", []string{"installer", "--unknown", "a", "b", "c", "d"}},
		{"unmatched seven argument form", []string{"installer", "--unknown", "a", "b", "c", "d", "e"}},
		{"snapshot lock separator in wrong position", []string{"installer", "--snapshot-lock-exec", "root", "wrong", "cmd"}},
		{"snapshot lease separator in wrong position", []string{"installer", "--snapshot-lease-exec", "root", "snapshot", "wrong", "cmd"}},
		{"prune hub separator in wrong position", []string{"installer", "--prune-hub-snapshots", "root", "current", "legacy", "wrong", "retire"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if call, err := parseInvocation(tt.args); err == nil {
				t.Fatalf("parseInvocation() = %+v, want error", call)
			}
		})
	}
}

func TestInvalidCompiledAuthorityRefusesBeforeAtomicMutation(t *testing.T) {
	old := buildinfo.RuntimeHomeSnapshotBase64
	buildinfo.RuntimeHomeSnapshotBase64 = "invalid"
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "candidate"), filepath.Join(dir, "installed")
	if err := os.WriteFile(src, []byte("candidate"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("prior"), 0700); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := run([]string{"installer", src, dst}, &stderr); code != 1 {
		t.Fatalf("invalid stamped authority accepted: code%d", code)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "prior" {
		t.Fatal("invalid authority mutated installed image")
	}
}
func TestUsagePreservesExitAndDoesNotInitializeAuthority(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"installer"}, &stderr); code != usageExitCode {
		t.Fatal(code)
	}
}

func TestOwnedRuntimePathsUseInvokingHome(t *testing.T) {
	old := hostreqkit.RunningAsRootFn
	t.Cleanup(func() { hostreqkit.RunningAsRootFn = old })
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	hostreqkit.RunningAsRootFn = func() bool { return false }
	bin, pointer, err := ownedRuntimePaths()
	if err != nil {
		t.Fatal(err)
	}
	if bin != filepath.Join(home, ".vrooli", "bin") || pointer != filepath.Join(home, ".vrooli", "source-root") {
		t.Fatalf("ordinary paths: %s %s", bin, pointer)
	}
	account, err := osuser.Lookup("nobody")
	if err != nil || account.HomeDir == "" {
		t.Skip("no non-root passwd fixture")
	}
	hostreqkit.RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "nobody")
	t.Setenv("HOME", t.TempDir())
	bin, pointer, err = ownedRuntimePaths()
	if err != nil {
		t.Fatal(err)
	}
	if bin != filepath.Join(account.HomeDir, ".vrooli", "bin") || pointer != filepath.Join(account.HomeDir, ".vrooli", "source-root") {
		t.Fatalf("sudo invoking paths: %s %s", bin, pointer)
	}
}

func TestRuntimePathOverridesRejectNoncanonicalBeforeEffects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	var output bytes.Buffer
	if code := run([]string{"installer", "--verify-runtime-paths", filepath.Join(home, "other"), ""}, &output); code != 1 {
		t.Fatalf("noncanonical destination accepted: %d %s", code, output.String())
	}
	output.Reset()
	if code := run([]string{"installer", "--verify-runtime-paths", "", filepath.Join(home, "other")}, &output); code != 1 {
		t.Fatalf("noncanonical pointer accepted: %d %s", code, output.String())
	}
	output.Reset()
	if code := run([]string{"installer", "--verify-runtime-paths", filepath.Join(home, ".vrooli", "bin"), filepath.Join(home, ".vrooli", "source-root")}, &output); code != 0 {
		t.Fatalf("canonical overrides rejected: %d %s", code, output.String())
	}
}
