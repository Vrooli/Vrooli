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

	pre := readGoModFingerprint(apiDir)
	out, runErr := runner(ctx, apiDir, "go", args...)
	post := readGoModFingerprint(apiDir)

	relWorkDir := relPath(scenarioDir, apiDir)
	result := Result{
		Kind:       KindGo,
		Command:    "go " + strings.Join(args, " "),
		WorkingDir: relWorkDir,
		Output:     capOutput(string(out)),
	}
	for _, f := range []string{"go.mod", "go.sum"} {
		if pre[f] != post[f] {
			result.ModifiedTrackedFiles = true
			result.ModifiedPaths = append(result.ModifiedPaths, filepath.Join(relWorkDir, f))
		}
	}
	if runErr != nil {
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
