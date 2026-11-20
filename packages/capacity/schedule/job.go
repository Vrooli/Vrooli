package schedule

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// State is the durable lifecycle position of a job.
//
// queued and deferred are both "waiting" states; deferral differs only in why
// the job is waiting and is reported by Status. running means a lease is held
// and execution is in flight. completed, failed and canceled are terminal.
type State string

const (
	// StateQueued is a waiting job that has never been deferred.
	StateQueued State = "queued"
	// StateDeferred is a waiting job whose next attempt is gated by NotBefore.
	StateDeferred State = "deferred"
	// StateRunning is a job whose lease is held by a runner.
	StateRunning State = "running"
	// StateCompleted is a job that finished successfully.
	StateCompleted State = "completed"
	// StateFailed is a job that exhausted its attempts or its deadline.
	StateFailed State = "failed"
	// StateCanceled is a job an owner withdrew.
	StateCanceled State = "canceled"
)

// Terminal reports whether no further transition can run the job.
func (s State) Terminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCanceled:
		return true
	}
	return false
}

// Waiting reports whether the job can still be leased.
func (s State) Waiting() bool { return s == StateQueued || s == StateDeferred }

func (s State) valid() bool {
	switch s {
	case StateQueued, StateDeferred, StateRunning, StateCompleted, StateFailed, StateCanceled:
		return true
	}
	return false
}

// Mode selects how much background work the scheduler may start.
type Mode string

const (
	// ModeOff disables the scheduler entirely.
	ModeOff Mode = "off"
	// ModePassive runs only explicit interactive-priority jobs.
	ModePassive Mode = "passive"
	// ModeIdle fills capacity only while the host observation is idle and the
	// clock is inside a configured window.
	ModeIdle Mode = "idle"
	// ModeBalanced runs inside windows within budgets.
	ModeBalanced Mode = "balanced"
	// ModeEager takes any capacity the broker admits.
	ModeEager Mode = "eager"
)

// Valid reports whether the mode is one of the five defined modes.
func (m Mode) Valid() bool {
	switch m {
	case ModeOff, ModePassive, ModeIdle, ModeBalanced, ModeEager:
		return true
	}
	return false
}

// ParseMode converts an operator/config string into a Mode.
func ParseMode(raw string) (Mode, error) {
	mode := Mode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		return ModeBalanced, nil
	}
	if !mode.Valid() {
		return "", fmt.Errorf("schedule: unknown mode %q (want off|passive|idle|balanced|eager)", raw)
	}
	return mode, nil
}

// Priority is the admission priority a job runs under. The names and ordering
// match the capacity broker's tiers so the adapter does not translate policy.
type Priority string

const (
	// PriorityInteractive is user-facing live work.
	PriorityInteractive Priority = "interactive"
	// PriorityService is resident service work.
	PriorityService Priority = "service"
	// PriorityBatch is background work; deferred work defaults here.
	PriorityBatch Priority = "batch"
)

// Valid reports whether the priority is a known tier.
func (p Priority) Valid() bool {
	switch p {
	case PriorityInteractive, PriorityService, PriorityBatch:
		return true
	}
	return false
}

// rank orders tiers from lowest to highest.
func (p Priority) rank() int {
	switch p {
	case PriorityInteractive:
		return 3
	case PriorityService:
		return 2
	default:
		return 1
	}
}

// ResourceClass names the capacity dimension a job is admitted against.
type ResourceClass string

const (
	// ResourceClassCompute admits RAM + CPU through the broker.
	ResourceClassCompute ResourceClass = "compute"
	// ResourceClassVRAM admits GPU memory for one bound device identity.
	ResourceClassVRAM ResourceClass = "vram"
	// ResourceClassNone skips admission. It exists for dry runs and tests;
	// production work must use compute or vram.
	ResourceClassNone ResourceClass = "none"
)

// Valid reports whether the resource class is known.
func (r ResourceClass) Valid() bool {
	switch r {
	case ResourceClassCompute, ResourceClassVRAM, ResourceClassNone:
		return true
	}
	return false
}

// Estimate is a job's expected cost. Seconds is what the value density and the
// seconds budget are computed from; RAMBytes/VRAMBytes are what admission is
// asked for; Items and Tokens describe the work for status and actual-cost
// comparison.
type Estimate struct {
	Seconds   float64 `json:"seconds"`
	RAMBytes  int64   `json:"ramBytes"`
	VRAMBytes int64   `json:"vramBytes"`
	Items     int     `json:"items"`
	Tokens    int     `json:"tokens"`
}

// Validate rejects negative costs. Zero is allowed: a job with no measured
// estimate still runs, it just cannot be charged against a seconds budget.
func (e Estimate) Validate() error {
	if e.Seconds < 0 {
		return fmt.Errorf("%w: estimate seconds cannot be negative", ErrInvalidJob)
	}
	if e.RAMBytes < 0 || e.VRAMBytes < 0 {
		return fmt.Errorf("%w: estimate memory cannot be negative", ErrInvalidJob)
	}
	if e.Items < 0 || e.Tokens < 0 {
		return fmt.Errorf("%w: estimate items/tokens cannot be negative", ErrInvalidJob)
	}
	return nil
}

// Job is one durable unit of deferred work.
//
// ID is the idempotency key: submitting the same ID twice is rejected, so a
// producer that crashes between writing its own record and submitting can retry
// safely. PayloadRef is an opaque pointer the executor resolves (a query, a
// file, a run id); Checkpoint is opaque resumable progress owned by the
// executor. Neither is interpreted by this package.
type Job struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	Owner         string        `json:"owner"`
	Priority      Priority      `json:"priority"`
	ResourceClass ResourceClass `json:"resourceClass"`
	ModelAffinity string        `json:"modelAffinity"`
	Estimate      Estimate      `json:"estimate"`
	Value         float64       `json:"value"`
	NotBefore     time.Time     `json:"notBefore"`
	Deadline      *time.Time    `json:"deadline,omitempty"`
	BatchKey      string        `json:"batchKey,omitempty"`
	MaxBatch      int           `json:"maxBatch,omitempty"`
	MaxAttempts   int           `json:"maxAttempts,omitempty"`
	PayloadRef    string        `json:"payloadRef,omitempty"`
	Checkpoint    string        `json:"checkpoint,omitempty"`
	State         State         `json:"state"`
	Attempts      int           `json:"attempts"`
	Deferrals     int           `json:"deferrals"`
	LastReason    string        `json:"lastReason,omitempty"`

	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	FinishedAt     *time.Time `json:"finishedAt,omitempty"`
	LeaseOwner     string     `json:"leaseOwner,omitempty"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt,omitempty"`
}

// Validate reports whether the job can be submitted. It checks only structural
// invariants; policy concerns (windows, budgets) are evaluated at run time.
func (j Job) Validate() error {
	if strings.TrimSpace(j.Kind) == "" {
		return fmt.Errorf("%w: kind is required", ErrInvalidJob)
	}
	if strings.ContainsAny(j.Kind, " \t\n") {
		return fmt.Errorf("%w: kind %q must not contain whitespace", ErrInvalidJob, j.Kind)
	}
	if strings.TrimSpace(j.Owner) == "" {
		return fmt.Errorf("%w: owner is required", ErrInvalidJob)
	}
	if j.Priority != "" && !j.Priority.Valid() {
		return fmt.Errorf("%w: unknown priority %q", ErrInvalidJob, j.Priority)
	}
	if j.ResourceClass != "" && !j.ResourceClass.Valid() {
		return fmt.Errorf("%w: unknown resource class %q", ErrInvalidJob, j.ResourceClass)
	}
	if j.Value < 0 {
		return fmt.Errorf("%w: value cannot be negative", ErrInvalidJob)
	}
	if j.MaxBatch < 0 {
		return fmt.Errorf("%w: max batch cannot be negative", ErrInvalidJob)
	}
	if j.MaxAttempts < 0 {
		return fmt.Errorf("%w: max attempts cannot be negative", ErrInvalidJob)
	}
	if err := j.Estimate.Validate(); err != nil {
		return err
	}
	if j.Deadline != nil && !j.Deadline.After(j.NotBefore) {
		return fmt.Errorf("%w: deadline must be after not_before", ErrInvalidJob)
	}
	return nil
}

// normalize fills defaults for a new job. It is applied by Submit, never by
// scanning, so a stored job always round-trips exactly as written.
func (j Job) normalize(now time.Time) (Job, error) {
	if err := j.Validate(); err != nil {
		return Job{}, err
	}
	if strings.TrimSpace(j.ID) == "" {
		j.ID = NewID("job")
	}
	if j.Priority == "" {
		j.Priority = PriorityBatch
	}
	if j.ResourceClass == "" {
		j.ResourceClass = ResourceClassCompute
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = DefaultMaxAttempts
	}
	if j.NotBefore.IsZero() {
		j.NotBefore = now
	}
	j.NotBefore = j.NotBefore.UTC()
	if j.Deadline != nil {
		at := j.Deadline.UTC()
		if !at.After(j.NotBefore) {
			return Job{}, fmt.Errorf("%w: deadline must be after not_before", ErrInvalidJob)
		}
		j.Deadline = &at
	}
	j.State = StateQueued
	j.CreatedAt = now
	j.UpdatedAt = now
	return j, nil
}

// Due reports whether the job's NotBefore has arrived at at.
func (j Job) Due(at time.Time) bool { return !j.NotBefore.After(at) }

// Overdue reports whether the job's deadline has passed at at.
func (j Job) Overdue(at time.Time) bool {
	return j.Deadline != nil && !j.Deadline.After(at)
}

// ValueDensity is the selection score of a set of jobs: total declared value per
// estimated second. A zero-second batch is treated as one second so an
// unestimated job still competes instead of dividing by zero.
func ValueDensity(value, seconds float64) float64 {
	if seconds < 1 {
		seconds = 1
	}
	if value < 0 {
		value = 0
	}
	return value / seconds
}

// NewID returns a random, prefixed identifier. Callers may supply their own job
// IDs; this exists so producers do not have to invent an identity scheme.
func NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

// ErrInvalidJob marks a structurally invalid job or policy value.
var ErrInvalidJob = errors.New("schedule: invalid job")
