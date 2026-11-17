package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	apilifecycle "github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// emergencyDrainTimeout bounds how long a force-lifecycle restart waits for
// admitted requests and owner work to finish. The operator has already accepted
// that in-flight work may be interrupted, so a handler that never returns must
// not hold the replacement forever; a normal restart keeps waiting for the
// operation's own context.
var emergencyDrainTimeout = 60 * time.Second

func (r *Runner) startOperationFenceOwner(ctx context.Context, scenario, variant, fenceOperationID string) (scenarioruntime.StartOperation, error) {
	store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
	if err != nil {
		return scenarioruntime.StartOperation{}, err
	}
	owner, readErr := store.GetStartOperationFenceOwner(ctx, scenario, variant, fenceOperationID)
	closeErr := store.Close()
	return owner, errors.Join(readErr, closeErr)
}

type providerFence struct {
	client             *apilifecycle.Client
	scenario           string
	providerInstanceID string
	operationID        string
	token              string
	revision           int64
	interlock          string
	persistence        commonv1.LifecycleFencePersistence
	// recoveryFenceOperationID names a prior terminal attempt's linked durable
	// fence that may surface after a provider implementation transition. It is
	// used only with explicit lifecycle risk acceptance and an exact drained
	// durable proof from the replacement provider.
	recoveryFenceOperationID string
	recoveryReason           string
	recoveryLog              func(string, int64)
	active                   bool
}

// prepareProviderFence negotiates the standard scenario-owned lifecycle
// protocol. A live provider without the protocol stays serving: process
// ownership alone cannot close its ingress or prove owner quiescence.
func (r *Runner) prepareProviderFence(ctx context.Context, item scenario.Scenario, emergency bool, overrideReason, operationID string, prior *scenarioruntime.StartOperation, priorReadErr error) (*providerFence, error) {
	return r.prepareProviderFenceWithLink(ctx, item, emergency, overrideReason, operationID, prior, priorReadErr, nil)
}

// prepareProviderFenceWithLink records the durable owner as soon as Prepare
// has validated the closed fence identity. Drain may be an unbounded,
// server-owned wait; recovery must be attributable even while it is pending.
func (r *Runner) prepareProviderFenceWithLink(ctx context.Context, item scenario.Scenario, emergency bool, overrideReason, operationID string, prior *scenarioruntime.StartOperation, priorReadErr error, link func(string) error) (*providerFence, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if emergency && len([]byte(strings.TrimSpace(overrideReason))) < 1 {
		return nil, fmt.Errorf("lifecycle emergency override for %q requires a reason", item.Slug)
	}
	// Agent Manager is the execution owner, not a user workload provider. Its
	// own persistent Web Console sessions and durable run recovery are designed
	// to survive an Agent Manager process restart, so requiring Agent Manager to
	// fence itself creates a circular lifecycle dependency. Other scenarios keep
	// the ordinary provider-fence contract.
	if item.Slug == "agent-manager" {
		return nil, nil
	}
	if !scenarioDeclaresPortEnv(item, "API_PORT") {
		return nil, nil
	}
	view, err := r.lookupRegistryRuntime(ctx, item)
	if err != nil {
		if emergency {
			r.logProviderFenceOverride(item.Slug, overrideReason, "runtime registry unavailable: "+err.Error())
			return nil, nil
		}
		return nil, fmt.Errorf("lifecycle provider for %q is not authoritative", item.Slug)
	}
	// A stopped scenario has no provider process to fence. Restart still uses
	// the normal start pipeline, which will perform any required owner
	// maintenance and start a fresh instance. Keep rejecting a present but
	// non-authoritative instance below: that state may represent a live or
	// otherwise unsafe runtime whose ownership cannot be established.
	if !view.Present || view.ProviderAbsent {
		return nil, nil
	}
	if !view.Authoritative || view.Ports["API_PORT"] <= 0 {
		if emergency {
			r.logProviderFenceOverride(item.Slug, overrideReason, "live runtime has no authoritative API provider endpoint")
			return nil, nil
		}
		return nil, fmt.Errorf("lifecycle provider for %q is not authoritative", item.Slug)
	}
	client, err := apilifecycle.NewClient(&http.Client{Timeout: 10 * time.Second}, fmt.Sprintf("http://127.0.0.1:%d", view.Ports["API_PORT"]))
	if err != nil {
		return nil, err
	}
	instanceID := strings.TrimSpace(view.Instance.InstanceID)
	if instanceID == "" {
		return nil, fmt.Errorf("lifecycle provider for %q has no authoritative instance ID", item.Slug)
	}
	derivedOperationID := "restart-" + instanceID
	var recoveryStanding *commonv1.LifecycleStanding
	statusResponse, statusErr := client.Status(ctx, &commonv1.LifecycleStatusRequest{})
	if statusErr != nil {
		code := connect.CodeOf(statusErr)
		if code != connect.CodeNotFound && code != connect.CodeUnimplemented {
			if emergency {
				r.logWarn("Lifecycle maintenance proof bypassed by explicit operator risk acceptance",
					logx.AttrScenario, item.Slug,
					"stage", "status",
					"lifecycle_override_reason", strings.TrimSpace(overrideReason),
					"risk", "in-flight requests or owner work may be interrupted",
					"error", statusErr.Error(),
				)
				return nil, nil
			}
			return nil, fmt.Errorf("read lifecycle fence status for %q: %w", item.Slug, statusErr)
		}
		r.logWarn("Lifecycle provider does not implement the Status contract; proceeding without a provider fence",
			logx.AttrScenario, item.Slug,
			"stage", "status",
			"code", code.String(),
		)
		return nil, nil
	} else {
		if statusResponse == nil || statusResponse.GetStanding() == nil {
			return nil, fmt.Errorf("lifecycle provider %q returned no standing from Status", item.Slug)
		}
		standing := statusResponse.GetStanding()
		if err := validateStatusStanding(item.Slug, standing); err != nil {
			return nil, err
		}
		if standing.GetAdmissionClosed() {
			if requested := strings.TrimSpace(operationID); requested != "" && requested != standing.GetOperationId() {
				return nil, fmt.Errorf("lifecycle provider %q has closed fence %q; refusing different operation %q", item.Slug, standing.GetOperationId(), requested)
			}
			if strings.TrimSpace(operationID) == "" {
				exactOwnerFound := prior != nil && prior.FenceOperationID == standing.GetOperationId()
				// The latest attempt can be a successful, unlinked start (for
				// example, a recovery start after a failed replacement). Resolve
				// ownership by the provider's exact closed fence before considering
				// the latest attempt. This keeps a retained successful owner visible
				// without treating a different fence's success as authority.
				if !exactOwnerFound {
					owner, ownerErr := r.startOperationFenceOwner(ctx, item.Slug, item.Variant, standing.GetOperationId())
					if ownerErr == nil {
						if owner.Scenario != item.Slug || owner.Variant != (scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}).Normalize().Variant || (owner.Operation != lifecycleOperationRestart && owner.Operation != "start") || !owner.IsTerminal() {
							return nil, fmt.Errorf("lifecycle provider %q fence %q has a nonterminal or mismatched retained owner", item.Slug, standing.GetOperationId())
						}
						prior = &owner
						exactOwnerFound = true
					} else if !errors.Is(ownerErr, scenarioruntime.ErrNotFound) {
						return nil, fmt.Errorf("read exact lifecycle fence owner for %q fence %q: %w", item.Slug, standing.GetOperationId(), ownerErr)
					}
				}
				switch {
				case emergency && priorReadErr == nil &&
					prior != nil && prior.Scenario == item.Slug && prior.Variant == (scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}).Normalize().Variant &&
					prior.Operation == lifecycleOperationRestart && prior.IsTerminal() && prior.FenceOperationID == "":
					// Explicit recovery may adopt an unlinked fence only when the
					// exact scenario's prior restart is terminal. The caller already
					// holds the scenario lock; Prepare below must echo the provider's
					// exact operation/token/revision, then link this attempt before
					// any stop/replacement effect.
				case priorReadErr != nil:
					return nil, fmt.Errorf("lifecycle provider %q has closed fence %q and its start-operation owner could not be read: %w", item.Slug, standing.GetOperationId(), priorReadErr)
				case prior != nil && prior.Scenario == item.Slug && prior.Variant == (scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}).Normalize().Variant && (prior.Operation == lifecycleOperationRestart || prior.Operation == lifecycleOperationStart) && prior.FenceOperationID == standing.GetOperationId() && prior.IsTerminal() && (prior.Status == scenarioruntime.StartOperationStatusFailed || prior.Status == scenarioruntime.StartOperationStatusAbandoned):
					// A failed replacement may have registered a new provider
					// instance. Only the exact linked terminal attempt can recover.
				case emergency && prior != nil && prior.Scenario == item.Slug && prior.Variant == (scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}).Normalize().Variant && (prior.Operation == lifecycleOperationRestart || prior.Operation == lifecycleOperationStart) && prior.FenceOperationID == standing.GetOperationId() && prior.IsTerminal():
					// Exact operation-to-fence linkage is the owner proof,
					// including a succeeded attempt whose durable fence survived a
					// provider implementation transition.
				case link != nil && !exactOwnerFound && priorReadErr == nil && (prior == nil || prior.IsTerminal()) &&
					standing.GetFencePersistence() == commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_PROCESS_LOCAL &&
					standing.GetProviderInstanceId() == instanceID && standing.GetOperationId() == derivedOperationID:
					// The current process-local provider generation is itself the
					// owner of this fence. Adopt it only when its reported instance
					// matches the registered serving instance and its fence ID is the
					// deterministic ID for that generation. The current attempt is
					// linked immediately after Prepare echoes this exact identity and
					// before Drain or Stop. Durable or cross-generation fences still
					// require their retained operation owner.
				default:
					owner := "none"
					if prior != nil {
						owner = fmt.Sprintf("scenario=%s variant=%s operation=%s status=%s fence=%s", prior.Scenario, prior.Variant, prior.Operation, prior.Status, prior.FenceOperationID)
					}
					return nil, fmt.Errorf("lifecycle provider %q has closed fence %q without an exact retained start-operation owner; refusing recovery (derived=%s emergency=%t prior_read_error=%t owner %s)", item.Slug, standing.GetOperationId(), derivedOperationID, emergency, priorReadErr != nil, owner)
				}
				operationID = standing.GetOperationId()
			}
			recoveryStanding = standing
		}
	}
	if strings.TrimSpace(operationID) == "" {
		operationID = derivedOperationID
	}
	recoveryFenceOperationID := ""
	if emergency && priorReadErr == nil && prior != nil && prior.Scenario == item.Slug &&
		prior.Variant == (scenarioruntime.InstanceKey{Scenario: item.Slug, Variant: item.Variant}).Normalize().Variant &&
		prior.IsTerminal() && prior.FenceOperationID != "" {
		recoveryFenceOperationID = prior.FenceOperationID
	}
	reason := "control-plane scenario replacement"
	if emergency {
		reason += ": " + strings.TrimSpace(overrideReason)
	}
	resp, err := client.Prepare(ctx, &commonv1.LifecyclePrepareRequest{OperationId: operationID, Reason: reason, EmergencyOverride: emergency, OverrideReason: strings.TrimSpace(overrideReason)})
	if err != nil {
		if code := connect.CodeOf(err); code == connect.CodeNotFound || code == connect.CodeUnimplemented {
			if emergency {
				r.logWarn("Lifecycle maintenance contract bypassed by explicit operator risk acceptance",
					logx.AttrScenario, item.Slug,
					"stage", "prepare",
					"lifecycle_override_reason", strings.TrimSpace(overrideReason),
					"risk", "in-flight requests or owner work may be interrupted",
					"error", err.Error(),
				)
				return nil, nil
			}
			return nil, fmt.Errorf("live lifecycle provider %q does not implement the maintenance fence Prepare contract; refusing replacement: the API and external-helper owners must provide generation-bound ingress closure and quiescence before pinned process exit", item.Slug)
		}
		return nil, fmt.Errorf("prepare lifecycle fence for %q (operation %s): %w", item.Slug, operationID, err)
	}
	if err := apilifecycle.ValidateStanding(resp.GetStanding()); err != nil {
		return nil, err
	}
	standing := resp.GetStanding()
	if err := validateStatusStanding(item.Slug, standing); err != nil {
		return nil, err
	}
	if standing.GetOperationId() != operationID {
		return nil, fmt.Errorf("lifecycle provider %q returned operation %q for requested operation %q", item.Slug, standing.GetOperationId(), operationID)
	}
	if !standing.GetAdmissionClosed() {
		return nil, fmt.Errorf("lifecycle provider %q did not close admission", item.Slug)
	}
	if recoveryStanding != nil {
		if standing.GetProvider() != recoveryStanding.GetProvider() || standing.GetProviderInstanceId() != recoveryStanding.GetProviderInstanceId() ||
			standing.GetOperationId() != recoveryStanding.GetOperationId() || standing.GetFenceToken() != recoveryStanding.GetFenceToken() || standing.GetRevision() != recoveryStanding.GetRevision() {
			return nil, fmt.Errorf("lifecycle provider %q changed the retained fence identity during recovery", item.Slug)
		}
	}
	if standing.GetDrained() {
		if err := validateDrainedStanding(item.Slug, operationID, standing, standing); err != nil {
			return nil, err
		}
	}
	if link != nil {
		if err := link(operationID); err != nil {
			return nil, fmt.Errorf("persist provider fence recovery link for %q before drain: %w", item.Slug, err)
		}
	}
	newFence := func(standing *commonv1.LifecycleStanding) *providerFence {
		return &providerFence{
			client: client, scenario: item.Slug, providerInstanceID: standing.GetProviderInstanceId(),
			operationID: operationID, token: standing.GetFenceToken(), revision: standing.GetRevision(), active: true,
			interlock: standing.GetInterlock(), persistence: standing.GetFencePersistence(),
			recoveryFenceOperationID: recoveryFenceOperationID, recoveryReason: strings.TrimSpace(overrideReason),
			recoveryLog: func(operationID string, revision int64) {
				r.logWarn("Recovered retained durable lifecycle fence after provider implementation transition",
					logx.AttrScenario, item.Slug,
					"prior_durable_fence_operation_id", operationID,
					"revision", revision,
					"lifecycle_override_reason", strings.TrimSpace(overrideReason),
				)
			},
		}
	}
	// Explicit force-lifecycle acceptance may bypass a complete, simple
	// provider inventory when exactly one kind of work remains: admitted
	// requests, or tracked processes with no admitted requests. Never combine
	// the two counters in a bypass, and never skip component or pending-owner
	// quiescence. The process case is needed to recover stale PID inventory
	// after PID reuse; process identity must still be respected by the provider
	// when it next evaluates its tracker.
	bypassAdmittedRequests := standing.GetAdmitting() > 0 && standing.GetRemaining() == 0
	bypassTrackedWork := standing.GetAdmitting() == 0 && standing.GetRemaining() > 0
	if emergency && !standing.GetDrained() && standing.GetAdmissionClosed() &&
		standing.GetInventoryComplete() && (bypassAdmittedRequests || bypassTrackedWork) &&
		!standing.GetQuiescencePending() && len(standing.GetManagedComponents()) == 0 &&
		emergencyBlockersAreOverrideable(standing) {
		risk := "in-flight requests may be interrupted by process replacement"
		if bypassTrackedWork {
			risk = "tracked process work may be interrupted by process replacement"
		}
		r.logWarn("Lifecycle drain bypassed by explicit operator risk acceptance",
			logx.AttrScenario, item.Slug,
			"lifecycle_override_reason", strings.TrimSpace(overrideReason),
			"admitting", standing.GetAdmitting(),
			"remaining", standing.GetRemaining(),
			"risk", risk,
		)
		return newFence(standing), nil
	}
	// A completed exact retained fence may already contain this control plane's
	// successful completion. Reuse only the same provider/fence/generation set;
	// a tombstone or a newly asserted drained Prepare cannot create that proof.
	retainedComponentCompletion := recoveryStanding != nil && recoveryStanding.GetDrained() && standing.GetDrained() && validateComponentStopStanding(item.Slug, recoveryStanding, standing) == nil
	if standing.GetQuiescencePending() || (len(standing.GetManagedComponents()) > 0 && !retainedComponentCompletion) || (!standing.GetDrained() && (!emergency || !onlyOverrideableLifecycleBlockers(standing))) {
		// Drain is a server-owned wait for admitted work, unlike the bounded
		// Status/Prepare probes. Its lifetime follows the lifecycle operation's
		// context; a ten-second HTTP probe ceiling must not interrupt it.
		drainClient, err := apilifecycle.NewClient(&http.Client{}, fmt.Sprintf("http://127.0.0.1:%d", view.Ports["API_PORT"]))
		if err != nil {
			return nil, err
		}
		components := standing.GetManagedComponents()
		if standing.GetQuiescencePending() || len(components) > 0 {
			quiesced, quiesceErr := drainClient.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: operationID, FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), QuiesceComponents: true})
			if quiesceErr != nil {
				return nil, fmt.Errorf("quiesce managed components for %q: %w", item.Slug, quiesceErr)
			}
			q := quiesced.GetStanding()
			if err := validateComponentStopStanding(item.Slug, standing, q); err != nil {
				return nil, err
			}
			components = q.GetManagedComponents()
			stopCtx, cancel := context.WithTimeout(ctx, teardownTerminatePolicy.Timeout)
			stopErr := r.stopManagedComponents(stopCtx, item, components)
			cancel()
			if stopErr != nil {
				return nil, fmt.Errorf("stop managed components for %q: %w", item.Slug, stopErr)
			}
		}
		drainRequest := &commonv1.LifecycleDrainRequest{OperationId: operationID, FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision(), ComponentExits: components}
		drainCtx := ctx
		if emergency {
			// The provider enforces the bound and names what is still running;
			// the local deadline only backstops a provider that ignores it.
			drainRequest.Timeout = durationpb.New(emergencyDrainTimeout)
			var cancelDrain context.CancelFunc
			drainCtx, cancelDrain = context.WithTimeout(ctx, emergencyDrainTimeout+10*time.Second)
			defer cancelDrain()
		}
		drain, drainErr := drainClient.Drain(drainCtx, drainRequest)
		if drainErr != nil {
			if emergency && ctx.Err() == nil && (connect.CodeOf(drainErr) == connect.CodeDeadlineExceeded || errors.Is(drainErr, context.DeadlineExceeded)) {
				if fence := r.acceptStuckDrain(ctx, client, newFence, item.Slug, overrideReason, operationID, standing, drainErr); fence != nil {
					return fence, nil
				}
			}
			return nil, fmt.Errorf("drain lifecycle fence for %q (operation %s): %w", item.Slug, operationID, drainErr)
		}
		if err := apilifecycle.ValidateStanding(drain.GetStanding()); err != nil {
			return nil, err
		}
		drained := drain.GetStanding()
		if err := validateDrainedStanding(item.Slug, operationID, standing, drained); err != nil {
			return nil, err
		}
		standing = drained
	}
	return newFence(standing), nil
}

// acceptStuckDrain decides whether a force-lifecycle restart may replace a
// provider whose bounded drain expired. The operator accepted interrupting
// in-flight requests and detached owner work, so work that never finishes is not
// allowed to hold the replacement forever. Nothing else is waived: the fence
// identity must still match, admission must be closed, the inventory complete,
// no component may await quiescence, and every blocker must be overrideable.
// The stuck work is logged with the override reason so the interruption is
// attributable. A nil result means the drain failure stands.
func (r *Runner) acceptStuckDrain(ctx context.Context, client *apilifecycle.Client, newFence func(*commonv1.LifecycleStanding) *providerFence, slug, overrideReason, operationID string, prepared *commonv1.LifecycleStanding, drainErr error) *providerFence {
	statusCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	response, err := client.Status(statusCtx, &commonv1.LifecycleStatusRequest{OperationId: operationID, FenceToken: prepared.GetFenceToken(), Revision: prepared.GetRevision()})
	if err != nil || response == nil || response.GetStanding() == nil {
		return nil
	}
	standing := response.GetStanding()
	if validateStatusStanding(slug, standing) != nil ||
		standing.GetOperationId() != operationID || standing.GetFenceToken() != prepared.GetFenceToken() || standing.GetRevision() != prepared.GetRevision() ||
		standing.GetProviderInstanceId() != prepared.GetProviderInstanceId() ||
		!standing.GetAdmissionClosed() || !standing.GetInventoryComplete() || standing.GetQuiescencePending() ||
		len(standing.GetManagedComponents()) > 0 || !emergencyBlockersAreOverrideable(standing) {
		return nil
	}
	r.logWarn("Lifecycle drain gave up on stuck work by explicit operator risk acceptance",
		logx.AttrScenario, slug,
		"lifecycle_override_reason", strings.TrimSpace(overrideReason),
		"drain_timeout", emergencyDrainTimeout.String(),
		"admitting", standing.GetAdmitting(),
		"remaining", standing.GetRemaining(),
		"stuck_work", describeStuckWork(standing),
		"drain_error", drainErr.Error(),
		"risk", "in-flight requests or owner work are interrupted by process replacement",
	)
	return newFence(standing)
}

// describeStuckWork names the work a provider still reports, longest-listed
// first as the provider ordered it, so a lifecycle log shows what was cut off.
func describeStuckWork(standing *commonv1.LifecycleStanding) string {
	const limit = 16
	items := standing.GetWork()
	parts := make([]string, 0, limit+1)
	for i, item := range items {
		if i == limit {
			parts = append(parts, fmt.Sprintf("and %d more", len(items)-limit))
			break
		}
		parts = append(parts, fmt.Sprintf("%s [%s] %s", item.GetId(), item.GetKind(), item.GetStatus()))
	}
	if len(parts) == 0 {
		return "provider listed no work items"
	}
	return strings.Join(parts, "; ")
}

func emergencyBlockersAreOverrideable(standing *commonv1.LifecycleStanding) bool {
	if standing == nil {
		return false
	}
	for _, blocker := range standing.GetBlockers() {
		if blocker == nil || blocker.GetDisposition() != commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE {
			return false
		}
	}
	return true
}

func (r *Runner) logProviderFenceOverride(scenario, reason, detail string) {
	r.logWarn("Lifecycle provider authority bypassed by explicit operator risk acceptance",
		logx.AttrScenario, scenario,
		"lifecycle_override_reason", strings.TrimSpace(reason),
		"risk", "provider ingress and owner work could not be proven drained before replacement",
		"detail", detail,
	)
}

func onlyOverrideableLifecycleBlockers(s *commonv1.LifecycleStanding) bool {
	if s == nil || !s.GetInventoryComplete() || !s.GetAdmissionClosed() || s.GetAdmitting() != 0 || s.GetRemaining() < 0 || len(s.GetManagedComponents()) > 0 || len(s.GetBlockers()) == 0 {
		return false
	}
	for _, blocker := range s.GetBlockers() {
		if blocker.GetDisposition() != commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE {
			return false
		}
	}
	return true
}

func validateComponentStopStanding(scenario string, prepared, quiesced *commonv1.LifecycleStanding) error {
	if err := validateStatusStanding(scenario, quiesced); err != nil {
		return err
	}
	if !quiesced.GetAdmissionClosed() || !quiesced.GetComponentStopReady() || quiesced.GetProviderInstanceId() != prepared.GetProviderInstanceId() || quiesced.GetOperationId() != prepared.GetOperationId() || quiesced.GetFenceToken() != prepared.GetFenceToken() || quiesced.GetRevision() != prepared.GetRevision() {
		return fmt.Errorf("lifecycle provider %q changed fence or has not quiesced owned work", scenario)
	}
	if quiesced.GetQuiescencePending() || quiesced.GetAdmitting() != 0 {
		return fmt.Errorf("lifecycle provider %q still requires owner quiescence or has admitted work", scenario)
	}
	expected := map[string]string{}
	for _, c := range prepared.GetManagedComponents() {
		if c.GetName() == "" || c.GetGeneration() == "" || expected[c.GetName()] != "" {
			return fmt.Errorf("invalid prepared component inventory")
		}
		expected[c.GetName()] = c.GetGeneration()
	}
	if len(expected) != len(quiesced.GetManagedComponents()) {
		return fmt.Errorf("managed component inventory changed during quiescence")
	}
	for _, c := range quiesced.GetManagedComponents() {
		if expected[c.GetName()] != c.GetGeneration() || c.GetGeneration() == "" {
			return fmt.Errorf("managed component generation changed during quiescence")
		}
		delete(expected, c.GetName())
	}
	return nil
}

func scenarioDeclaresPortEnv(item scenario.Scenario, envVar string) bool {
	for _, port := range item.Manifest.Ports {
		if strings.EqualFold(strings.TrimSpace(port.EnvVar), strings.TrimSpace(envVar)) {
			return true
		}
	}
	return false
}

func validateStatusStanding(scenario string, standing *commonv1.LifecycleStanding) error {
	if standing == nil {
		return fmt.Errorf("lifecycle provider %q returned no standing", scenario)
	}
	if standing.GetAdmitting() < 0 || standing.GetRemaining() < 0 {
		return fmt.Errorf("lifecycle provider %q returned negative admitted or remaining work counts", scenario)
	}
	if strings.TrimSpace(standing.GetProvider()) != scenario || strings.TrimSpace(standing.GetProviderInstanceId()) == "" {
		return fmt.Errorf("lifecycle provider %q returned mismatched or incomplete provider identity", scenario)
	}
	if !standing.GetInventoryComplete() {
		detail := lifecycleInventoryDiagnostic(standing)
		if detail == "" {
			return fmt.Errorf("lifecycle provider %q returned incomplete lifecycle inventory", scenario)
		}
		return fmt.Errorf("lifecycle provider %q returned incomplete lifecycle inventory (%s)", scenario, detail)
	}
	if standing.GetAdmissionClosed() && (strings.TrimSpace(standing.GetOperationId()) == "" || strings.TrimSpace(standing.GetFenceToken()) == "") {
		return fmt.Errorf("lifecycle provider %q reports a closed fence without an operation ID and token", scenario)
	}
	if reportedScenario := strings.TrimSpace(standing.GetScenario()); reportedScenario != "" && reportedScenario != scenario {
		return fmt.Errorf("lifecycle provider %q returned standing for scenario %q", scenario, reportedScenario)
	}
	return nil
}

// lifecycleInventoryDiagnostic includes bounded aggregate counts in a refusal
// without copying provider-authored strings into control-plane errors or logs.
func lifecycleInventoryDiagnostic(standing *commonv1.LifecycleStanding) string {
	if standing == nil {
		return ""
	}
	var blockers, blocking, overrideable, unknown, unspecified int
	for _, blocker := range standing.GetBlockers() {
		if blocker == nil {
			continue
		}
		blockers++
		switch blocker.GetDisposition() {
		case commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_BLOCKING:
			blocking++
		case commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_OVERRIDEABLE:
			overrideable++
		case commonv1.LifecycleBlockerDisposition_LIFECYCLE_BLOCKER_DISPOSITION_UNKNOWN:
			unknown++
		default:
			unspecified++
		}
	}
	var work, components int
	for _, item := range standing.GetWork() {
		if item != nil {
			work++
		}
	}
	for _, component := range standing.GetManagedComponents() {
		if component != nil {
			components++
		}
	}
	return fmt.Sprintf("inventory counts: blockers=%d (blocking=%d, overrideable=%d, unknown=%d, unspecified=%d), work=%d, managed_components=%d", blockers, blocking, overrideable, unknown, unspecified, work, components)
}

func validateDrainedStanding(scenario, operationID string, prepared, drained *commonv1.LifecycleStanding) error {
	if err := validateStatusStanding(scenario, drained); err != nil {
		return err
	}
	if drained.GetOperationId() != operationID {
		return fmt.Errorf("lifecycle provider %q drain returned operation %q, want %q", scenario, drained.GetOperationId(), operationID)
	}
	if drained.GetProvider() != prepared.GetProvider() || drained.GetProviderInstanceId() != prepared.GetProviderInstanceId() {
		return fmt.Errorf("lifecycle provider %q drain changed provider identity", scenario)
	}
	if drained.GetFenceToken() != prepared.GetFenceToken() || drained.GetRevision() != prepared.GetRevision() {
		return fmt.Errorf("lifecycle provider %q drain changed fence identity", scenario)
	}
	if drained.GetInterlock() != prepared.GetInterlock() || drained.GetFencePersistence() != prepared.GetFencePersistence() {
		return fmt.Errorf("lifecycle provider %q drain changed fence persistence contract", scenario)
	}
	if !drained.GetAdmissionClosed() {
		return fmt.Errorf("lifecycle provider %q reopened admission while draining", scenario)
	}
	if drained.GetQuiescencePending() || drained.GetAdmitting() != 0 || drained.GetRemaining() != 0 || len(drained.GetBlockers()) != 0 {
		return fmt.Errorf("lifecycle provider %q reports drained with pending owner quiescence or work", scenario)
	}
	if !drained.GetDrained() {
		return fmt.Errorf("lifecycle provider %q did not report drained work", scenario)
	}
	return nil
}

func (f *providerFence) resume(ctx context.Context) error {
	if f == nil || !f.active {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	status, err := f.client.Status(ctx, &commonv1.LifecycleStatusRequest{})
	if err != nil {
		return fmt.Errorf("read lifecycle provider generation before resume: %w", err)
	}
	standing := status.GetStanding()
	if err := validateStatusStanding(f.scenario, standing); err != nil {
		return err
	}
	if standing.GetProviderInstanceId() != f.providerInstanceID {
		if !standing.GetAdmissionClosed() {
			if f.persistence == commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE {
				return fmt.Errorf("replacement lifecycle provider %q opened without resuming durable fence %q", f.scenario, f.operationID)
			}
			// A process-local provider starts open. Its predecessor's token must
			// never be replayed against the replacement.
			f.active = false
			return nil
		}
		// A durable owner deliberately retains the admission gate across process
		// replacement. Its fresh complete inventory and exact frozen identity
		// must prove the same drained operation. An unspecified predecessor can
		// be upgraded by that explicit durable proof; a known process-local
		// predecessor cannot. None of these safety facts creates authorization.
		if f.recoveryFenceOperationID != "" && f.recoveryFenceOperationID == standing.GetOperationId() &&
			f.recoveryReason != "" && standing.GetFencePersistence() == commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE &&
			standing.GetDrained() && !standing.GetQuiescencePending() && standing.GetAdmitting() == 0 && standing.GetRemaining() == 0 && len(standing.GetBlockers()) == 0 {
			if f.recoveryLog != nil {
				f.recoveryLog(standing.GetOperationId(), standing.GetRevision())
			}
			resumed, err := f.client.Resume(ctx, &commonv1.LifecycleResumeRequest{OperationId: standing.GetOperationId(), FenceToken: standing.GetFenceToken(), Revision: standing.GetRevision()})
			if err != nil {
				return fmt.Errorf("resume retained durable provider fence %q: %w", standing.GetOperationId(), err)
			}
			opened := resumed.GetStanding()
			if err := validateStatusStanding(f.scenario, opened); err != nil {
				return err
			}
			if opened.GetProviderInstanceId() != standing.GetProviderInstanceId() || opened.GetAdmissionClosed() || opened.GetOperationId() != "" {
				return fmt.Errorf("replacement lifecycle provider %q did not confirm durable fence recovery", f.scenario)
			}
			f.active = false
			return nil
		}
		if (f.persistence != commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_UNSPECIFIED && f.persistence != commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE) ||
			standing.GetFencePersistence() != commonv1.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE ||
			f.interlock == "" || standing.GetInterlock() != f.interlock ||
			standing.GetOperationId() != f.operationID || standing.GetFenceToken() != f.token || standing.GetRevision() != f.revision ||
			!standing.GetDrained() || standing.GetQuiescencePending() || standing.GetAdmitting() != 0 || standing.GetRemaining() != 0 || len(standing.GetBlockers()) != 0 {
			return fmt.Errorf("replacement lifecycle provider %q generation %q has no exact drained durable fence proof", standing.GetProvider(), standing.GetProviderInstanceId())
		}
		resumed, err := f.client.Resume(ctx, &commonv1.LifecycleResumeRequest{OperationId: f.operationID, FenceToken: f.token, Revision: f.revision})
		if err != nil {
			return err
		}
		opened := resumed.GetStanding()
		if err := validateStatusStanding(f.scenario, opened); err != nil {
			return err
		}
		if opened.GetProviderInstanceId() != standing.GetProviderInstanceId() || opened.GetAdmissionClosed() {
			return fmt.Errorf("replacement lifecycle provider %q did not confirm reopened admission", f.scenario)
		}
		f.active = false
		return nil
	}
	_, err = f.client.Resume(ctx, &commonv1.LifecycleResumeRequest{OperationId: f.operationID, FenceToken: f.token, Revision: f.revision})
	if err == nil {
		f.active = false
	}
	return err
}
