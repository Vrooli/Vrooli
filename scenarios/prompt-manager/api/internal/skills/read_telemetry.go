// Skill-read telemetry: who read which skill, and whether discovery is what
// put it in front of them.
//
// Why this exists: `UsageCount` was incremented only by `prompt-manager skill
// use`, a verb nothing in the system called, so every skill reported zero reads
// and skill-optimizer's "high usage" selection rung was unpowered. Recording at
// the read handler counts every consumer — CLI, UI, MCP, other scenarios —
// exactly once per resolved skill.
//
// The discovery join is the part that carries the signal. Discovery already
// records what it *returned*; this records what was actually *read*. A skill
// returned often and read rarely is not popular — it is a search-precision
// defect, and it costs discovery budget every time it is offered.
//
// DOC: docs/agent-system/FRAMEWORK_HEALTH.md § Three reachability classes
package skills

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"prompt-manager/internal/store"
)

// discoveryLookback bounds how far back a discover call may sit and still be
// credited with a read. Long enough for an agent to read a returned skill
// several turns later; short enough that an unrelated later read is not
// attributed to a stale call.
const discoveryLookback = 2 * time.Hour

// discoverySnapshotTTL caps how often the recorder re-reads the discovery log.
// Reads are far more frequent than discover calls, so re-parsing the log per
// read would put a JSONL scan on a hot path for data that changes slowly.
const discoverySnapshotTTL = 15 * time.Second

// usageRecorder is the existing per-skill counter surface. Kept as an interface
// so the recorder can be built in tests without the metrics repository.
type usageRecorder interface {
	RecordUsage(skillID string) (int, time.Time, error)
}

// ReadRecorder records one SkillRead per resolved skill and joins it against
// recent discovery calls by the same caller.
type ReadRecorder struct {
	reads *store.SkillReadStore
	calls *store.DiscoveryCallStore
	usage usageRecorder
	now   func() time.Time

	mu           sync.Mutex
	snapshot     []store.DiscoveryCall
	snapshotAt   time.Time
	snapshotOnce bool
}

// NewReadRecorder builds a recorder. Any dependency may be nil: a nil store or
// counter disables that half of the recording rather than failing the read.
// Telemetry must never be the reason a skill fails to be served.
func NewReadRecorder(reads *store.SkillReadStore, calls *store.DiscoveryCallStore, usage usageRecorder) *ReadRecorder {
	return &ReadRecorder{reads: reads, calls: calls, usage: usage, now: time.Now}
}

// Record attributes and persists one read per skill id. Errors are swallowed by
// design — see NewReadRecorder.
func (rr *ReadRecorder) Record(r *http.Request, skillIDs []string) {
	rr.RecordWithRunID(r, skillIDs, "")
}

// RecordWithRunID records a read with a run ID obtained from server-side
// identity verification. The transport token is intentionally not decoded in
// this recorder; the handler owns verification and passes only trusted claims.
func (rr *ReadRecorder) RecordWithRunID(r *http.Request, skillIDs []string, runID string) {
	if rr == nil || len(skillIDs) == 0 {
		return
	}
	attribution := store.CallerFromRequest(r)
	if attribution.Kind == "" && strings.TrimSpace(runID) != "" {
		attribution.Kind = "agent-member"
	}
	at := rr.now().UTC().Format(time.RFC3339)

	for _, skillID := range skillIDs {
		if skillID == "" {
			continue
		}
		if rr.usage != nil {
			_, _, _ = rr.usage.RecordUsage(skillID)
		}
		if rr.reads == nil {
			continue
		}
		agentRunID := attribution.RunID
		if strings.TrimSpace(runID) != "" {
			agentRunID = strings.TrimSpace(runID)
		}
		entry := store.SkillRead{
			At:         at,
			SkillID:    skillID,
			Caller:     attribution.Caller,
			CallerKind: attribution.Kind,
			AgentRunID: agentRunID,
		}
		if callID, ok := rr.discoverySource(attribution.Caller, skillID); ok {
			entry.ViaDiscovery = true
			entry.DiscoveryCallID = callID
		}
		_ = rr.reads.Append(entry)
	}
}

// discoverySource reports the most recent discover call by the same caller that
// returned this skill inside the lookback window.
//
// An empty caller never matches. Without attribution there is no way to tell
// one agent's discover call from another's, and crediting a read to whichever
// call happened to be recent would manufacture conversions that did not happen.
func (rr *ReadRecorder) discoverySource(caller, skillID string) (string, bool) {
	if rr.calls == nil || caller == "" {
		return "", false
	}
	cutoff := rr.now().UTC().Add(-discoveryLookback)
	best := ""
	bestAt := time.Time{}
	for _, call := range rr.discoverySnapshot() {
		if call.Caller != caller {
			continue
		}
		at, err := time.Parse(time.RFC3339, call.At)
		if err != nil || at.Before(cutoff) {
			continue
		}
		if !at.After(bestAt) {
			continue
		}
		for _, result := range call.Results {
			if result.ID == skillID {
				best = call.ID
				bestAt = at
				break
			}
		}
	}
	return best, best != ""
}

// discoverySnapshot returns the retained discovery calls, refreshing at most
// once per discoverySnapshotTTL.
//
// The read is deliberately unwindowed. Windowing here would apply the lookback
// against the store's clock while discoverySource applies it against this
// recorder's clock, so a call could be dropped before the only code that owns
// the attribution window ever saw it. One clock decides what is recent, and it
// is rr.now. The store bounds its own file by retention and entry count, so
// reading all of it stays cheap.
func (rr *ReadRecorder) discoverySnapshot() []store.DiscoveryCall {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	now := rr.now()
	if rr.snapshotOnce && now.Sub(rr.snapshotAt) < discoverySnapshotTTL {
		return rr.snapshot
	}
	calls, err := rr.calls.ReadSince(0)
	if err != nil {
		return rr.snapshot
	}
	rr.snapshot = calls
	rr.snapshotAt = now
	rr.snapshotOnce = true
	return rr.snapshot
}
