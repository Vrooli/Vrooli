// Package modelselection exposes read-only, resource-owned coding role choices.
package modelselection

import (
	"context"
	"log"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	modelselectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/modelselection"
	modelselectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/modelselection/modelselection_v1connect"
	"web-console/internal/module"
)

type Choice struct {
	Model       string
	Source      string
	Provider    string
	LaunchModel string
	Effort      string
	SourceLabel string
	IsDefault   bool
	// ContextWindowTokens is zero when the window is unknown, never unlimited.
	ContextWindowTokens int64
	ContextLimitTokens  []int64
	Metered             bool
	DisplayName         string
}

type Result struct {
	PolicyStatus string
	Availability string
	Runner       string
	Role         string
	Provenance   string
	ObservedAt   string
	PolicyDigest string
	ModelFlag    string
	Choices      []Choice
	Detail       string
	CatalogStale bool
	// OperatorCommand is how to refresh a list the action cannot.
	OperatorCommand string
	// CommandLiteral says the launch command runs as written.
	CommandLiteral bool
}

type Service interface {
	GetRoleModelChoices(ctx context.Context, agentID, targetID, launchCommand string) Result
}

type handler struct{ service Service }

func (h *handler) GetRoleModelChoices(ctx context.Context, req *connect.Request[modelselectionv1.GetRoleModelChoicesRequest]) (*connect.Response[modelselectionv1.GetRoleModelChoicesResponse], error) {
	result := h.service.GetRoleModelChoices(ctx, req.Msg.GetAgentId(), req.Msg.GetTargetId(), req.Msg.GetLaunchCommand())
	response := &modelselectionv1.GetRoleModelChoicesResponse{
		PolicyStatus:    result.PolicyStatus,
		Availability:    result.Availability,
		Runner:          result.Runner,
		Role:            result.Role,
		Provenance:      result.Provenance,
		ObservedAt:      result.ObservedAt,
		PolicyDigest:    result.PolicyDigest,
		ModelFlag:       result.ModelFlag,
		Detail:          result.Detail,
		CatalogStale:    result.CatalogStale,
		OperatorCommand: result.OperatorCommand,
		CommandLiteral:  result.CommandLiteral,
	}
	for _, choice := range result.Choices {
		item := &modelselectionv1.RoleModelChoice{Model: choice.Model, Source: choice.Source, Provider: choice.Provider, LaunchModel: choice.LaunchModel, Effort: choice.Effort, SourceLabel: choice.SourceLabel, IsDefault: choice.IsDefault, ContextWindowTokens: choice.ContextWindowTokens, ContextLimitTokens: choice.ContextLimitTokens, Metered: choice.Metered, DisplayName: choice.DisplayName}
		response.Choices = append(response.Choices, item)
	}
	return connect.NewResponse(response), nil
}

func Module(service Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, endpoint := modelselectionconnect.NewModelSelectionServiceHandler(&handler{service: service})
	return module.Module{
		Name: "modelselection",
		Mount: func(router *mux.Router) {
			connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: endpoint})
		},
		Endpoints: Endpoints,
	}
}
