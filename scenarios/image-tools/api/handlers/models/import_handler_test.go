package models

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	internalhfmeta "image-tools/internal/hfmeta"
	internalmodels "image-tools/internal/models"

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
		Source:                 "stabilityai/stable-diffusion-xl-base-1.0",
		Id:                     "imported-sdxl",
		Architecture:           "sdxl",
		AttestCommercialRights: true,
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

// TestImportModelNeedsNoRightsAttestationForLocalUse proves licensing never blocks
// normal local use: an unattested import registers, installs, is selectable as
// untested, and keeps a conditional label that refuses hosted use.
func TestImportModelNeedsNoRightsAttestationForLocalUse(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	h.deps.Hfmeta = fakeHfmeta{meta: sdxlRepoMeta()}

	resp, err := h.ImportModel(context.Background(), connect.NewRequest(&modelsv1.ImportModelRequest{
		Source:       "stabilityai/stable-diffusion-xl-base-1.0",
		Id:           "unattested-sdxl",
		Architecture: "sdxl",
	}))
	if err != nil {
		t.Fatalf("an unattested import must register for local use: %v", err)
	}
	if sub.submitted != 1 {
		t.Fatalf("expected the install job to be submitted, got %d", sub.submitted)
	}
	m, err := installer.Resolve(context.Background(), "unattested-sdxl")
	if err != nil {
		t.Fatalf("unattested model was not registered: %v", err)
	}
	if m.CapabilityLabels.CommercialUse != internalmodels.CommercialUseConditional {
		t.Fatalf("commercial use = %q, want conditional", m.CapabilityLabels.CommercialUse)
	}
	if err := internalmodels.RequireLocalUsePermitted(context.Background(), nil, m); err != nil {
		t.Fatalf("local use refused: %v", err)
	}
	if err := internalmodels.RequireUsePermitted(context.Background(), nil, m, internalmodels.RightsUseHostedInference); err == nil {
		t.Fatal("hosted use of an unattested import must stay refused")
	}
	// It is graded untested until evidence arrives, never missing a verdict.
	if g := resp.Msg.GetModel().GetVerdict().GetGrade(); g != modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED {
		t.Fatalf("verdict = %s, want untested", g)
	}
}

// TestInspectModelSourceReportsTheAdapterAndWhatIsMissing proves the dry run says,
// in plain words, which adapter a diffusers repo resolved to and what still stands
// between an unrunnable source and a working model.
func TestInspectModelSourceReportsTheAdapterAndWhatIsMissing(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	repo := func(class string) internalhfmeta.Metadata {
		return internalhfmeta.Metadata{
			Source: "acme/m", RepoID: "acme/m", Revision: "abc", Layout: internalhfmeta.LayoutDiffusersRepo,
			PipelineClass: class, Files: []internalhfmeta.FileInfo{{Path: "model_index.json"}, {Path: "transformer/a.safetensors"}},
		}
	}
	h.deps.Hfmeta = fakeHfmeta{meta: repo("Flux2KleinPipeline")}
	known, err := h.InspectModelSource(context.Background(), connect.NewRequest(&modelsv1.InspectModelSourceRequest{Source: "acme/m"}))
	if err != nil {
		t.Fatal(err)
	}
	if known.Msg.RuntimeFamily != "flux-2-klein" || len(known.Msg.Needs) != 0 || !contains(known.Msg.OfferedOperations, "text_to_image") {
		t.Fatalf("family=%q needs=%v ops=%v", known.Msg.RuntimeFamily, known.Msg.Needs, known.Msg.OfferedOperations)
	}

	h.deps.Hfmeta = fakeHfmeta{meta: repo("FluxNextPipeline")}
	unknown, err := h.InspectModelSource(context.Background(), connect.NewRequest(&modelsv1.InspectModelSourceRequest{Source: "acme/m"}))
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Msg.RuntimeFamily != "" || len(unknown.Msg.Needs) != 1 || !strings.Contains(unknown.Msg.Needs[0], "families.go") {
		t.Fatalf("family=%q needs=%v", unknown.Msg.RuntimeFamily, unknown.Msg.Needs)
	}
	if _, err := h.ImportModel(context.Background(), connect.NewRequest(&modelsv1.ImportModelRequest{Source: "acme/m", Id: "imported-next"})); connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "families.go") {
		t.Fatalf("import with no adapter = %v, want invalid_argument naming the missing piece", err)
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

// TestAddCustomModelNeedsOnlyIDOperationBackendAndSource is the ONNX path: the CLI
// sends no licence, hardware or asset, and the model still registers with an
// asset derived from the URL, a recorded unverified licence, and a verdict of
// untested.
func TestAddCustomModelNeedsOnlyIDOperationBackendAndSource(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	resp, err := h.AddCustomModel(context.Background(), connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model:       &modelsv1.Model{Id: "my-depth", Operations: []string{"depth_map"}, Backend: "onnxruntime"},
		DownloadUrl: "https://example.test/files/depth_v3.onnx",
	}))
	if err != nil {
		t.Fatalf("add-custom with only the shortest facts: %v", err)
	}
	m := resp.Msg.GetModel()
	if m.GetVerdict().GetGrade() != modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED || m.GetCapabilityLabels().GetLicense() != "unverified" {
		t.Fatalf("verdict=%s licence=%q", m.GetVerdict().GetGrade(), m.GetCapabilityLabels().GetLicense())
	}
	stored, err := installer.Resolve(context.Background(), "my-depth")
	if err != nil {
		t.Fatal(err)
	}
	if !stored.HasFetchStrategy() || len(stored.Source.Assets) != 1 || stored.Source.Assets[0].Kind != internalmodels.ArtifactONNX {
		t.Fatalf("source = %+v", stored.Source)
	}
}
