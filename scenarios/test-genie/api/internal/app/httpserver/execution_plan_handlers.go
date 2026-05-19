package httpserver

import (
	"errors"
	"net/http"

	"test-genie/internal/shared"
)

func (s *Server) handlePreviewExecutionPlan(w http.ResponseWriter, r *http.Request) {
	input, err := decodeSuiteExecutionInput(r)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if s.executionPlanner == nil {
		s.writeError(w, http.StatusInternalServerError, "execution planner unavailable")
		return
	}

	preview, err := s.executionPlanner.Preview(r.Context(), input.Request)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("execution preview failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to build execution plan")
		return
	}

	s.writeJSON(w, http.StatusOK, preview)
}
