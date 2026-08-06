package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/teamcontract"
)

func TestValidateOperatingModelsRejectsUnanchoredFeedbackStep(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.FeedbackLoop.Steps[0].References = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_feedback_step_unanchored")
}

func TestValidateOperatingModelsRejectsUnbackedFeedbackReference(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.FeedbackLoop.Steps[0].References = []string{"ghost-surface/*"}

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_feedback_reference_unbacked")
}

func TestValidateOperatingModelsRejectsUnstructuredGapItems(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Gaps.Items = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_gaps_items_missing")
}

func TestValidateOperatingModelsRejectsUnanchoredGapItem(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Gaps.Items[0].References = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_gap_item_unanchored")
}

func TestValidateOperatingModelsRejectsMissingAdoptionCommands(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Adoption.Commands = models[0].Sections.Adoption.Commands[:1]

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_adoption_command_missing")
	assertOperatingFindingDetail(t, result, "operating_model_adoption_command_missing", "diff")
}

func TestOperatingModelGoldenFixtureValidatesAndCoversAllSections(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	runtime := operatingModelDiscoverabilityRuntime(t, model, true, true)

	result := ValidateOperatingModels([]OperatingModelDocument{model}, runtime, "team-a", "g")
	if result.Errors != 0 || result.Warnings != 0 {
		t.Fatalf("golden operating-model fixture should validate cleanly: %+v", result.Findings)
	}

	coverage := BuildOperatingModelCoverage([]OperatingModelDocument{model}, runtime, "team-a", "g")
	if len(coverage) != 1 {
		t.Fatalf("coverage length=%d, want 1: %+v", len(coverage), coverage)
	}
	docs := coverage[0].Docs
	if docs.RequiredSectionsPresent != docs.RequiredSectionsTotal || docs.RequiredSectionsTotal == 0 {
		t.Fatalf("golden fixture should cover every required section: %+v", docs)
	}
	if docs.TopicCatalogRows != 1 || docs.TopicCatalogMatched != 1 || docs.TopicCatalogGraphOnly != 0 || docs.TopicCatalogDocsOnly != 0 {
		t.Fatalf("golden fixture topic coverage drifted: %+v", docs)
	}
	if docs.DecisionsRows != 1 || docs.DecisionsMatched != 1 || docs.DecisionsGraphOnly != 0 || docs.DecisionsDocsOnly != 0 {
		t.Fatalf("golden fixture decision coverage drifted: %+v", docs)
	}
	if docs.ExternalInputsRows != 1 || docs.ExternalInputsBackedRows != 1 || docs.ExternalInputsUnbackedRows != 0 {
		t.Fatalf("golden fixture external input coverage drifted: %+v", docs)
	}
	if docs.OutputsRows != 1 || docs.OutputsBackedRows != 1 || docs.OutputsUnbackedRows != 0 {
		t.Fatalf("golden fixture output coverage drifted: %+v", docs)
	}
	if docs.FeedbackSteps != 1 || docs.FeedbackAnchoredSteps != 1 || docs.FeedbackUnbackedReferences != 0 {
		t.Fatalf("golden fixture feedback coverage drifted: %+v", docs)
	}
	if docs.GapsItems != 1 || docs.GapsAnchoredItems != 1 {
		t.Fatalf("golden fixture gap coverage drifted: %+v", docs)
	}
	if docs.AdoptionValidationCommands != 3 ||
		docs.PlanOfRecordRegistration != OperatingCoverageStatusEnforced ||
		docs.ReadmeDiscoverability != OperatingCoverageStatusEnforced {
		t.Fatalf("golden fixture discoverability coverage drifted: %+v", docs)
	}
}

func TestOperatingModelRegisteredRulesHaveFailureFixtures(t *testing.T) {
	cases := []struct {
		rule    string
		mutate  func(*OperatingModelDocument)
		runtime func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime
	}{
		{rule: "operating_model_required_section_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs = OperatingExternalInputsTable{}
		}},
		{rule: "operating_model_duplicate_section", mutate: func(model *OperatingModelDocument) {
			model.Sections.Mission.Duplicates = []int{3}
		}},
		{rule: "operating_model_decisions_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Headers = []string{"decision context", "owner", "purpose"}
		}},
		{rule: "operating_model_decisions_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Rows = nil
		}},
		{rule: "operating_model_decisions_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Rows[0].ExpectedEvidenceTrigger = ""
		}},
		{rule: "operating_model_external_inputs_table_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Table = false
		}},
		{rule: "operating_model_external_inputs_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Headers = []string{"input", "enters through", "first handler", "notes"}
		}},
		{rule: "operating_model_external_inputs_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows = nil
		}},
		{rule: "operating_model_external_inputs_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].RoutingRule = ""
		}},
		{rule: "operating_model_external_inputs_producer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
		}},
		{rule: "operating_model_external_inputs_entry_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].EntrySurface = "`topic:ghost-inbox/*`"
		}},
		{rule: "operating_model_external_inputs_drainer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].Drainer = "member:ghost"
		}},
		{rule: "operating_model_outputs_table_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Table = false
		}},
		{rule: "operating_model_outputs_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Headers = []string{"output", "downstream consumer", "path"}
		}},
		{rule: "operating_model_outputs_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows = nil
		}},
		{rule: "operating_model_outputs_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Purpose = ""
		}},
		{rule: "operating_model_outputs_surface_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
		}},
		{rule: "operating_model_outputs_consumer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Consumer = "ghost"
		}},
		{rule: "operating_model_feedback_steps_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps = nil
		}},
		{rule: "operating_model_feedback_step_unanchored", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps[0].References = nil
		}},
		{rule: "operating_model_feedback_reference_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps[0].References = []string{"ghost-surface/*"}
		}},
		{rule: "operating_model_gaps_items_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Gaps.Items = nil
		}},
		{rule: "operating_model_gap_item_unanchored", mutate: func(model *OperatingModelDocument) {
			model.Sections.Gaps.Items[0].References = nil
		}},
		{rule: "operating_model_adoption_command_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Adoption.Commands = model.Sections.Adoption.Commands[:1]
		}},
		{rule: "operating_model_plan_of_record_missing", runtime: func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime {
			return operatingModelDiscoverabilityRuntime(t, model, false, true)
		}},
		{rule: "operating_model_readme_link_missing", runtime: func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime {
			return operatingModelDiscoverabilityRuntime(t, model, true, false)
		}},
	}

	fixtured := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			model := operatingModelDocumentFixture(t)
			if tc.mutate != nil {
				tc.mutate(&model)
			}
			runtime := OperatingGraphRuntime{}
			if tc.runtime != nil {
				runtime = tc.runtime(t, model)
			}
			result := ValidateOperatingModels([]OperatingModelDocument{model}, runtime, "team-a", "g")
			assertOperatingFinding(t, result, tc.rule)
			fixtured[tc.rule] = true
		})
	}

	for _, rule := range DefaultOperatingModelRules() {
		if !fixtured[rule.ID()] {
			t.Fatalf("registered operating-model rule %q has no deliberate failure fixture", rule.ID())
		}
	}
}

func TestValidateOperatingModelsChecksPlanOfRecordManifest(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writeFile("docs/team-a/manifest.json", `{
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [
    {
      "id": "entrypoint",
      "path": ".",
      "documents": [{
        "path": "README.md",
        "required": true,
        "validation": {"requiredHeadings": ["Start here for agents", "Folder map"]}
      }]
    },
    {
      "id": "operating",
      "path": "operating/",
      "required": true,
      "documents": [{"path": "OPERATING_MODEL.md", "required": true}]
    },
    {
      "id": "taxonomies",
      "path": "taxonomies/",
      "packages": [{"id": "signal", "path": "signal/", "requiredFiles": ["README.md", "taxonomy.json"]}]
    }
  ]
}`)
	writeFile("docs/team-a/README.md", "# Team A\n\n## Start here for agents\n")
	writeFile("docs/team-a/operating/OPERATING_MODEL.md", "# Operating\n")
	writeFile("docs/team-a/taxonomies/signal/README.md", "# Signal\n")

	model := operatingModelDocumentFixture(t)
	model.Team = "team-a"
	model.Source.Path = "docs/team-a/operating/OPERATING_MODEL.md"
	result := ValidateOperatingModels([]OperatingModelDocument{model}, OperatingGraphRuntime{RepoRoot: repoRoot}, "team-a", "g")

	assertOperatingFinding(t, result, "por_required_heading_missing")
	assertOperatingFinding(t, result, "por_package_required_file_missing")
}

func TestValidateOperatingModelsRejectsPlanOfRecordNotebookSurface(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writeFile("docs/team-a/manifest.json", `{
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [
    {"id": "entrypoint", "path": ".", "required": true, "documents": [{"path": "README.md", "required": true}]},
    {"id": "operating", "path": "operating/", "required": true, "documents": [{"path": "OPERATING_MODEL.md", "required": true}]}
  ]
}`)
	writeFile("docs/team-a/README.md", "# Team A\n\n## Migration Notes\n\nHistorical context is allowed here.\n")
	writeFile("docs/team-a/operating/OPERATING_MODEL.md", "# Operating\n")
	writeFile("docs/team-a/notebook/NOTE.md", "# Note\n")

	model := operatingModelDocumentFixture(t)
	model.Team = "team-a"
	model.Source.Path = "docs/team-a/operating/OPERATING_MODEL.md"
	result := ValidateOperatingModels([]OperatingModelDocument{model}, OperatingGraphRuntime{RepoRoot: repoRoot}, "team-a", "g")

	assertOperatingFinding(t, result, "por_notebook_surface")
}

func TestLoadAllTaxonomiesDiscoversPackagedTaxonomyJSON(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, "docs", "team-a", "taxonomies", "signal", "taxonomy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir taxonomy dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"id":"team-a-signal","signalTypes":[{"id":"bug"}]}`), 0o644); err != nil {
		t.Fatalf("write taxonomy: %v", err)
	}

	registry, err := LoadAllTaxonomies(repoRoot)
	if err != nil {
		t.Fatalf("LoadAllTaxonomies: %v", err)
	}
	if _, ok := registry["team-a-signal"]; !ok {
		t.Fatalf("expected packaged taxonomy to load, got ids=%v", registry.IDs())
	}
}

func TestExtractOperatingGraphBlocksRejectsUnsupportedAllowMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
allow: graph_edge_unbacked
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
`+"```"+`
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := ExtractOperatingGraphBlocks(path, "docs/x.md")
	if err == nil || !strings.Contains(err.Error(), `metadata field "allow" is not supported`) {
		t.Fatalf("expected unsupported allow metadata error, got %v", err)
	}
}

func TestMarketingOperatingModelUsesReadableAnnotatedLabels(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "operating", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/operating/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	graph := blocks[0].Graph
	if len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		t.Fatalf("expected populated marketing graph, nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	}
	mon := operatingNodeByID(t, graph, "MON")
	if mon.Kind != "team" || mon.Value != "monetization" || mon.Display != "Monetization team" {
		t.Fatalf("bad MON node: %+v", mon)
	}
	if strings.Contains(mon.Display, "team:") || strings.Contains(mon.RawLabel, "team:monetization") {
		t.Fatalf("MON visual label should not contain machine token: %+v", mon)
	}
}

func TestMarketingOperatingModelCentralizesTypedObservationDrainage(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "operating", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/operating/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	rels := NewOperatingRelationshipSet(BuildGraphOperatingRelationships(blocks[0]))
	wantBrandManagerRead := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: "brand-manager", Topic: "marketing-craft-observation/*"}
	if !operatingRelationshipSetContains(rels, wantBrandManagerRead) {
		t.Fatalf("marketing craft observation must drain through brand-manager; relationships=%+v", rels.All())
	}
	for _, member := range []string{"researcher", "oss-advertiser", "subscription-advertiser", "publisher"} {
		forbidden := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: member, Topic: "marketing-craft-observation/*"}
		if operatingRelationshipSetContains(rels, forbidden) {
			t.Fatalf("raw marketing craft observation should not be a direct runtime read for %s", member)
		}
	}
}

func TestValidateOperatingModelsKeepsMarketingScenarioQAMetaGreen(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	storeDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store")
	if _, err := os.Stat(storeDir); err != nil {
		t.Skipf("bundled prompt-manager store not available: %v", err)
	}
	blocks, err := LoadOperatingGraphBlocks(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingGraphBlocks: %v", err)
	}
	models, err := LoadOperatingModelDocuments(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingModelDocuments: %v", err)
	}
	runtime, err := BuildOperatingGraphRuntime(repoRoot, storeDir)
	if err != nil {
		t.Fatalf("BuildOperatingGraphRuntime: %v", err)
	}

	cases := []struct {
		team string
		id   string
	}{
		{team: "marketing-crew", id: "marketing-operating-model"},
		{team: "scenario-qa", id: "scenario-qa-operating-model"},
		{team: "meta-optimization", id: "meta-optimization-operating-model"},
	}
	for _, tc := range cases {
		t.Run(tc.team, func(t *testing.T) {
			result := ValidateOperatingModels(models, runtime, tc.team, tc.id)
			if result.Errors != 0 || result.Warnings != 0 {
				t.Fatalf("unexpected validation findings for %s/%s: %+v", tc.team, tc.id, result.Findings)
			}

			diffs := DiffOperatingGraphs(blocks, runtime, tc.team, tc.id)
			if len(diffs) != 0 {
				t.Fatalf("unexpected diff for %s/%s: %+v", tc.team, tc.id, diffs)
			}

			coverage := BuildOperatingModelCoverage(models, runtime, tc.team, tc.id)
			if len(coverage) != 1 {
				t.Fatalf("coverage length=%d, want 1: %+v", len(coverage), coverage)
			}
			docCoverage := coverage[0].Docs
			if docCoverage.TopicCatalogRows != docCoverage.TopicCatalogMatched || docCoverage.TopicCatalogGraphOnly != 0 || docCoverage.TopicCatalogDocsOnly != 0 {
				t.Fatalf("unexpected topic catalog coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.DecisionsRows != docCoverage.DecisionsMatched || docCoverage.DecisionsGraphOnly != 0 || docCoverage.DecisionsDocsOnly != 0 {
				t.Fatalf("unexpected decision coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.DecisionsMetadataComplete != docCoverage.DecisionsRows || docCoverage.DecisionsMetadataIncomplete != 0 {
				t.Fatalf("unexpected decision metadata coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.RequiredSectionsPresent != docCoverage.RequiredSectionsTotal || docCoverage.RequiredSectionsTotal == 0 {
				t.Fatalf("unexpected required-section coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.ExternalInputsTable != OperatingCoverageStatusEnforced ||
				docCoverage.ExternalInputsRows == 0 ||
				docCoverage.ExternalInputsBackedRows != docCoverage.ExternalInputsRows ||
				docCoverage.ExternalInputsUnbackedRows != 0 {
				t.Fatalf("unexpected external-input coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.OutputsTable != OperatingCoverageStatusEnforced ||
				docCoverage.OutputsRows == 0 ||
				docCoverage.OutputsBackedRows != docCoverage.OutputsRows ||
				docCoverage.OutputsUnbackedRows != 0 {
				t.Fatalf("unexpected outputs coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.FeedbackSteps == 0 || docCoverage.FeedbackAnchoredSteps != docCoverage.FeedbackSteps || docCoverage.FeedbackUnbackedReferences != 0 {
				t.Fatalf("unexpected feedback coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.GapsItems == 0 || docCoverage.GapsAnchoredItems != docCoverage.GapsItems {
				t.Fatalf("unexpected gaps coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.AdoptionValidationCommands != 3 || docCoverage.PlanOfRecordRegistration != OperatingCoverageStatusEnforced || docCoverage.ReadmeDiscoverability != OperatingCoverageStatusEnforced {
				t.Fatalf("unexpected adoption/discoverability coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			for _, rel := range coverage[0].Relationships {
				if rel.GraphOnly != 0 || rel.RuntimeOnly != 0 {
					t.Fatalf("unexpected relationship coverage drift for %s/%s relationship %s: %+v", tc.team, tc.id, rel.Relationship, rel)
				}
			}
		})
	}
}

func mustParseGraph(t *testing.T, lines []string) OperatingGraph {
	t.Helper()
	graph, err := ParseOperatingMermaid("g", lines, 1)
	if err != nil {
		t.Fatalf("parse graph: %v", err)
	}
	return graph
}

func operatingNodeByID(t *testing.T, graph OperatingGraph, id string) OperatingGraphNode {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.ID == id {
			return node
		}
	}
	t.Fatalf("node %q missing from %+v", id, graph.Nodes)
	return OperatingGraphNode{}
}

func assertOperatingFinding(t *testing.T, result OperatingGraphValidationResult, rule string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Rule == rule {
			return
		}
	}
	t.Fatalf("finding %q missing from %+v", rule, result.Findings)
}

func assertOperatingFindingDetail(t *testing.T, result OperatingGraphValidationResult, rule, detailFragment string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Rule == rule && strings.Contains(f.Detail, detailFragment) {
			return
		}
	}
	t.Fatalf("finding rule=%q detail containing %q missing from %+v", rule, detailFragment, result.Findings)
}

func countOperatingDiffs(diffs []OperatingGraphContractDiff, kind string) int {
	var count int
	for _, diff := range diffs {
		if diff.Kind == kind {
			count++
		}
	}
	return count
}

func operatingRelationshipSetContains(set OperatingRelationshipSet, rel OperatingRelationship) bool {
	for _, candidate := range set.All() {
		if operatingGraphRelationshipsEquivalent(candidate, rel) {
			return true
		}
	}
	return false
}

func TestExtractOperatingModelDocumentsRejectsDuplicateContractGraphs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g1
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
  T1["topic:first/*"]
  T1 --> A
`+"```"+`
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:first/*`"+` | live | member:a | member:a | First. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|

<!-- prompt-manager-graph:
id: g2
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
  T2["topic:second/*"]
  T2 --> A
`+"```"+`
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:second/*`"+` | live | member:a | member:a | Second. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ExtractOperatingModelDocuments(path, "docs/x/OPERATING_MODEL.md")
	if err == nil || !strings.Contains(err.Error(), "one contract graph") {
		t.Fatalf("expected duplicate contract graph rejection, got %v", err)
	}
}

func operatingDiffBlock(t *testing.T, lines []string) OperatingGraphBlock {
	t.Helper()
	return OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph:    mustParseGraph(t, lines),
		Source:   OperatingGraphSource{Path: "docs/test/OPERATING_MODEL.md", Line: 1, FenceLine: 2},
	}
}

func operatingModelDocumentFixture(t *testing.T) OperatingModelDocument {
	t.Helper()
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  OP(["external:operator"])`,
		`  A["member:a"]`,
		`  T[("topic:first/*")]`,
		`  D{"decision:model-update"}`,
		"  OP --> A",
		"  A --> T",
		"  T --> A",
		"  A --> D",
		"  D --> A",
	})
	block.Docs = OperatingGraphDocs{
		TopicCatalog: OperatingTopicCatalogTable{
			Present:    true,
			HeaderLine: 10,
			Rows: []OperatingTopicCatalogRow{{
				Topic:      "first/*",
				Status:     "live",
				StatusKind: OperatingTopicStatusLive,
				Writers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Readers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Purpose:    "First.",
				SourceLine: 12,
				RawTopic:   "`topic:first/*`",
			}},
		},
		Decisions: OperatingDecisionTable{
			Present:    true,
			HeaderLine: 14,
			Headers:    []string{"decision context", "owner", "purpose", "expected evidence / trigger", "accepted effect"},
			Rows: []OperatingDecisionRow{{
				Decision:                "model-update",
				Owners:                  []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Purpose:                 "Update the model contract.",
				ExpectedEvidenceTrigger: "Evidence from `topic:first/*`.",
				AcceptedEffect:          "Operator-approved operating-model document update.",
				SourceLine:              16,
				RawDecision:             "`model-update`",
			}},
		},
	}
	return OperatingModelDocument{
		ID:     "g",
		Team:   "team-a",
		Source: OperatingModelSource{Path: "docs/test/OPERATING_MODEL.md", Line: 1},
		Sections: OperatingModelSections{
			Mission:        OperatingMarkdownSection{Heading: "Mission", Present: true, Line: 1},
			Scope:          OperatingMarkdownSection{Heading: "Scope", Present: true, Line: 2},
			OperatingLoops: OperatingMarkdownSection{Heading: "Operating Loops", Present: true, Line: 3},
			Graph:          OperatingGraphSection{OperatingGraphBlock: block, Heading: "Operating Graph", Present: true},
			TopicCatalog:   block.Docs.TopicCatalog,
			Decisions:      block.Docs.Decisions,
			ExternalInputs: OperatingExternalInputsTable{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "External Inputs / Triggers", Present: true, Line: 16},
				HeaderLine:               17,
				Headers:                  []string{"producer / trigger", "entry surface", "drainer", "routing rule"},
				Table:                    true,
				Rows: []OperatingExternalInputRow{{
					ProducerTrigger: "Operator",
					EntrySurface:    "direct member context",
					Drainer:         "member:a",
					RoutingRule:     "Route directly.",
					SourceLine:      19,
				}},
			},
			Outputs: OperatingOutputsTable{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Outputs / Downstream Consumers", Present: true, Line: 20},
				HeaderLine:               21,
				Headers:                  []string{"output", "surface", "consumer", "purpose"},
				Table:                    true,
				Rows: []OperatingOutputRow{{
					Output:     "Output",
					Surface:    "`topic:first/*`",
					Consumer:   "member:a",
					Purpose:    "Read it.",
					SourceLine: 23,
				}},
			},
			FeedbackLoop: OperatingFeedbackSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Feedback / Capability Improvement Loop", Present: true, Line: 18},
				Steps: []OperatingFeedbackStep{{
					Text:       "Review `topic:first/*`.",
					References: []string{"topic:first/*"},
					SourceLine: 19,
				}},
			},
			Gaps: OperatingGapsSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Current Implementation Gaps", Present: true, Line: 19},
				Items: []OperatingGapItem{{
					Text:       "`topic[future]:second/*` remains target-state until a producer exists.",
					References: []string{"topic[future]:second/*"},
					SourceLine: 20,
				}},
			},
			Adoption: OperatingAdoptionSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Adoption / Validation", Present: true, Line: 24},
				Commands: []OperatingAdoptionCommand{
					{Command: "prompt-manager graph operating-model validate --team team-a --id g", SourceLine: 25},
					{Command: "prompt-manager graph operating-model diff --team team-a --id g", SourceLine: 26},
					{Command: "prompt-manager graph operating-model coverage --team team-a --id g", SourceLine: 27},
				},
			},
		},
		Graphs: []OperatingGraphBlock{block},
	}
}

func operatingModelDiscoverabilityRuntime(t *testing.T, model OperatingModelDocument, includePlanOfRecord, includeReadmeLink bool) OperatingGraphRuntime {
	t.Helper()
	repoRoot := t.TempDir()
	readmePath := operatingModelTeamReadmePath(model.Source.Path)
	if readmePath != "" && includeReadmeLink {
		absReadmePath := filepath.Join(repoRoot, filepath.FromSlash(readmePath))
		if err := os.MkdirAll(filepath.Dir(absReadmePath), 0o755); err != nil {
			t.Fatalf("create README dir: %v", err)
		}
		if err := os.WriteFile(absReadmePath, []byte("See OPERATING_MODEL.md.\n"), 0o644); err != nil {
			t.Fatalf("write README fixture: %v", err)
		}
	}
	loaded := &LoadedTeamContract{TeamID: model.Team}
	members := []MemberTopics{{
		Ref: MemberRef{Team: model.Team, Member: "a"},
		Topics: Topics{
			Intake:            []IntakeEntry{{Prefix: "first/*"}},
			Output:            []OutputEntry{{Prefix: "first/*", DestinationKind: DestinationKnowledge}},
			DecisionsOwned:    []string{"model-update"},
			DecisionsConsumed: []string{"model-update"},
			ExternalProducers: []string{"operator"},
		},
		Exists: true,
	}}
	loaded.Contract = &teamcontract.OperatingContract{
		DecisionContext: map[string]teamcontract.DecisionContext{"model-update": {OwnerMemberIDs: []string{"a"}}},
		Members:         map[string]teamcontract.MemberContract{"a": {}},
	}
	loaded.TopicCatalog = []TopicCatalogEntry{{
		Prefix:  "first/*",
		Status:  "live",
		Purpose: "First.",
	}}
	if includePlanOfRecord {
		loaded.PlanOfRecordDocuments = []teamcontract.PlanOfRecordDocument{{
			ID: "operating-model",
			Paths: []teamcontract.PathRef{{
				Base: "repo-root",
				Path: model.Source.Path,
			}},
		}}
	}
	return OperatingGraphRuntime{
		RepoRoot:       repoRoot,
		Members:        members,
		PromptSections: derivedTopicContractPromptSections(members, TeamContractRegistry{model.Team: loaded}),
		Contracts: TeamContractRegistry{
			model.Team: loaded,
		},
	}
}

func TestValidateOperatingModelsRejectsUnbackedExternalInputs(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
	models[0].Sections.ExternalInputs.Rows[0].EntrySurface = "`topic:ghost-inbox/*`"
	models[0].Sections.ExternalInputs.Rows[0].Drainer = "member:ghost"

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_external_inputs_producer_unbacked")
	assertOperatingFinding(t, result, "operating_model_external_inputs_entry_unbacked")
	assertOperatingFinding(t, result, "operating_model_external_inputs_drainer_unbacked")
}

func TestValidateOperatingModelsRejectsUnbackedOutputs(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
	models[0].Sections.Outputs.Rows[0].Consumer = "member:ghost"

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_outputs_surface_unbacked")
	assertOperatingFinding(t, result, "operating_model_outputs_consumer_unbacked")
}

func TestOperatingModelReferenceIndexNormalizesDocumentReferences(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Sections.Outputs.Rows[0].Surface = "`topic:first/*`"
	index := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{})

	assertOperatingModelReference(t, index, OperatingModelReferenceKindTopic, "", "first/*", "topic_catalog")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindDecision, "", "model-update", "decisions")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindMember, "", "a", "topic_catalog")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindTopic, OperatingGraphQualifierFuture, "second/*", "gaps")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindCommand, "", "prompt-manager graph operating-model validate --team team-a --id g", "adoption")

	inputAssurance := index.ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if !inputAssurance.Backed() {
		t.Fatalf("expected fixture external input row to be fully backed: %+v", inputAssurance)
	}
	outputAssurance := index.OutputAssurance(model.Sections.Outputs.Rows[0])
	if !outputAssurance.Backed() {
		t.Fatalf("expected fixture output row to be fully backed: %+v", outputAssurance)
	}
	if !index.FeedbackReferenceAssurance("topic:first/*").Backed {
		t.Fatalf("expected feedback topic reference to be backed")
	}
}

func TestOperatingModelReferenceIndexOwnsCoverageAssurance(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
	model.Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
	model.Sections.FeedbackLoop.Steps[0].References = []string{"topic:first/*", "topic:never-seen/*"}
	index := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{})

	inputBacked, inputUnbacked := externalInputsCoverageCounts(index)
	if inputBacked != 0 || inputUnbacked != 1 {
		t.Fatalf("expected coverage to count reference-index external input assurance, got backed=%d unbacked=%d", inputBacked, inputUnbacked)
	}
	outputBacked, outputUnbacked := outputsCoverageCounts(index)
	if outputBacked != 0 || outputUnbacked != 1 {
		t.Fatalf("expected coverage to count reference-index output assurance, got backed=%d unbacked=%d", outputBacked, outputUnbacked)
	}
	steps, anchored, unbackedRefs := feedbackLoopCoverageCounts(index)
	if steps != 1 || anchored != 1 || unbackedRefs != 1 {
		t.Fatalf("expected coverage to count reference-index feedback assurance, got steps=%d anchored=%d unbacked=%d", steps, anchored, unbackedRefs)
	}
}

func TestOperatingModelReferenceIndexExpandsRuntimeActorGroups(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Graphs[0].Metadata.Extra = map[string]string{
		"actor_alias.available-drainers": "group:available-drainers",
		"actor_group.available-drainers": "team-members",
	}
	model.Sections.ExternalInputs.Rows[0].Drainer = "available-drainers"

	withoutRuntime := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{}).ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if withoutRuntime.Drainer {
		t.Fatalf("expected unresolved team-members group to leave drainer unbacked: %+v", withoutRuntime)
	}

	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{"a": {}},
			}},
		},
	}
	withRuntime := NewOperatingModelReferenceIndex(model, runtime).ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if !withRuntime.Drainer {
		t.Fatalf("expected runtime-expanded team-members group to back drainer: %+v", withRuntime)
	}
}

func assertOperatingModelReference(t *testing.T, index OperatingModelReferenceIndex, kind OperatingModelReferenceKind, qualifier OperatingGraphQualifier, value, surface string) {
	t.Helper()
	for _, ref := range index.References {
		if ref.Kind == kind && ref.Qualifier == qualifier && ref.Value == value && ref.Surface == surface {
			return
		}
	}
	t.Fatalf("missing normalized reference kind=%s qualifier=%s value=%q surface=%q in %+v", kind, qualifier, value, surface, index.References)
}
