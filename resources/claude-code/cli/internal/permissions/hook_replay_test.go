package permissions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type denyReplayCase struct {
	pattern  string
	block    string
	nearMiss string
}

// replayCases deliberately contains command text only. The replay harness
// passes each string as JSON to the hook executable; it never invokes a shell
// with any of these values.
func replayCases(home string) []denyReplayCase {
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
		{pattern: "Bash(rm -rf $HOME*)", block: "rm -rf " + home + "/policy-probe", nearMiss: "rm -r " + home + "/policy-probe"},
		{pattern: "Bash(rm -rf /*)", block: "rm -rf /tmp/policy-probe", nearMiss: "rm -r /tmp/policy-probe"},
		{pattern: "Bash(rm -rf ~*)", block: "rm -rf " + home + "/policy-probe", nearMiss: "rm -rf /tmp/policy-probe"},
		{pattern: "Bash(shutdown*)", block: "shutdown --dry-run", nearMiss: "shut-down --dry-run"},
		{pattern: "Bash(sudo rm -rf*)", block: "sudo rm -rf /tmp/policy-probe", nearMiss: "sudo rm -r /tmp/policy-probe"},
		{pattern: "Bash(systemctl poweroff*)", block: "systemctl poweroff --dry-run", nearMiss: "systemctl status"},
		{pattern: "Bash(systemctl reboot*)", block: "systemctl reboot --dry-run", nearMiss: "systemctl restart --dry-run"},
	}
}

func TestBashDenyHookReplay(t *testing.T) {
	adapter := newTestAdapter(t)
	if err := adapter.installHookScript(); err != nil {
		t.Fatalf("install hook script: %v", err)
	}
	home := t.TempDir()
	cases := replayCases(home)
	patterns := make([]string, 0, len(cases))
	for _, testCase := range cases {
		patterns = append(patterns, testCase.pattern)
	}

	blockPasses := 0
	nearMissPasses := 0
	for _, testCase := range cases {
		if got := invokeHook(t, adapter.HookScriptPath(), home, testCase.block, patterns); got == 2 {
			blockPasses++
		} else {
			t.Errorf("block case %q exited %d, want 2", testCase.block, got)
		}
		if got := invokeHook(t, adapter.HookScriptPath(), home, testCase.nearMiss, []string{testCase.pattern}); got == 0 {
			nearMissPasses++
		} else {
			t.Errorf("near-miss case %q exited %d, want 0", testCase.nearMiss, got)
		}
	}
	t.Logf("replay result: block %d/%d, near-miss %d/%d", blockPasses, len(cases), nearMissPasses, len(cases))
}

func invokeHook(t *testing.T, scriptPath, home, command string, patterns []string) int {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"tool_input": map[string]string{"command": command}})
	if err != nil {
		t.Fatalf("marshal replay payload: %v", err)
	}
	cmd := exec.Command(scriptPath, patterns...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Env = replaceEnv(os.Environ(), "HOME", home)
	err = cmd.Run()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	t.Fatalf("run hook for %q: %v", command, err)
	return -1
}

func replaceEnv(environment []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			continue
		}
		result = append(result, entry)
	}
	return append(result, fmt.Sprintf("%s=%s", key, filepath.Clean(value)))
}
