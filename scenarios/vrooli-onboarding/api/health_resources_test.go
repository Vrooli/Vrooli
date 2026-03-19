package main

import (
	"net/http"
	"testing"
)

// TestResourceHealthEndpoint verifies GET /api/v1/resources/health returns health statuses.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthEndpoint(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResStopped})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources, ok := resp["resources"].([]any)
	if !ok || len(resources) != 2 {
		t.Fatalf("expected 2 resources, got %v", resp["resources"])
	}

	healthyCount, ok := resp["healthy_count"].(float64)
	if !ok {
		t.Fatal("missing healthy_count field")
	}
	if healthyCount != 1 {
		t.Errorf("expected healthy_count=1, got %v", healthyCount)
	}
}

// TestResourceHealthAvailability verifies running resources are marked available.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthAvailability(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources := resp["resources"].([]any)
	first := resources[0].(map[string]any)
	if first["available"] != true {
		t.Errorf("expected running resource to be available=true, got %v", first["available"])
	}
	if first["last_checked"] == nil || first["last_checked"] == "" {
		t.Error("expected last_checked to be set")
	}
}

// TestResourceHealthUnreachable verifies stopped resources are marked unavailable.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthUnreachable(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResStopped})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources := resp["resources"].([]any)
	first := resources[0].(map[string]any)
	if first["available"] != false {
		t.Errorf("expected stopped resource to be available=false, got %v", first["available"])
	}
}

// TestResourceHealthAllStopped verifies healthy_count is 0 when all resources stopped.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthAllStopped(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResStopped, testResMystery})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	healthyCount := resp["healthy_count"].(float64)
	if healthyCount != 0 {
		t.Errorf("expected healthy_count=0 when all stopped, got %v", healthyCount)
	}
}

// TestResourceHealthManyResources verifies correct counting with many resources.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthManyResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		testResPostgres, testResRedis, testResOllama, testResPostgis, testResJudge0,
	})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources := resp["resources"].([]any)
	if len(resources) != 5 {
		t.Errorf("expected 5 resources, got %d", len(resources))
	}

	total := resp["total"].(float64)
	if int(total) != 5 {
		t.Errorf("expected total=5, got %v", total)
	}
}

// TestResourceHealthResponseStructure verifies all expected fields exist in response.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthResponseStructure(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	requiredFields := []string{"resources", "total", "healthy_count", "checked_at"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("response missing required field %q", field)
		}
	}

	resources := resp["resources"].([]any)
	first := resources[0].(map[string]any)
	resourceFields := []string{"name", "status", "category", "available", "last_checked"}
	for _, field := range resourceFields {
		if _, ok := first[field]; !ok {
			t.Errorf("resource entry missing required field %q", field)
		}
	}
}

// TestResourceHealthEmptyResources verifies response when no resources exist.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthEmptyResources(t *testing.T) {
	srv := newTestServer(t, []map[string]string{})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources, ok := resp["resources"].([]any)
	if !ok {
		t.Fatal("response missing 'resources' array")
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources, got %d", len(resources))
	}

	total := resp["total"].(float64)
	if int(total) != 0 {
		t.Errorf("expected total=0, got %v", total)
	}

	healthyCount := resp["healthy_count"].(float64)
	if healthyCount != 0 {
		t.Errorf("expected healthy_count=0, got %v", healthyCount)
	}
}

// TestResourceHealthCheckedAtPresent verifies the checked_at timestamp is present and non-empty.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthCheckedAtPresent(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	checkedAt, ok := resp["checked_at"].(string)
	if !ok || checkedAt == "" {
		t.Error("expected non-empty checked_at timestamp")
	}
}

// TestResourceHealthMixedStatuses verifies correct availability when resources have mixed statuses.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthMixedStatuses(t *testing.T) {
	srv := newTestServer(t, []map[string]string{
		testResPostgres, // running
		testResStopped,  // stopped
		testResOllama,   // installed (not running)
	})

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	resources := resp["resources"].([]any)
	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	// Verify each resource has the correct availability
	for _, r := range resources {
		resource := r.(map[string]any)
		name := resource["name"].(string)
		available := resource["available"].(bool)
		switch name {
		case "postgres":
			if !available {
				t.Error("running postgres should be available")
			}
		case "redis":
			if available {
				t.Error("stopped redis should not be available")
			}
		case "ollama":
			if available {
				t.Error("installed-but-not-running ollama should not be available")
			}
		}
	}
}
