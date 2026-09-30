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
// tree) and reuses the same AppendJSONL + trimJSONLLines primitives as the
// action run-history audit.
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
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if err := AppendJSONL(s.path, miss); err != nil {
		return err
	}
	if err := s.prune(now); err != nil {
		return err
	}
	return trimJSONLLines(s.path, s.maxEntries)
}

// ReadSince returns misses with an At timestamp within the given window
// (relative to now), newest entries last (file order).
func (s *DiscoveryMissStore) ReadSince(window time.Duration) ([]DiscoveryMiss, error) {
	all, err := s.readAll()
	if err != nil {
		return nil, err
	}
	if window <= 0 {
		return all, nil
	}
	cutoff := s.now().UTC().Add(-window)
	out := make([]DiscoveryMiss, 0, len(all))
	for _, miss := range all {
		if at, err := time.Parse(time.RFC3339, miss.At); err == nil && at.Before(cutoff) {
			continue
		}
		out = append(out, miss)
	}
	return out, nil
}

func (s *DiscoveryMissStore) prune(now time.Time) error {
	all, err := s.readAll()
	if err != nil {
		return err
	}
	cutoff := now.Add(-s.retention)
	kept := make([]DiscoveryMiss, 0, len(all))
	dropped := false
	for _, miss := range all {
		if at, err := time.Parse(time.RFC3339, miss.At); err == nil && at.Before(cutoff) {
			dropped = true
			continue
		}
		kept = append(kept, miss)
	}
	if !dropped {
		return nil
	}
	return s.rewrite(kept)
}

func (s *DiscoveryMissStore) readAll() ([]DiscoveryMiss, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []DiscoveryMiss
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var miss DiscoveryMiss
		if err := json.Unmarshal([]byte(line), &miss); err != nil {
			continue // skip malformed lines rather than failing the read
		}
		out = append(out, miss)
	}
	return out, nil
}

func (s *DiscoveryMissStore) rewrite(entries []DiscoveryMiss) error {
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

func newMissID(now time.Time) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return strings.ReplaceAll(now.Format("20060102T150405.000000000"), ".", "")
	}
	return now.Format("20060102T150405") + "-" + hex.EncodeToString(buf)
}
