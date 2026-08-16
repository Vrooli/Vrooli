package hostreqkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvokingUserPrefersSudoUserWhenRoot(t *testing.T) {
	origRoot := RunningAsRootFn
	defer func() { RunningAsRootFn = origRoot }()
	RunningAsRootFn = func() bool { return true }

	t.Setenv("SUDO_USER", "alice")
	t.Setenv("USER", "root")

	if got := InvokingUser(); got != "alice" {
		t.Fatalf("InvokingUser() = %q, want alice", got)
	}
}

func TestInvokingUserIDsReadsSudoEnvWhenRoot(t *testing.T) {
	origRoot := RunningAsRootFn
	defer func() { RunningAsRootFn = origRoot }()
	RunningAsRootFn = func() bool { return true }

	t.Setenv("SUDO_USER", "alice")
	t.Setenv("SUDO_UID", "1000")
	t.Setenv("SUDO_GID", "1001")

	uid, gid, ok := InvokingUserIDs()
	if !ok {
		t.Fatal("InvokingUserIDs ok = false, want true")
	}
	if uid != 1000 || gid != 1001 {
		t.Fatalf("InvokingUserIDs = (%d,%d), want (1000,1001)", uid, gid)
	}
}

func TestInvokingUserIDsNotOkCases(t *testing.T) {
	cases := []struct {
		name    string
		root    bool
		sudoUsr string
		uid     string
		gid     string
	}{
		{"not root", false, "alice", "1000", "1001"},
		{"root without sudo", true, "", "1000", "1001"},
		{"missing uid", true, "alice", "", "1001"},
		{"non-numeric uid", true, "alice", "x", "1001"},
		{"root uid rejected", true, "alice", "0", "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origRoot := RunningAsRootFn
			defer func() { RunningAsRootFn = origRoot }()
			RunningAsRootFn = func() bool { return tc.root }
			t.Setenv("SUDO_USER", tc.sudoUsr)
			t.Setenv("SUDO_UID", tc.uid)
			t.Setenv("SUDO_GID", tc.gid)
			if _, _, ok := InvokingUserIDs(); ok {
				t.Fatalf("InvokingUserIDs ok = true, want false for %q", tc.name)
			}
		})
	}
}

func TestInvokingUserFallsBackToUserWhenNotRoot(t *testing.T) {
	origRoot := RunningAsRootFn
	defer func() { RunningAsRootFn = origRoot }()
	RunningAsRootFn = func() bool { return false }

	// Even if SUDO_USER is set, when we're not root we report the
	// current user — there's nothing to drop privileges from.
	t.Setenv("SUDO_USER", "alice")
	t.Setenv("USER", "bob")

	if got := InvokingUser(); got != "bob" {
		t.Fatalf("InvokingUser() = %q, want bob (current user takes precedence when not root)", got)
	}
}

func TestInvokingUserHomeDirReadsPasswd(t *testing.T) {
	origRoot := RunningAsRootFn
	origLookup := lookupHomeFromPasswdFn
	defer func() {
		RunningAsRootFn = origRoot
		lookupHomeFromPasswdFn = origLookup
	}()
	RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "alice")
	t.Setenv("HOME", "/root")
	lookupHomeFromPasswdFn = func(user string) string {
		if user == "alice" {
			return "/home/alice"
		}
		return ""
	}

	got, err := InvokingUserHomeDir()
	if err != nil {
		t.Fatalf("InvokingUserHomeDir: %v", err)
	}
	if got != "/home/alice" {
		t.Fatalf("got %q, want /home/alice (must NOT return $HOME=/root under sudo)", got)
	}
}

func TestInvokingUserHomeDirFallsBackToHOMEWhenPasswdMisses(t *testing.T) {
	// AD-bound macOS / NIS-only setups have no /etc/passwd entry. The
	// fallback to $HOME is correct for the current process even if not
	// for cross-user lookups.
	origRoot := RunningAsRootFn
	origLookup := lookupHomeFromPasswdFn
	defer func() {
		RunningAsRootFn = origRoot
		lookupHomeFromPasswdFn = origLookup
	}()
	RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")
	t.Setenv("HOME", "/Users/alice")
	lookupHomeFromPasswdFn = func(user string) string { return "" }

	got, err := InvokingUserHomeDir()
	if err != nil {
		t.Fatalf("InvokingUserHomeDir: %v", err)
	}
	if got != "/Users/alice" {
		t.Fatalf("got %q, want /Users/alice fallback", got)
	}
}

func TestRunAsInvokingUserNoOpWhenNotRoot(t *testing.T) {
	// When not running as root there's nothing to drop privileges from.
	// The command should be invoked verbatim — no sudo wrap.
	origRoot := RunningAsRootFn
	origRun := RunCommandFn
	defer func() {
		RunningAsRootFn = origRoot
		RunCommandFn = origRun
	}()
	RunningAsRootFn = func() bool { return false }
	t.Setenv("SUDO_USER", "")
	t.Setenv("USER", "alice")

	var gotName string
	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := RunAsInvokingUser("go", []string{"install", "pkg"}, EnsureOptions{}); err != nil {
		t.Fatalf("RunAsInvokingUser: %v", err)
	}
	if gotName != "go" {
		t.Fatalf("name = %q, want bare go (no sudo wrap)", gotName)
	}
	if strings.Join(gotArgs, " ") != "install pkg" {
		t.Fatalf("args = %v", gotArgs)
	}
}

func TestRunAsInvokingUserWithInputNoOpWhenNotRoot(t *testing.T) {
	origRoot := RunningAsRootFn
	origRun := RunCommandInputFn
	defer func() {
		RunningAsRootFn = origRoot
		RunCommandInputFn = origRun
	}()
	RunningAsRootFn = func() bool { return false }
	t.Setenv("SUDO_USER", "")
	t.Setenv("USER", "alice")

	var gotName string
	var gotArgs []string
	var gotInput string
	RunCommandInputFn = func(name string, args []string, input string, _ EnsureOptions) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		gotInput = input
		return nil
	}

	if err := RunAsInvokingUserWithInput("vrooli", []string{"credentials", "store", "init"}, "operator secret", EnsureOptions{}); err != nil {
		t.Fatalf("RunAsInvokingUserWithInput: %v", err)
	}
	if gotName != "vrooli" || strings.Join(gotArgs, " ") != "credentials store init" || gotInput != "operator secret" {
		t.Fatalf("command = %q %v input=%q", gotName, gotArgs, gotInput)
	}
}

func TestRunAsInvokingUserDropsPrivilegesWhenRoot(t *testing.T) {
	// Running as root with SUDO_USER set: must wrap with `sudo -u <user>
	// -H -- <cmd> <args>` so the subprocess inherits the operator's HOME
	// and runs under their UID.
	origRoot := RunningAsRootFn
	origRun := RunCommandFn
	defer func() {
		RunningAsRootFn = origRoot
		RunCommandFn = origRun
	}()
	RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "alice")

	var gotName string
	var gotArgs []string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := RunAsInvokingUser("go", []string{"install", "pkg"}, EnsureOptions{}); err != nil {
		t.Fatalf("RunAsInvokingUser: %v", err)
	}
	if gotName != "sudo" {
		t.Fatalf("name = %q, want sudo", gotName)
	}
	want := "-u alice -H -- go install pkg"
	if got := strings.Join(gotArgs, " "); got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestRunAsInvokingUserNoOpWhenSudoUserIsRoot(t *testing.T) {
	// `sudo -u root vrooli setup` (or the unusual case where root sudo's
	// to root). No privileges to drop; just run directly.
	origRoot := RunningAsRootFn
	origRun := RunCommandFn
	defer func() {
		RunningAsRootFn = origRoot
		RunCommandFn = origRun
	}()
	RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "root")

	var gotName string
	RunCommandFn = func(name string, args []string, opts EnsureOptions) error {
		gotName = name
		return nil
	}

	if err := RunAsInvokingUser("go", []string{"version"}, EnsureOptions{}); err != nil {
		t.Fatalf("RunAsInvokingUser: %v", err)
	}
	if gotName != "go" {
		t.Fatalf("name = %q, want go (root SUDO_USER should not re-wrap with sudo)", gotName)
	}
}

func TestResolveCommandForInvokingUserPrefersPATH(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	LookPathFn = func(name string) (string, error) {
		if name == "protoc-gen-go" {
			return "/usr/bin/protoc-gen-go", nil
		}
		return "", os.ErrNotExist
	}

	cmd, ok := ResolveCommandForInvokingUser([]string{"protoc-gen-go"})
	if !ok || cmd != "protoc-gen-go" {
		t.Fatalf("got (%q, %v), want (protoc-gen-go, true)", cmd, ok)
	}
}

func TestResolveCommandForInvokingUserProbesUserDirsWhenPATHMisses(t *testing.T) {
	// Reproduces the failure mode: under sudo, root's PATH excludes
	// ~/.local/bin and ~/go/bin, so the bare LookPath misses even though
	// the binary is right there in the operator's home.
	restore := stubLookups(t)
	defer restore()

	tmp := t.TempDir()
	if err := os.MkdirAll(tmp+"/go/bin", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp+"/go/bin/protoc-gen-go", []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	LookPathFn = func(name string) (string, error) { return "", os.ErrNotExist }

	origRoot := RunningAsRootFn
	origLookup := lookupHomeFromPasswdFn
	defer func() {
		RunningAsRootFn = origRoot
		lookupHomeFromPasswdFn = origLookup
	}()
	RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "alice")
	lookupHomeFromPasswdFn = func(user string) string { return tmp }

	cmd, ok := ResolveCommandForInvokingUser([]string{"protoc-gen-go"})
	if !ok {
		t.Fatalf("expected user-dir probe to find binary at %s/go/bin/protoc-gen-go", tmp)
	}
	if cmd != "protoc-gen-go" {
		t.Fatalf("cmd = %q, want bare name", cmd)
	}
}

func TestResolveCommandForInvokingUserMissesWhenAbsentEverywhere(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	tmp := t.TempDir()
	LookPathFn = func(name string) (string, error) { return "", os.ErrNotExist }

	origRoot := RunningAsRootFn
	origLookup := lookupHomeFromPasswdFn
	defer func() {
		RunningAsRootFn = origRoot
		lookupHomeFromPasswdFn = origLookup
	}()
	RunningAsRootFn = func() bool { return false }
	t.Setenv("USER", "alice")
	lookupHomeFromPasswdFn = func(user string) string { return tmp }

	if _, ok := ResolveCommandForInvokingUser([]string{"protoc-gen-go"}); ok {
		t.Fatalf("ResolveCommandForInvokingUser should miss when binary is absent")
	}
}

func TestAugmentUserToolPathAddsManagedDirectoriesOnce(t *testing.T) {
	home := t.TempDir()
	for _, relative := range []string{"go/bin", ".local/bin", "bin"} {
		if err := os.MkdirAll(filepath.Join(home, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	existing := []string{"/opt/homebrew/bin", "/usr/local/go/bin", "/usr/local/bin", filepath.Join(home, "go", "bin")}
	got := strings.Split(AugmentUserToolPath(home, strings.Join(existing, string(os.PathListSeparator)), ""), string(os.PathListSeparator))
	wantPrefix := []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin")}
	if len(got) < len(wantPrefix) || !strings.EqualFold(strings.Join(got[:len(wantPrefix)], "\x00"), strings.Join(wantPrefix, "\x00")) {
		t.Fatalf("PATH prefix = %v, want %v", got, wantPrefix)
	}
	count := 0
	for _, entry := range got {
		if filepath.Clean(entry) == filepath.Join(home, "go", "bin") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("go/bin appears %d times in PATH %v, want once", count, got)
	}
}
