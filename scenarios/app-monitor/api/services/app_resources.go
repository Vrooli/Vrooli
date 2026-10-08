// Package services provides resource management services for app-monitor.
package services

import (
	"context"
	"fmt"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// cliClient is the shared typed Vrooli CLI client for the services package
// (resource and scenario discovery, status, and lifecycle). It decodes the
// vrooli.cli.v1 contracts instead of hand-parsing CLI JSON, so a CLI output
// change is a compile error here rather than a silently empty or wrong result.
var cliClient = vroolicli.New()

// ResourceStatus represents the status of a Vrooli resource as surfaced to the
// app-monitor UI. Every field is populated from the typed `vrooli resource
// status` contract (cliv1.ResourceStatus).
type ResourceStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
}

// GetResources returns the status of all Vrooli resources.
func (s *AppService) GetResources(ctx context.Context) ([]ResourceStatus, error) {
	resp, err := cliClient.ResourceStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vrooli resources: %w", err)
	}

	result := make([]ResourceStatus, 0, len(resp.GetResources()))
	for _, rs := range resp.GetResources() {
		if rs.GetResource().GetName() == "" {
			continue
		}
		result = append(result, toResourceStatus(rs))
	}
	return result, nil
}

// GetResource returns detailed information about a specific resource.
func (s *AppService) GetResource(ctx context.Context, resourceID string) (*ResourceStatus, error) {
	if strings.TrimSpace(resourceID) == "" {
		return nil, fmt.Errorf("resource_id is required")
	}

	resp, err := cliClient.ResourceStatus(ctx, resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource %s: %w", resourceID, err)
	}
	if resp.GetResource().GetResource().GetName() == "" {
		return nil, fmt.Errorf("resource %s not found", resourceID)
	}
	status := toResourceStatus(resp.GetResource())
	return &status, nil
}

// StartResource starts a Vrooli resource.
func (s *AppService) StartResource(ctx context.Context, resourceID string) error {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("resource_id is required")
	}
	if _, err := cliClient.Output(ctx, "resource", "start", resourceID); err != nil {
		return fmt.Errorf("failed to start resource %s: %w", resourceID, err)
	}
	return nil
}

// StopResource stops a Vrooli resource.
func (s *AppService) StopResource(ctx context.Context, resourceID string) error {
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("resource_id is required")
	}
	if _, err := cliClient.Output(ctx, "resource", "stop", resourceID); err != nil {
		return fmt.Errorf("failed to stop resource %s: %w", resourceID, err)
	}
	return nil
}

// toResourceStatus maps the typed cliv1.ResourceStatus onto the UI-facing
// ResourceStatus. The `type` chip carries the backing driver (e.g.
// "compose-service"); status is derived from the real runtime + registration
// flags the CLI reports.
func toResourceStatus(rs *cliv1.ResourceStatus) ResourceStatus {
	res := rs.GetResource()
	return ResourceStatus{
		ID:      res.GetName(),
		Name:    res.GetName(),
		Type:    res.GetDriver(),
		Status:  deriveResourceStatus(rs),
		Enabled: res.GetEnabled(),
		Running: rs.GetRunning(),
	}
}

// deriveResourceStatus collapses the runtime/registration/health flags into the
// UI status token. Health failures win over plain running/stopped state.
func deriveResourceStatus(rs *cliv1.ResourceStatus) string {
	res := rs.GetResource()
	switch {
	case !res.GetRegistered():
		return "unregistered"
	case strings.Contains(strings.ToLower(rs.GetStatusCode()), "error"),
		strings.Contains(strings.ToLower(rs.GetMessage()), "error"),
		strings.Contains(strings.ToLower(rs.GetMessage()), "failed"),
		rs.GetProbeError() != "":
		return "error"
	case rs.GetRunning():
		return "online"
	case res.GetEnabled():
		return "stopped"
	default:
		return "offline"
	}
}
