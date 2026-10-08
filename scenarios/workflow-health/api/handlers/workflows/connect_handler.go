package workflows

import (
	"context"
	"fmt"
	"log"
	"strings"

	"connectrpc.com/connect"

	internalsearch "workflow-health/internal/search"
	internalvalidation "workflow-health/internal/validation"

	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/workflow-health/v1/workflows"
)

type Deps struct {
	Logger *log.Logger
	Engine *internalvalidation.Engine
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Engine == nil {
		d.Engine = internalvalidation.NewEngine()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) SearchWorkflows(ctx context.Context, req *connect.Request[workflowsv1.SearchWorkflowsRequest]) (*connect.Response[workflowsv1.SearchWorkflowsResponse], error) {
	msg := req.Msg
	if strings.TrimSpace(msg.GetScenario()) == "" && strings.TrimSpace(msg.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario or path is required"))
	}
	report, err := h.deps.Engine.ValidateScenario(ctx, msg.GetScenario(), msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	results := internalsearch.SearchCatalog(report.Catalog, internalsearch.Options{
		Query:            msg.GetQuery(),
		Types:            msg.GetTypes(),
		IncludeFragments: msg.GetIncludeFragments(),
		Limit:            int(msg.GetLimit()),
	})
	resp := &workflowsv1.SearchWorkflowsResponse{
		Scenario: report.Scenario,
		Query:    msg.GetQuery(),
		Total:    int32(len(results)),
		Results:  make([]*workflowsv1.WorkflowSearchResult, 0, len(results)),
	}
	for _, result := range results {
		resp.Results = append(resp.Results, toProtoResult(result))
	}
	return connect.NewResponse(resp), nil
}

func toProtoResult(result internalsearch.Result) *workflowsv1.WorkflowSearchResult {
	return &workflowsv1.WorkflowSearchResult{
		Id:                   result.ID,
		LeafType:             result.LeafType,
		AssetType:            string(result.Asset.Type),
		Role:                 string(result.Asset.Role),
		Title:                result.Title,
		Snippet:              result.Snippet,
		Path:                 result.Asset.Path,
		Score:                result.Score,
		Runnable:             result.Runnable,
		Mutating:             result.Mutating,
		RequiresConfirmation: result.RequiresConfirmation,
		RequiresIsolation:    result.RequiresIsolation,
		SafetySummary:        result.SafetySummary,
		RequirementIds:       result.RequirementIDs,
		Selectors:            result.SelectorRefs,
		Routes:               result.RouteRefs,
		Labels:               result.LabelPairs,
		DependencyPaths:      result.DependencyPaths,
		Guardrails:           result.Guardrails,
	}
}
