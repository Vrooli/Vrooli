package testutil

import (
	"testing"
)

func TestWriteJSON(t *testing.T) {
	path := t.TempDir() + "/fixture.json"
	if err := WriteJSON(path, map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
}
