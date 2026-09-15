package lifecycle

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/vrooli/internal/scenario"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type providerFence struct {
	client      *apilifecycle.Client
	operationID string
	token       string
	revision    int64
	active      bool
}

// prepareProviderFence negotiates the standard scenario-owned lifecycle
// protocol. A scenario without the protocol is not special-cased here; it is
// simply an unguarded provider and remains the scenario's responsibility.
func (r *Runner) prepareProviderFence(ctx context.Context, item scenario.Scenario, emergency bool, overrideReason, operationID string) (*providerFence, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if emergency && len([]byte(strings.TrimSpace(overrideReason))) < 1 {
		return nil, fmt.Errorf("lifecycle emergency override for %q requires a reason", item.Slug)
	}
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("lifecycle provider for %q is not authoritative", item.Slug)
	}
	// Agent Manager owns the authoritative maintenance admission used by the
	// lifecycle gate. When its registry view is stale or lacks an API port,
	// defer to that owner read instead of failing here before the owner
	// precondition can run. The subsequent requireOwnerMaintenance call still
	// fails closed when the owner cannot provide a complete observation.
	if item.Slug == "agent-manager" && (!view.Authoritative || view.Ports["API_PORT"] <= 0) {
		return nil, nil
	}
	if !view.Authoritative || view.Ports["API_PORT"] <= 0 {
		return nil, fmt.Errorf("lifecycle provider for %q is not authoritative", item.Slug)
	}
	client, err := apilifecycle.NewClient(&http.Client{Timeout: 10 * time.Second}, fmt.Sprintf("http://127.0.0.1:%d", view.Ports["API_PORT"]))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(operationID) == "" {
		operationID = uuid.NewString()
	}
	reason := "control-plane scenario replacement"
	if emergency {
		reason += ": " + strings.TrimSpace(overrideReason)
	}
	resp, err := client.Prepare(ctx, &commonv1.LifecyclePrepareRequest{OperationId: operationID, Reason: reason, EmergencyOverride: emergency, OverrideReason: strings.TrimSpace(overrideReason)})
	if err != nil {
		if code := connect.CodeOf(err); code == connect.CodeNotFound || code == connect.CodeUnimplemented {
			return nil, nil
		}
		return nil, fmt.Errorf("prepare lifecycle fence for %q: %w", item.Slug, err)
	}
	if err := apilifecycle.ValidateStanding(resp.GetStanding()); err != nil {
		return nil, err
	}
	standing := resp.GetStanding()
	if !emergency && !standing.GetAdmissionClosed() {
		return nil, fmt.Errorf("lifecycle provider %q did not close admission", item.Slug)
	}
	if !emergency && !standing.GetDrained() {
		drain, drainErr := client.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: operationID, FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()})
		if drainErr != nil {
			return nil, fmt.Errorf("drain lifecycle fence for %q: %w", item.Slug, drainErr)
		}
		if err := apilifecycle.ValidateStanding(drain.GetStanding()); err != nil {
			return nil, err
		}
		standing = drain.GetStanding()
	}
	return &providerFence{client: client, operationID: operationID, token: standing.GetFenceToken(), revision: standing.GetRevision(), active: true}, nil
}

func (f *providerFence) resume(ctx context.Context) error {
	if f == nil || !f.active {
		return nil
	}
	_, err := f.client.Resume(ctx, &commonv1.LifecycleResumeRequest{OperationId: f.operationID, FenceToken: f.token, Revision: f.revision})
	return err
}
