package main

import (
	"strings"
	"testing"
)

func strptr(s string) *string { return &s }
func i64ptr(v int64) *int64   { return &v }

func TestParseExecutionOptions(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		delay       int64
		operation   string
		startedBy   string
		requireMode bool
		want        executionOptions
		wantErr     string
	}{
		{
			name:      "valid manual generator",
			mode:      "Manual",
			operation: "Generator",
			startedBy: "  me  ",
			want:      executionOptions{mode: "manual", delaySeconds: 0, operation: "generator", startedBy: "me"},
		},
		{
			name:      "scheduled with delay improver",
			mode:      "scheduled",
			delay:     30,
			operation: "improver",
			startedBy: "swarm-manager",
			want:      executionOptions{mode: "scheduled", delaySeconds: 30, operation: "improver", startedBy: "swarm-manager"},
		},
		{
			name:      "empty mode allowed when not required",
			mode:      "",
			operation: "generator",
			want:      executionOptions{mode: "", operation: "generator"},
		},
		{
			name:        "empty mode rejected when required",
			mode:        "",
			operation:   "generator",
			requireMode: true,
			wantErr:     "mode is required",
		},
		{
			name:      "invalid mode",
			mode:      "turbo",
			operation: "generator",
			wantErr:   `invalid mode "turbo"`,
		},
		{
			name:      "invalid operation",
			mode:      "manual",
			operation: "destroyer",
			wantErr:   `invalid operation "destroyer"`,
		},
		{
			name:      "negative delay",
			mode:      "scheduled",
			delay:     -1,
			operation: "generator",
			wantErr:   "delay-seconds must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExecutionOptions(strptr(tt.mode), i64ptr(tt.delay), strptr(tt.operation), strptr(tt.startedBy), tt.requireMode)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParsePolicyOptions(t *testing.T) {
	// Policy options always require mode and default operation to generator.
	got, err := parsePolicyOptions(strptr("yolo"), i64ptr(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.mode != "yolo" || got.operation != "generator" || got.startedBy != "swarm-manager" {
		t.Errorf("got %+v, want mode=yolo operation=generator startedBy=swarm-manager", got)
	}

	if _, err := parsePolicyOptions(strptr(""), i64ptr(0)); err == nil {
		t.Error("expected error for missing mode in policy options")
	}
	if _, err := parsePolicyOptions(strptr("nope"), i64ptr(0)); err == nil {
		t.Error("expected error for invalid mode in policy options")
	}
}
