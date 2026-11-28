package routing

import (
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestBuildCorrectionCandidates(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		response  *routingv1.QueryResponse
		wantCount int
		wantValue string
	}{
		{
			name:  "exact token is not a correction",
			query: "SearchHub",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "docs", Id: "searchhub", Title: "SearchHub",
			}}},
		},
		{
			name:  "unique typo candidate preserves submitted casing",
			query: "Managr",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "skills", Id: "manager-skill", Title: "Manager Skill", Path: "skills/manager",
			}}},
			wantCount: 1, wantValue: "manager",
		},
		{
			name:  "ambiguous candidates in one provider are suppressed",
			query: "searchhup",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "docs", Id: "searchhub", Title: "searchcup",
			}}},
		},
		{
			name:  "short token and stopword are suppressed",
			query: "the ux",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "docs", Id: "ui-guide", Title: "the user interface",
			}}},
		},
		{
			name:  "snippet text cannot supply correction evidence",
			query: "Managr",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "docs", Id: "guide", Snippet: "manager setup steps",
			}}},
		},
		{
			name:  "search is suppressed as a correction stopword",
			query: "searxh",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{{
				ProviderId: "docs", Id: "search-guide", Title: "Search Guide",
			}}},
		},
		{
			name:  "duplicate identity evidence dedupes",
			query: "Managr",
			response: &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{
				{ProviderId: "skills", Id: "manager-skill", Title: "Manager Skill", Path: "skills/manager"},
				{ProviderId: "skills", Id: "manager-skill", Title: "Manager Skill", Path: "skills/manager"},
			}},
			wantCount: 1, wantValue: "manager",
		},
		{
			name:  "competing provider candidates are suppressed",
			query: "searchhup",
			response: &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{
				{ProviderId: "docs", Hits: []*routingv1.SearchHit{{Id: "searchhub"}}},
				{ProviderId: "skills", Hits: []*routingv1.SearchHit{{Id: "searchcup"}}},
			}},
		},
		{
			name:  "same candidate across providers retains provider evidence",
			query: "Managr",
			response: &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{
				{ProviderId: "docs", Hits: []*routingv1.SearchHit{{Id: "manager-doc"}}},
				{ProviderId: "skills", Hits: []*routingv1.SearchHit{{Id: "manager-skill"}}},
			}},
			wantCount: 2, wantValue: "manager",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildCorrectionCandidates(test.query, test.response)
			if len(got) != test.wantCount {
				t.Fatalf("candidate count = %d, want %d: %+v", len(got), test.wantCount, got)
			}
			for _, candidate := range got {
				if candidate.GetCandidateValue() != test.wantValue {
					t.Errorf("candidate value = %q, want %q", candidate.GetCandidateValue(), test.wantValue)
				}
				if test.wantCount > 0 && candidate.GetSubmittedValue() != "Managr" {
					t.Errorf("submitted value = %q, want original token casing", candidate.GetSubmittedValue())
				}
				if candidate.GetStrength() != routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE {
					t.Errorf("strength = %s, want possible", candidate.GetStrength())
				}
				if candidate.GetProviderId() == "" || candidate.GetEvidence() == nil {
					t.Errorf("candidate is missing concrete provider evidence: %+v", candidate)
				}
			}
			if test.name == "same candidate across providers retains provider evidence" {
				if got[0].GetProviderId() != "docs" || got[1].GetProviderId() != "skills" {
					t.Errorf("provider evidence order = %q, %q; want docs then skills", got[0].GetProviderId(), got[1].GetProviderId())
				}
			}
		})
	}
}

func TestBuildCorrectionCandidatesUsesGroupHitReference(t *testing.T) {
	response := &routingv1.QueryResponse{Groups: []*routingv1.ProviderResultGroup{{
		ProviderId: "docs", Hits: []*routingv1.SearchHit{{Id: "manager-doc", Title: "manager"}},
	}}}
	got := buildCorrectionCandidates("Managr", response)
	if len(got) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(got))
	}
	ref := got[0].GetEvidence()
	if ref.GetProviderId() != "docs" || ref.GetHitIndex() != 0 || ref.GetGroupIndex() != 0 {
		t.Fatalf("evidence reference = %+v, want group 0 hit 0 from docs", ref)
	}
}
