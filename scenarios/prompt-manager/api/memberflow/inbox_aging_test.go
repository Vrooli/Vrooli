package memberflow

import (
	"errors"
	"testing"
	"time"
)

type stubKnowledgeQuery struct {
	byPrefix map[string][]InboxEntry
	allByTeam map[string][]InboxEntry
	err      error
}

func (s stubKnowledgeQuery) ListUnrouted(_ string, prefix string) ([]InboxEntry, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byPrefix[prefix], nil
}

func (s stubKnowledgeQuery) ListAll(team string) ([]InboxEntry, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.allByTeam[team], nil
}

func mt(team, member string, intake ...IntakeEntry) MemberTopics {
	return MemberTopics{
		Ref:    MemberRef{Team: team, Member: member},
		Exists: true,
		Topics: Topics{Intake: intake},
	}
}

func TestEnrichWithDrainStatus_NilQueryReturnsNothing(t *testing.T) {
	got := EnrichWithDrainStatus(nil, nil, InboxAgingOptions{})
	if len(got) != 0 {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestEnrichWithDrainStatus_NoEntriesIsClean(t *testing.T) {
	members := []MemberTopics{
		mt("t", "a", IntakeEntry{Prefix: "research-inbox/*", Taxonomy: "tx"}),
	}
	q := stubKnowledgeQuery{byPrefix: map[string][]InboxEntry{}}
	got := EnrichWithDrainStatus(members, q, InboxAgingOptions{})
	if len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

func TestEnrichWithDrainStatus_PilingInbox(t *testing.T) {
	members := []MemberTopics{
		mt("t", "a", IntakeEntry{Prefix: "research-inbox/*", Taxonomy: "tx"}),
	}
	entries := make([]InboxEntry, 6)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	for i := range entries {
		entries[i] = InboxEntry{ID: "e", Topic: "research-inbox/audience/foo", At: now}
	}
	q := stubKnowledgeQuery{byPrefix: map[string][]InboxEntry{"research-inbox/*": entries}}

	findings := EnrichWithDrainStatus(members, q, InboxAgingOptions{PilingAt: 5, Now: now})
	if len(findings) != 1 || findings[0].Rule != "piling_inbox" {
		t.Fatalf("expected one piling_inbox finding, got %+v", findings)
	}
	if findings[0].Severity != SeverityWarning {
		t.Fatalf("expected warning, got %s", findings[0].Severity)
	}
}

func TestEnrichWithDrainStatus_StalledDrain(t *testing.T) {
	members := []MemberTopics{
		mt("t", "a", IntakeEntry{Prefix: "research-inbox/*", Taxonomy: "tx"}),
	}
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	old := now.Add(-10 * 24 * time.Hour)
	entries := []InboxEntry{
		{ID: "1", At: old},
		{ID: "2", At: now.Add(-2 * time.Hour)},
	}
	q := stubKnowledgeQuery{byPrefix: map[string][]InboxEntry{"research-inbox/*": entries}}
	findings := EnrichWithDrainStatus(members, q, InboxAgingOptions{Now: now})

	var stalled int
	for _, f := range findings {
		if f.Rule == "stalled_drain" {
			stalled++
		}
	}
	if stalled != 1 {
		t.Fatalf("expected exactly one stalled_drain finding, got %d (%+v)", stalled, findings)
	}
}

func TestEnrichWithDrainStatus_BothRulesAtOnce(t *testing.T) {
	members := []MemberTopics{
		mt("t", "a", IntakeEntry{Prefix: "p/*", Taxonomy: "tx"}),
	}
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	old := now.Add(-30 * 24 * time.Hour)
	entries := make([]InboxEntry, 60)
	for i := range entries {
		entries[i] = InboxEntry{ID: "e", At: old}
	}
	q := stubKnowledgeQuery{byPrefix: map[string][]InboxEntry{"p/*": entries}}
	findings := EnrichWithDrainStatus(members, q, InboxAgingOptions{Now: now})

	rules := map[string]int{}
	for _, f := range findings {
		rules[f.Rule]++
	}
	if rules["piling_inbox"] != 1 || rules["stalled_drain"] != 1 {
		t.Fatalf("expected one each of piling_inbox and stalled_drain, got %+v", rules)
	}
}

func TestEnrichWithDrainStatus_QueryErrorBecomesWarning(t *testing.T) {
	members := []MemberTopics{
		mt("t", "a", IntakeEntry{Prefix: "p/*", Taxonomy: "tx"}),
	}
	q := stubKnowledgeQuery{err: errors.New("boom")}
	findings := EnrichWithDrainStatus(members, q, InboxAgingOptions{})
	if len(findings) != 1 || findings[0].Rule != "drain_status_unavailable" {
		t.Fatalf("expected drain_status_unavailable, got %+v", findings)
	}
}

func TestMergeFindings_AppendsAndCounts(t *testing.T) {
	r := ValidationResult{
		Findings: []Finding{{Rule: "orphan_input", Severity: SeverityError}},
		Errors:   1,
	}
	extra := []Finding{
		{Rule: "piling_inbox", Severity: SeverityWarning, Prefix: "p/*"},
		{Rule: "stalled_drain", Severity: SeverityWarning, Prefix: "p/*"},
	}
	merged := MergeFindings(r, extra)
	if merged.Errors != 1 || merged.Warnings != 2 {
		t.Fatalf("counts wrong: errs=%d warns=%d", merged.Errors, merged.Warnings)
	}
	if len(merged.Findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(merged.Findings))
	}
}

func TestMergeFindings_NoExtraNoChange(t *testing.T) {
	r := ValidationResult{Findings: []Finding{{Rule: "orphan_input"}}, Errors: 1}
	merged := MergeFindings(r, nil)
	if len(merged.Findings) != 1 || merged.Errors != 1 {
		t.Fatalf("unexpected merge: %+v", merged)
	}
}
