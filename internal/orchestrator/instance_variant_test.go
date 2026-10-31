package orchestrator

import (
	"context"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// TestDetailSelectsAddressedVariant proves that resolving a "name@variant"
// argument through Lookup returns the addressed instance's runtime — the live
// and shadow instances of one scenario report their own distinct ports, and
// stopping/looking up one never bleeds into the other. This is the read-path
// half of the P1 instance-addressing floor exposed through `vrooli scenario
// status/port`.
func TestDetailSelectsAddressedVariant(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("Alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT", Range: "18080-18090"}}),
	))

	livePort := openTestListener(t)
	shadowPort := openTestListener(t)
	seedVariantInstance(t, home, "alpha", scenarioruntime.DefaultVariant, "API_PORT", livePort)
	seedVariantInstance(t, home, "alpha", "shadow", "API_PORT", shadowPort)

	service := New(root, home, io.Discard, io.Discard)

	liveDetail, err := service.Detail("alpha")
	if err != nil {
		t.Fatalf("Detail(alpha): %v", err)
	}
	if got := liveDetail.Details.Ports["API_PORT"]; got != livePort {
		t.Fatalf("live API_PORT = %d, want %d", got, livePort)
	}

	shadowDetail, err := service.Detail("alpha@shadow")
	if err != nil {
		t.Fatalf("Detail(alpha@shadow): %v", err)
	}
	if got := shadowDetail.Details.Ports["API_PORT"]; got != shadowPort {
		t.Fatalf("shadow API_PORT = %d, want %d", got, shadowPort)
	}

	// The two variants must not collapse into one another.
	if livePort == shadowPort {
		t.Fatal("test setup error: live and shadow share a port")
	}
	if shadowDetail.Details.Ports["API_PORT"] == liveDetail.Details.Ports["API_PORT"] {
		t.Fatal("shadow Detail leaked the live instance's port")
	}

	resolved, err := service.ResolvePort("alpha@shadow", "API_PORT")
	if err != nil {
		t.Fatalf("ResolvePort(alpha@shadow): %v", err)
	}
	if resolved.Port != shadowPort {
		t.Fatalf("ResolvePort shadow = %d, want %d", resolved.Port, shadowPort)
	}
}

func seedVariantInstance(t *testing.T, home, scenarioName, variant, envVar string, port int) {
	t.Helper()
	ctx := context.Background()
	host, err := hostsession.DefaultProvider{}.Current(ctx, home)
	if err != nil {
		t.Fatalf("host session: %v", err)
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	key := scenarioruntime.InstanceKey{Scenario: scenarioName, Variant: variant}.Normalize()
	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-" + key.Slug(),
		Scenario:   scenarioName,
		Variant:    key.Variant,
		Status:     scenarioruntime.StatusStarting,
		Phase:      "develop",
		OwnerKind:  scenarioruntime.OwnerKindLifecycle,
		StartedAt:  time.Now().Add(-time.Minute).UTC(),
		WorkingDir: filepath.Join(home, "scenarios", key.Slug()),
		HostBootID: host.BootID,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease(%s): %v", key.Slug(), err)
	}
	instance, err = store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, scenarioruntime.StatusRunning, "develop")
	if err != nil {
		t.Fatalf("UpdateInstanceStatus(%s): %v", key.Slug(), err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-" + key.Slug() + "-api",
		InstanceID: instance.InstanceID,
		Scenario:   scenarioName,
		Variant:    key.Variant,
		PortName:   "api",
		EnvVar:     envVar,
		Port:       port,
		BindHost:   "127.0.0.1",
		Status:     scenarioruntime.ClaimStatusReserved,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim(%s): %v", key.Slug(), err)
	}
	if _, err := store.BindPortClaim(ctx, claim.ClaimID); err != nil {
		t.Fatalf("BindPortClaim(%s): %v", key.Slug(), err)
	}
}
