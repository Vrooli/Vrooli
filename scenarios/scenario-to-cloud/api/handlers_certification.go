package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/certification"
	"scenario-to-cloud/internal/httputil"
)

// registerCertificationRoutes exposes the authoritative matrix and its
// candidate-specific readiness report. It is read-only: only evidence-owner
// receipts can change the report.
func (s *Server) registerCertificationRoutes(api *mux.Router) {
	api.HandleFunc("/certification/matrix", s.handleGetCertificationMatrix).Methods("GET")
	api.HandleFunc("/certification/readiness", s.handleGetCertificationReadiness).Methods("GET")
}

func (s *Server) certificationEvidenceDir() (string, error) {
	if s != nil && strings.TrimSpace(s.certificationEvidenceOverride) != "" {
		return s.certificationEvidenceOverride, nil
	}
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return "", err
	}
	scenariosDir, err := bundle.ResolveScenariosDir(repoRoot)
	if err != nil {
		return "", err
	}
	return filepath.Join(scenariosDir, "scenario-to-cloud", "certification", "evidence"), nil
}

func (s *Server) handleGetCertificationMatrix(w http.ResponseWriter, _ *http.Request) {
	matrix, err := certification.LoadEmbedded()
	if err != nil {
		apierrors.Write(w, apierrors.Internal("certification matrix unavailable", err))
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"matrix": matrix})
}

func (s *Server) handleGetCertificationReadiness(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	candidate := certification.Candidate{
		ReleaseDigest:       strings.TrimSpace(query.Get("release_digest")),
		ConfigurationDigest: strings.TrimSpace(query.Get("configuration_digest")),
		ClosureDigest:       strings.TrimSpace(query.Get("closure_digest")),
	}
	if candidate.ReleaseDigest == "" || candidate.ConfigurationDigest == "" || candidate.ClosureDigest == "" {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "release_digest, configuration_digest and closure_digest are required"))
		return
	}
	evidenceDir, err := s.certificationEvidenceDir()
	if err != nil {
		apierrors.Write(w, apierrors.Internal("certification evidence unavailable", err))
		return
	}
	body, err := certification.Report(evidenceDir, candidate, time.Now().UTC())
	if err != nil {
		apierrors.Write(w, apierrors.Internal("certification readiness unavailable", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
