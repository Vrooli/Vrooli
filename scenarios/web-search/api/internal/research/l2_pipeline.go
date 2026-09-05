package research

import (
	"context"
	"strings"
	"unicode/utf8"

	"web-search/internal/findings"
	"web-search/internal/livesearch"
)

// Candidate is one L0 result the L2 pipeline may fetch and read.
type Candidate struct {
	URL   string
	Title string
}

// CandidateSet is one L0 candidate query's payload: the pages worth fetching
// plus the engine-degradation signal that rode the underlying live search.
type CandidateSet struct {
	Candidates []Candidate
	// DegradedEngines mirrors livesearch.SearchOutcome.DegradedEngines: the
	// upstream engines that did not answer the candidate query. A weak L2
	// synthesis with a non-empty list usually means partial inputs.
	DegradedEngines []livesearch.EngineIssue
}

// Searcher is the L2 candidate-URL seam: it returns the top-N L0 results for a
// query. The production impl wraps the live-search Service; tests inject a fake.
type Searcher interface {
	// Candidates returns up to topN candidate pages for query, best-first.
	Candidates(ctx context.Context, query string, topN int) (CandidateSet, error)
}

// LiveSearcher adapts the internal live-search Service to the Searcher seam.
type LiveSearcher struct {
	Service *livesearch.Service
}

// Candidates runs an L0 live search and projects the results to candidate URLs.
func (l LiveSearcher) Candidates(ctx context.Context, query string, topN int) (CandidateSet, error) {
	if l.Service == nil {
		return CandidateSet{}, nil
	}
	out, err := l.Service.Search(ctx, livesearch.SearchInput{Query: query, Limit: topN})
	if err != nil {
		return CandidateSet{}, err
	}
	cands := make([]Candidate, 0, len(out.Results))
	for _, r := range out.Results {
		if strings.TrimSpace(r.URL) == "" {
			continue
		}
		cands = append(cands, Candidate{URL: r.URL, Title: r.Title})
		if len(cands) >= topN {
			break
		}
	}
	return CandidateSet{Candidates: cands, DegradedEngines: out.DegradedEngines}, nil
}

// runL2 executes the synchronous L2 pipeline: candidates -> fetch each ->
// single-pass cited synthesis -> (optional) capture. It is the shared core both
// the RunL2 RPC and the L3 reconcile loop's "research the gap" step call.
//
// Fetch failures are tolerated per-page: a page that cannot be fetched is
// skipped and the synthesis runs over whatever was retrieved. With no fetched
// documents the pipeline abstains rather than fabricating.
func (s *Service) runL2(ctx context.Context, query string, topN int, capture bool) (L2Outcome, error) {
	query = strings.TrimSpace(query)
	if topN <= 0 {
		topN = DefaultTopN
	}
	if topN > MaxTopN {
		topN = MaxTopN
	}

	candSet, err := s.searcher.Candidates(ctx, query, topN)
	if err != nil {
		return L2Outcome{}, err
	}
	cands := candSet.Candidates

	docs := make([]Document, 0, len(cands))
	for _, c := range cands {
		text, ferr := s.fetcher.Fetch(ctx, c.URL)
		if ferr != nil {
			s.logger.Printf("research: L2 fetch %q failed (skipping): %v", c.URL, ferr)
			continue
		}
		if strings.TrimSpace(text) == "" {
			s.logger.Printf("research: L2 fetch %q returned no readable text (skipping)", c.URL)
			continue
		}
		docs = append(docs, Document{URL: c.URL, Title: c.Title, Text: text})
	}

	// Excerpting decides what part of each fetched page the model reads
	// (relevance-selected by default, positional under the escape hatch or on
	// embedder degradation). Citation indices stay stable: the excerpter
	// preserves document order.
	var excerpts []DocumentExcerpt
	syn := Abstain()
	switch {
	case len(cands) == 0:
		syn = AbstainWith(ReasonNoCandidates)
	case len(docs) == 0:
		syn = AbstainWith(ReasonAllFetchesEmpty)
	default:
		docs = s.excerpter.Select(ctx, query, docs)
		excerpts = excerptsForResponse(docs)
		syn, err = s.synthesizer.Synthesize(ctx, query, docs)
		if err != nil {
			return L2Outcome{}, err
		}
	}

	out := L2Outcome{
		Brief: Brief{
			Query:     query,
			Level:     LevelL2,
			Summary:   syn.Text,
			Citations: syn.Citations,
		},
		Abstained:       syn.Abstained,
		AbstainReason:   syn.AbstainReason,
		Excerpts:        excerpts,
		DegradedEngines: candSet.DegradedEngines,
	}

	// Auto-capture is opt-in for L2 and never fires on an abstention (there is no
	// grounded claim to persist).
	if capture && !syn.Abstained {
		ids, cerr := s.captureSynthesis(ctx, query, syn)
		if cerr != nil {
			s.logger.Printf("research: L2 capture failed (returning synthesis): %v", cerr)
		}
		out.CapturedFindingIDs = ids
	}
	return out, nil
}

// excerptPreviewChars caps each per-document excerpt mirrored onto the
// response — enough to see WHAT the model read without shipping whole pages.
const excerptPreviewChars = 800

// excerptsForResponse projects the (already excerpted) documents sent to the
// model into the response-side observability mirror, transport-capped.
func excerptsForResponse(docs []Document) []DocumentExcerpt {
	out := make([]DocumentExcerpt, len(docs))
	for i, d := range docs {
		out[i] = DocumentExcerpt{URL: d.URL, Title: d.Title, Excerpt: excerptPreview(d.Text)}
	}
	return out
}

// excerptPreview caps text at excerptPreviewChars bytes without ever emitting
// invalid UTF-8. proto3 string fields require valid UTF-8 and the excerpt is
// the only response field carrying raw fetched bytes — one bad byte fails the
// marshal of the WHOLE RunL2Response. Two hazards are scrubbed: bytes the page
// itself carried that aren't UTF-8, and a byte-cap cut landing mid-rune.
func excerptPreview(text string) string {
	text = strings.ToValidUTF8(text, "�")
	if len(text) <= excerptPreviewChars {
		return text
	}
	cut := excerptPreviewChars
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

// captureSynthesis writes the cited L2 synthesis as a single FINDING_SOURCE_L2
// finding, carrying every grounding citation. Returns the written finding ids.
func (s *Service) captureSynthesis(ctx context.Context, query string, syn Synthesis) ([]string, error) {
	if s.findings == nil {
		return nil, nil
	}
	claim := strings.TrimSpace(syn.Text)
	if claim == "" {
		return nil, nil
	}
	cites := make([]findings.NewCitation, 0, len(syn.Citations))
	for _, c := range syn.Citations {
		cites = append(cites, findings.NewCitation{URL: c.URL, Title: c.Title})
	}
	f, err := s.findings.Add(ctx, findings.NewFinding{
		Claim:      claim,
		Confidence: l2CaptureConfidence,
		Query:      query,
		Source:     findings.SourceL2,
		Citations:  cites,
	})
	if err != nil {
		return nil, err
	}
	return []string{f.ID}, nil
}

// l2CaptureConfidence is the confidence stamped on an auto-captured L2 finding.
// L2 is a single-pass synthesis (no cross-checking loop), so it lands below the
// high-confidence mutation gate — an L2 capture can seed the store but is not
// trusted enough to silently overwrite a contested claim.
const l2CaptureConfidence = 0.6

// Ensure LiveSearcher satisfies the seam at compile time.
var _ Searcher = LiveSearcher{}
