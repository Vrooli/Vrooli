package main

import (
	"fmt"
	"os"

	"github.com/vrooli/cli-core/cliutil"
)

// sandbox-resolve — CLI helper for sandbox-aware path resolution.
//
// Bash CLIs (tidiness-manager, scenario-auditor) shell out to this binary to
// resolve scenario paths when running inside a sandboxed agent. The underlying
// Go implementation lives in packages/cli-core/cliutil/sandbox.go.
//
// Why this exists:
// When an agent runs inside a sandbox (created by agent-manager), it inherits
// environment variables (VROOLI_SANDBOX_MERGED, VROOLI_SANDBOX_SCOPE) that
// describe the overlay filesystem. Go CLIs import cliutil directly, but bash
// CLIs need a way to call the same logic — this binary bridges that gap.
//
// Usage:
//
//	sandbox-resolve scenario <name>   Print absolute path to scenario dir
//	sandbox-resolve repo-root         Print effective repository root
//	sandbox-resolve in-scope <name>   Exit 0 if scenario is in sandbox scope, 1 if not
//	sandbox-resolve active            Exit 0 if sandbox is active, 1 if not
//
// Environment variables read:
//
//	VROOLI_SANDBOX_MERGED  Overlay merged directory path
//	VROOLI_SANDBOX_SCOPE   Relative scope path (e.g., "scenarios/my-scenario")
//	VROOLI_ROOT            Repository root (fallback when no sandbox)
//
// Examples (from bash):
//
//	scenario_dir=$(sandbox-resolve scenario my-scenario)
//	repo_root=$(sandbox-resolve repo-root)
//	if sandbox-resolve in-scope my-scenario; then echo "in scope"; fi
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scenario":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "sandbox-resolve scenario: missing scenario name")
			os.Exit(1)
		}
		fmt.Println(cliutil.ResolveScenarioPath(os.Args[2]))

	case "repo-root":
		fmt.Println(cliutil.ResolveRepoRoot())

	case "in-scope":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "sandbox-resolve in-scope: missing scenario name")
			os.Exit(1)
		}
		sbx := cliutil.DetectSandbox()
		if sbx.IsSandboxActive() && cliutil.ScenarioInScope(os.Args[2], sbx.Scope) {
			os.Exit(0)
		}
		os.Exit(1)

	case "active":
		sbx := cliutil.DetectSandbox()
		if sbx.IsSandboxActive() {
			os.Exit(0)
		}
		os.Exit(1)

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "sandbox-resolve: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: sandbox-resolve <command> [args]

Commands:
  scenario <name>   Print absolute path to scenario directory
  repo-root         Print effective repository root
  in-scope <name>   Exit 0 if scenario is in sandbox scope, 1 if not
  active            Exit 0 if sandbox is active, 1 if not
  help              Show this help

Environment:
  VROOLI_SANDBOX_MERGED   Overlay merged directory path
  VROOLI_SANDBOX_SCOPE    Relative scope path
  VROOLI_ROOT             Repository root (fallback)`)
}
