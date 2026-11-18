package integrations

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// CheckAgentStatus gets the status of a coding agent run as a proto Run.
// This is used by the reconciliation service to verify agent state after server restarts.
// On connection failure, re-resolves the agent-manager URL and retries once.
func (c *AgentManagerClient) CheckAgentStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	result, err := callAgentManager(ctx, c, true, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.GetRunResponse, error) {
		resp, err := client.GetRun(callCtx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	return result.GetRun(), nil
}

// GetEvents retrieves events for a run, optionally filtered by sequence.
func (c *AgentManagerClient) GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error) {
	after := afterSequence
	result, err := callAgentManager(ctx, c, true, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.GetRunEventsResponse, error) {
		resp, err := client.GetRunEvents(callCtx, connect.NewRequest(&apipb.GetRunEventsRequest{RunId: runID, AfterSequence: &after}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}

	// Translate proto events to inbox format
	events := make([]*TranslatedEvent, 0, len(result.GetEvents()))
	for _, protoEvent := range result.GetEvents() {
		event := TranslateProtoEvent(protoEvent)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// StopRun stops a running agent run.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) error {
	_, err := callAgentManager(ctx, c, false, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.StopRunResponse, error) {
		resp, err := client.StopRun(callCtx, connect.NewRequest(&apipb.StopRunRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return fmt.Errorf("stop run: %w", err)
	}
	return nil
}

// GetRunStatus gets the current status of a run.
func (c *AgentManagerClient) GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error) {
	result, err := callAgentManager(ctx, c, true, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.GetRunResponse, error) {
		resp, err := client.GetRun(callCtx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("get run status: %w", err)
	}

	run := result.GetRun()
	if run == nil {
		return nil, fmt.Errorf("run response missing run")
	}

	return &AgentRunStatus{
		RunID:           runID,
		Status:          ProtoRunStatusToLocal(run.GetStatus()),
		Phase:           protoRunPhaseToString(run.GetPhase()),
		ProgressPercent: int(run.GetProgressPercent()),
		SessionID:       run.GetSessionId(),
		ErrorMsg:        run.GetErrorMsg(),
	}, nil
}

// ListRuns retrieves a paginated list of runs from agent-manager.
func (c *AgentManagerClient) ListRuns(ctx context.Context, opts ListRunsOptions) (*ListRunsResult, error) {
	request := &apipb.ListRunsRequest{}
	if opts.TagPrefix != "" {
		request.TagPrefix = &opts.TagPrefix
	}
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		request.Limit = &limit
	}
	if opts.Offset > 0 {
		offset := int32(opts.Offset)
		request.Offset = &offset
	}
	if opts.Status != "" {
		status := domainpb.RunStatus_value["RUN_STATUS_"+strings.ToUpper(opts.Status)]
		if status == 0 {
			return nil, fmt.Errorf("unknown run status %q", opts.Status)
		}
		value := domainpb.RunStatus(status)
		request.Status = &value
	}
	result, err := callAgentManager(ctx, c, true, func(callCtx context.Context, client apiconnect.AgentManagerServiceClient) (*apipb.ListRunsResponse, error) {
		resp, err := client.ListRuns(callCtx, connect.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	runs := make([]AgentRunSummary, 0, len(result.GetRuns()))
	for _, run := range result.GetRuns() {
		runs = append(runs, protoRunToSummary(run))
	}

	return &ListRunsResult{
		Runs:    runs,
		Total:   int(result.GetTotal()),
		HasMore: result.GetHasMore(),
	}, nil
}

// protoRunToSummary converts a proto Run to an AgentRunSummary.
func protoRunToSummary(run *domainpb.Run) AgentRunSummary {
	summary := AgentRunSummary{
		RunID:           run.GetId(),
		TaskID:          run.GetTaskId(),
		Tag:             run.GetTag(),
		Status:          ProtoRunStatusToLocal(run.GetStatus()),
		Phase:           protoRunPhaseToString(run.GetPhase()),
		ProgressPercent: int(run.GetProgressPercent()),
	}
	if ts := run.GetCreatedAt(); ts != nil {
		summary.CreatedAt = ts.AsTime()
	}
	if ts := run.GetUpdatedAt(); ts != nil {
		summary.UpdatedAt = ts.AsTime()
	}
	return summary
}
