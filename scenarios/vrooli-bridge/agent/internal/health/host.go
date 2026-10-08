package health

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

// Machine health rides the capability inventory the agent already reports on
// every heartbeat, so Bridge stores it and Web Console renders it without a new
// wire field. Every observation is under the node-health capability with an id
// prefixed "node-health."; Web Console keys on that prefix to show a Health
// section instead of treating these as installable features. A healthy reading
// is Ready with the measurement as its detail; an unhealthy one is Missing with
// a stable code, the measurement, and the remedy ("<code>: <what>; <fix>").
const (
	HealthCapability = "node-health"
	HealthIDPrefix   = HealthCapability + "."

	HealthDiskID               = HealthIDPrefix + "disk"
	HealthMemoryID             = HealthIDPrefix + "memory"
	HealthSwapID               = HealthIDPrefix + "swap"
	HealthBootstrapArtifactsID = HealthIDPrefix + "bootstrap-artifacts"
	HealthLogsID               = HealthIDPrefix + "logs"

	HealthDiskLow                 = "disk_low"
	HealthDiskCritical            = "disk_critical"
	HealthMemoryPressure          = "memory_pressure"
	HealthMemoryCritical          = "memory_critical"
	HealthSwapHigh                = "swap_high"
	HealthBootstrapArtifactsLarge = "bootstrap_artifacts_large"
	HealthLogsLarge               = "logs_large"
)

// Thresholds. They are deliberately coarse: the point is to warn an operator
// before a node stops working, not to replace a metrics stack.
const (
	gib = 1 << 30

	diskLowBytes      = 10 * gib
	diskLowPercent    = 5.0
	diskCriticalBytes = 2 * gib
	diskCritPercent   = 2.0

	memoryPressurePercent = 10.0
	memoryCriticalPercent = 5.0
	memoryPSIPressure     = 10.0

	swapHighPercent    = 75.0
	swapHighOfRAMRatio = 0.5

	bootstrapArtifactsMaxDirs  = 3
	bootstrapArtifactsMaxBytes = 2 * gib

	logsMaxBytes     = 5 * gib
	logsMaxFileBytes = 1 * gib

	// dirWalkLimit bounds the size walk so a pathological tree cannot turn a
	// heartbeat refresh into minutes of I/O; a truncated walk reports "at least".
	dirWalkLimit = 50000
)

// Memory pressure levels as the darwin kernel reports them
// (kern.memorystatus_vm_pressure_level). Linux readers map onto the same scale.
const (
	pressureUnknown  = 0
	pressureNormal   = 1
	pressureWarn     = 2
	pressureCritical = 4
)

// hostMetrics is one reading of the machine's memory. Known fields are false
// when the platform has no cheap, reliable source; the observation then reads
// Unknown rather than guessing.
type hostMetrics struct {
	MemoryKnown   bool
	MemTotal      uint64
	MemAvailable  uint64 // 0 when the platform reports pressure but not availability
	PressureLevel int
	PSISomeAvg60  float64 // linux /proc/pressure/memory; <0 when unavailable
	SwapKnown     bool
	SwapTotal     uint64
	SwapUsed      uint64
	// SwapGrowsOnDemand is true where the OS adds swap files as needed
	// (macOS), so a nearly full swap total is not itself a risk.
	SwapGrowsOnDemand bool
	ReadError         string
	SwapReadError     string
}

// dirUsage is the bounded size of a directory tree.
type dirUsage struct {
	Exists       bool
	Bytes        uint64
	TopLevelDirs int
	LargestFile  string
	LargestBytes uint64
	Truncated    bool
	Err          string
}

// measureDir walks root, summing regular-file sizes, counting its immediate
// subdirectories and remembering the largest file. It never follows symlinks
// and stops after dirWalkLimit entries.
func measureDir(root string) dirUsage {
	usage := dirUsage{}
	info, err := os.Stat(root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			usage.Err = err.Error()
		}
		return usage
	}
	if !info.IsDir() {
		usage.Err = root + " is not a directory"
		return usage
	}
	usage.Exists = true
	seen := 0
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// An unreadable subtree is skipped; the total is then a lower bound.
			if entry != nil && entry.IsDir() && path != root {
				return fs.SkipDir
			}
			return nil
		}
		seen++
		if seen > dirWalkLimit {
			usage.Truncated = true
			return fs.SkipAll
		}
		if entry.IsDir() {
			if path != root && filepath.Dir(path) == root {
				usage.TopLevelDirs++
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return nil
		}
		size := uint64(fileInfo.Size()) // #nosec G115 -- regular-file sizes are non-negative.
		usage.Bytes += size
		if size > usage.LargestBytes {
			usage.LargestBytes = size
			usage.LargestFile = path
		}
		return nil
	})
	return usage
}

// BootstrapArtifactsDir is where Bridge onboarding stages prebuilt binaries on
// a node (one directory per onboarding run). It mirrors the path the
// onboarding SSH driver creates.
func BootstrapArtifactsDir(home string) string {
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "Vrooli", "Bridge", "bootstrap")
		}
		return filepath.Join(home, "AppData", "Local", "Vrooli", "Bridge", "bootstrap")
	}
	return filepath.Join(home, ".local", "lib", "vrooli-bridge", "bootstrap")
}

// VrooliLogsDir is the node's Vrooli runtime log directory.
func VrooliLogsDir(home string) string { return filepath.Join(home, ".vrooli", "logs") }

// HealthInputs is everything the classifier needs, gathered by the sampler.
type HealthInputs struct {
	DiskPath  string
	DiskFree  uint64
	DiskTotal uint64
	DiskErr   string
	Host      hostMetrics
	Artifacts dirUsage
	Logs      dirUsage
	Now       time.Time
}

// HealthObservations classifies one reading into the five health
// observations. It is pure so thresholds are tested without a real host.
func HealthObservations(in HealthInputs) []capabilityprobe.Observation {
	now := in.Now.UTC()
	return []capabilityprobe.Observation{
		diskObservation(in, now),
		memoryObservation(in.Host, now),
		swapObservation(in.Host, now),
		bootstrapArtifactsObservation(in.Artifacts, now),
		logsObservation(in.Logs, now),
	}
}

func healthObservation(id, label string, now time.Time) capabilityprobe.Observation {
	return capabilityprobe.Observation{Capability: HealthCapability, ID: id, Label: label, State: capabilityprobe.Ready, ProbedAt: now}
}

func unhealthy(observation capabilityprobe.Observation, code, what, fix string) capabilityprobe.Observation {
	observation.State = capabilityprobe.Missing
	observation.Detail = code + ": " + what + "; " + fix
	return observation
}

func unknown(observation capabilityprobe.Observation, why string) capabilityprobe.Observation {
	observation.State = capabilityprobe.Unknown
	observation.Detail = why
	return observation
}

func diskObservation(in HealthInputs, now time.Time) capabilityprobe.Observation {
	obs := healthObservation(HealthDiskID, "Disk space", now)
	obs.Path = in.DiskPath
	if in.DiskErr != "" || in.DiskTotal == 0 {
		reason := in.DiskErr
		if reason == "" {
			reason = "the volume reported no size"
		}
		return unknown(obs, "could not read free space on "+in.DiskPath+": "+reason)
	}
	percent := float64(in.DiskFree) / float64(in.DiskTotal) * 100
	what := fmt.Sprintf("%s free of %s (%.0f%%)", formatBytes(in.DiskFree), formatBytes(in.DiskTotal), percent)
	fix := "free space on this machine (old onboarding folders, logs, caches) before builds and databases start failing"
	switch {
	case in.DiskFree < diskCriticalBytes || percent < diskCritPercent:
		return unhealthy(obs, HealthDiskCritical, what, fix)
	case in.DiskFree < diskLowBytes || percent < diskLowPercent:
		return unhealthy(obs, HealthDiskLow, what, fix)
	}
	obs.Detail = what
	return obs
}

func memoryObservation(m hostMetrics, now time.Time) capabilityprobe.Observation {
	obs := healthObservation(HealthMemoryID, "Memory", now)
	if !m.MemoryKnown {
		reason := m.ReadError
		if reason == "" {
			reason = "this platform has no memory reading"
		}
		return unknown(obs, reason)
	}
	parts := []string{}
	percent := -1.0
	if m.MemTotal > 0 && m.MemAvailable > 0 {
		percent = float64(m.MemAvailable) / float64(m.MemTotal) * 100
		parts = append(parts, fmt.Sprintf("%s available of %s (%.0f%%)", formatBytes(m.MemAvailable), formatBytes(m.MemTotal), percent))
	} else if m.MemTotal > 0 {
		parts = append(parts, formatBytes(m.MemTotal)+" installed")
	}
	switch m.PressureLevel {
	case pressureNormal:
		parts = append(parts, "pressure normal")
	case pressureWarn:
		parts = append(parts, "pressure elevated")
	case pressureCritical:
		parts = append(parts, "pressure critical")
	}
	if m.PSISomeAvg60 >= 0 {
		parts = append(parts, fmt.Sprintf("stalled %.0f%% of the last minute", m.PSISomeAvg60))
	}
	what := strings.Join(parts, ", ")
	fix := "stop services this machine does not need, or give it more memory; heavy memory pressure makes every service slow and can kill processes"
	switch {
	case m.PressureLevel == pressureCritical || (percent >= 0 && percent < memoryCriticalPercent):
		return unhealthy(obs, HealthMemoryCritical, what, fix)
	case m.PressureLevel == pressureWarn || (percent >= 0 && percent < memoryPressurePercent) || m.PSISomeAvg60 >= memoryPSIPressure:
		return unhealthy(obs, HealthMemoryPressure, what, fix)
	}
	obs.Detail = what
	return obs
}

func swapObservation(m hostMetrics, now time.Time) capabilityprobe.Observation {
	obs := healthObservation(HealthSwapID, "Swap", now)
	if !m.SwapKnown {
		reason := m.SwapReadError
		if reason == "" {
			reason = "this platform has no swap reading"
		}
		return unknown(obs, reason)
	}
	if m.SwapTotal == 0 {
		obs.Detail = "no swap in use"
		return obs
	}
	percent := float64(m.SwapUsed) / float64(m.SwapTotal) * 100
	what := fmt.Sprintf("%s of %s swap in use (%.0f%%)", formatBytes(m.SwapUsed), formatBytes(m.SwapTotal), percent)
	// Swapped-out cold pages are harmless. Swap is a problem when a fixed swap
	// area is nearly full (the next allocation can kill a process), or when a
	// large share of memory lives in swap while the machine is short of memory.
	nearlyFull := !m.SwapGrowsOnDemand && percent >= swapHighPercent
	ofRAM := m.MemTotal > 0 && float64(m.SwapUsed) >= float64(m.MemTotal)*swapHighOfRAMRatio
	if nearlyFull || (ofRAM && underMemoryPressure(m)) {
		return unhealthy(obs, HealthSwapHigh, what, "this machine is short of memory and paging to disk; stop services it does not need or give it more memory")
	}
	obs.Detail = what
	return obs
}

// underMemoryPressure reports whether the memory reading shows the machine is
// short of memory, by the same thresholds the memory observation uses.
func underMemoryPressure(m hostMetrics) bool {
	if !m.MemoryKnown {
		return false
	}
	if m.PressureLevel >= pressureWarn || m.PSISomeAvg60 >= memoryPSIPressure {
		return true
	}
	return m.MemTotal > 0 && m.MemAvailable > 0 && float64(m.MemAvailable)/float64(m.MemTotal)*100 < memoryPressurePercent
}

func bootstrapArtifactsObservation(u dirUsage, now time.Time) capabilityprobe.Observation {
	obs := healthObservation(HealthBootstrapArtifactsID, "Onboarding leftovers", now)
	if u.Err != "" {
		return unknown(obs, "could not measure the onboarding artifact folder: "+u.Err)
	}
	if !u.Exists {
		obs.Detail = "no onboarding artifacts are stored"
		return obs
	}
	what := fmt.Sprintf("%d onboarding artifact folders use %s%s", u.TopLevelDirs, atLeast(u.Truncated), formatBytes(u.Bytes))
	if u.TopLevelDirs > bootstrapArtifactsMaxDirs || u.Bytes > bootstrapArtifactsMaxBytes {
		return unhealthy(obs, HealthBootstrapArtifactsLarge, what, "only the newest folder is in use; the rest are left over from earlier onboarding runs and can be removed")
	}
	obs.Detail = what
	return obs
}

func logsObservation(u dirUsage, now time.Time) capabilityprobe.Observation {
	obs := healthObservation(HealthLogsID, "Vrooli logs", now)
	if u.Err != "" {
		return unknown(obs, "could not measure the Vrooli log folder: "+u.Err)
	}
	if !u.Exists {
		obs.Detail = "no Vrooli logs are stored"
		return obs
	}
	what := fmt.Sprintf("logs use %s%s", atLeast(u.Truncated), formatBytes(u.Bytes))
	if u.LargestFile != "" {
		what += fmt.Sprintf("; largest is %s (%s)", filepath.Base(u.LargestFile), formatBytes(u.LargestBytes))
	}
	if u.Bytes > logsMaxBytes || u.LargestBytes > logsMaxFileBytes {
		return unhealthy(obs, HealthLogsLarge, what, "a service is writing more log than expected; check it for a crash loop, then trim old logs")
	}
	obs.Detail = what
	return obs
}

func atLeast(truncated bool) string {
	if truncated {
		return "at least "
	}
	return ""
}

func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	i := -1
	for value >= unit && i < len(suffixes)-1 {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", value, suffixes[i])
}
