package research

import (
	"context"
	"strings"

	"web-search/internal/findings"
	"web-search/internal/livesearch"
)

// Candidate is one L0 result the L2 pipeline may fetch and read.
type Candidate struct {
	URL   string
	Title string
}

// Searcher is the L2 candidate-URL seam: it returns the top-N L0 results for a
// query. The production impl wraps the live-search Service; tests inject a fake.
type Searcher interface {
	// Candidates returns up to topN candidate pages for query, best-first.
	Candidates(ctx context.Context, query string, topN int) ([]Candidate, error)
}

// LiveSearcher adapts the internal live-search Service to the Searcher seam.
type LiveSearcher struct {
	Service *livesearch.Service
}

// Candidates runs an L0 live search and projects the results to candidate URLs.
func (l LiveSearcher) Candidates(ctx context.Context, query string, topN int) ([]Candidate, error) {
	if l.Service == nil {
		return nil, nil
	}
	out, err := l.Service.Search(ctx, livesearch.SearchInput{Query: query, Limit: topN})
	if err != nil {
		return nil, err
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
	return cands, nil
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

	cands, err := s.searcher.Candidates(ctx, query, topN)
	if err != nil {
		return L2Outcome{}, err
	}

	docs := make([]Document, 0, len(cands))
	for _, c := range cands {
		text, ferr := s.fetcher.Fetch(ctx, c.URL)
		if ferr != nil {
			s.logger.Printf("research: L2 fetch %q failed (skipping): %v", c.URL, ferr)
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		docs = append(docs, Document{URL: c.URL, Title: c.Title, Text: text})
	}

	syn := Abstain()
	if len(docs) > 0 {
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
		Abstained: syn.Abstained,
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
