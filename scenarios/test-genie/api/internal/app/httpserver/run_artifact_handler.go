package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// handleGetRunArtifact streams a run's recorded artifact (video) by a
// run-relative path supplied as ?path=. The runs service resolves and
// traversal-guards the path; http.ServeFile handles range requests and caching
// so the browser <video> element can seek.
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
