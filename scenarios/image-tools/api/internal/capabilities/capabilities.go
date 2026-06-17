// Package capabilities is image-tools' seam onto host hardware facts (OS, CPU,
// memory, GPUs/VRAM) used to pick the right model tier for an AI operation.
//
// It reads those facts through the root `vrooli` CLI via the typed client
// packages/vrooli-cli-go (Client.HostInventory), NOT by talking to
// system-monitor. The Probe interface lets the rest of image-tools depend on a
// small domain type instead of the CLI proto, and lets tests inject a
// deterministic fake so model-selection logic is never wall-clock-blocked on a
// real host probe.
package capabilities

import (
	"context"
	"fmt"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// GPU is the subset of a detected GPU image-tools needs for VRAM-fit selection.
// A VRAMBytes of 0 means the probe could not determine VRAM ("unknown
// headroom") — never "0 bytes installed". Selection must treat unknown VRAM
// conservatively rather than disqualifying the GPU.
type GPU struct {
	Index         int
	Name          string
	VRAMBytes     uint64 // 0 == unknown
	VRAMUsedBytes uint64 // 0 == unknown
	Source        string // e.g. "nvidia-smi"
}

// VRAMKnown reports whether the probe returned a usable VRAM total for this GPU.
func (g GPU) VRAMKnown() bool { return g.VRAMBytes > 0 }

// VRAMFreeBytes returns best-effort free VRAM (total-used) and whether the
// figure is trustworthy. It is only meaningful when both totals are known.
func (g GPU) VRAMFreeBytes() (free uint64, known bool) {
	if g.VRAMBytes == 0 {
		return 0, false
	}
	if g.VRAMUsedBytes >= g.VRAMBytes {
		return 0, true
	}
	return g.VRAMBytes - g.VRAMUsedBytes, true
}

// Host is the hardware snapshot used by the model selector.
type Host struct {
	OS                   string // GOOS: "linux", "darwin", "windows"
	Arch                 string // GOARCH: "amd64", "arm64"
	Cores                int    // logical CPU cores; 0 == unknown
	TotalMemoryBytes     uint64
	AvailableMemoryBytes uint64
	GPUs                 []GPU
}

// HasGPU reports whether at least one GPU was detected.
func (h Host) HasGPU() bool { return len(h.GPUs) > 0 }

// MaxVRAMBytes returns the largest known VRAM total across detected GPUs and
// whether any GPU reported a usable figure. When known is false the caller must
// fall back to a CPU-capable model regardless of whether GPUs were detected.
func (h Host) MaxVRAMBytes() (max uint64, known bool) {
	for _, g := range h.GPUs {
		if g.VRAMKnown() && g.VRAMBytes > max {
			max, known = g.VRAMBytes, true
		}
	}
	return max, known
}

// Probe reports host hardware facts for AI-tier model selection.
type Probe interface {
	Inventory(ctx context.Context) (Host, error)
}

// CLIProbe is the production Probe: it shells the root `vrooli host inventory`
// command through the typed vrooli-cli-go client.
type CLIProbe struct {
	client *vroolicli.Client
}

// NewCLIProbe returns a Probe backed by the default vrooli CLI client.
func NewCLIProbe(opts ...vroolicli.Option) *CLIProbe {
	return &CLIProbe{client: vroolicli.New(opts...)}
}

// Inventory implements Probe by decoding the typed host-inventory contract into
// the image-tools domain Host.
func (p *CLIProbe) Inventory(ctx context.Context) (Host, error) {
	resp, err := p.client.HostInventory(ctx)
	if err != nil {
		return Host{}, fmt.Errorf("capabilities: host inventory: %w", err)
	}
	return hostFromProto(resp), nil
}

func hostFromProto(resp *cliv1.HostInventoryResponse) Host {
	if resp == nil {
		return Host{}
	}
	h := Host{
		OS:    resp.GetOs(),
		Arch:  resp.GetArch(),
		Cores: int(resp.GetCpu().GetCores()),
	}
	if mem := resp.GetMemory(); mem != nil {
		h.TotalMemoryBytes = mem.GetTotalBytes()
		h.AvailableMemoryBytes = mem.GetAvailableBytes()
	}
	for _, g := range resp.GetGpus() {
		h.GPUs = append(h.GPUs, GPU{
			Index:         int(g.GetIndex()),
			Name:          g.GetName(),
			VRAMBytes:     g.GetVramBytes(),
			VRAMUsedBytes: g.GetVramUsedBytes(),
			Source:        g.GetSource(),
		})
	}
	return h
}

// FakeProbe is a deterministic Probe for tests. Set Host to the snapshot to
// return, or Err to simulate a probe failure.
type FakeProbe struct {
	Host Host
	Err  error
}

// Inventory implements Probe.
func (f FakeProbe) Inventory(context.Context) (Host, error) {
	if f.Err != nil {
		return Host{}, f.Err
	}
	return f.Host, nil
}

// ensure both concrete types satisfy the interface at compile time.
var (
	_ Probe = (*CLIProbe)(nil)
	_ Probe = FakeProbe{}
)
