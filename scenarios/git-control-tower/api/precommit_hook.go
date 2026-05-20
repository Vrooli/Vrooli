package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	gctHookSentinel = "# managed by vrooli git-control-tower (precommit)"

	HookKindNone      = "none"
	HookKindGCT       = "gct"
	HookKindUser      = "user"
	HookKindFramework = "framework"
)

type HookInfo struct {
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	HooksPath string `json:"hooks_path"`
	Preview   string `json:"preview,omitempty"`
}

type HookInstallResult struct {
	Installed           bool      `json:"installed"`
	Reason              string    `json:"reason,omitempty"`
	Path                string    `json:"path,omitempty"`
	HooksPath           string    `json:"hooks_path,omitempty"`
	ExistingHookKind    string    `json:"existing_hook_kind,omitempty"`
	ExistingHookPreview string    `json:"existing_hook_preview,omitempty"`
	InstalledAt         time.Time `json:"installed_at,omitempty"`
}

// ReadInstalledHook inspects the repo's pre-commit hook (if any) and
// classifies it. It resolves gitfiles and `core.hooksPath`.
func ReadInstalledHook(ctx context.Context, repoDir string) (HookInfo, error) {
	hooksDir, _, err := resolveHooksDir(ctx, repoDir)
	if err != nil {
		return HookInfo{}, err
	}
	path := filepath.Join(hooksDir, "pre-commit")
	info := HookInfo{Kind: HookKindNone, Path: path, HooksPath: hooksDir}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return info, fmt.Errorf("read pre-commit hook: %w", err)
	}
	info.Preview = capHookPreview(string(data))
	if hasGCTSentinel(data) {
		info.Kind = HookKindGCT
		return info, nil
	}
	if kind := detectFrameworkHook(data); kind != "" {
		info.Kind = HookKindFramework
		return info, nil
	}
	info.Kind = HookKindUser
	return info, nil
}

// InstallHook writes a portable pre-commit hook that delegates to the
// given command. It refuses to clobber non-GCT hooks and falls back
// gracefully when an external hooks manager is configured.
func InstallHook(ctx context.Context, repoDir, command string) (HookInstallResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return HookInstallResult{Reason: "no command configured"}, fmt.Errorf("command is required")
	}
	hooksDir, external, err := resolveHooksDir(ctx, repoDir)
	if err != nil {
		return HookInstallResult{Reason: err.Error()}, err
	}
	result := HookInstallResult{Path: filepath.Join(hooksDir, "pre-commit"), HooksPath: hooksDir}
	if external != "" {
		result.Reason = fmt.Sprintf("external hook manager (core.hooksPath=%s)", external)
		result.ExistingHookKind = HookKindFramework
		return result, nil
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		result.Reason = fmt.Sprintf("create hooks dir: %v", err)
		return result, err
	}
	if existing, err := os.ReadFile(result.Path); err == nil {
		if !hasGCTSentinel(existing) {
			result.Reason = "existing pre-commit hook is not managed by git-control-tower"
			result.ExistingHookKind = HookKindUser
			if detectFrameworkHook(existing) != "" {
				result.ExistingHookKind = HookKindFramework
			}
			result.ExistingHookPreview = capHookPreview(string(existing))
			return result, nil
		}
	} else if !os.IsNotExist(err) {
		result.Reason = fmt.Sprintf("read existing hook: %v", err)
		return result, err
	}
	script := buildHookScript(command, repoDir)
	if err := writeHookScript(result.Path, script); err != nil {
		result.Reason = fmt.Sprintf("write hook: %v", err)
		return result, err
	}
	result.Installed = true
	result.InstalledAt = time.Now().UTC()
	result.Reason = "installed"
	return result, nil
}

// UninstallHook removes the pre-commit hook ONLY if it carries the
// GCT sentinel. Non-GCT hooks are left alone.
func UninstallHook(ctx context.Context, repoDir string) (HookInstallResult, error) {
	hooksDir, external, err := resolveHooksDir(ctx, repoDir)
	if err != nil {
		return HookInstallResult{Reason: err.Error()}, err
	}
	result := HookInstallResult{Path: filepath.Join(hooksDir, "pre-commit"), HooksPath: hooksDir}
	if external != "" {
		result.Reason = fmt.Sprintf("external hook manager (core.hooksPath=%s)", external)
		return result, nil
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		if os.IsNotExist(err) {
			result.Reason = "no hook installed"
			return result, nil
		}
		result.Reason = fmt.Sprintf("read hook: %v", err)
		return result, err
	}
	if !hasGCTSentinel(data) {
		result.Reason = "refuses to remove non-managed hook"
		result.ExistingHookKind = HookKindUser
		if detectFrameworkHook(data) != "" {
			result.ExistingHookKind = HookKindFramework
		}
		result.ExistingHookPreview = capHookPreview(string(data))
		return result, nil
	}
	if err := os.Remove(result.Path); err != nil {
		result.Reason = fmt.Sprintf("remove hook: %v", err)
		return result, err
	}
	result.Reason = "removed"
	return result, nil
}

func resolveHooksDir(ctx context.Context, repoDir string) (hooksDir, externalReason string, err error) {
	gitDir, err := resolveGitDir(repoDir)
	if err != nil {
		return "", "", err
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", "config", "--get", "core.hooksPath")
	cmd.Dir = repoDir
	out, runErr := cmd.Output()
	if runErr == nil {
		path := strings.TrimSpace(string(out))
		if path != "" {
			abs := path
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(repoDir, abs)
			}
			defaultHooks := filepath.Join(gitDir, "hooks")
			if filepath.Clean(abs) != filepath.Clean(defaultHooks) {
				return abs, path, nil
			}
			return abs, "", nil
		}
	}
	return filepath.Join(gitDir, "hooks"), "", nil
}

func resolveGitDir(repoDir string) (string, error) {
	gitPath := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("stat .git: %w", err)
	}
	if info.IsDir() {
		return gitPath, nil
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("read gitfile: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "gitdir:") {
			ref := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
			if !filepath.IsAbs(ref) {
				ref = filepath.Join(repoDir, ref)
			}
			return filepath.Clean(ref), nil
		}
	}
	return "", fmt.Errorf("gitfile %s has no gitdir entry", gitPath)
}

func buildHookScript(command, repoDir string) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env sh\n")
	b.WriteString(gctHookSentinel)
	b.WriteString("\n")
	b.WriteString("# Edit via the git-control-tower UI; manual edits will be overwritten on save.\n")
	b.WriteString("set -e\n")
	fmt.Fprintf(&b, "cd %s\n", shellQuote(repoDir))
	b.WriteString("exec ")
	b.WriteString(command)
	b.WriteString("\n")
	return b.String()
}

func writeHookScript(path, content string) error {
	tmp := path + ".gct-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o755); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmp, 0o755); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return os.Rename(tmp, path)
}

func hasGCTSentinel(data []byte) bool {
	return bytes.Contains(data, []byte(gctHookSentinel))
}

func detectFrameworkHook(data []byte) string {
	s := string(data)
	switch {
	case strings.Contains(s, "husky"):
		return "husky"
	case strings.Contains(s, "lefthook"):
		return "lefthook"
	case strings.Contains(s, "pre-commit.com") || strings.Contains(s, "pre-commit framework"):
		return "pre-commit"
	}
	return ""
}

func capHookPreview(value string) string {
	const limit = 2048
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n[preview truncated]"
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\n'\"\\$`(){}[]|&;<>*?!#~") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
