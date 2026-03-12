// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
// Package testutil tests verify the test utilities work correctly.
// These meta-tests ensure our testing infrastructure is reliable.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAssertStatus verifies the status assertion helper.
func TestAssertStatus(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		expectedCode   int
		shouldPass     bool
		category       string
	}{
		{
			name:         "matching_status",
			responseCode: http.StatusOK,
			expectedCode: http.StatusOK,
			shouldPass:   true,
			category:     "happy_path",
		},
		{
			name:         "created_status",
			responseCode: http.StatusCreated,
			expectedCode: http.StatusCreated,
			shouldPass:   true,
			category:     "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.responseCode)

			// This is a "happy path only" test - we only test successful assertions
			if tc.shouldPass {
				AssertStatus(t, rec, tc.expectedCode)
			}
		})
	}
}

// TestAssertContentType verifies the content type assertion helper.
func TestAssertContentType(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expected     string
		shouldPass   bool
		category     string
	}{
		{
			name:        "exact_match",
			contentType: "application/json",
			expected:    "application/json",
			shouldPass:  true,
			category:    "happy_path",
		},
		{
			name:        "prefix_match",
			contentType: "application/json; charset=utf-8",
			expected:    "application/json",
			shouldPass:  true,
			category:    "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", tc.contentType)

			if tc.shouldPass {
				AssertContentType(t, rec, tc.expected)
			}
		})
	}
}

// TestMakeRequest verifies the request factory helper.
func TestMakeRequest(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantMethod string
		category   string
	}{
		{
			name:       "get_request",
			method:     http.MethodGet,
			path:       "/api/v1/test",
			wantMethod: http.MethodGet,
			category:   "happy_path",
		},
		{
			name:       "post_request",
			method:     http.MethodPost,
			path:       "/api/v1/test",
			wantMethod: http.MethodPost,
			category:   "happy_path",
		},
		{
			name:       "empty_method_defaults_to_get",
			method:     "",
			path:       "/api/v1/test",
			wantMethod: http.MethodGet,
			category:   "edge_case",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			req := MakeRequest(t, tc.method, tc.path, nil)

			// ASSERT
			if req.Method != tc.wantMethod {
				t.Errorf("expected method %q, got %q", tc.wantMethod, req.Method)
			}
			if req.URL.Path != tc.path {
				t.Errorf("expected path %q, got %q", tc.path, req.URL.Path)
			}
			if ct := req.Header.Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", ct)
			}
		})
	}
}

// TestMakeJSONRequest verifies the JSON request factory helper.
func TestMakeJSONRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		category string
	}{
		{
			name:     "nil_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     nil,
			category: "boundary",
		},
		{
			name:     "map_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     map[string]string{"key": "value"},
			category: "happy_path",
		},
		{
			name:     "struct_body",
			method:   http.MethodPost,
			path:     "/api/v1/test",
			body:     struct{ Name string }{"test"},
			category: "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			req := MakeJSONRequest(t, tc.method, tc.path, tc.body)

			// ASSERT
			if req.Method != tc.method {
				t.Errorf("expected method %q, got %q", tc.method, req.Method)
			}
		})
	}
}

// TestStringPtr verifies the string pointer helper.
func TestStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		category string
	}{
		{
			name:     "non_empty_string",
			input:    "test value",
			category: "happy_path",
		},
		{
			name:     "empty_string",
			input:    "",
			category: "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := StringPtr(tc.input)

			// ASSERT
			if result == nil {
				t.Fatal("expected non-nil pointer")
			}
			if *result != tc.input {
				t.Errorf("expected %q, got %q", tc.input, *result)
			}
		})
	}
}

// TestMustParseJSON verifies the JSON parsing helper.
func TestMustParseJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantName string
		category string
	}{
		{
			name:     "simple_object",
			json:     `{"name":"test"}`,
			wantName: "test",
			category: "happy_path",
		},
		{
			name:     "nested_object",
			json:     `{"name":"nested","extra":"value"}`,
			wantName: "nested",
			category: "happy_path",
		},
	}

	type testStruct struct {
		Name string `json:"name"`
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// ACT
			result := MustParseJSON[testStruct](t, tc.json)

			// ASSERT
			if result.Name != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, result.Name)
			}
		})
	}
}

// TestAssertJSON verifies the JSON assertion helper.
func TestAssertJSON(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantKey  string
		category string
	}{
		{
			name:     "valid_json",
			body:     `{"key":"value"}`,
			wantKey:  "value",
			category: "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteString(tc.body)

			var result map[string]string
			AssertJSON(t, rec, &result)

			if result["key"] != tc.wantKey {
				t.Errorf("expected key %q, got %q", tc.wantKey, result["key"])
			}
		})
	}
}

// TestDecodeJSONResponse verifies the generic JSON decoder helper.
func TestDecodeJSONResponse(t *testing.T) {
	type testResponse struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name       string
		body       string
		wantName   string
		wantValue  int
		category   string
	}{
		{
			name:      "decode_struct",
			body:      `{"name":"test","value":42}`,
			wantName:  "test",
			wantValue: 42,
			category:  "happy_path",
		},
		{
			name:      "decode_with_extra_fields",
			body:      `{"name":"extra","value":100,"ignored":"field"}`,
			wantName:  "extra",
			wantValue: 100,
			category:  "edge_case",
		},
		{
			name:      "decode_partial_fields",
			body:      `{"name":"partial"}`,
			wantName:  "partial",
			wantValue: 0,
			category:  "boundary",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteString(tc.body)

			result := DecodeJSONResponse[testResponse](t, rec)

			if result.Name != tc.wantName {
				t.Errorf("expected name %q, got %q", tc.wantName, result.Name)
			}
			if result.Value != tc.wantValue {
				t.Errorf("expected value %d, got %d", tc.wantValue, result.Value)
			}
		})
	}
}

// TestReferenceFactory verifies the reference fixture factory.
func TestReferenceFactory(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		factory := NewReferenceFactory()
		ref := factory.Build()

		if ref.ID == "" {
			t.Error("expected non-empty default ID")
		}
		if ref.Slug != "test-reference" {
			t.Errorf("expected default slug 'test-reference', got %q", ref.Slug)
		}
		if ref.Template != "react-vite" {
			t.Errorf("expected default template 'react-vite', got %q", ref.Template)
		}
	})

	t.Run("with_custom_id", func(t *testing.T) {
		ref := NewReferenceFactory().WithID("custom-id").Build()
		if ref.ID != "custom-id" {
			t.Errorf("expected ID 'custom-id', got %q", ref.ID)
		}
	})

	t.Run("with_custom_slug", func(t *testing.T) {
		ref := NewReferenceFactory().WithSlug("my-slug").Build()
		if ref.Slug != "my-slug" {
			t.Errorf("expected slug 'my-slug', got %q", ref.Slug)
		}
	})

	t.Run("with_custom_name", func(t *testing.T) {
		ref := NewReferenceFactory().WithName("My Name").Build()
		if ref.Name != "My Name" {
			t.Errorf("expected name 'My Name', got %q", ref.Name)
		}
	})

	t.Run("with_custom_template", func(t *testing.T) {
		ref := NewReferenceFactory().WithTemplate("cli-only").Build()
		if ref.Template != "cli-only" {
			t.Errorf("expected template 'cli-only', got %q", ref.Template)
		}
	})

	t.Run("with_custom_path", func(t *testing.T) {
		ref := NewReferenceFactory().WithPath("/custom/path").Build()
		if ref.Path != "/custom/path" {
			t.Errorf("expected path '/custom/path', got %q", ref.Path)
		}
	})

	t.Run("with_custom_description", func(t *testing.T) {
		ref := NewReferenceFactory().WithDescription("Custom desc").Build()
		if ref.Description != "Custom desc" {
			t.Errorf("expected description 'Custom desc', got %q", ref.Description)
		}
	})

	t.Run("chained_modifications", func(t *testing.T) {
		ref := NewReferenceFactory().
			WithID("id-1").
			WithSlug("slug-1").
			WithName("Name 1").
			WithTemplate("template-1").
			WithPath("/path/1").
			WithDescription("Desc 1").
			Build()

		if ref.ID != "id-1" || ref.Slug != "slug-1" || ref.Name != "Name 1" {
			t.Errorf("chained modifications failed: %+v", ref)
		}
	})

	t.Run("build_returns_copy", func(t *testing.T) {
		factory := NewReferenceFactory()
		ref1 := factory.Build()
		ref2 := factory.Build()

		// Modify ref1
		ref1.Slug = "modified"

		// ref2 should not be affected
		if ref2.Slug == "modified" {
			t.Error("Build should return a copy, not a reference")
		}
	})
}

// TestCreateInputFactory verifies the create input fixture factory.
func TestCreateInputFactory(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		factory := NewCreateInputFactory()
		input := factory.Build()

		if input.Slug != "test-reference" {
			t.Errorf("expected default slug 'test-reference', got %q", input.Slug)
		}
		if input.Template != "react-vite" {
			t.Errorf("expected default template 'react-vite', got %q", input.Template)
		}
	})

	t.Run("with_custom_slug", func(t *testing.T) {
		input := NewCreateInputFactory().WithSlug("custom-slug").Build()
		if input.Slug != "custom-slug" {
			t.Errorf("expected slug 'custom-slug', got %q", input.Slug)
		}
	})

	t.Run("with_custom_name", func(t *testing.T) {
		input := NewCreateInputFactory().WithName("Custom Name").Build()
		if input.Name != "Custom Name" {
			t.Errorf("expected name 'Custom Name', got %q", input.Name)
		}
	})

	t.Run("with_custom_template", func(t *testing.T) {
		input := NewCreateInputFactory().WithTemplate("landing-page").Build()
		if input.Template != "landing-page" {
			t.Errorf("expected template 'landing-page', got %q", input.Template)
		}
	})

	t.Run("with_custom_path", func(t *testing.T) {
		input := NewCreateInputFactory().WithPath("/new/path").Build()
		if input.Path != "/new/path" {
			t.Errorf("expected path '/new/path', got %q", input.Path)
		}
	})

	t.Run("with_custom_description", func(t *testing.T) {
		input := NewCreateInputFactory().WithDescription("New desc").Build()
		if input.Description != "New desc" {
			t.Errorf("expected description 'New desc', got %q", input.Description)
		}
	})

	t.Run("chained_modifications", func(t *testing.T) {
		input := NewCreateInputFactory().
			WithSlug("s1").
			WithName("N1").
			WithTemplate("t1").
			WithPath("/p1").
			WithDescription("d1").
			Build()

		if input.Slug != "s1" || input.Name != "N1" || input.Template != "t1" {
			t.Errorf("chained modifications failed: %+v", input)
		}
	})
}

// TestConnectionFactory verifies the skill connection fixture factory.
func TestConnectionFactory(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		factory := NewConnectionFactory()
		conn := factory.Build()

		if conn.ID == "" {
			t.Error("expected non-empty default ID")
		}
		if conn.ReferenceID == "" {
			t.Error("expected non-empty default ReferenceID")
		}
		if conn.SkillID != "test-skill" {
			t.Errorf("expected default skill ID 'test-skill', got %q", conn.SkillID)
		}
		if conn.SkillVersion != "1.0.0" {
			t.Errorf("expected default version '1.0.0', got %q", conn.SkillVersion)
		}
	})

	t.Run("with_custom_id", func(t *testing.T) {
		conn := NewConnectionFactory().WithID("custom-conn-id").Build()
		if conn.ID != "custom-conn-id" {
			t.Errorf("expected ID 'custom-conn-id', got %q", conn.ID)
		}
	})

	t.Run("with_custom_reference_id", func(t *testing.T) {
		conn := NewConnectionFactory().WithReferenceID("ref-123").Build()
		if conn.ReferenceID != "ref-123" {
			t.Errorf("expected ReferenceID 'ref-123', got %q", conn.ReferenceID)
		}
	})

	t.Run("with_custom_skill_id", func(t *testing.T) {
		conn := NewConnectionFactory().WithSkillID("api-steer").Build()
		if conn.SkillID != "api-steer" {
			t.Errorf("expected SkillID 'api-steer', got %q", conn.SkillID)
		}
	})

	t.Run("with_custom_version", func(t *testing.T) {
		conn := NewConnectionFactory().WithSkillVersion("2.0.0").Build()
		if conn.SkillVersion != "2.0.0" {
			t.Errorf("expected SkillVersion '2.0.0', got %q", conn.SkillVersion)
		}
	})

	t.Run("with_custom_hash", func(t *testing.T) {
		conn := NewConnectionFactory().WithSkillContentHash("sha256:xyz").Build()
		if conn.SkillContentHash != "sha256:xyz" {
			t.Errorf("expected SkillContentHash 'sha256:xyz', got %q", conn.SkillContentHash)
		}
	})

	t.Run("chained_modifications", func(t *testing.T) {
		conn := NewConnectionFactory().
			WithID("id-1").
			WithReferenceID("ref-1").
			WithSkillID("skill-1").
			WithSkillVersion("v1").
			WithSkillContentHash("hash-1").
			Build()

		if conn.ID != "id-1" || conn.ReferenceID != "ref-1" || conn.SkillID != "skill-1" {
			t.Errorf("chained modifications failed: %+v", conn)
		}
	})

	t.Run("build_returns_copy", func(t *testing.T) {
		factory := NewConnectionFactory()
		conn1 := factory.Build()
		conn2 := factory.Build()

		// Modify conn1
		conn1.SkillID = "modified-skill"

		// conn2 should not be affected
		if conn2.SkillID == "modified-skill" {
			t.Error("Build should return a copy, not a reference")
		}
	})
}

// TestConnectInputFactory verifies the connect input fixture factory.
func TestConnectInputFactory(t *testing.T) {
	t.Run("default_values", func(t *testing.T) {
		factory := NewConnectInputFactory()
		input := factory.Build()

		if input.ReferenceID == "" {
			t.Error("expected non-empty default ReferenceID")
		}
		if input.SkillID != "test-skill" {
			t.Errorf("expected default skill ID 'test-skill', got %q", input.SkillID)
		}
		if input.SkillVersion != "1.0.0" {
			t.Errorf("expected default version '1.0.0', got %q", input.SkillVersion)
		}
	})

	t.Run("with_custom_reference_id", func(t *testing.T) {
		input := NewConnectInputFactory().WithReferenceID("ref-456").Build()
		if input.ReferenceID != "ref-456" {
			t.Errorf("expected ReferenceID 'ref-456', got %q", input.ReferenceID)
		}
	})

	t.Run("with_custom_skill_id", func(t *testing.T) {
		input := NewConnectInputFactory().WithSkillID("cli-steer").Build()
		if input.SkillID != "cli-steer" {
			t.Errorf("expected SkillID 'cli-steer', got %q", input.SkillID)
		}
	})

	t.Run("with_custom_version", func(t *testing.T) {
		input := NewConnectInputFactory().WithSkillVersion("3.0.0").Build()
		if input.SkillVersion != "3.0.0" {
			t.Errorf("expected SkillVersion '3.0.0', got %q", input.SkillVersion)
		}
	})

	t.Run("with_custom_hash", func(t *testing.T) {
		input := NewConnectInputFactory().WithSkillContentHash("hash-abc").Build()
		if input.SkillContentHash != "hash-abc" {
			t.Errorf("expected SkillContentHash 'hash-abc', got %q", input.SkillContentHash)
		}
	})

	t.Run("chained_modifications", func(t *testing.T) {
		input := NewConnectInputFactory().
			WithReferenceID("r1").
			WithSkillID("s1").
			WithSkillVersion("v1").
			WithSkillContentHash("h1").
			Build()

		if input.ReferenceID != "r1" || input.SkillID != "s1" || input.SkillVersion != "v1" {
			t.Errorf("chained modifications failed: %+v", input)
		}
	})
}
