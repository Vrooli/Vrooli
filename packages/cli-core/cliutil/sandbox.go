package cliutil

import (
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// Sandbox-aware path resolution for scenario CLIs
// ---------------------------------------------------------------------------
//
// When an agent runs inside a sandbox (created by agent-manager for isolated
// code editing), the agent process inherits three environment variables:
//
//   VROOLI_SANDBOX_ID     — UUID identifying the sandbox (for logging/debugging)
//   VROOLI_SANDBOX_MERGED — absolute path to the overlay's merged/ directory,
//                           which contains the agent's file changes layered
//                           over the real repository via overlayfs
//   VROOLI_SANDBOX_SCOPE  — relative path from project root defining what the
//                           overlay covers (e.g., "scenarios/my-scenario")
//
// CLI tools that the agent runs directly (test-genie, tidiness-manager, etc.)
// inherit these variables automatically. By checking them, a CLI can resolve
// scenario file paths to the sandbox's overlay instead of the real repository
// — allowing the agent to verify its own changes against tools like code
// quality scanners and test runners.
//
// Without this awareness, CLIs would resolve paths via VROOLI_ROOT (the real
// repo), and the agent's changes in the sandbox overlay would be invisible
// to those tools.
//
// This package provides Go equivalents of the bash sandbox path resolution
// functions in scripts/lib/scenario/runner.sh (sandbox::scenario_in_scope and
// sandbox::resolve_merged_path). The logic must stay in sync with those bash
// implementations.
//
// For more context on the sandbox lifecycle system, see:
//   - scenarios/agent-manager/api/internal/orchestration/run_executor.go
//     (SandboxEnvVars method — where the env vars are injected)
//   - scenarios/workspace-sandbox/docs/ARCHITECTURE.md
//     (scope vs acceptance design, overlay structure)
//   - scripts/lib/scenario/runner.sh
//     (bash implementation this code mirrors)
// ---------------------------------------------------------------------------

// SandboxEnv holds the sandbox environment variables extracted from os.Environ.
// A zero-value SandboxEnv (all empty strings) means no sandbox is active.
type SandboxEnv struct {
	// ID is the sandbox UUID (from VROOLI_SANDBOX_ID). Used for logging only.
	ID string

	// Merged is the absolute path to the overlay's merged/ directory (from
	// VROOLI_SANDBOX_MERGED). This is the root of the filesystem view that
	// includes the agent's changes layered over the real repo.
	Merged string

	// Scope is the relative path from project root that the overlay covers
	// (from VROOLI_SANDBOX_SCOPE). For example, "scenarios/my-scenario" means
	// the overlay only covers that single scenario directory. An empty scope
	// or "." means the overlay covers the entire project root.
	Scope string
}

// DetectSandbox reads sandbox environment variables from the current process.
// Returns a zero-value SandboxEnv if not running inside a sandbox.
func DetectSandbox() SandboxEnv {
	return SandboxEnv{
		ID:     os.Getenv("VROOLI_SANDBOX_ID"),
		Merged: os.Getenv("VROOLI_SANDBOX_MERGED"),
		Scope:  os.Getenv("VROOLI_SANDBOX_SCOPE"),
	}
}

// IsSandboxActive returns true if both Merged and Scope are set, indicating
// this process is running inside a sandboxed agent. The ID field is optional
// (used for logging only) and is not required for sandbox detection.
func (s SandboxEnv) IsSandboxActive() bool {
	return s.Merged != "" && s.Scope != ""
}

// ScenarioInScope checks whether a scenario slug falls within the sandbox's
// scoped path. This determines whether path resolution should redirect to the
// sandbox's merged directory or use the real repo.
//
// An agent sandboxing scenario A should NOT affect scenario B's path
// resolution — only in-scope scenarios are redirected, everything else
// uses the real repo.
//
// Scope matching rules (mirrors scripts/lib/scenario/runner.sh):
//
//	""  or "." or "/"        → whole repo is scoped, ALL scenarios match
//	"scenarios"              → all scenarios are scoped
//	"scenarios/foo"          → only scenario "foo" matches
//	"scenarios/foo/api"      → still matches "foo" (scope is deeper but within it)
//	"scenarios/other"        → does NOT match "foo"
//	"packages/shared"        → no scenarios match (scope outside scenarios/)
func ScenarioInScope(scenarioName, scope string) bool {
	// Empty scope or root scope means the overlay covers everything.
	if scope == "" || scope == "/" || scope == "." {
		return true
	}

	// Normalize: remove trailing slash.
	scope = strings.TrimRight(scope, "/")

	// If scope IS "scenarios", all scenarios are in scope.
	if scope == "scenarios" {
		return true
	}

	// If scope starts with "scenarios/", check if this specific scenario matches.
	// Extract the first path component after "scenarios/" and compare.
	// e.g. scope="scenarios/foo/api" → scopedName="foo" → matches scenario "foo"
	if strings.HasPrefix(scope, "scenarios/") {
		remainder := strings.TrimPrefix(scope, "scenarios/")
		// Take only the first path component (before any slash).
		scopedName := remainder
		if idx := strings.Index(remainder, "/"); idx >= 0 {
			scopedName = remainder[:idx]
		}
		return scenarioName == scopedName
	}

	// Scope is outside scenarios/ (e.g., "packages/shared") — no scenarios match.
	return false
}

// ResolveMergedPath computes the absolute path to a scenario within the
// sandbox's merged directory.
//
// The overlay's merged/ dir maps to the scoped directory, NOT the project root.
// This means the path within merged/ depends on the scope:
//
//	scope="scenarios/agent-inbox"  → merged/ IS the scenario → return merged
//	scope="scenarios"              → merged/ has scenario dirs → return merged/agent-inbox
//	scope="" or "." or "/"         → merged/ is project root  → return merged/scenarios/agent-inbox
//
// The algorithm: compute the scenario's full relative path ("scenarios/{name}"),
// then strip the scope prefix to get the remaining path within the merged dir.
func ResolveMergedPath(scenarioName, scope, merged string) string {
	// Normalize: remove trailing slash from scope.
	scope = strings.TrimRight(scope, "/")

	// The full canonical path to the scenario relative to project root.
	scenarioRel := filepath.Join("scenarios", scenarioName)

	// If scope is empty/root, merged/ is the project root — use full relative path.
	if scope == "" || scope == "/" || scope == "." {
		return filepath.Join(merged, scenarioRel)
	}

	// Strip the scope prefix from the scenario's relative path.
	if scenarioRel == scope {
		// Scope exactly matches the scenario path — merged/ IS the scenario dir.
		return merged
	}
	if strings.HasPrefix(scenarioRel, scope+"/") {
		// Scope is a parent dir — strip it to get the relative path within merged/.
		relative := strings.TrimPrefix(scenarioRel, scope+"/")
		return filepath.Join(merged, relative)
	}

	// Fallback: shouldn't happen if ScenarioInScope passed, but be safe.
	return filepath.Join(merged, scenarioRel)
}

// defaultRepoRoot returns the standard Vrooli repository root from env or
// filesystem convention.
//
// Deferred migration note: the HOME-based fallback is legacy compatibility and
// should not be treated as future-state repo-contract authority.
func defaultRepoRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "Vrooli")
	}
	return ""
}

// ResolveScenarioPath returns the absolute filesystem path to a scenario's
// directory, accounting for sandbox isolation when active.
//
// When called from within a sandboxed agent process:
//   - If the scenario is within the sandbox's scope, the returned path points
//     to the scenario within the overlay — reflecting any agent changes.
//   - If the scenario is outside the scope, the real repo path is returned.
//
// When called outside a sandbox, this falls back to VROOLI_ROOT resolution
// (VROOLI_ROOT env var → $HOME/Vrooli fallback).
//
// This is the primary function most CLIs should call. It combines
// DetectSandbox(), ScenarioInScope(), and ResolveMergedPath() into a single
// call with sensible defaults.
func ResolveScenarioPath(scenarioName string) string {
	sbx := DetectSandbox()
	if sbx.IsSandboxActive() && ScenarioInScope(scenarioName, sbx.Scope) {
		return ResolveMergedPath(scenarioName, sbx.Scope, sbx.Merged)
	}
	return filepath.Join(defaultRepoRoot(), "scenarios", scenarioName)
}

// ResolveRepoRoot returns the effective repository root directory.
//
// In a sandbox context where the scope covers the full project (scope is
// empty, ".", or "/"), returns the merged path as the effective root.
// Otherwise returns VROOLI_ROOT or $HOME/Vrooli fallback.
//
// Note: For scenario-scoped sandboxes (e.g., scope="scenarios/my-scenario"),
// the merged directory does NOT represent the full project root — it only
// contains that one scenario. In such cases, this function returns the real
// repo root. Use ResolveScenarioPath() instead when you need a specific
// scenario's path.
func ResolveRepoRoot() string {
	sbx := DetectSandbox()
	if sbx.IsSandboxActive() {
		scope := strings.TrimRight(sbx.Scope, "/")
		if scope == "" || scope == "/" || scope == "." {
			return sbx.Merged
		}
	}
	return defaultRepoRoot()
}
