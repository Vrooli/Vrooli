package httpserver

import (
	"net/http"
)

func (s *Server) handleListPhases(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	descriptors := s.phaseCatalog.DescribePhases()
	payload := map[string]interface{}{
		"items": descriptors,
		"count": len(descriptors),
	}
	s.writeJSON(w, http.StatusOK, payload)
}
