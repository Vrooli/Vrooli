package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internalhfmeta "image-tools/internal/hfmeta"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// fakeHfmeta returns a recorded Metadata for any source.
type fakeHfmeta struct{ meta internalhfmeta.Metadata }

func (f fakeHfmeta) Inspect(_ context.Context, source string) (internalhfmeta.Metadata, error) {
	m := f.meta
	if m.Source == "" {
		m.Source = source
	}
	return m, nil
}

func sdxlRepoMeta() internalhfmeta.Metadata {
	return internalhfmeta.Metadata{
		Source:        "stabilityai/stable-diffusion-xl-base-1.0",
		RepoID:        "stabilityai/stable-diffusion-xl-base-1.0",
		Revision:      "462165984030d82259a11f4367a4eed129e94a7b",
		Layout:        internalhfmeta.LayoutDiffusersRepo,
		PipelineClass: "StableDiffusionXLPipeline",
		License:       "openrail++",
		Files:         []internalhfmeta.FileInfo{{Path: "model_index.json", Size: 600}, {Path: "unet/x.safetensors", Size: 5_000_000_000}},
	}
}

func TestInspectModelSource(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	h.deps.Hfmeta = fakeHfmeta{meta: sdxlRepoMeta()}

	resp, err := h.InspectModelSource(context.Background(), connect.NewRequest(&modelsv1.InspectModelSourceRequest{
		Source: "stabilityai/stable-diffusion-xl-base-1.0",
	}))
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	m := resp.Msg
	if m.GetLayout() != modelsv1.ModelLayout_MODEL_LAYOUT_DIFFUSERS_REPO {
		t.Errorf("layout = %v", m.GetLayout())
	}
	if a := m.GetArchitecture(); a.GetArchitecture() != "sdxl" || a.GetConfidence() != "high" {
		t.Errorf("arch inference = %+v", a)
	}
	if m.GetProposed() == nil || m.GetProposed().GetBackend() != "diffusers" {
		t.Errorf("proposed backend = %+v", m.GetProposed())
	}
	if !contains(m.GetOfferedOperations(), "text_to_image") || !contains(m.GetOfferedOperations(), "inpaint") {
		t.Errorf("offered ops = %v (want text_to_image + derived inpaint)", m.GetOfferedOperations())
	}
}

func TestInspectModelSourceRejectsEmpty(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	h.deps.Hfmeta = fakeHfmeta{meta: sdxlRepoMeta()}
	_, err := h.InspectModelSource(context.Background(), connect.NewRequest(&modelsv1.InspectModelSourceRequest{Source: "  "}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for empty source, got %v", err)
	}
}

func TestImportModelRegistersAndSubmitsJob(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	h.deps.Hfmeta = fakeHfmeta{meta: sdxlRepoMeta()}

	resp, err := h.ImportModel(context.Background(), connect.NewRequest(&modelsv1.ImportModelRequest{
		Source:       "stabilityai/stable-diffusion-xl-base-1.0",
		Id:           "imported-sdxl",
		Architecture: "sdxl",
	}))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if resp.Msg.GetModel().GetId() != "imported-sdxl" {
		t.Fatalf("model id = %q", resp.Msg.GetModel().GetId())
	}
	if resp.Msg.GetJobId() == "" || sub.submitted != 1 {
		t.Fatalf("expected one install job submitted, got job=%q submitted=%d", resp.Msg.GetJobId(), sub.submitted)
	}
	// The add-only entry is now resolvable as a custom model.
	if _, err := installer.Resolve(context.Background(), "imported-sdxl"); err != nil {
		t.Fatalf("imported entry not registered: %v", err)
	}
}

func TestImportModelRejectsUnresolvedArchitecture(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	// Metadata with no pipeline class / tags ⇒ inference none; no confirmed arch.
	h.deps.Hfmeta = fakeHfmeta{meta: internalhfmeta.Metadata{
		Source: "Org/Mystery", RepoID: "Org/Mystery", Layout: internalhfmeta.LayoutDiffusersRepo,
		Files: []internalhfmeta.FileInfo{{Path: "model_index.json"}},
	}}
	_, err := h.ImportModel(context.Background(), connect.NewRequest(&modelsv1.ImportModelRequest{
		Source: "Org/Mystery", Id: "imported-mystery",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid_argument for unresolved architecture, got %v", err)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
