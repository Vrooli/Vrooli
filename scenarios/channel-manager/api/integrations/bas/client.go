// Package bas adapts a durable Channel Manager action to BAS's generated
// workflow API. It sends references only: a session-profile id and action id.
package bas

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	core "channel-manager/internal/channelmanager"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	baseecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

type resolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}
type Client struct {
	resolver resolver
	http     *http.Client
}

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Dispatch(ctx context.Context, profileRef, workflowRef, actionID string) (string, []string, error) {
	if profileRef == "" || workflowRef == "" || actionID == "" {
		return "", nil, fmt.Errorf("BAS dispatch requires profile, workflow, and action")
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	if err != nil {
		return "", nil, fmt.Errorf("resolve BAS: %w", err)
	}
	client := basconnect.NewWorkflowsServiceClient(c.http, strings.TrimRight(base, "/"))
	response, err := client.ExecuteWorkflow(ctx, connect.NewRequest(&basapi.ExecuteWorkflowRequest{WorkflowId: workflowRef, Parameters: &baseecution.ExecutionParameters{SessionProfileId: &profileRef, SaveSessionProfileId: &profileRef, Variables: map[string]string{"action_id": actionID}}}))
	if err != nil || response == nil || response.Msg == nil || response.Msg.ExecutionId == "" {
		if err == nil {
			err = fmt.Errorf("BAS returned no execution id")
		}
		return "", nil, err
	}
	return response.Msg.ExecutionId, nil, nil
}

// Inspect reads BAS's generic execution and evidence-manifest contracts. It
// returns stable identifiers only; protected artifact bytes and session state
// remain within BAS.
func (c *Client) Inspect(ctx context.Context, executionID string) (core.BrowserExecution, error) {
	if strings.TrimSpace(executionID) == "" {
		return core.BrowserExecution{}, fmt.Errorf("BAS inspection requires an execution id")
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	if err != nil {
		return core.BrowserExecution{}, fmt.Errorf("resolve BAS: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := basconnect.NewExecutionsServiceClient(c.http, strings.TrimRight(base, "/"))
	execution, err := client.GetExecution(ctx, connect.NewRequest(&basapi.GetExecutionRequest{ExecutionId: executionID}))
	if err != nil || execution == nil || execution.Msg == nil || execution.Msg.Execution == nil {
		if err == nil {
			err = fmt.Errorf("BAS returned no execution")
		}
		return core.BrowserExecution{}, err
	}
	result := core.BrowserExecution{ExecutionID: execution.Msg.Execution.GetExecutionId(), Status: strings.ToLower(execution.Msg.Execution.GetStatus().String()), Failure: execution.Msg.Execution.GetError()}
	pack, packErr := client.GetExecutionReplayPackage(ctx, connect.NewRequest(&basapi.GetExecutionArtifactsRequest{ExecutionId: executionID}))
	if packErr != nil || pack == nil || pack.Msg == nil || pack.Msg.Evidence == nil {
		return result, nil // status remains useful while BAS finalizes artifacts.
	}
	for _, artifact := range pack.Msg.Evidence.Artifacts {
		if artifact != nil && artifact.Id != "" {
			result.ArtifactRefs = append(result.ArtifactRefs, artifact.Id)
		}
	}
	return result, nil
}
