package langrecover

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RecoverPnpm runs the appropriate pnpm recovery command for the given
// signature in <scenarioDir>/ui/. Behavior:
//
//   - PnpmSignatureOutdatedLockfile → pnpm install --ignore-workspace --no-frozen-lockfile
//     (rewrites pnpm-lock.yaml in place to match package.json)
//   - PnpmSignatureLinkingFailed     → rm -rf node_modules && pnpm install --ignore-workspace
//     (re-creates node_modules from a clean state)
//
// The --ignore-workspace flag is required because the root pnpm-workspace.yaml
// excludes scenarios; running without it picks up the wrong dependency graph.
func RecoverPnpm(ctx context.Context, runner Runner, scenarioDir string, sig PnpmSignature) (Result, error) {
	if runner == nil {
		runner = DefaultRunner
	}
	uiDir := filepath.Join(scenarioDir, "ui")
	if _, err := os.Stat(filepath.Join(uiDir, "package.json")); err != nil {
		return Result{Kind: KindPnpm}, fmt.Errorf("no package.json under %s: %w", uiDir, err)
	}

	relWorkDir := relPath(scenarioDir, uiDir)
	pre := readPnpmFingerprint(uiDir)
	var outBuilder strings.Builder
	var args []string

	switch sig {
	case PnpmSignatureOutdatedLockfile:
		args = []string{"install", "--ignore-workspace", "--no-frozen-lockfile"}
	case PnpmSignatureLinkingFailed:
		// Best-effort removal — RemoveAll on a missing path is a no-op, and a
		// stale node_modules with permission issues will surface in the
		// reinstall output anyway.
		if err := os.RemoveAll(filepath.Join(uiDir, "node_modules")); err != nil {
			outBuilder.WriteString(fmt.Sprintf("=== rm -rf node_modules failed: %v ===\n", err))
		} else {
			outBuilder.WriteString("=== rm -rf node_modules ===\n")
		}
		args = []string{"install", "--ignore-workspace"}
	case PnpmSignatureNone:
		return Result{}, nil
	default:
		return Result{}, fmt.Errorf("unknown pnpm signature %v", sig)
	}

	out, runErr := runner(ctx, uiDir, "pnpm", args...)
	outBuilder.Write(out)
	post := readPnpmFingerprint(uiDir)

	result := Result{
		Kind:       KindPnpm,
		Command:    "pnpm " + strings.Join(args, " "),
		WorkingDir: relWorkDir,
		Output:     capOutput(outBuilder.String()),
	}
	for _, f := range []string{"pnpm-lock.yaml", "package.json"} {
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

func readPnpmFingerprint(uiDir string) map[string]string {
	out := map[string]string{}
	for _, f := range []string{"pnpm-lock.yaml", "package.json"} {
		data, err := os.ReadFile(filepath.Join(uiDir, f))
		if err != nil {
			out[f] = ""
			continue
		}
		out[f] = string(data)
	}
	return out
}
