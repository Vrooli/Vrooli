package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/tts-config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != DefaultConfig() {
		t.Fatalf("expected defaults %+v, got %+v", DefaultConfig(), cfg)
	}
}

func TestConfig_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tts-config.json")
	cfg := Config{AutoEnabled: true, DefaultResponseFormat: "wav", KokoroVoice: "af_heart", KokoroSpeed: 1}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != cfg {
		t.Fatalf("expected %+v, got %+v", cfg, loaded)
	}
}

func TestConfigPatchPreservesAndUpdatesResponseFormat(t *testing.T) {
	base := Config{AutoEnabled: true, DefaultResponseFormat: "wav", KokoroVoice: "af_heart", KokoroSpeed: 1}
	if got := (ConfigPatch{}).Apply(base); got != base {
		t.Fatalf("empty patch changed config: %+v", got)
	}
	format := "opus"
	got := (ConfigPatch{DefaultResponseFormat: &format}).Apply(base)
	if got.DefaultResponseFormat != format {
		t.Fatalf("response format = %q, want %q", got.DefaultResponseFormat, format)
	}
}

func TestLoadConfigMigratesLegacyBackendResponseFormat(t *testing.T) {
	for _, tc := range []struct{ legacy, want string }{
		{legacy: "flac", want: "flac"},
		{legacy: "kokoro", want: DefaultResponseFormat},
		{legacy: "", want: DefaultResponseFormat},
	} {
		path := filepath.Join(t.TempDir(), "tts-config.json")
		if err := os.WriteFile(path, []byte(`{"backend":"`+tc.legacy+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := LoadConfig(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.DefaultResponseFormat != tc.want {
			t.Fatalf("legacy %q migrated to %q, want %q", tc.legacy, got.DefaultResponseFormat, tc.want)
		}
	}
}
