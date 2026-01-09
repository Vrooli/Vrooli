package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestHandleListScenarioTemplates(t *testing.T) {
	// Save and restore original VROOLI_ROOT
	origRoot := os.Getenv("VROOLI_ROOT")
	defer func() {
		if origRoot != "" {
			os.Setenv("VROOLI_ROOT", origRoot)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
	}()

	// Create temporary vrooli root with templates
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)

	templatesDir := filepath.Join(tmpRoot, "scripts", "scenarios", "templates")
	testTemplateDir := filepath.Join(templatesDir, "test-template")
	if err := os.MkdirAll(testTemplateDir, 0755); err != nil {
		t.Fatalf("Failed to create template dir: %v", err)
	}

	// Create a valid template.json
	templateJSON := `{
		"name": "test-template",
		"displayName": "Test Template",
		"description": "A test scenario template",
		"stack": ["Go", "React"],
		"requiredVars": {
			"port": {
				"flag": "--port",
				"description": "API port",
				"default": "8080"
			}
		},
		"optionalVars": {
			"debug": {
				"flag": "--debug",
				"description": "Enable debug mode",
				"default": "false"
			}
		}
	}`

	manifestPath := filepath.Join(testTemplateDir, "template.json")
	if err := os.WriteFile(manifestPath, []byte(templateJSON), 0644); err != nil {
		t.Fatalf("Failed to create template.json: %v", err)
	}

	// Test the handler
	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	handleListScenarioTemplates(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleListScenarioTemplates() status = %d, want %d, body: %s",
			w.Code, http.StatusOK, w.Body.String())
	}

	var response ScenarioTemplateListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(response.Templates))
	}

	if len(response.Templates) > 0 {
		tmpl := response.Templates[0]
		if tmpl.Name != "test-template" {
			t.Errorf("Template name = %q, want %q", tmpl.Name, "test-template")
		}
		if tmpl.DisplayName != "Test Template" {
			t.Errorf("Template displayName = %q, want %q", tmpl.DisplayName, "Test Template")
		}
		if len(tmpl.Stack) != 2 {
			t.Errorf("Template stack length = %d, want 2", len(tmpl.Stack))
		}
		if len(tmpl.Required) != 1 {
			t.Errorf("Template required vars = %d, want 1", len(tmpl.Required))
		}
		if len(tmpl.Optional) != 1 {
			t.Errorf("Template optional vars = %d, want 1", len(tmpl.Optional))
		}
	}
}

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestHandleGetScenarioTemplate(t *testing.T) {
	// Save and restore original VROOLI_ROOT
	origRoot := os.Getenv("VROOLI_ROOT")
	defer func() {
		if origRoot != "" {
			os.Setenv("VROOLI_ROOT", origRoot)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
	}()

	// Create temporary vrooli root with templates
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)

	templatesDir := filepath.Join(tmpRoot, "scripts", "scenarios", "templates")
	testTemplateDir := filepath.Join(templatesDir, "minimal-template")
	if err := os.MkdirAll(testTemplateDir, 0755); err != nil {
		t.Fatalf("Failed to create template dir: %v", err)
	}

	// Create a minimal template.json (tests default values)
	templateJSON := `{
		"description": "Minimal template without name or displayName"
	}`

	manifestPath := filepath.Join(testTemplateDir, "template.json")
	if err := os.WriteFile(manifestPath, []byte(templateJSON), 0644); err != nil {
		t.Fatalf("Failed to create template.json: %v", err)
	}

	tests := []struct {
		name           string
		templateName   string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "existing template",
			templateName:   "minimal-template",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response ScenarioTemplateResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				// Should default name to the template directory name
				if response.Name != "minimal-template" {
					t.Errorf("Template name = %q, want %q", response.Name, "minimal-template")
				}
				// Should generate display name from template name
				if response.DisplayName == "" {
					t.Error("Expected display name to be generated")
				}
				if response.Description != "Minimal template without name or displayName" {
					t.Errorf("Template description = %q", response.Description)
				}
			},
		},
		{
			name:           "nonexistent template",
			templateName:   "nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if !contains(w.Body.String(), "not found") {
					t.Error("Expected 'not found' error message")
				}
			},
		},
		{
			name:           "empty template name",
			templateName:   "",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				if !contains(w.Body.String(), "required") {
					t.Error("Expected 'required' error message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/"+tt.templateName, nil)
			req = mux.SetURLVars(req, map[string]string{"name": tt.templateName})
			w := httptest.NewRecorder()

			handleGetScenarioTemplate(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleGetScenarioTemplate() status = %d, want %d, body: %s",
					w.Code, tt.expectedStatus, w.Body.String())
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestLoadScenarioTemplateManifest(t *testing.T) {
	// Save and restore original VROOLI_ROOT
	origRoot := os.Getenv("VROOLI_ROOT")
	defer func() {
		if origRoot != "" {
			os.Setenv("VROOLI_ROOT", origRoot)
		} else {
			os.Unsetenv("VROOLI_ROOT")
		}
	}()

	// Create temporary vrooli root with templates
	tmpRoot := t.TempDir()
	os.Setenv("VROOLI_ROOT", tmpRoot)

	templatesDir := filepath.Join(tmpRoot, "scripts", "scenarios", "templates")
	validTemplateDir := filepath.Join(templatesDir, "valid-template")
	if err := os.MkdirAll(validTemplateDir, 0755); err != nil {
		t.Fatalf("Failed to create template dir: %v", err)
	}

	// Create valid template.json
	validJSON := `{
		"name": "valid",
		"displayName": "Valid Template",
		"description": "Test template",
		"stack": ["TypeScript"]
	}`
	validPath := filepath.Join(validTemplateDir, "template.json")
	if err := os.WriteFile(validPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to create valid template.json: %v", err)
	}

	// Create invalid template.json (malformed JSON)
	invalidTemplateDir := filepath.Join(templatesDir, "invalid-template")
	if err := os.MkdirAll(invalidTemplateDir, 0755); err != nil {
		t.Fatalf("Failed to create invalid template dir: %v", err)
	}
	invalidPath := filepath.Join(invalidTemplateDir, "template.json")
	if err := os.WriteFile(invalidPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to create invalid template.json: %v", err)
	}

	tests := []struct {
		name      string
		tmplName  string
		wantError bool
		checkFunc func(*testing.T, *TemplateManifest)
	}{
		{
			name:      "valid template",
			tmplName:  "valid-template",
			wantError: false,
			checkFunc: func(t *testing.T, manifest *TemplateManifest) {
				if manifest.Name != "valid" {
					t.Errorf("Manifest name = %q, want %q", manifest.Name, "valid")
				}
				if manifest.DisplayName != "Valid Template" {
					t.Errorf("Manifest displayName = %q", manifest.DisplayName)
				}
				if len(manifest.Stack) != 1 || manifest.Stack[0] != "TypeScript" {
					t.Errorf("Manifest stack = %v", manifest.Stack)
				}
			},
		},
		{
			name:      "nonexistent template",
			tmplName:  "nonexistent",
			wantError: true,
		},
		{
			name:      "invalid JSON",
			tmplName:  "invalid-template",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := loadScenarioTemplateManifest(tt.tmplName)

			if tt.wantError {
				if err == nil {
					t.Error("loadScenarioTemplateManifest() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("loadScenarioTemplateManifest() unexpected error: %v", err)
				}
				if tt.checkFunc != nil {
					tt.checkFunc(t, manifest)
				}
			}
		})
	}
}

// [REQ:PCT-CATALOG-ENUMERATE] Catalog enumerates all scenarios and resources with PRD status
func TestBuildVarList(t *testing.T) {
	tests := []struct {
		name     string
		vars     map[string]templateVarDefinition
		wantLen  int
		wantNil  bool
		checkVal func(*testing.T, []ScenarioTemplateVar)
	}{
		{
			name:    "empty map returns nil",
			vars:    map[string]templateVarDefinition{},
			wantNil: true,
		},
		{
			name:    "nil map returns nil",
			vars:    nil,
			wantNil: true,
		},
		{
			name: "single var",
			vars: map[string]templateVarDefinition{
				"port": {
					Name:        "port",
					Flag:        "--port",
					Description: "API port",
					Default:     "8080",
					Required:    true,
				},
			},
			wantLen: 1,
			checkVal: func(t *testing.T, vars []ScenarioTemplateVar) {
				if vars[0].Name != "port" {
					t.Errorf("Var name = %q, want %q", vars[0].Name, "port")
				}
				if vars[0].Flag != "--port" {
					t.Errorf("Var flag = %q", vars[0].Flag)
				}
				if !vars[0].Required {
					t.Error("Var should be required")
				}
			},
		},
		{
			name: "multiple vars sorted alphabetically",
			vars: map[string]templateVarDefinition{
				"zebra": {Name: "zebra", Required: false},
				"alpha": {Name: "alpha", Required: true},
				"beta":  {Name: "beta", Required: false},
			},
			wantLen: 3,
			checkVal: func(t *testing.T, vars []ScenarioTemplateVar) {
				if vars[0].Name != "alpha" {
					t.Errorf("First var name = %q, want %q", vars[0].Name, "alpha")
				}
				if vars[1].Name != "beta" {
					t.Errorf("Second var name = %q, want %q", vars[1].Name, "beta")
				}
				if vars[2].Name != "zebra" {
					t.Errorf("Third var name = %q, want %q", vars[2].Name, "zebra")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildVarList(tt.vars)

			if tt.wantNil {
				if result != nil {
					t.Errorf("buildVarList() = %v, want nil", result)
				}
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("buildVarList() length = %d, want %d", len(result), tt.wantLen)
			}

			if tt.checkVal != nil {
				tt.checkVal(t, result)
			}
		})
	}
}
