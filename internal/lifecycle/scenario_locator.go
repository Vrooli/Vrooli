package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/scenariocli"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

func LocateTestGenieCLI(lookPath func(string) (string, error), root, home string) (string, error) {
	if override := strings.TrimSpace(os.Getenv("VROOLI_TEST_GENIE_CLI")); override != "" {
		if isExecutable(override) {
			return override, nil
		}
	}
	_ = lookPath
	path, err := scenariocli.ResolveExecutable(root, home, "test-genie")
	if err != nil {
		homeCLI, _ := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
		homeCLI = filepath.Join(homeCLI, "test-genie")
		return "", fmt.Errorf("test-genie CLI not found via manifest-driven resolution (checked VROOLI_TEST_GENIE_CLI and attempted %s): %w", homeCLI, err)
	}
	return path, nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

func LocateScenarioCompletenessCLI(lookPath func(string) (string, error), root string) (string, error) {
	_ = lookPath
	home, _ := config.HomeDir()
	path, err := scenariocli.ResolveExecutable(root, home, "scenario-completeness-scoring")
	if err != nil {
		homeCLI, _ := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
		homeCLI = filepath.Join(homeCLI, "scenario-completeness-scoring")
		return "", fmt.Errorf("scenario-completeness-scoring CLI not found via manifest-driven resolution (attempted %s): %w", homeCLI, err)
	}
	return path, nil
}
