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
