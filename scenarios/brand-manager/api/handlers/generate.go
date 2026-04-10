// Package handlers - AI-powered brand element generation.
// [REQ:BM-REQ-AI-CHAIN] [REQ:BM-REQ-AI-TEXT] [REQ:BM-REQ-AI-IMAGE]
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"brand-manager/aigen"
	"brand-manager/domain"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GenerateRequest specifies what brand elements to generate.
type GenerateRequest struct {
	Elements []string `json:"elements"` // "colors", "typography", "voice"
	Model    string   `json:"model,omitempty"`
}

// GenerateResponse contains generated brand elements and metadata.
type GenerateResponse struct {
	BrandID   string                 `json:"brand_id"`
	Generated map[string]interface{} `json:"generated"`
	Provider  string                 `json:"provider"`
	Model     string                 `json:"model"`
	Applied   []string               `json:"applied"` // which elements were applied to brand
}

// GenerateImageRequest specifies an image to generate.
type GenerateImageRequest struct {
	Type  string `json:"type"` // "logo", "favicon"
	Model string `json:"model,omitempty"`
}

// GenerateImageResponse contains the generated image metadata.
type GenerateImageResponse struct {
	BrandID  string `json:"brand_id"`
	AssetID  string `json:"asset_id"`
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	FilePath string `json:"file_path"`
}

// aiChain returns the configured AI provider chain, or nil if no providers are available.
func (h *Handlers) aiChain() *aigen.Chain {
	if h.chain != nil {
		return h.chain
	}

	var providers []aigen.Provider

	if h.cfg.OllamaURL != "" {
		providers = append(providers, aigen.NewOllamaProvider(h.cfg.OllamaURL, h.cfg.OllamaModel))
	}
	if h.cfg.OpenRouterAPIKey != "" {
		providers = append(providers, aigen.NewOpenRouterProvider(
			h.cfg.OpenRouterAPIKey,
			h.cfg.OpenRouterTextModel,
			h.cfg.OpenRouterImageModel,
		))
	}

	if len(providers) == 0 {
		return nil
	}
	return aigen.NewChain(providers...)
}

// requireAIChain resolves the AI chain and writes an error if unavailable.
// Returns (nil, true) when an error was written, so the caller can return early.
func (h *Handlers) requireAIChain(w http.ResponseWriter) (*aigen.Chain, bool) {
	chain := h.aiChain()
	if chain == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "no AI providers configured. Set OLLAMA_URL or OPENROUTER_API_KEY.",
		})
		return nil, true
	}
	return chain, false
}

// GenerateBrandElements handles POST /api/v1/brands/{id}/generate.
// Uses the AIProviderChain to generate brand elements (colors, typography, voice).
// [REQ:BM-REQ-AI-CHAIN] [REQ:BM-REQ-AI-TEXT]
func (h *Handlers) GenerateBrandElements(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Elements) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "elements is required (e.g. [\"colors\", \"typography\", \"voice\"])",
		})
		return
	}

	chain, done := h.requireAIChain(w)
	if done {
		return
	}

	if !chain.Available(r.Context()) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "no AI providers are currently available",
		})
		return
	}

	generated := make(map[string]interface{})
	var applied []string
	var lastProvider, lastModel string

	for _, elem := range req.Elements {
		var prompt string
		switch strings.ToLower(elem) {
		case "colors":
			prompt = aigen.ColorPrompt(brand.Name, brand.Description, brand.Notes)
		case "typography":
			prompt = aigen.TypographyPrompt(brand.Name, brand.Description, brand.Notes)
		case "voice":
			prompt = aigen.VoicePrompt(brand.Name, brand.Description, brand.Notes)
		default:
			generated[elem] = map[string]string{"error": fmt.Sprintf("unsupported element: %s", elem)}
			continue
		}

		resp, err := chain.GenerateText(r.Context(), aigen.TextRequest{
			Prompt: prompt,
			Model:  req.Model,
		})
		if err != nil {
			generated[elem] = map[string]string{"error": err.Error()}
			continue
		}

		lastProvider = resp.Provider
		lastModel = resp.Model

		parsed, err := parseGeneratedJSON(resp.Text)
		if err != nil {
			generated[elem] = map[string]interface{}{
				"raw":   resp.Text,
				"error": "failed to parse AI response as JSON",
			}
			continue
		}

		generated[elem] = parsed
		if applyErr := applyGeneratedElement(brand, elem, parsed); applyErr == nil {
			applied = append(applied, elem)
		}
	}

	if len(applied) > 0 {
		brand.Version++
		if err := h.brands.Update(r.Context(), brand); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save generated elements"})
			return
		}
	}

	writeJSON(w, http.StatusOK, GenerateResponse{
		BrandID:   brandID,
		Generated: generated,
		Provider:  lastProvider,
		Model:     lastModel,
		Applied:   applied,
	})
}

// GenerateBrandImage handles POST /api/v1/brands/{id}/generate/image.
// Uses the AIProviderChain to generate brand images (logo, favicon).
// [REQ:BM-REQ-AI-CHAIN] [REQ:BM-REQ-AI-IMAGE]
func (h *Handlers) GenerateBrandImage(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	var req GenerateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Type != "logo" && req.Type != "favicon" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "type must be 'logo' or 'favicon'",
		})
		return
	}

	chain, done := h.requireAIChain(w)
	if done {
		return
	}

	primaryColor := ""
	if brand.Colors != nil {
		primaryColor = brand.Colors.Primary
	}

	var prompt string
	var width, height int
	switch req.Type {
	case "logo":
		prompt = aigen.LogoPrompt(brand.Name, brand.Description, primaryColor)
		width, height = 512, 512
	case "favicon":
		prompt = aigen.FaviconPrompt(brand.Name, primaryColor)
		width, height = 64, 64
	}

	resp, err := chain.GenerateImage(r.Context(), aigen.ImageRequest{
		Prompt: prompt,
		Model:  req.Model,
		Width:  width,
		Height: height,
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("%s-%s.png", req.Type, brandID[:8])
	assetDir := filepath.Join(h.cfg.AssetBasePath, brandID)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create asset directory"})
		return
	}

	filePath := filepath.Join(assetDir, filename)
	if err := os.WriteFile(filePath, resp.Data, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to write image file"})
		return
	}

	assetID := uuid.New().String()
	if h.assets != nil {
		asset := &domain.Asset{
			ID:       assetID,
			BrandID:  brandID,
			Filename: filename,
			MimeType: resp.MimeType,
			FilePath: filePath,
			Size:     int64(len(resp.Data)),
		}
		if err := h.assets.Create(r.Context(), asset); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save asset record"})
			return
		}
	}

	writeJSON(w, http.StatusOK, GenerateImageResponse{
		BrandID:  brandID,
		AssetID:  assetID,
		Type:     req.Type,
		Provider: resp.Provider,
		Model:    resp.Model,
		FilePath: filePath,
	})
}

// parseGeneratedJSON extracts JSON from an AI response that may contain markdown fences.
func parseGeneratedJSON(text string) (map[string]interface{}, error) {
	text = strings.TrimSpace(text)

	// Strip markdown code fences
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Find JSON object boundaries
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// applyGeneratedElement applies parsed generation results to the brand struct.
func applyGeneratedElement(brand *domain.Brand, element string, data map[string]interface{}) error {
	switch element {
	case "colors":
		colors := &domain.Colors{}
		if v, ok := data["primary"].(string); ok {
			colors.Primary = v
		}
		if v, ok := data["secondary"].(string); ok {
			colors.Secondary = v
		}
		if v, ok := data["accent"].(string); ok {
			colors.Accent = v
		}
		if v, ok := data["background"].(string); ok {
			colors.Background = v
		}
		if v, ok := data["surface"].(string); ok {
			colors.Surface = v
		}
		if v, ok := data["text"].(string); ok {
			colors.Text = v
		}
		if v, ok := data["error"].(string); ok {
			colors.Error = v
		}
		brand.Colors = colors

	case "typography":
		typo := &domain.Typography{}
		if v, ok := data["heading_font"].(string); ok {
			typo.HeadingFont = v
		}
		if v, ok := data["body_font"].(string); ok {
			typo.BodyFont = v
		}
		if v, ok := data["mono_font"].(string); ok {
			typo.MonoFont = v
		}
		if v, ok := data["base_font_size"].(string); ok {
			typo.BaseFontSize = v
		}
		brand.Typography = typo

	case "voice":
		voice := &domain.Voice{}
		if v, ok := data["tone"].(string); ok {
			voice.Tone = v
		}
		if v, ok := data["style"].(string); ok {
			voice.Style = v
		}
		if kw, ok := data["keywords"].([]interface{}); ok {
			for _, k := range kw {
				if s, ok := k.(string); ok {
					voice.Keywords = append(voice.Keywords, s)
				}
			}
		}
		brand.Voice = voice

	default:
		return fmt.Errorf("unsupported element: %s", element)
	}
	return nil
}
