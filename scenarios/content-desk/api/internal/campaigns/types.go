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
	ID     string
	Name   string
	Status string
}

type Slot struct {
	Channel  string
	Format   string
	Capacity int
	Reserved int
}
