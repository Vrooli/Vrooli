package permissions

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/agentharness"
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

// guardHost is a host with no ambient state: a home holding the repository,
// a separate temporary root, and an empty policy store.
type guardHost struct {
	env              GuardEnv
	home, repo, temp string
}

func newGuardHost(t *testing.T) guardHost {
	t.Helper()
	base := t.TempDir()
	h := guardHost{home: filepath.Join(base, "home"), temp: filepath.Join(base, "tmp")}
	h.repo = filepath.Join(h.home, "Vrooli")
	for _, dir := range []string{filepath.Join(h.repo, ".git"), filepath.Join(h.repo, "scratch"), filepath.Join(h.repo, "scenarios", "app"), h.temp} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	h.env = GuardEnv{
		Home: h.home,
		Runtime: agentharness.Runtime{
			Profile: agentharness.ProfileAdvisory,
			Store:   agentharness.NewBundleStore(filepath.Join(base, "policy")),
			Removal: func(cwd string) agentharness.RemovalContext {
				return agentharness.RemovalContext{
					WorkingDirectory: cwd, Home: h.home, RepoRoot: h.repo, ProjectConfigDir: ".vrooli", ScratchDir: "scratch",
					TempRoots: []string{h.temp}, Lookup: func(string) (string, bool) { return "", false }, Stat: os.Lstat,
				}
			},
		},
		LogPath: filepath.Join(base, "log"),
	}
	return h
}

type guardRun struct {
	exit           int
	stdout, stderr string
}

// runGuard exercises the full hook entrypoint, including JSON decoding and the
// exit-code contract Claude Code reads.
func runGuard(t *testing.T, env GuardEnv, cwd, mode, command string, patterns []string) guardRun {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"cwd": cwd, "permission_mode": mode, "tool_name": "Bash", "tool_input": map[string]string{"command": command}})
	if err != nil {
		t.Fatalf("marshal replay payload: %v", err)
	}
	var stdout, stderr bytes.Buffer
	exit := RunHookGuard(bytes.NewReader(payload), &stdout, &stderr, patterns, env)
	return guardRun{exit: exit, stdout: stdout.String(), stderr: stderr.String()}
}

func TestGuardDecidesDeletionThroughTheSharedRuntime(t *testing.T) {
	h := newGuardHost(t)
	cases := []struct {
		name, cwd, mode, command string
		wantExit                 int
		wantAsk                  bool
	}{
		{"filesystem root", h.repo, "default", "rm -rf /", GuardExitDeny, false},
		{"home", h.repo, "default", "rm -rf ~", GuardExitDeny, false},
		{"repository root", h.repo, "default", "rm -rf " + h.repo, GuardExitDeny, false},
		{"git history", h.repo, "default", "rm .git/index.lock", GuardExitDeny, false},
		{"temporary tree", h.repo, "default", "rm -rf " + filepath.Join(h.temp, "work"), GuardExitContinue, false},
		{"named repository file", h.repo, "bypassPermissions", "rm scenarios/web-console/ui/src/lib/viewportCorrection.ts", GuardExitContinue, false},
		{"compound deletion in temp", h.repo, "default", "cd " + h.temp + " && rm -rf build", GuardExitContinue, false},
		{"read-only find pipeline", h.repo, "default", "find . -name '*.go' | head", GuardExitContinue, false},
		{"deletion word in a search", h.repo, "default", "grep -rn 'rm -rf' docs | wc -l", GuardExitContinue, false},
		{"repository tree asks", h.repo, "default", "rm -r scenarios/app", GuardExitContinue, true},
		{"ask becomes deny when nobody can confirm", h.repo, "bypassPermissions", "rm -r scenarios/app", GuardExitDeny, false},
		{"ask becomes deny in an unknown mode", h.repo, "", "rm -r scenarios/app", GuardExitDeny, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runGuard(t, h.env, tc.cwd, tc.mode, tc.command, nil)
			if got.exit != tc.wantExit || strings.Contains(got.stdout, `"ask"`) != tc.wantAsk {
				t.Fatalf("%q exited %d (stdout %q, stderr %q), want exit %d ask=%v", tc.command, got.exit, got.stdout, got.stderr, tc.wantExit, tc.wantAsk)
			}
		})
	}
}

// TestGuardAsksThroughClaudesPermissionDecision pins the structured output
// Claude Code reads to raise its own confirmation prompt.
func TestGuardAsksThroughClaudesPermissionDecision(t *testing.T) {
	h := newGuardHost(t)
	got := runGuard(t, h.env, h.repo, "acceptEdits", "rm -r scenarios/app", nil)
	var output struct {
		HookSpecificOutput map[string]string `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &output); err != nil || got.exit != GuardExitContinue {
		t.Fatalf("exit %d stdout %q: %v", got.exit, got.stdout, err)
	}
	decision := output.HookSpecificOutput
	if decision["hookEventName"] != "PreToolUse" || decision["permissionDecision"] != "ask" || !strings.Contains(decision["permissionDecisionReason"], "recursive deletion") {
		t.Fatalf("permission decision = %v", decision)
	}
}

// TestRemovalDenyPatternsDeferToTheEngine keeps textual rm/find/truncate
// patterns from denying the locations the engine declares safe.
func TestRemovalDenyPatternsDeferToTheEngine(t *testing.T) {
	h := newGuardHost(t)
	patterns := []string{"Bash(rm -rf /*)", "Bash(find *)", "Bash(truncate *)"}
	for _, command := range []string{
		"rm -rf " + filepath.Join(h.temp, "x"),
		"find " + h.temp + " -name '*.log'",
		"truncate -s 0 " + filepath.Join(h.temp, "out.bin"),
	} {
		if got := runGuard(t, h.env, h.repo, "default", command, patterns); got.exit != GuardExitContinue {
			t.Errorf("%q exited %d (%s), want %d", command, got.exit, got.stderr, GuardExitContinue)
		}
	}
}

func TestGuardRejectsMalformedInput(t *testing.T) {
	h := newGuardHost(t)
	for name, payload := range map[string]string{
		"not json":       "not-json",
		"missing field":  `{"tool_input":{}}`,
		"empty command":  `{"tool_input":{"command":""}}`,
		"wrong type":     `{"tool_input":{"command":42}}`,
		"unclosed quote": `{"cwd":"/","tool_input":{"command":"rm -rf '"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if got := RunHookGuard(strings.NewReader(payload), io.Discard, io.Discard, nil, h.env); got != GuardExitDeny {
				t.Fatalf("payload %q exited %d, want %d", payload, got, GuardExitDeny)
			}
		})
	}
}

// TestGuardExplainsItsRefusal keeps the operator-facing reason on stderr, which
// is the only channel Claude surfaces when a hook denies a call.
func TestGuardExplainsItsRefusal(t *testing.T) {
	h := newGuardHost(t)
	got := runGuard(t, h.env, h.repo, "default", "rm -rf /etc", nil)
	if got.exit != GuardExitDeny || !strings.Contains(got.stderr, "directly under the filesystem root") {
		t.Fatalf("exit %d stderr %q, want a deny that names the reason", got.exit, got.stderr)
	}
}

// TestGuardRecordsDecisionsInItsAuditLog protects the operator-visible record
// of what the hook allowed, asked about, and refused.
func TestGuardRecordsDecisionsInItsAuditLog(t *testing.T) {
	h := newGuardHost(t)
	runGuard(t, h.env, h.repo, "default", "rm -rf /etc", nil)
	runGuard(t, h.env, h.repo, "default", "rm -r scenarios/app", nil)
	data, err := os.ReadFile(h.env.LogPath)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	for _, want := range []string{"BLOCKED", "rm -rf /etc", "ASK", "rm -r scenarios/app"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("audit log lacks %q: %q", want, string(data))
		}
	}
}

func TestBashDenyHookReplay(t *testing.T) {
	h := newGuardHost(t)
	cases := replayCases()
	patterns := make([]string, 0, len(cases))
	for _, testCase := range cases {
		patterns = append(patterns, testCase.pattern)
	}
	for _, testCase := range cases {
		if got := runGuard(t, h.env, h.repo, "default", testCase.block, patterns); got.exit != GuardExitDeny {
			t.Errorf("block case %q exited %d, want %d", testCase.block, got.exit, GuardExitDeny)
		}
		if got := runGuard(t, h.env, h.repo, "default", testCase.nearMiss, []string{testCase.pattern}); got.exit != GuardExitContinue {
			t.Errorf("near-miss case %q exited %d (%s), want %d", testCase.nearMiss, got.exit, got.stderr, GuardExitContinue)
		}
	}
}

// TestGuardIsShellFree is the portability guarantee: the hook must reach a
// decision with no interpreter on the host.
func TestGuardIsShellFree(t *testing.T) {
	h := newGuardHost(t)
	t.Setenv("PATH", t.TempDir())
	if got := runGuard(t, h.env, h.repo, "default", "rm -rf /etc", nil); got.exit != GuardExitDeny {
		t.Fatalf("guard needed an interpreter: exit %d, want %d", got.exit, GuardExitDeny)
	}
}
