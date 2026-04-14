package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestGlossaryAll verifies GET /api/v1/glossary returns all entries.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossaryAll(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected non-empty entries list")
	}

	count, ok := resp["count"].(float64)
	if !ok || count == 0 {
		t.Fatal("expected non-zero count")
	}
}

// TestGlossarySearch verifies searching glossary by term.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearch(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=database")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected matching entries for 'database'")
	}

	if resp["query"] != "database" {
		t.Errorf("expected query=database, got %v", resp["query"])
	}
}

// TestGlossarySearchNoMatch verifies empty search results.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchNoMatch(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=xyznonexistent")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	if entries, ok := resp["entries"].([]any); ok && len(entries) > 0 {
		t.Error("expected no matching entries")
	}
}

// TestSetupOrder verifies GET /api/v1/setup-order returns dependency-sorted resources.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrder(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres, testResPostgis, testResRedis})

	w := doGet(t, srv, "/api/v1/setup-order")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	order, ok := resp["setup_order"].([]any)
	if !ok || len(order) != 3 {
		t.Fatalf("expected 3 ordered resources, got %v", resp["setup_order"])
	}

	// postgres should come before postgis (postgis depends on postgres)
	var postgresOrder, postgisOrder float64
	for _, item := range order {
		entry := item.(map[string]any)
		if entry["name"] == "postgres" {
			postgresOrder = entry["order"].(float64)
		}
		if entry["name"] == "postgis" {
			postgisOrder = entry["order"].(float64)
		}
	}

	if postgresOrder >= postgisOrder {
		t.Errorf("postgres (order=%v) should come before postgis (order=%v)", postgresOrder, postgisOrder)
	}
}

// TestSetupOrderCircularDeps verifies handling of circular dependencies.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrderCircularDeps(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResJudge0})

	w := doGet(t, srv, "/api/v1/setup-order")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	// judge0 depends on postgres and redis which aren't available,
	// but it should still appear in the order
	order, ok := resp["setup_order"].([]any)
	if !ok || len(order) == 0 {
		t.Fatal("expected non-empty setup order even with missing deps")
	}
}

// TestSetupOrderLoadError verifies 500 when resources file is missing.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrderLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer(nil)

	w := doGet(t, srv, "/api/v1/setup-order")
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestResourceHealthLoadError verifies 500 when resources file is missing.
// [REQ:REQ-P1-001] - Resource Health API
func TestResourceHealthLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer(nil)

	w := doGet(t, srv, "/api/v1/resources/health")
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestConfigValidateInvalidJSON verifies 400 for malformed JSON body.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateInvalidJSON(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate", "{bad json")
	requireStatus(t, w, http.StatusBadRequest)
}

// TestConfigValidateDisabledResource verifies warnings for disabled resources.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateDisabledResource(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"postgres": {"enabled": false, "name": "postgres"}}}`)
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	valid, _ := resp["valid"].(bool)
	if !valid {
		t.Error("expected valid=true for known but disabled resource")
	}

	results := resp["results"].([]any)
	result := results[0].(map[string]any)
	warnings, ok := result["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Error("expected warning about disabled resource")
	}
}

// TestConfigGenerateLoadError verifies 500 when resources file cannot be loaded.
// [REQ:REQ-P0-004] - Config Generation
func TestConfigGenerateLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer(nil)

	w := doPost(t, srv, "/api/v1/config/generate", `{"resources": ["postgres"]}`)
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestConfigValidateLoadError verifies 500 when resources file cannot be loaded.
// [REQ:REQ-P0-005] - Config Validation
func TestConfigValidateLoadError(t *testing.T) {
	stubResourceStatusJSON(t, nil, errors.New("command failed"))
	srv := NewServer(nil)

	w := doPost(t, srv, "/api/v1/config/validate",
		`{"resources": {"postgres": {"enabled": true, "name": "postgres"}}}`)
	requireStatus(t, w, http.StatusInternalServerError)
}

// TestTopoSortEmpty verifies topological sort handles empty input.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestTopoSortEmpty(t *testing.T) {
	result := topoSortResources(nil, nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %d items", len(result))
	}
}

// TestTopoSortSingleNoDeps verifies a single resource with no dependencies.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestTopoSortSingleNoDeps(t *testing.T) {
	resources := []Resource{{Name: "redis", Category: "database", Status: "running"}}
	result := topoSortResources(resources, map[string][]string{})
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Order != 1 {
		t.Errorf("expected order=1 for resource with no deps, got %d", result[0].Order)
	}
}

// TestTopoSortChainedDeps verifies correct ordering for a dependency chain (A -> B -> C).
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestTopoSortChainedDeps(t *testing.T) {
	resources := []Resource{
		{Name: "c", Category: "general"},
		{Name: "b", Category: "general"},
		{Name: "a", Category: "general"},
	}
	deps := map[string][]string{
		"b": {"a"},
		"c": {"b"},
	}
	result := topoSortResources(resources, deps)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	orderMap := make(map[string]int, 3)
	for _, r := range result {
		orderMap[r.Name] = r.Order
	}
	if orderMap["a"] >= orderMap["b"] {
		t.Errorf("a (order=%d) should come before b (order=%d)", orderMap["a"], orderMap["b"])
	}
	if orderMap["b"] >= orderMap["c"] {
		t.Errorf("b (order=%d) should come before c (order=%d)", orderMap["b"], orderMap["c"])
	}
}

// TestTopoSortMutualCircularDeps verifies both resources appear when they depend on each other.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestTopoSortMutualCircularDeps(t *testing.T) {
	resources := []Resource{
		{Name: "x", Category: "general"},
		{Name: "y", Category: "general"},
	}
	deps := map[string][]string{
		"x": {"y"},
		"y": {"x"},
	}
	result := topoSortResources(resources, deps)
	if len(result) != 2 {
		t.Fatalf("expected 2 results even with circular deps, got %d", len(result))
	}
	// Both should get the catch-all order since neither can be placed first
	if result[0].Order == 0 || result[1].Order == 0 {
		t.Error("circular deps should still get assigned an order > 0")
	}
}

// TestGlossarySearchCaseInsensitive verifies search is case-insensitive.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchCaseInsensitive(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=DATABASE")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected matching entries for case-insensitive 'DATABASE'")
	}
}

// TestGlossarySearchDescription verifies search matches descriptions not just terms.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchDescription(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=caching")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries, ok := resp["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("expected match when searching descriptions ('caching' should match redis description)")
	}

	// Verify at least one entry actually mentions "caching" in term or description
	foundCaching := false
	for _, e := range entries {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		term, _ := entry["term"].(string)
		desc, _ := entry["description"].(string)
		if strings.Contains(strings.ToLower(term), "cach") || strings.Contains(strings.ToLower(desc), "cach") {
			foundCaching = true
			break
		}
	}
	if !foundCaching {
		t.Errorf("expected at least one result to contain 'cach' in term or description, got %v", entries)
	}
}

// TestGlossaryEntryStructure verifies each glossary entry has all required fields.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossaryEntryStructure(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries := resp["entries"].([]any)
	for i, e := range entries {
		entry := e.(map[string]any)
		if _, ok := entry["term"]; !ok {
			t.Errorf("entry[%d] missing 'term' field", i)
		}
		if _, ok := entry["description"]; !ok {
			t.Errorf("entry[%d] missing 'description' field", i)
		}
		if _, ok := entry["category"]; !ok {
			t.Errorf("entry[%d] missing 'category' field", i)
		}
	}
}

// TestGlossarySearchMultipleMatches verifies search returns all matching entries.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchMultipleMatches(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	// "service" should match multiple entries (descriptions mention "service")
	w := doGet(t, srv, "/api/v1/glossary?q=service")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries, ok := resp["entries"].([]any)
	if !ok {
		t.Fatal("expected entries array")
	}
	if len(entries) < 2 {
		t.Errorf("expected multiple matches for 'service', got %d", len(entries))
	}

	// Verify every returned entry actually contains "service" in term or description
	for i, e := range entries {
		entry, ok := e.(map[string]any)
		if !ok {
			t.Errorf("entries[%d] is not a map", i)
			continue
		}
		term, _ := entry["term"].(string)
		desc, _ := entry["description"].(string)
		if !strings.Contains(strings.ToLower(term), "service") && !strings.Contains(strings.ToLower(desc), "service") {
			t.Errorf("entries[%d] (term=%q) does not contain 'service' in term or description", i, term)
		}
	}

	// Verify count matches entries
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count' field")
	}
	if int(count) != len(entries) {
		t.Errorf("count=%v but entries has %d items", count, len(entries))
	}
}

// TestGlossarySearchQueryEchoed verifies the query field is returned in filtered results.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchQueryEchoed(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=port")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	if resp["query"] != "port" {
		t.Errorf("expected query=port, got %v", resp["query"])
	}
	// "port" appears as a glossary term, so we should have a match
	count := int(resp["count"].(float64))
	if count == 0 {
		t.Error("expected at least one match for 'port'")
	}
}

// TestGlossaryCountMatchesEntries verifies the count field matches the entries length.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossaryCountMatchesEntries(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	entries := resp["entries"].([]any)
	count := int(resp["count"].(float64))
	if count != len(entries) {
		t.Errorf("count=%d but entries has %d items", count, len(entries))
	}
}

// TestGlossarySearchWhitespace verifies whitespace-only queries return zero results and don't crash.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchWhitespace(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=%20%20")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	// Whitespace query returns 0 count with null/empty entries
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count' field")
	}
	if int(count) != 0 {
		t.Errorf("expected count=0 for whitespace query, got %v", count)
	}
	// Query should be echoed back
	if resp["query"] != "  " {
		t.Errorf("expected query to be echoed back, got %v", resp["query"])
	}
}

// TestGlossarySearchSpecialChars verifies search handles special characters gracefully.
// [REQ:REQ-P2-002] - Technical Term Glossary
func TestGlossarySearchSpecialChars(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResPostgres})

	w := doGet(t, srv, "/api/v1/glossary?q=%3Cscript%3E")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	// Should not crash; returns 0 matches for HTML injection attempt
	count, ok := resp["count"].(float64)
	if !ok {
		t.Fatal("response missing 'count' field")
	}
	if int(count) != 0 {
		t.Errorf("expected count=0 for special chars query, got %v", count)
	}
}

// TestSetupOrderNoDeps verifies setup order with a resource that has no dependencies.
// [REQ:REQ-P2-001] - Setup Order Algorithm
func TestSetupOrderNoDeps(t *testing.T) {
	srv := newTestServer(t, []map[string]string{testResRedis})

	w := doGet(t, srv, "/api/v1/setup-order")
	requireStatus(t, w, http.StatusOK)

	var resp map[string]any
	decodeJSON(t, w, &resp)

	order, ok := resp["setup_order"].([]any)
	if !ok || len(order) != 1 {
		t.Fatalf("expected 1 ordered resource, got %v", resp["setup_order"])
	}

	entry, ok := order[0].(map[string]any)
	if !ok {
		t.Fatal("setup_order entry is not a map")
	}
	if entry["name"] != "redis" {
		t.Errorf("expected redis, got %v", entry["name"])
	}
	if entry["order"].(float64) != 1 {
		t.Errorf("expected order=1 for single resource, got %v", entry["order"])
	}
}
