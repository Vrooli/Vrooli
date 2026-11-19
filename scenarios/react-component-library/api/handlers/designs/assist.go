package designs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/santhosh-tekuri/jsonschema/v5"
	aiinference "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/encoding/protojson"

	"react-component-library/internal/designassist"
	"react-component-library/internal/designinference"
	domain "react-component-library/internal/designs"
)

// maxAssistRequestBytes bounds the prompt envelope. The design itself is read
// from storage; the client never uploads a second copy.
const maxAssistRequestBytes = 64 << 10

// Assistant is the durable inference seam. `designinference.Service` satisfies
// it; tests substitute a fake so no gateway is needed.
type Assistant interface {
	Run(context.Context, string, designinference.Request) (designinference.Operation, error)
}

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,199}$`)

type assistRequest struct {
	Prompt         string             `json:"prompt"`
	Scope          designassist.Scope `json:"scope"`
	Revision       *int               `json:"revision"`
	IdempotencyKey string             `json:"idempotencyKey"`
}

type assistOperation struct {
	ID           string `json:"id"`
	State        string `json:"state"`
	Provider     string `json:"provider,omitempty"`
	Model        string `json:"model,omitempty"`
	Detail       string `json:"detail,omitempty"`
	InputTokens  int64  `json:"inputTokens,omitempty"`
	OutputTokens int64  `json:"outputTokens,omitempty"`
}

type assistResponse struct {
	Operation assistOperation        `json:"operation"`
	Revision  int                    `json:"revision"`
	Proposal  *designassist.Proposal `json:"proposal,omitempty"`
}

// assist proposes a patch. It never writes the design: the proposal is
// validated, returned, and applied (or discarded) by the studio as one
// undoable command.
func (h *Handler) assist(w http.ResponseWriter, r *http.Request) {
	if h.assistant == nil {
		writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "Design assistance is not configured.")
		return
	}
	var body assistRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAssistRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "Request body must be a JSON assistance request.")
		return
	}
	if body.Revision == nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", "revision is required so a proposal cannot be made against a stale document.")
		return
	}
	if !idempotencyPattern.MatchString(body.IdempotencyKey) {
		writeError(w, http.StatusBadRequest, "invalid_argument", "idempotencyKey must be 8 to 200 letters, digits, dots, colons, underscores or dashes.")
		return
	}

	id := mux.Vars(r)["id"]
	doc, err := h.repository.Get(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "No design with that id.")
		return
	}
	if err != nil {
		h.internalError(w, "Load design for assistance", err)
		return
	}
	if doc.Revision != *body.Revision {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  map[string]string{"code": "conflict", "message": "design was modified elsewhere"},
			"design": doc,
		})
		return
	}

	view, err := designassist.ParseDocument(doc.Frames, doc.Theme)
	if err != nil {
		h.internalError(w, "Read design for assistance", err)
		return
	}
	view.Name = doc.Name
	inferenceRequest, allowed, err := designassist.Prepare(view, body.Scope, body.Prompt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_argument", messageOf(err))
		return
	}
	if doc.Scenario != nil && doc.Page != nil && strings.TrimSpace(*doc.Scenario) != "" && strings.TrimSpace(*doc.Page) != "" {
		inferenceRequest.Scope = &designinference.Scope{Scenario: *doc.Scenario, Page: *doc.Page}
	}

	operation, err := h.assistant.Run(r.Context(), body.IdempotencyKey, inferenceRequest)
	if err != nil {
		if errors.Is(err, designinference.ErrConflict) {
			writeError(w, http.StatusConflict, "conflict", "That idempotency key is already bound to a different request.")
			return
		}
		h.logger.Printf("design assist dispatch failed: %v", err)
		writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "The AI gateway could not be reached. Try again when it is running.")
		return
	}
	result := assistOperation{ID: operation.ID, State: operation.State, Detail: operation.Detail}
	if operation.ResponseJSON != "" {
		var response aiinference.RunResponse
		if err := protojson.Unmarshal([]byte(operation.ResponseJSON), &response); err != nil {
			h.internalError(w, "Read inference response", err)
			return
		}
		result.Provider, result.Model = response.Provider, response.Model
		if response.Usage != nil {
			result.InputTokens, result.OutputTokens = response.Usage.InputTokens, response.Usage.OutputTokens
		}
		if response.Error != nil && response.Error.Message != "" {
			result.Detail = response.Error.Message
		}
		if operation.State == "completed" {
			proposal, err := decodeProposal(inferenceRequest.Schema, response.ValueJson)
			if err != nil {
				writeError(w, http.StatusUnprocessableEntity, "invalid_proposal", fmt.Sprintf("The model returned an unusable patch: %.300s", err.Error()))
				return
			}
			if err := designassist.Validate(view, allowed, proposal); err != nil {
				writeError(w, http.StatusUnprocessableEntity, "invalid_proposal", fmt.Sprintf("The model returned a patch this design cannot apply: %.300s", messageOf(err)))
				return
			}
			writeJSON(w, http.StatusOK, assistResponse{Operation: result, Revision: doc.Revision, Proposal: &proposal})
			return
		}
	}

	switch operation.State {
	case "dispatch_unknown":
		writeError(w, http.StatusServiceUnavailable, "ai_unavailable", "The AI gateway did not answer. Nothing was changed.")
	case "failed":
		detail := strings.TrimSpace(operation.Detail)
		if detail == "" {
			detail = "The AI gateway refused the request."
		}
		writeError(w, http.StatusBadGateway, "ai_failed", "The AI gateway could not produce a patch: "+strings.TrimSpace(truncate(detail, 400)))
	default:
		writeError(w, http.StatusBadGateway, "ai_failed", "The AI gateway returned no patch.")
	}
}

// decodeProposal re-checks the JSON the gateway claimed to validate and then
// decodes it strictly. Two independent checks are cheap; a lenient decode is
// how an unknown field silently becomes an operation nobody validated.
func decodeProposal(schemaJSON, valueJSON string) (designassist.Proposal, error) {
	if strings.TrimSpace(valueJSON) == "" {
		return designassist.Proposal{}, errors.New("the model returned no value")
	}
	var value any
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return designassist.Proposal{}, fmt.Errorf("value is not JSON: %w", err)
	}
	schema, err := jsonschema.CompileString("design-assist.json", schemaJSON)
	if err != nil {
		return designassist.Proposal{}, fmt.Errorf("patch schema is invalid: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return designassist.Proposal{}, err
	}
	var proposal designassist.Proposal
	decoder := json.NewDecoder(strings.NewReader(valueJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proposal); err != nil {
		return designassist.Proposal{}, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return designassist.Proposal{}, errors.New("value has trailing JSON")
	}
	return proposal, nil
}

func messageOf(err error) string {
	if errors.Is(err, designassist.ErrInvalid) {
		return strings.TrimSpace(strings.TrimPrefix(err.Error(), designassist.ErrInvalid.Error()+":"))
	}
	return err.Error()
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
