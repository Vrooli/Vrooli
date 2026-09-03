package setpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
)

// Sustainer is the hysteresis every consumer shares: a breach becomes a
// finding only after it has been observed continuously for the bar's
// sustain, and clears the moment a reading is back under the bar. The first
// observation is stored so a one-shot process (the emergency watchdog) and
// a long-lived one (autoheal) count the window the same way.
type Sustainer struct {
	state SustainState
	now   func() time.Time
}

// SustainState stores the first observation of a named breach.
type SustainState interface {
	FirstObserved(name string) (time.Time, bool)
	Observe(name string, at time.Time) error
	Clear(name string) error
}

// NewSustainer builds a Sustainer over a state store.
func NewSustainer(state SustainState) *Sustainer {
	return &Sustainer{state: state, now: func() time.Time { return time.Now().UTC() }}
}

// WithClock replaces the clock; tests use it to move through the window.
func (s *Sustainer) WithClock(now func() time.Time) *Sustainer {
	s.now = now
	return s
}

// Breach records the observation and reports whether the named condition
// has been failing for at least sustain. A false reading clears the state.
// A zero sustain means the bar has no window and the first breach counts.
func (s *Sustainer) Breach(name string, failing bool, sustain time.Duration) bool {
	if !failing {
		_ = s.state.Clear(name)
		return false
	}
	now := s.now()
	first, ok := s.state.FirstObserved(name)
	if !ok || now.Before(first) {
		_ = s.state.Observe(name, now)
		return sustain <= 0
	}
	return now.Sub(first) >= sustain
}

// Since reports when the named breach was first observed.
func (s *Sustainer) Since(name string) (time.Time, bool) { return s.state.FirstObserved(name) }

// MemoryState keeps first observations in memory; long-lived processes use it.
type MemoryState struct {
	mu    sync.Mutex
	first map[string]time.Time
}

func NewMemoryState() *MemoryState { return &MemoryState{first: map[string]time.Time{}} }

func (m *MemoryState) FirstObserved(name string) (time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	at, ok := m.first[name]
	return at, ok
}

func (m *MemoryState) Observe(name string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.first[name] = at
	return nil
}

func (m *MemoryState) Clear(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.first, name)
	return nil
}

// FileState keeps first observations as one small JSON file per name under
// dir; a one-shot process reads the window back on its next run.
type FileState struct {
	Dir    string
	Prefix string
}

type fileObservation struct {
	FirstObserved time.Time `json:"first_observed"`
}

func (f FileState) path(name string) string { return filepath.Join(f.Dir, f.Prefix+name) }

func (f FileState) FirstObserved(name string) (time.Time, bool) {
	data, err := os.ReadFile(f.path(name))
	if err != nil {
		return time.Time{}, false
	}
	var obs fileObservation
	if json.Unmarshal(data, &obs) != nil || obs.FirstObserved.IsZero() {
		return time.Time{}, false
	}
	return obs.FirstObserved, true
}

func (f FileState) Observe(name string, at time.Time) error {
	if err := os.MkdirAll(f.Dir, tuning.PermPrivateDir); err != nil {
		return err
	}
	data, err := json.Marshal(fileObservation{FirstObserved: at})
	if err != nil {
		return err
	}
	return os.WriteFile(f.path(name), data, tuning.PermSecret)
}

func (f FileState) Clear(name string) error {
	err := os.Remove(f.path(name))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
