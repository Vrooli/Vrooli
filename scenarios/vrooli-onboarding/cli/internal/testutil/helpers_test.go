package testutil

import (
	"net/http"
	"testing"
)

func TestNewTestAppAndWriteJSON(t *testing.T) {
	app := NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	if _, err := app.Get("/probe", nil); err != nil {
		t.Fatal(err)
	}
	path := t.TempDir() + "/fixture.json"
	if err := WriteJSON(path, map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
}
