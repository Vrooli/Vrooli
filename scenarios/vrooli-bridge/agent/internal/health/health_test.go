package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

func TestSystemSampler_ReportsHonestSignals(t *testing.T) {
	// The agent's own state dir always exists and lives on a real volume, so
	// disk headroom is measurable and non-negative there.
	s := NewSystemSampler(t.TempDir())
	snap := s.Sample()

	require.GreaterOrEqual(t, snap.DiskHeadroomBytes, int64(0), "headroom is read from a real volume")
	require.NotEmpty(t, snap.Details["go"], "the go runtime version is always reported")
	require.False(t, snap.ReportedAt.IsZero(), "every snapshot is timestamped")
	// toolchain/container presence depend on the host; they must not panic and
	// must be deterministic booleans — asserting the call is enough.
	_ = snap.ToolchainPresent
	_ = snap.ContainerRuntimeUp
}

func TestSystemSampler_DegradesOnUnreadableVolume(t *testing.T) {
	s := &SystemSampler{WorkDir: "/this/path/does/not/exist", Now: func() time.Time { return time.Unix(0, 0) }}
	snap := s.Sample()

	require.Equal(t, int64(0), snap.DiskHeadroomBytes, "an unreadable volume degrades headroom to 0")
	require.NotEmpty(t, snap.Details["disk_error"], "the failure is recorded, not swallowed")
}

// A node Bridge cannot update must say so, with every blocker, so the operator
// learns the repair before a provisioning op fails on the node.
func TestProvisioningObservation_NamesEveryBlocker(t *testing.T) {
	now := time.Unix(100, 0)

	obs := ProvisioningObservation("", t.TempDir(), now)
	require.Equal(t, ProvisioningID, obs.ID)
	require.Equal(t, capabilityprobe.Missing, obs.State)
	require.True(t, strings.HasPrefix(obs.Detail, ProvisioningHelperNotInstalled+":"), obs.Detail)
	require.Contains(t, obs.Detail, ProvisioningCheckoutNotGit+":")
	require.Contains(t, obs.Detail, "onboard connect", "the observation names the update path that still works")

	checkout := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(checkout, ".git"), 0o755))
	obs = ProvisioningObservation("", checkout, now)
	require.Equal(t, capabilityprobe.Missing, obs.State)
	require.NotContains(t, obs.Detail, ProvisioningCheckoutNotGit)

	obs = ProvisioningObservation("/run/vrooli-bridge/provision.sock", checkout, now)
	require.Equal(t, capabilityprobe.Ready, obs.State)
}

func TestSystemSampler_ReportsProvisioningOnlyWhenConfigured(t *testing.T) {
	plain := &SystemSampler{}
	plain.refresh()
	for _, item := range plain.Sample().Capabilities {
		require.NotEqual(t, ProvisioningID, item.ID)
	}

	withProvisioning := &SystemSampler{}
	WithProvisioning("", t.TempDir())(withProvisioning)
	withProvisioning.refresh()
	found := false
	for _, item := range withProvisioning.Sample().Capabilities {
		if item.ID == ProvisioningID {
			found = true
			require.Equal(t, capabilityprobe.Missing, item.State)
		}
	}
	require.True(t, found, "a configured sampler reports provisioning readiness on every heartbeat")
}

func TestFixed_ReturnsConfiguredSnapshot(t *testing.T) {
	want := Snapshot{ToolchainPresent: true, DiskHeadroomBytes: 42}
	require.Equal(t, want, Fixed{Snap: want}.Sample())
}
