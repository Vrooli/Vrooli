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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/capabilityprobe"
)

// The provisioning observation is the node's own answer to "can Bridge update
// this machine with `provision sync`?". Bridge keys on ProvisioningID and on the
// reason code that prefixes a missing observation's detail; keep these strings
// in lockstep with the Bridge provision adapter and api-core targetmodel.
const (
	ProvisioningCapability = "bridge-provisioning"
	ProvisioningID         = "bridge-provisioner"
	ProvisioningLabel      = "Remote updates (Bridge provisioning)"

	// ProvisioningHelperNotInstalled — the agent has no provisioning socket, so
	// every pushed provisioning command is refused on the node.
	ProvisioningHelperNotInstalled = "helper_not_installed"
	// ProvisioningCheckoutNotGit — the checkout the helper would fetch into is
	// not a git repository (typically a tree shipped by working-tree onboarding).
	ProvisioningCheckoutNotGit = "checkout_not_git"
)

// Option customises a SystemSampler before its refresh loop starts.
type Option func(*SystemSampler)

// WithProvisioning makes the sampler report whether the privileged provisioning
// path can run: socket is the helper IPC endpoint (empty = not installed) and
// checkoutDir is the directory the helper fetches into (empty = the agent's
// working directory).
func WithProvisioning(socket, checkoutDir string) Option {
	return func(s *SystemSampler) {
		s.provisioning = &provisioningConfig{socket: strings.TrimSpace(socket), checkoutDir: strings.TrimSpace(checkoutDir)}
	}
}

type provisioningConfig struct {
	socket      string
	checkoutDir string
}

// ProvisioningObservation reports provisioning readiness as a capability
// observation. It is missing with every blocker named, so an operator sees the
// whole repair rather than discovering the second blocker after fixing the first.
func ProvisioningObservation(socket, checkoutDir string, now time.Time) capabilityprobe.Observation {
	observation := capabilityprobe.Observation{
		Capability: ProvisioningCapability, ID: ProvisioningID, Label: ProvisioningLabel,
		State: capabilityprobe.Ready, ProbedAt: now.UTC(),
		Detail: "Bridge can update this node with `vrooli-bridge provision sync`",
	}
	dir := checkoutDir
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	var blockers []string
	if socket == "" {
		blockers = append(blockers, ProvisioningHelperNotInstalled+": the privileged provisioning helper is not installed, so `vrooli-bridge provision sync` is refused on this node")
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		blockers = append(blockers, ProvisioningCheckoutNotGit+": "+dir+" is not a git checkout, so provisioning has no revision to fetch")
	}
	if len(blockers) == 0 {
		return observation
	}
	observation.State = capabilityprobe.Missing
	observation.Path = dir
	observation.Detail = strings.Join(blockers, "; ") + "; update this node by re-running `vrooli-bridge onboard connect` for it"
	return observation
}

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
	provisioning              *provisioningConfig

	// HealthRefreshInterval paces the machine-health reading (disk, memory,
	// swap). Directory sizes are re-measured at most every dirRefreshInterval.
	HealthRefreshInterval time.Duration
	// Home is the node user's home, where onboarding artifacts and Vrooli logs
	// live. Empty resolves os.UserHomeDir.
	Home          string
	health        []capabilityprobe.Observation
	artifacts     dirUsage
	logs          dirUsage
	dirsMeasured  time.Time
	readHost      func() hostMetrics
	readDiskUsage func(string) (uint64, uint64, error)
}

// dirRefreshInterval bounds how often directory trees are walked; their size
// changes slowly and a walk is the only non-constant-time part of a reading.
const dirRefreshInterval = 10 * time.Minute

// NewSystemSampler constructs a SystemSampler measuring headroom on workDir
// (the agent passes its state dir, which lives on the work volume).
func NewSystemSampler(workDir string, opts ...Option) *SystemSampler {
	s := &SystemSampler{WorkDir: workDir, Now: time.Now, CapabilityRefreshInterval: 10 * time.Minute, HealthRefreshInterval: time.Minute}
	for _, opt := range opts {
		opt(s)
	}
	go s.refreshLoop()
	go s.healthLoop()
	return s
}

func (s *SystemSampler) healthLoop() {
	interval := s.HealthRefreshInterval
	if interval <= 0 {
		interval = time.Minute
	}
	s.refreshHealth()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.refreshHealth()
	}
}

// refreshHealth takes one machine-health reading. It runs off the heartbeat
// path so a slow volume or a large directory never delays liveness.
func (s *SystemSampler) refreshHealth() {
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	dir := s.WorkDir
	if dir == "" {
		dir = "."
	}
	readDisk := s.readDiskUsage
	if readDisk == nil {
		readDisk = diskUsage
	}
	readHost := s.readHost
	if readHost == nil {
		readHost = readHostMetrics
	}
	in := HealthInputs{DiskPath: dir, Host: readHost(), Now: now}
	free, total, err := readDisk(dir)
	if err != nil {
		in.DiskErr = err.Error()
	} else {
		in.DiskFree, in.DiskTotal = free, total
	}

	s.mu.RLock()
	artifacts, logs, measured := s.artifacts, s.logs, s.dirsMeasured
	s.mu.RUnlock()
	if measured.IsZero() || now.Sub(measured) >= dirRefreshInterval {
		home := s.Home
		if home == "" {
			home, _ = os.UserHomeDir()
		}
		if home == "" {
			artifacts = dirUsage{Err: "the node user's home directory is unknown"}
			logs = artifacts
		} else {
			artifacts = measureDir(BootstrapArtifactsDir(home))
			logs = measureDir(VrooliLogsDir(home))
		}
		measured = now
	}
	in.Artifacts, in.Logs = artifacts, logs
	observations := HealthObservations(in)

	s.mu.Lock()
	s.health = observations
	s.artifacts, s.logs, s.dirsMeasured = artifacts, logs, measured
	s.mu.Unlock()
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

// CapabilityRefresher is a sampler that can re-probe its capability inventory
// on demand. The periodic probe runs every 10 minutes, so an agent installed
// through Bridge was not reported for up to 10 minutes and every remote install
// ended "unconfirmed" against Web Console's short confirmation window.
type CapabilityRefresher interface {
	RefreshCapabilities()
}

// RefreshCapabilities re-probes the capability inventory now; the next
// heartbeat carries the result.
func (s *SystemSampler) RefreshCapabilities() { s.refresh() }

func (s *SystemSampler) refresh() {
	capabilities := capabilityprobe.Probe(context.Background(), capabilityprobe.AITools)
	now := time.Now().UTC()
	if s.provisioning != nil {
		capabilities = append(capabilities, ProvisioningObservation(s.provisioning.socket, s.provisioning.checkoutDir, now))
	}
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
	capabilities = append(capabilities, s.health...)
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
