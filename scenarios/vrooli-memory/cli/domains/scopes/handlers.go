package scopes

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes/scopesv1connect"
)

type handlers struct {
	client scopesconnect.ScopesServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: scopesconnect.NewScopesServiceClient(http, base)}
}

type facetInput struct {
	ID                 string `json:"id"`
	Label              string `json:"label"`
	Guidance           string `json:"guidance"`
	RetentionPolicy    string `json:"retention_policy"`
	CompactionEligible bool   `json:"compaction_eligible"`
	ResidentBudget     int32  `json:"resident_budget"`
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*scopesv1.CreateScopeResponse, error) {
	var inputs []facetInput
	if raw := ctx.Flag("facets-json"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
			return nil, fmt.Errorf("facets-json must be a JSON array: %w", err)
		}
	}
	facets := make([]*scopesv1.FacetSpec, 0, len(inputs))
	for _, input := range inputs {
		facets = append(facets, &scopesv1.FacetSpec{Id: input.ID, Label: input.Label, Guidance: input.Guidance, RetentionPolicy: input.RetentionPolicy, CompactionEligible: input.CompactionEligible, ResidentBudget: input.ResidentBudget})
	}
	frontier, err := positiveFlag(ctx.Flag("frontier-target"), 16)
	if err != nil {
		return nil, err
	}
	wake, err := positiveFlag(ctx.Flag("wake-budget"), 96)
	if err != nil {
		return nil, err
	}
	maxLines, err := positiveFlag(ctx.Flag("max-entry-lines"), 2)
	if err != nil {
		return nil, err
	}
	scope := &scopesv1.Scope{Id: ctx.Positional("id"), Label: ctx.Flag("label"), FrontierTarget: int32(frontier), WakeBudget: int32(wake), MaxEntryLines: int32(maxLines), Facets: facets}
	response, err := h.client.CreateScope(context.Background(), connect.NewRequest(&scopesv1.CreateScopeRequest{Scope: scope}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create memory scope", err, nil)
	}
	return response.Msg, nil
}

func positiveFlag(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value <= 0 {
		return 0, fmt.Errorf("scope numeric flags must be positive integers")
	}
	return value, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, msg *scopesv1.CreateScopeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created scope %s with frontier target %d and wake budget %d.", msg.GetScope().GetId(), msg.GetScope().GetFrontierTarget(), msg.GetScope().GetWakeBudget())}}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*scopesv1.ListScopesResponse, error) {
	response, err := h.client.ListScopes(context.Background(), connect.NewRequest(&scopesv1.ListScopesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list memory scopes", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *scopesv1.ListScopesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetScopes()))
	for _, scope := range msg.GetScopes() {
		results = append(results, fmt.Sprintf("%s (%s): frontier=%d wake=%d max-lines=%d", scope.GetId(), scope.GetLabel(), scope.GetFrontierTarget(), scope.GetWakeBudget(), scope.GetMaxEntryLines()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d memory scope(s).", len(results))}, Results: results}
}
