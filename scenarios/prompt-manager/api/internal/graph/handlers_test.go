package graph

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// ---------------------------------------------------------------------------
// GetGraph tests
// ---------------------------------------------------------------------------

func TestGetGraph_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent, Label: "Bot"}},
			[]Edge{{From: "n1", To: "s1", Kind: EdgeCLIRead}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph", nil)
	h.GetGraph(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var idx GraphIndex
	if err := json.Unmarshal(rr.Body.Bytes(), &idx); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(idx.Graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(idx.Graph.Nodes))
	}
}

func TestGetGraph_Error(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{getErr: errors.New("get fail")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph", nil)
	h.GetGraph(rr, req)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Regenerate tests
// ---------------------------------------------------------------------------

func TestRegenerate_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent}},
			nil, nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/graph/regenerate", nil)
	h.Regenerate(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var idx GraphIndex
	if err := json.Unmarshal(rr.Body.Bytes(), &idx); err != nil {
		t.Fatalf("decode error: %v", err)
	}
}

func TestRegenerate_RegenError(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{regenErr: errors.New("regen fail")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/graph/regenerate", nil)
	h.Regenerate(rr, req)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestRegenerate_GetError(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{getErr: errors.New("get fail after regen")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/graph/regenerate", nil)
	h.Regenerate(rr, req)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetOrphans tests
// ---------------------------------------------------------------------------

func TestGetOrphans_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "s1", Type: NodeSkill, Label: "Orphan"},
				{ID: "s2", Type: NodeSkill, Label: "Referenced"},
			},
			[]Edge{{From: "a1", To: "s2", Kind: EdgeCLIRead}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/orphans", nil)
	h.GetOrphans(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "s1" {
		t.Errorf("expected orphan s1, got %+v", nodes)
	}
}

func TestGetOrphans_Empty(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(nil, nil, nil),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/orphans", nil)
	h.GetOrphans(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected empty list, got %d", len(nodes))
	}
}

func TestGetOrphans_Error(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{getErr: errors.New("fail")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/orphans", nil)
	h.GetOrphans(rr, req)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetSkillless tests
// ---------------------------------------------------------------------------

func TestGetSkillless_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "a1", Type: NodeAgent, Label: "Skillless"},
				{ID: "a2", Type: NodeAgent, Label: "Has Skill"},
			},
			[]Edge{{From: "a2", To: "s1", Kind: EdgeCLIRead}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/skillless", nil)
	h.GetSkillless(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "a1" {
		t.Errorf("expected skillless a1, got %+v", nodes)
	}
}

// ---------------------------------------------------------------------------
// GetEmptyTeams tests
// ---------------------------------------------------------------------------

func TestGetEmptyTeams_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "t1", Type: NodeTeam, Label: "Empty"},
				{ID: "t2", Type: NodeTeam, Label: "Has Members"},
			},
			[]Edge{{From: "t2", To: "a1", Kind: EdgeMembership}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/empty-teams", nil)
	h.GetEmptyTeams(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "t1" {
		t.Errorf("expected empty team t1, got %+v", nodes)
	}
}

// ---------------------------------------------------------------------------
// GetUnaffiliated tests
// ---------------------------------------------------------------------------

func TestGetUnaffiliated_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "a1", Type: NodeAgent, Label: "Lone Wolf"},
				{ID: "a2", Type: NodeAgent, Label: "Team Member"},
			},
			[]Edge{{From: "t1", To: "a2", Kind: EdgeMembership}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/unaffiliated", nil)
	h.GetUnaffiliated(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "a1" {
		t.Errorf("expected unaffiliated a1, got %+v", nodes)
	}
}

// ---------------------------------------------------------------------------
// GetPopular tests
// ---------------------------------------------------------------------------

func TestGetPopular_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "s1", Type: NodeSkill, Label: "Popular"},
				{ID: "s2", Type: NodeSkill, Label: "Less Popular"},
			},
			[]Edge{
				{From: "a1", To: "s1", Kind: EdgeCLIRead},
				{From: "a2", To: "s1", Kind: EdgeCLIRead},
				{From: "a1", To: "s2", Kind: EdgeCLIRead},
			},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/popular", nil)
	h.GetPopular(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) < 1 {
		t.Fatal("expected at least 1 popular node")
	}
	if nodes[0].ID != "s1" {
		t.Errorf("expected most popular s1 first, got %s", nodes[0].ID)
	}
}

func TestGetPopular_CustomLimit(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "s1", Type: NodeSkill},
				{ID: "s2", Type: NodeSkill},
				{ID: "s3", Type: NodeSkill},
			},
			[]Edge{
				{From: "a1", To: "s1", Kind: EdgeCLIRead},
				{From: "a2", To: "s2", Kind: EdgeCLIRead},
				{From: "a3", To: "s3", Kind: EdgeCLIRead},
			},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/popular?limit=1", nil)
	h.GetPopular(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var nodes []Node
	if err := json.Unmarshal(rr.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("expected 1 node with limit=1, got %d", len(nodes))
	}
}

// ---------------------------------------------------------------------------
// GetCycles tests
// ---------------------------------------------------------------------------

func TestGetCycles_NoCycles(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "s1", Type: NodeSkill},
				{ID: "s2", Type: NodeSkill},
			},
			[]Edge{{From: "s1", To: "s2", Kind: EdgeCLIRead}},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/cycles", nil)
	h.GetCycles(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var cycles [][]string
	if err := json.Unmarshal(rr.Body.Bytes(), &cycles); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(cycles) != 0 {
		t.Errorf("expected no cycles, got %v", cycles)
	}
}

func TestGetCycles_WithCycles(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{
				{ID: "s1", Type: NodeSkill},
				{ID: "s2", Type: NodeSkill},
			},
			[]Edge{
				{From: "s1", To: "s2", Kind: EdgeCLIRead},
				{From: "s2", To: "s1", Kind: EdgeCLIRead},
			},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/cycles", nil)
	h.GetCycles(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var cycles [][]string
	if err := json.Unmarshal(rr.Body.Bytes(), &cycles); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(cycles) == 0 {
		t.Error("expected at least 1 cycle")
	}
}

// ---------------------------------------------------------------------------
// GetNode tests
// ---------------------------------------------------------------------------

func TestGetNode_Found(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent, Label: "Bot"}},
			[]Edge{{From: "n1", To: "s1", Kind: EdgeCLIRead}},
			[]HealthScore{{NodeID: "n1", Score: 0.8, Factors: map[string]float64{"test": 0.8}}},
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/nodes/n1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "n1"})
	h.GetNode(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var detail nodeDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if detail.Node.ID != "n1" {
		t.Errorf("expected node n1, got %s", detail.Node.ID)
	}
	if len(detail.AdjacentEdges) != 1 {
		t.Errorf("expected 1 adjacent edge, got %d", len(detail.AdjacentEdges))
	}
	if detail.HealthScore == nil || detail.HealthScore.Score != 0.8 {
		t.Errorf("expected health score 0.8, got %+v", detail.HealthScore)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent}},
			nil, nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/nodes/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	h.GetNode(rr, req)

	if rr.Code != 404 {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetNode_Error(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{getErr: errors.New("fail")})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/nodes/n1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "n1"})
	h.GetNode(rr, req)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetNodeEdges tests
// ---------------------------------------------------------------------------

func TestGetNodeEdges_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent}},
			[]Edge{
				{From: "n1", To: "s1", Kind: EdgeCLIRead},
				{From: "t1", To: "n1", Kind: EdgeMembership},
				{From: "s1", To: "s2", Kind: EdgeDefaultScope}, // not adjacent to n1
			},
			nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/nodes/n1/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "n1"})
	h.GetNodeEdges(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var edges []Edge
	if err := json.Unmarshal(rr.Body.Bytes(), &edges); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("expected 2 adjacent edges, got %d", len(edges))
	}
}

func TestGetNodeEdges_NoEdges(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			[]Node{{ID: "n1", Type: NodeAgent}},
			nil, nil,
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/nodes/n1/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "n1"})
	h.GetNodeEdges(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var edges []Edge
	if err := json.Unmarshal(rr.Body.Bytes(), &edges); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

// ---------------------------------------------------------------------------
// GetHealthScores tests
// ---------------------------------------------------------------------------

func TestGetHealthScores_Success(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(
			nil, nil,
			[]HealthScore{
				{NodeID: "n1", Score: 0.9, Factors: map[string]float64{"f1": 0.9}},
				{NodeID: "n2", Score: 0.5, Factors: map[string]float64{"f1": 0.5}},
			},
		),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/health", nil)
	h.GetHealthScores(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var scores []HealthScore
	if err := json.Unmarshal(rr.Body.Bytes(), &scores); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(scores) != 2 {
		t.Errorf("expected 2 scores, got %d", len(scores))
	}
}

func TestGetHealthScores_Empty(t *testing.T) {
	h := NewHandlers(&mockGraphIndexProvider{
		idx: testIndex(nil, nil, nil),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/health", nil)
	h.GetHealthScores(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var scores []HealthScore
	if err := json.Unmarshal(rr.Body.Bytes(), &scores); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(scores))
	}
}

func TestGetHealthConfig_Success(t *testing.T) {
	cfg := DefaultHealthConfig()
	h := NewHandlers(
		&mockGraphIndexProvider{idx: testIndex(nil, nil, nil)},
		&mockHealthConfigStore{cfg: cfg},
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/graph/health-config", nil)
	h.GetHealthConfig(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var got HealthConfig
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.Team.OutgoingEdges != cfg.Team.OutgoingEdges {
		t.Fatalf("expected team outgoing %f, got %f", cfg.Team.OutgoingEdges, got.Team.OutgoingEdges)
	}
}

func TestPutHealthConfig_Success(t *testing.T) {
	idx := &mockGraphIndexProvider{idx: testIndex(nil, nil, nil)}
	cfgStore := &mockHealthConfigStore{cfg: DefaultHealthConfig()}
	h := NewHandlers(idx, cfgStore)

	body := `{
		"team":{"outgoingEdges":0.9,"incomingEdges":1,"codeUsage":0.5,"recentActivity":0.5},
		"agent":{"outgoingEdges":1,"incomingEdges":1,"codeUsage":0.5,"recentActivity":0.5},
		"skill":{"outgoingEdges":1,"incomingEdges":1,"codeUsage":0.5,"recentActivity":0.5},
		"cli":{"neutralCommands":["vrooli"],"externalToolScore":0,"scenarioFallbackScore":0}
	}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/graph/health-config", strings.NewReader(body))
	h.PutHealthConfig(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if cfgStore.lastPut == nil {
		t.Fatalf("expected config to be saved")
	}
	if cfgStore.lastPut.Team.OutgoingEdges != 0.9 {
		t.Fatalf("expected updated outgoingEdges, got %f", cfgStore.lastPut.Team.OutgoingEdges)
	}
}

func TestPutHealthConfig_ValidationError(t *testing.T) {
	h := NewHandlers(
		&mockGraphIndexProvider{idx: testIndex(nil, nil, nil)},
		&mockHealthConfigStore{cfg: DefaultHealthConfig()},
	)

	body := `{
		"team":{"outgoingEdges":0,"incomingEdges":0,"codeUsage":0,"recentActivity":0},
		"agent":{"outgoingEdges":1,"incomingEdges":1,"codeUsage":0.5,"recentActivity":0.5},
		"skill":{"outgoingEdges":1,"incomingEdges":1,"codeUsage":0.5,"recentActivity":0.5},
		"cli":{"neutralCommands":["vrooli"],"externalToolScore":0,"scenarioFallbackScore":0}
	}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/api/v1/graph/health-config", strings.NewReader(body))
	h.PutHealthConfig(rr, req)

	if rr.Code != 400 {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
