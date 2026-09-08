// Package health samples the node-agent's self-reported readiness — the
// HealthSnapshot it sends on connect and on every heartbeat (channel.proto).
// Dispatch only targets nodes whose snapshot shows they can do the work
// (OT-P0-003), so the sampler reports honestly: whether the `vrooli` toolchain
// is runnable, how much disk headroom remains on the work volume, and whether a
// container runtime is up. The Sampler interface is the seam — production uses
// SystemSampler; tests substitute a fixed snapshot so presence transitions are
// deterministic (no real disk/PATH dependence).
//
// Everything here is CGO_ENABLED=0-buildable across the full cross-compile
// matrix: disk headroom is read through build-tagged syscalls
// (disk_unix.go / disk_windows.go), and toolchain/container detection is a
// pure exec.LookPath probe.
package health

import (
	"context"
	"math"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

// Snapshot mirrors channel.HealthSnapshot in the agent's own vocabulary so the
// sampler never imports proto types. channel.snapshotToProto translates it.
type Snapshot struct {
	ToolchainPresent   bool
	DiskHeadroomBytes  int64
	ContainerRuntimeUp bool
	Details            map[string]string
	ReportedAt         time.Time
	Capabilities       []capabilityprobe.Observation
}

// Sampler produces a readiness Snapshot. It is the test seam: the live dial
// loop holds a Sampler, and tests inject a Fixed one.
type Sampler interface {
	Sample() Snapshot
}

// SystemSampler reads real readiness from the host. WorkDir is the volume whose
// free space is reported as disk headroom; an empty WorkDir measures the
// current directory.
type SystemSampler struct {
	WorkDir                   string
	Now                       func() time.Time
	CapabilityRefreshInterval time.Duration
	mu                        sync.RWMutex
	capabilities              []capabilityprobe.Observation
	capabilitiesAt            time.Time
	toolchainPresent          bool
	containerRuntimeUp        bool
}

// NewSystemSampler constructs a SystemSampler measuring headroom on workDir
// (the agent passes its state dir, which lives on the work volume).
func NewSystemSampler(workDir string) *SystemSampler {
	s := &SystemSampler{WorkDir: workDir, Now: time.Now, CapabilityRefreshInterval: 10 * time.Minute}
	go s.refreshLoop()
	return s
}

func (s *SystemSampler) refreshLoop() {
	interval := s.CapabilityRefreshInterval
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	s.refresh()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.refresh()
	}
}

func (s *SystemSampler) refresh() {
	capabilities := capabilityprobe.Probe(context.Background(), capabilityprobe.AITools)
	now := time.Now().UTC()
	s.mu.Lock()
	s.capabilities = append([]capabilityprobe.Observation(nil), capabilities...)
	s.capabilitiesAt = now
	s.toolchainPresent = binExists("vrooli")
	s.containerRuntimeUp = binExists("docker") || binExists("podman")
	s.mu.Unlock()
}

// Sample probes the host once. It never errors — a probe that fails degrades to
// "not ready" for that signal (recorded in Details) rather than dropping the
// whole heartbeat, so a node with, say, an unreadable work volume still reports
// liveness.
func (s *SystemSampler) Sample() Snapshot {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	dir := s.WorkDir
	if dir == "" {
		dir = "."
	}

	details := map[string]string{"go": runtime.Version()}

	free, err := diskFreeBytes(dir)
	if err != nil {
		details["disk_error"] = err.Error()
		free = 0
	}

	s.mu.RLock()
	toolchainPresent := s.toolchainPresent
	containerRuntimeUp := s.containerRuntimeUp
	capabilities := append([]capabilityprobe.Observation(nil), s.capabilities...)
	s.mu.RUnlock()
	return Snapshot{
		ToolchainPresent:   toolchainPresent,
		DiskHeadroomBytes:  clampToInt64(free),
		ContainerRuntimeUp: containerRuntimeUp,
		Details:            details,
		ReportedAt:         now().UTC(),
		Capabilities:       capabilities,
	}
}

// Fixed is a deterministic Sampler for tests.
type Fixed struct {
	Snap Snapshot
}

// Sample returns the fixed snapshot.
func (f Fixed) Sample() Snapshot { return f.Snap }

func binExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// clampToInt64 caps an unsigned byte count at the proto field's int64 max so a
// (practically impossible) >8 EiB volume can never wrap to a negative headroom.
func clampToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}
