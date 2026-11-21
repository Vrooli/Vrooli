package lifecycle

import (
	"testing"

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestProviderAbsenceRequiresDeadOwnerAndKnownUnboundPort(t *testing.T) {
	for _, name := range []string{"dead-api-live-ui", "expired-claim", "live-api", "listener", "unknown-listener", "missing-ref", "missing-claim", "foreign-boot", "starting", "unknown-pid", "live-helper", "foreign-ref", "undeclared-api"} {
		t.Run(name, func(t *testing.T) {
			item := scenario.Scenario{Manifest: scenario.ServiceManifest{
				Ports:      map[string]scenario.Port{"api": {EnvVar: "API_PORT"}},
				Components: map[string]scenario.Component{"backend": {Role: "api", Run: scenario.ComponentRun{Port: "api"}}, "frontend": {Role: "ui"}},
			}}
			instance := scenarioruntime.Instance{HostBootID: "boot", Status: scenarioruntime.StatusRunning}
			pid, uiPID := 101, 102
			claims := []scenarioruntime.PortClaim{{PortName: "api", Port: 18000}}
			refs := []scenarioruntime.ProcessRef{{Step: "start-backend", PID: &pid, HostBootID: "boot"}, {Step: "start-frontend", PID: &uiPID, HostBootID: "boot"}}
			state := network.ListenerState{Known: true}
			liveAPI := false
			switch name {
			case "expired-claim":
				claims[0].Status = scenarioruntime.ClaimStatusExpired
			case "live-api":
				liveAPI = true
			case "listener":
				state.Listening = true
			case "unknown-listener":
				state.Known = false
			case "missing-ref":
				refs = refs[1:]
			case "missing-claim":
				claims = nil
			case "foreign-boot":
				instance.HostBootID = "old"
			case "starting":
				instance.Status = scenarioruntime.StatusStarting
			case "unknown-pid":
				refs[0].PID = nil
			case "live-helper":
				refs = append(refs, scenarioruntime.ProcessRef{Step: "start-helper", PID: &uiPID, HostBootID: "boot"})
			case "foreign-ref":
				refs[0].HostBootID = "old"
			case "undeclared-api":
				delete(item.Manifest.Components, "backend")
			}
			got := providerKnownAbsent(item, instance, "boot", claims, refs, func(int) network.ListenerState { return state }, func(p int) bool { return p == uiPID || liveAPI })
			if got != (name == "dead-api-live-ui" || name == "expired-claim") {
				t.Fatalf("providerKnownAbsent = %v", got)
			}
		})
	}
}

func TestProviderAbsenceAcceptsOnlyDeadPrelaunchLifecycleStartup(t *testing.T) {
	for _, name := range []string{"dead-owner-setup", "live-owner", "missing-owner", "supervised-owner", "component-phase", "has-process-ref", "authored-setup-exec", "unknown-listener", "listener", "missing-api-claim"} {
		t.Run(name, func(t *testing.T) {
			item := scenario.Scenario{Manifest: scenario.ServiceManifest{Ports: map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}}}
			ownerPID, apiPID := 101, 102
			instance := scenarioruntime.Instance{
				HostBootID: "boot", Status: scenarioruntime.StatusStarting,
				OwnerKind: scenarioruntime.OwnerKindLifecycle, OwnerPID: &ownerPID, Phase: "setup",
			}
			if name == "authored-setup-exec" {
				item.Manifest.Lifecycle.Setup.Steps = []scenario.PhaseStep{{Name: "untracked", Exec: []string{"sh", "-c", "true"}}}
			}
			claims := []scenarioruntime.PortClaim{{PortName: "api", EnvVar: "API_PORT", Port: 18000, Status: scenarioruntime.ClaimStatusReserved}}
			var refs []scenarioruntime.ProcessRef
			state := network.ListenerState{Known: true}
			ownerLive := false
			switch name {
			case "live-owner":
				ownerLive = true
			case "missing-owner":
				instance.OwnerPID = nil
			case "supervised-owner":
				instance.OwnerKind = scenarioruntime.OwnerKindSupervisor
			case "component-phase":
				instance.Phase = "develop"
			case "has-process-ref":
				refs = []scenarioruntime.ProcessRef{{Step: "start-api", PID: &apiPID, HostBootID: "boot"}}
			case "unknown-listener":
				state.Known = false
			case "listener":
				state.Listening = true
			case "missing-api-claim":
				claims = nil
			}

			got := providerKnownAbsent(item, instance, "boot", claims, refs,
				func(int) network.ListenerState { return state },
				func(pid int) bool { return pid == ownerPID && ownerLive },
			)
			if got != (name == "dead-owner-setup") {
				t.Fatalf("providerKnownAbsent = %v, want %v", got, name == "dead-owner-setup")
			}
		})
	}
}
