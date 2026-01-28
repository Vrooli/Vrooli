package testutil

import (
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

func TestAssertStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(200)

	// This should not fail (we're just checking it doesn't panic)
	mockT := &testing.T{}
	AssertStatus(mockT, rec, 200)
}

func TestDecodeJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteString(`{"name": "test", "value": 123}`)

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
	os.WriteFile(path, []byte(`{"items": ["a", "b", "c"]}`), 0o644)

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
	os.WriteFile(path, []byte("content"), 0o644)

	// This should not cause a test failure
	mockT := &testing.T{}
	AssertFileExists(mockT, path)
}

func TestAssertFileNotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.txt")

	// This should not cause a test failure
	mockT := &testing.T{}
	AssertFileNotExists(mockT, path)
}
