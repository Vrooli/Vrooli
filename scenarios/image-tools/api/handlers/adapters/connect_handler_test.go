package adapters

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
	internaladapters "image-tools/internal/adapters"
)

// newHandler builds the Connect adapter over the real embedded seed catalog.
// The store, installer and model registry are deliberately nil: read paths must
// stay usable in a read-only wiring, and the write paths must refuse with an
// actionable code rather than panicking.
func newHandler(t *testing.T) (*connectHandler, *internaladapters.Registry) {
	t.Helper()
	registry, err := internaladapters.Load()
	if err != nil {
		t.Fatal(err)
	}
	return NewConnectHandler(Deps{Registry: registry}), registry
}

func TestListAdaptersReturnsSeedCatalogAndFiltersByKind(t *testing.T) {
	h, registry := newHandler(t)

	all, err := h.ListAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListAdaptersRequest{}))
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all.Msg.GetAdapters()) != len(registry.Adapters()) {
		t.Fatalf("listed %d adapters, want %d", len(all.Msg.GetAdapters()), len(registry.Adapters()))
	}

	kind := string(registry.Adapters()[0].Kind)
	filtered, err := h.ListAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListAdaptersRequest{Kind: kind}))
	if err != nil {
		t.Fatalf("list by kind: %v", err)
	}
	if len(filtered.Msg.GetAdapters()) == 0 {
		t.Fatalf("kind %q returned no adapters", kind)
	}
	for _, a := range filtered.Msg.GetAdapters() {
		if a.GetKind() != kind {
			t.Errorf("adapter %q kind = %q, want %q", a.GetId(), a.GetKind(), kind)
		}
	}
}

func TestListAdaptersUnknownKindReturnsEmptyNotError(t *testing.T) {
	h, _ := newHandler(t)
	resp, err := h.ListAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListAdaptersRequest{Kind: "does-not-exist"}))
	if err != nil {
		t.Fatalf("unknown kind: %v", err)
	}
	if len(resp.Msg.GetAdapters()) != 0 {
		t.Fatalf("unknown kind returned %d adapters", len(resp.Msg.GetAdapters()))
	}
}

func TestGetAdapterFoundAndMissing(t *testing.T) {
	h, registry := newHandler(t)
	want := registry.Adapters()[0]

	resp, err := h.GetAdapter(context.Background(), connect.NewRequest(&adaptersv1.GetAdapterRequest{Id: want.ID}))
	if err != nil {
		t.Fatalf("get known: %v", err)
	}
	if resp.Msg.GetAdapter().GetId() != want.ID {
		t.Fatalf("id = %q, want %q", resp.Msg.GetAdapter().GetId(), want.ID)
	}

	_, err = h.GetAdapter(context.Background(), connect.NewRequest(&adaptersv1.GetAdapterRequest{Id: "absent"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing adapter code = %v, want NotFound", connect.CodeOf(err))
	}
}

func TestWritePathsRefuseWithoutTheirSeams(t *testing.T) {
	h, registry := newHandler(t)
	id := registry.Adapters()[0].ID

	_, err := h.SetAdapterEnabled(context.Background(), connect.NewRequest(&adaptersv1.SetAdapterEnabledRequest{Id: id, Enabled: true}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("set enabled without store = %v, want Internal", connect.CodeOf(err))
	}

	_, err = h.InstallAdapter(context.Background(), connect.NewRequest(&adaptersv1.InstallAdapterRequest{Id: id}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("install without installer = %v, want Unimplemented", connect.CodeOf(err))
	}

	_, err = h.RemoveAdapter(context.Background(), connect.NewRequest(&adaptersv1.RemoveAdapterRequest{Id: id}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("remove without installer = %v, want Unimplemented", connect.CodeOf(err))
	}
}

func TestListCompatibleAdaptersRequiresASelector(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.ListCompatibleAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListCompatibleAdaptersRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty selector code = %v, want InvalidArgument", connect.CodeOf(err))
	}

	_, err = h.ListCompatibleAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListCompatibleAdaptersRequest{ModelId: "unknown"}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("model_id without registry code = %v, want Internal", connect.CodeOf(err))
	}
}

func TestListCompatibleAdaptersByArchitecture(t *testing.T) {
	h, registry := newHandler(t)
	arch := string(registry.Adapters()[0].Architecture)

	resp, err := h.ListCompatibleAdapters(context.Background(), connect.NewRequest(&adaptersv1.ListCompatibleAdaptersRequest{Architecture: arch}))
	if err != nil {
		t.Fatalf("compatible: %v", err)
	}
	if resp.Msg.GetArchitecture() != arch {
		t.Fatalf("architecture = %q, want %q", resp.Msg.GetArchitecture(), arch)
	}
	if len(resp.Msg.GetAdapters()) != len(registry.Compatible(registry.Adapters()[0].Architecture)) {
		t.Fatalf("compatible count = %d, want %d", len(resp.Msg.GetAdapters()), len(registry.Compatible(registry.Adapters()[0].Architecture)))
	}
}

func TestDoctorCatalogFoldsRegistryLint(t *testing.T) {
	h, _ := newHandler(t)
	resp, err := h.DoctorCatalog(context.Background(), connect.NewRequest(&adaptersv1.DoctorCatalogRequest{}))
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	// The seed catalog must be clean: a finding here means the shipped rows
	// drifted from the licensing/install invariants the contract promises.
	if !resp.Msg.GetOk() {
		t.Fatalf("seed catalog doctor findings = %v", resp.Msg.GetFindings())
	}
}

func TestAddCustomErrorMapsActionableCauses(t *testing.T) {
	if got := connect.CodeOf(addCustomError(internaladapters.ErrCustomShadowsSeed)); got != connect.CodeInvalidArgument {
		t.Errorf("shadowing seed code = %v, want InvalidArgument", got)
	}
	if got := connect.CodeOf(addCustomError(errors.New("disk"))); got != connect.CodeInternal {
		t.Errorf("generic code = %v, want Internal", got)
	}
}
