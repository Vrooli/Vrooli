package health

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

type captureHTTPClient struct {
	body []byte
}

func (c *captureHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var err error
	c.body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"ok"}}]}`)),
		Header:     make(http.Header),
	}, nil
}

func TestGenerateEncodesImageContentPartsAtProviderBoundary(t *testing.T) {
	client := &captureHTTPClient{}
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	data := base64.StdEncoding.EncodeToString([]byte("image"))
	if _, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "vision-model", "describe", nil, 32, nil, []ImageInput{{MediaType: "image/png", DataB64: data}}); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Messages []struct {
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(bytes.NewReader(client.body)).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Messages) != 1 || len(payload.Messages[0].Content) != 2 {
		t.Fatalf("unexpected message payload: %s", client.body)
	}
	if payload.Messages[0].Content[1].Type != "image_url" || !strings.HasPrefix(payload.Messages[0].Content[1].ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("image content part = %+v", payload.Messages[0].Content[1])
	}
}

// A nil temperature must leave the key off the wire entirely. OpenRouter omits
// absent parameters rather than substituting a default, and several upstream
// families reject the field, so "unset" has to be expressible.
func TestGenerateOmitsTemperatureWhenUnset(t *testing.T) {
	client := &captureHTTPClient{}
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	if _, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "text-model", "write", nil, 0, nil, nil); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(client.body, &payload); err != nil {
		t.Fatal(err)
	}
	if _, present := payload["temperature"]; present {
		t.Fatalf("temperature must be absent when unset, body = %s", client.body)
	}
	if _, present := payload["max_tokens"]; present {
		t.Fatalf("max_tokens must be absent when unset, body = %s", client.body)
	}
}

// An explicit deterministic request is a pointer to 0, not an absent field.
// Before this contract existed the flag defaulted to 0.7 and was always
// serialised, so every gateway fallback ran non-deterministically.
func TestGenerateSerialisesExplicitZeroTemperature(t *testing.T) {
	client := &captureHTTPClient{}
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	zero := 0.0
	if _, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "text-model", "classify", &zero, 512, nil, nil); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Temperature *float64 `json:"temperature"`
		MaxTokens   int      `json:"max_tokens"`
	}
	if err := json.Unmarshal(client.body, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Temperature == nil || *payload.Temperature != 0 {
		t.Fatalf("temperature = %v, want explicit 0; body = %s", payload.Temperature, client.body)
	}
	if payload.MaxTokens != 512 {
		t.Fatalf("max_tokens = %d, want 512", payload.MaxTokens)
	}
}

type statusHTTPClient struct {
	status int
	header http.Header
	body   io.ReadCloser
	err    error
}

func (c *statusHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if c.err != nil {
		return nil, c.err
	}
	header := c.header
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{StatusCode: c.status, Header: header, Body: c.body}, nil
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failingReadCloser) Close() error             { return nil }

func TestGenerateClassifiesProviderStatusWithObservedRetryAfter(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	cases := []struct {
		name       string
		status     int
		retryAfter string
		body       string
		wantCode   string
	}{
		{name: "402 is account credits", status: http.StatusPaymentRequired, body: `{"error":{"message":"insufficient credits"}}`, wantCode: CodeInsufficientCredits},
		{name: "429 is a rate limit", status: http.StatusTooManyRequests, retryAfter: "17", body: `{"error":{"message":"slow down"}}`, wantCode: CodeRateLimited},
		{name: "503 is an overload", status: http.StatusServiceUnavailable, retryAfter: "5", body: `{"error":{"message":"upstream overloaded"}}`, wantCode: CodeProviderOverloaded},
		{name: "504 is an overload", status: http.StatusGatewayTimeout, body: `{"error":{"message":"gateway timeout"}}`, wantCode: CodeProviderOverloaded},
		{name: "other statuses stay generic", status: http.StatusBadRequest, body: `{"error":{"message":"bad request"}}`, wantCode: CodeProviderFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			header := make(http.Header)
			if tc.retryAfter != "" {
				header.Set("Retry-After", tc.retryAfter)
			}
			client := &statusHTTPClient{status: tc.status, header: header, body: io.NopCloser(strings.NewReader(tc.body))}
			_, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "text-model", "hi", nil, 0, nil, nil)
			providerErr, ok := AsProviderError(err)
			if !ok {
				t.Fatalf("error = %T %v, want *ProviderError", err, err)
			}
			if providerErr.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", providerErr.Code, tc.wantCode)
			}
			if providerErr.HTTPStatus != tc.status {
				t.Fatalf("http status = %d, want %d", providerErr.HTTPStatus, tc.status)
			}
			if providerErr.RetryAfter != tc.retryAfter {
				t.Fatalf("retry-after = %q, want %q (observed value preserved verbatim)", providerErr.RetryAfter, tc.retryAfter)
			}
			line, markerErr := providerErr.MarkerLine()
			if markerErr != nil {
				t.Fatalf("MarkerLine: %v", markerErr)
			}
			if !strings.HasPrefix(string(line), ProviderErrorMarker) || !strings.Contains(string(line), `"code":"`+tc.wantCode+`"`) {
				t.Fatalf("marker = %s", line)
			}
		})
	}
}

func TestGenerateClassifiesStreamFailureAfterSuccess(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &statusHTTPClient{status: http.StatusOK, body: failingReadCloser{}}
	_, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "text-model", "hi", nil, 0, nil, nil)
	providerErr, ok := AsProviderError(err)
	if !ok {
		t.Fatalf("error = %T %v, want *ProviderError", err, err)
	}
	if providerErr.Code != CodeStreamFailed || providerErr.HTTPStatus != http.StatusOK {
		t.Fatalf("stream failure = %+v, want stream_failed after HTTP 200", providerErr)
	}
}

func TestGenerateClassifiesTransportFailure(t *testing.T) {
	runtime := resourceenv.Runtime{APIBaseURL: "https://openrouter.example/v1"}
	client := &statusHTTPClient{err: io.ErrClosedPipe}
	_, err := Generate(context.Background(), client, runtime, auth.Credentials{APIKey: "test-key"}, "text-model", "hi", nil, 0, nil, nil)
	providerErr, ok := AsProviderError(err)
	if !ok {
		t.Fatalf("error = %T %v, want *ProviderError", err, err)
	}
	if providerErr.Code != CodeUnreachable {
		t.Fatalf("code = %q, want %q", providerErr.Code, CodeUnreachable)
	}
}
