package phasecache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/freshness-go/treedigest"
	"test-genie/internal/orchestrator/phases"
)

// Identity is the four-part cache identity. Each field is supplied by the
// owner of that identity; the cache never substitutes timestamps or file
// modification times.
type Identity struct {
	ScopedInputDigest      string
	ProviderBuildIdentity  string
	DescriptorSnapshotHash string
	ExecutionConfiguration string
}

type Entry struct {
	Key   string                 `json:"key"`
	RunID string                 `json:"runId"`
	Phase phases.ExecutionResult `json:"phase"`
}

type Store struct{ root string }

var demotionMu sync.Mutex

func New(root string) *Store { return &Store{root: filepath.Join(root, "phase-cache")} }

func Key(id Identity) string {
	payload := strings.Join([]string{"v1", id.ScopedInputDigest, id.ProviderBuildIdentity, id.DescriptorSnapshotHash, id.ExecutionConfiguration}, "\n")
	sum := sha256.Sum256([]byte(payload))
	return "pc:sha256:" + hex.EncodeToString(sum[:])
}

func (s *Store) Load(key string) (Entry, bool, error) {
	if s == nil || strings.TrimSpace(key) == "" {
		return Entry{}, false, nil
	}
	data, err := os.ReadFile(filepath.Join(s.root, safeName(key)+".json"))
	if os.IsNotExist(err) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, err
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return Entry{}, false, fmt.Errorf("decode phase cache entry: %w", err)
	}
	if entry.Key != key || entry.Phase.Status != "passed" {
		return Entry{}, false, nil
	}
	if s.IsDemoted(key) {
		return Entry{}, false, nil
	}
	return entry, true, nil
}

// ShouldAudit deterministically samples cache hits. Stable sampling keeps a
// retry from changing policy halfway through a run while still exercising the
// provider on a bounded fraction of otherwise reusable results.
func (s *Store) ShouldAudit(key, runID string) bool {
	percent := 10
	if raw := strings.TrimSpace(os.Getenv("TEST_GENIE_PHASE_CACHE_AUDIT_PERCENT")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			percent = parsed
		}
	}
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	sum := sha256.Sum256([]byte(key + "\n" + runID))
	return int(sum[0])%100 < percent
}

// Demote records a cache key that produced a different result during audit.
// Demotion is fail-closed: the key will never serve the stale result again,
// while a later source/provider/config identity naturally gets a new key.
func (s *Store) Demote(key, reason string) error {
	if s == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	demotionMu.Lock()
	defer demotionMu.Unlock()
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.root, "demotions.json")
	demotions := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &demotions)
	}
	demotions[key] = strings.TrimSpace(reason)
	data, err := json.MarshalIndent(demotions, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.root, ".phase-demotions-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *Store) IsDemoted(key string) bool {
	if s == nil || strings.TrimSpace(key) == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(s.root, "demotions.json"))
	if err != nil {
		return false
	}
	var demotions map[string]string
	if json.Unmarshal(data, &demotions) != nil {
		return false
	}
	_, found := demotions[key]
	return found
}

// Equivalent compares the observable phase result and excludes run-local
// paths, timing, and cache provenance. A cache audit must judge behavior, not
// whether two executions happened to take the same number of milliseconds.
func Equivalent(a, b phases.ExecutionResult) bool {
	a.DurationSeconds, a.DurationMilliseconds, a.PredictedDurationMilliseconds, a.LogPath = 0, 0, 0, ""
	b.DurationSeconds, b.DurationMilliseconds, b.PredictedDurationMilliseconds, b.LogPath = 0, 0, 0, ""
	a.CacheHit, a.CacheSourceRunID = false, ""
	b.CacheHit, b.CacheSourceRunID = false, ""
	// Provider metrics describe this execution's timing and host resources;
	// they are deliberately not part of the observable validation result.
	a.Metrics, b.Metrics = nil, nil
	a.Observations = cacheComparableObservations(a.Observations)
	b.Observations = cacheComparableObservations(b.Observations)
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

// cacheComparableObservations excludes orchestration diagnostics that describe
// how a phase was admitted, rather than what the provider observed. Admission
// can legitimately differ between runs under queue pressure; allowing that
// warning to poison a cache audit would demote deterministic provider results.
func cacheComparableObservations(observations []phases.Observation) []phases.Observation {
	if len(observations) == 0 {
		return observations
	}
	filtered := make([]phases.Observation, 0, len(observations))
	for _, observation := range observations {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(observation.Text)), "scheduler ") {
			continue
		}
		filtered = append(filtered, observation)
	}
	return filtered
}

func (s *Store) Save(key, runID string, phase phases.ExecutionResult) error {
	if s == nil || strings.TrimSpace(key) == "" || phase.Status != "passed" {
		return nil
	}
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	entry := Entry{Key: key, RunID: runID, Phase: phase}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("encode phase cache entry: %w", err)
	}
	tmp, err := os.CreateTemp(s.root, ".phase-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(s.root, safeName(key)+".json"))
}

func ScopedDigest(root string, inputs []string) (string, error) {
	return treedigest.ComputeScoped(root, inputs)
}

func safeName(key string) string {
	return strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(key)
}
