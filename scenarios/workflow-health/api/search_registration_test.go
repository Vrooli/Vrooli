package main

import (
	"path/filepath"
	"runtime"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

func searchJSONForTest(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(here), "..", ".vrooli", "search.json")
}

func TestSearchJSONMapsWorkflowProviders(t *testing.T) {
	file, err := aisearch.LoadSearchFile(searchJSONForTest(t))
	if err != nil {
		t.Fatalf("load .vrooli/search.json: %v", err)
	}
	if len(file.Providers) != 3 {
		t.Fatalf("want 3 workflow providers, got %d", len(file.Providers))
	}

	descriptors, err := searchregister.Descriptors(file)
	if err != nil {
		t.Fatalf("map search.json to registry descriptors: %v", err)
	}
	if len(descriptors) != 3 {
		t.Fatalf("want 3 descriptors, got %d", len(descriptors))
	}

	wantTypes := map[string]string{
		"workflow-health.workflows": "workflow.flow",
		"workflow-health.tests":     "workflow.test",
		"workflow-health.fragments": "workflow.fragment",
	}
	for _, descriptor := range descriptors {
		wantType, ok := wantTypes[descriptor.GetProviderId()]
		if !ok {
			t.Fatalf("unexpected provider_id %q", descriptor.GetProviderId())
		}
		if got := descriptor.GetProviderGroup(); got != "workflow-health" {
			t.Errorf("%s provider_group = %q, want workflow-health", descriptor.GetProviderId(), got)
		}
		if got := descriptor.GetType(); got != wantType {
			t.Errorf("%s type = %q, want %q", descriptor.GetProviderId(), got, wantType)
		}
		if got := descriptor.GetEndpoint().GetHttpJson().GetScenarioId(); got != "workflow-health" {
			t.Errorf("%s endpoint scenario_id = %q, want workflow-health", descriptor.GetProviderId(), got)
		}
		if got := descriptor.GetEndpoint().GetHttpJson().GetPath(); got != "/vrooli.workflow_health.v1.workflows.WorkflowSearchService/SearchWorkflows" {
			t.Errorf("%s endpoint path = %q", descriptor.GetProviderId(), got)
		}
		mapping := descriptor.GetResultMapping()
		if mapping == nil {
			t.Fatalf("%s result_mapping must be present", descriptor.GetProviderId())
		}
		if got := mapping.GetIdField(); got != "id" {
			t.Errorf("%s id_field = %q, want id", descriptor.GetProviderId(), got)
		}
		if got := mapping.GetPathField(); got != "path" {
			t.Errorf("%s path_field = %q, want path", descriptor.GetProviderId(), got)
		}
	}
}
