package phasecache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/vrooli/freshness-go/treedigest"
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
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

// normalizedVerdict is the stable behavior-bearing projection used by cache
// audits. Timing, paths, metrics, orchestration observations, and cache
// provenance can differ between correct runs; status, finding identities, and
// the provider's maturity standing cannot.
type normalizedVerdict struct {
	Status     string
	FindingSet []string
	Standing   string
}

// Equivalent compares the normalized validation verdict rather than the
// marshalled execution result. This prevents ordering and run-local metadata
// from demoting a deterministic cache entry while still detecting changed
// findings and maturity standing.
func Equivalent(a, b phases.ExecutionResult) bool {
	return reflect.DeepEqual(normalized(a), normalized(b))
}

func normalized(result phases.ExecutionResult) normalizedVerdict {
	verdict := normalizedVerdict{Status: strings.TrimSpace(result.Status), Standing: assessmentStanding(result)}
	for _, finding := range result.Findings {
		if finding == nil {
			continue
		}
		verdict.FindingSet = append(verdict.FindingSet, findingIdentity(finding))
	}
	sort.Strings(verdict.FindingSet)
	return verdict
}

func findingIdentity(finding *architecturev1.ArchitectureFinding) string {
	token := strings.TrimSpace(finding.GetStableId())
	if token == "" {
		token = findingid.For(finding)
	}
	locations := append([]string(nil), finding.GetLocations()...)
	sort.Strings(locations)
	subject := ""
	if target := finding.GetSubject(); target != nil {
		subject = strings.Join([]string{target.GetKind().String(), target.GetId(), target.GetRoot()}, "\x1e")
	}
	return strings.Join([]string{
		token,
		strings.TrimSpace(finding.GetCode()),
		finding.GetSeverity().String(),
		subject,
		strings.Join(locations, "\x1e"),
	}, "\x1f")
}

func assessmentStanding(result phases.ExecutionResult) string {
	if result.Assessment == nil || result.Assessment.GetPresentation() == nil {
		return ""
	}
	presentation := result.Assessment.GetPresentation()
	return strings.Join([]string{
		presentation.GetCurrentLevel(),
		presentation.GetCurrentLevelLabel(),
		strconv.FormatBool(presentation.GetClean()),
		presentation.GetNextLevel(),
	}, "\x1f")
}

// cacheComparableObservations excludes orchestration diagnostics that describe
// how a phase was admitted, rather than what the provider observed. Admission
// can legitimately differ between runs under queue pressure; allowing that
// warning to poison a cache audit would demote deterministic provider results.
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
