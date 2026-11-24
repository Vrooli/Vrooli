package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	apprun "test-genie/internal/app/runs"

	"github.com/gorilla/mux"
)

const experienceCohortRequestMaxBytes = 1 << 20

func (s *Server) handlePersistExperienceCohortReceipt(w http.ResponseWriter, r *http.Request) {
	if s.runsService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "run evidence store is unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, experienceCohortRequestMaxBytes)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input apprun.ExperienceCohortInput
	if err := decoder.Decode(&input); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid experience cohort receipt request")
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		s.writeError(w, http.StatusBadRequest, "experience cohort receipt request must contain one JSON object")
		return
	}
	stored, err := s.runsService.PersistExperienceCohortReceipt(r.Context(), input)
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusRequestTimeout
		}
		s.writeError(w, status, err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, stored)
}

func (s *Server) handleGetExperienceCohortReceipt(w http.ResponseWriter, r *http.Request) {
	if s.runsService == nil {
		s.writeError(w, http.StatusServiceUnavailable, "run evidence store is unavailable")
		return
	}
	stored, err := s.runsService.GetExperienceCohortReceipt(r.Context(), mux.Vars(r)["receiptID"])
	if err != nil {
		s.writeError(w, http.StatusNotFound, "experience cohort receipt was not found")
		return
	}
	s.writeJSON(w, http.StatusOK, stored)
}
