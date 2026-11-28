package experimentarms

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"

	aisearch "github.com/vrooli/ai-go/search"
	registry "search-hub/internal/experimentarms"
)

const maxRequestBytes = 1 << 20

const (
	writeCapability = "search-hub:write"
	writeOperation  = "embedding_experiment.manage"
)

type handler struct {
	registry *registry.Registry
	stage    *registry.FleetGenerationStage
}

type createRequest struct {
	ExperimentID string                        `json:"experiment_id"`
	ArmID        string                        `json:"arm_id"`
	ProviderID   string                        `json:"provider_id"`
	CallerID     string                        `json:"caller_id"`
	BuildID      string                        `json:"build_id"`
	Recipe       aisearch.EmbeddingRecipeInput `json:"recipe"`
	Evidence     []registry.EvidenceRef        `json:"evidence"`
}

type transitionRequest struct {
	ExpectedState   registry.State       `json:"expected_state"`
	ExpectedVersion int64                `json:"expected_version"`
	To              registry.State       `json:"to"`
	IdempotencyKey  string               `json:"idempotency_key"`
	Receipt         registry.EvidenceRef `json:"receipt,omitempty"`
}

func (h handler) create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !authorizeMutation(w, r, request.ArmID, false) {
		return
	}
	recipe, err := aisearch.NewEmbeddingRecipe(request.Recipe)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	arm, err := h.registry.Create(r.Context(), registry.ArmCreate{ExperimentID: request.ExperimentID, ArmID: request.ArmID, ProviderID: request.ProviderID, CallerID: request.CallerID, BuildID: request.BuildID, Recipe: recipe, Evidence: request.Evidence})
	if err != nil {
		if errors.Is(err, registry.ErrImmutableConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, arm)
}

func (h handler) get(w http.ResponseWriter, r *http.Request) {
	arm, err := h.registry.Get(r.Context(), strings.TrimSpace(mux.Vars(r)["armID"]))
	if err != nil {
		if errors.Is(err, registry.ErrArmNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, arm)
}

func (h handler) transition(w http.ResponseWriter, r *http.Request) {
	var body transitionRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	armID := strings.TrimSpace(mux.Vars(r)["armID"])
	if !authorizeMutation(w, r, armID, requiresIndependentHuman(body.To)) {
		return
	}
	arm, err := h.registry.Transition(r.Context(), registry.TransitionRequest{ArmID: armID, ExpectedState: body.ExpectedState, ExpectedVersion: body.ExpectedVersion, To: body.To, IdempotencyKey: body.IdempotencyKey, Receipt: body.Receipt})
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrArmNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, registry.ErrVersionConflict), errors.Is(err, registry.ErrIdempotencyConflict), errors.Is(err, registry.ErrImmutableConflict):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, registry.ErrIllegalTransition):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, arm)
}

func (h handler) stageGeneration(w http.ResponseWriter, r *http.Request) {
	armID := strings.TrimSpace(mux.Vars(r)["armID"])
	if !authorizeMutation(w, r, armID, false) {
		return
	}
	if h.stage == nil {
		writeError(w, http.StatusServiceUnavailable, registry.ErrFleetGenerationUnavailable.Error())
		return
	}
	generation, err := h.stage.Stage(r.Context(), armID)
	if err != nil {
		switch {
		case errors.Is(err, registry.ErrArmNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, registry.ErrVersionConflict), errors.Is(err, registry.ErrImmutableConflict), errors.Is(err, registry.ErrIllegalTransition), errors.Is(err, registry.ErrFleetStageRequiresAllocatedArm):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, generation)
}

// authorizeMutation binds all mutable arm operations to a verified principal,
// the Search Hub write capability, the exact arm target, and the owner-approved
// operation grant. Promotion, rollback, and deletion decisions remain human-only.
func authorizeMutation(w http.ResponseWriter, r *http.Request, armID string, humanDecision bool) bool {
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() {
		writeError(w, http.StatusUnauthorized, "verified Search Hub identity required")
		return false
	}
	target := "search-hub:embedding-arm/" + strings.TrimSpace(armID)
	if _, err := authn.RequireOperation(r.Context(), writeCapability, writeOperation, target); err != nil {
		writeAuthorizationError(w, err)
		return false
	}
	if humanDecision {
		if _, err := authn.RequireIndependentHuman(r.Context()); err != nil {
			writeError(w, http.StatusForbidden, "this lifecycle decision requires an independent verified human")
			return false
		}
	}
	return true
}

func requiresIndependentHuman(to registry.State) bool {
	switch to {
	case registry.StatePromoting, registry.StateRollingBack, registry.StateDeleted:
		return true
	default:
		return false
	}
}

func writeAuthorizationError(w http.ResponseWriter, err error) {
	if failure, ok := identity.FailureFromError(err); ok {
		status := http.StatusUnauthorized
		if failure.Class == identity.FailureUnavailable {
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, err.Error())
		return
	}
	writeError(w, http.StatusForbidden, err.Error())
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
