package playbooksclaims

import (
	"errors"
	"time"
)

// Mode names the playbooks execution path under a claim.
type Mode string

const (
	ModeRouted   Mode = "routed"
	ModeFallback Mode = "fallback"
)

// TTL is how long a claim is valid past its last heartbeat before it
// becomes reclaimable by a fresh acquire attempt.
const TTL = 2 * time.Minute

// HeartbeatInterval is the cadence at which an active run refreshes
// its claim. 4× heartbeat fits inside TTL to tolerate one missed beat.
const HeartbeatInterval = 30 * time.Second

// Claim is one row of the playbooks_claims table.
type Claim struct {
	ScenarioName string
	RunID        string
	Mode         Mode
	StartedBy    string
	AcquiredAt   time.Time
	HeartbeatAt  time.Time
	ExpiresAt    time.Time
}

// Alive reports whether the claim's heartbeat is still fresh relative to now.
func (c Claim) Alive(now time.Time) bool {
	return c.ExpiresAt.After(now)
}

// AcquireInput captures the caller identity for a new claim.
type AcquireInput struct {
	ScenarioName string
	RunID        string
	Mode         Mode
	StartedBy    string
}

// ErrBusy means another live claim already owns the scenario.
// Holder carries the active claim for caller-side messaging.
type ErrBusy struct {
	Holder Claim
}

func (e *ErrBusy) Error() string {
	return "scenario_busy: playbooks claim held by run " + e.Holder.RunID
}

// IsBusy reports whether err is an *ErrBusy (or wraps one).
func IsBusy(err error) bool {
	var b *ErrBusy
	return errors.As(err, &b)
}

// ErrLeaseMismatch is returned by Release/Heartbeat/ForceBreak when the
// caller-provided run_id does not match the current row owner.
var ErrLeaseMismatch = errors.New("playbooks claim lease mismatch")

// ErrNotFound is returned by Get/Release when no claim exists for the scenario.
var ErrNotFound = errors.New("playbooks claim not found")
