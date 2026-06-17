package capabilities

import (
	"context"
	"errors"
	"testing"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func TestHostFromProto(t *testing.T) {
	resp := &cliv1.HostInventoryResponse{
		Os:   "linux",
		Arch: "amd64",
		Cpu:  &cliv1.HostCPU{Cores: 32},
		Memory: &cliv1.HostMemory{
			TotalBytes:     65435598848,
			AvailableBytes: 48691671040,
		},
		Gpus: []*cliv1.HostGPU{
			{
				Index:         0,
				Name:          "NVIDIA GeForce RTX 4070 Ti SUPER",
				VramBytes:     17171480576,
				VramUsedBytes: 13670285312,
				Source:        "nvidia-smi",
			},
		},
	}

	h := hostFromProto(resp)
	if h.OS != "linux" || h.Arch != "amd64" {
		t.Fatalf("os/arch: got %q/%q", h.OS, h.Arch)
	}
	if h.Cores != 32 {
		t.Fatalf("cores: got %d want 32", h.Cores)
	}
	if h.TotalMemoryBytes != 65435598848 || h.AvailableMemoryBytes != 48691671040 {
		t.Fatalf("memory: got %d/%d", h.TotalMemoryBytes, h.AvailableMemoryBytes)
	}
	if !h.HasGPU() {
		t.Fatal("expected a GPU")
	}
	if got := h.GPUs[0]; got.Name == "" || got.VRAMBytes != 17171480576 {
		t.Fatalf("gpu: %+v", got)
	}
}

func TestHostFromProtoNilSafe(t *testing.T) {
	h := hostFromProto(nil)
	if h.HasGPU() || h.OS != "" || h.Cores != 0 {
		t.Fatalf("nil response should yield zero Host, got %+v", h)
	}
}

func TestMaxVRAMBytes(t *testing.T) {
	tests := []struct {
		name      string
		gpus      []GPU
		wantMax   uint64
		wantKnown bool
	}{
		{"none", nil, 0, false},
		{"unknown-vram", []GPU{{VRAMBytes: 0}}, 0, false},
		{"single", []GPU{{VRAMBytes: 8 << 30}}, 8 << 30, true},
		{"max-of-many", []GPU{{VRAMBytes: 8 << 30}, {VRAMBytes: 24 << 30}, {VRAMBytes: 0}}, 24 << 30, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Host{GPUs: tt.gpus}
			gotMax, gotKnown := h.MaxVRAMBytes()
			if gotMax != tt.wantMax || gotKnown != tt.wantKnown {
				t.Fatalf("got %d/%v want %d/%v", gotMax, gotKnown, tt.wantMax, tt.wantKnown)
			}
		})
	}
}

func TestVRAMFreeBytes(t *testing.T) {
	if _, known := (GPU{VRAMBytes: 0}).VRAMFreeBytes(); known {
		t.Fatal("unknown total must not report known free")
	}
	free, known := (GPU{VRAMBytes: 10, VRAMUsedBytes: 4}).VRAMFreeBytes()
	if !known || free != 6 {
		t.Fatalf("got %d/%v want 6/true", free, known)
	}
	// Used >= total (overcommit / stale read) clamps to 0, still "known".
	if free, known := (GPU{VRAMBytes: 10, VRAMUsedBytes: 12}).VRAMFreeBytes(); !known || free != 0 {
		t.Fatalf("overcommit: got %d/%v want 0/true", free, known)
	}
}

func TestFakeProbe(t *testing.T) {
	want := Host{OS: "linux", Cores: 8}
	var p Probe = FakeProbe{Host: want}
	got, err := p.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.OS != "linux" || got.Cores != 8 {
		t.Fatalf("got %+v", got)
	}

	sentinel := errors.New("probe down")
	if _, err := (FakeProbe{Err: sentinel}).Inventory(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel error, got %v", err)
	}
}
