package world

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore(t.TempDir())
	s.now = func() time.Time { return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC) }
	return s
}

func TestLoadConfigDefaultsWhenMissing(t *testing.T) {
	s := newTestStore(t)
	cfg, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg != DefaultConfig() {
		t.Fatalf("expected default config, got %+v", cfg)
	}
}

func TestSaveConfigRoundTripStampsUpdatedAt(t *testing.T) {
	s := newTestStore(t)
	in := Config{Scene: "office", QualityProfile: "medium", QualityAuto: false, PeriodMode: "night", TwoDMode: true, ShowDiagnostics: true, Scale: 1.5}
	saved, err := s.SaveConfig(in)
	if err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if saved.UpdatedAt != "2026-09-02T12:00:00Z" {
		t.Fatalf("updatedAt not stamped: %q", saved.UpdatedAt)
	}
	loaded, err := s.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded != saved {
		t.Fatalf("round trip mismatch: %+v vs %+v", loaded, saved)
	}
}

func TestSaveConfigRejectsBadValues(t *testing.T) {
	s := newTestStore(t)
	cases := map[string]Config{
		"scene":          {Scene: "moon", QualityProfile: "high", PeriodMode: "clock", Scale: 1},
		"qualityProfile": {Scene: "park", QualityProfile: "insane", PeriodMode: "clock", Scale: 1},
		"periodMode":     {Scene: "park", QualityProfile: "high", PeriodMode: "noon", Scale: 1},
		"scale":          {Scene: "park", QualityProfile: "high", PeriodMode: "clock", Scale: 9},
	}
	for name, cfg := range cases {
		if _, err := s.SaveConfig(cfg); err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("%s: expected validation error naming the field, got %v", name, err)
		}
	}
	if _, err := os.Stat(s.configPath()); !os.IsNotExist(err) {
		t.Fatalf("invalid config must not be written")
	}
}

func TestLoadConfigMalformedIsAnError(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.configPath(), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadConfig(); err == nil {
		t.Fatal("malformed config must not be replaced by defaults silently")
	}
	if err := os.WriteFile(s.configPath(), []byte(`{"scene":"moon","qualityProfile":"high","periodMode":"clock","scale":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadConfig(); err == nil || !strings.Contains(err.Error(), filepath.Base(s.configPath())) {
		t.Fatalf("out-of-range saved config must fail with the path, got %v", err)
	}
}
