package app

import (
	"encoding/base64"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/policy"
)

func TestResolvePromptPrefersFlag(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("hello", "", []string{"ignored"}, strings.NewReader("stdin"))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestDecodeOneShotBYOKRequestIsTypedBoundedAndRedacted(t *testing.T) {
	request, err := decodeOneShotBYOK(strings.NewReader(`{"api_key":"private-key","role":"summarize.default","prompt":"hello","max_tokens":64}`))
	if err != nil {
		t.Fatal(err)
	}
	if request.APIKey != "private-key" || request.Role != "summarize.default" {
		t.Fatalf("request=%+v", request)
	}
	for _, bad := range []string{`{"api_key":"private-key","prompt":"hello","max_tokens":64}`, strings.Repeat("x", (1<<20)+1)} {
		_, err := decodeOneShotBYOK(strings.NewReader(bad))
		if err == nil || strings.Contains(err.Error(), "private-key") {
			t.Fatalf("unsafe invalid request error: %v", err)
		}
	}
}

func TestResolvePromptUsesTrailingArgs(t *testing.T) {
	t.Parallel()

	got, err := resolvePrompt("", "", []string{"hello", "world"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestResolvePromptUsesFile(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/prompt.txt"
	if err := os.WriteFile(path, []byte("from file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolvePrompt("", path, nil, strings.NewReader(""))
	if err != nil {
		t.Fatalf("resolvePrompt() error = %v", err)
	}
	if got != "from file" {
		t.Fatalf("resolvePrompt() = %q", got)
	}
}

func TestExtractPrimaryText(t *testing.T) {
	t.Parallel()

	got, err := extractPrimaryText([]byte(`{"choices":[{"message":{"content":"hello world"}}]}`))
	if err != nil {
		t.Fatalf("extractPrimaryText() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("extractPrimaryText() = %q", got)
	}
}

func TestImageResultFormatKeepsRawDefaultAndSelectsGatewayEnvelope(t *testing.T) {
	raw := []byte(`{"data":[{"b64_json":"aGVsbG8="}],"usage":{"cost":0}}`)
	got, err := formatImageResponse(raw, "raw")
	if err != nil || string(got) != string(raw) {
		t.Fatalf("raw format = %s, err=%v", got, err)
	}
	got, err = formatImageResponse(raw, "gateway-v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `"schema_version":1`) || !strings.Contains(string(got), `"actual_cost_usd":0`) {
		t.Fatalf("gateway format = %s", got)
	}
	if err := validateImageResultFormat("unknown"); err == nil {
		t.Fatal("unsupported result format accepted")
	}
}

func TestResolveImageInputBuildsBoundedProviderNeutralPayload(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("image"))
	selected := policy.ResolvedPolicyModel{Model: "vision-model", Modalities: policy.Modalities{Input: []policy.Modality{"text", "image"}}}
	prompt, images, err := resolveImageInput(strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"`+data+`"}]}`), selected)
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "describe" || len(images) != 1 || images[0].MediaType != "image/png" || images[0].DataB64 != data {
		t.Fatalf("unexpected decoded input: prompt=%q images=%+v", prompt, images)
	}
}

func TestResolveImageInputRejectsRoleWithoutImageModality(t *testing.T) {
	selected := policy.ResolvedPolicyModel{Model: "text-only", Modalities: policy.Modalities{Input: []policy.Modality{"text"}}}
	_, _, err := resolveImageInput(strings.NewReader(`{"prompt":"describe","images":[{"media_type":"image/png","data_b64":"aGVsbG8="}]}`), selected)
	if err == nil || !strings.Contains(err.Error(), "image_input.capability_mismatch") {
		t.Fatalf("expected typed capability mismatch, got %v", err)
	}
}

func TestImageControlsForCarriesRoleBoundAndCallerCeiling(t *testing.T) {
	maxTokens := 2048
	resolved := policy.ResolvedPolicyModel{Model: "google/gemini-3-pro-image", RequestDefaults: &policy.RequestDefaults{Resolution: "1K", MaxTokens: &maxTokens}}
	controls, err := imageControlsFor(resolved, 0.25)
	if err != nil {
		t.Fatal(err)
	}
	if controls.Resolution != "1K" || controls.MaxOutputTokens != 2048 || controls.MaxCostMicros != 250_000 {
		t.Fatalf("controls = %+v", controls)
	}
	bare, err := imageControlsFor(policy.ResolvedPolicyModel{}, 0)
	if err != nil || bare.MaxOutputTokens != 0 || bare.MaxCostMicros != 0 {
		t.Fatalf("bare controls = %+v err=%v", bare, err)
	}
	for _, bad := range []float64{-0.01, math.NaN(), math.Inf(1)} {
		if _, err := imageControlsFor(resolved, bad); err == nil {
			t.Fatalf("max-cost-usd %v accepted", bad)
		}
	}
}

// The shipped policy must bound every image role whose primary model is
// token-priced at a premium: without request_defaults.max_tokens OpenRouter
// reserves the endpoint's full completion allowance (several dollars).
func TestShippedPolicyBoundsPremiumTokenPricedImageRoles(t *testing.T) {
	p, err := policy.LoadFile("../../../model-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"image.edit.identity", "image.generate.quality"} {
		resolved, err := p.ResolveRole(role)
		if err != nil {
			t.Fatal(err)
		}
		if resolved.RequestDefaults == nil || resolved.RequestDefaults.MaxTokens == nil || *resolved.RequestDefaults.MaxTokens <= 0 || resolved.RequestDefaults.Resolution == "" {
			t.Errorf("%s (%s) declares no output-token bound/resolution: %+v", role, resolved.Model, resolved.RequestDefaults)
		}
	}
}
