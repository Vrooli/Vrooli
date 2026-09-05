package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
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
// and reuses the same AppendJSONL + trimJSONLLines primitives as the discovery
// -miss audit, in a SEPARATE file so miss-mining semantics stay clean.
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
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if err := AppendJSONL(s.path, call); err != nil {
		return err
	}
	if err := s.prune(now); err != nil {
		return err
	}
	return trimJSONLLines(s.path, s.maxEntries)
}

// ReadSince returns calls with an At timestamp within the given window
// (relative to now), newest entries last (file order).
func (s *DiscoveryCallStore) ReadSince(window time.Duration) ([]DiscoveryCall, error) {
	all, err := s.readAll()
	if err != nil {
		return nil, err
	}
	if window <= 0 {
		return all, nil
	}
	cutoff := s.now().UTC().Add(-window)
	out := make([]DiscoveryCall, 0, len(all))
	for _, call := range all {
		if at, err := time.Parse(time.RFC3339, call.At); err == nil && at.Before(cutoff) {
			continue
		}
		out = append(out, call)
	}
	return out, nil
}

func (s *DiscoveryCallStore) prune(now time.Time) error {
	all, err := s.readAll()
	if err != nil {
		return err
	}
	cutoff := now.Add(-s.retention)
	kept := make([]DiscoveryCall, 0, len(all))
	dropped := false
	for _, call := range all {
		if at, err := time.Parse(time.RFC3339, call.At); err == nil && at.Before(cutoff) {
			dropped = true
			continue
		}
		kept = append(kept, call)
	}
	if !dropped {
		return nil
	}
	return s.rewrite(kept)
}

func (s *DiscoveryCallStore) readAll() ([]DiscoveryCall, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []DiscoveryCall
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var call DiscoveryCall
		if err := json.Unmarshal([]byte(line), &call); err != nil {
			continue // skip malformed lines rather than failing the read
		}
		out = append(out, call)
	}
	return out, nil
}

func (s *DiscoveryCallStore) rewrite(entries []DiscoveryCall) error {
	var b strings.Builder
	for _, entry := range entries {
		raw, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	return os.WriteFile(s.path, []byte(b.String()), 0o644)
}

func newCallID(now time.Time) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strings.ReplaceAll(now.Format("20060102T150405.000000000"), ".", "")
	}
	return now.Format("20060102T150405") + "-" + hex.EncodeToString(buf)
}
