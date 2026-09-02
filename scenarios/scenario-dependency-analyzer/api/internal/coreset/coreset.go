// Package coreset exposes the database-free supervision closure through the
// Scenario Dependency Analyzer's existing core-set surface.
package coreset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/vrooli/internal/app/supervision"
)

// Result is retained as a compatibility alias for the existing API and CLI.
type Result = apicoreset.Report

// Compute returns the live set rooted at scenariosDir. It never reads a
// database and always retains the operator seed.
func Compute(scenariosDir string) Result {
	authority := apicoreset.Authority{
		Seed:        apicoreset.CoreSeedScenarios(),
		TrustedBase: apicoreset.TrustedBaseScenarios(),
	}
	return compute(scenariosDir, authority)
}

func compute(scenariosDir string, authority apicoreset.Authority) Result {
	return supervision.Compute(scenariosDir, authority)
}

// ValidateTrustedBaseClosure rejects an operator grant whose must-start
// scenario closure escapes the trusted base or cannot be read.
func ValidateTrustedBaseClosure(repoRoot string) error {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return fmt.Errorf("repository root is required")
	}
	authority, err := apicoreset.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("load operator core authority: %w", err)
	}
	result := compute(filepath.Join(repoRoot, "scenarios"), authority)
	if len(result.TrustedBaseViolations) > 0 {
		return fmt.Errorf("trusted-base required dependency closure is invalid: %v", result.TrustedBaseViolations)
	}
	for _, name := range authority.TrustedBase {
		if message, ok := result.LoadErrors[name]; ok {
			return fmt.Errorf("trusted-base member %q cannot be verified: %s", name, message)
		}
	}
	return nil
}

// ValidateConfiguredTrustedBaseClosure applies the strict check only when the
// repository has declared operator state.
func ValidateConfiguredTrustedBaseClosure(repoRoot string) error {
	statePath := filepath.Join(strings.TrimSpace(repoRoot), ".vrooli", "operator-state.json")
	if _, err := os.Stat(statePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect operator core authority: %w", err)
	}
	return ValidateTrustedBaseClosure(repoRoot)
}
