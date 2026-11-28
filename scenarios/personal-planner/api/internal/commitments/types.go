package commitments

import "fmt"

const (
	StateProposed         = "proposed"
	StateActive           = "active"
	StateFulfilled        = "fulfilled"
	StateCancelled        = "cancelled"
	RiskUnknown           = "unknown"
	AcknowledgmentUnknown = "unknown"
)

type Commitment struct {
	ID, Result, DefinitionOfDone, PromisedBoundary, Timezone, Beneficiary, Assumptions, ScopeExclusions, State, Risk, AcknowledgmentStatus string
	CreatedAt, UpdatedAt, Revision                                                                                                         int64
}

type Revision struct {
	ID, CommitmentID, PromisedBoundary, Assumptions, ScopeExclusions, Reason, AcknowledgmentStatus string
	CreatedAt, Revision                                                                            int64
}

type CreateInput struct {
	Result, DefinitionOfDone, PromisedBoundary, Timezone, Beneficiary, Assumptions, ScopeExclusions, State string
}

type ReviseInput struct {
	ID, PromisedBoundary, Assumptions, ScopeExclusions, Reason, AcknowledgmentStatus string
	ExpectedRevision                                                                 int64
}

type ErrInvalidCommitment struct{ Field, Reason string }

func (e ErrInvalidCommitment) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrCommitmentNotFound struct{ ID string }

func (e ErrCommitmentNotFound) Error() string { return fmt.Sprintf("commitment %q not found", e.ID) }

type ErrRevisionConflict struct{ ID string }

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("commitment %q changed; reload before updating", e.ID)
}
