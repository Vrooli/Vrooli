// Package operations exposes the canonical aggregate-activity surface used by
// the Operations Center page (UI), `/api/v1/operations` (HTTP), and any
// future fan-in surface that needs a bird's-eye view of agentic work.
//
// The package owns one seam: a time-bounded join over (a) the agentactivity
// ledger, (b) the operating-mode round projection, and (c) the governance
// lane caps. Everything the operations view needs comes through Aggregator —
// callers do not reach past it into agentactivity / operatingmode / settings
// to recompute parts of the view.
//
// See docs/internal/SEAMS.md "Operations Aggregate" for the contract.
package operations
