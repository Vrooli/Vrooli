package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	data := map[string]string{"key": "value"}
	WriteJSONFile(t, path, data)

	// Verify file was created
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if string(content) != "{\n  \"key\": \"value\"\n}" {
		t.Errorf("Unexpected content: %s", content)
	}
}

func TestWriteJSONFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "dir", "test.json")

	data := map[string]int{"num": 42}
	WriteJSONFile(t, path, data)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("File not created in nested directory")
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	WriteFile(t, path, "hello world")

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("File not created: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("Unexpected content: %s", content)
	}
}

func TestMakeDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	MakeDir(t, nested)

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path is not a directory")
	}
}

func TestDecodeJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, err := rec.WriteString(`{"name": "test", "value": 123}`); err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	result := DecodeJSON[testStruct](t, rec)
	if result.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", result.Name)
	}
	if result.Value != 123 {
		t.Errorf("Expected value 123, got %d", result.Value)
	}
}

func TestReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte(`{"items": ["a", "b", "c"]}`), 0o644); err != nil {
		t.Fatalf("Failed to write JSON file: %v", err)
	}

	type testData struct {
		Items []string `json:"items"`
	}

	result := ReadJSONFile[testData](t, path)
	if len(result.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result.Items))
	}
}

func TestAssertFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	AssertFileExists(t, path)
}

func TestAssertFileNotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.txt")

	AssertFileNotExists(t, path)
}

func TestAssertStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{name: "ok", status: http.StatusOK},
		{name: "accepted", status: http.StatusAccepted},
		{name: "server_error", status: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.status)

			AssertStatus(t, rec, tc.status)
		})
	}
}
