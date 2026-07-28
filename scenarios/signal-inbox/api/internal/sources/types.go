// Package sources owns ingestion adapters and the centrally enforced safety
// envelope around them. It writes through the signals service; it never edits
// the immutable journal directly.
package sources

import (
	"context"
	"fmt"
	"io"
	"time"

	"signal-inbox/internal/signals"
)

type RiskTier int

const (
	RiskUnspecified RiskTier = iota
	RiskTier0
	RiskTier1
	RiskTier2
)

type Descriptor struct {
	ID       string
	Kind     string
	RiskTier RiskTier
}

type Adapter interface {
	Descriptor() Descriptor
	Parse(context.Context, io.Reader) ([]signals.CaptureInput, error)
}

type State struct {
	AdapterID, Kind string
	RiskTier        RiskTier
	Enabled         bool
	LastRunAt       time.Time
	LastError       string
	DisabledReason  string
}

type ImportResult struct {
	RunID, AdapterID            string
	Created, Duplicated, Failed int
}

type Repository interface {
	GetState(context.Context, string) (State, bool, error)
	PutState(context.Context, State) error
	AppendRun(context.Context, ImportResult, time.Time, time.Time) error
}

type CaptureService interface {
	Capture(context.Context, signals.CaptureInput) (signals.CaptureResult, error)
}

type ErrInvalidDescriptor struct{ Reason string }

func (e ErrInvalidDescriptor) Error() string { return e.Reason }

type ErrAdapterDisabled struct{ ID, Reason string }

func (e ErrAdapterDisabled) Error() string {
	return fmt.Sprintf("adapter %q is disabled: %s", e.ID, e.Reason)
}

type ErrUnknownAdapter struct{ ID string }

func (e ErrUnknownAdapter) Error() string { return fmt.Sprintf("unknown adapter %q", e.ID) }

type ErrAnomalousResponse struct{ Reason string }

func (e ErrAnomalousResponse) Error() string { return e.Reason }
