package models

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	internalmodels "image-tools/internal/models"
)

func TestDoctorCatalogProjectsModelLifecycle(t *testing.T) {
	ctx := context.Background()
	reg, err := internalmodels.Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	database := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(ctx, database, apidb.SchemaProviderFunc(internalmodels.Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	installer := &internalmodels.Installer{
		Root:   t.TempDir(),
		Reg:    reg,
		Custom: internalmodels.NewCustomStore(database),
		State:  internalmodels.NewInstallStore(database),
	}
	h := NewConnectHandler(Deps{
		Registry:  reg,
		Store:     internalmodels.NewStore(database),
		Installer: installer,
	})

	var readyModel internalmodels.Model
	for _, model := range reg.Models() {
		if model.Backend == internalmodels.BackendBuiltin && model.Enabled && model.CapabilityLabels.CommercialUse == internalmodels.CommercialUseYes {
			readyModel = model
			break
		}
	}
	if readyModel.ID == "" {
		t.Fatal("registry has no enabled commercial-clean builtin model fixture")
	}
	if _, err := installer.Install(ctx, readyModel.ID, nil); err != nil {
		t.Fatalf("install weightless fixture: %v", err)
	}

	want, err := installer.Lifecycle(ctx, readyModel.ID)
	if err != nil {
		t.Fatalf("read fixture lifecycle: %v", err)
	}
	if !want.Ready() {
		t.Fatalf("fixture lifecycle = %+v, want ready", want)
	}

	resp, err := h.DoctorCatalog(ctx, connect.NewRequest(&modelsv1.DoctorCatalogRequest{}))
	if err != nil {
		t.Fatalf("doctor catalog: %v", err)
	}
	if got, want := len(resp.Msg.GetModels()), len(reg.Models()); got != want {
		t.Fatalf("doctor lifecycle model count = %d, want one row per catalog model (%d)", got, want)
	}
	var got *modelsv1.ModelLifecycle
	for _, model := range resp.Msg.GetModels() {
		if model.GetModelId() == readyModel.ID {
			got = model
			break
		}
	}
	if got == nil {
		t.Fatalf("doctor omitted lifecycle for %q", readyModel.ID)
	}
	if got.GetState() != modelsv1.ModelLifecycleState_MODEL_LIFECYCLE_STATE_READY || got.GetReason() != want.Reason || got.GetArtifactHash() != want.ArtifactHash || got.GetRuntimeHash() != want.RuntimeHash {
		t.Fatalf("doctor lifecycle = %+v, want state/reason/evidence from installer: %+v", got, want)
	}
}
