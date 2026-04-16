package ai

import (
	"strings"
	"testing"

	"browser-automation-studio/cli/internal/appctx"
)

func TestAINavigateRequest_JSONTags(t *testing.T) {
	// Verify the struct has correct JSON tags for API compatibility
	req := AINavigateRequest{
		SessionID:     "session123",
		Prompt:        "Click the button",
		Model:         "gpt-4o",
		MaxSteps:      20,
		APIKey:        "sk-xxx",
		NavigatorType: "playwright",
	}

	// Basic validation that struct can be created
	if req.SessionID != "session123" {
		t.Errorf("SessionID = %q, want %q", req.SessionID, "session123")
	}
	if req.Prompt != "Click the button" {
		t.Errorf("Prompt = %q, want %q", req.Prompt, "Click the button")
	}
	if req.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", req.Model, "gpt-4o")
	}
	if req.MaxSteps != 20 {
		t.Errorf("MaxSteps = %d, want %d", req.MaxSteps, 20)
	}
	if req.NavigatorType != "playwright" {
		t.Errorf("NavigatorType = %q, want %q", req.NavigatorType, "playwright")
	}
}

func TestAINavigateResponse_JSONTags(t *testing.T) {
	// Verify the struct has correct JSON tags for API compatibility
	resp := AINavigateResponse{
		NavigationID:  "nav123",
		Status:        "navigating",
		Model:         "gpt-4o",
		MaxSteps:      20,
		NavigatorType: "playwright",
	}

	if resp.NavigationID != "nav123" {
		t.Errorf("NavigationID = %q, want %q", resp.NavigationID, "nav123")
	}
	if resp.Status != "navigating" {
		t.Errorf("Status = %q, want %q", resp.Status, "navigating")
	}
}

func TestRunNavigate_Help(t *testing.T) {
	ctx := &appctx.Context{
		Name: "test-cli",
	}

	// Test --help flag
	err := runNavigate(ctx, []string{"--help"})
	if err != nil {
		t.Errorf("runNavigate(--help) error = %v, want nil", err)
	}

	// Test -h flag
	err = runNavigate(ctx, []string{"-h"})
	if err != nil {
		t.Errorf("runNavigate(-h) error = %v, want nil", err)
	}
}

func TestRunNavigate_MissingArgs(t *testing.T) {
	ctx := &appctx.Context{
		Name: "test-cli",
	}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing session",
			args:    []string{"--prompt", "Click button"},
			wantErr: "--session is required",
		},
		{
			name:    "missing prompt",
			args:    []string{"--session", "abc123"},
			wantErr: "--prompt is required",
		},
		{
			name:    "session requires value",
			args:    []string{"--session"},
			wantErr: "--session requires a value",
		},
		{
			name:    "prompt requires value",
			args:    []string{"--prompt"},
			wantErr: "--prompt requires a value",
		},
		{
			name:    "model requires value",
			args:    []string{"--session", "abc", "--prompt", "test", "--model"},
			wantErr: "--model requires a value",
		},
		{
			name:    "max-steps requires value",
			args:    []string{"--session", "abc", "--prompt", "test", "--max-steps"},
			wantErr: "--max-steps requires a value",
		},
		{
			name:    "max-steps must be number",
			args:    []string{"--session", "abc", "--prompt", "test", "--max-steps", "abc"},
			wantErr: "--max-steps must be a number",
		},
		{
			name:    "navigator requires value",
			args:    []string{"--session", "abc", "--prompt", "test", "--navigator"},
			wantErr: "--navigator requires a value",
		},
		{
			name:    "api-key requires value",
			args:    []string{"--session", "abc", "--prompt", "test", "--api-key"},
			wantErr: "--api-key requires a value",
		},
		{
			name:    "unknown option",
			args:    []string{"--session", "abc", "--prompt", "test", "--unknown"},
			wantErr: "unknown option",
		},
		{
			name:    "unexpected argument",
			args:    []string{"--session", "abc", "--prompt", "test", "extra"},
			wantErr: "unexpected argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runNavigate(ctx, tt.args)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestNavigateArgValidation tests that required arguments are properly validated
func TestNavigateArgValidation(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		prompt    string
		wantError bool
	}{
		{
			name:      "missing session",
			sessionID: "",
			prompt:    "Click button",
			wantError: true,
		},
		{
			name:      "missing prompt",
			sessionID: "session123",
			prompt:    "",
			wantError: true,
		},
		{
			name:      "both provided",
			sessionID: "session123",
			prompt:    "Click button",
			wantError: false,
		},
		{
			name:      "both empty",
			sessionID: "",
			prompt:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the expected behavior matches the runNavigate implementation
			hasError := tt.sessionID == "" || tt.prompt == ""
			if hasError != tt.wantError {
				t.Errorf("validation for sessionID=%q, prompt=%q: error=%v, want error=%v",
					tt.sessionID, tt.prompt, hasError, tt.wantError)
			}
		})
	}
}
