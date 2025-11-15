package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		statusCode int
	}{
		{
			name:       "bad request",
			message:    "Invalid input",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "internal server error",
			message:    "Database connection failed",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "not found",
			message:    "Resource not found",
			statusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, tt.message, tt.statusCode)

			if w.Code != tt.statusCode {
				t.Errorf("expected status code %d, got %d", tt.statusCode, w.Code)
			}

			body := w.Body.String()
			if body == "" {
				t.Error("expected non-empty response body")
			}
		})
	}
}

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		data       interface{}
		wantStatus int
	}{
		{
			name: "simple map",
			data: map[string]string{
				"message": "success",
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "struct",
			data: struct {
				ID    int    `json:"id"`
				Title string `json:"title"`
			}{
				ID:    123,
				Title: "Test",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "nil data",
			data:       nil,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondJSON(w, tt.wantStatus, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status code %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
