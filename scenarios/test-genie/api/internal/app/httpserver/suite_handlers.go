package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"test-genie/internal/queue"
	"test-genie/internal/shared"
)

func (s *Server) handleCreateSuiteRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload queue.QueueSuiteRequestInput
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	req, err := s.suiteRequests.Queue(r.Context(), payload)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("suite request insert failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to persist suite request")
		return
	}

	s.log("suite request created", map[string]interface{}{
		"id":       req.ID.String(),
		"scenario": req.ScenarioName,
		"types":    req.RequestedTypes,
	})
	s.writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleListSuiteRequests(w http.ResponseWriter, r *http.Request) {
	suites, err := s.suiteRequests.List(r.Context(), queue.MaxSuiteListPage)
	if err != nil {
		s.log("listing suite requests failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load suite requests")
		return
	}

	payload := map[string]interface{}{
		"items": suites,
		"count": len(suites),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleGetSuiteRequest(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	rawID := strings.TrimSpace(params["id"])
	if rawID == "" {
		s.writeError(w, http.StatusBadRequest, "suite request id is required")
		return
	}

	requestID, err := uuid.Parse(rawID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "suite request id must be a valid UUID")
		return
	}

	req, err := s.suiteRequests.Get(r.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "suite request not found")
			return
		}
		s.log("fetching suite request by id failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load requested suite")
		return
	}

	s.writeJSON(w, http.StatusOK, req)
}
