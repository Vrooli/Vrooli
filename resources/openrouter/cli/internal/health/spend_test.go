package health

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/openrouter/cli/internal/auth"
	resourceenv "github.com/vrooli/vrooli/resources/openrouter/cli/internal/env"
)

func TestGenerateWithSpendCapSendsAllOwnerCapsBeforeDispatch(t *testing.T) {
	var methods []string
	var chat map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodGet {
			if r.Header.Get("Authorization") != "Bearer owner-key" {
				t.Errorf("endpoint metadata request omitted owner credential")
			}
			_, _ = io.WriteString(w, `{"data":{"endpoints":[{"provider_name":"OpenAI","context_length":10000,"max_completion_tokens":5000,"pricing":{"prompt":"0.00001","completion":"0.00002","request":"0","image":"0.0001"}}]}}`)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&chat); err != nil {
			t.Errorf("decode chat body: %v", err)
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer server.Close()
	runtime := resourceenv.Runtime{APIBaseURL: server.URL + "/api/v1"}
	image := []byte("reference")
	imageHash := sha256.Sum256(image)
	binding := SpendBinding{MaxCostMicros: 10_000_000, OperationID: "12345678-1234-4234-9234-123456789abc", InputFingerprint: strings.Repeat("a", 64), ReferenceSHA256: []string{hex.EncodeToString(imageHash[:])}, FundingRoute: "byok", AccountIdentity: "vrooli/openrouter"}
	_, authorization, err := GenerateWithSpendCap(context.Background(), http.DefaultClient, runtime, auth.Credentials{APIKey: "owner-key"}, "openai/gpt-4o", "make a geometry recipe", nil, 500, nil, []ImageInput{{MediaType: "image/png", DataB64: "cmVmZXJlbmNl"}}, binding, []string{"OpenAI"})
	if err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != "GET /api/v1/models/openai/gpt-4o/endpoints" || methods[1] != "POST /api/v1/chat/completions" {
		t.Fatalf("unexpected provider request sequence: %v", methods)
	}
	var provider struct {
		AllowFallbacks    bool                   `json:"allow_fallbacks"`
		RequireParameters bool                   `json:"require_parameters"`
		Only              []string               `json:"only"`
		MaxPrice          map[string]json.Number `json:"max_price"`
		ZDR               bool                   `json:"zdr"`
		DataCollection    string                 `json:"data_collection"`
	}
	if err := json.Unmarshal(chat["provider"], &provider); err != nil {
		t.Fatal(err)
	}
	if provider.AllowFallbacks || !provider.RequireParameters || len(provider.Only) != 1 || provider.Only[0] != "OpenAI" || !provider.ZDR || provider.DataCollection != "deny" {
		t.Fatalf("provider routing controls were not pinned: %+v", provider)
	}
	for _, key := range []string{"prompt", "completion", "request", "image"} {
		if _, ok := provider.MaxPrice[key]; !ok {
			t.Fatalf("max_price omitted explicit %s cap: %v", key, provider.MaxPrice)
		}
	}
	assertWorstCaseWithinCap(t, provider.MaxPrice, authorization.InputTokensBound, authorization.OutputTokensBound, authorization.ImageCount, binding.MaxCostMicros)
	if authorization.Status != "authorized" || authorization.Model != "openai/gpt-4o" || authorization.FundingRoute != "byok" || authorization.AccountIdentity != "vrooli/openrouter" || authorization.InputFingerprint != binding.InputFingerprint || authorization.OperationID != binding.OperationID || authorization.ContextWindowTokens != 10000 {
		t.Fatalf("authorization does not bind the exact dispatch: %+v", authorization)
	}
}

func TestGenerateWithZeroCostRequiresCurrentAllZeroPricesAndPrivacyControls(t *testing.T) {
	var chat map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"data":{"endpoints":[{"provider_name":"Phala","context_length":10000,"max_completion_tokens":5000,"pricing":{"prompt":"0","completion":"0","discount":0}}]}}`)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&chat); err != nil {
			t.Errorf("decode chat body: %v", err)
		}
		_, _ = io.WriteString(w, `{"model":"xingchen-agi/xing4.0-29b-a4b:free","provider_name":"Phala","usage":{"cost":0},"choices":[{"message":{"content":"response"}}]}`)
	}))
	defer server.Close()
	binding := SpendBinding{MaxCostMicros: 0, OperationID: "12345678-1234-4234-9234-123456789abc", InputFingerprint: strings.Repeat("d", 64), FundingRoute: "byok", AccountIdentity: "vrooli/openrouter"}
	_, authorization, err := GenerateWithSpendCap(context.Background(), http.DefaultClient, resourceenv.Runtime{APIBaseURL: server.URL + "/api/v1"}, auth.Credentials{APIKey: "owner-key"}, "xingchen-agi/xing4.0-29b-a4b:free", "fictional prompt", nil, 180, nil, nil, binding, []string{"Phala"})
	if err != nil {
		t.Fatal(err)
	}
	var provider struct {
		AllowFallbacks bool                   `json:"allow_fallbacks"`
		Only           []string               `json:"only"`
		ZDR            bool                   `json:"zdr"`
		DataCollection string                 `json:"data_collection"`
		MaxPrice       map[string]json.Number `json:"max_price"`
	}
	if err := json.Unmarshal(chat["provider"], &provider); err != nil {
		t.Fatal(err)
	}
	if provider.AllowFallbacks || len(provider.Only) != 1 || provider.Only[0] != "Phala" || !provider.ZDR || provider.DataCollection != "deny" {
		t.Fatalf("zero-cost privacy/routing controls were not pinned: %+v", provider)
	}
	for _, key := range []string{"prompt", "completion", "request", "image"} {
		if provider.MaxPrice[key] != "0" {
			t.Fatalf("max_price.%s must be zero, got %q", key, provider.MaxPrice[key])
		}
	}
	if authorization.MaxCostMicros != 0 || authorization.Model != "xingchen-agi/xing4.0-29b-a4b:free" {
		t.Fatalf("authorization did not preserve zero cap and exact model: %+v", authorization)
	}
}

func TestZeroCostRejectsResponseOutsideAuthorizedModelProviderAndCost(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "wrong model", response: `{"model":"other/model","provider_name":"Phala","usage":{"cost":0},"choices":[]}`},
		{name: "wrong provider", response: `{"model":"xingchen-agi/xing4.0-29b-a4b:free","provider_name":"Other","usage":{"cost":0},"choices":[]}`},
		{name: "nonzero cost", response: `{"model":"xingchen-agi/xing4.0-29b-a4b:free","provider_name":"Phala","usage":{"cost":"0.000001"},"choices":[]}`},
		{name: "missing cost", response: `{"model":"xingchen-agi/xing4.0-29b-a4b:free","provider_name":"Phala","usage":{},"choices":[]}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					_, _ = io.WriteString(w, `{"data":{"endpoints":[{"provider_name":"Phala","context_length":10000,"max_completion_tokens":5000,"pricing":{"prompt":"0","completion":"0","request":"0","image":"0"}}]}}`)
					return
				}
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			binding := SpendBinding{MaxCostMicros: 0, OperationID: "12345678-1234-4234-9234-123456789abc", InputFingerprint: strings.Repeat("f", 64), FundingRoute: "byok", AccountIdentity: "vrooli/openrouter"}
			if _, _, err := GenerateWithSpendCap(context.Background(), http.DefaultClient, resourceenv.Runtime{APIBaseURL: server.URL + "/api/v1"}, auth.Credentials{APIKey: "key"}, "xingchen-agi/xing4.0-29b-a4b:free", "prompt", nil, 180, nil, nil, binding, []string{"Phala"}); err == nil {
				t.Fatalf("unverified zero-cost response was accepted: %s", tc.response)
			}
		})
	}
}

func TestZeroCostRefusesAnyNonzeroCurrentPriceBeforeChat(t *testing.T) {
	chatCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			chatCalls++
			return
		}
		_, _ = io.WriteString(w, `{"data":{"endpoints":[{"provider_name":"Phala","context_length":10000,"pricing":{"prompt":"0","completion":"0.0000001","request":"0","image":"0"}}]}}`)
	}))
	defer server.Close()
	binding := SpendBinding{MaxCostMicros: 0, OperationID: "12345678-1234-4234-9234-123456789abc", InputFingerprint: strings.Repeat("e", 64), FundingRoute: "byok", AccountIdentity: "vrooli/openrouter"}
	_, _, err := GenerateWithSpendCap(context.Background(), http.DefaultClient, resourceenv.Runtime{APIBaseURL: server.URL + "/api/v1"}, auth.Credentials{APIKey: "key"}, "xingchen-agi/xing4.0-29b-a4b:free", "prompt", nil, 180, nil, nil, binding, []string{"Phala"})
	if err == nil || chatCalls != 0 {
		t.Fatalf("nonzero current price with zero ceiling dispatched: err=%v chat calls=%d", err, chatCalls)
	}
}

func TestGenerateWithSpendCapRefusesMissingCurrentPriceClassBeforeChat(t *testing.T) {
	chatCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			chatCalls++
			return
		}
		_, _ = io.WriteString(w, `{"data":{"endpoints":[{"provider_name":"OpenAI","context_length":10000,"pricing":{"prompt":"0.00001","completion":"0.00002","request":"0"}}]}}`)
	}))
	defer server.Close()
	_, _, err := GenerateWithSpendCap(context.Background(), http.DefaultClient, resourceenv.Runtime{APIBaseURL: server.URL + "/api/v1"}, auth.Credentials{APIKey: "key"}, "openai/gpt-4o", "prompt", nil, 500, nil, []ImageInput{{MediaType: "image/png", DataB64: "YQ=="}}, SpendBinding{MaxCostMicros: 1000000, OperationID: "12345678-1234-4234-9234-123456789abc", InputFingerprint: strings.Repeat("b", 64), ReferenceSHA256: []string{strings.Repeat("c", 64)}, FundingRoute: "byok", AccountIdentity: "vrooli/openrouter"}, []string{"OpenAI"})
	if err == nil || chatCalls != 0 {
		t.Fatalf("missing current price class dispatched: err=%v chat calls=%d", err, chatCalls)
	}
}

func assertWorstCaseWithinCap(t *testing.T, prices map[string]json.Number, input, output, images int, maxMicros int64) {
	t.Helper()
	contribution := func(key string, units int64, divisor int64) *big.Rat {
		value, ok := new(big.Rat).SetString(prices[key].String())
		if !ok {
			t.Fatalf("invalid %s cap %q", key, prices[key])
		}
		return new(big.Rat).Quo(new(big.Rat).Mul(value, big.NewRat(units, 1)), big.NewRat(divisor, 1))
	}
	totalUSD := new(big.Rat)
	totalUSD.Add(totalUSD, contribution("prompt", int64(input), 1_000_000))
	totalUSD.Add(totalUSD, contribution("completion", int64(output), 1_000_000))
	totalUSD.Add(totalUSD, contribution("request", 1, 1))
	totalUSD.Add(totalUSD, contribution("image", int64(images), 1))
	capUSD := big.NewRat(maxMicros, 1_000_000)
	if totalUSD.Cmp(capUSD) > 0 {
		t.Fatalf("provider class caps total %s USD above user maximum %s", totalUSD.RatString(), capUSD.RatString())
	}
}
