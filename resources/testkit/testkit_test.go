package testkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHarnessAndWriteJSON(t *testing.T) {
	h := Handlers(t, WithInput("request"))
	if h.In == nil || h.Stdout == nil || h.Stderr == nil {
		t.Fatal("harness did not initialize all streams")
	}
	path := filepath.Join(t.TempDir(), "fixture.json")
	WriteJSON(t, path, map[string]string{"status": "ok"})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil || got["status"] != "ok" {
		t.Fatalf("fixture = %s, err = %v", data, err)
	}
}
