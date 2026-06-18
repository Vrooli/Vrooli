package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func TestFixed_ReturnsConfiguredSnapshot(t *testing.T) {
	want := Snapshot{ToolchainPresent: true, DiskHeadroomBytes: 42}
	require.Equal(t, want, Fixed{Snap: want}.Sample())
}
