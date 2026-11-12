package lifecycle

import (
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// providerKnownAbsent distinguishes a crashed provider from an unreachable live
// provider. A surviving UI must not make the dead API impossible to recover.
// Missing inventory, startup, foreign boots, live non-UI helpers and unknown
// listener observations all retain the ordinary maintenance fence requirement.
// This only skips an RPC to an absent provider; teardown still owns process
// identity checks and termination of the remaining UI.
func providerKnownAbsent(item scenario.Scenario, instance scenarioruntime.Instance, boot string, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef, listening func(int) network.ListenerState, alive func(int) bool) bool {
	if boot == "" || instance.HostBootID != boot {
		return false
	}
	// A lifecycle-owned startup that is still in planning or setup has not
	// entered the component launch phase: component launches are declared only
	// for develop. Authored setup commands are arbitrary and could leave a
	// provider behind, so they make absence unprovable just like a process ref.
	// If the exact owner is known dead, setup has no authored exec commands,
	// there are no process refs, and every claimed API port is known unbound,
	// the provider is absent despite the incomplete lease. This is deliberately
	// narrower than general stale-instance recovery.
	if instance.Status == scenarioruntime.StatusStarting {
		if instance.OwnerKind != scenarioruntime.OwnerKindLifecycle || instance.OwnerPID == nil || *instance.OwnerPID <= 0 || alive == nil || alive(*instance.OwnerPID) ||
			(instance.Phase != scenarioruntime.PhasePlanning && instance.Phase != "setup") || len(refs) != 0 || hasAuthoredSetupExec(item.Manifest) {
			return false
		}
		return apiClaimsKnownUnbound(item, claims, listening)
	}
	if instance.Status != scenarioruntime.StatusRunning {
		return false
	}
	apiSteps := map[string]bool{}
	uiSteps := map[string]bool{}
	for name, component := range item.Manifest.Components {
		if item.Manifest.PortEnvVar(component.Run.Port) == "API_PORT" {
			apiSteps["start-"+name] = true
		}
		if component.Role == "ui" {
			uiSteps["start-"+name] = true
		}
	}
	if len(apiSteps) != 1 {
		return false
	}
	if !apiClaimsKnownUnbound(item, claims, listening) {
		return false
	}
	apiRef := false
	for _, ref := range refs {
		if uiSteps[ref.Step] {
			continue
		}
		if ref.HostBootID != boot || ref.PID == nil || *ref.PID <= 0 || alive(*ref.PID) {
			return false
		}
		if apiSteps[ref.Step] {
			apiRef = true
		}
	}
	return apiRef
}

func hasAuthoredSetupExec(manifest scenario.ServiceManifest) bool {
	phase, ok := lookupPhase(manifest, phasesSetup)
	if !ok {
		return false
	}
	for _, step := range phase.Steps {
		if len(step.Exec) > 0 {
			return true
		}
	}
	return false
}

func apiClaimsKnownUnbound(item scenario.Scenario, claims []scenarioruntime.PortClaim, listening func(int) network.ListenerState) bool {
	apiClaim := false
	for _, claim := range claims {
		if claim.EnvVar != "API_PORT" && item.Manifest.PortEnvVar(claim.PortName) != "API_PORT" {
			continue
		}
		if claim.Port <= 0 || listening == nil {
			return false
		}
		state := listening(claim.Port)
		if !state.Known || state.Listening {
			return false
		}
		apiClaim = true
	}
	return apiClaim
}
