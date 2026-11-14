package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/administration"
)

func TestDecodeJSONBody_ValidJSON(t *testing.T) {
	var dst struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	body := bytes.NewBufferString(`{"name": "test", "value": 42}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	ok := decodeJSONBody(w, req, &dst)

	if !ok {
		t.Error("Expected decodeJSONBody to return true for valid JSON")
	}
	if dst.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", dst.Name)
	}
	if dst.Value != 42 {
		t.Errorf("Expected value 42, got %d", dst.Value)
	}
}

func TestDecodeJSONBody_InvalidJSON(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	ok := decodeJSONBody(w, req, &dst)

	if ok {
		t.Error("Expected decodeJSONBody to return false for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.ErrorType != ApiErrorTypeValidation {
		t.Errorf("Expected error type '%s', got '%s'", ApiErrorTypeValidation, errResp.ErrorType)
	}
}

func TestDecodeJSONBody_EmptyBody(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}

	body := bytes.NewBufferString(``)
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	w := httptest.NewRecorder()

	ok := decodeJSONBody(w, req, &dst)

	if ok {
		t.Error("Expected decodeJSONBody to return false for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRequireAuth_Authenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	claims := &administration.UserClaims{Email: "test@example.com"}
	ctx := context.WithValue(req.Context(), userClaimsKey, claims)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	email, ok := requireAuth(w, req)

	if !ok {
		t.Error("Expected requireAuth to return true for authenticated request")
	}
	if email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", email)
	}
}

func TestRequireAuth_Unauthenticated(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	email, ok := requireAuth(w, req)

	if ok {
		t.Error("Expected requireAuth to return false for unauthenticated request")
	}
	if email != "" {
		t.Errorf("Expected empty email, got '%s'", email)
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.ErrorType != ApiErrorTypeUnauthorized {
		t.Errorf("Expected error type '%s', got '%s'", ApiErrorTypeUnauthorized, errResp.ErrorType)
	}
}

func TestGetPathParam_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	val, ok := getPathParam(req, "id")

	if !ok {
		t.Error("Expected getPathParam to return true when param is present")
	}
	if val != "123" {
		t.Errorf("Expected '123', got '%s'", val)
	}
}

func TestGetPathParam_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req = mux.SetURLVars(req, map[string]string{})

	val, ok := getPathParam(req, "id")

	if ok {
		t.Error("Expected getPathParam to return false when param is missing")
	}
	if val != "" {
		t.Errorf("Expected empty string, got '%s'", val)
	}
}

func TestGetPathParamInt64_Valid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "12345"})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt64(w, req, "id")

	if !ok {
		t.Error("Expected getPathParamInt64 to return true for valid int")
	}
	if val != 12345 {
		t.Errorf("Expected 12345, got %d", val)
	}
}

func TestGetPathParamInt64_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt64(w, req, "id")

	if ok {
		t.Error("Expected getPathParamInt64 to return false for invalid int")
	}
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetPathParamInt64_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req = mux.SetURLVars(req, map[string]string{})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt64(w, req, "id")

	if ok {
		t.Error("Expected getPathParamInt64 to return false when param is missing")
	}
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetPathParamInt64_Negative(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/users/-5", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "-5"})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt64(w, req, "id")

	if !ok {
		t.Error("Expected getPathParamInt64 to return true for negative int")
	}
	if val != -5 {
		t.Errorf("Expected -5, got %d", val)
	}
}

func TestGetPathParamInt_Valid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "42"})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt(w, req, "id")

	if !ok {
		t.Error("Expected getPathParamInt to return true for valid int")
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestGetPathParamInt_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/items/xyz", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	w := httptest.NewRecorder()

	val, ok := getPathParamInt(w, req, "id")

	if ok {
		t.Error("Expected getPathParamInt to return false for invalid int")
	}
	if val != 0 {
		t.Errorf("Expected 0, got %d", val)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRequireQueryParam_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=hello", nil)
	w := httptest.NewRecorder()

	val, ok := requireQueryParam(w, req, "q")

	if !ok {
		t.Error("Expected requireQueryParam to return true when param is present")
	}
	if val != "hello" {
		t.Errorf("Expected 'hello', got '%s'", val)
	}
}

func TestRequireQueryParam_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()

	val, ok := requireQueryParam(w, req, "q")

	if ok {
		t.Error("Expected requireQueryParam to return false when param is missing")
	}
	if val != "" {
		t.Errorf("Expected empty string, got '%s'", val)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.ErrorType != ApiErrorTypeValidation {
		t.Errorf("Expected error type '%s', got '%s'", ApiErrorTypeValidation, errResp.ErrorType)
	}
}

func TestRequireQueryParam_Empty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?q=", nil)
	w := httptest.NewRecorder()

	val, ok := requireQueryParam(w, req, "q")

	if ok {
		t.Error("Expected requireQueryParam to return false for empty param")
	}
	if val != "" {
		t.Errorf("Expected empty string, got '%s'", val)
	}
}

func TestGetQueryParam_Present(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?category=books", nil)

	val := getQueryParam(req, "category")

	if val != "books" {
		t.Errorf("Expected 'books', got '%s'", val)
	}
}

func TestGetQueryParam_Missing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search", nil)

	val := getQueryParam(req, "category")

	if val != "" {
		t.Errorf("Expected empty string, got '%s'", val)
	}
}

func TestWriteJSONSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSONSuccess(w, "Operation completed")

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if resp["message"] != "Operation completed" {
		t.Errorf("Expected message='Operation completed', got %v", resp["message"])
	}
}

func TestWriteJSONSuccessData(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"id":   123,
		"name": "test",
	}
	writeJSONSuccessData(w, data)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["id"].(float64) != 123 {
		t.Errorf("Expected id=123, got %v", resp["id"])
	}
	if resp["name"] != "test" {
		t.Errorf("Expected name='test', got %v", resp["name"])
	}
}

func TestWriteJSONSuccessSimple(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSONSuccessSimple(w)

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("Expected success=true, got %v", resp["success"])
	}
	if _, hasMessage := resp["message"]; hasMessage {
		t.Error("Expected no message field in simple success response")
	}
}
