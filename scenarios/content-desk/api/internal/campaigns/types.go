package campaigns

import "errors"

const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusClosed   = "closed"
)

var (
	ErrEvidenceRequired = errors.New("campaign activation requires at least one evidence reference")
	ErrSlotExhausted    = errors.New("campaign artifact slot budget is exhausted")
)

// Launch-asset readiness tiers. A slot reports its strongest attached draft
// tier so an advisory reader can tell approved work from reviewable and
// in-progress work, and from a genuinely empty slot (never a silent zero).
const (
	LaunchAssetReadinessEmpty          = "empty"
	LaunchAssetReadinessInProgress     = "in_progress"
	LaunchAssetReadinessReadyForReview = "ready_for_review"
	LaunchAssetReadinessApproved       = "approved"
)

type Campaign struct {
	ID            string
	Name          string
	Status        string
	ScenarioNames []string
}

type Slot struct {
	Channel  string
	Format   string
	Capacity int
	Reserved int
}

type LaunchAssetSlot struct {
	CampaignID, CampaignName, Channel, Format string
	Capacity                                  int
	Reserved                                  int
	// DraftCount is the legacy approved/published count; it mirrors
	// ApprovedCount for compatibility.
	DraftCount int
	// ApprovedCount counts drafts that reached approved or published.
	ApprovedCount int
	// ReadyForReviewCount counts drafts that produced a reviewable asset
	// (drafted/checking/reviewed) but are not yet operator-approved.
	ReadyForReviewCount int
	// InProgressCount counts remaining non-abandoned drafts (requested,
	// drafting, blocked) so an empty slot is distinguishable from work in
	// flight.
	InProgressCount int
	// Readiness is the strongest attached tier: empty, in_progress,
	// ready_for_review, or approved.
	Readiness string
}

// ReadinessTier derives the strongest launch-asset tier from the slot's
// attached draft counts. Approved always outranks reviewable, reviewable
// outranks in-progress, and a slot with no attached work is empty.
func (s LaunchAssetSlot) ReadinessTier() string {
	switch {
	case s.ApprovedCount > 0:
		return LaunchAssetReadinessApproved
	case s.ReadyForReviewCount > 0:
		return LaunchAssetReadinessReadyForReview
	case s.InProgressCount > 0:
		return LaunchAssetReadinessInProgress
	default:
		return LaunchAssetReadinessEmpty
	}
}
