package scenariocli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
)

var userHomeDir = os.UserHomeDir

// Stored sidecar paths and fingerprints are recomputed by later releases.
// These identity algorithms are pinned in the root persistence contract.
// scenarioFreshnessSpec returns the canonical freshness contract used by both
// cli-installer (at install time) and cli-core's runtime StaleChecker. Both
// must evaluate the same spec to avoid perpetual reinstall loops.
func scenarioFreshnessSpec(item scenarioCLI) (cliutil.FreshnessSpec, error) {
	var customInputs []string
	if item.config != nil && item.config.Freshness != nil {
		customInputs = item.config.Freshness.Inputs
	}
	switch item.config.Adapter.Kind {
	case "go_module":
		return cliutil.CanonicalScenarioGoModuleFreshnessSpec(item.scenarioPath, item.modulePath, item.binaryName, customInputs), nil
	default:
		return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported scenario CLI adapter kind %q", item.config.Adapter.Kind)
	}
}

func installDir(home string) (string, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		resolved, err := userHomeDir()
		if err != nil {
			return "", err
		}
		home = resolved
	}
	return repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
}

func installedBinaryPath(home, binaryName string) (string, error) {
	dir, err := installDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, strings.TrimSpace(binaryName)), nil
}

func installMetadataPath(home string, item scenarioCLI) (string, error) {
	binPath, err := installedBinaryPath(home, item.binaryName)
	if err != nil {
		return "", err
	}
	return binPath + ".build.meta", nil
}

func computeScenarioCLIFingerprint(item scenarioCLI) (string, error) {
	spec, err := scenarioFreshnessSpec(item)
	if err != nil {
		return "", err
	}
	return cliutil.ComputeFreshnessFingerprint(spec)
}
