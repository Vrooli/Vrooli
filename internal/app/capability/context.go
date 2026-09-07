package capabilityapp

import "github.com/vrooli/vrooli/internal/operatorcapability"

// LedgerOptions selects the capability ledger readout.
type LedgerOptions struct {
	Fleet bool
	Query string
	JSON  bool
}

// ConformanceOptions selects the capability declaration check.
type ConformanceOptions struct {
	DeclarationsOnly bool
	JSON             bool
}

// WorkflowOptions selects a capability catalog/status/preview/apply request.
// Preview and apply consume Request; catalog and status do not.
type WorkflowOptions struct {
	Action  string
	JSON    bool
	Request operatorcapability.ActionRequest
}
