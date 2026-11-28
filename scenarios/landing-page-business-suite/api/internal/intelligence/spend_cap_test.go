package intelligence

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type capMetadataClient struct {
	*MockOpenRouterClient
	endpoints   []OpenRouterEndpointMetadata
	metadataErr error
}

func (c *capMetadataClient) CurrentEndpoints(context.Context, string) ([]OpenRouterEndpointMetadata, error) {
	return c.endpoints, c.metadataErr
}

type capUsageFake struct {
	*MockUsageService
	reserveErr   error
	reserved     int64
	multiplier   int64
	reserveCalls int
	retainCalls  int
	reserveArgs  []any
	retainArgs   []any
}

func (u *capUsageFake) ReserveBusinessAccountUSDSpendCap(_ context.Context, account, actor, tier, app string, capMicros int64, operation string) (string, int64, int64, error) {
	u.reserveCalls++
	u.reserveArgs = []any{account, actor, tier, app, capMicros, operation}
	if u.reserveErr != nil {
		return "", 0, 0, u.reserveErr
	}
	return "reservation", u.reserved, u.multiplier, nil
}
func (u *capUsageFake) RetainUncertainBusinessAccountReservation(_ context.Context, account, actor, reservation string) error {
	u.retainCalls++
	u.retainArgs = []any{account, actor, reservation}
	return nil
}

func spendRequest(cost int64, maxTokens int) AIRequest {
	data := []byte("reference-image")
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	return AIRequest{BusinessAccountID: "acct-spend-cap", Model: "openai/gpt-4o-mini", Messages: []AIMessage{{Role: "user", Content: "create"}}, Images: []AIImage{{Data: base64.StdEncoding.EncodeToString(data), MediaType: "image/png", SHA256: digest}}, MaxTokens: maxTokens, Metadata: AIMetadata{AppBundleKey: "image-tools"}, SpendLimit: &SpendLimit{MaxCostMicros: cost, OperationID: "123e4567-e89b-42d3-a456-426614174000", InputFingerprint: strings.Repeat("a", 64), ReferenceSHA256: []string{digest}, ImageCount: 1, FundingRoute: "vrooli_credits"}, spendLimitTrusted: true, creditOwnerVerified: true}
}

func TestSpendCappedInferenceReservesBeforeDispatchAndForwardsHardCaps(t *testing.T) {
	usage := &capUsageFake{MockUsageService: NewMockUsageService(), reserved: 100, multiplier: 10000}
	accountCalls := []string{}
	accounts := &MockAccountService{GetBusinessSubscriptionFn: func(actor, account string) (*landing_page_business_suite_v1.SubscriptionStatus, error) {
		accountCalls = append(accountCalls, actor+"/"+account)
		tier := "pro"
		return &landing_page_business_suite_v1.SubscriptionStatus{PlanTier: &tier}, nil
	}}
	client := &capMetadataClient{MockOpenRouterClient: &MockOpenRouterClient{}, endpoints: []OpenRouterEndpointMetadata{{Provider: "owner", ContextTokens: 2048, PromptPrice: .000000001, CompletionPrice: .000000002, RequestPrice: .000000003, ImagePrice: .000000004, RequestPriceKnown: true, ImagePriceKnown: true}}}
	client.ChatFn = func(_ context.Context, req OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
		if usage.retainCalls != 1 {
			t.Fatal("provider dispatched before durable hold was retained")
		}
		if req.MaxTokens != 32 || req.Provider == nil || req.Provider.AllowFallbacks || !req.Provider.RequireParameters || len(req.Provider.Only) != 1 {
			t.Fatalf("request was not owner-bounded: %+v", req)
		}
		if req.Provider.MaxPrice.Prompt <= 0 || req.Provider.MaxPrice.Completion <= 0 || req.Provider.MaxPrice.Request <= 0 || req.Provider.MaxPrice.Image <= 0 {
			t.Fatalf("all price caps must be explicit: %+v", req.Provider.MaxPrice)
		}
		parts, ok := req.Messages[0].Content.([]map[string]any)
		if !ok || len(parts) != 2 {
			t.Fatalf("reference image not forwarded to provider: %#v", req.Messages[0].Content)
		}
		return &OpenRouterChatResponse{ID: "provider-call", Model: req.Model, Content: "{}", Usage: OpenRouterUsage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, CostMicros: 5, CostReported: true}}, nil
	}
	service := &MeteredInferenceService{usageService: usage, accountService: accounts, openRouterClient: client}
	response, err := service.ExecuteChat(context.Background(), "member@example.test", spendRequest(1000, 32))
	if err != nil {
		t.Fatal(err)
	}
	if response.AccountingStatus != "settled" || response.CreditsCharged != 5 || response.SpendAuthorization == nil {
		t.Fatalf("unexpected capped response: %+v", response)
	}
	if usage.reserveCalls != 1 || len(usage.Calls["FinalizeBusinessAccountReservationWithCost"]) != 1 {
		t.Fatalf("reserve/settle calls = %d/%d", usage.reserveCalls, len(usage.Calls["FinalizeBusinessAccountReservationWithCost"]))
	}
	want := []any{"acct-spend-cap", "member@example.test", "pro", "image-tools", int64(1000), "123e4567-e89b-42d3-a456-426614174000"}
	if !reflect.DeepEqual(usage.reserveArgs, want) {
		t.Fatalf("account reservation args = %#v, want %#v", usage.reserveArgs, want)
	}
	if !reflect.DeepEqual(accountCalls, []string{"member@example.test/acct-spend-cap"}) {
		t.Fatalf("subscription lookup identity/account = %#v", accountCalls)
	}
	if !reflect.DeepEqual(usage.retainArgs, []any{"acct-spend-cap", "member@example.test", "reservation"}) {
		t.Fatalf("uncertain hold binding = %#v", usage.retainArgs)
	}
	settlement := usage.Calls["FinalizeBusinessAccountReservationWithCost"][0].([]interface{})
	if settlement[0] != "acct-spend-cap" || settlement[1] != "member@example.test" || settlement[2] != "reservation" || settlement[3] != int64(5) || settlement[4] != "image-tools" || settlement[5] != "openai/gpt-4o-mini" || settlement[9] != "openrouter" {
		t.Fatalf("account-scoped, source-attributed settlement = %#v", settlement)
	}
	for _, legacy := range []string{"ReserveCredits", "RetainUncertainReservation", "FinalizeReservation"} {
		if len(usage.Calls[legacy]) != 0 {
			t.Fatalf("spend cap used legacy personal billing path %s: %#v", legacy, usage.Calls[legacy])
		}
	}
}

func TestSpendCappedInferenceDoesNotDispatchWhenCurrentMetadataOrReservationFails(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		metadataErr, reserveErr error
	}{
		{name: "metadata unavailable", metadataErr: errors.New("stale or unavailable")},
		{name: "credit hold unavailable", reserveErr: errors.New("insufficient credits")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usage := &capUsageFake{MockUsageService: NewMockUsageService(), reserveErr: tc.reserveErr, reserved: 100, multiplier: 10000}
			client := &capMetadataClient{MockOpenRouterClient: &MockOpenRouterClient{}, metadataErr: tc.metadataErr, endpoints: []OpenRouterEndpointMetadata{{Provider: "owner", ContextTokens: 2048, PromptPrice: .000000001, CompletionPrice: .000000002, RequestPrice: .000000003, ImagePrice: .000000004, RequestPriceKnown: true, ImagePriceKnown: true}}}
			calls := 0
			client.ChatFn = func(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
				calls++
				return nil, nil
			}
			service := &MeteredInferenceService{usageService: usage, accountService: &MockAccountService{}, openRouterClient: client}
			if _, err := service.ExecuteChat(context.Background(), "member@example.test", spendRequest(1000, 32)); err == nil {
				t.Fatal("expected authorization failure")
			}
			if calls != 0 {
				t.Fatal("provider was called without current authorization")
			}
		})
	}
}

func TestSpendCappedInferenceRetainsUnknownProviderCost(t *testing.T) {
	usage := &capUsageFake{MockUsageService: NewMockUsageService(), reserved: 100, multiplier: 10000}
	client := &capMetadataClient{MockOpenRouterClient: &MockOpenRouterClient{}, endpoints: []OpenRouterEndpointMetadata{{Provider: "owner", ContextTokens: 2048, PromptPrice: .000000001, CompletionPrice: .000000002, RequestPrice: .000000003, ImagePrice: .000000004, RequestPriceKnown: true, ImagePriceKnown: true}}}
	client.ChatFn = func(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
		return &OpenRouterChatResponse{ID: "call", Model: "openai/gpt-4o-mini", Usage: OpenRouterUsage{PromptTokens: 3, CompletionTokens: 2}}, nil
	}
	service := &MeteredInferenceService{usageService: usage, accountService: &MockAccountService{}, openRouterClient: client}
	response, err := service.ExecuteChat(context.Background(), "member@example.test", spendRequest(1000, 32))
	if err != nil {
		t.Fatal(err)
	}
	if response.AccountingStatus != "pending_reconciliation" || usage.retainCalls != 1 || len(usage.Calls["FinalizeReservation"]) != 0 {
		t.Fatalf("unknown cost was not held: response=%+v usage=%+v", response, usage)
	}
}

func TestSpendCappedInferenceRetainsAccountReservationAfterProviderFailure(t *testing.T) {
	usage := &capUsageFake{MockUsageService: NewMockUsageService(), reserved: 100, multiplier: 10000}
	client := &capMetadataClient{MockOpenRouterClient: &MockOpenRouterClient{}, endpoints: []OpenRouterEndpointMetadata{{Provider: "owner", ContextTokens: 2048, PromptPrice: .000000001, CompletionPrice: .000000002, RequestPrice: .000000003, ImagePrice: .000000004, RequestPriceKnown: true, ImagePriceKnown: true}}}
	client.ChatFn = func(context.Context, OpenRouterChatRequest) (*OpenRouterChatResponse, error) {
		return nil, errors.New("synthetic ambiguous transport failure")
	}
	service := &MeteredInferenceService{usageService: usage, accountService: &MockAccountService{}, openRouterClient: client}
	if _, err := service.ExecuteChat(context.Background(), "member@example.test", spendRequest(1000, 32)); err == nil || !strings.Contains(err.Error(), "provider outcome retained pending") {
		t.Fatalf("ambiguous provider outcome error = %v", err)
	}
	if usage.retainCalls != 2 || len(usage.Calls["FinalizeBusinessAccountReservationWithCost"]) != 0 || len(usage.Calls["ReleaseBusinessAccountReservation"]) != 0 {
		t.Fatalf("uncertain hold was not retained without settlement/release: retain=%d finalize=%#v release=%#v", usage.retainCalls, usage.Calls["FinalizeBusinessAccountReservationWithCost"], usage.Calls["ReleaseBusinessAccountReservation"])
	}
	if !reflect.DeepEqual(usage.retainArgs, []any{"acct-spend-cap", "member@example.test", "reservation"}) {
		t.Fatalf("uncertain account hold binding = %#v", usage.retainArgs)
	}
}

func TestSpendCapProviderPolicyIsBoundedAndDisablesFallback(t *testing.T) {
	policy, contextBound, err := capProviderPolicy([]OpenRouterEndpointMetadata{
		{Provider: "zeta", ContextTokens: 8192, PromptPrice: .0000001, CompletionPrice: .000001, RequestPrice: 0, ImagePrice: .0001, RequestPriceKnown: true, ImagePriceKnown: true},
		{Provider: "alpha", ContextTokens: 4096, PromptPrice: .0000001, CompletionPrice: .000001, RequestPrice: 0, ImagePrice: .0001, RequestPriceKnown: true, ImagePriceKnown: true},
	}, 1024, 2, 900_001)
	if err != nil {
		t.Fatal(err)
	}
	if contextBound != 4096 {
		t.Fatalf("context bound = %d, want minimum eligible context 4096", contextBound)
	}
	if policy.AllowFallbacks || !policy.RequireParameters || len(policy.Only) != 2 || policy.Only[0] != "alpha" || policy.Only[1] != "zeta" {
		t.Fatalf("unexpected provider policy: %+v", policy)
	}
	if policy.MaxPrice.Prompt <= 0 || policy.MaxPrice.Completion <= 0 || policy.MaxPrice.Request < 0 || policy.MaxPrice.Image <= 0 {
		t.Fatalf("all four explicit price caps are required: %+v", policy.MaxPrice)
	}
	if math.IsInf(policy.MaxPrice.Prompt, 0) || math.IsNaN(policy.MaxPrice.Prompt) {
		t.Fatal("prompt cap is not finite")
	}
	encoded, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	price := body["max_price"].(map[string]any)
	for _, key := range []string{"prompt", "completion", "request", "image"} {
		if _, ok := price[key]; !ok {
			t.Fatalf("max_price omitted explicit %s cap: %s", key, encoded)
		}
	}
}

func TestSpendCapMaxPriceUsesOpenRouterUnitsAndBoundsAggregateCeiling(t *testing.T) {
	policy, contextBound, err := capProviderPolicy([]OpenRouterEndpointMetadata{{
		Provider: "official-shape", ContextTokens: 4096, PromptPrice: 0.00000015, CompletionPrice: 0.0000006, RequestPrice: 0.002, ImagePrice: 0.004, RequestPriceKnown: true, ImagePriceKnown: true,
	}}, 128, 2, 100_000)
	if err != nil {
		t.Fatal(err)
	}
	inputBound := contextBound - 128
	if inputBound != 3968 {
		t.Fatalf("input bound = %d", inputBound)
	}
	body, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		MaxPrice OpenRouterMaxPrice `json:"max_price"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.MaxPrice.Prompt < 0.00000015*1_000_000 || decoded.MaxPrice.Prompt >= 25 || decoded.MaxPrice.Completion < 0.0000006*1_000_000 || decoded.MaxPrice.Completion >= 250 {
		t.Fatalf("token caps are not USD per million tokens: %+v", decoded.MaxPrice)
	}
	// Prompt/completion endpoint metadata is USD/token; max_price token
	// controls are USD/million tokens. Request/image remain USD per unit.
	worstUSD := decoded.MaxPrice.Prompt/1_000_000*float64(inputBound) +
		decoded.MaxPrice.Completion/1_000_000*128 + decoded.MaxPrice.Request + decoded.MaxPrice.Image*2
	if worstUSD > 0.1 {
		t.Fatalf("serialized caps allow %.12f USD above the 0.1 USD ceiling", worstUSD)
	}
}

func TestOpenRouterCurrentEndpointMetadataTracksAbsentOptionalCategories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models/openai/gpt-4o-mini/endpoints" {
			t.Fatalf("unexpected metadata path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"endpoints":[{"provider_name":"complete","context_length":8192,"pricing":{"prompt":"0.000001","completion":"0.000002","request":"0","image":"0.000003"}},{"provider_name":"no-separate-fees","context_length":8192,"pricing":{"prompt":"0.00000015","completion":"0.0000006","input_cache_read":"0.0000001","discount":"0"}},{"provider_name":"missing","context_length":8192,"pricing":{"prompt":"0.000001","completion":"0.000002","request":"0","image":null}},{"provider_name":"extra-fee","context_length":8192,"pricing":{"prompt":"0.000001","completion":"0.000002","request":"0","image":"0.000003","cache_write":"0.00001"}}]}}`))
	}))
	defer server.Close()
	client := NewOpenRouterClient(OpenRouterClientOptions{APIKey: "test", BaseURL: server.URL}).(*httpOpenRouterClient)
	got, err := client.CurrentEndpoints(context.Background(), "openai/gpt-4o-mini")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Provider != "complete" || got[0].ContextTokens != 8192 || got[0].ImagePrice != 0.000003 || !got[0].RequestPriceKnown || !got[0].ImagePriceKnown || got[1].Provider != "no-separate-fees" || got[1].RequestPriceKnown || got[1].ImagePriceKnown {
		t.Fatalf("unexpected fully priced endpoints: %+v", got)
	}
	policy, _, err := capProviderPolicy(got[1:], 128, 2, 100_000)
	if err != nil {
		t.Fatal(err)
	}
	if policy.MaxPrice.Request != 0 || policy.MaxPrice.Image != 0 || len(policy.Only) != 1 || policy.Only[0] != "no-separate-fees" {
		t.Fatalf("absent optional categories must be explicitly capped at zero: %+v", policy)
	}
}

func TestCreditCapConversionRoundsUpAndDetectsOverflow(t *testing.T) {
	got, err := ceilMulDiv(1, 1000001, 10000)
	if err != nil || got != 101 {
		t.Fatalf("ceil conversion = %d, %v; want 101", got, err)
	}
	if _, err := ceilMulDiv(math.MaxInt64, math.MaxInt64, 1); err == nil {
		t.Fatal("overflowing conversion was accepted")
	}
}
