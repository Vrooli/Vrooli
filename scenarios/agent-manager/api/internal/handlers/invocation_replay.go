package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runsignal"

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

func (h *Handler) invocationReadModelFilter(r *http.Request) (invocationreadmodel.Filter, error) {
	q := r.URL.Query()
	filter := invocationreadmodel.Filter{Ownership: q.Get("ownership"), Outcome: q.Get("outcome"), Executable: q.Get("executable"), Fingerprint: q.Get("fingerprint"), ProfileID: q.Get("profile_id"), RunnerType: q.Get("runner_type"), Model: q.Get("model"), TagPrefix: q.Get("tag_prefix"), RunStatus: q.Get("run_status"), ToolName: q.Get("tool_name"), EpisodePattern: q.Get("episode_pattern"), EpisodeCauseScope: q.Get("episode_cause_scope"), EpisodeFingerprint: q.Get("episode_fingerprint"), SelfReportRuleID: q.Get("self_report_rule_id"), SelfReportCauseScope: q.Get("self_report_cause_scope"), TargetScenario: q.Get("target_scenario"), Operation: q.Get("operation")}
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
	if name := strings.TrimSpace(q.Get("cohort")); name != "" {
		definition, _, err := h.svc.ShowCohort(r.Context(), name, 1)
		if err != nil {
			return filter, err
		}
		if err := json.Unmarshal([]byte(definition.FilterJSON), &filter); err != nil {
			return filter, fmt.Errorf("decode cohort %q: %w", name, err)
		}
	}
	return filter, nil
}

func (h *Handler) AggregateInvocationFacts(w http.ResponseWriter, r *http.Request) {
	filter, err := h.invocationReadModelFilter(r)
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
	filter, err := h.invocationReadModelFilter(r)
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

func (h *Handler) EpisodeCohort(w http.ResponseWriter, r *http.Request) {
	filter, err := h.invocationReadModelFilter(r)
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
	cohort, err := h.svc.EpisodeCohort(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cohort)
}

func (h *Handler) CompareEpisodeCohorts(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Left          invocationreadmodel.Filter `json:"left"`
		Right         invocationreadmodel.Filter `json:"right"`
		ChangeBinding string                     `json:"changeBinding,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeSimpleError(w, r, "body", "left and right filters are required")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeSimpleError(w, r, "limit", "limit must be positive")
			return
		}
		limit = parsed
	}
	comparison, err := h.svc.CompareEpisodeCohorts(r.Context(), request.Left, request.Right, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if request.ChangeBinding != "" {
		comparison.ChangeBinding = request.ChangeBinding
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (h *Handler) EpisodeTrend(w http.ResponseWriter, r *http.Request) {
	filter, err := h.invocationReadModelFilter(r)
	if err != nil {
		writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
		return
	}
	bucket := 24 * time.Hour
	if raw := r.URL.Query().Get("bucket"); raw != "" {
		parsed, parseErr := time.ParseDuration(raw)
		if parseErr != nil || parsed <= 0 {
			writeSimpleError(w, r, "bucket", "bucket must be a positive duration")
			return
		}
		bucket = parsed
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 {
			writeSimpleError(w, r, "limit", "limit must be positive")
			return
		}
	}
	trend, err := h.svc.EpisodeTrend(r.Context(), filter, bucket, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"classifierVersion": runsignal.EpisodeClassifierVersion, "bucket": bucket.String(), "rows": trend})
}

func (h *Handler) PublishRecurringFriction(w http.ResponseWriter, r *http.Request) {
	filter, err := h.invocationReadModelFilter(r)
	if err != nil {
		writeSimpleError(w, r, "time_window", "from and to must be RFC3339 timestamps")
		return
	}
	cap := 25
	if raw := r.URL.Query().Get("cap"); raw != "" {
		cap, err = strconv.Atoi(raw)
		if err != nil || cap < 1 {
			writeSimpleError(w, r, "cap", "cap must be positive")
			return
		}
	}
	result, err := h.svc.PublishRecurringFriction(r.Context(), filter, cap)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) InvocationMetrics(w http.ResponseWriter, r *http.Request) {
	filter, err := h.invocationReadModelFilter(r)
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

func (h *Handler) DefineInvocationCohort(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name          string          `json:"name"`
		Filter        json.RawMessage `json:"filter"`
		ChangeBinding string          `json:"changeBinding,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Name == "" || len(request.Filter) == 0 {
		writeSimpleError(w, r, "body", "name and filter are required")
		return
	}
	definition, err := h.svc.DefineCohort(r.Context(), request.Name, string(request.Filter), request.ChangeBinding)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, definition)
}

func (h *Handler) ListInvocationCohorts(w http.ResponseWriter, r *http.Request) {
	definitions, err := h.svc.ListCohorts(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, definitions)
}

func (h *Handler) ShowInvocationCohort(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeSimpleError(w, r, "limit", "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	definition, cohort, err := h.svc.ShowCohort(r.Context(), mux.Vars(r)["name"], limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"definition": definition, "cohort": cohort})
}

func (h *Handler) DeleteInvocationCohort(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCohort(r.Context(), mux.Vars(r)["name"]); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
