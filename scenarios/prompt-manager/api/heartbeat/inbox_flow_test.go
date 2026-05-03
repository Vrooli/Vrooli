package heartbeat

import (
	"strings"
	"testing"

	"prompt-manager/memberflow"
)

func ptr(s string) *string { return &s }

func mkInputs(team, agent string, topics memberflow.Topics, taxonomies map[string]*memberflow.Taxonomy) *inboxFlowInputs {
	return &inboxFlowInputs{
		teamID:     team,
		agentID:    agent,
		memberFlow: memberflow.MemberTopics{Ref: memberflow.MemberRef{Team: team, Member: agent}, Topics: topics, Exists: true},
		taxonomies: taxonomies,
	}
}

func TestRenderInboxFlow_NilOrEmptyReturnsEmpty(t *testing.T) {
	if got := RenderInboxFlow(nil); got != "" {
		t.Errorf("nil input: got %q, want empty", got)
	}
	got := RenderInboxFlow(mkInputs("t", "m", memberflow.Topics{}, nil))
	if got != "" {
		t.Errorf("empty intake: got %q, want empty", got)
	}
}

func TestRenderInboxFlow_SingleIntake_AllSectionsPresent(t *testing.T) {
	tx := &memberflow.Taxonomy{
		ID:      "marketing-research",
		PoRPath: "docs/marketing/SIGNAL_TAXONOMY.md",
		SignalTypes: []memberflow.TaxonomySignalType{
			{ID: "audience-pain", DefaultMethod: "audience-pain-mining", DefaultDestinationPrefix: "audience-scan/<slug>"},
			{ID: "competitor", DefaultMethod: "competitor-positioning-scan", DefaultDestinationPrefix: "competitor/<slug>"},
		},
		PendingMethodSkills: []string{"audience-pain-mining", "competitor-positioning-scan"},
	}
	in := mkInputs("marketing-crew", "researcher",
		memberflow.Topics{
			Intake: []memberflow.IntakeEntry{{
				Prefix:          "research-inbox/*",
				Taxonomy:        "marketing-research",
				ClassifierSkill: "marketing-signal-classifier",
			}},
			Output: []memberflow.OutputEntry{
				{Prefix: "audience-scan/*", DestinationKind: memberflow.DestinationKnowledge, Schema: "audience-scan"},
				{Prefix: "monetization-benchmark-adjacent/*", DestinationKind: memberflow.DestinationKnowledge, DestinationTeam: ptr("monetization"), Schema: "monetization-benchmark-adjacent"},
			},
			DecisionsOwned:       []string{"audience-update", "channel-strategy-update"},
			DecisionsConsumed:    []string{"capability-gap"},
			RaisesCapabilityGaps: true,
			ExternalProducers:    []string{"vision-walk", "operator"},
		},
		map[string]*memberflow.Taxonomy{"marketing-research": tx},
	)
	out := RenderInboxFlow(in)
	if out == "" {
		t.Fatal("expected non-empty render")
	}
	must := []string{
		"# Inbox Flow",
		"## Inbox: `research-inbox/*`",
		"`marketing-crew`",
		"`marketing-research`",
		"`marketing-signal-classifier`",
		"prompt-manager team knowledge-list marketing-crew --topic-prefix=research-inbox/",
		"## Drain procedure (universal)",
		"prompt-manager team knowledge-delete marketing-crew",
		"## Destinations",
		"audience-scan/*",
		"monetization-benchmark-adjacent/*",
		"`monetization`",
		"## Decisions",
		"audience-update",
		"channel-strategy-update",
		"capability-gap",
		"## Default dispatch (from taxonomy)",
		"audience-pain",
		"competitor-positioning-scan",
		"Pending method skills",
		"vision-walk",
	}
	for _, sub := range must {
		if !strings.Contains(out, sub) {
			t.Errorf("rendered output missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestRenderInboxFlow_TwoIntakes_CrossTeam(t *testing.T) {
	tx := &memberflow.Taxonomy{
		ID:          "monetization-validation",
		PoRPath:     "docs/monetization/VALIDATION_TAXONOMY.md",
		SignalTypes: []memberflow.TaxonomySignalType{{ID: "pricing-comp-needed", DefaultMethod: "pricing-comp-capture"}},
	}
	in := mkInputs("monetization", "market-validator",
		memberflow.Topics{
			Intake: []memberflow.IntakeEntry{
				{Prefix: "validation-inbox/*", Taxonomy: "monetization-validation", ClassifierSkill: "market-validation-triage"},
				{Prefix: "monetization-benchmark-adjacent/*", Taxonomy: "monetization-validation", ClassifierSkill: "market-validation-triage", SourceTeam: ptr("marketing-crew")},
			},
			Output: []memberflow.OutputEntry{
				{Prefix: "monetization-benchmark/*", DestinationKind: memberflow.DestinationKnowledge, Schema: "market-scan"},
			},
		},
		map[string]*memberflow.Taxonomy{"monetization-validation": tx},
	)
	out := RenderInboxFlow(in)
	if !strings.Contains(out, "## Inbox: `validation-inbox/*`") {
		t.Errorf("missing first inbox header:\n%s", out)
	}
	if !strings.Contains(out, "## Inbox: `monetization-benchmark-adjacent/*`") {
		t.Errorf("missing second inbox header:\n%s", out)
	}
	if !strings.Contains(out, "Source team | `marketing-crew`") {
		t.Errorf("cross-team source not surfaced:\n%s", out)
	}
	// Dispatch table for a single taxonomy must render only once even when
	// two intakes share it.
	if got := strings.Count(out, "Taxonomy `monetization-validation`"); got != 1 {
		t.Errorf("expected dispatch rendered once for shared taxonomy; got %d", got)
	}
}

func TestRenderInboxFlow_NoTaxonomy_FlagsTransitional(t *testing.T) {
	// An intake without a taxonomy still renders, but the section flags
	// the missing taxonomy. (The validator's missing_taxonomy rule fires
	// in parallel; the renderer just surfaces the gap visually.)
	in := mkInputs("team-x", "legacy",
		memberflow.Topics{
			Intake: []memberflow.IntakeEntry{{Prefix: "legacy-inbox/*"}},
		},
		map[string]*memberflow.Taxonomy{},
	)
	out := RenderInboxFlow(in)
	if !strings.Contains(out, "_none declared_") {
		t.Errorf("expected 'none declared' marker; got:\n%s", out)
	}
}

func TestRenderInboxFlow_UnknownTaxonomy_FlagsIt(t *testing.T) {
	in := mkInputs("team-x", "m",
		memberflow.Topics{
			Intake: []memberflow.IntakeEntry{{Prefix: "x/*", Taxonomy: "no-such-tx"}},
		},
		map[string]*memberflow.Taxonomy{},
	)
	out := RenderInboxFlow(in)
	if !strings.Contains(out, "NOT FOUND in registry") {
		t.Errorf("expected 'NOT FOUND in registry' marker; got:\n%s", out)
	}
}

func TestRenderInboxFlow_NoClassifier(t *testing.T) {
	tx := &memberflow.Taxonomy{ID: "tx", SignalTypes: []memberflow.TaxonomySignalType{{ID: "auto", DefaultMethod: "auto-method"}}}
	in := mkInputs("t", "m",
		memberflow.Topics{
			Intake: []memberflow.IntakeEntry{{Prefix: "auto-inbox/*", Taxonomy: "tx"}},
		},
		map[string]*memberflow.Taxonomy{"tx": tx},
	)
	out := RenderInboxFlow(in)
	if !strings.Contains(out, "_none_ — the topic prefix is taken as the deterministic signal-type") {
		t.Errorf("expected deterministic-signal note; got:\n%s", out)
	}
}

func TestPrefixForList(t *testing.T) {
	if got := prefixForList("research-inbox/*"); got != "research-inbox/" {
		t.Errorf("got %q", got)
	}
	if got := prefixForList("validation-inbox/staleness"); got != "validation-inbox/staleness" {
		t.Errorf("got %q", got)
	}
}

func TestDeriveRepoRoot(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	cases := []struct {
		store, want string
	}{
		{"/x/scenarios/prompt-manager/store", "/x"},
		{"/foo/bar/baz/qux/store", "/foo/bar"},
		{"", ""},
	}
	for _, c := range cases {
		if got := deriveRepoRoot(c.store); got != c.want {
			t.Errorf("deriveRepoRoot(%q) = %q, want %q", c.store, got, c.want)
		}
	}
}

func TestDeriveRepoRoot_HonorsVrooliRoot(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "/explicit/root")
	if got := deriveRepoRoot("/anything/else"); got != "/explicit/root" {
		t.Errorf("VROOLI_ROOT not honored; got %q", got)
	}
}
