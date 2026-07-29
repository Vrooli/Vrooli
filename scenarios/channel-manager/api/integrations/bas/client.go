// Package bas adapts a durable Channel Manager action to BAS's generated
// workflow API. It sends references only: a session-profile id and action id.
package bas

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	baseecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

const workflowID = "channel-manager-dispatch"

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

func (c *Client) Dispatch(ctx context.Context, profileRef, actionID string) (string, []string, error) {
	if profileRef == "" || actionID == "" {
		return "", nil, fmt.Errorf("BAS dispatch requires profile and action")
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
	if err != nil {
		return "", nil, fmt.Errorf("resolve BAS: %w", err)
	}
	client := basconnect.NewWorkflowsServiceClient(c.http, strings.TrimRight(base, "/"))
	response, err := client.ExecuteWorkflow(ctx, connect.NewRequest(&basapi.ExecuteWorkflowRequest{WorkflowId: workflowID, Parameters: &baseecution.ExecutionParameters{SessionProfileId: &profileRef, SaveSessionProfileId: &profileRef, Variables: map[string]string{"action_id": actionID}}}))
	if err != nil || response == nil || response.Msg == nil || response.Msg.ExecutionId == "" {
		if err == nil {
			err = fmt.Errorf("BAS returned no execution id")
		}
		return "", nil, err
	}
	return response.Msg.ExecutionId, nil, nil
}
