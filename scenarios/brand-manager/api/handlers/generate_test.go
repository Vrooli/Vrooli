// Tests for AI generation handlers.
// [REQ:BM-REQ-AI-CHAIN] [REQ:BM-REQ-AI-TEXT] [REQ:BM-REQ-AI-IMAGE]
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"brand-manager/aigen"
	"brand-manager/config"
	"brand-manager/domain"
	"brand-manager/handlers"
	"brand-manager/repository/mocks"

	"github.com/gorilla/mux"
)

// mockAIProvider implements aigen.Provider for handler tests. [REQ:BM-REQ-AI-CHAIN]
type mockAIProvider struct {
	name      string
	available bool
	textResp  *aigen.TextResponse
	textErr   error
	imageResp *aigen.ImageResponse
	imageErr  error
}

func (m *mockAIProvider) Name() string                     { return m.name }
func (m *mockAIProvider) Available(_ context.Context) bool { return m.available }
func (m *mockAIProvider) GenerateText(_ context.Context, _ aigen.TextRequest) (*aigen.TextResponse, error) {
	return m.textResp, m.textErr
}
func (m *mockAIProvider) GenerateImage(_ context.Context, _ aigen.ImageRequest) (*aigen.ImageResponse, error) {
	return m.imageResp, m.imageErr
}

func setupGenerateServer(t *testing.T, provider aigen.Provider) (*mux.Router, *mocks.BrandRepository) {
	t.Helper()
	brandRepo := mocks.NewBrandRepository()
	versionRepo := mocks.NewVersionRepository()
	assignRepo := mocks.NewAssignmentRepository()
	assetRepo := mocks.NewAssetRepository()

	cfg := config.Default()
	cfg.AssetBasePath = t.TempDir()

	var counter atomic.Int64
	chain := aigen.NewChain(provider)
	h := handlers.New(brandRepo, versionRepo, assignRepo).
		WithAssets(assetRepo).
		WithConfig(cfg).
		WithAIChain(chain).
		WithIDFunc(func() string {
			return fmt.Sprintf("gen-id-%d", counter.Add(1))
		})

	router := mux.NewRouter()
	h.RegisterRoutes(router)
	return router, brandRepo
}

// TestGenerateBrandElements_Success tests text generation. [REQ:BM-REQ-AI-TEXT]
func TestGenerateBrandElements_Success(t *testing.T) {
	colorJSON := `{"primary":"#3366cc","secondary":"#6699ff","accent":"#ff6633","background":"#ffffff","surface":"#f5f5f5","text":"#1a1a1a","error":"#cc3333"}`
	provider := &mockAIProvider{
		name:      "mock",
		available: true,
		textResp:  &aigen.TextResponse{Text: colorJSON, Provider: "mock", Model: "test-model"},
	}

	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-gen-1", Name: "GenTest", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-gen-1/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["provider"] != "mock" {
		t.Errorf("expected provider 'mock', got %v", resp["provider"])
	}
	if applied, ok := resp["applied"].([]interface{}); !ok || len(applied) == 0 {
		t.Error("expected 'colors' in applied list")
	}
}

// TestGenerateBrandElements_NoProviders tests 503 when no AI configured. [REQ:BM-REQ-AI-CHAIN]
func TestGenerateBrandElements_NoProviders(t *testing.T) {
	brandRepo := mocks.NewBrandRepository()
	versionRepo := mocks.NewVersionRepository()
	assignRepo := mocks.NewAssignmentRepository()

	// Use an empty chain (no providers) to simulate no AI configured
	emptyChain := aigen.NewChain()
	h := handlers.New(brandRepo, versionRepo, assignRepo).
		WithConfig(config.Default()).
		WithAIChain(emptyChain)

	router := mux.NewRouter()
	h.RegisterRoutes(router)

	brand := &domain.Brand{ID: "brand-no-ai", Name: "NoAI", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-no-ai/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// TestGenerateBrandElements_EmptyElements tests validation. [REQ:BM-REQ-AI-TEXT]
func TestGenerateBrandElements_EmptyElements(t *testing.T) {
	provider := &mockAIProvider{name: "mock", available: true}
	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-empty-el", Name: "Empty", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"elements":[]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-empty-el/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestGenerateBrandElements_BrandNotFound tests 404 for missing brand. [REQ:BM-REQ-AI-TEXT]
func TestGenerateBrandElements_BrandNotFound(t *testing.T) {
	provider := &mockAIProvider{name: "mock", available: true}
	router, _ := setupGenerateServer(t, provider)

	body := `{"elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/nonexistent/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// TestGenerateBrandElements_TypographyAndVoice tests multi-element generation. [REQ:BM-REQ-AI-TEXT]
func TestGenerateBrandElements_TypographyAndVoice(t *testing.T) {
	typoJSON := `{"heading_font":"Inter","body_font":"Open Sans","mono_font":"Fira Code","base_font_size":"16px"}`
	provider := &mockAIProvider{
		name:      "mock",
		available: true,
		textResp:  &aigen.TextResponse{Text: typoJSON, Provider: "mock", Model: "m"},
	}

	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-multi", Name: "Multi", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"elements":["typography"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-multi/generate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestGenerateBrandImage_Success tests image generation. [REQ:BM-REQ-AI-IMAGE]
func TestGenerateBrandImage_Success(t *testing.T) {
	provider := &mockAIProvider{
		name:      "img-mock",
		available: true,
		imageResp: &aigen.ImageResponse{Data: []byte("fakepng"), MimeType: "image/png", Provider: "img-mock", Model: "dalle"},
	}

	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-img-1", Name: "ImgTest", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"type":"logo"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-img-1/generate/image", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["type"] != "logo" {
		t.Errorf("expected type 'logo', got %v", resp["type"])
	}
}

// TestGenerateBrandImage_InvalidType tests type validation. [REQ:BM-REQ-AI-IMAGE]
func TestGenerateBrandImage_InvalidType(t *testing.T) {
	provider := &mockAIProvider{name: "mock", available: true}
	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-img-bad", Name: "BadType", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"type":"banner"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-img-bad/generate/image", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// TestGenerateBrandImage_ProviderError tests image generation error. [REQ:BM-REQ-AI-IMAGE]
func TestGenerateBrandImage_ProviderError(t *testing.T) {
	provider := &mockAIProvider{
		name:      "fail-img",
		available: true,
		imageErr:  fmt.Errorf("image generation failed"),
	}

	router, brandRepo := setupGenerateServer(t, provider)

	brand := &domain.Brand{ID: "brand-img-fail", Name: "FailImg", Version: 1}
	brandRepo.Create(context.Background(), brand)

	body := `{"type":"favicon"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-img-fail/generate/image", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}
