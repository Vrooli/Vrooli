package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func certificationTestServer(t *testing.T) (*Server, *mux.Router) {
	t.Helper()
	srv := &Server{certificationEvidenceOverride: t.TempDir()}
	router := mux.NewRouter()
	srv.registerCertificationRoutes(router.PathPrefix("/api/v1").Subrouter())
	return srv, router
}

func TestCertificationMatrixEndpointExposesTwoGroundProfile(t *testing.T) {
	_, router := certificationTestServer(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/certification/matrix", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Matrix struct {
			MatrixRevision         string   `json:"matrix_revision"`
			TargetGroundVocabulary []string `json:"target_ground_vocabulary"`
			GroundApplicability    []any    `json:"ground_applicability"`
		} `json:"matrix"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode matrix response: %v", err)
	}
	if body.Matrix.MatrixRevision != "cloud-launch-v2-two-grounds" {
		t.Fatalf("matrix revision = %q", body.Matrix.MatrixRevision)
	}
	if strings.Join(body.Matrix.TargetGroundVocabulary, ",") != "qemu,real_vps" {
		t.Fatalf("target grounds = %v", body.Matrix.TargetGroundVocabulary)
	}
	if len(body.Matrix.GroundApplicability) != 266 {
		t.Fatalf("ground applicability entries = %d, want 266", len(body.Matrix.GroundApplicability))
	}
}

func TestCertificationReadinessRequiresCandidateAndReportsAllGroundCells(t *testing.T) {
	_, router := certificationTestServer(t)
	query := url.Values{
		"release_digest":       {"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		"configuration_digest": {"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		"closure_digest":       {"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/certification/readiness?"+query.Encode(), nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Ready         bool  `json:"ready"`
		RequiredCells int   `json:"required_cells"`
		MissingCells  []any `json:"missing_cells"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if body.Ready || body.RequiredCells != 532 || len(body.MissingCells) != 532 {
		t.Fatalf("readiness = ready:%v required:%d missing:%d; empty target evidence must leave all 532 cells open", body.Ready, body.RequiredCells, len(body.MissingCells))
	}
}

func TestCertificationReadinessRejectsIncompleteCandidate(t *testing.T) {
	_, router := certificationTestServer(t)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/certification/readiness?release_digest=sha256%3A111", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
}
