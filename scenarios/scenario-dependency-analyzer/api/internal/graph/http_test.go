package graph

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	types "scenario-dependency-analyzer/internal/types"
)

type fakeGraphService struct {
	err error
}

func (s fakeGraphService) GenerateGraph(graphType string) (*types.DependencyGraph, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &types.DependencyGraph{
		Type: graphType,
		Nodes: []types.GraphNode{
			{ID: "core-a", Type: "scenario", Group: "scenarios"},
			{ID: "consumer", Type: "scenario", Group: "scenarios"},
		},
		Edges: []types.GraphEdge{
			{Source: "consumer", Target: "core-a", Type: "scenario", Required: true, Weight: 2},
		},
	}, nil
}

func (s fakeGraphService) GraphCentrality(coreSeeds []string, scenario string) (*types.GraphCentralityReport, error) {
	graph, err := s.GenerateGraph("combined")
	if err != nil {
		return nil, err
	}
	return CalculateCentrality(graph, coreSeeds, scenario), nil
}

func TestGetGraph(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), fakeGraphService{}, func() string { return t.TempDir() })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/combined", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response types.DependencyGraph
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Type != "combined" {
		t.Fatalf("graph type = %q, want combined", response.Type)
	}
}

func TestGetGraphRejectsInvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), fakeGraphService{}, func() string { return t.TempDir() })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/unknown", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestGetGraphCentrality(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), fakeGraphService{}, func() string { return t.TempDir() })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/centrality?scenario=consumer", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response types.GraphCentralityReport
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Scenario != "consumer" {
		t.Fatalf("scenario = %q, want consumer", response.Scenario)
	}
}

func TestDetectCycles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), fakeGraphService{}, func() string { return t.TempDir() })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/combined/cycles", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response CycleDetectionResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Severity != "none" {
		t.Fatalf("severity = %q, want none", response.Severity)
	}
}

func TestGraphAdapterMapsServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHTTPRoutes(router.Group("/api/v1"), fakeGraphService{err: errors.New("graph unavailable")}, func() string { return t.TempDir() })

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/graph/combined", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d. body: %s", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
	}
}
