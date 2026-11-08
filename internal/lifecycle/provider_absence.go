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
	if boot == "" || instance.HostBootID != boot || instance.Status != scenarioruntime.StatusRunning {
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
	apiClaim := false
	for _, claim := range claims {
		if claim.EnvVar != "API_PORT" && item.Manifest.PortEnvVar(claim.PortName) != "API_PORT" {
			continue
		}
		if claim.Port <= 0 {
			return false
		}
		state := listening(claim.Port)
		if !state.Known || state.Listening {
			return false
		}
		apiClaim = true
	}
	if !apiClaim {
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
