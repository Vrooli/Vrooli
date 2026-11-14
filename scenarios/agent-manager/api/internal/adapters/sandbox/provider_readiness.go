package sandbox

import (
	"context"
	"strings"

	"agent-manager/internal/domain"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

// ProviderReadinessReader exposes a read-only Workspace Sandbox lifecycle
// observation. It does not prepare, drain, or resume a maintenance fence.
type ProviderReadinessReader interface {
	ReadProviderReadiness(context.Context) domain.ProviderReadiness
}

const workspaceSandboxProviderName = "workspace-sandbox"

// ReadProviderReadiness reads the Workspace Sandbox lifecycle owner directly.
// A failed, incomplete, or foreign response is always represented as unknown.
func (p *WorkspaceSandboxProvider) ReadProviderReadiness(ctx context.Context) domain.ProviderReadiness {
	unknown := domain.ProviderReadiness{
		Provider: workspaceSandboxProviderName,
		State:    domain.ProviderReadinessUnknown,
		Known:    false,
		Message:  "Workspace Sandbox readiness could not be confirmed.",
	}
	baseURL := strings.TrimRight(strings.TrimSpace(p.baseURL), "/")
	if baseURL == "" || p.httpClient == nil {
		return unknown
	}
	client := commonconnect.NewLifecycleMaintenanceServiceClient(p.httpClient, baseURL)
	response, err := client.Status(ctx, connect.NewRequest(&commonv1.LifecycleStatusRequest{}))
	if err != nil || response == nil || response.Msg == nil || response.Msg.GetStanding() == nil {
		return unknown
	}
	standing := response.Msg.GetStanding()
	if standing.GetProvider() != workspaceSandboxProviderName || standing.GetScenario() != workspaceSandboxProviderName {
		return unknown
	}
	if standing.GetObservedAt() == nil || standing.GetObservedAt().CheckValid() != nil {
		return unknown
	}
	observedAt := standing.GetObservedAt().AsTime().UTC()
	if standing.GetAdmissionClosed() {
		return domain.ProviderReadiness{
			Provider:   workspaceSandboxProviderName,
			State:      domain.ProviderReadinessClosed,
			Known:      true,
			ObservedAt: &observedAt,
			Message:    "Workspace Sandbox is in lifecycle maintenance.",
		}
	}
	if standing.GetPhase() != commonv1.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN {
		return unknown
	}
	return domain.ProviderReadiness{
		Provider:   workspaceSandboxProviderName,
		State:      domain.ProviderReadinessOpen,
		Known:      true,
		ObservedAt: &observedAt,
		Message:    "Workspace Sandbox is ready for agent runs.",
	}
}

var _ ProviderReadinessReader = (*WorkspaceSandboxProvider)(nil)
