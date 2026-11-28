package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/internal/operatorstate"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	resourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources/resourcesv1connect"
	resourceshandler "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/handlers/resources"
)

func TestResourceCatalogRepositoryToBundleLossAndRecoveryRetainsSavedChoice(t *testing.T) {
	previousEnvironment, previousRoots, previousPath := rootEnvironment, operatorStateRoots, operatorStatePath
	t.Cleanup(func() {
		rootEnvironment, operatorStateRoots, operatorStatePath = previousEnvironment, previousRoots, previousPath
	})
	operatorStateRoots = nil

	repository := t.TempDir()
	bundle := t.TempDir()
	storageRoot := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(repository, "resources", "optional-index", "resource.json"), `{"name":"optional-index","display_name":"Optional index","category":"search","dependencies":[]}`)
	if err := os.MkdirAll(filepath.Join(repository, "scenarios"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bundle, "catalog", "scenarios"), 0o700); err != nil {
		t.Fatal(err)
	}

	rootEnvironment = mapEnvironment{"VROOLI_ROOT": repository, "VROOLI_STORAGE_ROOT": storageRoot}
	operatorStatePath = func() (string, error) {
		roots, err := resolveRoots()
		if err != nil {
			return "", err
		}
		return filepath.Join(roots.StateRoot, operatorstate.StateFile), nil
	}
	ctx := context.Background()
	state := operatorStateService()
	initial, err := state.Load(ctx)
	if err != nil {
		t.Fatalf("load initial operator state: %v", err)
	}
	if _, err := state.ApplyAtRevision(ctx, operatorstate.Revision(initial), []byte(`{"resources":{"optional-index":{"enabled":true}}}`)); err != nil {
		t.Fatalf("save optional resource choice: %v", err)
	}
	statePath, err := state.Path(ctx)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}

	path, rpcHandler := resourcesconnect.NewResourcesServiceHandler(resourceshandler.NewConnectHandler(resourceService()))
	router := mux.NewRouter()
	router.PathPrefix(path).Handler(rpcHandler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := resourcesconnect.NewResourcesServiceClient(server.Client(), server.URL)

	initialCatalog, err := client.ListDerivedResources(ctx, connect.NewRequest(&resourcesv1.ListDerivedResourcesRequest{}))
	if err != nil {
		t.Fatalf("repository catalog: %v", err)
	}
	if len(initialCatalog.Msg.GetResources()) != 1 || !initialCatalog.Msg.GetResources()[0].GetEnabled() {
		t.Fatalf("repository selection = %+v", initialCatalog.Msg.GetResources())
	}

	// The supported desktop configuration moves immutable catalogs to BUNDLE_ROOT
	// while keeping writable operator choices under VROOLI_STORAGE_ROOT.
	rootEnvironment = mapEnvironment{"BUNDLE_ROOT": bundle, "VROOLI_STORAGE_ROOT": storageRoot}
	_, err = client.ListDerivedResources(ctx, connect.NewRequest(&resourcesv1.ListDerivedResourcesRequest{}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("missing bundle resource catalog = %v, want Unavailable", err)
	}
	afterLoss, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterLoss) != string(before) {
		t.Fatal("catalog loss changed the saved operator resource choice")
	}

	write(filepath.Join(bundle, "catalog", "resources", "optional-index", "resource.json"), `{"name":"optional-index","display_name":"Optional index","category":"search","dependencies":[]}`)
	recovered, err := client.ListDerivedResources(ctx, connect.NewRequest(&resourcesv1.ListDerivedResourcesRequest{}))
	if err != nil {
		t.Fatalf("recovered bundle catalog: %v", err)
	}
	if len(recovered.Msg.GetResources()) != 1 || recovered.Msg.GetResources()[0].GetName() != "optional-index" || !recovered.Msg.GetResources()[0].GetEnabled() {
		t.Fatalf("recovered choice = %+v", recovered.Msg.GetResources())
	}

	// A malformed staged manifest is unsupported configuration, not an empty
	// catalog that silently hides the operator's saved choice.
	write(filepath.Join(bundle, "catalog", "resources", "broken", "resource.json"), `{not-json`)
	_, err = client.ListDerivedResources(ctx, connect.NewRequest(&resourcesv1.ListDerivedResourcesRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("malformed resource catalog = %v, want FailedPrecondition", err)
	}
	finalState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(finalState) != string(before) {
		t.Fatal("unsupported catalog changed the saved operator resource choice")
	}
}
