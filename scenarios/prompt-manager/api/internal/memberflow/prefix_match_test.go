package memberflow

import (
	"errors"
	"testing"
)

func TestEnrichWithKeyPrefixMismatch_NilQueryReturnsNil(t *testing.T) {
	got := EnrichWithKeyPrefixMismatch([]MemberTopics{
		mt("alpha", "researcher", IntakeEntry{Prefix: "research-inbox/*", Taxonomy: "marketing-research"}),
	}, nil)
	if got != nil {
		t.Fatalf("expected nil findings when query is nil, got %v", got)
	}
}

func TestEnrichWithKeyPrefixMismatch_AllEntriesCovered(t *testing.T) {
	members := []MemberTopics{
		{
			Ref:    MemberRef{Team: "alpha", Member: "researcher"},
			Exists: true,
			Topics: Topics{
				Intake: []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "marketing-research"}},
				Output: []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: "knowledge"}},
			},
		},
	}
	q := stubKnowledgeQuery{
		allByTeam: map[string][]InboxEntry{
			"alpha": {
				{ID: "knw-1", Topic: "research-inbox/audience/foo"},
				{ID: "knw-2", Topic: "audience-scan/2026-04-23"},
			},
		},
	}
	findings := EnrichWithKeyPrefixMismatch(members, q)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings, got %d: %+v", len(findings), findings)
	}
}

func TestEnrichWithKeyPrefixMismatch_FlagsUndeclaredEntry(t *testing.T) {
	members := []MemberTopics{
		{
			Ref:    MemberRef{Team: "alpha", Member: "researcher"},
			Exists: true,
			Topics: Topics{
				Output: []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: "knowledge"}},
			},
		},
	}
	q := stubKnowledgeQuery{
		allByTeam: map[string][]InboxEntry{
			"alpha": {
				{ID: "knw-1", Topic: "audience-scan/foo"},     // declared
				{ID: "knw-2", Topic: "competitor-record/bar"}, // NOT declared
				{ID: "knw-3", Topic: "audience-scan-flat-1"},  // would be flat-form, not slash-form
			},
		},
	}
	findings := EnrichWithKeyPrefixMismatch(members, q)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (competitor + flat), got %d: %+v", len(findings), findings)
	}
	for _, f := range findings {
		if f.Rule != "topic_key_prefix_mismatch" || f.Severity != SeverityWarning {
			t.Errorf("unexpected finding shape: %+v", f)
		}
		if f.Team != "alpha" {
			t.Errorf("expected team=alpha, got %q", f.Team)
		}
	}
}

func TestEnrichWithKeyPrefixMismatch_ScopesByTeam(t *testing.T) {
	members := []MemberTopics{
		{
			Ref:    MemberRef{Team: "alpha", Member: "a"},
			Exists: true,
			Topics: Topics{Output: []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: "knowledge"}}},
		},
		{
			Ref:    MemberRef{Team: "beta", Member: "b"},
			Exists: true,
			Topics: Topics{Output: []OutputEntry{{Prefix: "competitor-record/*", DestinationKind: "knowledge"}}},
		},
	}
	q := stubKnowledgeQuery{
		allByTeam: map[string][]InboxEntry{
			// "competitor-record/x" is declared in team beta but NOT in team alpha.
			// alpha-side it should fire; beta-side it's fine.
			"alpha": {{ID: "knw-1", Topic: "competitor-record/x"}},
			"beta":  {{ID: "knw-2", Topic: "competitor-record/y"}},
		},
	}
	findings := EnrichWithKeyPrefixMismatch(members, q)
	if len(findings) != 1 {
		t.Fatalf("expected 1 cross-team-isolation finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Team != "alpha" {
		t.Errorf("expected finding scoped to alpha, got %q", findings[0].Team)
	}
}

// A member that declares its output with the `<name>` notation TOPICS_SCHEMA.md
// teaches by example has declared its prefix. Before segment-wise matching, the
// declaration matched nothing and every entry it wrote was reported as drift —
// and the finding's own remedy text ("declare it on topics.json") would have led
// the team to replace a precise declaration with a wildcard to silence a bug.
func TestEnrichWithKeyPrefixMismatch_PlaceholderDeclarationCoversItsKeys(t *testing.T) {
	members := []MemberTopics{
		{
			Ref:    MemberRef{Team: "alpha", Member: "contrarian"},
			Exists: true,
			Topics: Topics{
				Output: []OutputEntry{
					{Prefix: "review-evidence/<work-item-id>", DestinationKind: "knowledge"},
					{Prefix: "friction-report/<scope>/<date>/<slug>", DestinationKind: "knowledge"},
				},
			},
		},
	}
	q := stubKnowledgeQuery{
		allByTeam: map[string][]InboxEntry{
			"alpha": {
				{ID: "knw-1", Topic: "review-evidence/work-1778803361775636366"},
				{ID: "knw-2", Topic: "friction-report/toolchain/2026-07-31/slow-build"},
			},
		},
	}
	if findings := EnrichWithKeyPrefixMismatch(members, q); len(findings) != 0 {
		t.Fatalf("placeholder declarations should cover their own keys, got %d findings: %+v",
			len(findings), findings)
	}
}

// The placeholder must not become a blanket wildcard: it binds exactly one
// segment, so a key at a different depth or under a different root still drifts.
func TestEnrichWithKeyPrefixMismatch_PlaceholderIsNotABlanketWildcard(t *testing.T) {
	members := []MemberTopics{
		{
			Ref:    MemberRef{Team: "alpha", Member: "contrarian"},
			Exists: true,
			Topics: Topics{
				Output: []OutputEntry{{Prefix: "review-evidence/<work-item-id>", DestinationKind: "knowledge"}},
			},
		},
	}
	q := stubKnowledgeQuery{
		allByTeam: map[string][]InboxEntry{
			"alpha": {{ID: "knw-1", Topic: "quality-audit/dec-42"}},
		},
	}
	findings := EnrichWithKeyPrefixMismatch(members, q)
	if len(findings) != 1 {
		t.Fatalf("expected the unrelated root to still be reported, got %d: %+v", len(findings), findings)
	}
	if findings[0].Rule != "topic_key_prefix_mismatch" {
		t.Errorf("expected topic_key_prefix_mismatch, got %q", findings[0].Rule)
	}
}

func TestEnrichWithKeyPrefixMismatch_QueryError(t *testing.T) {
	q := stubKnowledgeQuery{err: errors.New("backend down")}
	members := []MemberTopics{{Ref: MemberRef{Team: "alpha"}, Exists: true}}
	findings := EnrichWithKeyPrefixMismatch(members, q)
	if len(findings) != 1 {
		t.Fatalf("expected 1 unavailable finding, got %d", len(findings))
	}
	if findings[0].Rule != "topic_key_query_unavailable" {
		t.Errorf("expected rule topic_key_query_unavailable, got %q", findings[0].Rule)
	}
}
