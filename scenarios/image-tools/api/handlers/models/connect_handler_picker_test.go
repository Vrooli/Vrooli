package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"image-tools/internal/capabilities"
	internalmodels "image-tools/internal/models"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// The picker reads the model lifecycle's actionable reason without rebuilding
// environment, build and smoke decisions in the transport layer.
func TestReadyStateUsesLifecycleReason(t *testing.T) {
	backend := &modelsv1.BackendReadiness{Ready: true, InstallTier: "auto"}
	for _, reason := range []string{"ready", "env_not_provisioned", "smoke_failed", "smoke_pending", "smoke_unsupported", "build_not_ready"} {
		state := internalmodels.ModelLifecycle{Reason: reason}
		if got := readyState(true, true, "ok", backend, state); got != reason {
			t.Fatalf("%s projected as %s", reason, got)
		}
		if got := readyState(false, true, "ok", backend, state); got != "needs_model_install" {
			t.Fatalf("missing weights projected as %s", got)
		}
	}
}

func TestListOperationModelsMarksConditionalRightsAsReviewRequired(t *testing.T) {
	h, reg := newTestHandler(t, cpuOnlyHost)
	var conditional internalmodels.Model
	for _, model := range reg.Models() {
		if model.CapabilityLabels.CommercialUse == internalmodels.CommercialUseConditional && len(model.Operations) > 0 {
			conditional = model
			break
		}
	}
	if conditional.ID == "" {
		t.Fatal("seed should include a conditional-rights model for this regression")
	}
	resp, err := h.ListOperationModels(context.Background(), connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: conditional.Operations[0]}))
	if err != nil {
		t.Fatalf("list operation models: %v", err)
	}
	candidate := findCandidate(resp.Msg, conditional.ID)
	if candidate == nil {
		t.Fatalf("conditional model %q missing from its operation picker", conditional.ID)
	}
	if candidate.GetReadyState() != "commercial_rights_review_required" {
		t.Fatalf("conditional model ready state = %q, want commercial_rights_review_required", candidate.GetReadyState())
	}
}

// findCandidate returns the named model's candidate row from a picker response.
func findCandidate(resp *modelsv1.ListOperationModelsResponse, id string) *modelsv1.CandidateModel {
	for _, c := range resp.GetCandidates() {
		if c.GetModel().GetId() == id {
			return c
		}
	}
	return nil
}

// TestListOperationModels_DerivedCandidateSurfaced proves the capability-matrix
// behavior (plan Phase 6) for both halves of the no-vaporware gate using one base
// checkpoint (sd-1.5, architecture sd15, declaring only text_to_image/
// image_to_image):
//   - inpaint: a PROVEN sd15 derivation — sd-1.5 appears as a derived candidate
//     with its caveat and a real install/backend ready_state (not the unproven
//     state), so once provisioned it can be selected. It never vanishes for not
//     declaring inpaint.
//   - edit_instruct: a still-UNPROVEN sd15 derivation — sd-1.5 appears as a derived
//     candidate with an honest derived_pipeline_unproven ready_state and is never
//     offered for execution.
func TestListOperationModels_DerivedCandidateSurfaced(t *testing.T) {
	h, _ := newTestHandler(t, gpuTestHost)

	// Proven derived op: inpaint.
	respIn, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "inpaint"}))
	if err != nil {
		t.Fatalf("ListOperationModels(inpaint): %v", err)
	}
	sd15 := findCandidate(respIn.Msg, "sd-1.5")
	if sd15 == nil {
		t.Fatal("base sd-1.5 checkpoint must appear in the inpaint picker as a derived candidate")
	}
	if sd15.GetSupport() != "derived" {
		t.Fatalf("sd-1.5 inpaint support = %q, want derived", sd15.GetSupport())
	}
	if sd15.GetCaveat() == "" {
		t.Fatal("derived candidate must carry a caveat")
	}
	if sd15.GetReadyState() == "derived_pipeline_unproven" {
		t.Fatal("sd-1.5 inpaint is a PROVEN sd15 derivation; ready_state must not be derived_pipeline_unproven")
	}

	// Still-unproven derived op: edit_instruct (sd15 derives it, not yet proven).
	respEdit, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "edit_instruct"}))
	if err != nil {
		t.Fatalf("ListOperationModels(edit_instruct): %v", err)
	}
	sd15e := findCandidate(respEdit.Msg, "sd-1.5")
	if sd15e == nil {
		t.Fatal("base sd-1.5 checkpoint must appear in the edit_instruct picker as a derived candidate")
	}
	if sd15e.GetSupport() != "derived" {
		t.Fatalf("sd-1.5 edit_instruct support = %q, want derived", sd15e.GetSupport())
	}
	if sd15e.GetReadyState() != "derived_pipeline_unproven" {
		t.Fatalf("sd-1.5 edit_instruct ready_state = %q, want derived_pipeline_unproven (no faked-green)", sd15e.GetReadyState())
	}
	if sd15e.GetSelected() {
		t.Fatal("an unproven derived candidate must never be the selected model")
	}
}

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

func TestListOperationModelsDoesNotAdvertiseUnsupportedHostAsRunnable(t *testing.T) {
	h, _ := newTestHandler(t, capabilities.Host{OS: "darwin", Arch: "arm64", Cores: 8})
	resp, err := h.ListOperationModels(context.Background(),
		connect.NewRequest(&modelsv1.ListOperationModelsRequest{Operation: "deblur"}))
	if err != nil {
		t.Fatalf("ListOperationModels(deblur): %v", err)
	}
	nafnet := findCandidate(resp.Msg, "nafnet")
	if nafnet == nil {
		t.Fatal("NAFNet should remain visible in the picker with its unsupported-host reason")
	}
	if nafnet.GetFit().GetRunnable() {
		t.Fatal("NAFNet must not be advertised runnable outside its declared os_arch set")
	}
	if nafnet.GetFit().GetFitClass() != "unsupported_os" {
		t.Fatalf("NAFNet fit class = %q, want unsupported_os", nafnet.GetFit().GetFitClass())
	}
	if nafnet.GetReadyState() != "unsupported" {
		t.Fatalf("NAFNet ready state = %q, want unsupported", nafnet.GetReadyState())
	}
	if nafnet.GetSelected() || resp.Msg.GetSelectedId() != "" {
		t.Fatalf("unsupported NAFNet was selected: candidate=%t selected_id=%q", nafnet.GetSelected(), resp.Msg.GetSelectedId())
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
