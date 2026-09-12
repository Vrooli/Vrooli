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

func TestDefaultBudgetConfig(t *testing.T) {
	cfg := DefaultBudgetConfig()

	if cfg.Minor != 4000 {
		t.Errorf("Minor = %d, want 4000", cfg.Minor)
	}
	if cfg.Moderate != 8000 {
		t.Errorf("Moderate = %d, want 8000", cfg.Moderate)
	}
	if cfg.Major != 12000 {
		t.Errorf("Major = %d, want 12000", cfg.Major)
	}
	if cfg.Architectural != 18000 {
		t.Errorf("Architectural = %d, want 18000", cfg.Architectural)
	}

	// Must be ascending
	if err := ValidateBudgetConfig(cfg); err != nil {
		t.Errorf("default config should be valid, got: %v", err)
	}
}

func TestValidateBudgetConfig_Valid(t *testing.T) {
	cfg := BudgetConfig{Minor: 1000, Moderate: 2000, Major: 3000, Architectural: 5000}
	if err := ValidateBudgetConfig(cfg); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
}

func TestValidateBudgetConfig_Invalid(t *testing.T) {
	tests := []struct {
		name string
		cfg  BudgetConfig
	}{
		{"zero minor", BudgetConfig{Minor: 0, Moderate: 2000, Major: 3000, Architectural: 5000}},
		{"negative", BudgetConfig{Minor: -1, Moderate: 2000, Major: 3000, Architectural: 5000}},
		{"exceeds max", BudgetConfig{Minor: 1000, Moderate: 2000, Major: 3000, Architectural: 300000}},
		{"not ascending", BudgetConfig{Minor: 5000, Moderate: 2000, Major: 3000, Architectural: 10000}},
		{"equal tiers", BudgetConfig{Minor: 1000, Moderate: 1000, Major: 3000, Architectural: 5000}},
	}

	for _, tc := range tests {
		if err := ValidateBudgetConfig(tc.cfg); err == nil {
			t.Errorf("%s: expected validation error, got nil", tc.name)
		}
	}
}

func TestBudgetConfig_ForTier(t *testing.T) {
	cfg := DefaultBudgetConfig()

	tests := []struct {
		tier string
		want int
		ok   bool
	}{
		{"minor", 4000, true},
		{"moderate", 8000, true},
		{"major", 12000, true},
		{"architectural", 18000, true},
		{"unknown", 0, false},
		{"", 0, false},
	}

	for _, tc := range tests {
		got, ok := cfg.ForTier(tc.tier)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ForTier(%q) = (%d, %v), want (%d, %v)", tc.tier, got, ok, tc.want, tc.ok)
		}
	}
}

func TestBudgetConfig_ToMap(t *testing.T) {
	cfg := DefaultBudgetConfig()
	m := cfg.ToMap()

	if m["minor"] != 4000 || m["moderate"] != 8000 || m["major"] != 12000 || m["architectural"] != 18000 {
		t.Errorf("ToMap returned unexpected values: %v", m)
	}
}

func TestBudgetConfigStore_Get_NoFile(t *testing.T) {
	dir := t.TempDir()
	s := NewBudgetConfigStore(dir)

	cfg, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	defaults := DefaultBudgetConfig()
	if cfg != defaults {
		t.Errorf("expected defaults, got %+v", cfg)
	}
}

func TestBudgetConfigStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	// Create the config/ subdirectory
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	s := NewBudgetConfigStore(dir)

	custom := BudgetConfig{Minor: 2000, Moderate: 5000, Major: 10000, Architectural: 20000}
	if err := s.Put(context.Background(), custom); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, err := s.Get(context.Background())
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != custom {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, custom)
	}
}

func TestBudgetConfigStore_Put_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	s := NewBudgetConfigStore(dir)

	invalid := BudgetConfig{Minor: 0, Moderate: 100, Major: 200, Architectural: 300}
	if err := s.Put(context.Background(), invalid); err == nil {
		t.Error("expected validation error for invalid config")
	}
}

// MockBudgetConfigProvider for testing Discover() with custom budgets.
type MockBudgetConfigProvider struct {
	cfg BudgetConfig
	err error
}

func (m *MockBudgetConfigProvider) Get(_ context.Context) (BudgetConfig, error) {
	return m.cfg, m.err
}

func TestGetBudgetConfig_Handler(t *testing.T) {
	dir := t.TempDir()
	store := NewBudgetConfigStore(dir)

	h := NewHandlers(&Service{})
	h.SetBudgetConfigStore(store)

	req, _ := http.NewRequest("GET", "/api/v1/config/budgets", nil)
	rr := httptest.NewRecorder()

	h.GetBudgetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var cfg BudgetConfig
	if err := json.NewDecoder(rr.Body).Decode(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg != DefaultBudgetConfig() {
		t.Errorf("expected defaults, got %+v", cfg)
	}
}

func TestPutBudgetConfig_Handler(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}

	store := NewBudgetConfigStore(dir)
	h := NewHandlers(&Service{})
	h.SetBudgetConfigStore(store)

	custom := BudgetConfig{Minor: 3000, Moderate: 6000, Major: 9000, Architectural: 15000}
	body, _ := json.Marshal(custom)
	req, _ := http.NewRequest("PUT", "/api/v1/config/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PutBudgetConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var returned BudgetConfig
	if err := json.NewDecoder(rr.Body).Decode(&returned); err != nil {
		t.Fatal(err)
	}
	if returned != custom {
		t.Errorf("expected %+v, got %+v", custom, returned)
	}
}

func TestPutBudgetConfig_Handler_Invalid(t *testing.T) {
	dir := t.TempDir()
	store := NewBudgetConfigStore(dir)
	h := NewHandlers(&Service{})
	h.SetBudgetConfigStore(store)

	invalid := BudgetConfig{Minor: -1, Moderate: 100, Major: 200, Architectural: 300}
	body, _ := json.Marshal(invalid)
	req, _ := http.NewRequest("PUT", "/api/v1/config/budgets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.PutBudgetConfig(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestGetBudgetConfig_Handler_NoStore(t *testing.T) {
	h := NewHandlers(&Service{})

	req, _ := http.NewRequest("GET", "/api/v1/config/budgets", nil)
	rr := httptest.NewRecorder()

	h.GetBudgetConfig(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rr.Code)
	}
}
