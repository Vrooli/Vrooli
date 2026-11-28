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

type RouteKind string

const (
	RouteKindLocal        RouteKind = "local"
	RouteKindOwnKey       RouteKind = "own_key"
	RouteKindVrooliCredit RouteKind = "vrooli_credits"
	RouteKindHosted       RouteKind = "hosted"
)

type LimitKind string

const (
	LimitKindNoMonetarySpend    LimitKind = "no_monetary_spend"
	LimitKindSelectableCredits  LimitKind = "selectable_credits_cap"
	LimitKindNotReported        LimitKind = "not_reported"
	LimitKindOwnerEnforcedValue LimitKind = "owner_enforced_value"
)

type ApplicableLimit struct {
	Kind         LimitKind
	MaxValue     int64
	Unit         string
	Provenance   string
	Options      []int64
	DefaultValue int64
}

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
	// ContextWindowSource identifies the verified harness reader behind the window.
	ContextWindowSource string
	ContextLimitTokens  []int64
	Metered             bool
	DisplayName         string
	PolicyCurrent       bool
	RouteKind           RouteKind
	ApplicableLimit     ApplicableLimit
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
	CommandLiteral               bool
	SourceConfigurationSupported bool
}

type Service interface {
	GetRoleModelChoices(ctx context.Context, agentID, targetID, launchCommand string) Result
}

type accountService interface {
	GetAccountRoleModelChoices(context.Context, string, string, string, string) Result
}

type handler struct{ service Service }

func (h *handler) GetRoleModelChoices(ctx context.Context, req *connect.Request[modelselectionv1.GetRoleModelChoicesRequest]) (*connect.Response[modelselectionv1.GetRoleModelChoicesResponse], error) {
	result := h.service.GetRoleModelChoices(ctx, req.Msg.GetAgentId(), req.Msg.GetTargetId(), req.Msg.GetLaunchCommand())
	if service, ok := h.service.(accountService); ok && req.Msg.GetBusinessAccountId() != "" {
		result = service.GetAccountRoleModelChoices(ctx, req.Msg.GetAgentId(), req.Msg.GetTargetId(), req.Msg.GetLaunchCommand(), req.Msg.GetBusinessAccountId())
	}
	response := &modelselectionv1.GetRoleModelChoicesResponse{
		PolicyStatus:                 result.PolicyStatus,
		Availability:                 result.Availability,
		Runner:                       result.Runner,
		Role:                         result.Role,
		Provenance:                   result.Provenance,
		ObservedAt:                   result.ObservedAt,
		PolicyDigest:                 result.PolicyDigest,
		ModelFlag:                    result.ModelFlag,
		Detail:                       result.Detail,
		CatalogStale:                 result.CatalogStale,
		OperatorCommand:              result.OperatorCommand,
		CommandLiteral:               result.CommandLiteral,
		SourceConfigurationSupported: result.SourceConfigurationSupported,
	}
	for _, choice := range result.Choices {
		item := &modelselectionv1.RoleModelChoice{Model: choice.Model, Source: choice.Source, Provider: choice.Provider, LaunchModel: choice.LaunchModel, Effort: choice.Effort, SourceLabel: choice.SourceLabel, IsDefault: choice.IsDefault, ContextWindowTokens: choice.ContextWindowTokens, ContextWindowSource: choice.ContextWindowSource, ContextLimitTokens: choice.ContextLimitTokens, Metered: choice.Metered, DisplayName: choice.DisplayName, PolicyCurrent: choice.PolicyCurrent, RouteKind: routeKindToProto(choice.RouteKind), ApplicableLimit: &modelselectionv1.ApplicableLimit{Kind: limitKindToProto(choice.ApplicableLimit.Kind), MaxValue: choice.ApplicableLimit.MaxValue, Unit: choice.ApplicableLimit.Unit, Provenance: choice.ApplicableLimit.Provenance, Options: choice.ApplicableLimit.Options, DefaultValue: choice.ApplicableLimit.DefaultValue}}
		response.Choices = append(response.Choices, item)
	}
	return connect.NewResponse(response), nil
}

func routeKindToProto(kind RouteKind) modelselectionv1.RouteKind {
	switch kind {
	case RouteKindLocal:
		return modelselectionv1.RouteKind_ROUTE_KIND_LOCAL
	case RouteKindOwnKey:
		return modelselectionv1.RouteKind_ROUTE_KIND_OWN_KEY
	case RouteKindVrooliCredit:
		return modelselectionv1.RouteKind_ROUTE_KIND_VROOLI_CREDITS
	case RouteKindHosted:
		return modelselectionv1.RouteKind_ROUTE_KIND_HOSTED
	default:
		return modelselectionv1.RouteKind_ROUTE_KIND_UNSPECIFIED
	}
}

func limitKindToProto(kind LimitKind) modelselectionv1.LimitKind {
	switch kind {
	case LimitKindNoMonetarySpend:
		return modelselectionv1.LimitKind_LIMIT_KIND_NO_MONETARY_SPEND
	case LimitKindSelectableCredits:
		return modelselectionv1.LimitKind_LIMIT_KIND_SELECTABLE_CREDITS_CAP
	case LimitKindNotReported:
		return modelselectionv1.LimitKind_LIMIT_KIND_NOT_REPORTED
	case LimitKindOwnerEnforcedValue:
		return modelselectionv1.LimitKind_LIMIT_KIND_OWNER_ENFORCED_VALUE
	default:
		return modelselectionv1.LimitKind_LIMIT_KIND_UNSPECIFIED
	}
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
