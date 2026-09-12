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
	Capacity, Reserved, DraftCount            int
}
