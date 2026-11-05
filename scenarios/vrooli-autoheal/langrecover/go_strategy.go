package langrecover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RecoverGo runs the appropriate Go recovery command for the given signature
// in <scenarioDir>/api/. It returns a Result describing what ran, any tracked
// file changes (go.mod, go.sum), and a non-nil Err if the command itself
// failed. If signature is GoSignatureNone, RecoverGo is a no-op returning
// (Result{}, nil) — callers should signature-gate before invoking.
//
// The two-tier mapping (download for missing-sum, tidy for missing-module) is
// intentional: `go mod download` is read-only with respect to go.mod and
// cheap; `go mod tidy` rewrites both files and is more expensive. Picking the
// minimum-viable command keeps recovery audits small.
func RecoverGo(ctx context.Context, runner Runner, scenarioDir string, sig GoSignature) (Result, error) {
	if runner == nil {
		runner = DefaultRunner
	}
	apiDir := filepath.Join(scenarioDir, "api")
	if _, err := os.Stat(filepath.Join(apiDir, "go.mod")); err != nil {
		return Result{Kind: KindGo}, fmt.Errorf("no go.mod under %s: %w", apiDir, err)
	}

	relWorkDir := relPath(scenarioDir, apiDir)
	result, runErr := runGoRecoverySequence(ctx, runner, apiDir, relWorkDir, KindGo, sig)
	if result.Err == nil {
		result.Err = runErr
	}
	return result, nil
}

// runGoRecoverySequence executes the recovery command for the given signature
// against modDir, then escalates to `go mod tidy` if go.sum was not modified
// for a MissingSum signature. The escalation handles cases where `go mod
// download` exits 0 without adding the required hashes (observed on Go 1.25
// when go.sum already has the /go.mod entry but not the package h1 hash).
func runGoRecoverySequence(ctx context.Context, runner Runner, modDir, relWorkDir string, kind Kind, sig GoSignature) (Result, error) {
	var args []string
	switch sig {
	case GoSignatureMissingSum:
		args = []string{"mod", "download"}
	case GoSignatureMissingModule:
		args = []string{"mod", "tidy"}
	case GoSignatureNone:
		return Result{}, nil
	default:
		return Result{}, fmt.Errorf("unknown go signature %v", sig)
	}

	pre := readGoModFingerprint(modDir)
	out, runErr := runner(ctx, modDir, "go", args...)
	post := readGoModFingerprint(modDir)
	combinedOut := string(out)
	commands := []string{"go " + strings.Join(args, " ")}

	sumChanged := pre["go.sum"] != post["go.sum"]
	modChanged := pre["go.mod"] != post["go.mod"]
	// Escalate: for MissingSum, if `go mod download` didn't actually update
	// go.sum, fall back to `go mod tidy`. download can succeed silently
	// without adding the missing h1 hash in some Go versions / sum-only states.
	if sig == GoSignatureMissingSum && !sumChanged && runErr == nil {
		tidyOut, tidyErr := runner(ctx, modDir, "go", "mod", "tidy")
		post = readGoModFingerprint(modDir)
		combinedOut = appendOutput(combinedOut, "\n--- escalated: go mod tidy ---\n", string(tidyOut))
		commands = append(commands, "go mod tidy")
		if tidyErr != nil {
			runErr = tidyErr
		}
		sumChanged = pre["go.sum"] != post["go.sum"]
		modChanged = pre["go.mod"] != post["go.mod"]
	}

	result := Result{
		Kind:       kind,
		Command:    strings.Join(commands, " && "),
		WorkingDir: relWorkDir,
		Output:     capOutput(combinedOut),
		// Recovery re-runs minimal version selection, so record every module
		// whose version moved. Callers escalate ChangedVersionDeltas: an
		// unattended heal must never bump a direct dependency invisibly.
		VersionDeltas: diffGoModVersions(pre["go.mod"], post["go.mod"]),
	}
	if modChanged {
		result.ModifiedTrackedFiles = true
		result.ModifiedPaths = append(result.ModifiedPaths, filepath.Join(relWorkDir, "go.mod"))
	}
	if sumChanged {
		result.ModifiedTrackedFiles = true
		result.ModifiedPaths = append(result.ModifiedPaths, filepath.Join(relWorkDir, "go.sum"))
	}
	if runErr != nil {
		result.Err = runErr
	}
	return result, runErr
}

func appendOutput(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	return b.String()
}

// RecoverRepoRootGo runs the appropriate Go recovery command for the given
// signature directly in the repo root (which holds its own go.mod). Mirrors
// RecoverGo but does not append "api/" to the directory — used to heal a
// broken top-level workspace (e.g., after a shared-package change cascades
// into go.mod drift).
func RecoverRepoRootGo(ctx context.Context, runner Runner, repoRoot string, sig GoSignature) (Result, error) {
	if runner == nil {
		runner = DefaultRunner
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		return Result{Kind: KindRepoRoot}, fmt.Errorf("no go.mod at repo root %s: %w", repoRoot, err)
	}

	result, runErr := runGoRecoverySequence(ctx, runner, repoRoot, ".", KindRepoRoot, sig)
	if result.Err == nil {
		result.Err = runErr
	}
	return result, nil
}

// readGoModFingerprint reads go.mod and go.sum and returns a map of filename to
// content. Missing files yield empty strings. This is intentionally not a
// cryptographic hash — strategies care about *whether* a file changed, not
// what the change was, and equality of byte content is faster and clearer.
func readGoModFingerprint(apiDir string) map[string]string {
	out := map[string]string{}
	for _, f := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(apiDir, f))
		if err != nil {
			out[f] = ""
			continue
		}
		out[f] = string(data)
	}
	return out
}

func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

// outputCap bounds the per-strategy output captured in ActionResult.Output to
// keep autoheal incident logs from blowing up on a verbose `go mod tidy`.
const outputCap = 16000

func capOutput(value string) string {
	if len(value) <= outputCap {
		return value
	}
	return value[:outputCap] + "\n[output truncated]"
}
