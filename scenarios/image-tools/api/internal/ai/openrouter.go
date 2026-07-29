package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"
)

// OpenRouter remains Image Tools' last-resort cloud backend, but the resource
// owns every provider concern: policy resolution, credentials, endpoint choice,
// transport, and provider errors. Image Tools owns only its image operation,
// durable job, and output-blob responsibilities.
type resolvedImageRole struct {
	Model string `json:"model"`
}

type imageGenerator func(context.Context, string, string, string, int) ([]byte, error)

type openRouterProvider struct {
	resolveRole   func(context.Context, string) (resolvedImageRole, error)
	generateImage imageGenerator
}

func newOpenRouterProvider() *openRouterProvider {
	return &openRouterProvider{resolveRole: resolveOpenRouterRole, generateImage: generateOpenRouterImage}
}

func (p *openRouterProvider) Name() string { return models.BackendOpenRouter }
func (p *openRouterProvider) Operations() []string {
	return []string{"text_to_image", "image_to_image", "edit_instruct"}
}
func (p *openRouterProvider) Standalone() bool { return true }
func (p *openRouterProvider) IsCloud() bool    { return true }
func (p *openRouterProvider) GPUCapable() bool { return false }
func (p *openRouterProvider) Available(context.Context) bool {
	return p != nil && p.generateImage != nil
}

func (p *openRouterProvider) Availability(ctx context.Context) backends.Availability {
	if p.Available(ctx) {
		return backends.Availability{Available: true, Detail: "OpenRouter resource-backed cloud image generation", Provision: "configure resource-openrouter credentials and image policy roles"}
	}
	return backends.Availability{Available: false, Detail: "OpenRouter resource image transport is not configured", Provision: "configure resource-openrouter image generation"}
}

func roleForRequest(req backends.Request) string {
	if role := strings.TrimSpace(req.Params["openrouter_role"]); role != "" {
		return role
	}
	if req.Operation == "text_to_image" {
		return "image.generate.default"
	}
	return "image.edit.default"
}

func (p *openRouterProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: openrouter backend requires a local output path")
	}
	prompt := strings.TrimSpace(req.Params["prompt"])
	if prompt == "" {
		return backends.Result{}, fmt.Errorf("ai: openrouter image generation requires a prompt")
	}
	if p == nil || p.generateImage == nil {
		return backends.Result{}, fmt.Errorf("ai: resource-openrouter image transport is not configured")
	}
	role := roleForRequest(req)
	resolved, err := p.resolveRole(ctx, role)
	if err != nil {
		return backends.Result{}, err
	}
	inputFile := ""
	if req.Operation != "text_to_image" {
		inputFile, err = technique.Input0(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: openrouter source image: %w", err)
		}
	}
	raw, err := p.generateImage(ctx, role, prompt, inputFile, 1)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: resource-openrouter image generate: %w", err)
	}
	image, mediaType, err := decodeOpenRouterImage(raw)
	if err != nil {
		return backends.Result{}, err
	}
	if err := os.WriteFile(req.Output.LocalPath, image, 0o644); err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter write output: %w", err)
	}
	meta := map[string]string{"backend": models.BackendOpenRouter, "model": resolved.Model, "role": role}
	if mediaType != "" {
		meta["media_type"] = mediaType
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Tier: backends.TierBYOK, Meta: meta}, nil
}

func generateOpenRouterImage(ctx context.Context, role, prompt, inputFile string, outputCount int) ([]byte, error) {
	args := []string{"images", "generate", "--role", role, "--prompt", prompt, "--output-count", fmt.Sprintf("%d", outputCount)}
	if strings.TrimSpace(inputFile) != "" {
		args = append(args, "--input-file", inputFile)
	}
	return exec.CommandContext(ctx, "resource-openrouter", args...).Output()
}

type openRouterImagesResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func decodeOpenRouterImage(body []byte) ([]byte, string, error) {
	var parsed openRouterImagesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("ai: openrouter decode response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, "", fmt.Errorf("ai: openrouter error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 || strings.TrimSpace(parsed.Data[0].B64JSON) == "" {
		return nil, "", fmt.Errorf("ai: openrouter returned no image (the model may not support image output)")
	}
	image, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return nil, "", fmt.Errorf("ai: openrouter decode image data: %w", err)
	}
	if len(image) == 0 {
		return nil, "", fmt.Errorf("ai: openrouter image is empty")
	}
	return image, strings.TrimSpace(parsed.Data[0].MediaType), nil
}

func resolveOpenRouterRole(ctx context.Context, role string) (resolvedImageRole, error) {
	out, err := exec.CommandContext(ctx, "resource-openrouter", "policy", "resolve", "--role", role, "--json").Output()
	if err != nil {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter policy resolve %q: %w", role, err)
	}
	var resolved resolvedImageRole
	if err := json.Unmarshal(out, &resolved); err != nil {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter decode policy resolve %q: %w", role, err)
	}
	if strings.TrimSpace(resolved.Model) == "" {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter policy resolve %q returned no model", role)
	}
	return resolved, nil
}
