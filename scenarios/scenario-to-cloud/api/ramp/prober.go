package ramp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"

	"github.com/vrooli/api-core/targetmodel"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// RampID names this ramp in inventory targets and verdicts.
const RampID = "scenario-to-cloud"

// CapabilityCloudLaunch is the capability every reachable cloud target offers.
const CapabilityCloudLaunch = "cloud.launch"

// Prober projects the cloud deployments onto the spine inventory. Each
// deployment's target binding is one target (keyed like the deployment
// identity, machine:<id> or host:<host>); readiness comes from the typed
// health observation, never from the deployment record alone.
type Prober struct {
	Deployments Deployments
	Observer    Observer
	Now         func() time.Time
}

// Probe implements deliveryramp.Prober.
func (p Prober) Probe(ctx context.Context, request deliveryramp.ProbeRequest) (deliveryramp.Inventory, error) {
	if p.Deployments == nil {
		return deliveryramp.Inventory{}, fmt.Errorf("cloud prober has no deployment source")
	}
	now := time.Now()
	if p.Now != nil {
		now = p.Now()
	}
	deployments, err := p.Deployments.ListDeployments(ctx, domain.ListFilter{})
	if err != nil {
		return deliveryramp.Inventory{}, fmt.Errorf("list cloud deployments: %w", err)
	}
	inventory := deliveryramp.Inventory{Observed: now.UTC()}
	for _, dep := range deployments {
		if dep == nil {
			continue
		}
		target := targetFor(dep)
		if !transportAllowed(target.Transport.Kind, request.TransportKinds) {
			continue
		}
		if p.Observer != nil {
			observation, err := p.Observer.Observe(ctx, dep.ID)
			applyObservation(&target, observation, err)
		} else {
			markUnavailable(&target, "target_reach", "no health observer is configured", "configure the health observer and probe again")
		}
		inventory.Targets = append(inventory.Targets, target)
	}
	if len(inventory.Targets) == 0 {
		inventory.Targets = append(inventory.Targets, deliveryramp.UnavailableTarget("no cloud deployment matches the probe", "cloud deployment"))
	}
	if err := inventory.Validate(); err != nil {
		return deliveryramp.Inventory{}, err
	}
	return inventory, nil
}

// TargetKey is the spine target id for a deployment: the deployment's own
// target key, or the deployment id when the binding is empty.
func TargetKey(dep *domain.Deployment) string {
	if dep == nil {
		return ""
	}
	if key := dep.Target.Key(); key != "" {
		return key
	}
	if host := manifestHost(dep); host != "" {
		return "host:" + host
	}
	return "deployment:" + dep.ID
}

func manifestHost(dep *domain.Deployment) string {
	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil || m.Target.VPS == nil {
		return ""
	}
	return strings.TrimSpace(m.Target.VPS.Host)
}

func targetFor(dep *domain.Deployment) deliveryramp.Target {
	key := TargetKey(dep)
	transport := deliveryramp.Transport{ID: key, Available: true}
	switch dep.Target.Transport {
	case identity.TransportBridge:
		transport.Kind = deliveryramp.TransportBridge
	default:
		transport.Kind = targetmodel.TransportSSH
	}
	return deliveryramp.Target{
		ID: key, Ramp: RampID, Label: dep.Name, Platform: "vps", OS: "linux", DeviceKind: "host", Mode: "remote",
		NodeID: dep.Target.NodeID, Transport: transport, Capabilities: []string{CapabilityCloudLaunch}, Available: true,
		Revision: fmt.Sprintf("fence:%d", dep.Fence),
	}
}

func transportAllowed(kind deliveryramp.TransportKind, allowed []deliveryramp.TransportKind) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, candidate := range allowed {
		if candidate == kind {
			return true
		}
	}
	return false
}

func markUnavailable(target *deliveryramp.Target, missing, reason, next string) {
	target.Available = false
	target.MissingCapability = missing
	target.Reason = reason
	target.NextAction = next
	target.Transport.Available = false
	target.Transport.Reason = reason
}

// applyObservation carries the observation's own cause into the target.
// Unknown status or non-current freshness is unavailable, not degraded.
func applyObservation(target *deliveryramp.Target, obs *healthv1.HealthObservation, err error) {
	switch {
	case err != nil:
		markUnavailable(target, "target_reach", "health observation failed: "+err.Error(), "restore reach to the target and probe again")
	case obs == nil:
		markUnavailable(target, "target_reach", "health observer returned no observation", "restore reach to the target and probe again")
	case obs.GetFreshness() != healthv1.Freshness_FRESHNESS_CURRENT:
		markUnavailable(target, "target_reach", fmt.Sprintf("health observation is %s, not current", obs.GetFreshness()), "repair transport reach so a current observation exists")
	case obs.GetStatus() == healthv1.HealthStatus_HEALTH_STATUS_HEALTHY, obs.GetStatus() == healthv1.HealthStatus_HEALTH_STATUS_DEGRADED:
		target.Health = deliveryramp.TargetHealth{Status: strings.ToLower(strings.TrimPrefix(obs.GetStatus().String(), "HEALTH_STATUS_"))}
		target.LastSeenAt = obs.GetObservedAt().AsTime().UTC()
		for _, check := range obs.GetChecks() {
			target.Readiness = append(target.Readiness, targetmodel.ReadinessCheck{Identity: check.GetId(), Label: check.GetId(), Passed: check.GetStatus() == healthv1.CheckStatus_CHECK_STATUS_PASSED, Detail: check.GetReasonCode()})
		}
	default:
		markUnavailable(target, "target_health", fmt.Sprintf("health observation status is %s", obs.GetStatus()), "inspect the deployment health observation and repair the failed checks")
	}
}

var _ deliveryramp.Prober = Prober{}
