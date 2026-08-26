// Package artifactledger exposes the control plane's removal ledger to scenario
// consumers.
//
// Scenarios remove install-root artifacts too -- storage-manager's orphan
// reaper is the clearest case, and the hardest kind of removal to reconstruct
// after the fact, because it decides for itself what to delete. Those removals
// need receipts for exactly the same reason the control plane's do, but a
// scenario module cannot import internal/artifactledger: Go's internal rule
// admits only packages under github.com/vrooli/vrooli/, and scenario modules
// carry their own module paths. This package is the sanctioned door, in the
// same shape as packages/hostreq.
//
// The types are aliases rather than re-declarations. packages/hostreq
// deliberately re-declares its types to stop a read-only observation being
// mistaken for proof of a repair; there is no such boundary to defend here,
// because a scenario writing a receipt means precisely what the control plane
// means by it. Aliasing keeps one definition and removes any chance of the two
// drifting into subtly different receipt shapes.
package artifactledger

import ledger "github.com/vrooli/vrooli/internal/artifactledger"

// ErrAbandoned reports that a removal was refused once the lock was held,
// because the predicate that authorized it no longer held. Callers must treat
// it as skipped, never as reclaimed.
var ErrAbandoned = ledger.ErrAbandoned

// Schema identifies the receipt format written by this ledger.
const Schema = ledger.Schema

// Outcome values close an intent record.
const (
	OutcomeIntent  = ledger.OutcomeIntent
	OutcomeRemoved = ledger.OutcomeRemoved
	OutcomeAbsent  = ledger.OutcomeAbsent
	OutcomeFailed  = ledger.OutcomeFailed
	// OutcomeAbandoned records a removal refused once the lock was held,
	// because the predicate that authorized it no longer held.
	OutcomeAbandoned = ledger.OutcomeAbandoned
)

type (
	// Ledger appends removal receipts to the runtime state directory.
	Ledger = ledger.Ledger
	// Removal describes one artifact a caller intends to remove.
	Removal = ledger.Removal
	// Receipt is one durable line in the ledger.
	Receipt = ledger.Receipt
	// Identity is who performed a removal, graded by how much it can be trusted.
	Identity = ledger.Identity
	// Claimed is identity asserted through the environment. Never proof.
	Claimed = ledger.Claimed
	// Verified is identity authenticated upstream.
	Verified = ledger.Verified
)

// New returns a ledger writing beneath the runtime state directory for home.
func New(home string) (*Ledger, error) { return ledger.New(home) }

// NewAt returns a ledger writing into an explicit directory.
func NewAt(dir string) *Ledger { return ledger.NewAt(dir) }

// CurrentIdentity resolves the observed and claimed identity of this process.
// It never returns a verified identity; verification happens upstream.
func CurrentIdentity() Identity { return ledger.CurrentIdentity() }
