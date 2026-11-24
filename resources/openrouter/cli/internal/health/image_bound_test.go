package health

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

// geminiProPricing mirrors the observed google/gemini-3-pro-image endpoint:
// $0.00012 per image-output token, 32768 reservable completion tokens.
func geminiProPricing() ImageOutputPricing {
	price, _ := new(big.Rat).SetString("0.00012")
	return ImageOutputPricing{PricePerToken: price, EndpointMaxTokens: 32768}
}

func TestBoundImageOutputTokens(t *testing.T) {
	cases := []struct {
		name         string
		controls     ImageControls
		pricing      ImageOutputPricing
		wantTokens   int
		wantReserved int64
		wantErr      string
	}{
		{name: "policy default alone bounds the reservation", controls: ImageControls{MaxOutputTokens: 2048}, pricing: geminiProPricing(), wantTokens: 2048, wantReserved: 245_760},
		{name: "caller ceiling lowers the policy default", controls: ImageControls{MaxOutputTokens: 2048, MaxCostMicros: 100_000}, pricing: geminiProPricing(), wantTokens: 833, wantReserved: 99_960},
		{name: "caller ceiling above policy default keeps the policy default", controls: ImageControls{MaxOutputTokens: 2048, MaxCostMicros: 5_000_000}, pricing: geminiProPricing(), wantTokens: 2048, wantReserved: 245_760},
		{name: "caller ceiling alone sets the bound", controls: ImageControls{MaxCostMicros: 250_000}, pricing: geminiProPricing(), wantTokens: 2083, wantReserved: 249_960},
		{name: "never above what the endpoint can produce", controls: ImageControls{MaxOutputTokens: 100_000}, pricing: geminiProPricing(), wantTokens: 32768, wantReserved: 3_932_160},
		{name: "no controls leaves the request unbounded", controls: ImageControls{}, pricing: geminiProPricing(), wantTokens: 0, wantReserved: 0},
		{name: "ceiling below one token fails", controls: ImageControls{MaxCostMicros: 50}, pricing: geminiProPricing(), wantErr: "below the price of one output token"},
		{name: "ceiling without a published price fails closed", controls: ImageControls{MaxCostMicros: 250_000}, pricing: ImageOutputPricing{}, wantErr: "no image-output token price"},
		{name: "policy default without a price still bounds tokens", controls: ImageControls{MaxOutputTokens: 2048}, pricing: ImageOutputPricing{}, wantTokens: 2048, wantReserved: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BoundImageOutputTokens(tc.controls, tc.pricing)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.MaxTokens != tc.wantTokens || got.ReservedMicros != tc.wantReserved {
				t.Fatalf("got tokens=%d reserved=%d, want tokens=%d reserved=%d", got.MaxTokens, got.ReservedMicros, tc.wantTokens, tc.wantReserved)
			}
			if tc.controls.MaxCostMicros > 0 && got.ReservedMicros > tc.controls.MaxCostMicros {
				t.Fatalf("reservation %d exceeds caller ceiling %d", got.ReservedMicros, tc.controls.MaxCostMicros)
			}
		})
	}
}

// routedImageClient serves endpoint metadata for GET and records the POST.
type routedImageClient struct {
	endpointsBody string
	postStatus    int
	postBody      string
	postURL       string
	posted        []byte
}

func (c *routedImageClient) Do(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(c.endpointsBody))}, nil
	}
	c.postURL = req.URL.String()
	c.posted, _ = io.ReadAll(req.Body)
	status := c.postStatus
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(c.postBody))}, nil
}

// A chat-capable token-priced model: observed google/gemini-3-pro-image shape.
const geminiProEndpoints = `{"data":{"architecture":{"output_modalities":["image","text"]},"endpoints":[
 {"provider_name":"Google AI Studio","max_completion_tokens":32768,"supported_parameters":["max_tokens","seed"],"pricing":{"prompt":"0.000002","image_output":"0.00012"}},
 {"provider_name":"Other","max_completion_tokens":16384,"supported_parameters":["max_tokens"],"pricing":{"prompt":"0.000002","image_output":"0.0001"}}]}}`

// An images-only, token-priced model: observed bytedance-seed/seedream-4.5 shape.
const seedreamEndpoints = `{"data":{"architecture":{"output_modalities":["image"]},"endpoints":[
 {"provider_name":"Seed","max_completion_tokens":3686,"supported_parameters":[],"pricing":{"prompt":"0","image_output":"0.00000958083832335329"}}]}}`

const chatImageOK = `{"choices":[{"message":{"images":[{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]}}],"usage":{"cost":0.134}}`

func TestGenerateImageBoundedRequestGoesThroughChatWithMaxTokens(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{endpointsBody: geminiProEndpoints, postBody: chatImageOK}
	input := filepath.Join(t.TempDir(), "in.png")
	if err := os.WriteFile(input, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	controls := ImageControls{Resolution: "1K", MaxOutputTokens: 2048, MaxCostMicros: 100_000}
	body, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "google/gemini-3-pro-image", "a hat", input, 1, controls)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(client.postURL, "/chat/completions") {
		t.Fatalf("posted to %s", client.postURL)
	}
	var payload struct {
		Model       string            `json:"model"`
		MaxTokens   int               `json:"max_tokens"`
		Modalities  []string          `json:"modalities"`
		ImageConfig map[string]string `json:"image_config"`
		Messages    []struct {
			Content []struct {
				Type string `json:"type"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(client.posted, &payload); err != nil {
		t.Fatal(err)
	}
	// The most expensive endpoint ($0.00012/token) prices the bound so it holds
	// whichever endpoint serves: 100000 micros / 0.00012 = 833 tokens.
	if payload.MaxTokens != 833 || payload.Model != "google/gemini-3-pro-image" || payload.ImageConfig["image_size"] != "1K" || !slices.Equal(payload.Modalities, []string{"image", "text"}) {
		t.Fatalf("payload = %s", client.posted)
	}
	if len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 || payload.Messages[0].Content[1].Type != "image_url" {
		t.Fatalf("messages = %s", client.posted)
	}
	// The result is normalized to the images-endpoint shape with usage intact.
	gateway, err := NormalizeGatewayImageResult(body)
	if err != nil {
		t.Fatal(err)
	}
	var result GatewayImageResult
	if err := json.Unmarshal(gateway, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Images) != 1 || result.Images[0].DataB64 != "aGVsbG8=" || result.Images[0].MediaType != "image/png" || result.ActualCostUSD == nil || *result.ActualCostUSD != 0.134 {
		t.Fatalf("normalized = %s", gateway)
	}
}

func TestGenerateImageWithoutControlsUsesImagesEndpointUnbounded(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{endpointsBody: "not json", postBody: `{"data":[]}`}
	if _, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "bytedance-seed/seedream-4.5", "a hat", "", 1, ImageControls{}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(client.postURL, "/images") || strings.Contains(string(client.posted), "max_tokens") {
		t.Fatalf("posted %s %s", client.postURL, client.posted)
	}
}

func TestGenerateImageImagesOnlyModelUnderCeilingThatCoversItsWorstCase(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{endpointsBody: seedreamEndpoints, postBody: `{"data":[]}`}
	// Worst case 3686 * 0.00000958 = ~$0.0353 fits a $0.25 ceiling: nothing to bound.
	if _, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "bytedance-seed/seedream-4.5", "a hat", "", 1, ImageControls{Resolution: "1K", MaxCostMicros: 250_000}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(client.postURL, "/images") || !strings.Contains(string(client.posted), `"resolution":"1K"`) || strings.Contains(string(client.posted), "max_tokens") {
		t.Fatalf("posted %s %s", client.postURL, client.posted)
	}
}

func TestGenerateImageRefusesCeilingItCannotEnforce(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{endpointsBody: seedreamEndpoints}
	_, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "bytedance-seed/seedream-4.5", "a hat", "", 1, ImageControls{MaxCostMicros: 10_000})
	if err == nil || !strings.Contains(err.Error(), "cannot be enforced") {
		t.Fatalf("err = %v", err)
	}
	if client.posted != nil {
		t.Fatalf("request was sent without a provable bound: %s", client.posted)
	}
}

func TestGenerateImageFailsClosedWhenCallerCeilingCannotBePriced(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{endpointsBody: "not json"}
	_, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "google/gemini-3-pro-image", "a hat", "", 1, ImageControls{MaxCostMicros: 250_000})
	if err == nil || !strings.Contains(err.Error(), "cannot bound image request") {
		t.Fatalf("err = %v", err)
	}
	if client.posted != nil {
		t.Fatalf("request was sent without a provable bound: %s", client.posted)
	}
}

func TestGenerateImageRefusalReportsRequestedReservationAndLimitSource(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &routedImageClient{
		endpointsBody: geminiProEndpoints,
		postStatus:    http.StatusPaymentRequired,
		postBody:      `{"error":{"code":402,"message":"This request's maximum cost exceeds your available credits.","metadata":{"limit_source":"openrouter_key_limit","reason":"weight_exceeds_budget"}}}`,
	}
	_, err := GenerateImage(context.Background(), client, runtime, auth.Credentials{APIKey: "k"}, "google/gemini-3-pro-image", "a hat", "", 1, ImageControls{MaxOutputTokens: 2048})
	providerErr, ok := AsProviderError(err)
	if !ok || providerErr.Code != CodeInsufficientCredits || providerErr.HTTPStatus != http.StatusPaymentRequired {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{"maximum cost exceeds", "limit_source=openrouter_key_limit", "reason=weight_exceeds_budget", "max_tokens=2048", "$0.245760", "unbounded worst case $3.932160"} {
		if !strings.Contains(providerErr.Message, want) {
			t.Errorf("message %q missing %q", providerErr.Message, want)
		}
	}
}

func TestChatImageResponseSurfacesRefusalReportedInA200Body(t *testing.T) {
	body := []byte(`{"error":{"code":429,"message":"rate limited upstream"}}`)
	_, err := chatImageResponseAsImages(body, nil)
	providerErr, ok := AsProviderError(err)
	if !ok || providerErr.Code != CodeRateLimited || !strings.Contains(providerErr.Message, "rate limited upstream") {
		t.Fatalf("err = %v", err)
	}
}

func TestChatImageResponseWithoutImageKeepsUsageForGateway(t *testing.T) {
	out, err := chatImageResponseAsImages([]byte(`{"choices":[{"message":{"content":"sorry"}}],"usage":{"cost":0.01}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"data":[]`) || !strings.Contains(string(out), `"cost":0.01`) {
		t.Fatalf("out = %s", out)
	}
}

func TestDescribeImageRefusalWithoutBoundSaysSo(t *testing.T) {
	got := describeImageRefusal([]byte(`{"error":{"message":"nope"}}`), nil)
	if !strings.Contains(got, "nope") || !strings.Contains(got, "no output-token bound was sent") {
		t.Fatalf("got %q", got)
	}
}
