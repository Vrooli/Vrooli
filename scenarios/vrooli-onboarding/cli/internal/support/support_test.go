package support

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSupportDecodingAndFiles(t *testing.T) {
	var value struct {
		Name string `json:"name"`
	}
	if err := Decode([]byte(`{"success":true,"data":{"name":"demo"}}`), &value); err != nil || value.Name != "demo" {
		t.Fatalf("envelope decode = %#v/%v", value, err)
	}
	if string(DecodeRaw([]byte(`{"success":true,"data":{"ok":true}}`))) != `{"ok":true}` {
		t.Fatal("DecodeRaw did not unwrap envelope")
	}
	if got := EnvelopeMessage([]byte(`{"success":true,"data":{"message":"done"}}`)); got != "done" {
		t.Fatalf("EnvelopeMessage = %q", got)
	}
	if got := BuildQuery(map[string]string{"q": " demo ", "empty": " "}); got.Get("q") != "demo" || got.Get("empty") != "" {
		t.Fatalf("BuildQuery = %v", got)
	}
	path := filepath.Join(t.TempDir(), "value.json")
	if err := os.WriteFile(path, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadJSONFile(path, true); err != nil {
		t.Fatal(err)
	}
	if err := WriteOutput(filepath.Join(t.TempDir(), "nested", "out.json"), []byte("ok")); err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(MapRows(map[string]interface{}{"a": 1})); err != nil {
		t.Fatal(err)
	}
}

func TestSupportFormattingAndValidation(t *testing.T) {
	if FormatTime("") != "unknown" || FormatTime("not-a-time") != "not-a-time" {
		t.Fatal("unexpected time formatting")
	}
	if FormatTimeValue(time.Time{}) != "unknown" || PtrString(nil) != "" || ShortID("123456789") != "12345678" {
		t.Fatal("unexpected helper formatting")
	}
	if _, err := ReadJSONFile("", true); err == nil {
		t.Fatal("required empty JSON path should fail")
	}
	if err := ParseFlags(NewFlagSet("test"), []string{"--unknown"}); err == nil {
		t.Fatal("unknown flag should fail")
	}
}
