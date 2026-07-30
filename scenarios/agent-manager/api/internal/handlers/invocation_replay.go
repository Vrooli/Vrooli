package handlers

import (
	"net/http"
	"strconv"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// ReplayInvocationFacts rebuilds one durable run projection from retained
// events. An unreplayable response is successful and preserves old facts.
func (h *Handler) ReplayInvocationFacts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	result, err := h.svc.ReplayInvocationFacts(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func invocationReadModelFilter(r *http.Request) (invocationreadmodel.Filter, error) {
	q := r.URL.Query()
	filter := invocationreadmodel.Filter{Ownership: q.Get("ownership"), Outcome: q.Get("outcome"), Executable: q.Get("executable"), Fingerprint: q.Get("fingerprint"), ProfileID: q.Get("profile_id"), RunnerType: q.Get("runner_type"), Model: q.Get("model"), TagPrefix: q.Get("tag_prefix"), RunStatus: q.Get("run_status")}
	for _, target := range []struct {
		raw  string
		into **time.Time
	}{{q.Get("from"), &filter.From}, {q.Get("to"), &filter.To}} {
		if target.raw != "" {
			value, err := time.Parse(time.RFC3339, target.raw)
			if err != nil {
				return filter, err
			}
			*target.into = &value
		}
	}
	return filter, nil
}

func (h *Handler) AggregateInvocationFacts(w http.ResponseWriter, r *http.Request) {
	filter, err := invocationReadModelFilter(r)
	if err != nil {
		writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
		return
	}
	dimension := r.URL.Query().Get("dimension")
	if dimension == "" {
		writeSimpleError(w, r, "dimension", "dimension is required")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 {
			writeSimpleError(w, r, "limit", "limit must be a positive integer")
			return
		}
	}
	rows, err := h.svc.AggregateInvocationFacts(r.Context(), filter, dimension, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *Handler) SelectInvocationCohort(w http.ResponseWriter, r *http.Request) {
	filter, err := invocationReadModelFilter(r)
	if err != nil {
		writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 {
			writeSimpleError(w, r, "limit", "limit must be a positive integer")
			return
		}
	}
	cohort, err := h.svc.SelectInvocationCohort(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cohort)
}

func (h *Handler) InvocationMetrics(w http.ResponseWriter, r *http.Request) {
	filter, err := invocationReadModelFilter(r)
	if err != nil {
		writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
		return
	}
	metrics, err := h.svc.InvocationMetrics(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func (h *Handler) RefreshInvocationFacts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	result, err := h.svc.RefreshInvocationFacts(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ReplayInvocationCorpus(w http.ResponseWriter, r *http.Request) {
	filter := orchestration.ReplayFilter{TagPrefix: r.URL.Query().Get("tag_prefix")}
	if raw := r.URL.Query().Get("profile_id"); raw != "" {
		profileID, err := uuid.Parse(raw)
		if err != nil {
			writeSimpleError(w, r, "profile_id", "invalid UUID format for profile ID")
			return
		}
		filter.ProfileID = &profileID
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			writeSimpleError(w, r, "limit", "limit must be a positive integer")
			return
		}
		filter.Limit = value
	}
	for _, target := range []struct {
		raw  string
		into **time.Time
	}{{r.URL.Query().Get("from"), &filter.From}, {r.URL.Query().Get("to"), &filter.To}} {
		if target.raw != "" {
			value, err := time.Parse(time.RFC3339, target.raw)
			if err != nil {
				writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
				return
			}
			*target.into = &value
		}
	}
	if raw := r.URL.Query().Get("status"); raw != "" {
		status := domain.RunStatus(raw)
		filter.Status = &status
	}
	result, err := h.svc.ReplayInvocationCorpus(r.Context(), filter, r.URL.Query().Get("mode") == "refresh")
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
