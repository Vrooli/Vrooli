package lifecycle

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func listenerInspection(pids ...int) network.PortInspection {
	listeners := make([]network.PortListener, 0, len(pids))
	for _, pid := range pids {
		listeners = append(listeners, network.PortListener{PID: pid})
	}
	return network.PortInspection{
		Inspection: network.ListenerInspection{Available: true, Tool: "stub"},
		Listeners:  listeners,
	}
}

func managedListenerEnv(scenarioName, variant string) map[string]string {
	return map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
		"VROOLI_SCENARIO":          scenarioName,
		scenarioruntime.EnvVariant: variant,
	}
}

func TestReclaimSameInstanceListenerOnlySignalsExactInstance(t *testing.T) {
	signaled := make([]int, 0)
	runner := &Runner{deps: lifecycleDeps{
		inspectPort: func(port int) network.PortInspection {
			if port != 18196 {
				t.Fatalf("inspectPort(%d), want 18196", port)
			}
			return listenerInspection(101, 202, 303, 404)
		},
		readProcessEnv: func(pid int) (map[string]string, error) {
			switch pid {
			case 101:
				return managedListenerEnv("alpha", "live"), nil
			case 202:
				return managedListenerEnv("alpha", "shadow"), nil
			case 303:
				return managedListenerEnv("beta", "live"), nil
			case 404:
				return map[string]string{}, nil
			default:
				return nil, fmt.Errorf("unexpected pid %d", pid)
			}
		},
		signalPID: func(pid int, force bool) error {
			signaled = append(signaled, pid)
			return nil
		},
		isPIDRunning: func(int) bool { return false },
		sleep:        func(time.Duration) {},
	}}

	item := scenario.Scenario{Slug: "alpha", Variant: "live"}
	if err := runner.reclaimSameInstanceListener(context.Background(), item, 18196); err != nil {
		t.Fatalf("reclaimSameInstanceListener: %v", err)
	}
	if len(signaled) != 1 || signaled[0] != 101 {
		t.Fatalf("signaled = %v, want [101] (only the exact scenario+variant)", signaled)
	}
}

func TestKillClaimedPortOrphansUsesClaimEvidenceWithoutCrossingScenarios(t *testing.T) {
	signaled := make([]int, 0)
	runner := &Runner{deps: lifecycleDeps{
		inspectPort: func(port int) network.PortInspection {
			if port != 18196 {
				t.Fatalf("inspectPort(%d), want 18196", port)
			}
			return listenerInspection(101, 202, 303, 404)
		},
		readProcessEnv: func(pid int) (map[string]string, error) {
			switch pid {
			case 101:
				// The claim observed this PID on the port, but it lost the
				// managed environment (for example a hand-started child).
				return map[string]string{}, nil
			case 202:
				return managedListenerEnv("beta", "live"), nil
			case 303:
				return managedListenerEnv("alpha", "live"), nil
			case 404:
				return map[string]string{}, nil
			default:
				return nil, fmt.Errorf("unexpected pid %d", pid)
			}
		},
		signalPID: func(pid int, force bool) error {
			signaled = append(signaled, pid)
			return nil
		},
		isPIDRunning: func(int) bool { return false },
		sleep:        func(time.Duration) {},
	}}

	ports := map[int]struct{}{18196: {}}
	claimListeners := map[int][]int{18196: {101}}
	key := scenarioruntime.InstanceKey{Scenario: "alpha", Variant: "live"}
	if err := runner.killClaimedPortOrphansContext(context.Background(), ports, claimListeners, key); err != nil {
		t.Fatalf("killClaimedPortOrphansContext: %v", err)
	}
	if len(signaled) != 2 {
		t.Fatalf("signaled = %v, want claim-observed 101 and same-instance 303", signaled)
	}
	seen := map[int]bool{}
	for _, pid := range signaled {
		seen[pid] = true
	}
	if !seen[101] || !seen[303] {
		t.Fatalf("signaled = %v, want both 101 (claim evidence) and 303 (env match)", signaled)
	}
	if seen[202] || seen[404] {
		t.Fatalf("signaled = %v; beta-owned 202 and unknown 404 must not be killed", signaled)
	}
}

func TestBeginRuntimeRegistryStopCapturesDynamicPortClaimEvidence(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario:   "alpha",
		Variant:    "live",
		Status:     scenarioruntime.StatusRunning,
		HostBootID: "boot-test",
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		store.Close()
		t.Fatalf("CreateLease: %v", err)
	}
	listenerPID := 5151
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID:  instance.InstanceID,
		Scenario:    "alpha",
		Variant:     "live",
		PortName:    "api",
		EnvVar:      "API_PORT",
		Port:        18196,
		Status:      scenarioruntime.ClaimStatusBound,
		ListenerPID: &listenerPID,
	}); err != nil {
		store.Close()
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "boot-test", SessionID: "session-test"}, nil
		}
	})
	stop, err := runner.beginRuntimeRegistryStop(ctx, "alpha", "live")
	if err != nil {
		t.Fatalf("beginRuntimeRegistryStop: %v", err)
	}
	defer stop.close()
	if _, ok := stop.claimedPorts[18196]; !ok {
		t.Fatalf("claimedPorts = %v, want 18196", stop.claimedPorts)
	}
	pids := stop.claimListenerPIDs[18196]
	if len(pids) != 1 || pids[0] != listenerPID {
		t.Fatalf("claimListenerPIDs[18196] = %v, want [%d]", pids, listenerPID)
	}
}

// TestCleanupReclaimsDynamicClaimedPortOrphan reproduces the web-console
// restart failure: the instance's tracked process record is gone, but an
// earlier API child still listens on the dynamically claimed API port. Stop
// must clear that listener and verify the claimed port before releasing the
// claim, otherwise the next start fails with "port already in use".
func TestCleanupReclaimsDynamicClaimedPortOrphan(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")

	ctx := context.Background()
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		Scenario:   "alpha",
		Variant:    "live",
		Status:     scenarioruntime.StatusRunning,
		HostBootID: "boot-test",
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		store.Close()
		t.Fatalf("CreateLease: %v", err)
	}
	const orphanPID = 5151
	const claimedPort = 18196
	if _, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		InstanceID:  instance.InstanceID,
		Scenario:    "alpha",
		Variant:     "live",
		PortName:    "api",
		EnvVar:      "API_PORT",
		Port:        claimedPort,
		Status:      scenarioruntime.ClaimStatusBound,
		ListenerPID: intPtr(orphanPID),
	}); err != nil {
		store.Close()
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	listening := true
	var signaled []int
	runner := newLifecycleRunnerForTest(t, root, home, func(deps *lifecycleDeps) {
		deps.hostSession = func(context.Context, string) (hostsession.Snapshot, error) {
			return hostsession.Snapshot{BootID: "boot-test", SessionID: "session-test"}, nil
		}
		deps.readScenarioRecords = func(string, string) ([]process.Record, error) { return nil, nil }
		deps.inspectPort = func(port int) network.PortInspection {
			if port == claimedPort && listening {
				return listenerInspection(orphanPID)
			}
			return network.PortInspection{}
		}
		deps.readProcessEnv = func(pid int) (map[string]string, error) {
			if pid != orphanPID {
				return nil, fmt.Errorf("unexpected pid %d", pid)
			}
			return managedListenerEnv("alpha", "live"), nil
		}
		deps.listeningPIDs = func(port int) ([]int, error) {
			if port == claimedPort && listening {
				return []int{orphanPID}, nil
			}
			return nil, nil
		}
		deps.signalPID = func(pid int, force bool) error {
			signaled = append(signaled, pid)
			listening = false
			return nil
		}
		deps.isPIDRunning = func(int) bool { return false }
		deps.sleep = func(time.Duration) {}
	})

	if err := runner.cleanupScenarioRuntimeWithRegistryContext(ctx, "alpha", "live", "", false, true, false); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if len(signaled) != 1 || signaled[0] != orphanPID {
		t.Fatalf("signaled = %v, want [%d]", signaled, orphanPID)
	}

	verifyStore, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer verifyStore.Close()
	claims, err := verifyStore.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{InstanceID: instance.InstanceID})
	if err != nil {
		t.Fatalf("ListPortClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].Status != scenarioruntime.ClaimStatusReleased {
		t.Fatalf("claims = %#v, want released claim after reclaiming the orphan", claims)
	}
}
