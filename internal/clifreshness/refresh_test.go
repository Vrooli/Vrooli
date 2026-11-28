package clifreshness

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type refreshCalls struct {
	built      int
	execd      int
	activation bool
	execPath   string
}

// stubRefresh replaces the refresh seams for one test. checkout is the digest
// the checkout declares; embedded is the digest this build carries.
func stubRefresh(t *testing.T, checkout, embedded string, buildErr error) *refreshCalls {
	t.Helper()
	calls := &refreshCalls{}
	previousCheckout, previousEmbedded := surfaceDigestFn, embeddedDigestFn
	previousBuild, previousActivation, previousExec, previousExe := buildLocalCLI, activationActive, runRefreshedCLI, executablePath
	t.Cleanup(func() {
		surfaceDigestFn, embeddedDigestFn = previousCheckout, previousEmbedded
		buildLocalCLI, activationActive, runRefreshedCLI, executablePath = previousBuild, previousActivation, previousExec, previousExe
	})
	surfaceDigestFn = func(string) (string, error) { return checkout, nil }
	embeddedDigestFn = func() (string, error) { return embedded, nil }
	buildLocalCLI = func(_ context.Context, _, _ string) error {
		calls.built++
		return buildErr
	}
	activationActive = func() bool { return calls.activation }
	executablePath = func() (string, error) { return calls.execPath, nil }
	runRefreshedCLI = func(local string, _ []string, _ io.Writer) (int, bool) {
		calls.execd++
		_ = local
		return 0, true
	}
	t.Setenv(EnvDepth, "")
	t.Setenv(EnvDisable, "")
	for _, key := range []string{"INVOCATION_ID", "NOTIFY_SOCKET", "JOURNAL_STREAM", "VROOLI_RUNTIME_SUPERVISOR", "LAUNCH_JOB_NAME", "XPC_SERVICE_NAME"} {
		t.Setenv(key, "")
	}
	return calls
}

// installedProcess makes the running executable look like the installed image,
// not the repository build.
func installedProcess(calls *refreshCalls) { calls.execPath = "/usr/local/bin/vrooli" }

func TestMaybeReexecDoesNothingWhenTheBuildServesTheCheckout(t *testing.T) {
	calls := stubRefresh(t, "same", "same", nil)
	installedProcess(calls)
	var stderr bytes.Buffer
	if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &stderr); refreshed {
		t.Fatal("a current build was refreshed")
	}
	if calls.built != 0 || calls.execd != 0 || stderr.Len() != 0 {
		t.Fatalf("a current build did work: %+v stderr=%q", calls, stderr.String())
	}
}

func TestMaybeReexecRefreshesAndReportsOneLine(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", nil)
	installedProcess(calls)
	root := t.TempDir()
	var stderr bytes.Buffer
	if _, refreshed := MaybeReexec(context.Background(), root, []string{"status"}, &stderr); !refreshed {
		t.Fatal("a stale build was not refreshed")
	}
	if calls.built != 1 || calls.execd != 1 {
		t.Fatalf("refresh did not build and exec once: %+v", calls)
	}
	if strings.Count(strings.TrimSpace(stderr.String()), "\n") != 0 {
		t.Fatalf("a refresh printed more than one line: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "older than cli/manifest.json") {
		t.Fatalf("the one line does not explain why: %q", stderr.String())
	}
	if published := publishedSurface(LocalCLIPath(root)); published != "checkout" {
		t.Fatalf("published surface = %q; a sibling would rebuild again", published)
	}
}

// A published build for the same surface must not be rebuilt again; otherwise
// every invocation of the installed image would pay for a build.
func TestMaybeReexecReusesAPublishedBuildWithoutBuilding(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", nil)
	installedProcess(calls)
	root := t.TempDir()
	local := LocalCLIPath(root)
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local+".surface", []byte("checkout\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if _, refreshed := MaybeReexec(context.Background(), root, []string{"status"}, &stderr); !refreshed {
		t.Fatal("a published build was not used")
	}
	if calls.built != 0 || calls.execd != 1 {
		t.Fatalf("a published build was rebuilt: %+v", calls)
	}
	if stderr.Len() != 0 {
		t.Fatalf("reusing a published build was noisy: %q", stderr.String())
	}
}

func TestMaybeReexecKeepsTheCurrentBuildWhenTheSourceDoesNotCompile(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", errors.New("internal/lifecycle: undefined: thing"))
	installedProcess(calls)
	var stderr bytes.Buffer
	if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &stderr); refreshed {
		t.Fatal("a failed refresh replaced the process")
	}
	if calls.execd != 0 || calls.built != 1 {
		t.Fatalf("failed refresh did not fall back: %+v", calls)
	}
	if !strings.Contains(stderr.String(), "continuing with this build") || !strings.Contains(stderr.String(), "undefined") {
		t.Fatalf("the failure was not explained in one line: %q", stderr.String())
	}
}

func TestMaybeReexecNeverRefreshesDuringActivation(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", nil)
	installedProcess(calls)
	calls.activation = true
	var stderr bytes.Buffer
	if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &stderr); refreshed {
		t.Fatal("a refresh ran during a root activation")
	}
	if calls.built != 0 {
		t.Fatal("a refresh built during a root activation")
	}
}

func TestMaybeReexecHonoursTheLoopGuardAndKillSwitch(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", nil)
	installedProcess(calls)
	t.Setenv(EnvDepth, "1")
	if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &bytes.Buffer{}); refreshed {
		t.Fatal("a process that already refreshed refreshed again")
	}
	t.Setenv(EnvDepth, "")
	t.Setenv(EnvDisable, "0")
	if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &bytes.Buffer{}); refreshed {
		t.Fatal("the kill switch was ignored")
	}
	if calls.built != 0 {
		t.Fatalf("a disabled refresh did work: %+v", calls)
	}
}

func TestSkipForCommandLeavesBuildingServicesAndHelpAlone(t *testing.T) {
	for _, args := range [][]string{
		nil, {"version"}, {"--version"}, {"--help"}, {"-h"}, {"help"}, {"completion"},
		{"build"}, {"install"}, {"develop"}, {"setup"}, {"__privilege-broker"}, {"--json", "install"},
		{"runtime", "supervisor", "run"}, {"runtime", "watchdog"}, {"runtime", "update", "rebaseline", "x"},
		{"agent", "launch", "opencode"}, {"agent", "exec", "run"}, {"agent", "review"}, {"watchdog", "run"},
	} {
		if !skipForCommand(args) {
			t.Fatalf("refused to skip %v", args)
		}
	}
	for _, args := range [][]string{{"status"}, {"agent", "list"}, {"--json", "status"}, {"scenario", "list"}, {"hygiene", "report"}} {
		if skipForCommand(args) {
			t.Fatalf("skipped %v", args)
		}
	}
}

// A service manager marks its children. Re-executing one replaces the image its
// unit identifies as the MainPID's image, which is exactly the condition a root
// transition's verification refuses.
func TestServiceContextIsNeverRefreshed(t *testing.T) {
	calls := stubRefresh(t, "checkout", "embedded", nil)
	calls.execPath = "/usr/local/bin/vrooli"
	for _, key := range []string{"INVOCATION_ID", "NOTIFY_SOCKET", "JOURNAL_STREAM", "VROOLI_RUNTIME_SUPERVISOR", "LAUNCH_JOB_NAME", "XPC_SERVICE_NAME"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "set")
			var stderr bytes.Buffer
			if _, refreshed := MaybeReexec(context.Background(), t.TempDir(), []string{"status"}, &stderr); refreshed {
				t.Fatalf("%s: a service process re-executed", key)
			}
			if calls.built != 0 {
				t.Fatalf("%s: a service process built a new CLI", key)
			}
		})
	}
}
