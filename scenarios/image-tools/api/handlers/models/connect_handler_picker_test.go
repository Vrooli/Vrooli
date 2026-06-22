package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"image-tools/internal/capabilities"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// TestListOperationModels checks the picker data source: a GPU host's summary is
// reported, every candidate carries a fit class + ready_state, and a weightless
// builtin op (naturalize) with an available backend reads as "ready".
// gpuTestHost is a 16 GB GPU host with all VRAM free (deterministic fit).
var gpuTestHost = capabilities.Host{
	OS:               "linux",
	Arch:             "amd64",
	Cores:            16,
	TotalMemoryBytes: 32 << 30,
	GPUs:             []capabilities.GPU{{Name: "test-gpu", VRAMBytes: 16 << 30}},
}

func TestListOperationModels(t *testing.T) {
	h, _ := newTestHandler(t, gpuTestHost)

	resp, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "naturalize"}))
	if err != nil {
		t.Fatalf("ListOperationModels: %v", err)
	}
	msg := resp.Msg

	if !msg.GetHost().GetHasGpu() || msg.GetHost().GetVramTotalGb() == 0 {
		t.Fatalf("host summary should reflect the GPU host: %+v", msg.GetHost())
	}
	if len(msg.GetCandidates()) == 0 {
		t.Fatal("expected at least one candidate for naturalize")
	}
	if msg.GetSelectedId() == "" {
		t.Fatal("expected a selected model for a runnable builtin op")
	}

	var builtin *modelsv1.CandidateModel
	for _, c := range msg.GetCandidates() {
		if c.GetFit().GetFitClass() == "" {
			t.Fatalf("candidate %q missing fit class", c.GetModel().GetId())
		}
		if c.GetModel().GetBackend() == "builtin" {
			builtin = c
		}
	}
	if builtin == nil {
		t.Fatal("expected a builtin-backed naturalize candidate")
	}
	if builtin.GetReadyState() != "ready" {
		t.Fatalf("builtin naturalize ready_state = %q, want ready", builtin.GetReadyState())
	}
	if builtin.GetBackend().GetInstallTier() != "builtin" {
		t.Fatalf("builtin backend install_tier = %q, want builtin", builtin.GetBackend().GetInstallTier())
	}
}

// TestListOperationModelsNoGPUHost confirms the host summary degrades honestly
// on a CPU-only host (no GPU, CPU-capable candidates still runnable).
func TestListOperationModelsNoGPUHost(t *testing.T) {
	h, _ := newTestHandler(t, capabilities.Host{OS: "linux", Arch: "amd64", Cores: 8})

	resp, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "naturalize"}))
	if err != nil {
		t.Fatalf("ListOperationModels: %v", err)
	}
	if resp.Msg.GetHost().GetHasGpu() {
		t.Fatal("CPU-only host must not report a GPU")
	}
}

// TestListOperationModelsUnknownOperation rejects an op outside the vocabulary.
func TestListOperationModelsUnknownOperation(t *testing.T) {
	h, _ := newTestHandler(t, cpuOnlyHost)

	_, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "definitely-not-an-op"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown operation should be InvalidArgument, got %v", err)
	}
}