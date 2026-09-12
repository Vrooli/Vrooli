package custody

import (
	"context"
	"time"
)

type Record struct {
	ID           int64
	DocumentHash string
	Step         string
	Tier         int
	Provider     string
	Locality     string
	Profile      string
	PrivacyClass string
	State        string
	Reason       string
	Remedy       string
	StartedAt    time.Time
	Duration     time.Duration
}

type Receipt struct {
	DocumentHash string   `json:"document_hash"`
	SelfAttested bool     `json:"self_attested"`
	Records      []Record `json:"records"`
}

type Repository interface {
	Append(context.Context, Record) error
	List(context.Context, string) ([]Record, error)
}
