package capabilityprobe

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A `#!/usr/bin/env node` tool whose interpreter is absent must say so. On
// minimouse (2026-09-16) codex and grok exited 127 with exactly this output
// while claude/opencode/agy — native binaries — reported versions fine. The
// probe called all three failures "its version could not be read", so the
// launcher told the operator to refresh a probe that was already reporting
// correctly, and never mentioned the runtime that was actually missing.
func TestVersionFailureDetailNamesTheMissingInterpreter(t *testing.T) {
	detail := versionFailureDetail("codex", "env: node: No such file or directory\n", errors.New("exit status 127"))
	for _, want := range []string{"codex", "node", "PATH"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail %q does not mention %q", detail, want)
		}
	}
	if strings.Contains(detail, "could not be read") {
		t.Fatalf("detail fell back to the opaque wording: %q", detail)
	}
}

// A hang and a refusal are different operator situations and must not share a
// message.
func TestVersionFailureDetailDistinguishesTimeoutFromExit(t *testing.T) {
	timeout := versionFailureDetail("grok", "", context.DeadlineExceeded)
	if !strings.Contains(timeout, "budget") {
		t.Fatalf("timeout detail = %q", timeout)
	}
	exited := versionFailureDetail("grok", "boom", errors.New("exit status 2"))
	if strings.Contains(exited, "budget") || !strings.Contains(exited, "boom") {
		t.Fatalf("exit detail = %q, want the command's own output and no timeout wording", exited)
	}
}

// Only a shebang loader's "env: X: No such file or directory" is an
// interpreter fault. Arbitrary output that merely mentions a file must not be
// reported as a missing runtime.
func TestMissingInterpreterMatchesOnlyTheLoaderMessage(t *testing.T) {
	if name, ok := missingInterpreter("env: node: No such file or directory"); !ok || name != "node" {
		t.Fatalf("interpreter = %q, %t", name, ok)
	}
	for _, line := range []string{
		"config.json: No such file or directory",
		"env: ",
		"error: something else entirely",
	} {
		if _, ok := missingInterpreter(line); ok {
			t.Fatalf("%q was misread as a missing interpreter", line)
		}
	}
}

// The probe must widen PATH so an interpreted tool resolves its runtime, while
// leaving the operator's own entries in front of ours.
func TestManagedEnvironWidensPathWithoutShadowingExistingEntries(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	var path string
	for _, item := range managedEnviron() {
		if value, ok := strings.CutPrefix(item, "PATH="); ok {
			path = value
		}
	}
	if !strings.HasPrefix(path, "/usr/bin:/bin") {
		t.Fatalf("managed PATH %q reordered the inherited entries", path)
	}
	if !strings.Contains(path, "/usr/local/bin") {
		t.Fatalf("managed PATH %q lacks /usr/local/bin, where a Homebrew node lives", path)
	}
	if strings.Count(path, "/usr/bin:") > 1 {
		t.Fatalf("managed PATH %q duplicated an inherited entry", path)
	}
}

// ManagedLookPath and the probe's child PATH must stay one list: a tool we can
// find whose interpreter we cannot is the bug this package just shipped.
func TestManagedLookPathSearchesTheSameEntriesAsTheChildPath(t *testing.T) {
	home := t.TempDir()
	entries := ManagedPathEntries(home)
	for _, want := range []string{
		"/usr/local/bin",
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".vrooli", "bin"),
	} {
		if !containsEntry(entries, want) {
			t.Fatalf("managed entries %v lack %q", entries, want)
		}
	}
}

func containsEntry(entries []string, want string) bool {
	for _, entry := range entries {
		if entry == want {
			return true
		}
	}
	return false
}

// Every version command is bounded on its own. ProbeWith always supplies a
// context carrying the whole-probe deadline, so the old `if !hasDeadline`
// guard disabled the per-command bound on the only path that used it: one
// hanging tool could burn the entire budget and leave the tools after it
// unreadable.
func TestRunVersionBoundsEachCommandUnderACallerDeadline(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("sleep is not available on this host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	start := time.Now()
	if _, err := runVersion(ctx, sleep, []string{"60"}); err == nil {
		t.Fatal("a hanging version command returned success")
	}
	if elapsed := time.Since(start); elapsed > DefaultCommandTimeout+5*time.Second {
		t.Fatalf("version command ran %s; the per-command bound did not apply", elapsed)
	}
	if ctx.Err() != nil {
		t.Fatal("the caller's context was cancelled; the per-command bound must not consume it")
	}
}

// End to end: a tool that is an interpreted script reports the runtime, not a
// shrug. The interpreter is deliberately absent from every managed directory.
func TestProbeReportsAnInterpretedToolsMissingRuntime(t *testing.T) {
	if _, err := exec.LookPath("env"); err != nil {
		t.Skip("env is not available on this host")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "faketool")
	contents := "#!/usr/bin/env vrooli-absent-interpreter\n"
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil { //nolint:gosec // an executable fixture must be executable
		t.Fatal(err)
	}
	got := ProbeWith(context.Background(),
		[]Definition{{Capability: "ai-cli", ID: "faketool", Command: script, VersionArg: []string{"--version"}}},
		func(string) (string, error) { return script, nil }, runVersion, time.Now)
	if len(got) != 1 || got[0].State != Unknown {
		t.Fatalf("observation = %+v", got)
	}
	if !strings.Contains(got[0].Detail, "vrooli-absent-interpreter") {
		t.Fatalf("detail %q does not name the missing interpreter", got[0].Detail)
	}
}
