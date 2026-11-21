package providerreadiness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultRestartCooldown is how long a provider is left alone after a restart
// when the only thing that staled it again is churn outside its own tree.
//
// The case this exists for: an agent spends hours editing a widely shared
// package. Every suite run in that window finds most providers stale, restarts
// up to the cap, and finds them stale again on the next run because the shared
// package moved again. Nothing is wrong — the staleness is real each time — but
// rebuilding the same providers over and over buys almost nothing, since the
// churn is incidental to the judgment each provider is being asked for.
//
// The cooldown deliberately does NOT apply to a provider whose own code changed.
// That is the case where a stale result is actively misleading rather than
// merely dated, so it is always worth the restart.
const DefaultRestartCooldown = 30 * time.Minute

// restartLedger remembers when each provider was last restarted for staleness.
// It persists across runs because the Manager does not: the churn window this
// guards against spans many runs, so in-memory state would never fire.
type restartLedger struct {
	path string
	mu   sync.Mutex
}

type restartRecord struct {
	RestartedAt time.Time `json:"restartedAt"`
	Class       string    `json:"class"`
}

type restartLedgerFile struct {
	Providers map[string]restartRecord `json:"providers"`
}

func newRestartLedger(path string) *restartLedger {
	if path == "" {
		return nil
	}
	return &restartLedger{path: path}
}

func (l *restartLedger) load() restartLedgerFile {
	out := restartLedgerFile{Providers: map[string]restartRecord{}}
	if l == nil {
		return out
	}
	raw, err := os.ReadFile(l.path)
	if err != nil {
		return out
	}
	// A corrupt ledger must not block restarts; an empty ledger simply means
	// "no cooldown is in effect", which is the safe direction.
	if err := json.Unmarshal(raw, &out); err != nil || out.Providers == nil {
		return restartLedgerFile{Providers: map[string]restartRecord{}}
	}
	return out
}

// cooling reports whether provider was restarted recently enough that another
// restart should be skipped, given the class of change that staled it.
func (l *restartLedger) cooling(provider string, class StalenessClass, window time.Duration, now time.Time) (bool, time.Duration) {
	// A provider whose own code changed is never held back: that is the case
	// where running the old binary produces a misleading verdict.
	if class == StalenessOwnCode || class == StalenessRebuilt {
		return false, 0
	}
	if l == nil || window <= 0 {
		return false, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.load().Providers[provider]
	if !ok || rec.RestartedAt.IsZero() {
		return false, 0
	}
	elapsed := now.Sub(rec.RestartedAt)
	if elapsed >= window {
		return false, 0
	}
	return true, window - elapsed
}

func (l *restartLedger) record(provider string, class StalenessClass, now time.Time) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	state := l.load()
	state.Providers[provider] = restartRecord{RestartedAt: now, Class: string(class)}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	// A failed write only means the next run has no cooldown to consult, which
	// degrades to today's behavior rather than breaking anything.
	_ = os.WriteFile(l.path, raw, 0o644)
}

// NewRestartLedgerAt builds a cooldown ledger stored at path. Exported so the
// orchestrator can place it beside its other run state; an empty path yields a
// nil ledger, which disables the cooldown.
func NewRestartLedgerAt(path string) *restartLedger { return newRestartLedger(path) }
