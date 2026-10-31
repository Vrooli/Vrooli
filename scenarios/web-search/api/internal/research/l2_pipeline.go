package research

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"web-search/internal/evidence"
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
	Fresh   bool
}

// Candidates runs an L0 live search and projects the results to candidate URLs.
func (l LiveSearcher) Candidates(ctx context.Context, query string, topN int) (CandidateSet, error) {
	if l.Service == nil {
		return CandidateSet{}, nil
	}
	out, err := l.Service.Search(ctx, livesearch.SearchInput{Query: query, Limit: topN, Fresh: l.Fresh})
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
// cited synthesis (one pass for an untyped query, one bounded pass per declared
// question) -> (optional) capture. It is the shared core both
// the RunL2 RPC and the L3 reconcile loop's "research the gap" step call.
//
// Fetch failures are tolerated per-page: a page that cannot be fetched is
// skipped and the synthesis runs over whatever was retrieved. With no fetched
// documents the pipeline abstains rather than fabricating.
func (s *Service) runL2(ctx context.Context, query string, topN int, capture bool, questions []ResearchQuestion, maxEvidenceBytes int) (L2Outcome, error) {
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

	docs, receiptIDs, failures := s.collectDocuments(ctx, cands, maxEvidenceBytes)

	// Excerpting decides what part of each fetched page the model reads
	// (relevance-selected by default, positional under the escape hatch or on
	// embedder degradation). Citation indices stay stable: the excerpter
	// preserves document order.
	var excerpts []DocumentExcerpt
	syn := Abstain()
	var questionAssessments []ClaimAssessment
	var claimsByQuestion map[string][]string
	switch {
	case len(cands) == 0:
		syn = AbstainWith(ReasonNoCandidates)
	case len(docs) == 0:
		syn = AbstainWith(ReasonAllFetchesEmpty)
	default:
		docs = s.excerpter.Select(ctx, query, docs)
		excerpts = excerptsForResponse(docs)
		if len(questions) > 0 {
			syn, questionAssessments, claimsByQuestion, err = s.synthesizeQuestions(ctx, questions, docs)
		} else {
			syn, err = s.synthesizer.Synthesize(ctx, query, docs)
		}
		if err != nil {
			// Preserve the immutable retrieval evidence and explicit abstention
			// state even when the verifier/model is unavailable. Callers receive
			// the error and must not treat this as supported output.
			return L2Outcome{
				Brief:     Brief{Query: query, Level: LevelL2, Summary: abstainNote},
				Abstained: true, AbstainReason: ReasonSynthesisUnavailable,
				Excerpts: excerpts, DegradedEngines: candSet.DegradedEngines,
				EvidenceReceiptIDs: receiptIDs, FetchFailures: failures,
			}, err
		}
	}

	out := L2Outcome{
		Brief: Brief{
			Query:     query,
			Level:     LevelL2,
			Summary:   syn.Text,
			Citations: syn.Citations,
		},
		Abstained:          syn.Abstained,
		AbstainReason:      syn.AbstainReason,
		Excerpts:           excerpts,
		DegradedEngines:    candSet.DegradedEngines,
		EvidenceReceiptIDs: receiptIDs,
		FetchFailures:      failures,
	}
	if !syn.Abstained {
		for i := range syn.Citations {
			idx := syn.Citations[i].ResultIndex
			if idx >= 0 && idx < len(docs) && docs[idx].ReceiptID != "" && s.receiptStore != nil {
				if receipt, err := s.receiptStore.GetReceipt(ctx, docs[idx].ReceiptID); err == nil {
					syn.Citations[i].RetrievedAt = receipt.RetrievedAt
				}
			}
		}
		if len(questions) > 0 {
			out.Assessments = questionAssessments
			out.Coverage = EvaluateQuestionCoverage(questions, out.Assessments, claimsByQuestion)
		} else {
			out.Assessments = assessSynthesis(syn, docs)
		}
	}
	if len(questions) > 0 && len(out.Coverage) == 0 {
		out.Coverage = EvaluateQuestionCoverage(questions, out.Assessments, claimsByQuestion)
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

// synthesizeQuestions answers each caller-declared question independently over
// the same retained documents. A single combined model response can cite a
// source while silently omitting one requested part; separate passes keep the
// coverage denominator authoritative and let one supported part survive beside
// an explicitly unresolved part.
func (s *Service) synthesizeQuestions(ctx context.Context, questions []ResearchQuestion, docs []Document) (Synthesis, []ClaimAssessment, map[string][]string, error) {
	var summary strings.Builder
	var citations []Citation
	var assessments []ClaimAssessment
	claimsByQuestion := make(map[string][]string, len(questions))
	allAbstained := true
	for _, question := range questions {
		syn, err := s.synthesizer.Synthesize(ctx, question.Prompt, docs)
		if err != nil {
			return Synthesis{}, nil, nil, err
		}
		if syn.Abstained || strings.TrimSpace(syn.Text) == "" {
			continue
		}
		allAbstained = false
		questionAssessments := assessSynthesisWithPrefix(syn, docs, "question-"+question.ID)
		for _, assessment := range questionAssessments {
			claimsByQuestion[question.ID] = append(claimsByQuestion[question.ID], assessment.ClaimID)
		}
		assessments = append(assessments, questionAssessments...)
		if summary.Len() > 0 {
			summary.WriteString("\n")
		}
		fmt.Fprintf(&summary, "%s: %s", question.Prompt, strings.TrimSpace(syn.Text))
		citations = append(citations, syn.Citations...)
	}
	if allAbstained {
		return AbstainWith(ReasonModelAbstained), nil, claimsByQuestion, nil
	}
	return Synthesis{Text: summary.String(), Citations: citations}, assessments, claimsByQuestion, nil
}

func assessSynthesis(syn Synthesis, docs []Document) []ClaimAssessment {
	return assessSynthesisWithPrefix(syn, docs, "claim")
}

func assessSynthesisWithPrefix(syn Synthesis, docs []Document, prefix string) []ClaimAssessment {
	assessments := make([]ClaimAssessment, 0, len(syn.Citations))
	for i, citation := range syn.Citations {
		if citation.ResultIndex < 0 || citation.ResultIndex >= len(docs) {
			continue
		}
		doc := docs[citation.ResultIndex]
		assessment := ClaimAssessment{ClaimID: fmt.Sprintf("%s-%d", prefix, i), Disposition: AssessClaimSupport(syn.Text, []string{doc.Text})}
		if doc.ReceiptID != "" || doc.PassageID != "" {
			assessment.Evidence = []EvidencePassageRef{{ReceiptID: doc.ReceiptID, PassageID: doc.PassageID, ContentHash: evidence.ContentHash([]byte(doc.Text)), ExtractionRevision: "readable-text-v1"}}
		}
		if len(assessment.Evidence) == 0 {
			assessment.Reason = "no_retained_passage"
			assessment.Disposition = AssessmentUnknown
		}
		assessments = append(assessments, assessment)
	}
	return assessments
}

type fetchedDocument struct {
	doc     Document
	receipt string
	failure *FetchFailure
}

func (s *Service) collectDocuments(ctx context.Context, cands []Candidate, maxEvidenceBytes int) ([]Document, []string, []FetchFailure) {
	if len(cands) == 0 {
		return nil, nil, nil
	}
	workers := s.fetchConcurrency
	if workers <= 0 || workers > len(cands) {
		workers = len(cands)
	}
	jobs := make(chan int)
	results := make([]fetchedDocument, len(cands))
	globalPermits := make(chan struct{}, s.fetchConcurrency)
	hostPermits := map[string]chan struct{}{}
	var hostMu sync.Mutex
	hostPermit := func(rawURL string) chan struct{} {
		host := strings.ToLower(rawURL)
		if parsed, err := url.Parse(rawURL); err == nil && parsed.Hostname() != "" {
			host = strings.ToLower(parsed.Hostname())
		}
		hostMu.Lock()
		defer hostMu.Unlock()
		if permit, ok := hostPermits[host]; ok {
			return permit
		}
		permit := make(chan struct{}, s.fetchPerHostConcurrency)
		hostPermits[host] = permit
		return permit
	}
	acquire := func(permit chan struct{}) bool {
		select {
		case permit <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				index, ok := <-jobs
				if !ok {
					return
				}
				candidate := cands[index]
				if !acquire(globalPermits) {
					results[index].failure = &FetchFailure{URL: candidate.URL, Code: "cancelled", Message: "fetch cancelled before admission", Retryable: false}
					continue
				}
				permit := hostPermit(candidate.URL)
				if !acquire(permit) {
					<-globalPermits
					results[index].failure = &FetchFailure{URL: candidate.URL, Code: "cancelled", Message: "fetch cancelled while queued for host", Retryable: false}
					continue
				}
				// Both the permit send and ctx.Done can become ready together.
				// Recheck after admission so cancellation never turns queued work
				// into a network request merely because select chose the permit.
				if ctx.Err() != nil {
					<-permit
					<-globalPermits
					results[index].failure = &FetchFailure{URL: candidate.URL, Code: "cancelled", Message: "fetch cancelled after admission", Retryable: false}
					continue
				}
				text, receiptID, err := s.fetchCandidate(ctx, candidate)
				<-permit
				<-globalPermits
				if err != nil {
					results[index].failure = &FetchFailure{URL: candidate.URL, ReceiptID: receiptID, Code: "fetch_failed", Message: err.Error(), Retryable: ctx.Err() == nil}
					continue
				}
				if strings.TrimSpace(text) == "" {
					results[index].failure = &FetchFailure{URL: candidate.URL, Code: "empty_content", Message: "fetch returned no readable text"}
					continue
				}
				results[index] = fetchedDocument{doc: Document{URL: candidate.URL, Title: candidate.Title, Text: text, ReceiptID: receiptID}, receipt: receiptID}
			}
		}()
	}
enqueue:
	for i := range cands {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break enqueue
		}
	}
	close(jobs)
	wg.Wait()
	var docs []Document
	var receipts []string
	var failures []FetchFailure
	if maxEvidenceBytes <= 0 {
		maxEvidenceBytes = 4 << 20
	}
	usedBytes := 0
	for _, result := range results {
		if result.failure != nil {
			failures = append(failures, *result.failure)
			if result.failure.ReceiptID != "" {
				receipts = append(receipts, result.failure.ReceiptID)
			}
			continue
		}
		remaining := maxEvidenceBytes - usedBytes
		if remaining <= 0 {
			failures = append(failures, FetchFailure{URL: result.doc.URL, ReceiptID: result.doc.ReceiptID, Code: "evidence_limit", Message: "retained evidence byte budget exhausted", Retryable: false})
			if result.doc.ReceiptID != "" {
				receipts = append(receipts, result.doc.ReceiptID)
			}
			continue
		}
		if len([]byte(result.doc.Text)) > remaining {
			result.doc.Text = boundedUTF8Prefix(result.doc.Text, remaining)
			failures = append(failures, FetchFailure{URL: result.doc.URL, ReceiptID: result.doc.ReceiptID, Code: "evidence_limit", Message: "source content was bounded by the request evidence byte budget", Retryable: false})
		}
		usedBytes += len([]byte(result.doc.Text))
		if result.doc.ReceiptID != "" {
			if passage, err := s.receiptStore.CreatePassage(ctx, result.doc.ReceiptID, 0, len([]byte(result.doc.Text))); err == nil {
				result.doc.PassageID = passage.PassageID
			}
			receipts = append(receipts, result.doc.ReceiptID)
		}
		docs = append(docs, result.doc)
	}
	return docs, receipts, failures
}

func boundedUTF8Prefix(text string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len([]byte(text)) <= maxBytes {
		return text
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

func (s *Service) fetchCandidate(ctx context.Context, candidate Candidate) (string, string, error) {
	if observed, ok := s.fetcher.(ObservationFetcher); ok && s.receiptStore != nil {
		observation, err := observed.FetchObservation(ctx, candidate.URL)
		if err != nil {
			return "", s.failureReceipt(ctx, candidate.URL, err), err
		}
		receipt, err := s.receiptStore.CreateReceipt(ctx, observation)
		if err != nil {
			return "", s.failureReceipt(ctx, candidate.URL, err), err
		}
		return string(observation.Content), receipt.ReceiptID, nil
	}
	text, err := s.fetcher.Fetch(ctx, candidate.URL)
	if err != nil && s.receiptStore != nil {
		return "", s.failureReceipt(ctx, candidate.URL, err), err
	}
	return text, "", err
}

func (s *Service) failureReceipt(ctx context.Context, rawURL string, cause error) string {
	if s.receiptStore == nil {
		return ""
	}
	receipt, err := s.receiptStore.CreateReceipt(ctx, evidence.NewObservation{URL: rawURL, RetrievedAt: s.now().UTC(), ExtractionRevision: "fetch-failure-v1", Retention: "metadata-only", FailureCode: "fetch_failed"})
	if err != nil {
		s.logger.Printf("research: failed to persist fetch failure for %q: %v (original: %v)", rawURL, err, cause)
		return ""
	}
	return receipt.ReceiptID
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
		cites = append(cites, findings.NewCitation{URL: c.URL, Title: c.Title, RetrievedAt: c.RetrievedAt})
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

func (l LiveSearcher) FreshCandidates(ctx context.Context, query string, n int) (CandidateSet, error) {
	l.Fresh = true
	return l.Candidates(ctx, query, n)
}
