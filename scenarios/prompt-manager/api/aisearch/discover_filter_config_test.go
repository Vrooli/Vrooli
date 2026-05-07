package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDiscoverFilterConfig(t *testing.T) {
	cfg := DefaultDiscoverFilterConfig()

	if cfg.IncludeDrafts {
		t.Error("IncludeDrafts should default to false")
	}
	if len(cfg.ExcludeModes) != 1 || cfg.ExcludeModes[0] != "scope" {
		t.Errorf("ExcludeModes should default to [scope], got %v", cfg.ExcludeModes)
	}
	if len(cfg.ExcludeIDs) != 0 {
		t.Errorf("ExcludeIDs should default to empty, got %v", cfg.ExcludeIDs)
	}
	if len(cfg.ExcludeTags) != 0 {
		t.Errorf("ExcludeTags should default to empty, got %v", cfg.ExcludeTags)
	}

	if err := ValidateDiscoverFilterConfig(cfg); err != nil {
		t.Errorf("default config should be valid, got: %v", err)
	}
}

func TestValidateDiscoverFilterConfig_Valid(t *testing.T) {
	cfg := DiscoverFilterConfig{
		IncludeDrafts: true,
		ExcludeModes:  []string{"scope", "meta"},
		ExcludeIDs:    []string{"skill-1", "skill-2"},
		ExcludeTags:   []string{"deprecated"},
	}
	if err := ValidateDiscoverFilterConfig(cfg); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateDiscoverFilterConfig_Invalid(t *testing.T) {
	tooMany := make([]string, maxExcludeEntries+1)
	for i := range tooMany {
		tooMany[i] = "x"
	}

	tests := []struct {
		name string
		cfg  DiscoverFilterConfig
	}{
		{"too many modes", DiscoverFilterConfig{ExcludeModes: tooMany}},
		{"too many ids", DiscoverFilterConfig{ExcludeIDs: tooMany}},
		{"too many tags", DiscoverFilterConfig{ExcludeTags: tooMany}},
	}

	for _, tc := range tests {
		if err := ValidateDiscoverFilterConfig(tc.cfg); err == nil {
			t.Errorf("%s: expected validation error, got nil", tc.name)
		}
	}
}

func TestDiscoverFilterConfigStore_Get_NoFile(t *testing.T) {
	dir := t.TempDir()
	s := NewDiscoverFilterConfigStore(dir)

	cfg, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := DefaultDiscoverFilterConfig()
	if cfg.IncludeDrafts != defaults.IncludeDrafts {
		t.Errorf("IncludeDrafts: got %v, want %v", cfg.IncludeDrafts, defaults.IncludeDrafts)
	}
	if len(cfg.ExcludeModes) != len(defaults.ExcludeModes) || cfg.ExcludeModes[0] != defaults.ExcludeModes[0] {
		t.Errorf("ExcludeModes: got %v, want %v", cfg.ExcludeModes, defaults.ExcludeModes)
	}
}

func TestDiscoverFilterConfigStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := NewDiscoverFilterConfigStore(dir)

	custom := DiscoverFilterConfig{
		IncludeDrafts: true,
		ExcludeModes:  []string{"scope", "meta"},
		ExcludeIDs:    []string{"skill-abc"},
		ExcludeTags:   []string{"deprecated"},
	}
	if err := s.Put(context.Background(), custom); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.IncludeDrafts != custom.IncludeDrafts {
		t.Errorf("IncludeDrafts: got %v, want %v", got.IncludeDrafts, custom.IncludeDrafts)
	}
	if len(got.ExcludeModes) != 2 || got.ExcludeModes[0] != "scope" || got.ExcludeModes[1] != "meta" {
		t.Errorf("ExcludeModes: got %v, want %v", got.ExcludeModes, custom.ExcludeModes)
	}
	if len(got.ExcludeIDs) != 1 || got.ExcludeIDs[0] != "skill-abc" {
		t.Errorf("ExcludeIDs: got %v, want %v", got.ExcludeIDs, custom.ExcludeIDs)
	}
	if len(got.ExcludeTags) != 1 || got.ExcludeTags[0] != "deprecated" {
		t.Errorf("ExcludeTags: got %v, want %v", got.ExcludeTags, custom.ExcludeTags)
	}
}

func TestDiscoverFilterConfigStore_Put_Invalid(t *testing.T) {
	dir := t.TempDir()
	s := NewDiscoverFilterConfigStore(dir)

	tooMany := make([]string, maxExcludeEntries+1)
	for i := range tooMany {
		tooMany[i] = "x"
	}
	invalid := DiscoverFilterConfig{ExcludeModes: tooMany}
	if err := s.Put(context.Background(), invalid); err == nil {
		t.Error("expected validation error for invalid config")
	}
}

// MockDiscoverFilterConfigProvider for testing Discover() with custom filters.
type MockDiscoverFilterConfigProvider struct {
	cfg DiscoverFilterConfig
	err error
}

func (m *MockDiscoverFilterConfigProvider) Get(_ context.Context) (DiscoverFilterConfig, error) {
	return m.cfg, m.err
}

func TestGetDiscoverFilterConfig_Handler(t *testing.T) {
	dir := t.TempDir()
	store := NewDiscoverFilterConfigStore(dir)

	h := NewHandlers(&Service{})
	h.SetDiscoverFilterConfigStore(store)

	req, _ := http.NewRequest("GET", "/api/v1/config/discover-filters", nil)
	rr := httptest.NewRecorder()

	h.GetDiscoverFilterConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var cfg DiscoverFilterConfig
	if err := json.NewDecoder(rr.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.IncludeDrafts {
		t.Error("expected IncludeDrafts=false in defaults")
	}
	if len(cfg.ExcludeModes) != 1 || cfg.ExcludeModes[0] != "scope" {
		t.Errorf("expected ExcludeModes=[scope], got %v", cfg.ExcludeModes)
	}
}

func TestPutDiscoverFilterConfig_Handler(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewDiscoverFilterConfigStore(dir)
	h := NewHandlers(&Service{})
	h.SetDiscoverFilterConfigStore(store)

	custom := DiscoverFilterConfig{
		IncludeDrafts: true,
		ExcludeModes:  []string{"tools"},
		ExcludeIDs:    []string{"s1"},
	}
	body, _ := json.Marshal(custom)
	req, _ := http.NewRequest("PUT", "/api/v1/config/discover-filters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PutDiscoverFilterConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var returned DiscoverFilterConfig
	if err := json.NewDecoder(rr.Body).Decode(&returned); err != nil {
		t.Fatal(err)
	}
	if !returned.IncludeDrafts {
		t.Error("expected IncludeDrafts=true")
	}
}

func TestGetDiscoverFilterConfig_Handler_NoStore(t *testing.T) {
	h := NewHandlers(&Service{})

	req, _ := http.NewRequest("GET", "/api/v1/config/discover-filters", nil)
	rr := httptest.NewRecorder()

	h.GetDiscoverFilterConfig(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}
