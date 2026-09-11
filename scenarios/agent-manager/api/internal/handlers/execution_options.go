package handlers

import (
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/protoconv"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

func (h *Handler) ListExecutionOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.svc.ListExecutionOptions(r.Context(), strings.TrimSpace(r.URL.Query().Get("role")))
	if err != nil {
		writeError(w, r, err)
		return
	}
	response := executionOptionsToProto(options)
	writeProtoJSON(w, http.StatusOK, response)
}

func executionOptionsToProto(options []orchestration.ExecutionOption) *apipb.ListExecutionOptionsResponse {
	response := &apipb.ListExecutionOptionsResponse{}
	for _, option := range options {
		item := &apipb.ExecutionOption{RunnerType: protoconv.RunnerTypeToProto(domain.RunnerType(option.RunnerType)), Available: option.Available, Message: option.Message, NativeObjective: option.NativeObjective, SandboxModesWithNativeObjective: append([]string(nil), option.SandboxModesWithNativeObjective...), DefaultModel: option.DefaultModel, EffortLevels: append([]string(nil), option.EffortLevels...)}
		for _, model := range option.Models {
			item.Models = append(item.Models, &apipb.ModelOption{Id: model.ID, CanonicalModel: model.CanonicalModel, IsDefault: model.IsDefault})
		}
		response.Options = append(response.Options, item)
	}
	return response
}

var _ orchestration.ExecutionOptionsService = (*orchestration.Orchestrator)(nil)
