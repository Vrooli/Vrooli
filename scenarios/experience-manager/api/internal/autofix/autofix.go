// Package autofix hosts experience-manager's deterministic remediation
// registry on the shared maturity-go substrate.
package autofix

import "github.com/vrooli/maturity-go/autofix"

// NewRegistry returns the scenario's fixer registry. Phase 1 intentionally
// registers no fixers.
func NewRegistry() *autofix.Registry {
	return autofix.NewRegistry()
}
