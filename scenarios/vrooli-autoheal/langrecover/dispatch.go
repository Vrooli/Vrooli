package langrecover

import (
	"os"
	"path/filepath"
)

// Decision describes which strategy applies for a given (failure log,
// scenario layout) pair. Callers use this to gate the corresponding
// auto-heal action ID.
type Decision struct {
	Kind        Kind
	GoSig       GoSignature
	PnpmSig     PnpmSignature
	ScenarioDir string
}

// Has reports whether the decision selected a healable strategy.
func (d Decision) Has() bool {
	if d.Kind == KindGo || d.Kind == KindRepoRoot {
		return d.GoSig != GoSignatureNone
	}
	if d.Kind == KindPnpm {
		return d.PnpmSig != PnpmSignatureNone
	}
	return false
}

// Decide inspects the failure log and scenario directory layout to pick a
// recovery strategy. Returns Decision{} when nothing matches.
//
// Decision rules (first match wins):
//  1. Go signature in log + api/go.mod present → KindGo
//  2. pnpm signature in log + ui/package.json present → KindPnpm
//  3. otherwise → no-op
func Decide(failureLog, scenarioDir string) Decision {
	if scenarioDir == "" {
		return Decision{}
	}
	hasGo := exists(filepath.Join(scenarioDir, "api", "go.mod"))
	hasPnpm := exists(filepath.Join(scenarioDir, "ui", "package.json"))

	if hasGo {
		if sig := DetectGoSignature(failureLog); sig != GoSignatureNone {
			return Decision{Kind: KindGo, GoSig: sig, ScenarioDir: scenarioDir}
		}
	}
	if hasPnpm {
		if sig := DetectPnpmSignature(failureLog); sig != PnpmSignatureNone {
			return Decision{Kind: KindPnpm, PnpmSig: sig, ScenarioDir: scenarioDir}
		}
	}
	return Decision{}
}

// DecideRepoRoot inspects a failure log and the repo root path; if a Go
// signature matches and go.mod is present at the repo root, returns a
// Decision{Kind: KindRepoRoot}. Used to heal a broken top-level workspace
// distinct from any single scenario.
func DecideRepoRoot(failureLog, repoRoot string) Decision {
	if repoRoot == "" || !exists(filepath.Join(repoRoot, "go.mod")) {
		return Decision{}
	}
	if sig := DetectGoSignature(failureLog); sig != GoSignatureNone {
		return Decision{Kind: KindRepoRoot, GoSig: sig, ScenarioDir: repoRoot}
	}
	return Decision{}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
