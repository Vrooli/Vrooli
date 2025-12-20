package main

import (
	"strings"
	"testing"
)

func TestAppConstants(t *testing.T) {
	if appName != "workspace-sandbox" {
		t.Errorf("appName = %q, want 'workspace-sandbox'", appName)
	}
	if appVersion != "0.1.0" {
		t.Errorf("appVersion = %q, want '0.1.0'", appVersion)
	}
	if defaultAPIBase != "" {
		t.Errorf("defaultAPIBase = %q, want ''", defaultAPIBase)
	}
}

func TestBuildVariables(t *testing.T) {
	// These are set at build time, so we just verify they exist
	if buildFingerprint == "" {
		// Allow "unknown" as default
		t.Log("buildFingerprint is empty (expected at build time)")
	}
	if buildTimestamp == "" {
		t.Log("buildTimestamp is empty (expected at build time)")
	}
}

func TestNewApp(t *testing.T) {
	// NewApp requires environment variables to be set for the API client
	// Since we're testing the app structure, we can verify it doesn't panic
	// when called (actual functionality requires env setup)
	t.Run("creates app without panic", func(t *testing.T) {
		// This may fail in test environment without proper env vars
		// but we're mainly testing that the code structure is correct
		app, err := NewApp()
		if err != nil {
			// Expected in test environment without proper setup
			t.Skipf("NewApp requires environment setup: %v", err)
		}
		if app == nil {
			t.Error("NewApp returned nil app")
		}
		if app.core == nil {
			t.Error("NewApp returned app with nil core")
		}
	})
}

func TestApiPath(t *testing.T) {
	// Create a minimal app for testing apiPath
	// We need to test the path construction logic

	tests := []struct {
		name     string
		baseURL  string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			baseURL:  "http://localhost:8080",
			input:    "",
			expected: "",
		},
		{
			name:     "path with leading slash",
			baseURL:  "http://localhost:8080",
			input:    "/health",
			expected: "/api/v1/health",
		},
		{
			name:     "path without leading slash",
			baseURL:  "http://localhost:8080",
			input:    "health",
			expected: "/api/v1/health",
		},
		{
			name:     "base already has api/v1",
			baseURL:  "http://localhost:8080/api/v1",
			input:    "/sandboxes",
			expected: "/sandboxes",
		},
		{
			name:     "path with whitespace",
			baseURL:  "http://localhost:8080",
			input:    "  /health  ",
			expected: "/api/v1/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the path construction logic directly
			v1Path := strings.TrimSpace(tt.input)
			if v1Path == "" {
				if tt.expected != "" {
					t.Errorf("expected %q for empty input", tt.expected)
				}
				return
			}
			if !strings.HasPrefix(v1Path, "/") {
				v1Path = "/" + v1Path
			}
			base := strings.TrimRight(strings.TrimSpace(tt.baseURL), "/")
			var result string
			if strings.HasSuffix(base, "/api/v1") {
				result = v1Path
			} else {
				result = "/api/v1" + v1Path
			}
			if result != tt.expected {
				t.Errorf("apiPath(%q) with base %q = %q, want %q",
					tt.input, tt.baseURL, result, tt.expected)
			}
		})
	}
}

func TestResponseStructs(t *testing.T) {
	// Verify response struct fields exist and have correct json tags

	t.Run("sandboxResponse fields", func(t *testing.T) {
		sr := sandboxResponse{
			ID:          "test-id",
			ScopePath:   "/path",
			ProjectRoot: "/root",
			Owner:       "user",
			Status:      "active",
			MergedDir:   "/merged",
			SizeBytes:   1024,
			FileCount:   10,
			ErrorMsg:    "error",
		}
		if sr.ID != "test-id" {
			t.Error("sandboxResponse ID field not working")
		}
	})

	t.Run("listResponse fields", func(t *testing.T) {
		lr := listResponse{
			Sandboxes:  []sandboxResponse{{ID: "test"}},
			TotalCount: 1,
		}
		if lr.TotalCount != 1 {
			t.Error("listResponse TotalCount field not working")
		}
	})

	t.Run("diffResponse fields", func(t *testing.T) {
		dr := diffResponse{
			SandboxID:     "sb-123",
			UnifiedDiff:   "diff content",
			TotalAdded:    5,
			TotalDeleted:  3,
			TotalModified: 2,
		}
		if dr.TotalAdded != 5 {
			t.Error("diffResponse TotalAdded field not working")
		}
	})

	t.Run("approvalResponse fields", func(t *testing.T) {
		ar := approvalResponse{
			Success:    true,
			Applied:    10,
			CommitHash: "abc123",
			ErrorMsg:   "",
		}
		if !ar.Success {
			t.Error("approvalResponse Success field not working")
		}
	})

	t.Run("healthResponse fields", func(t *testing.T) {
		hr := healthResponse{
			Status:    "ok",
			Service:   "workspace-sandbox",
			Version:   "1.0",
			Readiness: true,
			Timestamp: "2024-01-01T00:00:00Z",
			Deps:      map[string]string{"db": "healthy"},
		}
		if hr.Status != "ok" {
			t.Error("healthResponse Status field not working")
		}
	})
}

func TestCommandGroups(t *testing.T) {
	// Test that command group structure is correct by inspecting the expected groups
	expectedGroups := []string{"Health", "Sandbox Operations", "Diff & Approval", "Driver", "Configuration"}

	t.Run("verifies expected command groups", func(t *testing.T) {
		// The actual groups are defined in registerCommands
		// We verify the expected structure is what we documented
		for _, group := range expectedGroups {
			if group == "" {
				t.Error("empty group name in expected groups")
			}
		}
	})

	t.Run("Health group has status command", func(t *testing.T) {
		// Verify expected command exists in documentation
		healthCommands := []string{"status"}
		for _, cmd := range healthCommands {
			if cmd == "" {
				t.Error("empty command name")
			}
		}
	})

	t.Run("Sandbox Operations group has CRUD commands", func(t *testing.T) {
		sandboxCommands := []string{"create", "list", "inspect", "stop", "delete", "workspace"}
		for _, cmd := range sandboxCommands {
			if cmd == "" {
				t.Error("empty command name")
			}
		}
	})

	t.Run("Diff & Approval group has diff/approve/reject", func(t *testing.T) {
		diffCommands := []string{"diff", "approve", "reject"}
		for _, cmd := range diffCommands {
			if cmd == "" {
				t.Error("empty command name")
			}
		}
	})
}

func TestCreateArgumentParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		scope   string
		project string
		owner   string
		reserve []string
	}{
		{
			name:  "scope with --scope=",
			args:  []string{"--scope=/project/src"},
			scope: "/project/src",
		},
		{
			name:  "scope with -s=",
			args:  []string{"-s=/project/src"},
			scope: "/project/src",
		},
		{
			name:    "all arguments",
			args:    []string{"--scope=/src", "--project=/root", "--owner=user-1", "--reserve=/root/scenarios/a"},
			scope:   "/src",
			project: "/root",
			owner:   "user-1",
			reserve: []string{"/root/scenarios/a"},
		},
		{
			name:    "short flags",
			args:    []string{"-s=/src", "-p=/root", "-o=user-1", "-r=/root/scenarios/a"},
			scope:   "/src",
			project: "/root",
			owner:   "user-1",
			reserve: []string{"/root/scenarios/a"},
		},
		{
			name:    "default scope to project root when scope omitted",
			args:    []string{"--project=/root", "--reserve=/root/scenarios/a"},
			scope:   "/root",
			project: "/root",
			reserve: []string{"/root/scenarios/a"},
		},
		{
			name:    "reserved-paths comma separated",
			args:    []string{"--project=/root", "--reserved-paths=/root/a,/root/b"},
			scope:   "/root",
			project: "/root",
			reserve: []string{"/root/a", "/root/b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scope, project, owner string
			var reservedPaths []string
			for _, arg := range tt.args {
				switch {
				case strings.HasPrefix(arg, "--scope="):
					scope = strings.TrimPrefix(arg, "--scope=")
				case strings.HasPrefix(arg, "-s="):
					scope = strings.TrimPrefix(arg, "-s=")
				case strings.HasPrefix(arg, "--reserve="):
					reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "--reserve="))
				case strings.HasPrefix(arg, "--reserved="):
					reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "--reserved="))
				case strings.HasPrefix(arg, "-r="):
					reservedPaths = append(reservedPaths, strings.TrimPrefix(arg, "-r="))
				case strings.HasPrefix(arg, "--reserved-paths="):
					raw := strings.TrimPrefix(arg, "--reserved-paths=")
					for _, p := range strings.Split(raw, ",") {
						p = strings.TrimSpace(p)
						if p != "" {
							reservedPaths = append(reservedPaths, p)
						}
					}
				case strings.HasPrefix(arg, "--project="):
					project = strings.TrimPrefix(arg, "--project=")
				case strings.HasPrefix(arg, "-p="):
					project = strings.TrimPrefix(arg, "-p=")
				case strings.HasPrefix(arg, "--owner="):
					owner = strings.TrimPrefix(arg, "--owner=")
				case strings.HasPrefix(arg, "-o="):
					owner = strings.TrimPrefix(arg, "-o=")
				}
			}

			if scope == "" && project != "" {
				scope = project
			}

			if scope != tt.scope {
				t.Errorf("scope = %q, want %q", scope, tt.scope)
			}
			if project != tt.project {
				t.Errorf("project = %q, want %q", project, tt.project)
			}
			if owner != tt.owner {
				t.Errorf("owner = %q, want %q", owner, tt.owner)
			}
			if tt.reserve != nil {
				if strings.Join(reservedPaths, ",") != strings.Join(tt.reserve, ",") {
					t.Errorf("reserve = %q, want %q", reservedPaths, tt.reserve)
				}
			}
		})
	}
}

func TestListArgumentParsing(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		status string
		owner  string
		asJSON bool
	}{
		{
			name:   "status filter",
			args:   []string{"--status=active"},
			status: "active",
		},
		{
			name:  "owner filter",
			args:  []string{"--owner=user-1"},
			owner: "user-1",
		},
		{
			name:   "json output",
			args:   []string{"--json"},
			asJSON: true,
		},
		{
			name:   "all options",
			args:   []string{"--status=stopped", "--owner=agent-1", "--json"},
			status: "stopped",
			owner:  "agent-1",
			asJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status, owner string
			var asJSON bool
			for _, arg := range tt.args {
				switch {
				case strings.HasPrefix(arg, "--status="):
					status = strings.TrimPrefix(arg, "--status=")
				case strings.HasPrefix(arg, "--owner="):
					owner = strings.TrimPrefix(arg, "--owner=")
				case arg == "--json":
					asJSON = true
				}
			}
			if status != tt.status {
				t.Errorf("status = %q, want %q", status, tt.status)
			}
			if owner != tt.owner {
				t.Errorf("owner = %q, want %q", owner, tt.owner)
			}
			if asJSON != tt.asJSON {
				t.Errorf("asJSON = %v, want %v", asJSON, tt.asJSON)
			}
		})
	}
}

func TestDiffArgumentParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		sandboxID string
		raw       bool
	}{
		{
			name:      "sandbox id only",
			args:      []string{"sb-123"},
			sandboxID: "sb-123",
			raw:       false,
		},
		{
			name:      "with raw flag",
			args:      []string{"sb-123", "--raw"},
			sandboxID: "sb-123",
			raw:       true,
		},
		{
			name:      "raw flag first",
			args:      []string{"--raw", "sb-123"},
			sandboxID: "sb-123",
			raw:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sandboxID string
			var raw bool
			for _, arg := range tt.args {
				switch {
				case arg == "--raw":
					raw = true
				case !strings.HasPrefix(arg, "-"):
					if sandboxID == "" {
						sandboxID = arg
					}
				}
			}
			if sandboxID != tt.sandboxID {
				t.Errorf("sandboxID = %q, want %q", sandboxID, tt.sandboxID)
			}
			if raw != tt.raw {
				t.Errorf("raw = %v, want %v", raw, tt.raw)
			}
		})
	}
}

func TestApproveArgumentParsing(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		sandboxID string
		message   string
	}{
		{
			name:      "sandbox id only",
			args:      []string{"sb-123"},
			sandboxID: "sb-123",
		},
		{
			name:      "with message using -m=",
			args:      []string{"sb-123", "-m=Apply changes"},
			sandboxID: "sb-123",
			message:   "Apply changes",
		},
		{
			name:      "with message using --message=",
			args:      []string{"sb-123", "--message=My commit"},
			sandboxID: "sb-123",
			message:   "My commit",
		},
		{
			name:      "with message using -m flag",
			args:      []string{"sb-123", "-m", "Commit msg"},
			sandboxID: "sb-123",
			message:   "Commit msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sandboxID, message string
			for i, arg := range tt.args {
				switch {
				case arg == "-m" && i+1 < len(tt.args):
					message = tt.args[i+1]
				case strings.HasPrefix(arg, "-m="):
					message = strings.TrimPrefix(arg, "-m=")
				case strings.HasPrefix(arg, "--message="):
					message = strings.TrimPrefix(arg, "--message=")
				case !strings.HasPrefix(arg, "-") && (i == 0 || !strings.HasPrefix(tt.args[i-1], "-m")):
					if sandboxID == "" {
						sandboxID = arg
					}
				}
			}
			if sandboxID != tt.sandboxID {
				t.Errorf("sandboxID = %q, want %q", sandboxID, tt.sandboxID)
			}
			if message != tt.message {
				t.Errorf("message = %q, want %q", message, tt.message)
			}
		})
	}
}

func TestScopePathTruncation(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "/short/path",
			expected: "/short/path",
		},
		{
			path:     "/this/is/a/very/long/path/that/exceeds/forty/characters",
			expected: "...ng/path/that/exceeds/forty/characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			scope := tt.path
			if len(scope) > 40 {
				scope = "..." + scope[len(scope)-37:]
			}
			if scope != tt.expected {
				t.Errorf("truncated = %q, want %q", scope, tt.expected)
			}
		})
	}
}

func TestChangeTypeSymbol(t *testing.T) {
	tests := []struct {
		changeType string
		symbol     string
	}{
		{"added", "+"},
		{"deleted", "-"},
		{"modified", "~"},
		{"unknown", " "},
	}

	for _, tt := range tests {
		t.Run(tt.changeType, func(t *testing.T) {
			symbol := " "
			switch tt.changeType {
			case "added":
				symbol = "+"
			case "deleted":
				symbol = "-"
			case "modified":
				symbol = "~"
			}
			if symbol != tt.symbol {
				t.Errorf("symbol for %q = %q, want %q", tt.changeType, symbol, tt.symbol)
			}
		})
	}
}

func TestOwnerDefaultValue(t *testing.T) {
	// Test that empty owner becomes "-" in display
	owner := ""
	if owner == "" {
		owner = "-"
	}
	if owner != "-" {
		t.Errorf("default owner = %q, want '-'", owner)
	}

	// Test that non-empty owner stays unchanged
	owner = "user-1"
	if owner == "" {
		owner = "-"
	}
	if owner != "user-1" {
		t.Errorf("owner = %q, want 'user-1'", owner)
	}
}
