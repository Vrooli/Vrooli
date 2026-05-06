package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"brand-manager/domain"
)

// [REQ:BM-REQ-DESIGN-GEN] [REQ:BM-REQ-DESIGN-CONTENT] [REQ:BM-REQ-APPLY-CSS]

func TestGenerateDesignLanguage(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{
		ID:          "b1",
		Name:        "Test Brand",
		Description: "A test brand for design language generation",
		Version:     2,
		Identity: &domain.Identity{
			DisplayName: "Test Brand Display",
			Tagline:     "Testing excellence",
			LogoPath:    "/assets/logo.png",
			FaviconPath: "/assets/favicon.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#3498db",
			Secondary:  "#2ecc71",
			Background: "#ffffff",
			Surface:    "#f5f5f5",
			Text:       "#333333",
		},
		Typography: &domain.Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Open Sans",
			BaseFontSize: "16px",
		},
		Voice: &domain.Voice{
			Tone:     "professional",
			Style:    "concise",
			Keywords: []string{"reliable", "modern"},
		},
		Notes: "Additional design notes here.",
	})

	req := httptest.NewRequest("POST", "/api/v1/brands/b1/design-language", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		BrandID  string `json:"brand_id"`
		Markdown string `json:"markdown"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.BrandID != "b1" {
		t.Errorf("BrandID = %q, want %q", resp.BrandID, "b1")
	}

	md := resp.Markdown

	// Verify content sections exist [REQ:BM-REQ-DESIGN-CONTENT]
	sections := []string{
		"# Test Brand DESIGN.md",
		"## Identity",
		"## Color System",
		"## Typography",
		"## Voice & Personality",
		"## Visual Patterns & Metaphors",
		"## Notes",
	}
	for _, section := range sections {
		if !strings.Contains(md, section) {
			t.Errorf("missing section: %q", section)
		}
	}

	// Verify identity details
	if !strings.Contains(md, "Test Brand Display") {
		t.Error("missing display name in output")
	}

	// Verify color table
	if !strings.Contains(md, "#3498db") {
		t.Error("missing primary color in output")
	}

	// Verify typography
	if !strings.Contains(md, "Inter") {
		t.Error("missing heading font in output")
	}

	// Verify voice
	if !strings.Contains(md, "professional") {
		t.Error("missing tone in output")
	}
	if !strings.Contains(md, "reliable, modern") {
		t.Error("missing keywords in output")
	}

	// Verify notes
	if !strings.Contains(md, "Additional design notes here.") {
		t.Error("missing notes in output")
	}
}

func TestGenerateDesignLanguageNotFound(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("POST", "/api/v1/brands/missing/design-language", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// [REQ:BM-REQ-DESIGN-CONTENT] Tests all voice fields appear in the rendered markdown
func TestDesignLanguageVoiceFields(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{
		ID:      "voice-test",
		Name:    "Voice Test",
		Version: 1,
		Voice: &domain.Voice{
			Tone:     "playful",
			Style:    "conversational",
			Keywords: []string{"friendly", "approachable", "fun"},
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/brands/voice-test/design-language", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Markdown string `json:"markdown"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	md := resp.Markdown

	if !strings.Contains(md, "playful") {
		t.Error("missing tone 'playful' in voice section")
	}
	if !strings.Contains(md, "conversational") {
		t.Error("missing style 'conversational' in voice section")
	}
	if !strings.Contains(md, "friendly") {
		t.Error("missing keyword 'friendly' in voice section")
	}
	if !strings.Contains(md, "approachable") {
		t.Error("missing keyword 'approachable' in voice section")
	}
}

// [REQ:BM-REQ-DESIGN-GEN] Tests that the color table renders all defined colors
func TestDesignLanguageColorTable(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{
		ID:      "color-test",
		Name:    "Color Test",
		Version: 1,
		Colors: &domain.Colors{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Background: "#0000ff",
			Surface:    "#cccccc",
			Text:       "#111111",
		},
	})

	req := httptest.NewRequest("POST", "/api/v1/brands/color-test/design-language", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Markdown string `json:"markdown"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	for _, hex := range []string{"#ff0000", "#00ff00", "#0000ff", "#cccccc", "#111111"} {
		if !strings.Contains(resp.Markdown, hex) {
			t.Errorf("missing color %s in design language output", hex)
		}
	}
}

func TestGenerateDesignLanguageMinimalBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{
		ID:      "b2",
		Name:    "Minimal Brand",
		Version: 1,
	})

	req := httptest.NewRequest("POST", "/api/v1/brands/b2/design-language", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var resp struct {
		Markdown string `json:"markdown"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	// Even minimal brand should have structure
	if !strings.Contains(resp.Markdown, "# Minimal Brand") {
		t.Error("missing brand name heading")
	}
	if !strings.Contains(resp.Markdown, "_Not yet defined._") {
		t.Error("expected 'Not yet defined' for empty facets")
	}
}
