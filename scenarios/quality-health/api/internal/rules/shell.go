package rules

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"quality-health/internal/hostbin"
)

// Seams so tests drive resolution and execution deterministically without a
// host bash/shellcheck.
var (
	shellResolve = hostbin.Resolve
	shellRun     = defaultShellRun
)

// shellLintTimeout bounds each bash -n / shellcheck invocation. Syntax checks
// are near-instant; this only guards against a pathological file.
const shellLintTimeout = 30 * time.Second

// defaultShellRun runs a tool against one file and reports its combined output
// and whether it exited zero.
func defaultShellRun(name string, args ...string) (output string, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), shellLintTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// evalShellSyntax syntax-checks the scenario's shell scripts. It is the
// successor to the old `bash -n` CLI lint dropped in the Test Genie unit-phase
// cutover; quality-health owns shell *syntax* lint while unit-health owns bats
// *testing*. It targets `.sh` files only (`.bats` files use bats-specific
// syntax that `bash -n` cannot parse and are validated by running them under
// unit-health). On a platform without bash it emits an explicit degraded
// finding rather than a false pass.
func evalShellSyntax(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleShellSyntaxLint)
	root := ctx.Inventory.RootPath
	if root == "" {
		return nil
	}

	scripts := discoverShellScripts(root)
	if len(scripts) == 0 {
		return nil
	}

	bash, hasBash := shellResolve([]string{"bash"})
	if !hasBash {
		return []Finding{shellDegraded(ctx, rule, len(scripts))}
	}
	shellcheck, hasShellcheck := shellResolve([]string{"shellcheck"})

	var syntaxErrs []string
	var shellcheckErrs []string
	for _, f := range scripts {
		if out, ok := shellRun(bash, "-n", f); !ok {
			syntaxErrs = append(syntaxErrs, fmt.Sprintf("%s: %s", relOrBase(root, f), firstLine(out)))
		}
		if hasShellcheck {
			if out, ok := shellRun(shellcheck, "--severity=error", "--format=gcc", f); !ok {
				shellcheckErrs = append(shellcheckErrs, fmt.Sprintf("%s: %s", relOrBase(root, f), firstLine(out)))
			}
		}
	}

	if len(syntaxErrs) == 0 && len(shellcheckErrs) == 0 {
		return nil
	}

	var parts []string
	if len(syntaxErrs) > 0 {
		parts = append(parts, fmt.Sprintf("bash -n failures (%d): %s", len(syntaxErrs), strings.Join(cap10(syntaxErrs), "; ")))
	}
	if len(shellcheckErrs) > 0 {
		parts = append(parts, fmt.Sprintf("shellcheck errors (%d): %s", len(shellcheckErrs), strings.Join(cap10(shellcheckErrs), "; ")))
	}
	observed := fmt.Sprintf("%d shell script(s) scanned; %d syntax error(s)", len(scripts), len(syntaxErrs)+len(shellcheckErrs))
	tool := "bash -n"
	if hasShellcheck {
		tool = "bash -n + shellcheck"
	}
	return []Finding{ruleFinding(ctx, rule, root,
		"Shell scripts failed syntax linting ("+tool+")",
		strings.Join(parts, " | "),
		"all `.sh` scripts pass `bash -n` (and shellcheck when available)",
		observed,
	)}
}

// shellDegraded reports that shell lint could not run because no bash binary is
// available (e.g. Windows without WSL). It is informational and non-gating, but
// explicit — it must never read as a pass.
func shellDegraded(ctx EvalContext, rule Rule, count int) Finding {
	f := ruleFinding(ctx, rule, ctx.Inventory.RootPath,
		"Shell syntax lint skipped: no bash binary available",
		fmt.Sprintf("%d shell script(s) present but no `bash` was found on this host to syntax-check them", count),
		"a bash binary available to run `bash -n`",
		"bash not found; shell scripts were not syntax-checked",
	)
	f.Severity = "info"
	return f
}

// discoverShellScripts returns the scenario's `.sh` files, skipping vendored and
// build output trees.
func discoverShellScripts(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git", "dist", "build", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".sh" {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func relOrBase(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return filepath.Base(path)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func cap10(in []string) []string {
	if len(in) > 10 {
		return in[:10]
	}
	return in
}
