// Package providerpool owns durable provider/credential-pool reservation and
// typed runner-limit recovery for a finite effort. One pool document is shared
// by every descendant that draws on the same credential/allowance pool, so
// concurrent or recursive workers cannot oversubscribe it.
//
// It keeps two things in one owner document:
//
//   - a typed observation log, so an exhausted session/weekly allowance, a
//     rate limit, credit/spend exhaustion or an auth refusal pauses the pool
//     with an explicit reset condition instead of a generic retry; and
//   - conservative reservations, so a dispatch is accounted before an owner is
//     contacted and an unknown charge stays reserved rather than reading as
//     zero.
//
// A known reset time clears the pause only when the clock passes the observed
// reset. An unknown reset stays blocked until a newer explicit recovery
// observation arrives; a repeated observation does not clear it, so a heartbeat
// cannot become a polling timer.
package providerpool

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Runner-limit recovery classifications. The vocabulary mirrors the effort
// policy so an owner observation maps to exactly one recovery behavior.
const (
	ClassContextCapacity     = "context_capacity"
	ClassSubscriptionSession = "subscription_session"
	ClassSubscriptionWeekly  = "subscription_weekly"
	ClassRateLimit           = "rate_limit"
	ClassProviderOverload    = "provider_overload"
	ClassAPICreditExhausted  = "api_credit_exhausted"
	ClassSpendLimit          = "spend_limit"
	ClassAuthPolicyDenied    = "auth_or_policy_denied"
	ClassDispatchUncertain   = "dispatch_uncertain"
	ClassRecovered           = "recovered"
)

var classVocabulary = map[string]struct{}{
	ClassContextCapacity:     {},
	ClassSubscriptionSession: {},
	ClassSubscriptionWeekly:  {},
	ClassRateLimit:           {},
	ClassProviderOverload:    {},
	ClassAPICreditExhausted:  {},
	ClassSpendLimit:          {},
	ClassAuthPolicyDenied:    {},
	ClassDispatchUncertain:   {},
	ClassRecovered:           {},
}

// blockingClasses pause new pool dispatch. Context capacity is a per-session
// condition and dispatch uncertainty is a per-operation condition, so neither
// pauses the whole pool.
var blockingClasses = map[string]struct{}{
	ClassSubscriptionSession: {},
	ClassSubscriptionWeekly:  {},
	ClassRateLimit:           {},
	ClassProviderOverload:    {},
	ClassAPICreditExhausted:  {},
	ClassSpendLimit:          {},
	ClassAuthPolicyDenied:    {},
}

// Reservation lifecycle states.
const (
	ReservationHeld    = "held"
	ReservationSettled = "settled"
)

// Sentinel errors. Callers map these to their own transport shapes; the package
// stays free of HTTP concerns.
var (
	ErrPoolMismatch        = errors.New("provider pool reference does not match")
	ErrUnknownClass        = errors.New("unknown runner-limit class")
	ErrPoolBlocked         = errors.New("provider pool is paused pending recovery")
	ErrPoolSaturated       = errors.New("provider pool concurrency is saturated")
	ErrAllowanceExhausted  = errors.New("shared provider allowance is exhausted")
	ErrUnknownReservation  = errors.New("provider reservation does not exist")
	ErrReservationConflict = errors.New("provider reservation identity is reused")
	ErrReservationActive   = errors.New("provider reservation is still unresolved")
)

// BlocksPool reports whether an observation class pauses new pool dispatch.
func BlocksPool(class string) bool {
	_, ok := blockingClasses[class]
	return ok
}

// KnownClass reports whether a class is part of the recovery vocabulary.
func KnownClass(class string) bool {
	_, ok := classVocabulary[class]
	return ok
}

// Observation is one owner-sourced runner-limit or recovery event with its
// provenance. ResetAt is the observed reset time when the owner supplied one;
// empty means unknown and requires a later explicit recovery observation.
type Observation struct {
	Pool          string   `json:"pool"`
	Class         string   `json:"class"`
	Runner        string   `json:"runner,omitempty"`
	Provider      string   `json:"provider,omitempty"`
	Model         string   `json:"model,omitempty"`
	Window        string   `json:"window,omitempty"`
	RetryAfter    string   `json:"retry_after,omitempty"`
	ResetAt       string   `json:"reset_at,omitempty"`
	ObservedAt    string   `json:"observed_at"`
	Source        string   `json:"source,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	SessionRef    string   `json:"session_ref,omitempty"`
	CheckpointRef string   `json:"checkpoint_ref,omitempty"`
	Uncertain     bool     `json:"uncertain,omitempty"`
	Detail        string   `json:"detail,omitempty"`
}

// State is the derived, durable admission state for one pool. Eligible is the
// one owner wake eligibility condition: a known reset is a time condition; an
// unknown reset requires a new owner/account observation.
type State struct {
	Pool                string `json:"pool"`
	Eligible            bool   `json:"eligible"`
	Blocked             bool   `json:"blocked"`
	Class               string `json:"class,omitempty"`
	Reason              string `json:"reason,omitempty"`
	ResetAt             string `json:"reset_at,omitempty"`
	ResetKnown          bool   `json:"reset_known"`
	RequiresObservation bool   `json:"requires_observation"`
	ObservedAt          string `json:"observed_at,omitempty"`
	Source              string `json:"source,omitempty"`
	Runner              string `json:"runner,omitempty"`
	Provider            string `json:"provider,omitempty"`
	Model               string `json:"model,omitempty"`
	// Limit is the pool's persisted effective capacity. It lets one read show
	// the admission bound currently in force, including an amendment refresh
	// that happened before the next reservation.
	Limit Limit `json:"limit"`
}

// Limit is the pool capacity derived from the approved aggregate effort grant.
// MaxConcurrent bounds simultaneous dispatches; MaxUnits bounds the shared
// allowance (zero means the aggregate grant left units unspecified). Unknown
// charges reserve at least UnknownChargeUnits.
type Limit struct {
	MaxConcurrent      int   `json:"max_concurrent"`
	MaxUnits           int64 `json:"max_units"`
	UnknownChargeUnits int64 `json:"unknown_charge_units,omitempty"`
}

// Reservation is one accounted dispatch against the shared pool allowance. It
// is charged at Reserve, before the owner is contacted. UsageUnresolved means
// the charge could not be measured and stays conservatively held.
type Reservation struct {
	AttemptID       string `json:"attempt_id"`
	EffortID        string `json:"effort_id,omitempty"`
	Pool            string `json:"pool"`
	Units           int64  `json:"units"`
	UsedUnits       int64  `json:"used_units"`
	Known           bool   `json:"known"`
	UsageUnresolved bool   `json:"usage_unresolved"`
	State           string `json:"state"`
	At              string `json:"at"`
}

// Totals summarizes pool investment for one effort's read address.
type Totals struct {
	Held       int   `json:"held"`
	Settled    int   `json:"settled"`
	Unresolved int   `json:"unresolved"`
	HeldUnits  int64 `json:"held_units"`
	UsedUnits  int64 `json:"used_units"`
}

type event struct {
	Observation
	Cleared bool `json:"cleared,omitempty"`
}

type state struct {
	Pool         string        `json:"pool"`
	Limit        Limit         `json:"limit"`
	Observations []event       `json:"observations"`
	Reservations []Reservation `json:"reservations"`
}

// Pool is the durable, single-writer owner state for one shared pool.
// Mutations are serialized in-process and persisted atomically.
type Pool struct {
	pool string
	path string
	now  func() time.Time
	mu   sync.Mutex
}

// New roots a pool document for one pool reference at the given file path.
func New(pool, path string) *Pool {
	return &Pool{pool: strings.TrimSpace(pool), path: path, now: time.Now}
}

// NewForRoot roots a pool document under a directory, using a filesystem-safe
// document name derived from the pool reference.
func NewForRoot(root, pool string) (*Pool, error) {
	pool = strings.TrimSpace(pool)
	if pool == "" || strings.ContainsAny(pool, `/\`) {
		return nil, fmt.Errorf("pool reference must be a single path segment")
	}
	return New(pool, filepath.Join(root, sanitizePoolName(pool)+".json")), nil
}

// Pool returns the pool reference this document owns.
func (p *Pool) Pool() string { return p.pool }

// SetClock overrides the clock; tests use it for deterministic reset handling.
func (p *Pool) SetClock(now func() time.Time) { p.now = now }

// Observe records a typed runner-limit or recovery observation and returns the
// derived pool state.
func (p *Pool) Observe(obs Observation) (State, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	obs.Pool = strings.TrimSpace(obs.Pool)
	if obs.Pool != "" && obs.Pool != p.pool {
		return State{}, fmt.Errorf("%w: %q", ErrPoolMismatch, obs.Pool)
	}
	obs.Pool = p.pool
	obs.Class = strings.TrimSpace(obs.Class)
	if !KnownClass(obs.Class) {
		return State{}, fmt.Errorf("%w: %q", ErrUnknownClass, obs.Class)
	}
	now := p.now()
	if strings.TrimSpace(obs.ObservedAt) == "" {
		obs.ObservedAt = now.UTC().Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, obs.ObservedAt); err != nil {
		return State{}, fmt.Errorf("observed_at must be RFC3339")
	}
	if strings.TrimSpace(obs.ResetAt) != "" {
		if _, err := time.Parse(time.RFC3339, obs.ResetAt); err != nil {
			return State{}, fmt.Errorf("reset_at must be RFC3339")
		}
	}
	obs.Evidence = append([]string(nil), obs.Evidence...)

	current, err := p.load()
	if err != nil {
		return State{}, err
	}
	current.Observations = append(current.Observations, event{Observation: obs, Cleared: obs.Class == ClassRecovered})
	if err := p.save(current); err != nil {
		return State{}, err
	}
	return deriveState(p.pool, current.Observations, current.Limit, now), nil
}

// State derives the current admission state without recording an observation.
func (p *Pool) State() (State, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, err := p.load()
	if err != nil {
		return State{}, err
	}
	return deriveState(p.pool, current.Observations, current.Limit, p.now()), nil
}

// Eligibility reports the current admission state. It is an alias for State so
// an owner wake condition reads naturally.
func (p *Pool) Eligibility() (State, error) { return p.State() }

// SetLimit persists the pool's effective capacity without recording a
// reservation. It is how an effort amendment refreshes a pool whose approved
// aggregate grant changed, so a read or a later reservation observes the
// amended bound instead of a stale one. A repeated set is idempotent, and a
// negative bound is refused rather than silently widening or narrowing
// admission.
func (p *Pool) SetLimit(limit Limit) (State, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if limit.MaxConcurrent < 0 || limit.MaxUnits < 0 || limit.UnknownChargeUnits < 0 {
		return State{}, fmt.Errorf("provider pool limit must not be negative")
	}
	current, err := p.load()
	if err != nil {
		return State{}, err
	}
	current.Limit = limit
	if err := p.save(current); err != nil {
		return State{}, err
	}
	return deriveState(p.pool, current.Observations, current.Limit, p.now()), nil
}

// Limit returns the persisted effective capacity. A zero dimension means the
// pool has no bound there, which is why zero fields are read as "unspecified"
// rather than as an unlimited grant by callers that derive defaults.
func (p *Pool) Limit() (Limit, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, err := p.load()
	if err != nil {
		return Limit{}, err
	}
	return current.Limit, nil
}

// AttributedTo reports whether every effort-attributed reservation on this pool
// belongs to effortID and at least one does. It anchors an amendment refresh to
// the effort that actually used the pool: a pool shared with a different effort
// is not attributable, so an amendment cannot overwrite another effort's bound.
func (p *Pool) AttributedTo(effortID string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	effortID = strings.TrimSpace(effortID)
	if effortID == "" {
		return false, nil
	}
	current, err := p.load()
	if err != nil {
		return false, err
	}
	found := false
	for _, r := range current.Reservations {
		ref := strings.TrimSpace(r.EffortID)
		if ref == "" {
			continue
		}
		if ref != effortID {
			return false, nil
		}
		found = true
	}
	return found, nil
}

// ListForRoot returns every persisted pool document under a root directory,
// sorted by pool reference. It lets an owner refresh the bounds derived from an
// amended grant without knowing the pool names in advance. A missing directory
// is an empty set, not an error, so a first amendment before any dispatch is a
// no-op rather than a failure.
func ListForRoot(root string) ([]*Pool, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list provider pools: %w", err)
	}
	pools := make([]*Pool, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		ref := strings.TrimSuffix(entry.Name(), ".json")
		if raw, err := os.ReadFile(path); err == nil {
			var meta struct {
				Pool string `json:"pool"`
			}
			if json.Unmarshal(raw, &meta) == nil && strings.TrimSpace(meta.Pool) != "" {
				ref = strings.TrimSpace(meta.Pool)
			}
		}
		if ref == "" || ref == "." || ref == ".." {
			continue
		}
		pools = append(pools, New(ref, path))
	}
	sort.Slice(pools, func(i, j int) bool { return pools[i].pool < pools[j].pool })
	return pools, nil
}

// Reserve accounts one dispatch against the pool before the owner is contacted.
// A saturated concurrency bound or exhausted shared allowance refuses admission;
// a blocked pool refuses until its reset condition is met. A repeated reservation
// of the same attempt identity with the same work is idempotent.
func (p *Pool) Reserve(limit Limit, r Reservation) (Reservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, err := p.load()
	if err != nil {
		return Reservation{}, err
	}

	r.AttemptID = strings.TrimSpace(r.AttemptID)
	if r.AttemptID == "" {
		return Reservation{}, fmt.Errorf("attempt_id is required")
	}
	if r.Pool != "" && strings.TrimSpace(r.Pool) != p.pool {
		return Reservation{}, fmt.Errorf("%w: %q", ErrPoolMismatch, r.Pool)
	}
	if r.Units < 0 {
		return Reservation{}, fmt.Errorf("units must not be negative")
	}
	if state := deriveState(p.pool, current.Observations, current.Limit, p.now()); !state.Eligible {
		return Reservation{}, fmt.Errorf("%w: %s", ErrPoolBlocked, state.Class)
	}

	for i := range current.Reservations {
		if current.Reservations[i].AttemptID != r.AttemptID {
			continue
		}
		existing := current.Reservations[i]
		if existing.State == ReservationHeld && existing.Units == r.Units && existing.EffortID == r.EffortID {
			return existing, nil
		}
		return Reservation{}, fmt.Errorf("%w: %q", ErrReservationConflict, r.AttemptID)
	}

	heldCount, heldUnits := activeUsage(current.Reservations, limit)
	if limit.MaxConcurrent > 0 && heldCount+1 > limit.MaxConcurrent {
		return Reservation{}, fmt.Errorf("%w: %d of %d", ErrPoolSaturated, heldCount, limit.MaxConcurrent)
	}
	charge := r.Units
	if !r.Known {
		charge = unknownCharge(limit, r.Units)
	}
	if limit.MaxUnits > 0 && heldUnits+charge > limit.MaxUnits {
		return Reservation{}, fmt.Errorf("%w: %d of %d units", ErrAllowanceExhausted, heldUnits, limit.MaxUnits)
	}

	r.Pool = p.pool
	r.State = ReservationHeld
	r.UsageUnresolved = !r.Known
	r.At = p.now().UTC().Format(time.RFC3339)
	current.Limit = limit
	current.Reservations = append(current.Reservations, r)
	if err := p.save(current); err != nil {
		return Reservation{}, err
	}
	return r, nil
}

// Settle records terminal usage for a reservation. A known measurement releases
// the held charge; an unknown measurement keeps the reservation conservatively
// held until a later known settle resolves it, so unknown usage never reads as
// zero. A repeated settle is idempotent.
func (p *Pool) Settle(attemptID string, used int64, known bool) (Reservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	if used < 0 {
		return Reservation{}, fmt.Errorf("used units must not be negative")
	}
	current, err := p.load()
	if err != nil {
		return Reservation{}, err
	}
	for i := range current.Reservations {
		r := &current.Reservations[i]
		if r.AttemptID != attemptID {
			continue
		}
		if r.State == ReservationSettled {
			if r.UsageUnresolved && known {
				r.UsageUnresolved = false
				r.Known = true
				r.UsedUnits = used
				if err := p.save(current); err != nil {
					return Reservation{}, err
				}
			}
			return *r, nil
		}
		if !known {
			r.UsageUnresolved = true
			if err := p.save(current); err != nil {
				return Reservation{}, err
			}
			return *r, nil
		}
		r.Known = true
		r.UsageUnresolved = false
		r.UsedUnits = used
		r.State = ReservationSettled
		if err := p.save(current); err != nil {
			return Reservation{}, err
		}
		return *r, nil
	}
	return Reservation{}, fmt.Errorf("%w: %q", ErrUnknownReservation, attemptID)
}

// Release finalizes a reservation whose effect never started. It refuses to
// release an unresolved charge, because unknown usage must stay reserved.
func (p *Pool) Release(attemptID string) (Reservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	current, err := p.load()
	if err != nil {
		return Reservation{}, err
	}
	for i := range current.Reservations {
		r := &current.Reservations[i]
		if r.AttemptID != attemptID {
			continue
		}
		if r.State == ReservationSettled {
			return *r, nil
		}
		if r.UsageUnresolved {
			return Reservation{}, fmt.Errorf("%w: %q", ErrReservationActive, attemptID)
		}
		r.Known = true
		r.UsedUnits = 0
		r.State = ReservationSettled
		if err := p.save(current); err != nil {
			return Reservation{}, err
		}
		return *r, nil
	}
	return Reservation{}, fmt.Errorf("%w: %q", ErrUnknownReservation, attemptID)
}

// Reservations returns every reservation for this pool, sorted by attempt
// identity, so a caller can reconcile outstanding dispatches.
func (p *Pool) Reservations() ([]Reservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, err := p.load()
	if err != nil {
		return nil, err
	}
	out := append([]Reservation(nil), current.Reservations...)
	sort.Slice(out, func(i, j int) bool { return out[i].AttemptID < out[j].AttemptID })
	return out, nil
}

// Totals summarizes held and settled investment for the pool.
func (p *Pool) Totals() (Totals, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	current, err := p.load()
	if err != nil {
		return Totals{}, err
	}
	result := Totals{}
	for _, r := range current.Reservations {
		switch r.State {
		case ReservationHeld:
			result.Held++
			result.HeldUnits += r.Units
			if r.UsageUnresolved {
				result.Unresolved++
			}
		case ReservationSettled:
			result.Settled++
			result.UsedUnits += r.UsedUnits
		}
	}
	return result, nil
}

func deriveState(pool string, observations []event, limit Limit, now time.Time) State {
	state := State{Pool: pool, Eligible: true, Limit: limit}
	for _, ev := range observations {
		obs := ev.Observation
		if ev.Cleared || obs.Class == ClassRecovered {
			state = State{Pool: pool, Eligible: true, ObservedAt: obs.ObservedAt, Source: obs.Source, Limit: limit}
			continue
		}
		if !BlocksPool(obs.Class) {
			continue
		}
		if obs.ResetAt != "" {
			if reset, err := time.Parse(time.RFC3339, obs.ResetAt); err == nil && !now.Before(reset) {
				continue
			}
		}
		state.Eligible = false
		state.Blocked = true
		state.Class = obs.Class
		state.Reason = strings.TrimSpace(obs.Detail)
		if state.Reason == "" {
			state.Reason = obs.Class
		}
		state.ResetAt = obs.ResetAt
		state.ResetKnown = obs.ResetAt != ""
		state.RequiresObservation = obs.ResetAt == ""
		state.ObservedAt = obs.ObservedAt
		state.Source = obs.Source
		state.Runner = obs.Runner
		state.Provider = obs.Provider
		state.Model = obs.Model
	}
	return state
}

func activeUsage(reservations []Reservation, limit Limit) (int, int64) {
	count := 0
	var units int64
	for _, r := range reservations {
		if r.State != ReservationHeld {
			continue
		}
		count++
		if r.Known {
			units += r.Units
		} else {
			units += unknownCharge(limit, r.Units)
		}
	}
	return count, units
}

func unknownCharge(limit Limit, units int64) int64 {
	if units > 0 {
		return units
	}
	if limit.UnknownChargeUnits > 0 {
		return limit.UnknownChargeUnits
	}
	return 1
}

func (p *Pool) load() (state, error) {
	raw, err := os.ReadFile(p.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state{Pool: p.pool}, nil
		}
		return state{}, fmt.Errorf("read provider pool: %w", err)
	}
	var current state
	if err := json.Unmarshal(raw, &current); err != nil {
		return state{}, fmt.Errorf("decode provider pool: %w", err)
	}
	if current.Pool == "" {
		current.Pool = p.pool
	}
	if current.Pool != p.pool {
		return state{}, fmt.Errorf("provider pool mismatch: document holds %q, want %q", current.Pool, p.pool)
	}
	return current, nil
}

func (p *Pool) save(current state) error {
	current.Pool = p.pool
	if err := os.MkdirAll(filepath.Dir(p.path), 0o700); err != nil {
		return fmt.Errorf("create provider pool dir: %w", err)
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal provider pool: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(p.path), ".provider-pool-*.tmp")
	if err != nil {
		return fmt.Errorf("create provider pool temp: %w", err)
	}
	tempName := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, p.path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

func sanitizePoolName(pool string) string {
	var builder strings.Builder
	for _, r := range pool {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	name := builder.String()
	if name == "" || name == "." || name == ".." {
		return "pool"
	}
	return name
}
