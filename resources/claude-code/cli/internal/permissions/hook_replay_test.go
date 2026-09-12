package permissions

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type denyReplayCase struct {
	pattern  string
	block    string
	nearMiss string
}

// replayCases deliberately contains command text only. The replay harness
// hands each string to the guard as JSON data; it never invokes a shell with
// any of these values.
func replayCases() []denyReplayCase {
	return []denyReplayCase{
		{pattern: "Bash(dd *of=/dev/sd*)", block: "dd if=/dev/zero of=/dev/sda", nearMiss: "dd if=/dev/zero of=/tmp/policy-probe"},
		{pattern: "Bash(git branch --delete --force*)", block: "git branch --delete --force old-branch", nearMiss: "git branch --delete old-branch"},
		{pattern: "Bash(git branch -D*)", block: "git branch -D old-branch", nearMiss: "git branch --delete old-branch"},
		{pattern: "Bash(git checkout .*)", block: "git checkout .", nearMiss: "git checkout README.md"},
		{pattern: "Bash(git clean -f*)", block: "git clean -f .", nearMiss: "git clean -n ."},
		{pattern: "Bash(git commit*)", block: "git commit --dry-run -m policy-probe", nearMiss: "git status"},
		{pattern: "Bash(git push*)", block: "git push origin main", nearMiss: "git fetch origin main"},
		{pattern: "Bash(git rebase*)", block: "git rebase main", nearMiss: "git merge main"},
		{pattern: "Bash(git reset --hard*)", block: "git reset --hard HEAD", nearMiss: "git reset --soft HEAD"},
		{pattern: "Bash(git reset --merge*)", block: "git reset --merge HEAD", nearMiss: "git reset --keep HEAD"},
		{pattern: "Bash(git restore .*)", block: "git restore .", nearMiss: "git restore --staged README.md"},
		{pattern: "Bash(git revert*)", block: "git revert HEAD", nearMiss: "git log -1"},
		{pattern: "Bash(git stash*)", block: "git stash push --message policy-probe", nearMiss: "git status"},
		{pattern: "Bash(git worktree add*)", block: "git worktree add /tmp/policy-probe-tree HEAD", nearMiss: "git worktree list"},
		{pattern: "Bash(mkfs*)", block: "mkfs --dry-run /dev/sda", nearMiss: "makefs --dry-run"},
		{pattern: "Bash(reboot*)", block: "reboot --dry-run", nearMiss: "reb00t --dry-run"},
		{pattern: "Bash(shutdown*)", block: "shutdown --dry-run", nearMiss: "shut-down --dry-run"},
		{pattern: "Bash(systemctl poweroff*)", block: "systemctl poweroff --dry-run", nearMiss: "systemctl status"},
		{pattern: "Bash(systemctl reboot*)", block: "systemctl reboot --dry-run", nearMiss: "systemctl restart --dry-run"},
	}
}

// testGuardEnv builds a guard context with no ambient host state, so a
// decision depends only on what the case declares.
func testGuardEnv(t *testing.T, home, repo string, roots ...string) GuardEnv {
	t.Helper()
	return GuardEnv{
		Home:           home,
		RepoRoot:       repo,
		EphemeralRoots: roots,
		Lookup:         func(string) (string, bool) { return "", false },
		LogPath:        filepath.Join(t.TempDir(), "log"),
	}
}

// runGuard exercises the full hook entrypoint, including JSON decoding and the
// exit-code contract Claude Code reads.
func runGuard(t *testing.T, env GuardEnv, command string, patterns []string) int {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": command}})
	if err != nil {
		t.Fatalf("marshal replay payload: %v", err)
	}
	return RunHookGuard(bytes.NewReader(payload), io.Discard, patterns, env)
}

func TestPathAwareDestructiveHook(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	safe := filepath.Join(t.TempDir(), "ephemeral-review")
	env := testGuardEnv(t, home, repo, os.TempDir())
	patterns := []string{"Bash(rm -rf /*)", "Bash(find *)", "Bash(truncate *)"}

	denyCases := []string{
		"rm -rf /",
		"rm -rf /etc",
		"rm -rf " + home,
		"rm -rf " + filepath.Join(home, "nested"),
		"rm -rf " + repo,
		"sudo rm -rf /",
		"find / -delete",
		"find " + repo + " -delete",
		"truncate -s 0 /etc/hosts",
		"rm -rf $UNSET_VAR",
		"rm -rf /tmp/*",
		"rm -rf " + safe + " && rm -rf /",
		// An unquoted interpreter invocation is refused. A quoted one
		// ("bash -c 'rm -rf /'") is not detected, because the destructive
		// scan requires whitespace before the verb. That gap predates the Go
		// port, which is behaviour-preserving on purpose.
		"bash -c rm -rf /",
		"rm",
	}
	for _, command := range denyCases {
		if got := runGuard(t, env, command, patterns); got != GuardExitDeny {
			t.Errorf("deny case %q exited %d, want %d", command, got, GuardExitDeny)
		}
	}

	allowCases := []string{
		"rm -rf " + safe,
		"rm -r " + safe,
		"sudo rm -rf " + safe,
		"rm -rf -- " + safe,
		"find " + safe + " -delete",
		"find " + safe + " -name '*.log'",
		"truncate -s 0 " + filepath.Join(safe, "output.bin"),
		"ls -la " + home,
	}
	for _, command := range allowCases {
		if got := runGuard(t, env, command, patterns); got != GuardExitContinue {
			t.Errorf("allow case %q exited %d, want %d", command, got, GuardExitContinue)
		}
	}
}

// TestGuardResolvesEnvironmentVariablesBeforeDeciding proves a path is judged
// by what it resolves to, not by the text the agent typed.
func TestGuardResolvesEnvironmentVariablesBeforeDeciding(t *testing.T) {
	home := t.TempDir()
	safe := filepath.Join(t.TempDir(), "workspace")
	env := testGuardEnv(t, home, "", os.TempDir())
	env.Lookup = func(name string) (string, bool) {
		switch name {
		case "SAFE_DIR":
			return safe, true
		case "HOME_DIR":
			return home, true
		}
		return "", false
	}
	if got := runGuard(t, env, "rm -rf $SAFE_DIR", nil); got != GuardExitContinue {
		t.Errorf("resolved ephemeral target exited %d, want %d", got, GuardExitContinue)
	}
	if got := runGuard(t, env, "rm -rf ${HOME_DIR}", nil); got != GuardExitDeny {
		t.Errorf("resolved protected target exited %d, want %d", got, GuardExitDeny)
	}
}

func TestPathAwareHookRejectsMalformedInput(t *testing.T) {
	env := testGuardEnv(t, t.TempDir(), "", os.TempDir())
	for name, payload := range map[string]string{
		"not json":       "not-json",
		"missing field":  `{"tool_input":{}}`,
		"empty command":  `{"tool_input":{"command":""}}`,
		"wrong type":     `{"tool_input":{"command":42}}`,
		"unclosed quote": `{"tool_input":{"command":"rm -rf '"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := RunHookGuard(strings.NewReader(payload), io.Discard, []string{"Bash(rm -rf /*)"}, env); got != GuardExitDeny {
				t.Fatalf("payload %q exited %d, want %d", payload, got, GuardExitDeny)
			}
		})
	}
}

// TestGuardExplainsItsRefusal keeps the operator-facing reason on stderr, which
// is the only channel Claude surfaces when a hook denies a call.
func TestGuardExplainsItsRefusal(t *testing.T) {
	env := testGuardEnv(t, t.TempDir(), "", os.TempDir())
	payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": "rm -rf /etc"}})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var stderr bytes.Buffer
	if got := RunHookGuard(bytes.NewReader(payload), &stderr, nil, env); got != GuardExitDeny {
		t.Fatalf("exit = %d, want %d", got, GuardExitDeny)
	}
	if !strings.Contains(stderr.String(), "depth-one system directory") {
		t.Fatalf("stderr does not explain the refusal: %q", stderr.String())
	}
}

// TestGuardRecordsDecisionsInItsAuditLog protects the operator-visible record
// of what the hook allowed and refused.
func TestGuardRecordsDecisionsInItsAuditLog(t *testing.T) {
	env := testGuardEnv(t, t.TempDir(), "", os.TempDir())
	if got := runGuard(t, env, "rm -rf /etc", nil); got != GuardExitDeny {
		t.Fatalf("exit = %d, want %d", got, GuardExitDeny)
	}
	data, err := os.ReadFile(env.LogPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if !strings.Contains(string(data), "BLOCKED") || !strings.Contains(string(data), "rm -rf /etc") {
		t.Fatalf("audit log does not record the refusal: %q", string(data))
	}
}

func TestBashDenyHookReplay(t *testing.T) {
	env := testGuardEnv(t, t.TempDir(), "", os.TempDir())
	cases := replayCases()
	patterns := make([]string, 0, len(cases))
	for _, testCase := range cases {
		patterns = append(patterns, testCase.pattern)
	}

	for _, testCase := range cases {
		if got := runGuard(t, env, testCase.block, patterns); got != GuardExitDeny {
			t.Errorf("block case %q exited %d, want %d", testCase.block, got, GuardExitDeny)
		}
		if got := runGuard(t, env, testCase.nearMiss, []string{testCase.pattern}); got != GuardExitContinue {
			t.Errorf("near-miss case %q exited %d, want %d", testCase.nearMiss, got, GuardExitContinue)
		}
	}
}

// TestGuardIsShellFree is the portability guarantee: the deny hook must reach a
// decision with no interpreter on the host.
func TestGuardIsShellFree(t *testing.T) {
	env := testGuardEnv(t, t.TempDir(), "", os.TempDir())
	env.Lookup = func(string) (string, bool) { return "", false }
	t.Setenv("PATH", t.TempDir())
	if got := runGuard(t, env, "rm -rf /etc", nil); got != GuardExitDeny {
		t.Fatalf("guard needed an interpreter: exit %d, want %d", got, GuardExitDeny)
	}
}
