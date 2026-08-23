package scenarioconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigThenDefault(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	config := filepath.Join(dir, "config.json")
	write(t, schema, `{"settings":{"attempts":{"type":"integer","default":3},"enabled":{"type":"boolean","default":true}}}`)
	write(t, config, `{"settings":{"attempts":7}}`)
	values, err := Load(config, schema)
	if err != nil {
		t.Fatal(err)
	}
	if values["attempts"] != float64(7) || values["enabled"] != true {
		t.Fatalf("values = %#v", values)
	}
	if err := os.WriteFile(config, []byte(`{"settings":{"attempts":"bad"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(config, schema); err == nil {
		t.Fatal("expected invalid type error")
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
