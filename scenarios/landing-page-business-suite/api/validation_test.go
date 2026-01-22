package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateEmail_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "test@example.com", "test@example.com"},
		{"with whitespace", "  Test@Example.COM  ", "test@example.com"},
		{"uppercase", "TEST@EXAMPLE.COM", "test@example.com"},
		{"subdomain", "user@mail.example.com", "user@mail.example.com"},
		{"plus addressing", "user+tag@example.com", "user+tag@example.com"},
		{"dot in local", "user.name@example.com", "user.name@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateEmail(tt.input)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr error
	}{
		{"empty", "", ErrEmailRequired},
		{"whitespace only", "   ", ErrEmailRequired},
		{"no at sign", "invalidemail", ErrEmailInvalid},
		{"no domain", "user@", ErrEmailInvalid},
		{"no local part", "@example.com", ErrEmailInvalid},
		{"double at", "user@@example.com", ErrEmailInvalid},
		{"spaces in email", "user @example.com", ErrEmailInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateEmail(tt.input)
			if err == nil {
				t.Error("Expected an error, got nil")
				return
			}
			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error %v, got %v", tt.expectedErr, err)
			}
		})
	}
}

func TestValidateEmailForHandler_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	email, ok := ValidateEmailForHandler(w, "  Test@Example.COM  ")

	if !ok {
		t.Error("Expected ValidateEmailForHandler to return true for valid email")
	}
	if email != "test@example.com" {
		t.Errorf("Expected 'test@example.com', got '%s'", email)
	}
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestValidateEmailForHandler_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	email, ok := ValidateEmailForHandler(w, "")

	if ok {
		t.Error("Expected ValidateEmailForHandler to return false for empty email")
	}
	if email != "" {
		t.Errorf("Expected empty string, got '%s'", email)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error != "Email is required" {
		t.Errorf("Expected error message 'Email is required', got '%s'", errResp.Error)
	}
}

func TestValidateEmailForHandler_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	email, ok := ValidateEmailForHandler(w, "invalidemail")

	if ok {
		t.Error("Expected ValidateEmailForHandler to return false for invalid email")
	}
	if email != "" {
		t.Errorf("Expected empty string, got '%s'", email)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error != "Invalid email format" {
		t.Errorf("Expected error message 'Invalid email format', got '%s'", errResp.Error)
	}
}

func TestRequireNonEmpty_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	value, ok := RequireNonEmpty(w, "  hello  ", "name")

	if !ok {
		t.Error("Expected RequireNonEmpty to return true for non-empty value")
	}
	if value != "hello" {
		t.Errorf("Expected 'hello', got '%s'", value)
	}
}

func TestRequireNonEmpty_Empty(t *testing.T) {
	w := httptest.NewRecorder()
	value, ok := RequireNonEmpty(w, "", "name")

	if ok {
		t.Error("Expected RequireNonEmpty to return false for empty value")
	}
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResp.Error != "name is required" {
		t.Errorf("Expected error message 'name is required', got '%s'", errResp.Error)
	}
}

func TestRequireNonEmpty_WhitespaceOnly(t *testing.T) {
	w := httptest.NewRecorder()
	value, ok := RequireNonEmpty(w, "   ", "name")

	if ok {
		t.Error("Expected RequireNonEmpty to return false for whitespace-only value")
	}
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
}

func TestValidateNonEmpty_Valid(t *testing.T) {
	err := ValidateNonEmpty("hello", "name")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateNonEmpty_Empty(t *testing.T) {
	err := ValidateNonEmpty("", "name")
	if err == nil {
		t.Error("Expected an error for empty value")
	}
	if err.Error() != "name is required" {
		t.Errorf("Expected 'name is required', got '%s'", err.Error())
	}
}

func TestValidateNonEmpty_WhitespaceOnly(t *testing.T) {
	err := ValidateNonEmpty("   ", "name")
	if err == nil {
		t.Error("Expected an error for whitespace-only value")
	}
}

func TestValidateInList_Valid(t *testing.T) {
	allowed := []string{"apple", "banana", "cherry"}
	err := ValidateInList("banana", "fruit", allowed)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateInList_Invalid(t *testing.T) {
	allowed := []string{"apple", "banana", "cherry"}
	err := ValidateInList("orange", "fruit", allowed)
	if err == nil {
		t.Error("Expected an error for value not in list")
	}
}

func TestValidateInList_EmptyList(t *testing.T) {
	err := ValidateInList("anything", "field", []string{})
	if err == nil {
		t.Error("Expected an error when list is empty")
	}
}

func TestValidateInList_CaseSensitive(t *testing.T) {
	allowed := []string{"Apple", "Banana"}
	err := ValidateInList("apple", "fruit", allowed)
	if err == nil {
		t.Error("Expected an error for case-mismatched value")
	}
}
