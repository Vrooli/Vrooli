package campaign

import "fmt"

// ErrCampaignNotFound is returned when a campaign id does not resolve.
type ErrCampaignNotFound struct{ ID string }

func (e ErrCampaignNotFound) Error() string {
	return fmt.Sprintf("campaign %q not found", e.ID)
}

// ErrFindingNotFound is returned when a (campaign, stableID) pair does not
// resolve.
type ErrFindingNotFound struct {
	CampaignID string
	StableID   string
}

func (e ErrFindingNotFound) Error() string {
	return fmt.Sprintf("finding %q not found in campaign %q", e.StableID, e.CampaignID)
}

// ErrInvalidInput is returned for empty/invalid required arguments.
type ErrInvalidInput struct{ Reason string }

func (e ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input: %s", e.Reason)
}

// ErrInvalidFindingTransition identifies a finding state edge the domain does not allow.
type ErrInvalidFindingTransition struct {
	From FindingStatus
	To   FindingStatus
}

func (e ErrInvalidFindingTransition) Error() string {
	return fmt.Sprintf("invalid finding status transition: %q -> %q", e.From, e.To)
}

// ErrFindingWriteConflict means another writer changed the stored state first.
type ErrFindingWriteConflict struct {
	CampaignID string
	StableID   string
	Expected   FindingStatus
}

func (e ErrFindingWriteConflict) Error() string {
	return fmt.Sprintf("finding %q in campaign %q no longer has expected status %q", e.StableID, e.CampaignID, e.Expected)
}

// ErrInvalidFindingStatus indicates corrupt or unsupported stored state.
type ErrInvalidFindingStatus struct{ Status FindingStatus }

func (e ErrInvalidFindingStatus) Error() string {
	return fmt.Sprintf("invalid persisted finding status %q", e.Status)
}
