package memberflow

import "testing"

// Behavioral tests for the Topic Catalog docs rules.
//
// These read the `## Topic Catalog` table beside a contract graph. None was
// named by a test at plan start, so the whole table's checks were unverified:
// a table could stop being parsed, or a status could stop being recognised, and
// the family would report nothing without anyone noticing.

// catalogContext wires a contract block carrying one Topic Catalog table.
func catalogContext(table OperatingTopicCatalogTable) RuleContext {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{
			ID:    "team-a-operating-model",
			Team:  "team-a",
			Scope: "team",
			Mode:  OperatingGraphModeContract,
		},
		Source: OperatingGraphSource{Path: "docs/team-a/operating/OPERATING_MODEL.md", FenceLine: 1},
		Docs:   OperatingGraphDocs{TopicCatalog: table},
	}
	runtime := OperatingGraphRuntime{}
	return RuleContext{
		OperatingGraphRuleContext: OperatingGraphRuleContext{
			Block:   block,
			Runtime: runtime,
			Index:   NewOperatingGraphContractIndex(block, runtime),
			Matcher: NewOperatingRelationshipMatcher(),
		},
	}
}

func TestTopicCatalogMissingFiresWhenTheTableIsAbsent(t *testing.T) {
	ctx := catalogContext(OperatingTopicCatalogTable{Present: false})
	findings := (graphTopicCatalogMissingRule{}).Check(ctx)
	if len(findings) != 1 {
		t.Fatalf("graph_topic_catalog_missing produced %d findings, want 1", len(findings))
	}
	if findings[0].Rule != "graph_topic_catalog_missing" {
		t.Errorf("rule = %q", findings[0].Rule)
	}

	// A present table must not fire it; otherwise the rule reports every graph.
	present := catalogContext(OperatingTopicCatalogTable{Present: true})
	if got := (graphTopicCatalogMissingRule{}).Check(present); len(got) != 0 {
		t.Errorf("graph_topic_catalog_missing fired on a present table: %+v", got)
	}
}

func TestTopicCatalogInvalidTopicFiresOnAnUnparseableToken(t *testing.T) {
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{
			// An unparsed token leaves Topic empty and keeps the raw text.
			{Topic: "", RawTopic: "Not A Topic", SourceLine: 12},
			{Topic: "audience-scan/*", RawTopic: "audience-scan/*", SourceLine: 13},
		},
	})
	findings := (graphTopicCatalogInvalidTopicRule{}).Check(ctx)
	if len(findings) != 1 {
		t.Fatalf("graph_topic_catalog_invalid_topic produced %d findings, want 1: %+v", len(findings), findings)
	}
	if findings[0].Rule != "graph_topic_catalog_invalid_topic" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}
	if findings[0].Line != 12 {
		t.Errorf("finding does not point at the offending row: line = %d, want 12", findings[0].Line)
	}
}

func TestTopicCatalogUnknownStatusFiresOnAnUnrecognisedStatus(t *testing.T) {
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{
			{Topic: "audience-scan/*", Status: "banana", StatusKind: OperatingTopicStatusUnknown, SourceLine: 20},
		},
	})
	findings := (graphTopicCatalogUnknownStatusRule{}).Check(ctx)
	if len(findings) != 1 {
		t.Fatalf("graph_topic_catalog_unknown_status produced %d findings, want 1", len(findings))
	}
	if findings[0].Rule != "graph_topic_catalog_unknown_status" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}
	if findings[0].Topic != "audience-scan/*" {
		t.Errorf("finding does not name the topic: %+v", findings[0])
	}
}

// Two more Topic Catalog rules that read a row's status against its qualifier
// and against the graph.
func TestTopicCatalogStatusQualifierDriftFiresWhenTheQualifierContradictsTheStatus(t *testing.T) {
	// A `target` status means the topic does not exist yet, so its token must
	// carry the future qualifier. Without it the catalog claims a live topic.
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "planned-record/*",
			Qualifier:  "",
			Status:     "target",
			StatusKind: OperatingTopicStatusTarget,
			SourceLine: 40,
		}},
	})
	findings := (graphTopicCatalogStatusQualifierDriftRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_topic_catalog_status_qualifier_drift did not fire")
	}
	if findings[0].Rule != "graph_topic_catalog_status_qualifier_drift" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Topic != "planned-record/*" {
		t.Errorf("finding does not name the topic: %+v", findings[0])
	}

	// The same row with the qualifier the status implies must stay silent.
	ok := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "planned-record/*",
			Qualifier:  string(OperatingGraphQualifierFuture),
			Status:     "target",
			StatusKind: OperatingTopicStatusTarget,
			SourceLine: 40,
		}},
	})
	if got := (graphTopicCatalogStatusQualifierDriftRule{}).Check(ok); len(got) != 0 {
		t.Errorf("graph_topic_catalog_status_qualifier_drift fired on a consistent row: %+v", got)
	}
}

func TestTopicCatalogTransitionalWithoutTargetFiresWhenNoReplacementIsNamed(t *testing.T) {
	// `live transitional` asserts the topic is on its way out. Without a future
	// replacement in the graph the row records an intention with no destination.
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "legacy-record/*",
			Status:     "live transitional",
			StatusKind: OperatingTopicStatusLiveTransitional,
			SourceLine: 50,
		}},
	})
	findings := (graphTopicCatalogTransitionalWithoutTargetRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_topic_catalog_transitional_without_target did not fire")
	}
	if findings[0].Rule != "graph_topic_catalog_transitional_without_target" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityWarning {
		t.Errorf("severity = %q, want warning", findings[0].Severity)
	}

	// A non-transitional row must not fire it.
	live := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "current-record/*",
			Status:     "live",
			StatusKind: OperatingTopicStatusLive,
			SourceLine: 51,
		}},
	})
	if got := (graphTopicCatalogTransitionalWithoutTargetRule{}).Check(live); len(got) != 0 {
		t.Errorf("graph_topic_catalog_transitional_without_target fired on a live row: %+v", got)
	}
}

// graph_topic_catalog_drift is the two-way check between the Topic Catalog
// table and the topic nodes drawn in the graph. Without it the table and the
// diagram beside it can describe different topic sets indefinitely.
func TestTopicCatalogDriftFiresWhenTheGraphDrawsATopicTheTableOmits(t *testing.T) {
	drawn := node("T", OperatingGraphNodeKindTopic, "undocumented-record/*")
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{
			ID: "team-a-operating-model", Team: "team-a", Scope: "team", Mode: OperatingGraphModeContract,
		},
		Source: OperatingGraphSource{Path: "docs/team-a/operating/OPERATING_MODEL.md", FenceLine: 1},
		Graph:  OperatingGraph{Nodes: []OperatingGraphNode{drawn}},
		Docs: OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{
			Present: true,
			Rows:    []OperatingTopicCatalogRow{{Topic: "some-other-record/*", SourceLine: 30}},
		}},
	}
	runtime := OperatingGraphRuntime{}
	ctx := RuleContext{OperatingGraphRuleContext: OperatingGraphRuleContext{
		Block: block, Runtime: runtime,
		Index:   NewOperatingGraphContractIndex(block, runtime),
		Matcher: NewOperatingRelationshipMatcher(),
	}}

	findings := (graphTopicCatalogDriftRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_topic_catalog_drift did not fire for a graph topic absent from the catalog")
	}
	if findings[0].Rule != "graph_topic_catalog_drift" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}

	// When the catalog documents the drawn topic the rule must stay silent.
	block.Docs.TopicCatalog.Rows = []OperatingTopicCatalogRow{{Topic: "undocumented-record/*", SourceLine: 30}}
	agreed := RuleContext{OperatingGraphRuleContext: OperatingGraphRuleContext{
		Block: block, Runtime: runtime,
		Index:   NewOperatingGraphContractIndex(block, runtime),
		Matcher: NewOperatingRelationshipMatcher(),
	}}
	if got := (graphTopicCatalogDriftRule{}).Check(agreed); len(got) != 0 {
		t.Errorf("graph_topic_catalog_drift fired on an agreeing catalog: %+v", got)
	}
}

// graph_topic_catalog_live_status_unbacked: a row asserting a live topic that
// the diagram beside it does not draw. The table would be claiming traffic the
// graph shows nowhere.
func TestTopicCatalogLiveStatusUnbackedFiresWhenTheGraphDrawsNoSuchTopic(t *testing.T) {
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "claimed-live-record/*",
			Status:     "live",
			StatusKind: OperatingTopicStatusLive,
			SourceLine: 100,
		}},
	})
	findings := (graphTopicCatalogLiveStatusUnbackedRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_topic_catalog_live_status_unbacked did not fire for a live row with no graph topic")
	}
	if findings[0].Rule != "graph_topic_catalog_live_status_unbacked" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Topic != "claimed-live-record/*" {
		t.Errorf("finding does not name the topic: %+v", findings[0])
	}
}

// graph_topic_catalog_purpose_drift survives generation: the table's purpose
// text is authored in team.json::topicCatalog and carried through, so a hand
// edit to a generated row still has something to disagree with.
func TestTopicCatalogPurposeDriftFiresWhenTheRowContradictsTheContract(t *testing.T) {
	ctx := catalogContext(OperatingTopicCatalogTable{
		Present: true,
		Rows: []OperatingTopicCatalogRow{{
			Topic:      "some-record/*",
			Status:     "live",
			StatusKind: OperatingTopicStatusLive,
			Purpose:    "a purpose the contract does not state",
			SourceLine: 120,
		}},
	})
	findings := (graphTopicCatalogPurposeDriftRule{}).Check(ctx)
	if len(findings) == 0 {
		t.Fatal("graph_topic_catalog_purpose_drift did not fire for a row with no contract entry")
	}
	if findings[0].Rule != "graph_topic_catalog_purpose_drift" {
		t.Errorf("rule = %q", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("severity = %q, want error", findings[0].Severity)
	}
}
