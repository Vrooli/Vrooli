package store

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

const (
	// maxDiscoveryMissEntries bounds the discovery-miss log size regardless of
	// age, so a burst of misses can never grow the file without limit.
	maxDiscoveryMissEntries = 5000
	// discoveryMissRetention drops misses older than this window on append, so
	// the log reflects recent unmet-capability signal rather than all-time
	// history.
	discoveryMissRetention = 30 * 24 * time.Hour
)

// DiscoveryMiss is one record of a discover/search that returned nothing useful
// (zero results or only sub-threshold matches). It is the durable signal the
// meta-optimization team mines for unmet-capability "alpha".
type DiscoveryMiss struct {
	ID          string  `json:"id"`
	At          string  `json:"at"` // RFC3339
	Query       string  `json:"query"`
	Type        string  `json:"type"` // skill | action | all
	TopScore    float64 `json:"topScore"`
	ResultCount int     `json:"resultCount"`
	Complexity  string  `json:"complexity,omitempty"`
	Caller      string  `json:"caller,omitempty"`
}

// DiscoveryMissStore persists discovery misses to a bounded, time-windowed
// JSONL file. It lives under the runtime-data root (never the git-tracked store
// tree) and uses the shared bounded JSONL append/read owner; its record shape and
// retention policy remain specific to discovery misses.
type DiscoveryMissStore struct {
	path       string
	now        func() time.Time
	maxEntries int
	retention  time.Duration
}

// NewDiscoveryMissStore builds a store rooted at runtimeDataDir, resolved by the
// caller through the api-core/storage path layer (no hard-coded ~/.vrooli).
func NewDiscoveryMissStore(runtimeDataDir string) *DiscoveryMissStore {
	return &DiscoveryMissStore{
		path:       filepath.Join(runtimeDataDir, "discovery-misses.jsonl"),
		now:        time.Now,
		maxEntries: maxDiscoveryMissEntries,
		retention:  discoveryMissRetention,
	}
}

// Append records one miss, stamping ID/At when absent, then prunes entries
// older than the retention window and trims the file to the size bound.
func (s *DiscoveryMissStore) Append(miss DiscoveryMiss) error {
	now := s.now().UTC()
	if strings.TrimSpace(miss.At) == "" {
		miss.At = now.Format(time.RFC3339)
	}
	if strings.TrimSpace(miss.ID) == "" {
		miss.ID = newMissID(now)
	}
	return appendBoundedJSONL(s.path, miss, now, s.maxEntries, s.retention, func(entry DiscoveryMiss) string {
		return entry.At
	})
}

// ReadSince returns misses with an At timestamp within the given window
// (relative to now), newest entries last (file order).
func (s *DiscoveryMissStore) ReadSince(window time.Duration) ([]DiscoveryMiss, error) {
	var now time.Time
	if window > 0 {
		now = s.now().UTC()
	}
	return readJSONLSince(s.path, window, now, func(entry DiscoveryMiss) string { return entry.At })
}

func newMissID(now time.Time) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strings.ReplaceAll(now.Format("20060102T150405.000000000"), ".", "")
	}
	return now.Format("20060102T150405") + "-" + hex.EncodeToString(buf)
}
