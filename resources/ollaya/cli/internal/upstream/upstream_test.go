package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
)

var request = Request{
	State:     json.RawMessage(`"Stop. That is not what I asked for."`),
	Questions: json.RawMessage(`{"intent":{"type":"choice","criteria":{"correction":"","praise":"","other":""}},"stop":{"type":"noul"}}`),
}

const answers = `{"intent":{"type":"choice","choice":"correction","confidence":0.8,"probabilities":{"correction":0.87,"praise":0.03,"other":0.1}},"stop":{"type":"noul","noul":0.93}}`

type captured struct {
	path, auth string
	body       map[string]json.RawMessage
}

func fake(t *testing.T, status int, response string, seen *captured) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen.path, seen.auth = r.URL.Path, r.Header.Get("Authorization")
		_ = json.Unmarshal(raw, &seen.body)
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestResidentsReadTheDaemonPlacementAndUnloadReleasesMemory(t *testing.T) {
	var unloadBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"winnow:e4b","model":"winnow:e4b","device":"CUDA:0","size_vram":9000000000},{"name":"laya:en","device":"cpu","size_vram":0}]}`))
		case "/api/decide":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &unloadBody)
			_, _ = w.Write([]byte(`{"model":"winnow:e4b","done_reason":"unload"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := Client{HTTP: http.DefaultClient, GetEnv: func(key string) string {
		if key == "OLLAYA_URL" {
			return server.URL
		}
		return ""
	}}
	residents, err := client.Residents(context.Background())
	if err != nil || len(residents) != 2 || residents[0].Name != "winnow:e4b" || residents[0].Device != "cuda:0" || residents[0].SizeVRAM != 9000000000 || residents[1].Device != "cpu" {
		t.Fatalf("Residents = %+v err=%v, want both models with lowercase devices and VRAM bytes", residents, err)
	}
	if err := client.Unload(context.Background(), "winnow:e4b"); err != nil {
		t.Fatal(err)
	}
	if unloadBody["model"] != "winnow:e4b" || unloadBody["keep_alive"] != float64(0) || unloadBody["state"] != nil {
		t.Fatalf("unload body = %v, want a stateless keep_alive 0 request", unloadBody)
	}
}

func TestLocalDecideUsesNativeEndpointAndReportsDaemonTimings(t *testing.T) {
	var seen captured
	server := fake(t, 200, `{"model":"laya:en","answers":`+answers+`,"usage":{"input_tokens":41,"output_tokens":0},"state_truncated":false,"eval_duration":16000000,"load_duration":0}`, &seen)
	client := Client{GetEnv: func(key string) string { return map[string]string{"OLLAYA_URL": server.URL}[key] }}
	digest := strings.Repeat("b", 64)
	result, err := client.Decide(context.Background(), policy.Resolution{Upstream: "local", Locality: "local", Protocol: policy.ProtocolOllaya, Model: "laya", ExpectedArtifactDigest: digest}, request)
	if err != nil {
		t.Fatal(err)
	}
	if seen.path != "/api/decide" || seen.auth != "" {
		t.Fatalf("path=%s auth=%q, want /api/decide without credentials", seen.path, seen.auth)
	}
	if string(seen.body["questions"]) != string(request.Questions) || string(seen.body["model"]) != `"laya:latest@sha256:`+digest+`"` {
		t.Fatalf("questions must reach the daemon byte-for-byte in caller order; got %s", seen.body["questions"])
	}
	if string(seen.body["state"]) != string(request.State) {
		t.Fatalf("state must reach the daemon without being replaced or omitted; got %s", seen.body["state"])
	}
	if keepAlive, present := seen.body["keep_alive"]; present {
		t.Fatalf("keep_alive = %s; residency belongs to the activity edge, not the client", keepAlive)
	}
	if result.Model != "laya:en" || *result.Usage.InputTokens != 41 || *result.EvalMS != 16 || result.CostUSD != nil {
		t.Fatalf("result = %+v, want the answering model, reported tokens and daemon eval time", result)
	}
}

func TestLocalDecideRequiresDigestQualifiedModelBeforeNetwork(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client := Client{GetEnv: func(key string) string {
		if key == "OLLAYA_URL" {
			return server.URL
		}
		return ""
	}}
	_, err := client.Decide(context.Background(), policy.Resolution{Upstream: "local", Locality: "local", Protocol: policy.ProtocolOllaya, Model: "laya"}, request)
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != CodeUnconfigured || called {
		t.Fatalf("err=%v; want an unconfigured refusal before sending a mutable model name", err)
	}
}

func TestHostedDecideSendsBearerAndUnwrapsCloudflare(t *testing.T) {
	var seen captured
	server := fake(t, 200, `{"result":{"model":"clef-flash","answers":`+answers+`,"usage":{"input_tokens":50}},"success":true,"errors":[]}`, &seen)
	result, err := Client{}.Decide(context.Background(), policy.Resolution{
		Upstream: "cloudflare", Locality: "remote", Protocol: policy.ProtocolSystemOne, Model: "clef-flash",
		Envelope: policy.EnvelopeCloudflare, Endpoint: server.URL + "/ai/run", Token: "secret",
	}, request)
	if err != nil {
		t.Fatal(err)
	}
	if seen.auth != "Bearer secret" || result.Model != "clef-flash" || result.EvalMS != nil {
		t.Fatalf("auth=%q result=%+v", seen.auth, result)
	}
	var decoded map[string]any
	if err := json.Unmarshal(result.Answers, &decoded); err != nil || len(decoded) != 2 {
		t.Fatalf("answers = %s, want both answers unwrapped", result.Answers)
	}
}

func TestEdenCostIsCarriedAsReported(t *testing.T) {
	var seen captured
	server := fake(t, 200, `{"model":"jev-1.13.0","answers":`+answers+`,"usage":{"input_tokens":430,"output_tokens":73},"cost":0.00001806}`, &seen)
	result, err := Client{}.Decide(context.Background(), policy.Resolution{Protocol: policy.ProtocolSystemOne, Endpoint: server.URL, Token: "k", Model: "typesafe/jev-latest"}, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.CostUSD == nil || *result.CostUSD != 0.00001806 || *result.Usage.OutputTokens != 73 {
		t.Fatalf("result = %+v, want the provider's own cost and usage", result)
	}
}

func TestFailuresAreClassifiedForGatewayFallback(t *testing.T) {
	cases := []struct {
		name, body string
		status     int
		code       string
	}{
		{"truncated native answer", `{"model":"laya:en","answers":{},"state_truncated":true}`, 200, CodeContextOverflow},
		{"hosted truncation refusal", `{"error":"state: dropped","code":"STATE_TRUNCATED"}`, 422, CodeContextOverflow},
		{"model not pulled", `{"error":"model not found","code":"MODEL_NOT_FOUND"}`, 404, CodeUnconfigured},
		{"bad key", `{"detail":{"message":"Must supply an API key!"}}`, 403, CodeUnconfigured},
		{"rate limited", `{"error":"slow down"}`, 429, CodeRateLimited},
		{"queue full", `{"error":"queue full","code":"QUEUE_FULL"}`, 503, CodeProviderOverloaded},
		{"model refuses options", `{"error":"140 options","code":"TOO_MANY_OPTIONS"}`, 422, CodeLimitsExceeded},
		{"load ran out of vram", `{"error":"insufficient_vram: needs 9 GiB","code":"MODEL_LOAD_FAILED"}`, 500, CodeInsufficientVRAM},
		{"onnx arena could not allocate", `{"error":"pplx-decider:flash failed to load: ERROR ort::logging: Exception during initialization: bfc_arena.cc:359 Failed to allocate memory for requested buffer of size 47185920","code":"MODEL_LOAD_FAILED"}`, 500, CodeInsufficientVRAM},
		{"llama.cpp cudaMalloc failed", `{"error":"credence failed to load: ERROR llama.cpp: ggml_backend_cuda_buffer_type_alloc_buffer: allocating 559.02 MiB on device 0: cudaMalloc failed: out of memory","code":"MODEL_LOAD_FAILED"}`, 500, CodeInsufficientVRAM},
		{"load failed for another reason", `{"error":"corrupt weights","code":"MODEL_LOAD_FAILED"}`, 500, CodeProviderFailed},
		{"empty answers", `{"model":"x","answers":null}`, 200, CodeProviderFailed},
		{"cloudflare failure", `{"success":false,"errors":[{"message":"quota"}]}`, 200, CodeProviderFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen captured
			server := fake(t, tc.status, tc.body, &seen)
			resolution := policy.Resolution{Protocol: policy.ProtocolSystemOne, Endpoint: server.URL, Token: "k", Model: "m"}
			if tc.name == "cloudflare failure" {
				resolution.Envelope = policy.EnvelopeCloudflare
			}
			_, err := Client{}.Decide(context.Background(), resolution, request)
			var failure *ProviderError
			if !errors.As(err, &failure) || failure.Code != tc.code {
				t.Fatalf("err = %v, want code %s", err, tc.code)
			}
			if tc.status == 429 && failure.RetryAfter != "7" {
				t.Fatalf("Retry-After = %q, want the observed header", failure.RetryAfter)
			}
		})
	}
}

func TestUnconfiguredAndUnreachableNeverCallOrHang(t *testing.T) {
	_, err := Client{}.Decide(context.Background(), policy.Resolution{Protocol: policy.ProtocolSystemOne, Missing: []string{"EDENAI_API_KEY"}}, request)
	var failure *ProviderError
	if !errors.As(err, &failure) || failure.Code != CodeUnconfigured {
		t.Fatalf("err = %v, want unconfigured before any request", err)
	}
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	_, err = Client{}.Decide(context.Background(), policy.Resolution{Protocol: policy.ProtocolSystemOne, Endpoint: url, Token: "k", Model: "m"}, request)
	if !errors.As(err, &failure) || failure.Code != CodeUnreachable {
		t.Fatalf("err = %v, want unreachable", err)
	}
	line := failure.MarkerLine()
	if line[:len(ProviderErrorMarker)] != ProviderErrorMarker {
		t.Fatalf("marker line %q must start with the runner's marker", line)
	}
}
