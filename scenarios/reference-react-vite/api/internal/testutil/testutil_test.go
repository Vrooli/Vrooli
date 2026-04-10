// Package testutil provides testing utilities and helpers for unit tests.
// [REQ:MOD-P0-007] Test architecture - co-located test for testutil package
package testutil

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssertStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   int
		shouldFail bool
	}{
		{
			name:       "matching_status_passes",
			statusCode: 200,
			expected:   200,
			shouldFail: false,
		},
		{
			name:       "matching_not_found_passes",
			statusCode: 404,
			expected:   404,
			shouldFail: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.statusCode)
			rec.Body.WriteString(`{"status": "ok"}`)

			// We can only test non-failing cases since AssertStatus calls t.Fatal
			if !tc.shouldFail {
				AssertStatus(t, rec, tc.expected)
			}
		})
	}
}

func TestAssertJSON(t *testing.T) {
	t.Run("valid_json_decodes_successfully", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Body.WriteString(`{"name": "test", "value": 42}`)

		var result struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		AssertJSON(t, rec, &result)

		if result.Name != "test" {
			t.Errorf("expected name 'test', got '%s'", result.Name)
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})
}

func TestAssertContentType(t *testing.T) {
	t.Run("matching_content_type_passes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json; charset=utf-8")

		AssertContentType(t, rec, "application/json")
	})
}

func TestMakeRequest(t *testing.T) {
	t.Run("creates_get_request_when_method_empty", func(t *testing.T) {
		req := MakeRequest(t, "", "/test/path", nil)

		if req.Method != "GET" {
			t.Errorf("expected method GET, got %s", req.Method)
		}
		if req.URL.Path != "/test/path" {
			t.Errorf("expected path /test/path, got %s", req.URL.Path)
		}
	})

	t.Run("creates_post_request", func(t *testing.T) {
		body := strings.NewReader(`{"key": "value"}`)
		req := MakeRequest(t, "POST", "/api/v1/tasks", body)

		if req.Method != "POST" {
			t.Errorf("expected method POST, got %s", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
		}
	})
}

func TestMakeJSONRequest(t *testing.T) {
	t.Run("marshals_body_to_json", func(t *testing.T) {
		body := map[string]string{"name": "test"}
		req := MakeJSONRequest(t, "POST", "/api/v1/tasks", body)

		if req.Method != "POST" {
			t.Errorf("expected method POST, got %s", req.Method)
		}
		if req.Body == nil {
			t.Error("expected body to be set")
		}
	})

	t.Run("handles_nil_body", func(t *testing.T) {
		req := MakeJSONRequest(t, "GET", "/api/v1/tasks", nil)

		if req.Method != "GET" {
			t.Errorf("expected method GET, got %s", req.Method)
		}
	})
}

func TestDecodeJSONResponse(t *testing.T) {
	t.Run("decodes_json_to_struct", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Body.WriteString(`{"id": "123", "name": "Test"}`)

		type Response struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		result := DecodeJSONResponse[Response](t, rec)

		if result.ID != "123" {
			t.Errorf("expected id '123', got '%s'", result.ID)
		}
		if result.Name != "Test" {
			t.Errorf("expected name 'Test', got '%s'", result.Name)
		}
	})
}

func TestMustParseJSON(t *testing.T) {
	t.Run("parses_json_string", func(t *testing.T) {
		type Data struct {
			Value int `json:"value"`
		}

		result := MustParseJSON[Data](t, `{"value": 100}`)

		if result.Value != 100 {
			t.Errorf("expected value 100, got %d", result.Value)
		}
	})
}

func TestStringPtr(t *testing.T) {
	s := "test"
	ptr := StringPtr(s)

	if ptr == nil {
		t.Error("expected non-nil pointer")
	}
	if *ptr != s {
		t.Errorf("expected '%s', got '%s'", s, *ptr)
	}
}

func TestIntPtr(t *testing.T) {
	i := 42
	ptr := IntPtr(i)

	if ptr == nil {
		t.Error("expected non-nil pointer")
	}
	if *ptr != i {
		t.Errorf("expected %d, got %d", i, *ptr)
	}
}

func TestTaskFactory(t *testing.T) {
	t.Run("creates_default_task", func(t *testing.T) {
		factory := NewTaskFactory()
		task := factory.Build()

		if task.ID == "" {
			t.Error("expected non-empty ID")
		}
		if task.Title == "" {
			t.Error("expected non-empty Title")
		}
	})

	t.Run("applies_builder_methods", func(t *testing.T) {
		factory := NewTaskFactory().
			WithID("custom-id").
			WithTitle("Custom Title")
		task := factory.Build()

		if task.ID != "custom-id" {
			t.Errorf("expected ID 'custom-id', got '%s'", task.ID)
		}
		if task.Title != "Custom Title" {
			t.Errorf("expected Title 'Custom Title', got '%s'", task.Title)
		}
	})
}

func TestProjectFactory(t *testing.T) {
	t.Run("creates_default_project", func(t *testing.T) {
		factory := NewProjectFactory()
		project := factory.Build()

		if project.ID == "" {
			t.Error("expected non-empty ID")
		}
		if project.Name == "" {
			t.Error("expected non-empty Name")
		}
	})

	t.Run("applies_builder_methods", func(t *testing.T) {
		factory := NewProjectFactory().
			WithID("custom-id").
			WithName("Custom Project").
			WithColor("#ff0000")
		project := factory.Build()

		if project.ID != "custom-id" {
			t.Errorf("expected ID 'custom-id', got '%s'", project.ID)
		}
		if project.Name != "Custom Project" {
			t.Errorf("expected Name 'Custom Project', got '%s'", project.Name)
		}
		if project.Color != "#ff0000" {
			t.Errorf("expected Color '#ff0000', got '%s'", project.Color)
		}
	})
}

func TestNoteFactory(t *testing.T) {
	t.Run("creates_default_note", func(t *testing.T) {
		factory := NewNoteFactory()
		note := factory.Build()

		if note.ID == "" {
			t.Error("expected non-empty ID")
		}
		if note.TaskID == "" {
			t.Error("expected non-empty TaskID")
		}
		if note.Content == "" {
			t.Error("expected non-empty Content")
		}
	})

	t.Run("applies_builder_methods", func(t *testing.T) {
		factory := NewNoteFactory().
			WithID("note-id").
			WithTaskID("task-id").
			WithContent("Custom content").
			WithAuthor("Custom Author")
		note := factory.Build()

		if note.ID != "note-id" {
			t.Errorf("expected ID 'note-id', got '%s'", note.ID)
		}
		if note.TaskID != "task-id" {
			t.Errorf("expected TaskID 'task-id', got '%s'", note.TaskID)
		}
		if note.Content != "Custom content" {
			t.Errorf("expected Content 'Custom content', got '%s'", note.Content)
		}
		if note.Author != "Custom Author" {
			t.Errorf("expected Author 'Custom Author', got '%s'", note.Author)
		}
	})
}
