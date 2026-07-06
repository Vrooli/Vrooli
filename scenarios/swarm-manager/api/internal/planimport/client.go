package planimport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// PlanFetcher fetches an authored plan read-only from plan-manager.
type PlanFetcher interface {
	GetPlan(ctx context.Context, id string) (*sharedv1.Plan, error)
}

// ConnectFetcher implements PlanFetcher over the plan-manager PlansService
// Connect API. It re-resolves the base URL per call (captured scenario URLs are
// short-lived) and is strictly read-only — GetPlan only, never a mutation (D3).
type ConnectFetcher struct {
	http *http.Client
}

// NewConnectFetcher builds a fetcher. A nil httpClient gets a 30s-timeout default.
func NewConnectFetcher(httpClient *http.Client) *ConnectFetcher {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &ConnectFetcher{http: httpClient}
}

// GetPlan resolves plan-manager and fetches one plan by id or slug.
func (f *ConnectFetcher) GetPlan(ctx context.Context, id string) (*sharedv1.Plan, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "plan-manager")
	if err != nil {
		return nil, fmt.Errorf("planimport: resolve plan-manager: %w", err)
	}
	client := plansconnect.NewPlansServiceClient(f.http, strings.TrimRight(baseURL, "/"))
	resp, err := client.GetPlan(ctx, connect.NewRequest(&plansv1.GetPlanRequest{Id: id}))
	if err != nil {
		return nil, fmt.Errorf("planimport: GetPlan %q: %w", id, err)
	}
	return resp.Msg.GetPlan(), nil
}
