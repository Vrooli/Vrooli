package memberflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func newRouter(h *Handlers) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/teams/{id}/members/{agentId}/topics", h.GetMember).Methods("GET")
	r.HandleFunc("/teams/{id}/members/{agentId}/topics", h.PutMember).Methods("PUT")
	r.HandleFunc("/teams/{id}/topics", h.GetTeam).Methods("GET")
	r.HandleFunc("/topics/graph", h.GetGraph).Methods("GET")
	r.HandleFunc("/operating-models", h.GetOperatingModels).Methods("GET")
	r.HandleFunc("/operating-models/validate", h.ValidateOperatingModelsHandler).Methods("GET")
	r.HandleFunc("/operating-models/diff", h.DiffOperatingModelsHandler).Methods("GET")
	r.HandleFunc("/operating-models/coverage", h.CoverageOperatingModelsHandler).Methods("GET")
	r.HandleFunc("/topics/drain-status", h.GetDrainStatus).Methods("GET")
	return r
}

func newStore(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return root
}

func newRepoStore(t *testing.T) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	storeDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store")
	if err := os.MkdirAll(filepath.Join(storeDir, "teams"), 0o755); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	return repoRoot, storeDir
}

func TestGetMember_NotPresent(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/teams/marketing-crew/members/researcher/topics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp MemberTopicsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Exists {
		t.Errorf("expected Exists=false")
	}
	if !resp.Topics.IsEmpty() {
		t.Errorf("expected empty Topics for non-existent file")
	}
}

func TestPutMember_RoundTrip(t *testing.T) {
	store := newStore(t)
	h := NewHandlers(store)
	r := newRouter(h)

	body := bytes.NewBufferString(`{
		"intake": [{"prefix": "research-inbox/*", "taxonomy": "marketing-research"}],
		"output": [{"prefix": "audience-scan/*", "destination_kind": "knowledge"}],
		"raises_capability_gaps": true
	}`)
	req := httptest.NewRequest("PUT", "/teams/marketing-crew/members/researcher/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body=%s", w.Code, w.Body.String())
	}

	// Read back
	req2 := httptest.NewRequest("GET", "/teams/marketing-crew/members/researcher/topics", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w2.Code)
	}
	var resp MemberTopicsResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Exists || !resp.Topics.RaisesCapabilityGaps {
		t.Errorf("round-trip lost data: %+v", resp)
	}
	if len(resp.Topics.Intake) != 1 || resp.Topics.Intake[0].Taxonomy != "marketing-research" {
		t.Errorf("intake mismatch: %+v", resp.Topics.Intake)
	}
}

func TestPutMember_RejectsMalformed(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	body := bytes.NewBufferString(`{ not json `)
	req := httptest.NewRequest("PUT", "/teams/t/members/m/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestPutMember_RejectsSchemaViolation(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	body := bytes.NewBufferString(`{"intake":[{"prefix":"*","taxonomy":"x"}]}`)
	req := httptest.NewRequest("PUT", "/teams/t/members/m/topics", body)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bare-* prefix, got %d", w.Code)
	}
}

func TestGetTeam_AggregatesMembers(t *testing.T) {
	store := newStore(t)
	h := NewHandlers(store)

	if err := WriteMember(store, "marketing-crew", "researcher", Topics{
		Intake: []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "marketing-research"}},
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
	if err := WriteMember(store, "marketing-crew", "publisher", Topics{}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}

	r := newRouter(h)
	req := httptest.NewRequest("GET", "/teams/marketing-crew/topics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp TeamTopicsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Team != "marketing-crew" {
		t.Errorf("team mismatch")
	}
	if len(resp.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(resp.Members))
	}
}

func TestGetGraph_BuildsExpectedNodes(t *testing.T) {
	store := newStore(t)
	if err := WriteMember(store, "marketing-crew", "researcher", Topics{
		Intake: []IntakeEntry{
			{Prefix: "research-inbox/*", Taxonomy: "marketing-research"},
		},
		Output: []OutputEntry{
			{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge},
		},
		ExternalProducers:    []string{"vision-walk", "operator"},
		DecisionsOwned:       []string{"audience-update"},
		RaisesCapabilityGaps: true,
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}

	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Member node + 2 external + 1 input prefix + 1 output prefix + 1 decision + 1 cap-gap = 7 nodes
	wantNodes := map[string]string{
		"member:marketing-crew/researcher": "member",
		"external:vision-walk":             "external",
		"external:operator":                "external",
		"prefix:research-inbox/*":          "knowledge_sink",
		"prefix:audience-scan/*":           "knowledge_sink",
		"decision:audience-update":         "decision",
		"capability-gap":                   "capability_gap",
	}
	if got := len(resp.Nodes); got != len(wantNodes) {
		t.Errorf("node count = %d, want %d (nodes=%+v)", got, len(wantNodes), resp.Nodes)
	}
	for _, n := range resp.Nodes {
		want, ok := wantNodes[n.ID]
		if !ok {
			t.Errorf("unexpected node %q (kind=%s)", n.ID, n.Kind)
			continue
		}
		if n.Kind != want {
			t.Errorf("node %q has kind=%q, want %q", n.ID, n.Kind, want)
		}
	}
}

func TestGetGraph_EmptyStore(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/graph", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Nodes) != 0 || len(resp.Edges) != 0 {
		t.Errorf("expected empty graph, got %+v", resp)
	}
}

func TestGetDrainStatus(t *testing.T) {
	store := newStore(t)
	r := newRouter(NewHandlers(store))
	req := httptest.NewRequest("GET", "/topics/drain-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status=%d", w.Code)
	}
}

func TestOperatingModelHandlersValidateAndDiffAgainstRuntime(t *testing.T) {
	repoRoot, storeDir := newRepoStore(t)
	writeOperatingGraphHandlerFixture(t, repoRoot, storeDir)

	r := newRouter(NewHandlers(storeDir))

	listReq := httptest.NewRequest("GET", "/operating-models?team=team-a&id=g", nil)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listW.Code, listW.Body.String())
	}
	var listResp OperatingModelListResponse
	if err := json.NewDecoder(listW.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Models) != 1 || listResp.Models[0].ID != "g" {
		t.Fatalf("unexpected list response: %+v", listResp)
	}

	validateReq := httptest.NewRequest("GET", "/operating-models/validate?team=team-a&id=g", nil)
	validateW := httptest.NewRecorder()
	r.ServeHTTP(validateW, validateReq)
	if validateW.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateW.Code, validateW.Body.String())
	}
	var validateResp OperatingModelValidationResponse
	if err := json.NewDecoder(validateW.Body).Decode(&validateResp); err != nil {
		t.Fatalf("decode validate: %v", err)
	}
	if validateResp.Validation.Errors != 2 {
		t.Fatalf("validation errors=%d, want 2: %+v", validateResp.Validation.Errors, validateResp.Validation.Findings)
	}
	assertOperatingFinding(t, validateResp.Validation, "graph_topic_unresolved")
	assertOperatingFinding(t, validateResp.Validation, "graph_edge_unbacked")

	diffReq := httptest.NewRequest("GET", "/operating-models/diff?team=team-a&id=g", nil)
	diffW := httptest.NewRecorder()
	r.ServeHTTP(diffW, diffReq)
	if diffW.Code != http.StatusOK {
		t.Fatalf("diff status=%d body=%s", diffW.Code, diffW.Body.String())
	}
	var diffResp OperatingModelDiffResponse
	if err := json.NewDecoder(diffW.Body).Decode(&diffResp); err != nil {
		t.Fatalf("decode diff: %v", err)
	}
	if got := countOperatingDiffs(diffResp.Diff, "graph_relationship_missing_in_runtime"); got != 1 {
		t.Fatalf("graph-to-runtime diff count=%d, want 1: %+v", got, diffResp.Diff)
	}

	coverageReq := httptest.NewRequest("GET", "/operating-models/coverage?team=team-a&id=g", nil)
	coverageW := httptest.NewRecorder()
	r.ServeHTTP(coverageW, coverageReq)
	if coverageW.Code != http.StatusOK {
		t.Fatalf("coverage status=%d body=%s", coverageW.Code, coverageW.Body.String())
	}
	var coverageResp OperatingModelCoverageResponse
	if err := json.NewDecoder(coverageW.Body).Decode(&coverageResp); err != nil {
		t.Fatalf("decode coverage: %v", err)
	}
	if len(coverageResp.Coverage) != 1 || coverageResp.Coverage[0].GraphID != "g" {
		t.Fatalf("unexpected coverage response: %+v", coverageResp)
	}
	topicRead := operatingCoverageByRelationship(coverageResp.Coverage[0].Relationships, string(operatingRelTopicRead))
	if topicRead.GraphShown != 2 || topicRead.RuntimeDeclared != 1 || topicRead.GraphOnly != 1 || topicRead.RuntimeOnly != 0 {
		t.Fatalf("unexpected topic_read coverage: %+v", topicRead)
	}
}

func TestOperatingModelHandlersReturnEmptyArraysForCleanResults(t *testing.T) {
	repoRoot, storeDir := newRepoStore(t)
	writeCleanOperatingGraphHandlerFixture(t, repoRoot, storeDir)

	r := newRouter(NewHandlers(storeDir))

	validateReq := httptest.NewRequest("GET", "/operating-models/validate?team=team-a&id=g", nil)
	validateW := httptest.NewRecorder()
	r.ServeHTTP(validateW, validateReq)
	if validateW.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateW.Code, validateW.Body.String())
	}
	if !bytes.Contains(validateW.Body.Bytes(), []byte(`"findings":[]`)) {
		t.Fatalf("validate response should render empty findings as []:\n%s", validateW.Body.String())
	}

	diffReq := httptest.NewRequest("GET", "/operating-models/diff?team=team-a&id=g", nil)
	diffW := httptest.NewRecorder()
	r.ServeHTTP(diffW, diffReq)
	if diffW.Code != http.StatusOK {
		t.Fatalf("diff status=%d body=%s", diffW.Code, diffW.Body.String())
	}
	if !bytes.Contains(diffW.Body.Bytes(), []byte(`"diff":[]`)) {
		t.Fatalf("diff response should render empty diff as []:\n%s", diffW.Body.String())
	}

	coverageReq := httptest.NewRequest("GET", "/operating-models/coverage?team=team-a&id=g", nil)
	coverageW := httptest.NewRecorder()
	r.ServeHTTP(coverageW, coverageReq)
	if coverageW.Code != http.StatusOK {
		t.Fatalf("coverage status=%d body=%s", coverageW.Code, coverageW.Body.String())
	}
	if !bytes.Contains(coverageW.Body.Bytes(), []byte(`"coverage":[`)) {
		t.Fatalf("coverage response should render coverage as []/array:\n%s", coverageW.Body.String())
	}
}

func TestOperatingModelHandlersValidatePromptSectionsFromProvider(t *testing.T) {
	repoRoot, storeDir := newRepoStore(t)
	writeCleanOperatingGraphHandlerFixture(t, repoRoot, storeDir)

	h := NewHandlers(storeDir)
	h.SetPromptSectionProvider(staticOperatingPromptSections{
		sections: map[MemberRef][]OperatingGraphPromptSection{},
	})
	r := newRouter(h)

	validateReq := httptest.NewRequest("GET", "/operating-models/validate?team=team-a&id=g", nil)
	validateW := httptest.NewRecorder()
	r.ServeHTTP(validateW, validateReq)
	if validateW.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateW.Code, validateW.Body.String())
	}
	var validateResp OperatingModelValidationResponse
	if err := json.NewDecoder(validateW.Body).Decode(&validateResp); err != nil {
		t.Fatalf("decode validate: %v", err)
	}
	assertOperatingFinding(t, validateResp.Validation, "graph_prompt_topic_contract_missing")

	coverageReq := httptest.NewRequest("GET", "/operating-models/coverage?team=team-a&id=g", nil)
	coverageW := httptest.NewRecorder()
	r.ServeHTTP(coverageW, coverageReq)
	if coverageW.Code != http.StatusOK {
		t.Fatalf("coverage status=%d body=%s", coverageW.Code, coverageW.Body.String())
	}
	var coverageResp OperatingModelCoverageResponse
	if err := json.NewDecoder(coverageW.Body).Decode(&coverageResp); err != nil {
		t.Fatalf("decode coverage: %v", err)
	}
	if got := coverageResp.Coverage[0].Prompts.TopicContractPresent; got != 0 {
		t.Fatalf("provider-backed prompt coverage present=%d, want 0", got)
	}
}

type staticOperatingPromptSections struct {
	sections map[MemberRef][]OperatingGraphPromptSection
}

func (p staticOperatingPromptSections) SectionsForMember(_ context.Context, team, member string) ([]OperatingGraphPromptSection, error) {
	return p.sections[MemberRef{Team: team, Member: member}], nil
}

func operatingCoverageByRelationship(rels []OperatingRelationshipCoverage, relationship string) OperatingRelationshipCoverage {
	for _, rel := range rels {
		if rel.Relationship == relationship {
			return rel
		}
	}
	return OperatingRelationshipCoverage{}
}

func writeOperatingGraphHandlerFixture(t *testing.T, repoRoot, storeDir string) {
	t.Helper()
	docsDir := filepath.Join(repoRoot, "docs", "test")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	graphDoc := `## Mission

Fixture mission.

## Scope

Fixture scope.

## Operating Loops

1. Drain fixture input.

## Operating Graph

<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
-->
` + "```mermaid" + `
flowchart LR
  %% @node M member:member-a
  M[Member A]
  %% @node OP external:operator
  OP([Operator])
  %% @node IN topic:research-inbox/*
  IN[(research-inbox/*)]
  %% @node NOTE topic:marketing-craft-observation/*
  NOTE[(marketing-craft-observation/*)]
  %% @node D decision:model-update
  D{model-update}
  OP --> IN
  OP --> M
  IN --> M
  NOTE --> M
  M --> D
` + "```" + `
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| ` + "`topic:research-inbox/*`" + ` | live | external:operator | member:member-a | Intake. |
| ` + "`topic:marketing-craft-observation/*`" + ` | live | | | Typed observation. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| ` + "`model-update`" + ` | member-a | Update fixture operating model. | ` + "`topic:research-inbox/*`" + ` evidence. | Team operating-model document changes. |

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | ` + "`topic:research-inbox/*`" + ` | member-a | Member drains operator input. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Fixture intake | ` + "`research-inbox/*`" + ` | member-a | Keep fixture intake queryable. |

## Feedback / Capability Improvement Loop

1. Review ` + "`topic:research-inbox/*`" + `.

## Current Implementation Gaps

1. ` + "`topic[future]:fixture-target/*`" + ` remains target-state until a fixture producer exists.

## Adoption / Validation

- ` + "`prompt-manager graph operating-model validate --team team-a --id g`" + `
- ` + "`prompt-manager graph operating-model diff --team team-a --id g`" + `
- ` + "`prompt-manager graph operating-model coverage --team team-a --id g`" + `
`
	if err := os.WriteFile(filepath.Join(docsDir, "OPERATING_MODEL.md"), []byte(graphDoc), 0o644); err != nil {
		t.Fatalf("write graph doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("[`OPERATING_MODEL.md`](OPERATING_MODEL.md)\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "teams", "team-a"), 0o755); err != nil {
		t.Fatalf("mkdir team: %v", err)
	}
	teamJSON := `{
  "id": "team-a",
  "topicCatalog": [
    {
      "prefix": "research-inbox/*",
      "status": "live",
      "purpose": "Intake."
    },
    {
      "prefix": "marketing-craft-observation/*",
      "status": "live",
      "purpose": "Typed observation."
    }
  ],
  "operatingContract": {
    "schemaVersion": 1,
    "documents": {
      "planOfRecord": [
        {
          "id": "g",
          "paths": [
            {
              "base": "repo-root",
              "path": "docs/test/OPERATING_MODEL.md"
            }
          ],
          "writePolicy": "operator-curated-via-decisions"
        }
      ],
      "sharedState": []
    },
    "decisionContexts": {
      "model-update": {
        "description": "Update fixture operating model."
      }
    },
    "knowledgeTopics": {},
    "members": {
      "member-a": {}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(storeDir, "teams", "team-a", "team.json"), []byte(teamJSON), 0o644); err != nil {
		t.Fatalf("write team.json: %v", err)
	}
	if err := WriteMember(storeDir, "team-a", "member-a", Topics{
		Intake:            []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "marketing-research"}},
		ExternalProducers: []string{"operator"},
		DecisionsOwned:    []string{"model-update"},
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
}

func writeCleanOperatingGraphHandlerFixture(t *testing.T, repoRoot, storeDir string) {
	t.Helper()
	docsDir := filepath.Join(repoRoot, "docs", "test")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	graphDoc := `## Mission

Fixture mission.

## Scope

Fixture scope.

## Operating Loops

1. Drain fixture input.

## Operating Graph

<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
-->
` + "```mermaid" + `
flowchart LR
  %% @node M member:member-a
  M[Member A]
  %% @node OP external:operator
  OP([Operator])
  %% @node IN topic:research-inbox/*
  IN[(research-inbox/*)]
  %% @node D decision:model-update
  D{model-update}
  OP --> IN
  OP --> M
  IN --> M
  M --> D
` + "```" + `
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| ` + "`topic:research-inbox/*`" + ` | live | external:operator | member:member-a | Intake. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| ` + "`model-update`" + ` | member-a | Update fixture operating model. | ` + "`topic:research-inbox/*`" + ` evidence. | Team operating-model document changes. |

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | ` + "`topic:research-inbox/*`" + ` | member-a | Member drains operator input. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Fixture intake | ` + "`research-inbox/*`" + ` | member-a | Keep fixture intake queryable. |

## Feedback / Capability Improvement Loop

1. Review ` + "`topic:research-inbox/*`" + `.

## Current Implementation Gaps

1. ` + "`topic[future]:fixture-target/*`" + ` remains target-state until a fixture producer exists.

## Adoption / Validation

- ` + "`prompt-manager graph operating-model validate --team team-a --id g`" + `
- ` + "`prompt-manager graph operating-model diff --team team-a --id g`" + `
- ` + "`prompt-manager graph operating-model coverage --team team-a --id g`" + `
`
	if err := os.WriteFile(filepath.Join(docsDir, "OPERATING_MODEL.md"), []byte(graphDoc), 0o644); err != nil {
		t.Fatalf("write graph doc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("[`OPERATING_MODEL.md`](OPERATING_MODEL.md)\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(storeDir, "teams", "team-a"), 0o755); err != nil {
		t.Fatalf("mkdir team: %v", err)
	}
	teamJSON := `{
  "id": "team-a",
  "topicCatalog": [
    {
      "prefix": "research-inbox/*",
      "status": "live",
      "purpose": "Intake."
    }
  ],
  "operatingContract": {
    "schemaVersion": 1,
    "documents": {
      "planOfRecord": [
        {
          "id": "g",
          "paths": [
            {
              "base": "repo-root",
              "path": "docs/test/OPERATING_MODEL.md"
            }
          ],
          "writePolicy": "operator-curated-via-decisions"
        }
      ],
      "sharedState": []
    },
    "decisionContexts": {
      "model-update": {
        "description": "Update fixture operating model."
      }
    },
    "knowledgeTopics": {},
    "members": {
      "member-a": {}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(storeDir, "teams", "team-a", "team.json"), []byte(teamJSON), 0o644); err != nil {
		t.Fatalf("write team.json: %v", err)
	}
	if err := WriteMember(storeDir, "team-a", "member-a", Topics{
		Intake:            []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "marketing-research"}},
		ExternalProducers: []string{"operator"},
		DecisionsOwned:    []string{"model-update"},
	}); err != nil {
		t.Fatalf("WriteMember: %v", err)
	}
}
