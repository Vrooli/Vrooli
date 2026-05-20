package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--quiet")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return dir
}

func gitConfig(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"config"}, args...)...)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config %v: %v: %s", args, err, out)
	}
}

func TestInstallHookCleanRepo(t *testing.T) {
	repo := initGitRepo(t)
	res, err := InstallHook(context.Background(), repo, "vrooli hygiene --fail-on error")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !res.Installed {
		t.Fatalf("expected installed, got %#v", res)
	}
	data, err := os.ReadFile(res.Path)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	if !strings.Contains(string(data), gctHookSentinel) {
		t.Fatalf("hook missing sentinel: %s", data)
	}
	if !strings.Contains(string(data), "vrooli hygiene --fail-on error") {
		t.Fatalf("hook missing command: %s", data)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(res.Path)
		if err != nil {
			t.Fatalf("stat hook: %v", err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("hook not executable: mode=%v", info.Mode())
		}
	}
}

func TestInstallHookUpgradeOverExistingGCTHook(t *testing.T) {
	repo := initGitRepo(t)
	first, err := InstallHook(context.Background(), repo, "echo first")
	if err != nil || !first.Installed {
		t.Fatalf("first install: %v %#v", err, first)
	}
	second, err := InstallHook(context.Background(), repo, "echo second")
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !second.Installed {
		t.Fatalf("expected re-install over GCT hook: %#v", second)
	}
	data, _ := os.ReadFile(second.Path)
	if !strings.Contains(string(data), "echo second") {
		t.Fatalf("hook not updated: %s", data)
	}
}

func TestInstallHookRefusesToClobberUserHook(t *testing.T) {
	repo := initGitRepo(t)
	hooks := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	existing := "#!/usr/bin/env bash\necho 'user hook'\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(existing), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := InstallHook(context.Background(), repo, "vrooli hygiene")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if res.Installed {
		t.Fatalf("expected fallback, got installed: %#v", res)
	}
	if res.ExistingHookKind != HookKindUser {
		t.Fatalf("expected ExistingHookKind=user, got %q", res.ExistingHookKind)
	}
	if !strings.Contains(res.ExistingHookPreview, "user hook") {
		t.Fatalf("preview missing existing content: %q", res.ExistingHookPreview)
	}
	data, _ := os.ReadFile(filepath.Join(hooks, "pre-commit"))
	if string(data) != existing {
		t.Fatalf("existing hook was modified: %s", data)
	}
}

func TestInstallHookDetectsHusky(t *testing.T) {
	repo := initGitRepo(t)
	hooks := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	husky := "#!/usr/bin/env sh\n. \"$(dirname \"$0\")/_/husky.sh\"\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(husky), 0o755); err != nil {
		t.Fatal(err)
	}
	res, _ := InstallHook(context.Background(), repo, "vrooli hygiene")
	if res.ExistingHookKind != HookKindFramework {
		t.Fatalf("expected framework, got %q", res.ExistingHookKind)
	}
}

func TestInstallHookFallsBackOnExternalHooksPath(t *testing.T) {
	repo := initGitRepo(t)
	external := t.TempDir()
	gitConfig(t, repo, "core.hooksPath", external)
	res, err := InstallHook(context.Background(), repo, "vrooli hygiene")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if res.Installed {
		t.Fatalf("expected fallback on external hooksPath, got installed: %#v", res)
	}
	if !strings.Contains(res.Reason, "core.hooksPath") {
		t.Fatalf("expected reason to mention core.hooksPath: %q", res.Reason)
	}
}

func TestUninstallHookOnlyRemovesGCTManaged(t *testing.T) {
	repo := initGitRepo(t)
	hooks := filepath.Join(repo, ".git", "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\necho user\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := UninstallHook(context.Background(), repo)
	if err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}
	if !strings.Contains(res.Reason, "refuses") {
		t.Fatalf("expected refusal reason, got %q", res.Reason)
	}
	if _, err := os.Stat(filepath.Join(hooks, "pre-commit")); err != nil {
		t.Fatalf("user hook should still exist: %v", err)
	}
	gctScript := buildHookScript("echo gct", repo)
	if err := os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte(gctScript), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err = UninstallHook(context.Background(), repo)
	if err != nil {
		t.Fatalf("UninstallHook gct: %v", err)
	}
	if !strings.Contains(res.Reason, "removed") {
		t.Fatalf("expected removed, got %q", res.Reason)
	}
	if _, err := os.Stat(filepath.Join(hooks, "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("hook should be removed: %v", err)
	}
}

func TestReadInstalledHookGitfileResolution(t *testing.T) {
	repo := t.TempDir()
	realGit := filepath.Join(repo, ".real-git")
	if err := os.MkdirAll(filepath.Join(realGit, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: "+realGit+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(realGit, "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte(buildHookScript("echo gct", repo)), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := ReadInstalledHook(context.Background(), repo)
	if err != nil {
		t.Fatalf("ReadInstalledHook: %v", err)
	}
	if info.Kind != HookKindGCT {
		t.Fatalf("expected gct kind via gitfile, got %q", info.Kind)
	}
	if info.Path != hookPath {
		t.Fatalf("expected path %q, got %q", hookPath, info.Path)
	}
}

func TestReadInstalledHookReportsNoneOnFreshRepo(t *testing.T) {
	repo := initGitRepo(t)
	info, err := ReadInstalledHook(context.Background(), repo)
	if err != nil {
		t.Fatalf("ReadInstalledHook: %v", err)
	}
	if info.Kind != HookKindNone {
		t.Fatalf("expected none, got %q", info.Kind)
	}
}
