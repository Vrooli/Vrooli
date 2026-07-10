package httpserver

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// handleGetRunArtifactByID streams bytes only after the opaque ID resolves in
// the selected run's verified catalog. Filesystem paths never enter the request
// or response contract.
func (s *Server) handleGetRunArtifactByID(w http.ResponseWriter, r *http.Request) {
	if s.runsService == nil {
		s.writeError(w, http.StatusNotFound, "runs service unavailable")
		return
	}
	vars := mux.Vars(r)
	artifactID := strings.TrimSpace(vars["artifactId"])
	if !validOpaqueArtifactID(artifactID) {
		s.writeError(w, http.StatusBadRequest, "invalid artifact id")
		return
	}
	artifact, abs, err := s.runsService.ResolveArtifactByID(vars["name"], vars["runId"], artifactID)
	if err != nil {
		switch {
		case errors.Is(err, sharedartifacts.ErrArtifactNotFound), os.IsNotExist(err):
			s.writeError(w, http.StatusNotFound, "artifact not found")
		case errors.Is(err, sharedartifacts.ErrUnsafeArtifact):
			s.writeError(w, http.StatusForbidden, "artifact reference is unsafe")
		case errors.Is(err, sharedartifacts.ErrInvalidArtifactCatalog), errors.Is(err, sharedartifacts.ErrUnsupportedArtifactCatalogVersion):
			s.writeError(w, http.StatusConflict, "artifact catalog is invalid")
		default:
			s.writeError(w, http.StatusBadRequest, "artifact unavailable")
		}
		return
	}
	if artifact.MediaType != "" {
		w.Header().Set("Content-Type", artifact.MediaType)
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// HTML/DOM evidence remains inert even when opened directly on the API
	// origin. Media and JSON/text artifacts are unaffected by this sandbox.
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	http.ServeFile(w, r, abs)
}

func validOpaqueArtifactID(id string) bool {
	if !strings.HasPrefix(id, "artifact_") || len(id) != len("artifact_")+32 {
		return false
	}
	for _, char := range strings.TrimPrefix(id, "artifact_") {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

// handleGetRunArtifact is the legacy relative-path route. New consumers use
// handleGetRunArtifactByID; this remains temporarily for the Phase 5 GCT
// consumer cutover and is not advertised by the typed artifact catalog.
func (s *Server) handleGetRunArtifact(w http.ResponseWriter, r *http.Request) {
	if s.runsService == nil {
		s.writeError(w, http.StatusNotFound, "runs service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenario := vars["name"]
	runID := vars["runId"]
	relPath := r.URL.Query().Get("path")

	abs, err := s.runsService.ResolveArtifact(scenario, runID, relPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// http.ServeContent infers most types from the extension; pin the common
	// video container explicitly since mime tables vary across hosts.
	if strings.EqualFold(filepath.Ext(abs), ".webm") {
		w.Header().Set("Content-Type", "video/webm")
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, abs)
}
