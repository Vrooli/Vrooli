package graph

import (
	"reflect"
	"testing"
)

func TestBuildArchetypeSignalsRetainsImporterSourceForEachStructuralSignal(t *testing.T) {
	snapshot := GraphSnapshot{
		Files: []FileNode{
			{ID: "f-handler", Path: "api/handlers/orders/connect.go", PackageID: "p-handler"},
			{ID: "f-storage", Path: "api/internal/orders/repository.go", PackageID: "p-orders"},
			{ID: "f-orchestration", Path: "api/internal/orders/runner.go", PackageID: "p-orders"},
		},
		Packages: []PackageNode{
			{ID: "p-handler", RepoPath: "api/handlers/orders"},
			{ID: "p-orders", RepoPath: "api/internal/orders"},
			{ID: "p-db", ImportPath: "database/sqlite"},
			{ID: "p-workflow", ImportPath: "workflow/state-machine"},
		},
		Imports: []ImportEdge{
			{From: "f-storage", ToPackageID: "p-db"},
			{From: "f-orchestration", ToPackageID: "p-workflow"},
		},
	}
	got := BuildArchetypeSignals(snapshot, "orders", []string{"api/internal/orders/**", "api/handlers/orders/**"})
	if !got.OwnsStorage || !got.HasWorkflow || !got.HasHTTPHandler {
		t.Fatalf("structural signals = %+v", got)
	}
	if !reflect.DeepEqual(got.StoragePaths, []string{"api/internal/orders/repository.go"}) {
		t.Fatalf("storage source paths = %v", got.StoragePaths)
	}
	if !reflect.DeepEqual(got.WorkflowPaths, []string{"api/internal/orders/runner.go"}) {
		t.Fatalf("workflow source paths = %v", got.WorkflowPaths)
	}
	if !reflect.DeepEqual(got.HTTPHandlerPaths, []string{"api/handlers/orders/connect.go"}) {
		t.Fatalf("handler source paths = %v", got.HTTPHandlerPaths)
	}
}
