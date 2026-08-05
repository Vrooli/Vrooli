package forest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"vrooli-memory/internal/inference"
)

// CandidateSource lets forest enforce retention and pin policy without owning
// journal or facets storage.
type (
	CandidateSource interface {
		CompactionCandidates(context.Context) ([]Candidate, error)
	}
	Candidate struct {
		ID, FacetID, Body string
		RetentionPolicy   string
		Vector            []float64
		CreatedAt         time.Time
		Depth, Generation int
		Kind              string
		Pinned            bool
	}
	Config struct {
		Target       int
		RecencyFloor time.Duration
	}
	Service struct {
		repo      Repository
		source    CandidateSource
		inference inference.Client
		config    Config
		now       func() time.Time
		runMu     sync.Mutex
	}
)

func NewService(repo Repository, source CandidateSource, client inference.Client, config Config) *Service {
	if config.Target <= 0 {
		config.Target = 16
	}
	if config.RecencyFloor <= 0 {
		config.RecencyFloor = 24 * time.Hour
	}
	return &Service{repo: repo, source: source, inference: client, config: config, now: func() time.Time { return time.Now().UTC() }}
}

// Run keeps the compactable portion of the mixed-depth frontier within its
// pressure target. Non-episode leaves remain roots by design: they are durable
// recall records, not candidates for lossy contextual compaction.
func (s *Service) Run(ctx context.Context) (CompactionResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	return s.runLocked(ctx)
}

func (s *Service) runLocked(ctx context.Context) (CompactionResult, error) {
	candidates, err := s.eligible(ctx)
	if err != nil {
		return CompactionResult{}, err
	}
	result := CompactionResult{EligibleFrontierBefore: len(candidates), EligibleFrontierAfter: len(candidates)}
	failedPairs := make(map[string]struct{})
	var lastSummarizeErr error
	for len(candidates) > s.config.Target {
		best, pairKey, ok := bestPairExcluding(candidates, failedPairs)
		if !ok {
			if lastSummarizeErr != nil {
				return result, fmt.Errorf("summarize compaction cluster: %w", lastSummarizeErr)
			}
			return result, fmt.Errorf("no eligible candidate pair remains while frontier is above target")
		}
		prompt := summaryPrompt(best[0], best[1])
		body, err := s.summarize(ctx, prompt)
		if err != nil {
			// A single malformed or provider-hostile cluster must not strand the
			// entire frontier. It is safe to leave this pair untouched and try the
			// next scored pair; an all-failing corpus still returns the last error
			// below without writing anything for the failed pair.
			failedPairs[pairKey] = struct{}{}
			lastSummarizeErr = err
			if len(failedPairs) >= 8 {
				return result, fmt.Errorf("summarize compaction cluster: %w", err)
			}
			continue
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return result, fmt.Errorf("summarize compaction cluster: empty summary")
		}
		vector, err := s.inference.Embed(ctx, body, inference.EmbeddingClustering)
		if err != nil {
			return result, fmt.Errorf("embed compaction summary: %w", err)
		}
		depth, generation := max(best[0].Depth, best[1].Depth)+1, max(best[0].Generation, best[1].Generation)+1
		_, err = s.repo.CreateSummary(ctx, Summary{Body: body, FacetID: best[0].FacetID, Vector: vector, Depth: depth, Generation: generation}, []Edge{{ChildID: best[0].ID, ChildKind: best[0].Kind}, {ChildID: best[1].ID, ChildKind: best[1].Kind}})
		if err != nil {
			return result, fmt.Errorf("write compaction summary: %w", err)
		}
		result.CompactedCount++
		candidates, err = s.eligible(ctx)
		if err != nil {
			return result, err
		}
		result.EligibleFrontierAfter = len(candidates)
	}
	return result, nil
}

// summarize retries the provider's transient stream-termination failure before
// the atomic write begins. A final error still leaves the forest untouched.
func (s *Service) summarize(ctx context.Context, prompt string) (string, error) {
	const maxAttempts = 2
	for attempt := 0; ; attempt++ {
		body, err := s.inference.Summarize(ctx, prompt)
		if err == nil || attempt == maxAttempts-1 || !strings.Contains(strings.ToLower(err.Error()), "unexpected eof") {
			return body, err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		}
	}
}

func (s *Service) Rebuild(ctx context.Context) (CompactionResult, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if err := s.repo.Rebuild(ctx); err != nil {
		return CompactionResult{}, err
	}
	return s.runLocked(ctx)
}

func (s *Service) Frontier(ctx context.Context) ([]Node, error) {
	return s.repo.Nodes(ctx, s.config.Target)
}

func (s *Service) eligible(ctx context.Context) ([]Candidate, error) {
	candidates, err := s.source.CompactionCandidates(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := s.now().Add(-s.config.RecencyFloor)
	out := candidates[:0]
	for _, c := range candidates {
		if !c.Pinned && (c.RetentionPolicy == "" || c.RetentionPolicy == "compact") && !c.CreatedAt.After(cutoff) && len(c.Vector) > 0 {
			out = append(out, c)
		}
	}
	return out, nil
}

func bestPair(candidates []Candidate) ([2]Candidate, bool) {
	selected, _, ok := bestPairExcluding(candidates, nil)
	return selected, ok
}

func bestPairExcluding(candidates []Candidate, excluded map[string]struct{}) ([2]Candidate, string, bool) {
	var selected [2]Candidate
	best := -1.0
	selectedKey := ""
	for i := range candidates {
		for j := i + 1; j < len(candidates); j++ {
			key := pairKey(candidates[i], candidates[j])
			if _, skip := excluded[key]; skip {
				continue
			}
			score := cosine(candidates[i].Vector, candidates[j].Vector)
			if score > best {
				best, selected, selectedKey = score, [2]Candidate{candidates[i], candidates[j]}, key
			}
		}
	}
	return selected, selectedKey, best >= 0
}

func pairKey(a, b Candidate) string {
	left, right := a.Kind+":"+a.ID, b.Kind+":"+b.ID
	if right < left {
		left, right = right, left
	}
	return left + "|" + right
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		return -1
	}
	return dot / math.Sqrt(aa*bb)
}

func summaryPrompt(a, b Candidate) string {
	ordered := []Candidate{a, b}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].CreatedAt.Before(ordered[j].CreatedAt) })
	return fmt.Sprintf("Summarize these episode memories for future agent context. Preserve supported facts and drop redundancy. A source may have EARLIER CONTEXT, STATUS EVIDENCE, and LATEST CONTEXT. STATUS EVIDENCE and LATEST CONTEXT are authoritative for the source's current state: do not headline, repeat, or infer a current status from EARLIER CONTEXT when either supersedes or conflicts with it. Sources may contain their own dated status updates: within each source, treat its latest explicit dated update as authoritative over earlier prose. Across sources, prefer the later source timestamp, including fractional seconds. Never turn a disagreement into a definite claim: when the excerpts do not establish a current state, say that status is unresolved and identify the competing states. Do not conjoin contradictory claims.\n\n[%s] %s\n\n[%s] %s", ordered[0].CreatedAt.Format(time.RFC3339Nano), boundedSummaryInput(ordered[0].Body), ordered[1].CreatedAt.Format(time.RFC3339Nano), boundedSummaryInput(ordered[1].Body))
}

func boundedSummaryInput(text string) string {
	const limit = 3500
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	const labelsBudget = 160
	const earlierLimit = 500
	const latestLimit = 1300
	const evidenceLimit = limit - labelsBudget - earlierLimit - latestLimit
	return "[EARLIER CONTEXT]\n" + string(runes[:earlierLimit]) + "\n[STATUS EVIDENCE — authoritative for current status]\n" + recentStatusEvidence(string(runes), evidenceLimit) + "\n[... middle of source omitted ...]\n[LATEST CONTEXT — authoritative for current status]\n" + string(runes[len(runes)-latestLimit:])
}

func recentStatusEvidence(text string, limit int) string {
	const radius = 180
	lower := strings.ToLower(text)
	keywords := []string{"shipped", "complete", "completed", "done", "executed", "remaining", "not built", "not started", "status:"}
	positions := make([]int, 0, len(keywords))
	for _, keyword := range keywords {
		if index := strings.LastIndex(lower, keyword); index >= 0 {
			positions = append(positions, index)
		}
	}
	sort.Ints(positions)
	var builder strings.Builder
	for _, position := range positions {
		start, end := max(0, position-radius), min(len(text), position+len("completed")+radius)
		excerpt := strings.TrimSpace(text[start:end])
		if excerpt == "" || strings.Contains(builder.String(), excerpt) {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n[... status passage boundary ...]\n")
		}
		if builder.Len()+len(excerpt) > limit {
			remaining := limit - builder.Len()
			if remaining > 0 {
				builder.WriteString(excerpt[:remaining])
			}
			break
		}
		builder.WriteString(excerpt)
	}
	if builder.Len() == 0 {
		return "No explicit status-bearing passage was found; do not infer a current status."
	}
	return builder.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
