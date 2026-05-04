package memberflow

import "testing"

// Tests for consumerSet (consumer_set.go).
//
// These tests cover the abstraction's own contract — Overlaps semantics,
// buildConsumerSet contributor logic, and entry tagging. The integration
// with ruleOrphanOutput is covered by validation_test.go and is not
// re-verified here.

func TestConsumerSet_ZeroValueOverlapsNothing(t *testing.T) {
	var s consumerSet
	if s.Overlaps("anything/*") {
		t.Errorf("zero value consumerSet should not overlap any prefix")
	}
	if s.Overlaps("audience-scan/2026-04-23/example") {
		t.Errorf("zero value consumerSet should not overlap concrete prefix")
	}
}

func TestConsumerSet_OverlapsEmptyPrefixIsFalse(t *testing.T) {
	var s consumerSet
	s.add(MemberRef{Team: "t", Member: "m"}, "audience-scan/*", consumerSourceIntake)

	if s.Overlaps("") {
		t.Errorf("Overlaps(\"\") should be false even when set has entries")
	}
	if s.Overlaps("   \t\n  ") {
		t.Errorf("Overlaps with whitespace-only prefix should be false")
	}
}

func TestConsumerSet_AddRejectsEmptyPrefix(t *testing.T) {
	var s consumerSet
	s.add(MemberRef{Team: "t", Member: "m"}, "", consumerSourceIntake)
	s.add(MemberRef{Team: "t", Member: "m"}, "   ", consumerSourceIntake)

	if got := len(s.entries); got != 0 {
		t.Errorf("empty/whitespace prefixes should be dropped; entries=%d", got)
	}
}

func TestConsumerSet_OverlapsExactPrefix(t *testing.T) {
	var s consumerSet
	s.add(MemberRef{Team: "t", Member: "m"}, "audience-scan/2026-04-23", consumerSourceIntake)

	if !s.Overlaps("audience-scan/2026-04-23") {
		t.Errorf("identical exact prefixes should overlap")
	}
	if s.Overlaps("audience-scan/2026-04-24") {
		t.Errorf("disjoint exact prefixes should not overlap")
	}
}

func TestConsumerSet_OverlapsWildcardSemantics(t *testing.T) {
	cases := []struct {
		name       string
		registered string
		query      string
		want       bool
	}{
		{"registered wildcard covers exact query", "audience-scan/*", "audience-scan/2026-04-23/abc", true},
		{"registered exact matches wildcard query", "audience-scan/2026-04-23", "audience-scan/*", true},
		{"narrower registered overlaps wider query", "audience-scan/2026-04-23/*", "audience-scan/*", true},
		{"wider registered overlaps narrower query", "audience-scan/*", "audience-scan/2026-04-23/*", true},
		{"sibling wildcards do not overlap", "audience-scan/*", "monetization-benchmark/*", false},
		{"divergent inner segments do not overlap", "audience-scan/2026/*", "audience-scan/2027/*", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s consumerSet
			s.add(MemberRef{Team: "t", Member: "m"}, tc.registered, consumerSourceIntake)
			if got := s.Overlaps(tc.query); got != tc.want {
				t.Errorf("Overlaps(%q) with registered %q = %v, want %v",
					tc.query, tc.registered, got, tc.want)
			}
		})
	}
}

func TestConsumerSet_AggregatesMultipleMembers(t *testing.T) {
	var s consumerSet
	s.add(MemberRef{Team: "team-a", Member: "alice"}, "shared/*", consumerSourceIntake)
	s.add(MemberRef{Team: "team-b", Member: "bob"}, "isolated/*", consumerSourceIntake)

	if !s.Overlaps("shared/something") {
		t.Errorf("alice's intake should be reachable through Overlaps")
	}
	if !s.Overlaps("isolated/elsewhere") {
		t.Errorf("bob's intake should be reachable through Overlaps")
	}
	if s.Overlaps("nobody-reads-this/*") {
		t.Errorf("unrelated prefix should not overlap any registered intake")
	}
}

func TestConsumerSet_TagsEntrySource(t *testing.T) {
	var s consumerSet
	s.add(MemberRef{Team: "team-a", Member: "alice"}, "intake-only/*", consumerSourceIntake)

	if got := len(s.entries); got != 1 {
		t.Fatalf("expected 1 entry, got %d", got)
	}
	got := s.entries[0]
	if got.Source != consumerSourceIntake {
		t.Errorf("source = %q, want %q", got.Source, consumerSourceIntake)
	}
	if got.Member.Team != "team-a" || got.Member.Member != "alice" {
		t.Errorf("member ref = %+v, want team-a/alice", got.Member)
	}
	if got.Prefix != "intake-only/*" {
		t.Errorf("prefix = %q, want %q", got.Prefix, "intake-only/*")
	}
}

func TestConsumerSet_PreservesDeclarationOrder(t *testing.T) {
	// Entry order is deterministic: members are visited in input order,
	// and intakes within a member are appended in declaration order.
	// Stability matters for reproducible diagnostic output in future
	// findings (Phase 1.6 may surface "consumed by ..." details).
	var s consumerSet
	s.add(MemberRef{Team: "team-a", Member: "alice"}, "first/*", consumerSourceIntake)
	s.add(MemberRef{Team: "team-a", Member: "alice"}, "second/*", consumerSourceIntake)
	s.add(MemberRef{Team: "team-b", Member: "bob"}, "third/*", consumerSourceIntake)

	want := []string{"first/*", "second/*", "third/*"}
	if got := len(s.entries); got != len(want) {
		t.Fatalf("entry count = %d, want %d", got, len(want))
	}
	for i, w := range want {
		if got := s.entries[i].Prefix; got != w {
			t.Errorf("entries[%d].Prefix = %q, want %q", i, got, w)
		}
	}
}

func TestBuildConsumerSet_ExternalProducersExcluded(t *testing.T) {
	// external_producers documents the producer-side anchor for this
	// member's own intake — it names who writes into the prefix from
	// outside the team graph, NOT a consumer of anyone else's output.
	// It must therefore stay out of the consumer set so orphan_output's
	// semantics ("does anyone read this?") remain clean.
	external := "report-bug-skill"
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "audience-scan/*", Taxonomy: "tx"}},
			ExternalProducers: []string{external},
		}),
		mkMember("team-b", "writer", Topics{
			Output: []OutputEntry{{Prefix: "audience-scan/2026-05-04/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 1 {
		t.Fatalf("expected 1 entry (intake only; external_producers excluded), got %d: %+v", got, s.entries)
	}
	if got := s.entries[0].Source; got != consumerSourceIntake {
		t.Errorf("entry source = %q, want %q", got, consumerSourceIntake)
	}
	if got := s.entries[0].Prefix; got != "audience-scan/*" {
		t.Errorf("entry prefix = %q, want %q", got, "audience-scan/*")
	}
	// The writer's output is reachable from the consumer's intake.
	if !s.Overlaps("audience-scan/2026-05-04/abc") {
		t.Errorf("intake on team-a should overlap team-b's output")
	}
}

func TestBuildConsumerSet_RegistersRequiredRead(t *testing.T) {
	// Required-read prefixes must be in the consumer set so a writer
	// declaring `output[].prefix = campaign-draft/*` does not get flagged
	// orphan_output when the only reader has it on `required_read[]`
	// rather than `intake[]`. Phase 1.4 populates real data; this test
	// asserts the wiring exists today.
	members := []MemberTopics{
		mkMember("marketing-crew", "publisher", Topics{
			RequiredRead: []RequiredReadEntry{
				{Prefix: "campaign-draft/*", Comment: "publish-proposal context"},
			},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 1 {
		t.Fatalf("expected 1 entry (single required_read), got %d", got)
	}
	if got := s.entries[0].Source; got != consumerSourceRequiredRead {
		t.Errorf("entry source = %q, want %q", got, consumerSourceRequiredRead)
	}
	if got := s.entries[0].Prefix; got != "campaign-draft/*" {
		t.Errorf("entry prefix = %q, want %q", got, "campaign-draft/*")
	}
	if !s.Overlaps("campaign-draft/2026-05-04/example") {
		t.Errorf("required_read prefix should overlap matching output")
	}
}

func TestBuildConsumerSet_RegistersEvidenceConsumed(t *testing.T) {
	// Evidence-consumed prefixes must be in the consumer set so that
	// outputs cited as decision evidence are not flagged as orphans.
	members := []MemberTopics{
		mkMember("monetization", "catalog-strategist", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{
				{
					Prefix:       "candidate-sku-record/*",
					ForDecisions: []string{"catalog-promotion", "sku-retirement"},
				},
			},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 1 {
		t.Fatalf("expected 1 entry (single evidence_consumed), got %d", got)
	}
	if got := s.entries[0].Source; got != consumerSourceEvidence {
		t.Errorf("entry source = %q, want %q", got, consumerSourceEvidence)
	}
	if got := s.entries[0].Prefix; got != "candidate-sku-record/*" {
		t.Errorf("entry prefix = %q, want %q", got, "candidate-sku-record/*")
	}
	if !s.Overlaps("candidate-sku-record/2026-05-04/abc") {
		t.Errorf("evidence_consumed prefix should overlap matching output")
	}
}

func TestBuildConsumerSet_AggregatesAllConsumerKinds(t *testing.T) {
	// A single member declaring all three consumer-side kinds should
	// produce three entries, ordered intake → required_read →
	// evidence_consumed (matching topics.json field ordering for
	// stable downstream output).
	members := []MemberTopics{
		mkMember("marketing-crew", "publisher", Topics{
			Intake: []IntakeEntry{
				{Prefix: "publish-routing/*", Taxonomy: "tx"},
			},
			RequiredRead: []RequiredReadEntry{
				{Prefix: "campaign-draft/*"},
			},
			EvidenceConsumed: []EvidenceConsumedEntry{
				{
					Prefix:       "audience-scan/*",
					ForDecisions: []string{"content-publish-proposal"},
				},
			},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 3 {
		t.Fatalf("expected 3 entries (intake + required_read + evidence_consumed), got %d", got)
	}
	wantOrder := []struct {
		prefix string
		source consumerSource
	}{
		{"publish-routing/*", consumerSourceIntake},
		{"campaign-draft/*", consumerSourceRequiredRead},
		{"audience-scan/*", consumerSourceEvidence},
	}
	for i, want := range wantOrder {
		got := s.entries[i]
		if got.Prefix != want.prefix || got.Source != want.source {
			t.Errorf("entries[%d] = (%q, %q), want (%q, %q)",
				i, got.Prefix, got.Source, want.prefix, want.source)
		}
	}
}

func TestBuildConsumerSet_PreservesDeclarationOrderAcrossKinds(t *testing.T) {
	// Within a member, all entries of a given kind are appended in
	// declaration order, then we move to the next kind. Across members,
	// member input order is preserved.
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			RequiredRead: []RequiredReadEntry{
				{Prefix: "alice-r1/*"},
				{Prefix: "alice-r2/*"},
			},
		}),
		mkMember("team-b", "bob", Topics{
			Intake: []IntakeEntry{
				{Prefix: "bob-i1/*", Taxonomy: "tx"},
			},
			EvidenceConsumed: []EvidenceConsumedEntry{
				{Prefix: "bob-e1/*", ForDecisions: []string{"d1"}},
				{Prefix: "bob-e2/*", ForDecisions: []string{"d2"}},
			},
		}),
	}
	s := buildConsumerSet(members)

	want := []struct {
		prefix string
		source consumerSource
	}{
		{"alice-r1/*", consumerSourceRequiredRead},
		{"alice-r2/*", consumerSourceRequiredRead},
		{"bob-i1/*", consumerSourceIntake},
		{"bob-e1/*", consumerSourceEvidence},
		{"bob-e2/*", consumerSourceEvidence},
	}
	if got := len(s.entries); got != len(want) {
		t.Fatalf("entry count = %d, want %d", got, len(want))
	}
	for i, w := range want {
		got := s.entries[i]
		if got.Prefix != w.prefix || got.Source != w.source {
			t.Errorf("entries[%d] = (%q, %q), want (%q, %q)",
				i, got.Prefix, got.Source, w.prefix, w.source)
		}
	}
}

func TestBuildConsumerSet_DropsEmptyPrefixesFromAllSources(t *testing.T) {
	// Phase 1.1 sources, like intake, drop empty/whitespace prefixes
	// silently. Topics.Validate is the layer that surfaces these as
	// shape errors; buildConsumerSet's contract is "robust against
	// malformed input."
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{{Prefix: "good-intake/*", Taxonomy: "tx"}},
			RequiredRead: []RequiredReadEntry{
				{Prefix: ""},
				{Prefix: "   "},
				{Prefix: "good-required/*"},
			},
			EvidenceConsumed: []EvidenceConsumedEntry{
				{Prefix: "  ", ForDecisions: []string{"d1"}},
				{Prefix: "good-evidence/*", ForDecisions: []string{"d1"}},
			},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 3 {
		t.Fatalf("expected 3 entries (one per non-empty prefix), got %d: %+v", got, s.entries)
	}
}

func TestBuildConsumerSet_EmptyMembers(t *testing.T) {
	s := buildConsumerSet(nil)
	if got := len(s.entries); got != 0 {
		t.Errorf("buildConsumerSet(nil) should be empty, got %d entries", got)
	}
	if s.Overlaps("anything/*") {
		t.Errorf("empty consumerSet should not overlap any prefix")
	}
}

func TestBuildConsumerSet_MultipleIntakesPerMember(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{
				{Prefix: "first/*", Taxonomy: "tx"},
				{Prefix: "second/*", Taxonomy: "tx"},
				{Prefix: "third/*", Taxonomy: "tx"},
			},
		}),
	}
	s := buildConsumerSet(members)

	if got := len(s.entries); got != 3 {
		t.Fatalf("expected 3 entries (one per intake), got %d", got)
	}
	for _, e := range s.entries {
		if e.Source != consumerSourceIntake {
			t.Errorf("entry source = %q, want %q (entry: %+v)",
				e.Source, consumerSourceIntake, e)
		}
		if e.Member.String() != "team-a/alice" {
			t.Errorf("entry member = %q, want team-a/alice (entry: %+v)",
				e.Member.String(), e)
		}
	}
}
