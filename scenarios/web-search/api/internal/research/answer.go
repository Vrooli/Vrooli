package research

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"web-search/internal/findings"
	"web-search/internal/livesearch"
)

// EvidencePolicy belongs to research, so all callers share freshness and source rules.
// A zero age requests live evidence. Reuse requires exact query identity or an
// explicitly selected finding; semantic similarity alone is not answer sufficiency.
type EvidencePolicy struct {
	Query          string
	Effort         string
	MaxAge         time.Duration
	SourceDomains  []string
	MinimumSources int
	TopN           int
	Capture        bool
	FindingID      string
	// MaxEvidenceBytes bounds retained/source content admitted to one request.
	// Zero means the documented server default and is resolved by the caller.
	MaxEvidenceBytes int
	Questions        []ResearchQuestion
	// ParentRunID groups this child research attempt under an owning L3 run.
	ParentRunID string
}

type AnswerOutcome struct {
	Status, Kind, Reason          string
	Brief                         Brief
	Results                       []livesearch.Result
	FindingIDs, CapturedIDs, Gaps []string
	Abstained, Cached             bool
	CheckedAt                     time.Time
	LiveCalls                     int
	Assessments                   []ClaimAssessment
	Coverage                      []QuestionCoverage
}

type LiveSearch interface {
	Search(context.Context, livesearch.SearchInput) (livesearch.SearchOutcome, error)
}

func (p *EvidencePolicy) Validate() error {
	p.Query = strings.TrimSpace(p.Query)
	p.applyDefaults()
	if err := ValidateContractBounds(p.Query, p.Questions, *p); err != nil {
		return err
	}
	if !map[string]bool{"l0": true, "l1": true, "l2": true}[p.Effort] {
		return fmt.Errorf("effort must be l0, l1, or l2")
	}
	if err := p.validateBounds(); err != nil {
		return err
	}
	return p.normalizeDomains()
}

func (p *EvidencePolicy) applyDefaults() {
	if p.Effort == "" {
		p.Effort = "l2"
	}
	if p.MinimumSources == 0 {
		p.MinimumSources = 1
	}
	if p.TopN == 0 {
		p.TopN = DefaultTopN
	}
}

func (p EvidencePolicy) validateBounds() error {
	if p.MaxAge < 0 || p.MaxAge > 180*24*time.Hour {
		return fmt.Errorf("max age must be within 0..180 days")
	}
	if p.MinimumSources < 1 || p.MinimumSources > 10 || p.TopN < 1 || p.TopN > 10 {
		return fmt.Errorf("source and page limits must be within 1..10")
	}
	if len(p.SourceDomains) > 10 {
		return fmt.Errorf("at most ten source domains are allowed")
	}
	return nil
}

func (p *EvidencePolicy) normalizeDomains() error {
	p.SourceDomains = append([]string(nil), p.SourceDomains...)
	for i, d := range p.SourceDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" || strings.ContainsAny(d, "/:@ ?#*\r\n\t") || !strings.Contains(d, ".") {
			return fmt.Errorf("source domain must be a hostname")
		}
		p.SourceDomains[i] = d
	}
	return nil
}

func allowedSource(raw string, domains []string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		return false
	}
	if len(domains) == 0 {
		return true
	}
	for _, d := range domains {
		if strings.EqualFold(u.Hostname(), d) || strings.HasSuffix(strings.ToLower(u.Hostname()), "."+d) {
			return true
		}
	}
	return false
}

func supportedSources(citations []Citation, domains []string) int {
	urls := make([]string, 0, len(citations))
	for _, c := range citations {
		if len(c.URL) > 2048 || len(c.Title) > 300 || !allowedSource(c.URL, domains) {
			return 0
		}
		urls = append(urls, c.URL)
	}
	return IndependentPublisherCount(urls)
}

func (s *Service) Answer(ctx context.Context, p EvidencePolicy) (out AnswerOutcome, err error) {
	if err = p.Validate(); err != nil {
		return out, err
	}
	defer func() {
		if err == nil {
			out = boundAnswer(out)
		}
	}()
	var reused bool
	out, reused = s.storedAnswer(ctx, p, s.now().UTC())
	if reused {
		return out, nil
	}
	out.LiveCalls = 1
	if err = s.collectLiveAnswer(ctx, p, &out); err != nil {
		out.Status = "unavailable"
		out.Reason = "live_research_unavailable"
		out.Gaps = append(out.Gaps, out.Reason)
		return out, nil
	}
	return s.finishAnswer(ctx, p, out), nil
}

func boundAnswer(out AnswerOutcome) AnswerOutcome {
	payload, err := json.Marshal(out)
	if err != nil || len(payload) > 48000 {
		out.Brief.Summary = ""
		out.Brief.Citations = nil
		out.Results = nil
		out.Status = "partial"
		out.Kind = "none"
		out.Abstained = true
		out.Reason = "answer_output_limit"
		out.Gaps = append(out.Gaps, out.Reason)
	}
	return out
}

func (s *Service) storedAnswer(ctx context.Context, p EvidencePolicy, now time.Time) (AnswerOutcome, bool) {
	out := AnswerOutcome{Status: "ok", Kind: "none", Brief: Brief{Query: p.Query, Level: p.Effort}, CheckedAt: now}
	if p.MaxAge == 0 || s.evidenceStore == nil {
		return out, false
	}
	ids, gaps := s.recallEvidenceIDs(ctx, p)
	out.Gaps = gaps
	rows, err := s.evidenceStore.GetMany(ctx, ids)
	if err != nil {
		out.Gaps = append(out.Gaps, "stored_evidence_unavailable")
	}
	for _, id := range ids {
		f, ok := rows[id]
		if !ok || !canReuseFinding(f, p, now) {
			continue
		}
		brief, ok := briefFromFinding(f, p)
		if !ok {
			continue
		}
		if len(p.Questions) > 0 {
			// Legacy findings have no per-question claim assessments, so they
			// cannot silently satisfy a newly typed multipart request.
			out.Coverage = EvaluateQuestionCoverage(p.Questions, nil, nil)
			continue
		}
		out.Kind = "stored_finding"
		out.Brief = brief
		out.FindingIDs = []string{f.ID}
		out.CheckedAt = f.RetrievalDate
		return out, true
	}
	return out, false
}

func (s *Service) recallEvidenceIDs(ctx context.Context, p EvidencePolicy) ([]string, []string) {
	if p.FindingID != "" {
		return []string{p.FindingID}, nil
	}
	hits, _, err := s.GatherRelatedFindings(ctx, p.Query, MaxGatherFindings)
	var ids, gaps []string
	if err != nil {
		gaps = append(gaps, "stored_evidence_unavailable")
	}
	for _, hit := range hits {
		ids = append(ids, hit.FindingID)
	}
	return ids, gaps
}

func canReuseFinding(f findings.Finding, p EvidencePolicy, now time.Time) bool {
	if !trustedFinding(f, now, p.MaxAge) {
		return false
	}
	return p.FindingID != "" || strings.EqualFold(strings.TrimSpace(f.Query), p.Query)
}

func trustedFinding(f findings.Finding, now time.Time, maxAge time.Duration) bool {
	return f.Status == findings.StatusActive && !math.IsNaN(f.Confidence) && f.Confidence >= 0.5 && f.Confidence <= 1 && !f.RetrievalDate.IsZero() && !f.RetrievalDate.After(now) && now.Sub(f.RetrievalDate) <= maxAge
}

func briefFromFinding(f findings.Finding, p EvidencePolicy) (Brief, bool) {
	brief := Brief{Query: p.Query, Level: p.Effort, Summary: f.Claim}
	for _, c := range f.Citations {
		brief.Citations = append(brief.Citations, Citation{URL: c.URL, Title: c.Title})
	}
	if len(f.Claim) > 10000 || len(brief.Citations) > 10 || supportedSources(brief.Citations, p.SourceDomains) < p.MinimumSources || strings.TrimSpace(f.Claim) == "" {
		return Brief{}, false
	}
	return brief, true
}

func (s *Service) collectLiveAnswer(ctx context.Context, p EvidencePolicy, out *AnswerOutcome) error {
	if p.Effort == "l2" {
		return s.collectPageAnswer(ctx, p, out)
	}
	return s.collectSearchAnswer(ctx, p, out)
}

func (s *Service) collectPageAnswer(ctx context.Context, p EvidencePolicy, out *AnswerOutcome) error {
	scoped := *s
	scoped.searcher = domainSearcher{inner: s.searcher, domains: p.SourceDomains}
	// Answer owns the final capture decision after sufficiency checks; the
	// delegated L2 pass must never enqueue a finding before that gate.
	l2Policy := p
	l2Policy.Capture = false
	live, err := scoped.RunL2WithPolicyAndParent(ctx, l2Policy, p.ParentRunID)
	if err != nil {
		return err
	}
	out.Brief = live.Brief
	out.Abstained = live.Abstained
	out.Reason = string(live.AbstainReason)
	out.Assessments = live.Assessments
	out.Coverage = live.Coverage
	if len(live.DegradedEngines) > 0 {
		out.Gaps = append(out.Gaps, "upstream_engines_degraded")
	}
	return nil
}

func (s *Service) collectSearchAnswer(ctx context.Context, p EvidencePolicy, out *AnswerOutcome) error {
	if s.live == nil {
		return fmt.Errorf("live search unavailable")
	}
	live, err := s.live.Search(ctx, livesearch.SearchInput{Query: sourceQuery(p.Query, p.SourceDomains), Limit: p.TopN, Synthesize: p.Effort == "l1", Fresh: true})
	if err != nil {
		return err
	}
	out.Cached = live.Cached
	if len(live.DegradedEngines) > 0 {
		out.Gaps = append(out.Gaps, "upstream_engines_degraded")
	}
	for _, result := range live.Results {
		if allowedSource(result.URL, p.SourceDomains) {
			out.Results = append(out.Results, result)
		}
	}
	if live.Degraded {
		out.Status = "partial"
		out.Reason = live.DegradedReason
	}
	if live.Synthesis != nil {
		copySnippetSynthesis(out, live.Synthesis)
	}
	return nil
}

func copySnippetSynthesis(out *AnswerOutcome, syn *livesearch.Synthesis) {
	out.Brief.Summary = syn.Text
	out.Abstained = syn.Abstained
	for _, c := range syn.Citations {
		out.Brief.Citations = append(out.Brief.Citations, Citation{URL: c.URL, Title: c.Title, ResultIndex: c.ResultIndex})
	}
}

func (s *Service) finishAnswer(ctx context.Context, p EvidencePolicy, out AnswerOutcome) AnswerOutcome {
	if len(p.Questions) > 0 && len(out.Coverage) == 0 {
		out.Coverage = EvaluateQuestionCoverage(p.Questions, out.Assessments, nil)
	}
	if len(out.Brief.Summary) > 10000 || len(out.Brief.Citations) > 10 {
		out.Reason = "answer_output_limit"
	}
	if sufficientAnswer(out, p) && questionsCovered(p.Questions, out) {
		out.Kind = "cited_synthesis"
		s.captureAnswer(ctx, p, &out)
	} else {
		rejectAnswer(p, &out)
		if len(p.Questions) > 0 {
			out.Reason = "question_coverage_incomplete"
		}
	}
	if len(out.Gaps) > 0 && out.Status == "ok" {
		out.Status = "partial"
	}
	return out
}

func questionsCovered(questions []ResearchQuestion, out AnswerOutcome) bool {
	if len(questions) == 0 {
		return true
	}
	for _, question := range questions {
		found := false
		for _, coverage := range out.Coverage {
			if coverage.QuestionID == question.ID && coverage.Status == "supported" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func sufficientAnswer(out AnswerOutcome, p EvidencePolicy) bool {
	if out.Abstained || len(out.Brief.Summary) > 10000 || len(out.Brief.Citations) > 10 || strings.TrimSpace(out.Brief.Summary) == "" || supportedSources(out.Brief.Citations, p.SourceDomains) < p.MinimumSources {
		return false
	}
	for _, assessment := range out.Assessments {
		if assessment.Disposition != AssessmentSupported || len(assessment.Evidence) == 0 {
			return false
		}
	}
	return true
}

func (s *Service) captureAnswer(ctx context.Context, p EvidencePolicy, out *AnswerOutcome) {
	if !p.Capture || p.Effort != "l2" {
		return
	}
	if s.findings == nil {
		out.Gaps = append(out.Gaps, "capture_unavailable")
		return
	}
	ids, err := s.captureSynthesis(ctx, p.Query, Synthesis{Text: out.Brief.Summary, Citations: out.Brief.Citations})
	out.CapturedIDs = ids
	if err != nil {
		out.Gaps = append(out.Gaps, "capture_failed")
	}
}

func rejectAnswer(p EvidencePolicy, out *AnswerOutcome) {
	out.Brief.Summary = ""
	out.Brief.Citations = nil
	if len(out.Results) > 0 {
		out.Kind = "raw_hits"
	}
	if p.Effort != "l0" || len(out.Results) == 0 {
		out.Abstained = true
		if out.Reason == "" {
			out.Reason = "evidence_requirements_unmet"
		}
		out.Gaps = append(out.Gaps, out.Reason)
	}
}

type domainSearcher struct {
	inner   Searcher
	domains []string
}

func (d domainSearcher) Candidates(ctx context.Context, q string, n int) (CandidateSet, error) {
	if d.inner == nil {
		return CandidateSet{}, fmt.Errorf("search unavailable")
	}
	var out CandidateSet
	var err error
	if fresh, ok := d.inner.(interface {
		FreshCandidates(context.Context, string, int) (CandidateSet, error)
	}); ok {
		out, err = fresh.FreshCandidates(ctx, sourceQuery(q, d.domains), n)
	} else {
		out, err = d.inner.Candidates(ctx, sourceQuery(q, d.domains), n)
	}
	if err != nil {
		return out, err
	}
	filtered := out.Candidates[:0]
	for _, c := range out.Candidates {
		if allowedSource(c.URL, d.domains) {
			filtered = append(filtered, c)
		}
	}
	out.Candidates = filtered
	return out, nil
}

func sourceQuery(query string, domains []string) string {
	if len(domains) == 0 {
		return query
	}
	terms := make([]string, 0, len(domains))
	for _, d := range domains {
		terms = append(terms, "site:"+d)
	}
	return query + " (" + strings.Join(terms, " OR ") + ")"
}
