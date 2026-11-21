package tts

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/tts-config.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := DefaultConfig()
	if cfg != want {
		t.Errorf("expected defaults %+v, got %+v", want, cfg)
	}
}

func TestConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tts-config.json")

	cfg := Config{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded != cfg {
		t.Errorf("expected %+v, got %+v", cfg, loaded)
	}
}

func TestConfigPatch_PreservesUnsetFields(t *testing.T) {
	base := Config{AutoEnabled: true, Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	patch := ConfigPatch{}
	result := patch.Apply(base)
	if result.AutoEnabled != true {
		t.Error("expected AutoEnabled to remain true with empty patch")
	}
	if result.Backend != "kokoro" {
		t.Error("expected Backend to remain kokoro with empty patch")
	}
	if result.KokoroVoice != "af_heart" {
		t.Error("expected KokoroVoice to remain af_heart with empty patch")
	}
	if result.KokoroSpeed != 1.0 {
		t.Error("expected KokoroSpeed to remain 1.0 with empty patch")
	}
}

func TestConfigPatch_AppliesField(t *testing.T) {
	base := Config{AutoEnabled: false, Backend: "browser", KokoroVoice: "af_heart", KokoroSpeed: 1.0}
	v := true
	backend := "kokoro"
	voice := "bf_emma"
	speed := 1.5
	patch := ConfigPatch{AutoEnabled: &v, Backend: &backend, KokoroVoice: &voice, KokoroSpeed: &speed}
	result := patch.Apply(base)
	if result.AutoEnabled != true {
		t.Error("expected AutoEnabled to be set to true")
	}
	if result.Backend != "kokoro" {
		t.Errorf("expected Backend kokoro, got %s", result.Backend)
	}
	if result.KokoroVoice != "bf_emma" {
		t.Errorf("expected KokoroVoice bf_emma, got %s", result.KokoroVoice)
	}
	if result.KokoroSpeed != 1.5 {
		t.Errorf("expected KokoroSpeed 1.5, got %f", result.KokoroSpeed)
	}
}
