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

	"test-genie/internal/orchestrator/phases"

	"github.com/vrooli/freshness-go/treedigest"
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
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

// cacheableStatuses are the phase verdicts the cache may reuse.
//
// "passed" and "failed" are both DETERMINED by the cache identity: the scoped
// input digest, the provider build identity, the descriptor snapshot, and the
// execution configuration together cover everything that could change the
// verdict. A failure under an unchanged identity is exactly as reusable as a
// pass, and re-deriving it was 36% of all phase time — 44.5 hours over the
// measured window — spent reproducing byte-identical failures.
//
// The counter-argument is that a failure may be caused by external state the
// digest does not cover, so a cached failure could hide a fix. The audit
// sampler is the answer: it already re-runs a sampled fraction of hits, and a
// phase whose cached failure has gone stale is demoted on the next audit. That
// is the same trust model the cache already applies to passes — and a stale
// cached PASS is the more dangerous of the two, because it reports success that
// was never verified.
//
// Every other status is excluded deliberately. "skipped", "missing",
// "not_executable", "not_run", and "provider_unavailable" describe the state of
// the RUN rather than the verdict of the phase; they say the phase did not
// happen, which is not a result to reuse.
var cacheableStatuses = map[string]bool{
	"passed": true,
	"failed": true,
}

// Cacheable reports whether a phase verdict may be stored and reused.
func Cacheable(status string) bool {
	return cacheableStatuses[strings.ToLower(strings.TrimSpace(status))]
}

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
	if entry.Key != key || !Cacheable(entry.Phase.Status) {
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

// Diff describes what changed between a cached result and a freshly executed
// one, for an audit that found them not Equivalent.
//
// The audit used to record only that a mismatch happened. That is enough to
// demote the entry and no help at all in deciding whether the cache was wrong
// or the world moved — the two questions an operator actually has when a
// mismatch appears. A count of mismatches tells you a number; naming the
// difference tells you where to look.
//
// The output is deliberately bounded: a verdict flip and a standing change are
// stated in full, and finding churn is summarised as counts plus a few example
// identities, so a phase that emits hundreds of findings does not produce an
// unreadable finding message.
func Diff(cached, fresh phases.ExecutionResult) string {
	before, after := normalized(cached), normalized(fresh)
	var parts []string

	if before.Status != after.Status {
		parts = append(parts, fmt.Sprintf("verdict %s -> %s", quoteOrNone(before.Status), quoteOrNone(after.Status)))
	}
	if before.Standing != after.Standing {
		parts = append(parts, "maturity standing changed")
	}

	added, removed := diffSets(before.FindingSet, after.FindingSet)
	if len(added) > 0 {
		parts = append(parts, fmt.Sprintf("%d finding(s) appeared (%s)", len(added), summarizeIdentities(added)))
	}
	if len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("%d finding(s) disappeared (%s)", len(removed), summarizeIdentities(removed)))
	}

	if len(parts) == 0 {
		// Equivalent compares the normalized verdict, so reaching here means
		// the two differ in a field normalization deliberately ignores. Say so
		// rather than emitting an empty explanation.
		return "results differ in a field the verdict comparison normalizes away"
	}
	return strings.Join(parts, "; ")
}

// diffSets returns the elements added and removed between two SORTED sets.
func diffSets(before, after []string) (added, removed []string) {
	inBefore := make(map[string]bool, len(before))
	for _, v := range before {
		inBefore[v] = true
	}
	inAfter := make(map[string]bool, len(after))
	for _, v := range after {
		inAfter[v] = true
		if !inBefore[v] {
			added = append(added, v)
		}
	}
	for _, v := range before {
		if !inAfter[v] {
			removed = append(removed, v)
		}
	}
	return added, removed
}

// maxDiffExamples bounds how many finding identities a diff names.
const maxDiffExamples = 3

// summarizeIdentities renders a few finding identities readably. A finding
// identity packs several fields behind unit separators; only the leading token
// (the stable id) is useful in a message.
func summarizeIdentities(identities []string) string {
	shown := identities
	suffix := ""
	if len(shown) > maxDiffExamples {
		shown = shown[:maxDiffExamples]
		suffix = fmt.Sprintf(", and %d more", len(identities)-maxDiffExamples)
	}
	tokens := make([]string, 0, len(shown))
	for _, id := range shown {
		token := id
		if idx := strings.IndexAny(token, "\x1e\x1f\n"); idx > 0 {
			token = token[:idx]
		}
		if token == "" {
			token = "(unnamed)"
		}
		tokens = append(tokens, token)
	}
	return strings.Join(tokens, ", ") + suffix
}

func quoteOrNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return strconv.Quote(s)
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
	if s == nil || strings.TrimSpace(key) == "" || !Cacheable(phase.Status) {
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
