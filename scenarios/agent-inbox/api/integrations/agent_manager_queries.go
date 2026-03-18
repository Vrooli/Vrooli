package integrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// CheckAgentStatus gets the status of a coding agent run as a proto Run.
// This is used by the reconciliation service to verify agent state after server restarts.
// On connection failure, re-resolves the agent-manager URL and retries once.
func (c *AgentManagerClient) CheckAgentStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	resp, err := c.doWithRetry(ctx, "GET", "/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
	}

	return result.GetRun(), nil
}

// GetEvents retrieves events for a run, optionally filtered by sequence.
func (c *AgentManagerClient) GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events?after_sequence=%d", runID, afterSequence)

	resp, err := c.doWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get events failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunEventsResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse events response: %w", err)
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
	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetRunStatus gets the current status of a run.
func (c *AgentManagerClient) GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error) {
	resp, err := c.doWithRetry(ctx, "GET", "/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
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
	path := buildListRunsPath(opts)

	resp, err := c.doWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list runs failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.ListRunsResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list runs response: %w", err)
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

// buildListRunsPath constructs the query path for listing runs.
func buildListRunsPath(opts ListRunsOptions) string {
	path := "/api/v1/runs?"
	params := make([]string, 0, 4)
	if opts.Status != "" {
		params = append(params, "status="+opts.Status)
	}
	if opts.TagPrefix != "" {
		params = append(params, "tag_prefix="+opts.TagPrefix)
	}
	if opts.Limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", opts.Limit))
	}
	if opts.Offset > 0 {
		params = append(params, fmt.Sprintf("offset=%d", opts.Offset))
	}
	return path + strings.Join(params, "&")
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
