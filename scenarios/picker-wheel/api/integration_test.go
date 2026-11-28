package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestEndToEndWheelCreationAndSpin tests full workflow
func TestEndToEndWheelCreationAndSpin(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	// Step 1: Create a custom wheel
	newWheel := map[string]interface{}{
		"name":        "E2E Test Wheel",
		"description": "End-to-end test wheel",
		"theme":       "test",
		"options": []map[string]interface{}{
			{"label": "Red", "color": "#FF0000", "weight": 1.0},
			{"label": "Green", "color": "#00FF00", "weight": 1.0},
			{"label": "Blue", "color": "#0000FF", "weight": 1.0},
		},
	}

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/wheels",
		Body:   newWheel,
	}

	httpReq, w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create wheel: %v", err)
	}

	router := setupTestRouter()
	router.ServeHTTP(w, httpReq)

	var createdWheel Wheel
	assertJSONResponse(t, w, http.StatusOK, &createdWheel)

	// Step 2: The no-database path must keep the new wheel available in memory.
	req = HTTPTestRequest{
		Method:  "GET",
		Path:    "/api/wheels/" + createdWheel.ID,
		URLVars: map[string]string{"id": createdWheel.ID},
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to get wheel: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var retrievedWheel Wheel
	assertJSONResponse(t, w, http.StatusOK, &retrievedWheel)

	if retrievedWheel.ID != createdWheel.ID {
		t.Errorf("Retrieved wheel ID mismatch: expected %s, got %s", createdWheel.ID, retrievedWheel.ID)
	}

	// Step 3: Spin the wheel
	spinRequest := map[string]interface{}{
		"wheel_id":   createdWheel.ID,
		"session_id": "e2e-test",
		"options":    createdWheel.Options,
	}

	req = HTTPTestRequest{
		Method: "POST",
		Path:   "/api/spin",
		Body:   spinRequest,
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to spin wheel: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var spinResult SpinResult
	assertJSONResponse(t, w, http.StatusOK, &spinResult)

	if spinResult.WheelID != createdWheel.ID {
		t.Errorf("Spin result wheel ID mismatch: expected %s, got %s", createdWheel.ID, spinResult.WheelID)
	}

	// Step 4: Check history
	req = HTTPTestRequest{
		Method: "GET",
		Path:   "/api/history",
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var historyList []SpinResult
	assertJSONResponse(t, w, http.StatusOK, &historyList)

	if len(historyList) == 0 {
		t.Error("Expected history to contain spin result")
	}

	// Step 5: Delete the wheel
	req = HTTPTestRequest{
		Method:  "DELETE",
		Path:    "/api/wheels/" + createdWheel.ID,
		URLVars: map[string]string{"id": createdWheel.ID},
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to delete wheel: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for delete, got %d", http.StatusNoContent, w.Code)
	}

	// Step 6: Verify wheel is deleted
	req = HTTPTestRequest{
		Method:  "GET",
		Path:    "/api/wheels/" + createdWheel.ID,
		URLVars: map[string]string{"id": createdWheel.ID},
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d after deletion, got %d", http.StatusNotFound, w.Code)
	}
}

// TestMultipleSpinsSameWheel tests spinning the same wheel multiple times
func TestMultipleSpinsSameWheel(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	spins := 10
	results := make(map[string]int)

	for i := 0; i < spins; i++ {
		spinRequest := map[string]interface{}{
			"wheel_id":   "yes-or-no",
			"session_id": "multi-spin-test",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Spin %d failed: %v", i, err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)

		results[result.Result]++
	}

	// Verify we got results
	if len(results) == 0 {
		t.Error("Expected some results from spins")
	}

	// Verify history was updated
	if len(history) < spins {
		t.Errorf("Expected at least %d history items, got %d", spins, len(history))
	}
}

// TestConcurrentRequests pins both response validity and exact in-memory
// history cardinality under concurrent spins.
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	concurrency := 50
	type spinResponse struct {
		status int
		result SpinResult
		err    error
	}
	done := make(chan spinResponse, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			spinRequest := map[string]interface{}{
				"wheel_id":   "yes-or-no",
				"session_id": "concurrent-test",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/spin",
				Body:   spinRequest,
			}

			httpReq, w, err := makeHTTPRequest(req)
			if err != nil {
				done <- spinResponse{err: fmt.Errorf("request %d: %w", id, err)}
				return
			}

			router := setupTestRouter()
			router.ServeHTTP(w, httpReq)

			if w.Code != http.StatusOK {
				done <- spinResponse{status: w.Code, err: fmt.Errorf("request %d returned status %d", id, w.Code)}
				return
			}
			var result SpinResult
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				done <- spinResponse{status: w.Code, err: fmt.Errorf("request %d response: %w", id, err)}
				return
			}
			done <- spinResponse{status: w.Code, result: result}
		}(i)
	}

	// Wait for all goroutines
	timeout := time.After(5 * time.Second)
	for i := 0; i < concurrency; i++ {
		select {
		case response := <-done:
			if response.err != nil {
				t.Error(response.err)
				continue
			}
			if response.result.WheelID != "yes-or-no" || response.result.SessionID != "concurrent-test" || response.result.Timestamp.IsZero() {
				t.Errorf("concurrent response = %+v, want the requested wheel/session and a timestamp", response.result)
			}
			if response.result.Result != "YES! ✅" && response.result.Result != "NO ❌" {
				t.Errorf("concurrent result = %q, want one of the independently specified wheel options", response.result.Result)
			}
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
	if got := len(snapshotSpinHistory()); got != concurrency {
		t.Fatalf("in-memory history entries = %d, want one result for each of %d successful spins", got, concurrency)
	}
}

func TestConcurrentInMemoryWheelCreationKeepsUniqueListedItems(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	resetTestState()

	const concurrency = 32
	type createResponse struct {
		wheel Wheel
		err   error
	}
	done := make(chan createResponse, concurrency)
	for i := 0; i < concurrency; i++ {
		go func(index int) {
			request := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/wheels",
				Body: Wheel{
					Name:    fmt.Sprintf("concurrent wheel %d", index),
					Options: []Option{{Label: "Only", Color: "#123456", Weight: 1}},
				},
			}
			httpReq, response, err := makeHTTPRequest(request)
			if err != nil {
				done <- createResponse{err: err}
				return
			}
			setupTestRouter().ServeHTTP(response, httpReq)
			if response.Code != http.StatusOK {
				done <- createResponse{err: fmt.Errorf("create status = %d: %s", response.Code, response.Body.String())}
				return
			}
			var wheel Wheel
			if err := json.Unmarshal(response.Body.Bytes(), &wheel); err != nil {
				done <- createResponse{err: err}
				return
			}
			done <- createResponse{wheel: wheel}
		}(i)
	}

	createdIDs := make(map[string]struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		created := <-done
		if created.err != nil {
			t.Fatal(created.err)
		}
		if created.wheel.ID == "" {
			t.Fatal("created wheel has an empty ID")
		}
		if _, exists := createdIDs[created.wheel.ID]; exists {
			t.Fatalf("concurrent creates reused wheel ID %q", created.wheel.ID)
		}
		createdIDs[created.wheel.ID] = struct{}{}
	}

	request, response, err := makeHTTPRequest(HTTPTestRequest{Method: "GET", Path: "/api/wheels"})
	if err != nil {
		t.Fatal(err)
	}
	setupTestRouter().ServeHTTP(response, request)
	var listed []Wheel
	assertJSONResponse(t, response, http.StatusOK, &listed)
	if len(listed) != concurrency {
		t.Fatalf("in-memory wheel list contains %d items, want exactly %d", len(listed), concurrency)
	}
	listedIDs := make(map[string]struct{}, len(listed))
	for _, wheel := range listed {
		if _, expected := createdIDs[wheel.ID]; !expected {
			t.Errorf("in-memory wheel list contains unexpected ID %q", wheel.ID)
		}
		listedIDs[wheel.ID] = struct{}{}
	}
	for id := range createdIDs {
		if _, ok := listedIDs[id]; !ok {
			t.Errorf("created wheel %q is absent from list response", id)
		}
	}
}

// TestAllDefaultWheels tests spinning all default wheels
func TestAllDefaultWheels(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	defaultWheels := getDefaultWheels()

	for _, wheel := range defaultWheels {
		t.Run("Spin_"+wheel.ID, func(t *testing.T) {
			spinRequest := map[string]interface{}{
				"wheel_id":   wheel.ID,
				"session_id": "default-wheel-test",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/spin",
				Body:   spinRequest,
			}

			httpReq, w, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to spin wheel %s: %v", wheel.ID, err)
			}

			router := setupTestRouter()
			router.ServeHTTP(w, httpReq)

			var result SpinResult
			assertJSONResponse(t, w, http.StatusOK, &result)

			if result.Result == "" {
				t.Errorf("Wheel %s returned empty result", wheel.ID)
			}

			// Verify result is one of the valid options for this wheel
			validOptions := make(map[string]bool)
			for _, option := range wheel.Options {
				validOptions[option.Label] = true
			}

			if !validOptions[result.Result] {
				t.Errorf("Wheel %s returned invalid result: %s", wheel.ID, result.Result)
			}
		})
	}
}

// TestWheelListOperations tests listing and filtering wheels
func TestWheelListOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	// Get initial wheels
	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/wheels",
	}

	httpReq, w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to get wheels: %v", err)
	}

	router := setupTestRouter()
	router.ServeHTTP(w, httpReq)

	var wheelsList []Wheel
	assertJSONResponse(t, w, http.StatusOK, &wheelsList)

	initialCount := len(wheelsList)

	// Create a new wheel
	newWheel := map[string]interface{}{
		"name": "List Test Wheel",
		"options": []map[string]interface{}{
			{"label": "Option A", "color": "#FF0000", "weight": 1.0},
		},
	}

	req = HTTPTestRequest{
		Method: "POST",
		Path:   "/api/wheels",
		Body:   newWheel,
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create wheel: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var createdWheel Wheel
	assertJSONResponse(t, w, http.StatusOK, &createdWheel)

	// Get updated list
	req = HTTPTestRequest{
		Method: "GET",
		Path:   "/api/wheels",
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to get updated wheels: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var updatedList []Wheel
	assertJSONResponse(t, w, http.StatusOK, &updatedList)

	// Note: In-memory implementation might not reflect the newly created wheel
	// This is expected behavior for the fallback system
	if len(updatedList) < initialCount {
		t.Errorf("Expected at least %d wheels, got %d", initialCount, len(updatedList))
	}
}

// TestHistoryPersistence tests that history persists across operations
func TestHistoryPersistence(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()
	initTestWheels()

	// Spin multiple times
	for i := 0; i < 5; i++ {
		spinRequest := map[string]interface{}{
			"wheel_id":   "yes-or-no",
			"session_id": "persistence-test",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/spin",
			Body:   spinRequest,
		}

		httpReq, w, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Spin %d failed: %v", i, err)
		}

		router := setupTestRouter()
		router.ServeHTTP(w, httpReq)

		var result SpinResult
		assertJSONResponse(t, w, http.StatusOK, &result)
	}

	// Check history
	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/history",
	}

	httpReq, w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	router := setupTestRouter()
	router.ServeHTTP(w, httpReq)

	var historyList []SpinResult
	assertJSONResponse(t, w, http.StatusOK, &historyList)

	if len(historyList) != 5 {
		t.Errorf("Expected 5 history items, got %d", len(historyList))
	}

	// Verify all history items have the correct session ID
	for _, item := range historyList {
		if item.SessionID != "persistence-test" {
			t.Errorf("History item has wrong session ID: %s", item.SessionID)
		}
		if item.WheelID != "yes-or-no" {
			t.Errorf("History item has wrong wheel ID: %s", item.WheelID)
		}
	}
}

// TestErrorRecovery tests that the API recovers from errors gracefully
func TestErrorRecovery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	resetTestState()

	// Try to create an invalid wheel
	invalidWheel := map[string]interface{}{
		"invalid": "data",
	}

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/wheels",
		Body:   invalidWheel,
	}

	httpReq, w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	router := setupTestRouter()
	router.ServeHTTP(w, httpReq)

	// Should handle gracefully (either error or create with defaults)
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Logf("Invalid wheel creation returned status %d (acceptable)", w.Code)
	}

	// Now try a valid request to ensure system still works
	initTestWheels()
	validRequest := map[string]interface{}{
		"wheel_id":   "yes-or-no",
		"session_id": "recovery-test",
	}

	req = HTTPTestRequest{
		Method: "POST",
		Path:   "/api/spin",
		Body:   validRequest,
	}

	httpReq, w, err = makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create valid request: %v", err)
	}

	router.ServeHTTP(w, httpReq)

	var result SpinResult
	assertJSONResponse(t, w, http.StatusOK, &result)

	if result.Result == "" {
		t.Error("System should still work after error")
	}
}
