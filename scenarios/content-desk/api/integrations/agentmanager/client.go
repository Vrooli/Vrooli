// Package agentmanager is Content Desk's narrow, output-only Agent Manager seam.
package agentmanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const profileKey = "content-desk/workbench"

type (
	Commission struct{ DraftID, Action, Body string }
	Receipt    struct{ TaskID, RunID, Status string }
	Result     struct{ Status, Body string }
	Runner     interface {
		Commission(context.Context, Commission) (Receipt, error)
		GetResult(context.Context, string) (Result, error)
	}
)

func (c *Client) GetResult(ctx context.Context, runID string) (Result, error) {
	if strings.TrimSpace(runID) == "" {
		return Result{}, fmt.Errorf("agent run id is required")
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return Result{}, fmt.Errorf("agent manager integration is not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, "agent-manager")
	if err != nil {
		return Result{}, fmt.Errorf("resolve agent manager: %w", err)
	}
	run, err := apiconnect.NewAgentManagerServiceClient(c.http, strings.TrimRight(baseURL, "/")).GetRun(callCtx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
	if err != nil || run.Msg.Run == nil {
		return Result{}, fmt.Errorf("get agent run: %w", err)
	}
	result := Result{Status: run.Msg.Run.Status.String()}
	if run.Msg.Run.Result != nil {
		result.Body = run.Msg.Run.Result.FinalOutput
	}
	return result, nil
}

type Client struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http *http.Client
}

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Commission(ctx context.Context, request Commission) (Receipt, error) {
	if request.DraftID == "" || request.Body == "" {
		return Receipt{}, fmt.Errorf("agent commission requires draft and body")
	}
	if request.Action != "draft" && request.Action != "evidence-hunt" && request.Action != "review" {
		return Receipt{}, fmt.Errorf("unsupported agent action %q", request.Action)
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return Receipt{}, fmt.Errorf("agent manager integration is not configured")
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, "agent-manager")
	if err != nil {
		return Receipt{}, fmt.Errorf("resolve agent manager: %w", err)
	}
	client := apiconnect.NewAgentManagerServiceClient(c.http, strings.TrimRight(baseURL, "/"))
	reconcile, err := client.ReconcileScenarioProfiles(callCtx, connect.NewRequest(&apipb.ReconcileScenarioProfilesRequest{Scenario: "content-desk"}))
	if err != nil {
		return Receipt{}, fmt.Errorf("reconcile governed profile: %w", err)
	}
	var profileID string
	for _, result := range reconcile.Msg.Results {
		if result.ProfileKey == profileKey {
			profileID = result.ProfileId
			break
		}
	}
	if profileID == "" {
		return Receipt{}, fmt.Errorf("governed profile %q was not reconciled", profileKey)
	}
	task, err := client.CreateTask(callCtx, connect.NewRequest(&apipb.CreateTaskRequest{Task: &domainpb.Task{Title: "Content Desk " + request.Action, Description: prompt(request), ScopePath: "scenarios/content-desk", ProjectRoot: ".", CreatedBy: "content-desk"}}))
	if err != nil || task.Msg.Task == nil {
		return Receipt{}, fmt.Errorf("create agent task: %w", err)
	}
	run, err := client.CreateRun(callCtx, connect.NewRequest(&apipb.CreateRunRequest{TaskId: task.Msg.Task.Id, AgentProfileId: &profileID, Prompt: ptr(prompt(request)), Tag: ptr("content-desk:" + request.DraftID + ":" + request.Action)}))
	if err != nil || run.Msg.Run == nil {
		return Receipt{}, fmt.Errorf("start agent run: %w", err)
	}
	return Receipt{TaskID: task.Msg.Task.Id, RunID: run.Msg.Run.Id, Status: run.Msg.Run.Status.String()}, nil
}
func ptr(s string) *string { return &s }
func prompt(request Commission) string {
	return "You are assisting an operator in Content Desk. Perform only the requested " + request.Action + " work. Do not approve, publish, change files, or use credentials. Return editable editorial text and a concise evidence/provenance note. Draft ID: " + request.DraftID + "\n\nCurrent draft:\n" + request.Body
}
