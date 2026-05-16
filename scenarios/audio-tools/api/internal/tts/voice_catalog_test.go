package tts

import (
	"errors"
	"testing"
)

func TestVerifyCatalog_AllCovered(t *testing.T) {
	entries := []AdapterCatalogEntry{
		{TierProvider: "local:kokoro-local", Mapping: LocalKokoroMapping},
	}
	if err := VerifyCatalog(entries); err != nil {
		t.Fatalf("VerifyCatalog returned %v; want nil", err)
	}
}

func TestVerifyCatalog_MissingCanonical(t *testing.T) {
	partial := AdapterVoiceMap{
		"voice.feminine.warm":   "af_bella",
		"voice.neutral.default": "af_nicole",
	}
	err := VerifyCatalog([]AdapterCatalogEntry{
		{TierProvider: "test", Mapping: partial},
	})
	if err == nil {
		t.Fatal("VerifyCatalog returned nil; want missing-canonical error")
	}
	var missing *MissingCanonicalVoiceError
	if !errors.As(err, &missing) {
		t.Fatalf("err = %v; want *MissingCanonicalVoiceError", err)
	}
	if missing.Adapter != "test" {
		t.Errorf("Adapter = %q; want %q", missing.Adapter, "test")
	}
}
