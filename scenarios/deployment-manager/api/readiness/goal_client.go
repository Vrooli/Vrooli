package readiness

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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

// GoalClosed reports only Swarm Manager's independently reviewed close-out
// state. It never treats goal existence or milestone creation as closure.
func (c *GoalClient) GoalClosed(ctx context.Context, name string) (bool, error) {
	if c == nil || c.ResolveURL == nil || strings.TrimSpace(name) == "" {
		return false, fmt.Errorf("swarm-manager goal client and goal name are required")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return false, fmt.Errorf("resolve swarm-manager: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := apiconnect.NewGoalServiceClient(httpClient, baseURL, connect.WithProtoJSON()).GetGoal(
		ctx, connect.NewRequest(&apipb.GetGoalRequest{Name: canonicalSwarmGoalName(name)}),
	)
	if err != nil {
		return false, fmt.Errorf("read readiness goal: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetGoal() == nil {
		return false, fmt.Errorf("read readiness goal returned an empty response")
	}
	return response.Msg.GetGoal().GetStatus() == "archived", nil
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
	client := apiconnect.NewGoalServiceClient(httpClient, baseURL, connect.WithProtoJSON())
	// Swarm Manager applies its own name sanitizer before persisting goals.
	// Use the same canonical form for lookups and milestone writes while
	// retaining the slash-delimited logical name in deployment-manager's
	// readiness contract.
	canonicalName := canonicalSwarmGoalName(spec.Name)
	existing, err := client.GetGoal(ctx, connect.NewRequest(&apipb.GetGoalRequest{Name: canonicalName}))
	if err == nil {
		if existing == nil || existing.Msg == nil || existing.Msg.GetGoal() == nil {
			return "", false, fmt.Errorf("look up readiness goal returned an empty response")
		}
		canonicalName = existing.Msg.GetGoal().GetName()
		if err := reconcileGoalMilestones(ctx, client, canonicalName, existing.Msg.GetGoal().GetMilestones(), spec.Milestones); err != nil {
			return "", false, err
		}
		return canonicalName, true, nil
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		return "", false, fmt.Errorf("look up readiness goal: %w", err)
	}
	created, err := client.CreateGoal(ctx, connect.NewRequest(&apipb.CreateGoalRequest{
		Name: spec.Name, Title: spec.Title, Description: spec.Description,
		Priority: int32(spec.Priority), ServesDeliverable: spec.ServesDeliverable,
	}))
	if err != nil {
		return "", false, fmt.Errorf("create readiness goal: %w", err)
	}
	if created == nil || created.Msg == nil {
		return "", false, fmt.Errorf("create readiness goal returned an empty response")
	}
	canonicalName = created.Msg.GetGoal().GetName()
	if canonicalName == "" {
		return "", false, fmt.Errorf("create readiness goal returned no canonical name")
	}
	if err := reconcileGoalMilestones(ctx, client, canonicalName, nil, spec.Milestones); err != nil {
		return "", false, err
	}
	return canonicalName, false, nil
}

type milestoneClient interface {
	CreateMilestone(context.Context, *connect.Request[apipb.CreateMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error)
	UpdateMilestone(context.Context, *connect.Request[apipb.UpdateMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error)
	ArchiveMilestone(context.Context, *connect.Request[apipb.ArchiveMilestoneRequest]) (*connect.Response[apipb.GoalResponse], error)
}

func reconcileGoalMilestones(ctx context.Context, client milestoneClient, goalName string, existing []*sharedpb.Milestone, desired []GoalMilestone) error {
	existingByName := make(map[string]*sharedpb.Milestone, len(existing))
	for _, milestone := range existing {
		if milestone != nil && milestone.GetArchivedAt() == "" {
			existingByName[milestone.GetName()] = milestone
		}
	}
	desiredNames := make(map[string]struct{}, len(desired))
	for _, milestone := range desired {
		desiredNames[milestone.Name] = struct{}{}
		message := &sharedpb.Milestone{Name: milestone.Name, Title: milestone.Title, Description: milestone.Description, AcceptanceCriteria: milestone.AcceptanceCriteria}
		if _, ok := existingByName[milestone.Name]; ok {
			if _, err := client.UpdateMilestone(ctx, connect.NewRequest(&apipb.UpdateMilestoneRequest{GoalName: goalName, Milestone: message})); err != nil {
				return fmt.Errorf("update readiness milestone %q: %w", milestone.Name, err)
			}
			continue
		}
		if _, err := client.CreateMilestone(ctx, connect.NewRequest(&apipb.CreateMilestoneRequest{GoalName: goalName, Milestone: message})); err != nil {
			return fmt.Errorf("create readiness milestone %q: %w", milestone.Name, err)
		}
	}
	for name := range existingByName {
		if _, ok := desiredNames[name]; ok {
			continue
		}
		if _, err := client.ArchiveMilestone(ctx, connect.NewRequest(&apipb.ArchiveMilestoneRequest{GoalName: goalName, MilestoneName: name})); err != nil {
			return fmt.Errorf("archive resolved readiness milestone %q: %w", name, err)
		}
	}
	return nil
}

func canonicalSwarmGoalName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, name)
}
