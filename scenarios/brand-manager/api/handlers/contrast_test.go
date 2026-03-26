package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/contrast"
	"brand-manager/repository/mocks"
)

// [REQ:BM-REQ-WCAG-CALC] [REQ:BM-REQ-WCAG-VALIDATE] [REQ:BM-REQ-WCAG-REJECT]

// newContrastHandler creates a minimal handler for contrast tests.
func newContrastHandler() *Handlers {
	return New(&mocks.BrandRepository{}, &mocks.VersionRepository{}, &mocks.AssignmentRepository{})
}

func TestCheckContrastEndpoint(t *testing.T) {
	h := newContrastHandler()

	body := `{"foreground":"#000000","background":"#FFFFFF"}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/check", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckContrast(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result contrast.PairResult
	json.NewDecoder(rr.Body).Decode(&result)
	if result.Ratio < 21.0 {
		t.Errorf("ratio = %.2f, want >= 21.0", result.Ratio)
	}
	if !result.AANormal {
		t.Error("expected AANormal = true")
	}
	if !result.AALarge {
		t.Error("expected AALarge = true")
	}
}

func TestCheckContrastLowRatio(t *testing.T) {
	h := newContrastHandler()

	body := `{"foreground":"#FFFFFF","background":"#FFFFFF"}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/check", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckContrast(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result contrast.PairResult
	json.NewDecoder(rr.Body).Decode(&result)
	if result.AANormal {
		t.Error("expected AANormal = false for white-on-white")
	}
}

func TestCheckContrastMissingFields(t *testing.T) {
	h := newContrastHandler()

	body := `{"foreground":"#000000"}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/check", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckContrast(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestCheckContrastInvalidColor(t *testing.T) {
	h := newContrastHandler()

	body := `{"foreground":"not-a-color","background":"#FFFFFF"}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/check", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckContrast(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestCheckBrandContrastEndpoint(t *testing.T) {
	h := newContrastHandler()

	body := `{
		"primary":"#1a365d",
		"secondary":"#2d3748",
		"accent":"#8B0000",
		"background":"#FFFFFF",
		"surface":"#F7FAFC",
		"text":"#1A202C"
	}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/brand", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckBrandContrast(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result contrast.BrandCheckResult
	json.NewDecoder(rr.Body).Decode(&result)
	if !result.PassAll {
		t.Error("expected PassAll = true for accessible brand")
		for _, p := range result.Pairs {
			t.Logf("  %s on %s: ratio=%.2f pass=%v", p.Foreground, p.Background, p.Ratio, p.AANormal)
		}
	}
	if len(result.Pairs) != 5 {
		t.Errorf("expected 5 pairs, got %d", len(result.Pairs))
	}
}

func TestCheckBrandContrastFailingPalette(t *testing.T) {
	h := newContrastHandler()

	body := `{
		"primary":"#CCCCCC",
		"accent":"#EEEEEE",
		"background":"#FFFFFF",
		"surface":"#F0F0F0",
		"text":"#AAAAAA"
	}`
	req := httptest.NewRequest("POST", "/api/v1/contrast/brand", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.CheckBrandContrast(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var result contrast.BrandCheckResult
	json.NewDecoder(rr.Body).Decode(&result)
	if result.PassAll {
		t.Error("expected PassAll = false for inaccessible palette")
	}
}
