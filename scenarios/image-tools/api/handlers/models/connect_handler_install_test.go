package models

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	"image-tools/internal/capabilities"
	internaljobs "image-tools/internal/jobs"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/testutil/db"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// fakeSubmitter records install-job submissions without running them.
type fakeSubmitter struct {
	last      internaljobs.Spec
	submitted int
}

func (f *fakeSubmitter) Submit(_ context.Context, spec internaljobs.Spec) (internaljobs.Job, error) {
	f.submitted++
	f.last = spec
	return internaljobs.Job{ID: "job-1", Operation: spec.Operation}, nil
}

// newInstallHandler builds a handler with a full Installer (fake downloader +
// generous fake disk) and a fake job submitter.
func newInstallHandler(t *testing.T) (*connectHandler, *internalmodels.Installer, *fakeSubmitter) {
	t.Helper()
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	installer := &internalmodels.Installer{
		Root:   t.TempDir(),
		Reg:    reg,
		Custom: internalmodels.NewCustomStore(d),
		State:  internalmodels.NewInstallStore(d),
		Download: downloaderTo(func(dest string) error {
			return os.WriteFile(dest, []byte("weights"), 0o644)
		}),
		DiskAvail: func(string) (int64, error) { return int64(100) << 30, nil },
	}
	sub := &fakeSubmitter{}
	h := NewConnectHandler(Deps{
		Registry:   reg,
		Store:      internalmodels.NewStore(d),
		Probe:      capabilities.FakeProbe{Host: cpuOnlyHost},
		Installer:  installer,
		Jobs:       sub,
		OpDefaults: internalmodels.NewOpDefaultStore(d),
	})
	return h, installer, sub
}

func TestSetAndListDefaults(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	ctx := context.Background()

	// Pin a non-seed-default model that serves text_to_image (sd-1.5 serves it).
	if _, err := h.SetDefaultModel(ctx, connect.NewRequest(&modelsv1.SetDefaultModelRequest{
		Operation: "text_to_image", ModelId: "sd-1.5",
	})); err != nil {
		t.Fatalf("set default: %v", err)
	}
	resp, err := h.ListDefaults(ctx, connect.NewRequest(&modelsv1.ListDefaultsRequest{}))
	if err != nil {
		t.Fatalf("list defaults: %v", err)
	}
	var found *modelsv1.OpDefault
	for _, d := range resp.Msg.Defaults {
		if d.Operation == "text_to_image" {
			found = d
		}
	}
	if found == nil || found.ModelId != "sd-1.5" || found.Source != "override" {
		t.Fatalf("expected text_to_image override sd-1.5, got %+v", found)
	}
}

func TestSetDefaultRejectsModelNotServingOp(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	// tesseract serves ocr, not text_to_image.
	_, err := h.SetDefaultModel(context.Background(), connect.NewRequest(&modelsv1.SetDefaultModelRequest{
		Operation: "text_to_image", ModelId: "tesseract",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", connect.CodeOf(err))
	}
}

type downloaderTo func(dest string) error

func (fn downloaderTo) Download(_ context.Context, _, dest string, _ func(int64, int64)) error {
	return fn(dest)
}

func TestInstallModelSubmitsJob(t *testing.T) {
	h, _, sub := newInstallHandler(t)
	resp, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: "sd-1.5"}))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if resp.Msg.AlreadyInstalled {
		t.Fatal("fresh model should not report already installed")
	}
	if resp.Msg.JobId == "" {
		t.Fatal("expected a job id")
	}
	if sub.submitted != 1 || sub.last.Operation != internalmodels.InstallJobOperation {
		t.Fatalf("expected one install-job submit, got %d op=%q", sub.submitted, sub.last.Operation)
	}
}

func TestInstallModelAlreadyInstalledSkipsJob(t *testing.T) {
	h, installer, sub := newInstallHandler(t)
	ctx := context.Background()
	if _, err := installer.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("pre-install: %v", err)
	}
	resp, err := h.InstallModel(ctx, connect.NewRequest(&modelsv1.InstallModelRequest{Id: "sd-1.5"}))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !resp.Msg.AlreadyInstalled {
		t.Fatal("expected already_installed=true")
	}
	if sub.submitted != 0 {
		t.Fatalf("no job should be submitted for an installed model, got %d", sub.submitted)
	}
}

func TestInstallModelUnknownNotFound(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	_, err := h.InstallModel(context.Background(), connect.NewRequest(&modelsv1.InstallModelRequest{Id: "nope"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("got %v, want NotFound", connect.CodeOf(err))
	}
}

func TestInstallStateSurfacesInListAndGet(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	ctx := context.Background()
	if _, err := installer.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := h.GetModel(ctx, connect.NewRequest(&modelsv1.GetModelRequest{Id: "sd-1.5"}))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Msg.Model.Install == nil || !got.Msg.Model.Install.Installed {
		t.Fatal("install state should report installed in GetModel")
	}
	if got.Msg.Model.Install.Checksum == "" {
		t.Fatal("installed model should carry a pinned checksum")
	}
}

func TestRemoveModelClearsInstall(t *testing.T) {
	h, installer, _ := newInstallHandler(t)
	ctx := context.Background()
	if _, err := installer.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := h.RemoveModel(ctx, connect.NewRequest(&modelsv1.RemoveModelRequest{Id: "sd-1.5"})); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if installer.Installed(ctx, "sd-1.5") {
		t.Fatal("model should not be installed after remove")
	}
}

func TestAddCustomModelRoundTrip(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	ctx := context.Background()

	weights := filepath.Join(t.TempDir(), "weights.bin")
	if err := os.WriteFile(weights, []byte("w"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	resp, err := h.AddCustomModel(ctx, connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model: &modelsv1.Model{
			Id:         "my-model",
			Operations: []string{"upscale"},
			Backend:    "onnxruntime",
		},
		LocalPath: weights,
	}))
	if err != nil {
		t.Fatalf("add custom: %v", err)
	}
	if !resp.Msg.Model.Custom {
		t.Fatal("custom entry should be flagged custom")
	}

	// It now appears in ListModels.
	all, err := h.ListModels(ctx, connect.NewRequest(&modelsv1.ListModelsRequest{Operation: "upscale"}))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, m := range all.Msg.Models {
		if m.Id == "my-model" {
			found = true
		}
	}
	if !found {
		t.Fatal("custom model not surfaced in ListModels")
	}
}

func TestAddCustomModelSeedCollisionInvalid(t *testing.T) {
	h, _, _ := newInstallHandler(t)
	_, err := h.AddCustomModel(context.Background(), connect.NewRequest(&modelsv1.AddCustomModelRequest{
		Model: &modelsv1.Model{Id: "sd-1.5", Operations: []string{"text_to_image"}, Backend: "x"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("got %v, want InvalidArgument", connect.CodeOf(err))
	}
}
