package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestTestHelpers tests the test helper functions themselves
func TestTestHelpers(t *testing.T) {
	t.Run("PtrString", func(t *testing.T) {
		str := "test"
		ptr := ptrString(str)
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != str {
			t.Errorf("Expected %s, got %s", str, *ptr)
		}

		// Test empty string
		empty := ""
		emptyPtr := ptrString(empty)
		if emptyPtr == nil || *emptyPtr != "" {
			t.Errorf("Expected empty string pointer, got %v", emptyPtr)
		}
	})

	t.Run("PtrInt", func(t *testing.T) {
		num := 42
		ptr := ptrInt(num)
		if ptr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *ptr != num {
			t.Errorf("Expected %d, got %d", num, *ptr)
		}

		// Test zero
		zero := 0
		zeroPtr := ptrInt(zero)
		if zeroPtr == nil || *zeroPtr != 0 {
			t.Errorf("Expected 0, got %v", zeroPtr)
		}

		// Test negative
		neg := -10
		negPtr := ptrInt(neg)
		if negPtr == nil || *negPtr != neg {
			t.Errorf("Expected %d, got %v", neg, negPtr)
		}
	})

	t.Run("PtrBool", func(t *testing.T) {
		trueVal := true
		truePtr := ptrBool(trueVal)
		if truePtr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *truePtr != true {
			t.Error("Expected true, got false")
		}

		falseVal := false
		falsePtr := ptrBool(falseVal)
		if falsePtr == nil {
			t.Fatal("Expected non-nil pointer")
		}
		if *falsePtr != false {
			t.Error("Expected false, got true")
		}
	})

	t.Run("Contains", func(t *testing.T) {
		// Test substring exists
		if !contains("hello world", "world") {
			t.Error("Expected contains to find 'world' in 'hello world'")
		}

		// Test substring doesn't exist
		if contains("hello world", "foo") {
			t.Error("Expected contains to not find 'foo' in 'hello world'")
		}

		// Test empty substring
		if !contains("hello", "") {
			t.Error("Expected contains to find empty string")
		}

		// Test case sensitivity
		if contains("hello", "HELLO") {
			t.Error("Expected contains to be case-sensitive")
		}
	})

	t.Run("IndexOf", func(t *testing.T) {
		// Test substring exists
		if indexOf("hello world", "world") != 6 {
			t.Error("Expected indexOf to find 'world' at index 6")
		}

		// Test substring doesn't exist
		if indexOf("hello world", "foo") != -1 {
			t.Error("Expected indexOf to return -1 for missing substring")
		}

		// Test empty substring
		if indexOf("hello", "") != 0 {
			t.Error("Expected indexOf to find empty string at index 0")
		}

		// Test first character
		if indexOf("hello", "h") != 0 {
			t.Error("Expected indexOf to find first character at index 0")
		}

		// Test substring at end
		if indexOf("hello", "o") != 4 {
			t.Error("Expected indexOf to find 'o' at index 4")
		}
	})
}

// TestMakeHTTPRequest tests the HTTP request helper
func TestMakeHTTPRequest(t *testing.T) {
	server := &APIServer{}
	router := mux.NewRouter()

	// Add a test handler
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}).Methods("GET")

	router.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Echo back the request body
		w.Write([]byte(`{"echoed": true}`))
	}).Methods("POST")

	t.Run("SimpleGETRequest", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if !contains(w.Body.String(), "success") {
			t.Errorf("Expected 'success' in response, got: %s", w.Body.String())
		}
	})

	t.Run("POSTWithJSONBody", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/echo",
			Body: map[string]interface{}{
				"test": "data",
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("POSTWithStringBody", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "POST",
			Path:   "/echo",
			Body:   `{"test": "string"}`,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WithCustomHeaders", func(t *testing.T) {
		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WithURLVars", func(t *testing.T) {
		// Add a route with URL parameters
		router.HandleFunc("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": "` + vars["id"] + `"}`))
		}).Methods("GET")

		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method:  "GET",
			Path:    "/items/123",
			URLVars: map[string]string{"id": "123"},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if !contains(w.Body.String(), "123") {
			t.Errorf("Expected '123' in response, got: %s", w.Body.String())
		}
	})
}

// TestAssertJSONResponse tests the JSON response assertion helper
func TestAssertJSONResponse(t *testing.T) {
	t.Run("ValidJSONResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key": "value", "number": 42}`))

		result := assertJSONResponse(t, w, http.StatusOK)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if result["key"] != "value" {
			t.Errorf("Expected key='value', got %v", result["key"])
		}

		if result["number"] != float64(42) {
			t.Errorf("Expected number=42, got %v", result["number"])
		}
	})

	t.Run("EmptyJSONObject", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))

		result := assertJSONResponse(t, w, http.StatusOK)

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		if len(result) != 0 {
			t.Errorf("Expected empty object, got %d keys", len(result))
		}
	})

	t.Run("NonSuccessStatus", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))

		// This should not parse JSON for error responses
		result := assertJSONResponse(t, w, http.StatusBadRequest)

		if result != nil {
			t.Error("Expected nil result for error status")
		}
	})
}

// TestAssertErrorResponse tests the error response assertion helper
func TestAssertErrorResponse(t *testing.T) {
	t.Run("ErrorWithMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid input"))

		// Should not panic
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})

	t.Run("ErrorWithoutMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))

		// Should not panic when no expected message
		assertErrorResponse(t, w, http.StatusNotFound, "")
	})

	t.Run("EmptyErrorBody", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusInternalServerError)

		// Should not panic with empty body
		assertErrorResponse(t, w, http.StatusInternalServerError, "")
	})
}

// TestSetupTestLogger tests the logger setup
func TestSetupTestLogger(t *testing.T) {
	t.Run("LoggerCleanup", func(t *testing.T) {
		cleanup := setupTestLogger()
		if cleanup == nil {
			t.Fatal("Expected cleanup function")
		}

		// Should be able to call cleanup
		cleanup()
	})
}

// TestHTTPTestRequest validates the test request struct
func TestHTTPTestRequest(t *testing.T) {
	t.Run("CreateRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
			Body:   nil,
			Headers: map[string]string{
				"Authorization": "Bearer token",
			},
			URLVars: map[string]string{
				"id": "123",
			},
		}

		if req.Method != "GET" {
			t.Errorf("Expected Method=GET, got %s", req.Method)
		}
		if req.Path != "/test" {
			t.Errorf("Expected Path=/test, got %s", req.Path)
		}
		if len(req.Headers) != 1 {
			t.Errorf("Expected 1 header, got %d", len(req.Headers))
		}
		if len(req.URLVars) != 1 {
			t.Errorf("Expected 1 URL var, got %d", len(req.URLVars))
		}
	})
}
