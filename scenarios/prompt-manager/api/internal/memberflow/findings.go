package memberflow

import "prompt-manager/internal/finding"

// Every validation result in this package is one type, defined in the leaf
// `finding` package so that teamcontract and heartbeat can produce and consume
// it without importing memberflow. These aliases are not a compatibility shim:
// they are the same type under the names the existing call sites already use,
// so `finding.Finding` and `memberflow.Finding` are indistinguishable to the
// compiler rather than converted between.
type (
	Finding               = finding.Finding
	OperatingGraphFinding = finding.Finding
	Severity              = finding.Severity
	Kind                  = finding.Kind
	RuleCatalog           = finding.RuleCatalog
	RuleCatalogEntry      = finding.RuleCatalogEntry
	RuleGroup             = finding.RuleGroup
)

const (
	SeverityError   = finding.SeverityError
	SeverityWarning = finding.SeverityWarning

	KindDeclaration = finding.KindDeclaration
	KindRuntime     = finding.KindRuntime
)

// NewRuleCatalog re-exports the leaf constructor so rule families keep reading
// as memberflow code.
func NewRuleCatalog(entries ...RuleCatalogEntry) (RuleCatalog, error) {
	return finding.NewRuleCatalog(entries...)
}
