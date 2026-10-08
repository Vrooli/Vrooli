package agentmanager

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// ListExecutionOptions returns the truthful runner/objective catalog owned by
// Agent Manager. Swarm uses it only for goal-session admission; it does not
// copy runner configuration into its own declarations.
func (c *HTTPClient) ListExecutionOptions(ctx context.Context, role string) ([]*apipb.ExecutionOption, error) {
	path := "/api/v1/execution-options"
	if role = strings.TrimSpace(role); role != "" {
		path += "?" + url.Values{"role": []string{role}}.Encode()
	}
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, readErrorResponse(resp)
	}
	var result apipb.ListExecutionOptionsResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Options == nil {
		return nil, fmt.Errorf("%w: execution options response omitted options", ErrRequestFailed)
	}
	return result.Options, nil
}

// ListExecutionOptions exposes Agent Manager's catalog through Swarm's
// integration seam without making HTTP details part of a route handler.
func (s *AgentService) ListExecutionOptions(ctx context.Context, role string) ([]*apipb.ExecutionOption, error) {
	if !s.enabled {
		return nil, fmt.Errorf("agent-manager not enabled")
	}
	return s.client.ListExecutionOptions(ctx, role)
}

// NativeGoalRunnersAvailable reports whether at least one available runner can
// execute a native objective. It is intentionally a capability query rather
// than a runner selection promise; Agent Manager remains the selector.
func (s *AgentService) NativeGoalRunnersAvailable(ctx context.Context) (bool, error) {
	if !s.enabled {
		return false, nil
	}
	options, err := s.client.ListExecutionOptions(ctx, "code.smart")
	if err != nil {
		return false, err
	}
	for _, option := range options {
		if option != nil && option.GetAvailable() && option.GetNativeObjective() {
			return true, nil
		}
	}
	return false, nil
}
