// Package descendantbudget owns durable, effort-wide descendant capacity
// accounting for a finite effort.
//
// One document per effort is shared by every branch, so concurrent or nested
// planners cannot each mint a private depth, active-descendant or premium
// allowance. The caps themselves come from the effort's approved aggregate
// grant; this package only accounts reservations against them.
//
// A reservation is charged before the descendant is dispatched, so an
// interrupted or lost start still consumes the shared allowance until it is
// explicitly released. Reservations are idempotent by attempt identity, so a
// replayed admission does not consume a second slot.
package descendantbudget

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

// Reservation lifecycle states.
const (
	ReservationActive = "active"
	ReservationEnded  = "ended"
)

// Sentinel errors. Callers map these to their own transport shapes; the
// package stays free of HTTP concerns.
var (
	ErrCapacityExhausted   = errors.New("effort-wide descendant allowance is exhausted")
	ErrDepthExceeded       = errors.New("descendant depth exceeds the approved maximum")
	ErrUnknownReservation  = errors.New("descendant reservation does not exist")
	ErrReservationConflict = errors.New("descendant reservation identity is reused")
	ErrParentNotActive     = errors.New("descendant parent is not active")
)

// Limit is the descendant capacity derived from the approved aggregate effort
// grant. MaxDepth is the maximum descendant depth below the root (root depth
// zero); MaxActiveDescendants bounds simultaneous descendants across every
// branch; MaxPremiumDescendants bounds simultaneous premium-profile
// descendants.
type Limit struct {
	MaxDepth              int `json:"max_depth"`
	MaxActiveDescendants  int `json:"max_active_descendants"`
	MaxPremiumDescendants int `json:"max_premium_descendants"`
}

// Reservation is one admitted descendant branch. Depth one is a direct child
// of the root; a reservation with ParentAttemptID inherits exactly one level
// below its active parent. Premium marks a descendant that draws on the
// scarce premium-profile allowance.
type Reservation struct {
	AttemptID       string `json:"attempt_id"`
	EffortID        string `json:"effort_id,omitempty"`
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	Depth           int    `json:"depth"`
	Premium         bool   `json:"premium"`
	State           string `json:"state"`
	At              string `json:"at"`
}

// Totals summarizes active capacity and the applied cap so one read reconciles
// the effective allowance.
type Totals struct {
	Active  int   `json:"active"`
	Ended   int   `json:"ended"`
	Premium int   `json:"premium_active"`
	Limit   Limit `json:"limit"`
}

type state struct {
	EffortID     string        `json:"effort_id"`
	Limit        Limit         `json:"limit"`
	Reservations []Reservation `json:"reservations"`
}

// Budget is the durable, single-writer owner state for one effort's descendant
// allowance. Mutations are serialized in-process and persisted atomically.
type Budget struct {
	effortID string
	path     string
	now      func() time.Time
	mu       sync.Mutex
}

// New roots a budget document for one effort at the given file path.
func New(effortID, path string) *Budget {
	return &Budget{effortID: strings.TrimSpace(effortID), path: path, now: time.Now}
}

// NewForRoot roots a budget document under a directory, using a
// filesystem-safe document name derived from the effort identity.
func NewForRoot(root, effortID string) (*Budget, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || effortID == "." || effortID == ".." || strings.ContainsAny(effortID, `/\`) {
		return nil, fmt.Errorf("effort_id must be a single path segment")
	}
	return New(effortID, filepath.Join(root, sanitizeEffortName(effortID)+".json")), nil
}

// EffortID returns the effort identity this budget owns.
func (b *Budget) EffortID() string { return b.effortID }

// SetClock overrides the clock; tests use it for deterministic timestamps.
func (b *Budget) SetClock(now func() time.Time) { b.now = now }

// Admit charges one descendant slot against the shared allowance before the
// descendant is dispatched. It refuses when the depth exceeds the approved
// maximum, the active-descendant cap is reached, or the premium cap is
// exhausted. A repeated admission of the same attempt identity with the same
// parent and premium flag is idempotent.
func (b *Budget) Admit(limit Limit, r Reservation) (Reservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	current, err := b.load()
	if err != nil {
		return Reservation{}, err
	}

	r.AttemptID = strings.TrimSpace(r.AttemptID)
	if r.AttemptID == "" {
		return Reservation{}, fmt.Errorf("attempt_id is required")
	}
	if r.EffortID != "" && strings.TrimSpace(r.EffortID) != b.effortID {
		return Reservation{}, fmt.Errorf("descendant effort reference does not match: %q", r.EffortID)
	}
	r.EffortID = b.effortID
	r.ParentAttemptID = strings.TrimSpace(r.ParentAttemptID)

	depth, err := resolveDepth(current.Reservations, r)
	if err != nil {
		return Reservation{}, err
	}

	for i := range current.Reservations {
		existing := current.Reservations[i]
		if existing.AttemptID != r.AttemptID {
			continue
		}
		if existing.State == ReservationActive && existing.Depth == depth &&
			existing.Premium == r.Premium && existing.ParentAttemptID == r.ParentAttemptID {
			return existing, nil
		}
		return Reservation{}, fmt.Errorf("%w: %q", ErrReservationConflict, r.AttemptID)
	}

	if depth < 1 {
		return Reservation{}, fmt.Errorf("descendant depth must be at least 1")
	}
	if limit.MaxDepth > 0 && depth > limit.MaxDepth {
		return Reservation{}, fmt.Errorf("%w: depth %d exceeds %d", ErrDepthExceeded, depth, limit.MaxDepth)
	}
	active, premium := activeCounts(current.Reservations)
	if limit.MaxActiveDescendants > 0 && active >= limit.MaxActiveDescendants {
		return Reservation{}, fmt.Errorf("%w: %d of %d active descendants", ErrCapacityExhausted, active, limit.MaxActiveDescendants)
	}
	if r.Premium {
		if limit.MaxPremiumDescendants <= 0 {
			return Reservation{}, fmt.Errorf("%w: no premium descendant is authorized", ErrCapacityExhausted)
		}
		if premium >= limit.MaxPremiumDescendants {
			return Reservation{}, fmt.Errorf("%w: %d of %d premium descendants", ErrCapacityExhausted, premium, limit.MaxPremiumDescendants)
		}
	}

	r.Depth = depth
	r.State = ReservationActive
	r.At = b.now().UTC().Format(time.RFC3339)
	current.Limit = limit
	current.Reservations = append(current.Reservations, r)
	if err := b.save(current); err != nil {
		return Reservation{}, err
	}
	return r, nil
}

// Release ends an active reservation. The cap it consumed becomes available
// again. A repeated release is idempotent.
func (b *Budget) Release(attemptID string) (Reservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	attemptID = strings.TrimSpace(attemptID)
	current, err := b.load()
	if err != nil {
		return Reservation{}, err
	}
	for i := range current.Reservations {
		r := &current.Reservations[i]
		if r.AttemptID != attemptID {
			continue
		}
		if r.State == ReservationEnded {
			return *r, nil
		}
		r.State = ReservationEnded
		if err := b.save(current); err != nil {
			return Reservation{}, err
		}
		return *r, nil
	}
	return Reservation{}, fmt.Errorf("%w: %q", ErrUnknownReservation, attemptID)
}

// Reservations returns every reservation for this effort, sorted by attempt
// identity, so a caller can reconcile outstanding descendants.
func (b *Budget) Reservations() ([]Reservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	current, err := b.load()
	if err != nil {
		return nil, err
	}
	out := append([]Reservation(nil), current.Reservations...)
	sort.Slice(out, func(i, j int) bool { return out[i].AttemptID < out[j].AttemptID })
	return out, nil
}

// Totals summarizes active and ended reservations plus the applied cap.
func (b *Budget) Totals() (Totals, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	current, err := b.load()
	if err != nil {
		return Totals{}, err
	}
	result := Totals{Limit: current.Limit}
	for _, r := range current.Reservations {
		switch r.State {
		case ReservationActive:
			result.Active++
			if r.Premium {
				result.Premium++
			}
		default:
			result.Ended++
		}
	}
	return result, nil
}

func resolveDepth(reservations []Reservation, r Reservation) (int, error) {
	if r.ParentAttemptID == "" {
		return r.Depth, nil
	}
	for i := range reservations {
		parent := reservations[i]
		if parent.AttemptID != r.ParentAttemptID {
			continue
		}
		if parent.State != ReservationActive {
			return 0, fmt.Errorf("%w: %q", ErrParentNotActive, r.ParentAttemptID)
		}
		if r.Depth != parent.Depth+1 {
			return 0, fmt.Errorf("descendant depth %d must be one below active parent depth %d", r.Depth, parent.Depth)
		}
		return parent.Depth + 1, nil
	}
	return 0, fmt.Errorf("%w: %q", ErrParentNotActive, r.ParentAttemptID)
}

func activeCounts(reservations []Reservation) (int, int) {
	active, premium := 0, 0
	for _, r := range reservations {
		if r.State != ReservationActive {
			continue
		}
		active++
		if r.Premium {
			premium++
		}
	}
	return active, premium
}

func (b *Budget) load() (state, error) {
	raw, err := os.ReadFile(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state{EffortID: b.effortID}, nil
		}
		return state{}, fmt.Errorf("read descendant budget: %w", err)
	}
	var current state
	if err := json.Unmarshal(raw, &current); err != nil {
		return state{}, fmt.Errorf("decode descendant budget: %w", err)
	}
	if current.EffortID == "" {
		current.EffortID = b.effortID
	}
	if current.EffortID != b.effortID {
		return state{}, fmt.Errorf("descendant budget mismatch: document holds %q, want %q", current.EffortID, b.effortID)
	}
	return current, nil
}

func (b *Budget) save(current state) error {
	current.EffortID = b.effortID
	if err := os.MkdirAll(filepath.Dir(b.path), 0o700); err != nil {
		return fmt.Errorf("create descendant budget dir: %w", err)
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal descendant budget: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(b.path), ".descendant-budget-*.tmp")
	if err != nil {
		return fmt.Errorf("create descendant budget temp: %w", err)
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
	if err := os.Rename(tempName, b.path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

func sanitizeEffortName(effortID string) string {
	var builder strings.Builder
	for _, r := range effortID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('_')
		}
	}
	name := builder.String()
	if name == "" || name == "." || name == ".." {
		return "effort"
	}
	return name
}
