package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalbackends "image-tools/internal/backends"
	"image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// busyGPUHost is a 16 GB GPU with 14 GB currently used by other owners: total
// VRAM covers SD 1.5 (6 GB), free VRAM does not.
var busyGPUHost = capabilities.Host{
	OS: "linux", Arch: "amd64", Cores: 16, TotalMemoryBytes: 32 << 30,
	GPUs: []capabilities.GPU{{Name: "test-gpu", UUID: "GPU-busy", VRAMBytes: 16 << 30, VRAMUsedBytes: 14 << 30}},
}

// gpuTestProvider is a ready, GPU-qualified backend, so a resolution that fits
// the GPU lands on the local-gpu tier.
type gpuTestProvider struct{ testProvider }

func (gpuTestProvider) GPUCapable() bool { return true }

// Every read-only preview must judge GPU fit by the rule real jobs use. A job
// is admitted by the host capacity broker, so it fits by TOTAL VRAM; a preview
// that used free VRAM told the operator a model would run on the CPU while the
// job ran it on the GPU (or the reverse).
func TestPreviewsJudgeGPUFitByTheRuleJobsUse(t *testing.T) {
	const model = "sd-1.5"
	for _, managed := range []bool{true, false} {
		name := "free VRAM when no broker arbitrates the GPU"
		if managed {
			name = "total VRAM when the capacity broker arbitrates the GPU"
		}
		t.Run(name, func(t *testing.T) {
			h, reg := newTestHandler(t, busyGPUHost)
			h.deps.CapacityManagedGPU = managed
			seeded, _ := reg.ByID(model)
			h.deps.Backends = internalbackends.New()
			if err := h.deps.Backends.Register(gpuTestProvider{testProvider{name: seeded.Backend, ops: []string{"text_to_image"}, available: true}}); err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()

			selected, err := h.SelectModel(ctx, connect.NewRequest(&modelsv1.SelectModelRequest{Operation: "text_to_image", OverrideId: model}))
			if err != nil {
				t.Fatalf("SelectModel: %v", err)
			}
			list, err := h.ListOperationModels(ctx, connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "text_to_image"}))
			if err != nil {
				t.Fatalf("ListOperationModels: %v", err)
			}
			candidate := findCandidate(list.Msg, model)
			if candidate == nil {
				t.Fatalf("%s missing from the picker", model)
			}
			explained, err := h.ExplainResolution(ctx, connect.NewRequest(&modelsv1.ExplainResolutionRequest{Operation: "text_to_image", ModelId: model}))
			if err != nil {
				t.Fatalf("ExplainResolution: %v", err)
			}

			if got := selected.Msg.GetGpuViable(); got != managed {
				t.Fatalf("SelectModel gpu_viable = %t, want %t", got, managed)
			}
			if got := candidate.GetFit().GetGpuViable(); got != managed {
				t.Fatalf("picker fit gpu_viable = %t, want %t", got, managed)
			}
			if got := explained.Msg.GetResolution().GetGpuViable(); got != managed {
				t.Fatalf("ExplainResolution gpu_viable = %t, want %t", got, managed)
			}
			wantTier := "local-cpu"
			if managed {
				wantTier = "local-gpu"
			}
			if got := explained.Msg.GetResolution().GetTier(); got != wantTier {
				t.Fatalf("ExplainResolution tier = %s, want %s", got, wantTier)
			}
		})
	}
}

// The selector's two modes are the same rule the previews are told to use.
func TestFitCapacityManagedMatchesTheSelectorRule(t *testing.T) {
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := reg.ByID("sd-1.5")
	if !ok {
		t.Fatal("sd-1.5 missing from the seed")
	}
	if !internalmodels.FitCapacityManaged(m, busyGPUHost, true).GPUViable {
		t.Fatal("total VRAM covers the model; a capacity-managed fit must be viable")
	}
	if internalmodels.FitCapacityManaged(m, busyGPUHost, false).GPUViable || internalmodels.Fit(m, busyGPUHost).GPUViable {
		t.Fatal("free VRAM does not cover the model; the unmanaged fit must not be viable")
	}
}
