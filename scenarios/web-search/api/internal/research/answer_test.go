package research_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"web-search/internal/findings"
	"web-search/internal/research"
)

type evidenceStore struct{ rows map[string]findings.Finding }

func (e evidenceStore) GetMany(context.Context, []string) (map[string]findings.Finding, error) {
	return e.rows, nil
}

// [REQ:REQ-P0-009] Reuse exact selected evidence without making a live call.
func TestAnswerStoredPolicy(t *testing.T) {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	good := findings.Finding{ID: "f", Query: "q", Claim: "supported claim", Confidence: 0.8, Status: findings.StatusActive, RetrievalDate: now.Add(-time.Hour), Citations: []findings.Citation{{URL: "https://docs.example.org/reference", Title: "Reference"}}}
	tests := []struct {
		name   string
		mutate func(*findings.Finding)
		maxAge time.Duration
		kind   string
	}{
		{"fresh", func(*findings.Finding) {}, 2 * time.Hour, "stored_finding"},
		{"current requires live", func(*findings.Finding) {}, 0, "none"},
		{"stale", func(f *findings.Finding) { f.RetrievalDate = now.Add(-3 * time.Hour) }, 2 * time.Hour, "none"},
		{"unknown date", func(f *findings.Finding) { f.RetrievalDate = time.Time{} }, 2 * time.Hour, "none"},
		{"future date", func(f *findings.Finding) { f.RetrievalDate = now.Add(time.Hour) }, 2 * time.Hour, "none"},
		{"disputed", func(f *findings.Finding) { f.Status = findings.StatusDisputed }, 2 * time.Hour, "none"},
		{"invalid confidence", func(f *findings.Finding) { f.Confidence = math.NaN() }, 2 * time.Hour, "none"},
		{"wrong source", func(f *findings.Finding) {
			f.Citations = []findings.Citation{{URL: "https://example.org.attacker.invalid"}}
		}, 2 * time.Hour, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := good
			tt.mutate(&f)
			search := &fakeSearcher{err: errors.New("offline")}
			svc := research.NewService(research.Deps{EvidenceStore: evidenceStore{map[string]findings.Finding{"f": f}}, Searcher: search, Fetcher: &fakeFetcher{}, Synthesizer: &fakeSynthesizer{}, Now: func() time.Time { return now }})
			out, err := svc.Answer(context.Background(), research.EvidencePolicy{Query: "q", FindingID: "f", MaxAge: tt.maxAge, SourceDomains: []string{"example.org"}})
			require.NoError(t, err)
			require.Equal(t, tt.kind, out.Kind)
			if tt.kind == "stored_finding" {
				require.Zero(t, out.LiveCalls)
				require.Empty(t, search.gotQuery)
				require.Equal(t, "supported claim", out.Brief.Summary)
				require.Equal(t, good.Citations[0].URL, out.Brief.Citations[0].URL)
			} else {
				require.Equal(t, "unavailable", out.Status)
			}
		})
	}
}

// [REQ:REQ-P0-009] Evidence limits apply before capture; source counts are distinct hosts.
func TestAnswerRejectsInsufficientSourcesBeforeCapture(t *testing.T) {
	store := newFindingsService(t)
	svc := research.NewService(research.Deps{Searcher: &fakeSearcher{candidates: []research.Candidate{{URL: "https://example.org/a", Title: "a"}}}, Fetcher: &fakeFetcher{textByURL: map[string]string{"https://example.org/a": "evidence"}}, Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: "claim", Citations: []research.Citation{{URL: "https://example.org/a"}, {URL: "https://example.org/b"}}}}, Findings: store})
	out, err := svc.Answer(context.Background(), research.EvidencePolicy{Query: "q", MinimumSources: 2, Capture: true})
	require.NoError(t, err)
	require.True(t, out.Abstained)
	require.Empty(t, out.Brief.Summary)
	require.Empty(t, out.CapturedIDs)
	require.Equal(t, "evidence_requirements_unmet", out.Reason)
}

func TestEvidencePolicyRejectsInvalidBounds(t *testing.T) {
	for _, p := range []research.EvidencePolicy{{Query: ""}, {Query: "q", MaxAge: -time.Second}, {Query: "q", MinimumSources: 11}, {Query: "q", SourceDomains: []string{"https://example.org"}}, {Query: "q", Effort: "l3"}} {
		require.Error(t, p.Validate())
	}
}

// [REQ:REQ-P0-009] Oversized model output is unresolved and is never captured.
func TestAnswerBoundsBeforeCapture(t *testing.T) {
	store := newFindingsService(t)
	svc := research.NewService(research.Deps{Searcher: &fakeSearcher{candidates: []research.Candidate{{URL: "https://example.org/a"}}}, Fetcher: &fakeFetcher{textByURL: map[string]string{"https://example.org/a": "evidence"}}, Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: strings.Repeat("x", 10001), Citations: []research.Citation{{URL: "https://example.org/a"}}}}, Findings: store})
	out, err := svc.Answer(context.Background(), research.EvidencePolicy{Query: "q", Capture: true})
	require.NoError(t, err)
	require.True(t, out.Abstained)
	require.Empty(t, out.Brief.Summary)
	require.Empty(t, out.CapturedIDs)
	require.Equal(t, "answer_output_limit", out.Reason)
}

// [REQ:REQ-P0-009] A successful answer cannot silently satisfy an unavailable capture.
func TestAnswerReportsUnavailableCapture(t *testing.T) {
	svc := research.NewService(research.Deps{Searcher: &fakeSearcher{candidates: []research.Candidate{{URL: "https://example.org/a"}}}, Fetcher: &fakeFetcher{textByURL: map[string]string{"https://example.org/a": "evidence"}}, Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: "supported", Citations: []research.Citation{{URL: "https://example.org/a"}}}}})
	out, err := svc.Answer(context.Background(), research.EvidencePolicy{Query: "q", Capture: true})
	require.NoError(t, err)
	require.Equal(t, "partial", out.Status)
	require.Equal(t, "supported", out.Brief.Summary)
	require.Contains(t, out.Gaps, "capture_unavailable")
}
