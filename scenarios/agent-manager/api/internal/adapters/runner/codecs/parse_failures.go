// Package codecs — parse_failures.go adds observability for the silent drops
// that used to make transcript format drift invisible. Historically a codec
// that could not parse a line returned (nil, nil) or swallowed a json.Unmarshal
// error, so a harness changing its wire format meant messages vanished from the
// conversation index with ZERO signal — the most insidious reliability gap in
// the search path (FINDINGS §A.1).
//
// This seam is OBSERVE-ONLY. Recording a parse failure never changes what a
// codec parses or emits; it only increments a counter and captures a bounded
// sample so `agent-manager conversation index status` can report that drift is
// happening. The counter is injected (ParseFailureCounter) with a process-wide
// default registry, so existing codec call sites and tests are unchanged.
package codecs

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// ParseFailureCounter records a single codec parse failure. Implementations
// MUST be safe for concurrent use: codecs decode many runs in parallel.
//
// The counter is deliberately side-effect-only — it returns nothing — so a
// recording call can never alter a codec's parse result. A nil counter is a
// no-op at the call site (see recordParseFailure), which is how codecs that
// want zero instrumentation opt out.
type ParseFailureCounter interface {
	// RecordParseFailure captures one dropped/unparseable line. site identifies
	// the codec + location (e.g. "claude:stream-json"); detail is the parser
	// error or a short reason. The raw line is NOT retained in full — only a
	// bounded prefix — to avoid unbounded growth and transcript leakage.
	RecordParseFailure(runID uuid.UUID, site, detail, lineSample string)
}

// DefaultParseFailureSampleCap bounds the sample ring and the retained line
// prefix so a storm of drift lines cannot grow memory without bound.
const (
	DefaultParseFailureSampleCap = 64
	parseFailureLineSampleMax    = 240
)

// Parse-failure site constants. A site identifies the codec + location that
// dropped a line. They are shared between the recording call sites and the
// tests so a typo'd expected site cannot pass vacuously and so a renamed site
// is a single-edit change across production and coverage.
const (
	siteClaudeStreamJSON   = "claude:stream-json"
	siteClaudeToolUseInput = "claude:tool-use-input"
	siteClaudeToolResult   = "claude:tool-result"
	siteClaudeResult       = "claude:result"
	siteClaudeSystemResult = "claude:system-result"
	siteClaudeToolInput    = "claude:tool-input"
	siteCodexStreamJSON    = "codex:stream-json"
	siteOpenCodeToolInput  = "opencode:tool-input"
	siteGrokToolInput      = "grok:tool-input"
)

// ParseFailureSample is one retained dropped-line record.
type ParseFailureSample struct {
	RunID      uuid.UUID `json:"run_id"`
	Site       string    `json:"site"`
	Detail     string    `json:"detail"`
	LineSample string    `json:"line_sample,omitempty"`
}

// ParseFailureRegistry is the default ParseFailureCounter: per-site counts plus
// a bounded ring of the most recent samples. It is the process-wide sink read
// by the conversation-index status surface.
type ParseFailureRegistry struct {
	mu       sync.Mutex
	total    uint64
	bySite   map[string]uint64
	samples  []ParseFailureSample // ring buffer, most-recent-wins
	nextSlot int
	cap      int
}

// NewParseFailureRegistry returns a registry with the given sample ring size.
// A non-positive size falls back to DefaultParseFailureSampleCap.
func NewParseFailureRegistry(sampleCap int) *ParseFailureRegistry {
	if sampleCap <= 0 {
		sampleCap = DefaultParseFailureSampleCap
	}
	return &ParseFailureRegistry{bySite: make(map[string]uint64), cap: sampleCap}
}

// RecordParseFailure satisfies ParseFailureCounter.
func (r *ParseFailureRegistry) RecordParseFailure(runID uuid.UUID, site, detail, lineSample string) {
	if r == nil {
		return
	}
	if len(lineSample) > parseFailureLineSampleMax {
		lineSample = lineSample[:parseFailureLineSampleMax]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total++
	if r.bySite == nil {
		r.bySite = make(map[string]uint64)
	}
	r.bySite[site]++
	sample := ParseFailureSample{RunID: runID, Site: site, Detail: detail, LineSample: lineSample}
	if len(r.samples) < r.cap {
		r.samples = append(r.samples, sample)
		return
	}
	// Ring is full: overwrite the oldest slot so only the most recent cap
	// samples are retained.
	r.samples[r.nextSlot] = sample
	r.nextSlot = (r.nextSlot + 1) % r.cap
}

// Snapshot is a read-only view of accumulated parse failures.
type Snapshot struct {
	Total   uint64               `json:"total"`
	BySite  map[string]uint64    `json:"by_site,omitempty"`
	Samples []ParseFailureSample `json:"samples,omitempty"`
}

// Snapshot returns the accumulated counts and the retained samples in
// chronological order (oldest first).
func (r *ParseFailureRegistry) Snapshot() Snapshot {
	if r == nil {
		return Snapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	bySite := make(map[string]uint64, len(r.bySite))
	for k, v := range r.bySite {
		bySite[k] = v
	}
	var samples []ParseFailureSample
	if len(r.samples) < r.cap {
		samples = append(samples, r.samples...)
	} else {
		// Reconstruct chronological order from the ring: slot nextSlot is the
		// oldest retained sample.
		samples = append(samples, r.samples[r.nextSlot:]...)
		samples = append(samples, r.samples[:r.nextSlot]...)
	}
	return Snapshot{Total: r.total, BySite: bySite, Samples: samples}
}

// defaultParseFailures is the process-wide sink. Codecs record here unless a
// test swaps it via SetDefaultParseFailureCounter. Guarded by its own mutex so
// a test swap races safely with concurrent decode.
var (
	defaultParseFailuresMu sync.RWMutex
	defaultParseFailures   ParseFailureCounter = NewParseFailureRegistry(DefaultParseFailureSampleCap)
)

// SetDefaultParseFailureCounter replaces the process-wide counter and returns
// the previous one so a test can restore it. Passing nil disables recording.
func SetDefaultParseFailureCounter(c ParseFailureCounter) ParseFailureCounter {
	defaultParseFailuresMu.Lock()
	defer defaultParseFailuresMu.Unlock()
	prev := defaultParseFailures
	defaultParseFailures = c
	return prev
}

// DefaultParseFailureSnapshot returns the current snapshot when the process-wide
// counter is a *ParseFailureRegistry (the default). A custom injected counter
// that is not a registry reports an empty snapshot.
func DefaultParseFailureSnapshot() Snapshot {
	defaultParseFailuresMu.RLock()
	c := defaultParseFailures
	defaultParseFailuresMu.RUnlock()
	if reg, ok := c.(*ParseFailureRegistry); ok {
		return reg.Snapshot()
	}
	return Snapshot{}
}

// decodeToolInput unmarshals a tool-call input payload into a map, recording a
// parse failure at site on error. It centralizes the marshal-to-map + record
// idiom repeated across codecs so every tool-input site reports failures
// uniformly. Behavior matches the hand-rolled sites it replaces: a nil/empty
// payload returns a nil map (no record), and a malformed payload returns nil
// after recording — so the emitted tool-call event is byte-identical.
func decodeToolInput(runID uuid.UUID, site string, raw json.RawMessage) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var input map[string]interface{}
	if err := json.Unmarshal(raw, &input); err != nil {
		recordParseFailure(runID, site, err.Error(), string(raw))
		return nil
	}
	return input
}

// recordParseFailure is the single internal entry point every codec silent-skip
// site calls. It routes to the process-wide counter under a read lock so it is
// safe against a concurrent test swap, and is a no-op when recording is
// disabled. It MUST remain side-effect-only with respect to parse results.
func recordParseFailure(runID uuid.UUID, site, detail, lineSample string) {
	defaultParseFailuresMu.RLock()
	c := defaultParseFailures
	defaultParseFailuresMu.RUnlock()
	if c == nil {
		return
	}
	c.RecordParseFailure(runID, site, detail, lineSample)
}
