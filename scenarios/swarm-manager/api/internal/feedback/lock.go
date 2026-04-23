package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock is the on-disk single-agent-per-initiative mutex.
//
// Only one active agent run — feedback or review — may hold the lock at a
// time. The lock is a JSON file (`.feedback-lock`) in the initiative folder
// storing the holder's metadata so UIs and CLI tools can surface "why is
// this initiative locked" rather than a bare boolean.
//
// Acquire is best-effort under stale locks: if the existing lock is older
// than MaxAge, it is considered abandoned and overwritten. Callers can
// force-acquire via AcquireOverride regardless of age — this is how the
// "override warning" UI flow works, where the user sees the current holder
// and consciously decides to preempt.
type Lock struct {
	// Dir returns the absolute initiative folder the lock file lives under.
	Dir func(initiativeName string) string
	// Clock is an optional time source; defaults to time.Now.
	Clock func() time.Time
	// MaxAge is the staleness threshold; defaults to DefaultLockMaxAge
	// when zero. Tests set a short MaxAge to exercise the stale-lock
	// sweep deterministically.
	MaxAge time.Duration
}

// Holder describes the run currently holding the lock.
type Holder struct {
	RunID          string `json:"run_id"`
	Purpose        string `json:"purpose"` // feedback | review | feedback_continue
	RoundNumber    int    `json:"round_number,omitempty"`
	AcquiredAt     string `json:"acquired_at"`
	AcquiredBy     string `json:"acquired_by,omitempty"`
	InitiativeName string `json:"initiative_name"`
}

// ErrLocked signals that Acquire found a live (non-stale) lock and the
// caller did not pass `override`. The returned error wraps LockConflict so
// callers can type-assert to render the current holder in UI.
var ErrLocked = errors.New("initiative feedback is locked by another agent")

// LockConflict carries the holder metadata for UI rendering.
type LockConflict struct {
	Holder Holder
}

func (c *LockConflict) Error() string {
	if c.Holder.RunID == "" {
		return ErrLocked.Error()
	}
	return fmt.Sprintf("%s (purpose=%s run=%s acquired=%s)",
		ErrLocked.Error(), c.Holder.Purpose, c.Holder.RunID, c.Holder.AcquiredAt)
}

func (c *LockConflict) Unwrap() error { return ErrLocked }

const (
	lockFileName = ".feedback-lock"

	// DefaultLockMaxAge is the staleness threshold used when
	// Lock.MaxAge is unset. Chosen to match agent-manager's max run
	// duration so a crash-stranded lock is swept by the next boot
	// sweep within roughly one run's worth of time.
	DefaultLockMaxAge = 2 * time.Hour
)

func (l *Lock) path(initiativeName string) string {
	return filepath.Join(l.Dir(initiativeName), lockFileName)
}

func (l *Lock) now() time.Time {
	if l.Clock != nil {
		return l.Clock().UTC()
	}
	return time.Now().UTC()
}

func (l *Lock) maxAge() time.Duration {
	if l.MaxAge > 0 {
		return l.MaxAge
	}
	return DefaultLockMaxAge
}

// Acquire takes the lock for `holder`. Returns *LockConflict if a live lock
// is already held. Callers wanting to preempt a live holder should call
// AcquireOverride instead.
func (l *Lock) Acquire(initiativeName string, holder Holder) error {
	return l.acquire(initiativeName, holder, false)
}

// AcquireOverride takes the lock unconditionally. Used when the user has
// confirmed they want to preempt the current holder via the override
// dialog. The caller is responsible for having shown the user the current
// holder (via Inspect) first.
func (l *Lock) AcquireOverride(initiativeName string, holder Holder) error {
	return l.acquire(initiativeName, holder, true)
}

func (l *Lock) acquire(initiativeName string, holder Holder, override bool) error {
	if holder.RunID == "" {
		return fmt.Errorf("holder.RunID is required")
	}
	if holder.Purpose == "" {
		return fmt.Errorf("holder.Purpose is required")
	}
	holder.InitiativeName = initiativeName
	holder.AcquiredAt = l.now().Format(time.RFC3339)

	if !override {
		existing, ok, err := l.inspect(initiativeName)
		if err != nil {
			return err
		}
		if ok && !l.isStale(existing) {
			return &LockConflict{Holder: existing}
		}
	}
	return l.write(initiativeName, holder)
}

// Release drops the lock if `runID` matches the current holder. Mismatched
// releases are treated as idempotent no-ops (the holder was already
// overridden or released elsewhere) — returns nil.
func (l *Lock) Release(initiativeName, runID string) error {
	existing, ok, err := l.inspect(initiativeName)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if existing.RunID != runID {
		return nil
	}
	if err := os.Remove(l.path(initiativeName)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

// Inspect returns the current holder if the lock is present and not stale.
// (nil, nil) means no live lock. Used by the override-warning UI path.
func (l *Lock) Inspect(initiativeName string) (*Holder, error) {
	h, ok, err := l.inspect(initiativeName)
	if err != nil {
		return nil, err
	}
	if !ok || l.isStale(h) {
		return nil, nil
	}
	return &h, nil
}

// SweepStale removes the lock if it is older than MaxAge. Returns whether
// a lock was swept. Safe to call on startup.
func (l *Lock) SweepStale(initiativeName string) (bool, error) {
	h, ok, err := l.inspect(initiativeName)
	if err != nil {
		return false, err
	}
	if !ok || !l.isStale(h) {
		return false, nil
	}
	if err := os.Remove(l.path(initiativeName)); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func (l *Lock) inspect(initiativeName string) (Holder, bool, error) {
	data, err := os.ReadFile(l.path(initiativeName))
	if err != nil {
		if os.IsNotExist(err) {
			return Holder{}, false, nil
		}
		return Holder{}, false, fmt.Errorf("read lock: %w", err)
	}
	var h Holder
	if err := json.Unmarshal(data, &h); err != nil {
		return Holder{}, false, fmt.Errorf("parse lock: %w", err)
	}
	return h, true, nil
}

func (l *Lock) isStale(h Holder) bool {
	if h.AcquiredAt == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, h.AcquiredAt)
	if err != nil {
		return true
	}
	return l.now().Sub(t.UTC()) > l.maxAge()
}

func (l *Lock) write(initiativeName string, h Holder) error {
	dir := l.Dir(initiativeName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lock: %w", err)
	}
	tmp := l.path(initiativeName) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write lock: %w", err)
	}
	if err := os.Rename(tmp, l.path(initiativeName)); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename lock: %w", err)
	}
	return nil
}
