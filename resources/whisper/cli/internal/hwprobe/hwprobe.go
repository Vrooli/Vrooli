// Package hwprobe is the cross-platform host-capabilities seam used by
// the whisper resource's model recommender. The Probe interface is the
// single boundary between the recommender (pure, testable) and the
// ambient world (gopsutil-equivalent stdlib reads, nvidia-smi
// subprocess). Production wires SystemProbe; tests inject FakeProbe
// from internal/hwprobe/mocks.
//
// Cross-platform strategy is stdlib-only (no gopsutil) for CGO_ENABLED=0
// portability:
//   - CPU cores: runtime.NumCPU().
//   - System RAM: /proc/meminfo on Linux, `sysctl -n hw.memsize` on
//     Darwin, `wmic ComputerSystem get TotalPhysicalMemory` on Windows.
//   - Discrete GPUs: nvidia-smi --query-gpu when present; otherwise
//     empty slice.
//   - Apple Silicon: GOOS=darwin && GOARCH=arm64 reports a single
//     "Apple GPU" with effective VRAM = 50% of system RAM (unified memory).
package hwprobe

import (
	"context"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GPU describes one detected GPU.
type GPU struct {
	Name     string
	VRAMBytes uint64
}

// HostCapabilities is the capability snapshot the recommender consumes.
// Apple Silicon synthesises a GPU entry to represent unified memory.
type HostCapabilities struct {
	OS            string
	Arch          string
	CPUCores      int
	TotalRAMBytes uint64
	GPUs          []GPU
}

// Probe is the host-capabilities seam.
//
// seam: hwprobe.Probe is the host-capability detection seam (SEAMS.md
// row "hwprobe.Probe"). Production wires SystemProbe; tests inject
// FakeProbe from internal/hwprobe/mocks.
type Probe interface {
	Detect(ctx context.Context) (HostCapabilities, error)
}

// SystemProbe is the production Probe implementation.
type SystemProbe struct {
	// RAMReader overrides system-RAM detection; nil = real reader.
	RAMReader func() (uint64, error)
	// GPUReader overrides GPU detection; nil = nvidia-smi exec.
	GPUReader func(ctx context.Context) ([]GPU, error)
	// CPUCount overrides CPU-core detection; 0 = runtime.NumCPU.
	CPUCount func() int
}

// Compile-time guarantee.
var _ Probe = (*SystemProbe)(nil)

// Detect returns the current host's capabilities. Reader-injection makes
// the implementation testable without t.Setenv or fragile build tags.
func (p *SystemProbe) Detect(ctx context.Context) (HostCapabilities, error) {
	caps := HostCapabilities{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	if p.CPUCount != nil {
		caps.CPUCores = p.CPUCount()
	} else {
		caps.CPUCores = runtime.NumCPU()
	}

	ramReader := p.RAMReader
	if ramReader == nil {
		ramReader = readSystemRAM
	}
	ram, ramErr := ramReader()
	if ramErr == nil {
		caps.TotalRAMBytes = ram
	}

	gpuReader := p.GPUReader
	if gpuReader == nil {
		gpuReader = readNvidiaGPUs
	}
	gpus, _ := gpuReader(ctx)
	// Apple Silicon: model unified memory as a synthetic GPU.
	if caps.OS == "darwin" && caps.Arch == "arm64" {
		half := caps.TotalRAMBytes / 2
		gpus = append(gpus, GPU{Name: "Apple GPU (unified)", VRAMBytes: half})
	}
	caps.GPUs = gpus
	return caps, nil
}

// readNvidiaGPUs invokes `nvidia-smi --query-gpu=name,memory.total
// --format=csv,noheader,nounits` and parses each line. Returns
// (nil, nil) when nvidia-smi is absent or fails — "no GPU" is the
// expected steady-state on most hosts and must not surface as an error.
func readNvidiaGPUs(ctx context.Context) ([]GPU, error) {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil, nil
	}
	cmd := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=name,memory.total",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	return parseNvidiaSmiCSV(string(out)), nil
}

// parseNvidiaSmiCSV converts `Name, MB` rows into []GPU. Garbage lines
// are silently dropped — operators see them in the explain output as a
// reduced GPU count rather than a hard failure.
func parseNvidiaSmiCSV(s string) []GPU {
	var out []GPU
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		mbStr := strings.TrimSpace(parts[1])
		mb, err := strconv.ParseUint(mbStr, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, GPU{Name: name, VRAMBytes: mb * 1024 * 1024})
	}
	return out
}

// readSystemRAM is the per-platform RAM probe. The implementation lives
// in hwprobe_<os>.go behind a build tag so each branch is compiled only
// on its target OS — no shared dispatcher avoids referencing functions
// that aren't built on the current platform.
