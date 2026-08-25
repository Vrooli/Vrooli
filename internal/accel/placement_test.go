package accel_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Feature: one placement verifier answers for every driver
//
//	As the control plane
//	I want the backend a running resource is actually on
//	So that "healthy" can stop meaning only that a process is up, and a silent
//	CPU session becomes a reported state instead of a 15-hour surprise.

// noSleep removes the retry backoff so tests do not wait.
func noSleep(context.Context, time.Duration) error { return nil }

// verifierFor builds a Verifier over a fixed snapshot and container probe.
func verifierFor(snapshot hostinventory.Snapshot, probe accel.ContainerProbe) accel.Verifier {
	return accel.Verifier{
		Facts:     accel.StaticFactSource{Inventory: snapshot},
		Container: probe,
		Attempts:  2,
		Sleep:     noSleep,
	}
}

// hostRunningOnCUDA is a snapshot where pid 4242 holds device memory.
func hostRunningOnCUDA(pid int) hostinventory.Snapshot {
	snapshot := hostWithCUDA()
	snapshot.GPUProcesses = []hostinventory.GPUProcess{
		{GPUIndex: 0, PID: pid, ProcessName: "llama-server", UsedBytes: 631242752},
	}
	return snapshot
}

// Scenario: a host process holding device memory is on its declared backend.
func TestVerifyPlacementReportsOKForAHostProcessOnItsDevice(t *testing.T) {
	// Given a managed-service resource whose pid appears in the compute rows
	verifier := verifierFor(hostRunningOnCUDA(4242), nil)

	// When its placement is verified against cuda
	placement, err := verifier.VerifyPlacement(context.Background(), "ollama", accel.HostProcess{PID: 4242, Name: "llama-server"}, accel.BackendCUDA)
	// Then it is ok, on cuda, with the device evidence in the reason
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateOK || placement.Observed != accel.BackendCUDA {
		t.Fatalf("placement = %+v, want state ok on cuda", placement)
	}
	if !strings.Contains(placement.Reason, "631242752") {
		t.Fatalf("reason = %q, want it to carry the observed device memory", placement.Reason)
	}
	if placement.Drifted() {
		t.Fatal("Drifted() = true, want false")
	}
}

// Scenario: a host process absent from the compute rows is on the CPU.
//
// This is the exact failure the audit found: ollama reported healthy while
// llama-server ran with no device at all.
func TestVerifyPlacementReportsDriftForAHostProcessOnTheCPU(t *testing.T) {
	// Given a host with a working GPU and a resource pid that holds no device
	verifier := verifierFor(hostRunningOnCUDA(9999), nil)

	// When placement is verified for a different pid
	placement, err := verifier.VerifyPlacement(context.Background(), "ollama", accel.HostProcess{PID: 4242, Name: "llama-server"}, accel.BackendCUDA)
	// Then it drifts to the cpu, and the reason says why
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateDrift {
		t.Fatalf("state = %q, want %q", placement.State, accel.StateDrift)
	}
	if placement.Observed != accel.BackendCPU {
		t.Fatalf("observed = %q, want %q", placement.Observed, accel.BackendCPU)
	}
	if !placement.Drifted() {
		t.Fatal("Drifted() = false, want true")
	}
	if !strings.Contains(placement.Reason, "running on the CPU") {
		t.Fatalf("reason = %q, want it to say the process is on the CPU", placement.Reason)
	}
}

// Scenario: a container that can open its device is on its declared backend.
func TestVerifyPlacementReportsOKForAContainerHoldingItsDevice(t *testing.T) {
	// Given a container probe that opens the device successfully
	probe := func(_ context.Context, container string, backend accel.Backend) (accel.AccessState, string) {
		if container != "vrooli-kokoro" || backend != accel.BackendCUDA {
			return accel.AccessUnknown, "unexpected probe arguments"
		}
		return accel.AccessOK, "container opened /dev/nvidiactl"
	}
	verifier := verifierFor(hostWithCUDA(), probe)

	// When the container's placement is verified
	placement, err := verifier.VerifyPlacement(context.Background(), "kokoro", accel.Container{Name: "vrooli-kokoro"}, accel.BackendCUDA)
	// Then it is ok on cuda
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateOK || placement.Observed != accel.BackendCUDA {
		t.Fatalf("placement = %+v, want state ok on cuda", placement)
	}
}

// Scenario: a container that lost its device produces the typed revoked error.
//
// This is the daemon-reload failure mode: the host GPU is healthy and the
// container's access is gone. The message and its repair command are matched on
// by runbooks, so they are asserted exactly.
func TestVerifyPlacementReportsRevokedAccessWithItsRepairCommand(t *testing.T) {
	// Given a container probe reporting the device open was refused
	probe := func(context.Context, string, accel.Backend) (accel.AccessState, string) {
		return accel.AccessRevoked, "operation not permitted"
	}
	verifier := verifierFor(hostWithCUDA(), probe)

	// When the container's placement is verified
	placement, err := verifier.VerifyPlacement(context.Background(), "kokoro", accel.Container{Name: "vrooli-kokoro"}, accel.BackendCUDA)

	// Then the typed revoked error comes back
	if !errors.Is(err, accel.ErrAccessRevoked) {
		t.Fatalf("VerifyPlacement() = %v, want ErrAccessRevoked", err)
	}
	// And the operator message is the one the drivers used to build
	want := "container \"vrooli-kokoro\" cannot open /dev/nvidiactl (operation not permitted); repair with `vrooli resource restart kokoro`"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
	// And the placement itself records the drift onto the cpu
	if placement.State != accel.StateDrift || placement.Observed != accel.BackendCPU {
		t.Fatalf("placement = %+v, want drift onto cpu", placement)
	}
}

// Scenario: an unreadable placement is unknown, never assumed ok.
func TestVerifyPlacementReportsUnknownRatherThanGuessing(t *testing.T) {
	cases := []struct {
		scenario   string
		verifier   accel.Verifier
		target     accel.PlacementTarget
		backend    accel.Backend
		wantReason string
	}{
		{
			scenario:   "Given no container probe is configured, Then a container target is unknown",
			verifier:   verifierFor(hostWithCUDA(), nil),
			target:     accel.Container{Name: "vrooli-kokoro"},
			backend:    accel.BackendCUDA,
			wantReason: "no container probe is configured",
		},
		{
			scenario:   "Given a compose service with no running container, Then it is unknown",
			verifier:   verifierFor(hostWithCUDA(), func(context.Context, string, accel.Backend) (accel.AccessState, string) { return accel.AccessOK, "" }),
			target:     accel.ComposeService{Project: "kokoro", Service: "kokoro"},
			backend:    accel.BackendCUDA,
			wantReason: "no running container",
		},
		{
			scenario:   "Given a host process with no pid, Then it is unknown",
			verifier:   verifierFor(hostWithCUDA(), nil),
			target:     accel.HostProcess{Name: "llama-server"},
			backend:    accel.BackendCUDA,
			wantReason: "no pid was supplied",
		},
		{
			scenario:   "Given a metal target on a container, Then it is unknown with a named reason",
			verifier:   verifierFor(hostWithCUDA(), func(context.Context, string, accel.Backend) (accel.AccessState, string) { return accel.AccessOK, "" }),
			target:     accel.Container{Name: "anything"},
			backend:    accel.BackendMetal,
			wantReason: "no container device probe",
		},
		{
			scenario:   "Given a rocm host process with no rocm-smi, Then it is unknown",
			verifier:   verifierFor(hostWithNoAccelerator(), nil),
			target:     accel.HostProcess{PID: 4242},
			backend:    accel.BackendROCm,
			wantReason: "is not installed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When placement is verified
			placement, err := tc.verifier.VerifyPlacement(context.Background(), "fixture", tc.target, tc.backend)
			// Then it is unknown, never ok
			if err != nil {
				t.Fatalf("VerifyPlacement() = %v, want nil", err)
			}
			if placement.State != accel.StateUnknown {
				t.Fatalf("state = %q, want %q (placement = %+v)", placement.State, accel.StateUnknown, placement)
			}
			// And the reason names what could not be read
			if !strings.Contains(placement.Reason, tc.wantReason) {
				t.Fatalf("reason = %q, want it to contain %q", placement.Reason, tc.wantReason)
			}
		})
	}
}

// Scenario: a metal target on linux is unknown with a named reason.
func TestVerifyPlacementReportsMetalUnsupportedOffDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		repocontracttest.SkipPlatform(t, "this scenario describes the non-darwin build")
	}
	// Given a host process on a linux host
	verifier := verifierFor(hostWithNoAccelerator(), nil)

	// When metal placement is verified
	placement, err := verifier.VerifyPlacement(context.Background(), "whisper", accel.HostProcess{PID: 4242}, accel.BackendMetal)
	// Then it is unknown and the reason names the platform
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateUnknown {
		t.Fatalf("state = %q, want %q", placement.State, accel.StateUnknown)
	}
	if !strings.Contains(placement.Reason, "only reachable on darwin") {
		t.Fatalf("reason = %q, want it to name the platform", placement.Reason)
	}
}

// Scenario: a cpu declaration needs no probe at all.
func TestVerifyPlacementAcceptsACPUDeclarationWithoutProbing(t *testing.T) {
	// Given a verifier with no fact source and no container probe
	verifier := accel.Verifier{}

	// When a cpu placement is verified for every target kind
	for _, target := range []accel.PlacementTarget{
		accel.HostProcess{PID: 1},
		accel.Container{Name: "anything"},
		accel.ComposeService{Project: "p", Service: "s"},
	} {
		placement, err := verifier.VerifyPlacement(context.Background(), "fixture", target, accel.BackendCPU)
		// Then it is ok without touching the host
		if err != nil {
			t.Fatalf("VerifyPlacement(%T) = %v, want nil", target, err)
		}
		if placement.State != accel.StateOK || placement.Observed != accel.BackendCPU {
			t.Fatalf("VerifyPlacement(%T) = %+v, want ok on cpu", target, placement)
		}
	}
}

// Scenario: every target kind is covered for every backend.
//
// The table is the implementation checklist: a backend or target that has no
// evidence source must say so rather than fall through to a wrong answer.
func TestVerifyPlacementCoversEveryTargetAndBackend(t *testing.T) {
	targets := map[string]accel.PlacementTarget{
		"host process":    accel.HostProcess{PID: 4242, Name: "server"},
		"container":       accel.Container{Name: "vrooli-fixture"},
		"compose service": accel.ComposeService{Project: "p", Service: "s", Container: "vrooli-fixture"},
	}
	probe := func(context.Context, string, accel.Backend) (accel.AccessState, string) {
		return accel.AccessUnknown, "probe returned nothing conclusive"
	}
	verifier := verifierFor(hostWithNoAccelerator(), probe)

	for targetName, target := range targets {
		for _, backend := range accel.AllBackends {
			t.Run(targetName+"/"+string(backend), func(t *testing.T) {
				// When placement is verified
				placement, err := verifier.VerifyPlacement(context.Background(), "fixture", target, backend)

				// Then it never panics and never returns an unexplained verdict
				if err != nil && !errors.Is(err, accel.ErrAccessRevoked) {
					t.Fatalf("VerifyPlacement() = %v, want nil or a typed revoked error", err)
				}
				if placement.State == "" {
					t.Fatal("state is empty; every combination must produce a verdict")
				}
				if placement.State != accel.StateOK && strings.TrimSpace(placement.Reason) == "" {
					t.Fatalf("state %q has an empty reason; a non-ok verdict must say why", placement.State)
				}
				if placement.Declared != backend {
					t.Fatalf("declared = %q, want %q", placement.Declared, backend)
				}
				// And the cpu backend is the only one that is ok on a host with
				// no accelerator
				if backend != accel.BackendCPU && placement.State == accel.StateOK {
					t.Fatalf("backend %q reported ok on a host with no accelerator", backend)
				}
			})
		}
	}
}

// Scenario: the verifier retries before declaring drift.
//
// A process that has just started may not have opened its device yet, so a
// single look would report drift on every cold start.
func TestVerifyPlacementRetriesBeforeDeclaringDrift(t *testing.T) {
	// Given a container probe that only succeeds on its third call
	calls := 0
	probe := func(context.Context, string, accel.Backend) (accel.AccessState, string) {
		calls++
		if calls < 3 {
			return accel.AccessUnknown, "container is still starting"
		}
		return accel.AccessOK, "container opened /dev/nvidiactl"
	}
	verifier := accel.Verifier{
		Facts:     accel.StaticFactSource{Inventory: hostWithCUDA()},
		Container: probe,
		Attempts:  3,
		Sleep:     noSleep,
	}

	// When placement is verified
	placement, err := verifier.VerifyPlacement(context.Background(), "kokoro", accel.Container{Name: "vrooli-kokoro"}, accel.BackendCUDA)
	// Then the later success wins
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateOK {
		t.Fatalf("state = %q after %d probes, want ok", placement.State, calls)
	}
	if calls != 3 {
		t.Fatalf("probe called %d times, want 3", calls)
	}
}

// Scenario: a cancelled context stops the retry loop.
func TestVerifyPlacementStopsWhenTheContextIsCancelled(t *testing.T) {
	// Given a context cancelled before the second attempt
	ctx, cancel := context.WithCancel(context.Background())
	probe := func(context.Context, string, accel.Backend) (accel.AccessState, string) {
		cancel()
		return accel.AccessUnknown, "still starting"
	}
	verifier := accel.Verifier{
		Facts:     accel.StaticFactSource{Inventory: hostWithCUDA()},
		Container: probe,
		Attempts:  3,
	}

	// When placement is verified
	_, err := verifier.VerifyPlacement(ctx, "kokoro", accel.Container{Name: "vrooli-kokoro"}, accel.BackendCUDA)

	// Then the cancellation surfaces instead of the loop running to completion
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("VerifyPlacement() = %v, want context.Canceled", err)
	}
}

// Scenario: the device-holding process is often a child of the supervised one.
//
// ollama supervises `ollama serve`, which spawns `llama-server`; only the child
// appears in the compute rows. Matching the supervised pid alone would report
// drift for a healthy, GPU-resident ollama.
func TestVerifyPlacementMatchesAChildProcessInsideTheArtifactTree(t *testing.T) {
	// Given a host where a child process holds the device
	snapshot := hostWithCUDA()
	snapshot.GPUProcesses = []hostinventory.GPUProcess{
		{GPUIndex: 0, PID: 2186513, ProcessName: "/artifacts/ollama/0.30.10/ollama_linux_amd64/lib/ollama/llama-server", UsedBytes: 631242752},
	}
	verifier := verifierFor(snapshot, nil)

	// When placement is verified for the supervised parent pid
	placement, err := verifier.VerifyPlacement(context.Background(), "ollama", accel.HostProcess{
		PID:              2186000,
		Name:             "ollama",
		ExecutablePrefix: "/artifacts/ollama",
	}, accel.BackendCUDA)
	// Then the child's device holding counts for the resource
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if placement.State != accel.StateOK || placement.Observed != accel.BackendCUDA {
		t.Fatalf("placement = %+v, want ok on cuda", placement)
	}
	// And the reason names the process that actually holds the device
	if !strings.Contains(placement.Reason, "llama-server") {
		t.Fatalf("reason = %q, want it to name the device-holding process", placement.Reason)
	}

	// And a compute process outside the resource's artifact tree does not count
	other := verifierFor(func() hostinventory.Snapshot {
		s := hostWithCUDA()
		s.GPUProcesses = []hostinventory.GPUProcess{
			{GPUIndex: 0, PID: 1997568, ProcessName: "/opt/google/chrome/chrome", UsedBytes: 91226112},
		}
		return s
	}(), nil)
	drifted, err := other.VerifyPlacement(context.Background(), "ollama", accel.HostProcess{
		PID:              2186000,
		ExecutablePrefix: "/artifacts/ollama",
	}, accel.BackendCUDA)
	if err != nil {
		t.Fatalf("VerifyPlacement() = %v, want nil", err)
	}
	if drifted.State != accel.StateDrift {
		t.Fatalf("state = %q, want drift; another process holding the device is not this resource", drifted.State)
	}
}
