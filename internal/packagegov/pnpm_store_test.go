package packagegov

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeStore makes `pnpm store path` answer from the environment: the store
// pnpm would pick is store/v3 unless npm_config_store_dir overrides it, exactly
// as a user pnpm config plus an override behaves.
func fakeStore(t *testing.T, configured string) {
	t.Helper()
	previous := pnpmStoreRunner
	pnpmStoreRunner = func(_ context.Context, _, _ string, env []string) (string, error) {
		base := configured
		for _, item := range env {
			if value, ok := strings.CutPrefix(item, "npm_config_store_dir="); ok {
				base = value
			}
		}
		return filepath.Join(base, "v10"), nil
	}
	t.Cleanup(func() { pnpmStoreRunner = previous })
}

func linkedTo(t *testing.T, dir, store string) {
	t.Helper()
	linkedWith(t, dir, "storeDir: "+store)
}

// linkedWith writes .modules.yaml with the given storeDir line, in either of
// the two spellings pnpm produces.
func linkedWith(t *testing.T, dir, storeLine string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "hoistPattern:\n  - '*'\n" + storeLine + "\nvirtualStoreDir: .pnpm\n"
	if err := os.WriteFile(filepath.Join(dir, "node_modules", ".modules.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func envValue(env []string, key string) string {
	value := ""
	for _, item := range env {
		if v, ok := strings.CutPrefix(item, key+"="); ok {
			value = v
		}
	}
	return value
}

// The evidence: packages/file-viewer's node_modules was linked against
// store/v10 while the user's pnpm config selects store/v3/v10, so a frozen
// install wanted to purge node_modules and aborted without a terminal.
func TestGuardPnpmStoreFollowsTheStoreNodeModulesIsLinkedAgainst(t *testing.T) {
	dir := t.TempDir()
	linkedTo(t, dir, "/home/u/.local/share/pnpm/store/v10")
	fakeStore(t, "/home/u/.local/share/pnpm/store/v3")

	env, note, err := GuardPnpmStore(context.Background(), []string{"pnpm", "install", "--frozen-lockfile"}, dir, []string{"PATH=/bin", "npm_config_store_dir=/stale"})
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "npm_config_store_dir"); got != "/home/u/.local/share/pnpm/store" {
		t.Fatalf("store base = %q, want the base of the recorded store", got)
	}
	if strings.Count(strings.Join(env, "\n"), "npm_config_store_dir=") != 1 {
		t.Fatalf("the override must replace, not duplicate: %v", env)
	}
	if !strings.Contains(note, "store/v10") || !strings.Contains(note, "/stale/v10") || !strings.Contains(note, "does not recreate") {
		t.Fatalf("the note must name both stores and why: %q", note)
	}
}

func TestGuardPnpmStoreLeavesEverythingElseAlone(t *testing.T) {
	env := []string{"PATH=/bin"}
	fakeStore(t, "/store/v3")
	matching := t.TempDir()
	linkedTo(t, matching, "/store/v3/v10")
	for name, tc := range map[string]struct {
		argv []string
		dir  string
	}{
		"stores agree":               {[]string{"pnpm", "install"}, matching},
		"no node_modules":            {[]string{"pnpm", "install"}, t.TempDir()},
		"not a node_modules command": {[]string{"pnpm", "run", "build"}, matching},
		"not pnpm":                   {[]string{"npm", "install"}, matching},
		"flag before the subcommand": {[]string{"pnpm", "--silent", "run", "build"}, matching},
	} {
		got, note, err := GuardPnpmStore(context.Background(), tc.argv, tc.dir, env)
		if err != nil || note != "" || strings.Join(got, ",") != "PATH=/bin" {
			t.Errorf("%s: env=%v note=%q err=%v", name, got, note, err)
		}
	}
}

func TestGuardPnpmStoreRefusesWhenTheRecordedStoreCannotBeSelected(t *testing.T) {
	dir := t.TempDir()
	linkedTo(t, dir, "/somewhere/odd-layout")
	fakeStore(t, "/store/v3")
	_, _, err := GuardPnpmStore(context.Background(), []string{"pnpm", "install"}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "/somewhere/odd-layout") || !strings.Contains(err.Error(), "must not do") {
		t.Fatalf("err = %v", err)
	}

	// The override has no effect (for example pnpm ignores it): refuse rather than run.
	linkedTo(t, dir, "/elsewhere/v10")
	previous := pnpmStoreRunner
	pnpmStoreRunner = func(context.Context, string, string, []string) (string, error) { return "/store/v3/v10", nil }
	t.Cleanup(func() { pnpmStoreRunner = previous })
	if _, _, err := GuardPnpmStore(context.Background(), []string{"pnpm", "install"}, dir, nil); err == nil || !strings.Contains(err.Error(), "cannot be reused") {
		t.Fatalf("an override that does not take effect must be refused: %v", err)
	}
}

func TestExplainPnpmPurgeAbortSaysWhatHappenedAndNothingChanged(t *testing.T) {
	cause := errors.New("exit status 1")
	got := ExplainPnpmPurgeAbort(cause, " ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY  Aborted removal of modules directory due to no TTY", "/repo/packages/x")
	if !errors.Is(got, cause) || !strings.Contains(got.Error(), "/repo/packages/x/node_modules") || !strings.Contains(got.Error(), "nothing was changed") {
		t.Fatalf("got %v", got)
	}
	if other := ExplainPnpmPurgeAbort(cause, "ERR_PNPM_FETCH_404", "/x"); other != cause {
		t.Fatalf("an unrelated failure must pass through unchanged: %v", other)
	}
}

func TestRunCommandsExplainsAPurgeAbortFromPnpmAndFollowsTheRecordedStore(t *testing.T) {
	dir := t.TempDir()
	linkedTo(t, dir, "/store/v10")
	fakeStore(t, "/store/v3")
	bin := t.TempDir()
	script := "#!/bin/sh\necho \"store=$npm_config_store_dir\"\necho ' ERR_PNPM_ABORTED_REMOVE_MODULES_DIR_NO_TTY  Aborted removal' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "pnpm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	err := RunCommandsWithOptions(dir, []CommandSpec{{Name: "install", Run: []string{filepath.Join(bin, "pnpm"), "install", "--frozen-lockfile"}}}, &out, &errOut, CommandOptions{Env: []string{"PATH=" + bin + ":/usr/bin:/bin"}})
	if err == nil || !strings.Contains(err.Error(), "will not do that without a terminal") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(out.String(), "store=/store") || !strings.Contains(errOut.String(), "reusing it") {
		t.Fatalf("the command must run against the recorded store and say so: out=%q err=%q", out.String(), errOut.String())
	}
}

func TestRecordedPnpmStoreReadsBothSpellingsOfModulesYaml(t *testing.T) {
	for _, line := range []string{`storeDir: /s/v10`, `  "storeDir": "/s/v10",`, `storeDir: '/s/v10'`} {
		dir := t.TempDir()
		linkedWith(t, dir, line)
		if got := recordedPnpmStore(dir); got != "/s/v10" {
			t.Errorf("%q parsed as %q", line, got)
		}
	}
}
