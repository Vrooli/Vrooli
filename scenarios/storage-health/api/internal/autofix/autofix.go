// Package autofix is storage-health's deterministic remediation registry. It
// binds per-rule Fixers to the shared github.com/vrooli/maturity-go/autofix
// orchestrator, so PreviewFix/ApplyFix over the shared ScenarioValidationService
// drive real, idempotent file edits.
//
// Each Fixer's Preview re-derives the broken state from the tree on disk, so a
// second Apply is a no-op (the orchestrator's idempotency guarantee). The
// detection here intentionally mirrors the matching analyzer in
// internal/validation so a fixer only ever remediates exactly what the analyzer
// flags — it never edits code the analyzer would leave alone.
//
// Coverage is deliberately conservative. Two targets are remediated
// deterministically: DB_ROWS_NOT_CLOSED (insert `defer rows.Close()`) and the
// unambiguous //go:embed sub-case of ENSURE_SCHEMAS_NOT_WIRED (scaffold a
// schema.go beside an un-embedded domain schema.sql). The other auto-class
// targets (ROUTED_SEAMS_UNWIRED, STORAGE_NAMESPACE_HARDCODED, SCHEMA_CENTRALIZED)
// require judgment that cannot be made mechanically without risking wrong edits,
// so they are left without a fixer and their maturity.json fixer_status stays
// "pending".
package autofix

import (
	autofixcore "github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Rule IDs this registry can remediate. They match the finding Codes emitted by
// internal/validation and the keys in .vrooli/maturity.json.
const (
	RuleDBRowsNotClosed     = "DB_ROWS_NOT_CLOSED"
	RuleEnsureSchemasUnwire = "ENSURE_SCHEMAS_NOT_WIRED"
)

// Candidate aliases the shared auto-fix candidate so storage-health consumers
// keep a single source of truth with the maturity-go/autofix orchestrator.
type Candidate = autofixcore.Candidate

// registry is the storage-health fixer set bound to the shared orchestrator.
var registry = autofixcore.NewRegistry(
	autofixcore.Fixer{RuleID: RuleDBRowsNotClosed, Preview: previewRowsClose, CanFix: canFixRowsClose},
	autofixcore.Fixer{RuleID: RuleEnsureSchemasUnwire, Preview: previewEnsureEmbed, CanFix: canFixEnsureEmbed},
)

// CoveredCodes is the set of finding Codes a registered fixer can remediate.
// The Connect handler stamps Finding.AutofixAvailable from this set so the
// shared AutofixableCount reflects exactly what this registry can fix.
var CoveredCodes = map[string]bool{
	RuleDBRowsNotClosed:     true,
	RuleEnsureSchemasUnwire: true,
}

// Preview returns the candidate edits for the requested rules (all when empty)
// without writing anything.
func Preview(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Preview(root, ruleIDs)
}

// Apply previews then writes the candidate edits for the requested rules.
// Idempotent: a second Apply over an already-fixed tree returns no candidates.
func Apply(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Apply(root, ruleIDs)
}

// CanFix reports whether the rule can currently remediate the given finding.
func CanFix(root, ruleID, findingPath string) bool {
	return registry.CanFix(root, ruleID, findingPath)
}

// PreviewFixResponse previews the registry's remediations for the resolved
// scenario directory and returns the shared FixResponse.
func PreviewFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	return registry.PreviewFixResponse(scenario, root, ruleIDs)
}

// ApplyFixResponse applies the registry's remediations for the resolved
// scenario directory and returns the shared FixResponse with applied=true.
func ApplyFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	return registry.ApplyFixResponse(scenario, root, ruleIDs)
}
