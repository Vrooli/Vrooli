package store

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

const (
	// maxDiscoveryCallEntries bounds the discovery-call log size regardless of
	// age, so a burst of discover calls can never grow the file without limit.
	maxDiscoveryCallEntries = 5000
	// discoveryCallRetention drops calls older than this window on append, so
	// the log reflects recent discovery behavior rather than all-time history.
	discoveryCallRetention = 30 * 24 * time.Hour
)

// DiscoveryCallResult is one result line within a recorded discover call. It
// captures just enough to reconstruct the threshold/budget story: the score
// (vs the active threshold) and the chars (vs the budget).
type DiscoveryCallResult struct {
	ID     string  `json:"id"`
	Score  float64 `json:"score"`
	Chars  int     `json:"chars"`
	Source string  `json:"source"`         // topic | search
	Type   string  `json:"type,omitempty"` // skill | action
}

// DiscoveryCall is one record of a discover invocation. Unlike DiscoveryMiss
// (which records only unmet queries), this is appended for EVERY discover call,
// so the pipeline's threshold-clipping, budget status, and trimming behavior is
// measurable from data rather than reasoned about from static defaults.
type DiscoveryCall struct {
	ID                string   `json:"id"`
	At                string   `json:"at"` // RFC3339
	Queries           []string `json:"queries"`
	Type              string   `json:"type"`                 // skill | action | all
	Complexity        string   `json:"complexity,omitempty"` // minor|moderate|major|architectural
	Threshold         float64  `json:"threshold"`            // active similarity floor at call time
	BudgetChars       int      `json:"budgetChars,omitempty"`
	TotalContentChars int      `json:"totalContentChars"`
	BudgetStatus      string   `json:"budgetStatus,omitempty"` // under | at | over
	ReturnedCount     int      `json:"returnedCount"`
	TrimmedCount      int      `json:"trimmedCount"` // skills dropped by the over-budget trim
	// ClippedBelowThreshold is the count of relevant results the threshold
	// dropped, measured by the opt-in probe (re-search at threshold 0). nil
	// means the call was not probed; 0 means probed and nothing was clipped.
	ClippedBelowThreshold *int                  `json:"clippedBelowThreshold,omitempty"`
	Results               []DiscoveryCallResult `json:"results,omitempty"`
	Caller                string                `json:"caller,omitempty"`
}

// DiscoveryCallStore persists discovery calls to a bounded, time-windowed JSONL
// file. It lives under the runtime-data root (never the git-tracked store tree)
// and uses the shared bounded JSONL owner while keeping call telemetry separate
// from discovery-miss records.
type DiscoveryCallStore struct {
	path       string
	now        func() time.Time
	maxEntries int
	retention  time.Duration
}

// NewDiscoveryCallStore builds a store rooted at runtimeDataDir, resolved by the
// caller through the api-core/storage path layer (no hard-coded ~/.vrooli).
func NewDiscoveryCallStore(runtimeDataDir string) *DiscoveryCallStore {
	return &DiscoveryCallStore{
		path:       filepath.Join(runtimeDataDir, "discovery-calls.jsonl"),
		now:        time.Now,
		maxEntries: maxDiscoveryCallEntries,
		retention:  discoveryCallRetention,
	}
}

// Append records one call, stamping ID/At when absent, then prunes entries
// older than the retention window and trims the file to the size bound.
func (s *DiscoveryCallStore) Append(call DiscoveryCall) error {
	now := s.now().UTC()
	if strings.TrimSpace(call.At) == "" {
		call.At = now.Format(time.RFC3339)
	}
	if strings.TrimSpace(call.ID) == "" {
		call.ID = newCallID(now)
	}
	return appendBoundedJSONL(s.path, call, now, s.maxEntries, s.retention, func(entry DiscoveryCall) string {
		return entry.At
	})
}

// ReadSince returns calls with an At timestamp within the given window
// (relative to now), newest entries last (file order).
func (s *DiscoveryCallStore) ReadSince(window time.Duration) ([]DiscoveryCall, error) {
	var now time.Time
	if window > 0 {
		now = s.now().UTC()
	}
	return readJSONLSince(s.path, window, now, func(entry DiscoveryCall) string { return entry.At })
}

func newCallID(now time.Time) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strings.ReplaceAll(now.Format("20060102T150405.000000000"), ".", "")
	}
	return now.Format("20060102T150405") + "-" + hex.EncodeToString(buf)
}
