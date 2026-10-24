package health

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

func observationByID(t *testing.T, observations []capabilityprobe.Observation, id string) capabilityprobe.Observation {
	t.Helper()
	for _, item := range observations {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no observation %q in %+v", id, observations)
	return capabilityprobe.Observation{}
}

func requireUnhealthy(t *testing.T, obs capabilityprobe.Observation, code string) {
	t.Helper()
	require.Equal(t, capabilityprobe.Missing, obs.State, obs.Detail)
	require.True(t, strings.HasPrefix(obs.Detail, code+": "), "detail leads with its stable code: %q", obs.Detail)
	require.Contains(t, obs.Detail, "; ", "an unhealthy reading names the remedy after the measurement")
}

// minimouse as measured on 2026-09-15: 170 GiB free, 8 GiB RAM, 2.8 of 4 GiB
// swap, 130 onboarding folders using 6.8 GiB, 294 MiB of logs.
func minimouseInputs() HealthInputs {
	return HealthInputs{
		DiskPath: "/Users/matthalloran8", DiskFree: 170 * gib, DiskTotal: 233 * gib,
		Host: hostMetrics{
			MemoryKnown: true, MemTotal: 8 * gib, PressureLevel: pressureNormal, PSISomeAvg60: -1,
			SwapKnown: true, SwapTotal: 4 * gib, SwapUsed: 2800 << 20,
		},
		Artifacts: dirUsage{Exists: true, TopLevelDirs: 130, Bytes: 6800 << 20},
		Logs:      dirUsage{Exists: true, Bytes: 294 << 20, LargestFile: "/Users/m/.vrooli/logs/ui-health-start-api.log", LargestBytes: 71 << 20},
		Now:       time.Unix(100, 0),
	}
}

func TestHealthObservationsClassifyAMinimouseReading(t *testing.T) {
	observations := HealthObservations(minimouseInputs())
	require.Len(t, observations, 5)
	for _, item := range observations {
		require.Equal(t, HealthCapability, item.Capability)
		require.True(t, strings.HasPrefix(item.ID, HealthIDPrefix), item.ID)
		require.NotEmpty(t, item.Label)
		require.NotEmpty(t, item.Detail, "every reading states its measurement")
	}

	disk := observationByID(t, observations, HealthDiskID)
	require.Equal(t, capabilityprobe.Ready, disk.State)
	require.Contains(t, disk.Detail, "170.0 GiB free of 233.0 GiB")

	memory := observationByID(t, observations, HealthMemoryID)
	require.Equal(t, capabilityprobe.Ready, memory.State)
	require.Contains(t, memory.Detail, "pressure normal")

	swap := observationByID(t, observations, HealthSwapID)
	require.Equal(t, capabilityprobe.Ready, swap.State, "2.7 of 4 GiB is under both swap thresholds")

	requireUnhealthy(t, observationByID(t, observations, HealthBootstrapArtifactsID), HealthBootstrapArtifactsLarge)
	require.Contains(t, observationByID(t, observations, HealthBootstrapArtifactsID).Detail, "130 onboarding artifact folders")

	logs := observationByID(t, observations, HealthLogsID)
	require.Equal(t, capabilityprobe.Ready, logs.State)
	require.Contains(t, logs.Detail, "ui-health-start-api.log")
}

func TestDiskThresholds(t *testing.T) {
	in := minimouseInputs()
	in.DiskFree = 8 * gib
	requireUnhealthy(t, diskObservation(in, in.Now), HealthDiskLow)

	in.DiskFree = 1 * gib
	requireUnhealthy(t, diskObservation(in, in.Now), HealthDiskCritical)

	// A small volume is judged by percentage as well as bytes.
	in.DiskTotal, in.DiskFree = 2000*gib, 60*gib
	requireUnhealthy(t, diskObservation(in, in.Now), HealthDiskLow)

	in.DiskErr = "permission denied"
	require.Equal(t, capabilityprobe.Unknown, diskObservation(in, in.Now).State)
}

func TestMemoryThresholds(t *testing.T) {
	now := time.Unix(1, 0)
	requireUnhealthy(t, memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 8 * gib, PressureLevel: pressureWarn, PSISomeAvg60: -1}, now), HealthMemoryPressure)
	requireUnhealthy(t, memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 8 * gib, PressureLevel: pressureCritical, PSISomeAvg60: -1}, now), HealthMemoryCritical)
	requireUnhealthy(t, memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 100 * gib, MemAvailable: 8 * gib, PSISomeAvg60: -1}, now), HealthMemoryPressure)
	requireUnhealthy(t, memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 100 * gib, MemAvailable: 4 * gib, PSISomeAvg60: -1}, now), HealthMemoryCritical)
	requireUnhealthy(t, memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 100 * gib, MemAvailable: 50 * gib, PSISomeAvg60: 25}, now), HealthMemoryPressure)

	healthy := memoryObservation(hostMetrics{MemoryKnown: true, MemTotal: 100 * gib, MemAvailable: 50 * gib, PSISomeAvg60: 0.5}, now)
	require.Equal(t, capabilityprobe.Ready, healthy.State)
	require.Contains(t, healthy.Detail, "50.0 GiB available of 100.0 GiB")

	require.Equal(t, capabilityprobe.Unknown, memoryObservation(hostMetrics{ReadError: "no reader"}, now).State)
}

func TestSwapThresholds(t *testing.T) {
	now := time.Unix(1, 0)
	// A nearly full fixed swap area is a risk on its own.
	requireUnhealthy(t, swapObservation(hostMetrics{SwapKnown: true, SwapTotal: 4 * gib, SwapUsed: 3500 << 20, MemTotal: 64 * gib}, now), HealthSwapHigh)
	// macOS grows swap on demand, so a full total alone is not a warning...
	require.Equal(t, capabilityprobe.Ready, swapObservation(hostMetrics{SwapKnown: true, SwapGrowsOnDemand: true, SwapTotal: 4 * gib, SwapUsed: 3900 << 20, MemTotal: 64 * gib, MemoryKnown: true, PressureLevel: pressureNormal, PSISomeAvg60: -1}, now).State)
	// ...but half of RAM in swap while memory is under pressure is.
	requireUnhealthy(t, swapObservation(hostMetrics{SwapKnown: true, SwapGrowsOnDemand: true, SwapTotal: 10 * gib, SwapUsed: 5 * gib, MemTotal: 8 * gib, MemoryKnown: true, PressureLevel: pressureWarn, PSISomeAvg60: -1}, now), HealthSwapHigh)
	// swarminator on 2026-09-15: 44 of 69 GiB swapped out, half of RAM free, no
	// stall. Cold pages in swap are not a problem.
	require.Equal(t, capabilityprobe.Ready, swapObservation(hostMetrics{SwapKnown: true, SwapTotal: 69 * gib, SwapUsed: 44 * gib, MemoryKnown: true, MemTotal: 61 * gib, MemAvailable: 30 * gib, PSISomeAvg60: 0}, now).State)
	require.Equal(t, "no swap in use", swapObservation(hostMetrics{SwapKnown: true}, now).Detail)
	require.Equal(t, capabilityprobe.Unknown, swapObservation(hostMetrics{}, now).State)
}

func TestArtifactAndLogThresholds(t *testing.T) {
	now := time.Unix(1, 0)
	require.Equal(t, capabilityprobe.Ready, bootstrapArtifactsObservation(dirUsage{Exists: true, TopLevelDirs: 1, Bytes: 600 << 20}, now).State)
	require.Equal(t, "no onboarding artifacts are stored", bootstrapArtifactsObservation(dirUsage{}, now).Detail)
	requireUnhealthy(t, bootstrapArtifactsObservation(dirUsage{Exists: true, TopLevelDirs: 2, Bytes: 3 * gib}, now), HealthBootstrapArtifactsLarge)

	requireUnhealthy(t, logsObservation(dirUsage{Exists: true, Bytes: 2 * gib, LargestFile: "/x/runaway.log", LargestBytes: 1500 << 20}, now), HealthLogsLarge)
	requireUnhealthy(t, logsObservation(dirUsage{Exists: true, Bytes: 6 * gib}, now), HealthLogsLarge)
	truncated := logsObservation(dirUsage{Exists: true, Bytes: gib, Truncated: true}, now)
	require.Contains(t, truncated.Detail, "at least", "a bounded walk says its total is a lower bound")
	require.Equal(t, capabilityprobe.Unknown, logsObservation(dirUsage{Err: "permission denied"}, now).State)
}

func TestMeasureDirCountsFoldersAndFindsTheLargestFile(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"artifacts-a", "artifacts-b"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, name, "nested"), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "artifacts-a", "vrooli"), make([]byte, 3000), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "artifacts-b", "nested", "agent"), make([]byte, 1000), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(root, "artifacts-a", "vrooli"), filepath.Join(root, "link")))

	usage := measureDir(root)
	require.True(t, usage.Exists)
	require.Empty(t, usage.Err)
	require.Equal(t, 2, usage.TopLevelDirs, "only immediate subdirectories are onboarding runs")
	require.Equal(t, uint64(4000), usage.Bytes, "symlinks are not followed or counted")
	require.Equal(t, uint64(3000), usage.LargestBytes)
	require.Equal(t, "vrooli", filepath.Base(usage.LargestFile))

	missing := measureDir(filepath.Join(root, "absent"))
	require.False(t, missing.Exists)
	require.Empty(t, missing.Err, "a missing folder is healthy, not an error")
}

func TestReadHostMetricsOnThisHost(t *testing.T) {
	m := readHostMetrics()
	if m.MemoryKnown {
		require.Positive(t, m.MemTotal)
	} else {
		require.NotEmpty(t, m.ReadError, "an unreadable platform says why")
	}
}

// A real reading on whatever host runs the test: every observation states a
// measurement or why it could not be taken. Run with -v to see the reading.
func TestLiveHostReadingIsClassified(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	s := &SystemSampler{WorkDir: home, Home: home}
	s.refreshHealth()
	for _, obs := range s.Sample().Capabilities {
		require.NotEmpty(t, obs.Detail, obs.ID)
		t.Logf("%-32s %-8s %s", obs.ID, obs.State, obs.Detail)
	}
}

func TestSamplerReportsHealthWithEveryHeartbeat(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(BootstrapArtifactsDir(home), "artifacts-1"), 0o700))
	s := &SystemSampler{
		WorkDir: home, Home: home, Now: func() time.Time { return time.Unix(500, 0) },
		readHost: func() hostMetrics {
			return hostMetrics{MemoryKnown: true, MemTotal: 8 * gib, PressureLevel: pressureWarn, PSISomeAvg60: -1}
		},
		readDiskUsage: func(string) (uint64, uint64, error) { return 1 * gib, 100 * gib, nil },
	}
	s.refreshHealth()
	snap := s.Sample()
	requireUnhealthy(t, observationByID(t, snap.Capabilities, HealthDiskID), HealthDiskCritical)
	requireUnhealthy(t, observationByID(t, snap.Capabilities, HealthMemoryID), HealthMemoryPressure)
	require.Contains(t, observationByID(t, snap.Capabilities, HealthBootstrapArtifactsID).Detail, "1 onboarding artifact folders")

	// Directory sizes are cached between walks; a disk error degrades only disk.
	s.readDiskUsage = func(string) (uint64, uint64, error) { return 0, 0, errors.New("stale mount") }
	s.refreshHealth()
	snap = s.Sample()
	require.Equal(t, capabilityprobe.Unknown, observationByID(t, snap.Capabilities, HealthDiskID).State)
	require.Equal(t, capabilityprobe.Ready, observationByID(t, snap.Capabilities, HealthBootstrapArtifactsID).State)
}
