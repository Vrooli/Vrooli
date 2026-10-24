package paths

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ResolveCapturesRoot returns an absolute base directory for capture output
// bundles. Relative --out values (and the default when --out is omitted)
// resolve under this root so capture artifacts land in scenario-owned storage
// instead of whatever working directory the API process happened to start in.
func ResolveCapturesRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_CAPTURES_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_CAPTURES_ROOT without normalization")
		}
		return value
	}

	if resolved := resolveScenarioStoragePath("captures"); resolved != "" {
		return resolved
	}

	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if scenarioDir, resolveErr := repocontract.ResolveScenarioPath(root, scenarioRoot); resolveErr == nil {
			return filepath.Join(scenarioDir, "data", "captures")
		} else if log != nil {
			log.WithError(resolveErr).Warn("Failed to resolve captures root from repo contract; falling back to cwd-derived path")
		}
	} else if log != nil {
		log.WithError(err).Warn("Failed to resolve repo root for captures; falling back to cwd-derived path")
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for captures root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "captures")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				captures := filepath.Join(dir, "data", "captures")
				if abs, err := filepath.Abs(captures); err == nil {
					return abs
				}
				return captures
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "captures")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}
