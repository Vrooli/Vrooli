package skills

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"prompt-manager/store"
	"time"

	"github.com/gorilla/mux"
)

// ExperimentHandlers provides HTTP handlers for experiment operations.
type ExperimentHandlers struct {
	experiments store.ExperimentStore
	variants    store.VariantStore
	skills      store.SkillStore
}

// NewExperimentHandlers creates experiment handlers.
func NewExperimentHandlers(experiments store.ExperimentStore, variants store.VariantStore, skills store.SkillStore) *ExperimentHandlers {
	return &ExperimentHandlers{experiments: experiments, variants: variants, skills: skills}
}

// ListExperiments handles GET /experiments
func (h *ExperimentHandlers) ListExperiments(w http.ResponseWriter, r *http.Request) {
	exps, err := h.experiments.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]ExperimentResponse, 0, len(exps))
	for _, e := range exps {
		resp = append(resp, h.experimentToResponse(r, e, false))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListExperimentsBySkill handles GET /skills/{id}/experiments
func (h *ExperimentHandlers) ListExperimentsBySkill(w http.ResponseWriter, r *http.Request) {
	skillID := mux.Vars(r)["id"]

	exps, err := h.experiments.ListBySkill(r.Context(), skillID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := make([]ExperimentResponse, 0, len(exps))
	for _, e := range exps {
		resp = append(resp, h.experimentToResponse(r, e, false))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetExperiment handles GET /experiments/{eid}
func (h *ExperimentHandlers) GetExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *exp, true))
}

// CreateExperiment handles POST /experiments
func (h *ExperimentHandlers) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	var req CreateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.SkillID == "" || req.Name == "" {
		http.Error(w, "id, skillId, and name are required", http.StatusBadRequest)
		return
	}
	if len(req.Arms) < 2 {
		http.Error(w, "at least 2 arms are required", http.StatusBadRequest)
		return
	}
	if err := validateWeights(req.Arms); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify skill exists
	if _, err := h.skills.Get(r.Context(), req.SkillID); err != nil {
		http.Error(w, "skill not found: "+req.SkillID, http.StatusBadRequest)
		return
	}

	// Verify all non-control variants exist
	for _, arm := range req.Arms {
		if arm.VariantID != store.ControlVariantID {
			if _, err := h.variants.Get(r.Context(), req.SkillID, arm.VariantID); err != nil {
				http.Error(w, "variant not found: "+arm.VariantID, http.StatusBadRequest)
				return
			}
		}
	}

	arms := make([]store.ExperimentArm, len(req.Arms))
	for i, a := range req.Arms {
		arms[i] = store.ExperimentArm{VariantID: a.VariantID, Weight: a.Weight}
	}

	exp := &store.Experiment{
		ID:         req.ID,
		SkillID:    req.SkillID,
		Name:       req.Name,
		Hypothesis: req.Hypothesis,
		Arms:       arms,
	}

	if err := h.experiments.Create(r.Context(), exp); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	created, err := h.experiments.Get(r.Context(), req.ID)
	if err != nil {
		http.Error(w, "created but failed to read back: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *created, false))
}

// UpdateExperiment handles PUT /experiments/{eid}
func (h *ExperimentHandlers) UpdateExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	existing, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if existing.Status != store.ExperimentStatusDraft {
		http.Error(w, "can only update draft experiments", http.StatusConflict)
		return
	}

	var req UpdateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Hypothesis != nil {
		existing.Hypothesis = *req.Hypothesis
	}
	if len(req.Arms) > 0 {
		if len(req.Arms) < 2 {
			http.Error(w, "at least 2 arms are required", http.StatusBadRequest)
			return
		}
		if err := validateWeights(req.Arms); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		arms := make([]store.ExperimentArm, len(req.Arms))
		for i, a := range req.Arms {
			arms[i] = store.ExperimentArm{VariantID: a.VariantID, Weight: a.Weight}
		}
		existing.Arms = arms
	}

	if err := h.experiments.Update(r.Context(), eid, existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, false))
}

// DeleteExperiment handles DELETE /experiments/{eid}
func (h *ExperimentHandlers) DeleteExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	if err := h.experiments.Delete(r.Context(), eid); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StartExperiment handles POST /experiments/{eid}/start
func (h *ExperimentHandlers) StartExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusDraft {
		http.Error(w, "can only start draft experiments", http.StatusConflict)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	exp.Status = store.ExperimentStatusRunning
	exp.StartedAt = &now

	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, false))
}

// ConcludeExperiment handles POST /experiments/{eid}/conclude
func (h *ExperimentHandlers) ConcludeExperiment(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "can only conclude running experiments", http.StatusConflict)
		return
	}

	var req ConcludeExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.WinnerVariantID == "" {
		http.Error(w, "winnerVariantId is required", http.StatusBadRequest)
		return
	}

	// Verify winner is one of the arms
	validWinner := false
	for _, arm := range exp.Arms {
		if arm.VariantID == req.WinnerVariantID {
			validWinner = true
			break
		}
	}
	if !validWinner {
		http.Error(w, "winnerVariantId must be one of the experiment's arms", http.StatusBadRequest)
		return
	}

	// Promote winner content to SKILL.md if winner is not control
	if req.WinnerVariantID != store.ControlVariantID {
		_, winnerContent, err := h.variants.GetWithContent(r.Context(), exp.SkillID, req.WinnerVariantID)
		if err != nil {
			http.Error(w, "failed to read winner variant: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update the skill's content with the winner content
		skill, err := h.skills.Get(r.Context(), exp.SkillID)
		if err != nil {
			http.Error(w, "failed to read skill: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := h.skills.Update(r.Context(), exp.SkillID, skill, &winnerContent); err != nil {
			http.Error(w, "failed to promote winner content: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	exp.Status = store.ExperimentStatusConcluded
	exp.ConcludedAt = &now
	exp.WinnerVariantID = &req.WinnerVariantID
	exp.Notes = req.Notes

	if err := h.experiments.Update(r.Context(), eid, exp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updated, _ := h.experiments.Get(r.Context(), eid)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.experimentToResponse(r, *updated, true))
}

// RecordOutcome handles POST /experiments/{eid}/outcomes
func (h *ExperimentHandlers) RecordOutcome(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	exp, err := h.experiments.Get(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if exp.Status != store.ExperimentStatusRunning {
		http.Error(w, "can only record outcomes for running experiments", http.StatusConflict)
		return
	}

	var req RecordOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.VariantID == "" || req.Source == "" {
		http.Error(w, "variantId and source are required", http.StatusBadRequest)
		return
	}

	outcome := store.ExperimentOutcome{
		VariantID:     req.VariantID,
		Source:        req.Source,
		SchemaVersion: req.SchemaVersion,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
		Data:          req.Data,
	}

	if err := h.experiments.RecordOutcome(r.Context(), eid, outcome); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListOutcomes handles GET /experiments/{eid}/outcomes
func (h *ExperimentHandlers) ListOutcomes(w http.ResponseWriter, r *http.Request) {
	eid := mux.Vars(r)["eid"]

	outcomes, err := h.experiments.ListOutcomes(r.Context(), eid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	resp := make([]ExperimentOutcomeResponse, 0, len(outcomes))
	for _, o := range outcomes {
		resp = append(resp, ExperimentOutcomeResponse{
			VariantID:     o.VariantID,
			Source:        o.Source,
			SchemaVersion: o.SchemaVersion,
			RecordedAt:    o.RecordedAt,
			Data:          o.Data,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// experimentToResponse converts a store experiment to API response.
func (h *ExperimentHandlers) experimentToResponse(r *http.Request, exp store.Experiment, includeCounts bool) ExperimentResponse {
	arms := make([]ExperimentArmResponse, len(exp.Arms))
	for i, a := range exp.Arms {
		name := a.VariantID
		if a.VariantID == store.ControlVariantID {
			name = "control (original)"
		} else if v, err := h.variants.Get(r.Context(), exp.SkillID, a.VariantID); err == nil {
			name = v.Name
		}
		arms[i] = ExperimentArmResponse{
			VariantID:   a.VariantID,
			VariantName: name,
			Weight:      a.Weight,
		}
	}

	resp := ExperimentResponse{
		ID:              exp.ID,
		SkillID:         exp.SkillID,
		Name:            exp.Name,
		Hypothesis:      exp.Hypothesis,
		Status:          exp.Status,
		Arms:            arms,
		StartedAt:       exp.StartedAt,
		ConcludedAt:     exp.ConcludedAt,
		WinnerVariantID: exp.WinnerVariantID,
		Notes:           exp.Notes,
		CreatedAt:       exp.CreatedAt,
		UpdatedAt:       exp.UpdatedAt,
		Revision:        exp.Revision,
	}

	if includeCounts {
		if counts, err := h.experiments.CountOutcomesByVariant(r.Context(), exp.ID); err == nil {
			resp.OutcomeCounts = counts
		}
	}

	return resp
}

// validateWeights checks that arm weights sum to approximately 1.0.
func validateWeights(arms []ExperimentArmInput) error {
	var sum float64
	for _, a := range arms {
		if a.Weight <= 0 || a.Weight > 1 {
			return experimentError("each weight must be in (0, 1], got %f for %s", a.Weight, a.VariantID)
		}
		sum += a.Weight
	}
	if math.Abs(sum-1.0) > 0.01 {
		return experimentError("arm weights must sum to 1.0 (got %f)", sum)
	}
	return nil
}

func experimentError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
