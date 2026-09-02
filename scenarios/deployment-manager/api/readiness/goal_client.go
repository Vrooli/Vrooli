package readiness

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/shared"
)

type GoalClient struct {
	ResolveURL func(context.Context) (string, error)
	HTTPClient *http.Client
}

func NewGoalClient() *GoalClient {
	return &GoalClient{
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURL(ctx, "swarm-manager", "API_PORT")
		},
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Open creates the deterministic readiness goal, or returns the existing goal
// when the same scenario/commit has already been opened. Milestones are added
// only on creation, preventing duplicate work on repeated release preparation.
func (c *GoalClient) Open(ctx context.Context, spec GoalSpec) (string, bool, error) {
	if c == nil || c.ResolveURL == nil {
		return "", false, fmt.Errorf("swarm-manager goal client is not configured")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return "", false, fmt.Errorf("resolve swarm-manager: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	client := apiconnect.NewGoalServiceClient(httpClient, baseURL)
	_, err = client.GetGoal(ctx, connect.NewRequest(&apipb.GetGoalRequest{Name: spec.Name}))
	if err == nil {
		return spec.Name, true, nil
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		return "", false, fmt.Errorf("look up readiness goal: %w", err)
	}
	_, err = client.CreateGoal(ctx, connect.NewRequest(&apipb.CreateGoalRequest{
		Name: spec.Name, Title: spec.Title, Description: spec.Description,
		Priority: int32(spec.Priority), ServesDeliverable: spec.ServesDeliverable,
	}))
	if err != nil {
		return "", false, fmt.Errorf("create readiness goal: %w", err)
	}
	for _, milestone := range spec.Milestones {
		_, err := client.CreateMilestone(ctx, connect.NewRequest(&apipb.CreateMilestoneRequest{
			GoalName:  spec.Name,
			Milestone: &sharedpb.Milestone{Name: milestone.Name, Title: milestone.Title, AcceptanceCriteria: milestone.AcceptanceCriteria},
		}))
		if err != nil {
			return "", false, fmt.Errorf("create readiness milestone %q: %w", milestone.Name, err)
		}
	}
	return spec.Name, false, nil
}
