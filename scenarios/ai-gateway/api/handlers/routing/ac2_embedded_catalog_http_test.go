package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"ai-gateway/internal/catalog"
	internalrouting "ai-gateway/internal/routing"
)

// This exercises the production private-model HTTP handler with the catalog
// that is compiled into the Gateway binary. It proves the packaged proposal
// records reach the handler's policy source while proposed model identities
// remain unavailable to the credits picker. It does not claim LPBS J13 launch
// or hosted-bundle serving.
func TestAC2EmbeddedCatalogFlowsThroughPrivateModelsHTTPWithoutLeakingDrafts(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("load embedded provider catalog: %v", err)
	}
	source := internalrouting.ResourcePolicySource{Catalog: &bundle}
	snapshot, err := source.Snapshot(context.Background(), "ac2-private-model-http")
	if err != nil {
		t.Fatalf("load embedded catalog into production policy source: %v", err)
	}

	candidateIDs := []string{
		"allgate", "apinex", "apmix", "featherless", "groqcloud", "indierouter",
		"inferray", "kilo_gateway", "kovarouter", "llmapi", "llmsrelay", "nexusapi",
		"open_gonka", "plugsky", "relaymodels", "router-one", "uuapi", "nanogpt", "openrouter",
	}
	candidateSet := make(map[string]bool, len(candidateIDs))
	for _, id := range candidateIDs {
		candidateSet[id] = true
	}
	seen := make(map[string]bool, len(candidateIDs))
	for _, record := range snapshot.Records {
		if candidateSet[record.Supplier] {
			seen[record.Supplier] = true
			if record.Terms.Review.Status != "proposed" {
				t.Errorf("embedded candidate %q review status = %q, want proposed", record.Supplier, record.Terms.Review.Status)
			}
		}
	}
	for _, id := range candidateIDs {
		if !seen[id] {
			t.Errorf("embedded policy source omitted candidate record %q", id)
		}
	}
	for _, route := range snapshot.Routes {
		if candidateSet[route.Supplier] {
			t.Errorf("proposal-only supplier %q exposed route %q in policy snapshot", route.Supplier, route.ID)
		}
	}
	for _, agreement := range snapshot.Agreements {
		if candidateSet[agreement.Supplier] {
			t.Errorf("proposal-only supplier %q exposed an agreement in policy snapshot", agreement.Supplier)
		}
	}

	draftModels := map[string]bool{}
	draftModelIdentityCount := 0
	for _, id := range candidateIDs {
		provider, ok := bundle.Providers[id]
		if !ok {
			continue
		}
		var policy struct {
			Models map[string]json.RawMessage `json:"models"`
		}
		if err := json.Unmarshal(provider.Policy, &policy); err != nil {
			t.Fatalf("decode %s embedded model drafts: %v", id, err)
		}
		for model := range policy.Models {
			draftModels[model] = true
			draftModelIdentityCount++
		}
	}
	if draftModelIdentityCount != 78 {
		t.Fatalf("embedded candidate model-draft identities = %d, want all 78 sourced model records across 19 candidate providers", draftModelIdentityCount)
	}

	// Use the same constructor as main.go. The model endpoint does not touch
	// plan/receipt storage, so this focused handler proof needs no database.
	service := internalrouting.NewResourcePrivatePolicyService(
		nil,
		[]internalrouting.PrivateFeature{{ID: "agent.session", Role: "chat", Profile: "balanced"}},
		nil,
	)
	handler := NewConnectHandler(Deps{PrivatePolicy: service})
	router := mux.NewRouter()
	mountPrivatePolicyRoutes(router, handler)
	server := httptest.NewServer(router)
	defer server.Close()

	response, err := http.Get(server.URL + "/api/v1/private/provider-routes/models")
	if err != nil {
		t.Fatalf("GET packaged private-model endpoint: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET packaged private-model endpoint status = %d", response.StatusCode)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("GET packaged private-model cache policy = %q, want no-store", got)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode private-model response: %v", err)
	}
	for _, item := range result.Data {
		if draftModels[item.ID] {
			t.Errorf("proposed model draft %q leaked through credits model endpoint", item.ID)
		}
	}
	if len(result.Data) == 0 && !strings.EqualFold(result.Reason, "no_eligible_route") {
		t.Fatalf("empty packaged model result reason = %q, want no_eligible_route", result.Reason)
	}
}

func TestAC2OfficialSupplierCatalogIdentitiesRemainProposalOnly(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("load embedded provider catalog: %v", err)
	}
	want := map[string]struct {
		endpoint    string
		source      string
		modelSource string
		models      []string
	}{
		"allgate":      {endpoint: "https://api.allgate.ai/v1", source: "https://api.allgate.ai/docs", modelSource: "https://api.allgate.ai/models", models: []string{"gemini-4-argon", "claude-opus-5-5", "gpt-6-sol", "MiniMax-M3", "deepseek-flash"}},
		"featherless":  {endpoint: "https://api.featherless.ai/v1", source: "https://featherless.ai/docs/api-overview-and-common-options", modelSource: "https://featherless.ai/docs/models-model-compatibility", models: []string{"mistral-3.5", "qwen3-8b", "minimax-m3"}},
		"indierouter":  {endpoint: "https://api.indierouter.ai/v1", source: "https://indierouter.ai/docs/", modelSource: "https://indierouter.ai/models/", models: []string{"deepseek-v4-flash", "deepseek-v4.1-flash", "deepseek-v4.1-flash-1m", "gemma-4-26b-a4b-it", "gemma-4-31b-it", "glm-5.3", "glm-5.3-flash", "glm-5.3-flash-1m", "nemotron-3.5-super-vl-120b-a12b", "qwen3.8-27b-fp8", "pplx-decider-v1.1"}},
		"kilo_gateway": {endpoint: "https://ai-gateway.kilo.ai/api/v1", source: "https://kilo.ai/docs/gateway/api-reference", modelSource: "https://kilo.ai/docs/gateway/models-and-providers", models: []string{"anthropic/claude-sonnet-4.6", "openai/gpt-5.4"}},
	}
	for id, expected := range want {
		provider, ok := bundle.Providers[id]
		if !ok {
			t.Fatalf("embedded provider catalog omitted %q", id)
		}
		if provider.Endpoint != expected.endpoint {
			t.Errorf("%s endpoint = %q, want documented base %q", id, provider.Endpoint, expected.endpoint)
		}
		var policy struct {
			Provider struct {
				Terms struct {
					Review struct {
						Status string `json:"status"`
					} `json:"review"`
				} `json:"terms"`
				Routes     []json.RawMessage `json:"hosted_routes"`
				Agreements []json.RawMessage `json:"agreements"`
			} `json:"provider"`
			Models map[string]struct {
				Eligible *bool `json:"default_eligible"`
			} `json:"models"`
			Research struct {
				Activation string   `json:"activation"`
				Evidence   []string `json:"evidence"`
			} `json:"research"`
		}
		if err := json.Unmarshal(provider.Policy, &policy); err != nil {
			t.Fatalf("decode %s policy: %v", id, err)
		}
		if policy.Provider.Terms.Review.Status != "proposed" || len(policy.Provider.Routes) != 0 || len(policy.Provider.Agreements) != 0 || len(policy.Models) != len(expected.models) {
			t.Errorf("documented identities activated %s: review=%q routes=%d agreements=%d models=%d", id, policy.Provider.Terms.Review.Status, len(policy.Provider.Routes), len(policy.Provider.Agreements), len(policy.Models))
		}
		for _, modelID := range expected.models {
			model, ok := policy.Models[modelID]
			if !ok || model.Eligible == nil || *model.Eligible {
				t.Errorf("%s model %q must be packaged as explicitly disabled", id, modelID)
			}
		}
		if !strings.HasPrefix(policy.Research.Activation, "denied_until_") {
			t.Errorf("%s activation gate = %q, want denied_until_*", id, policy.Research.Activation)
		}
		foundSource := false
		for _, source := range policy.Research.Evidence {
			if source == expected.source {
				foundSource = true
				break
			}
		}
		if !foundSource {
			t.Errorf("%s embedded research omitted official endpoint source %q", id, expected.source)
		}
		if expected.modelSource != "" {
			foundModelSource := false
			for _, source := range policy.Research.Evidence {
				if source == expected.modelSource {
					foundModelSource = true
					break
				}
			}
			if !foundModelSource {
				t.Errorf("%s embedded research omitted official model catalog source %q", id, expected.modelSource)
			}
		}
	}
}

func TestAC2IndieRouterPublicModelCardsStayDisabledAndSeparateFromVrooliRates(t *testing.T) {
	bundle, err := catalog.Embedded()
	if err != nil {
		t.Fatalf("load embedded provider catalog: %v", err)
	}
	provider, ok := bundle.Providers["indierouter"]
	if !ok {
		t.Fatal("embedded provider catalog omitted IndieRouter")
	}
	var policy struct {
		Provider struct {
			Terms struct {
				Review struct {
					Status string `json:"status"`
				} `json:"review"`
			} `json:"terms"`
			Routes     []json.RawMessage `json:"hosted_routes"`
			Agreements []json.RawMessage `json:"agreements"`
		} `json:"provider"`
		Models map[string]struct {
			DisplayName string           `json:"display_name"`
			Publisher   string           `json:"provider"`
			Endpoints   []string         `json:"endpoints"`
			Context     int              `json:"context_window_tokens"`
			Eligible    *bool            `json:"default_eligible"`
			Pricing     *json.RawMessage `json:"pricing"`
			Licence     *json.RawMessage `json:"licence"`
			Provenance  struct {
				Catalog struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"catalog"`
				Terms struct {
					Source     string `json:"source"`
					ObservedAt string `json:"observed_at"`
				} `json:"terms"`
			} `json:"provenance"`
			Caveats []string `json:"caveats"`
		} `json:"models"`
		Research struct {
			Activation string `json:"activation"`
		} `json:"research"`
	}
	if err := json.Unmarshal(provider.Policy, &policy); err != nil {
		t.Fatalf("decode embedded IndieRouter proposal catalog: %v", err)
	}
	if policy.Provider.Terms.Review.Status != "proposed" || len(policy.Provider.Routes) != 0 || len(policy.Provider.Agreements) != 0 || !strings.HasPrefix(policy.Research.Activation, "denied_until_") {
		t.Fatalf("IndieRouter catalog is not proposal-only: provider=%+v activation=%q", policy.Provider, policy.Research.Activation)
	}
	want := map[string]struct {
		name     string
		endpoint string
		context  int
	}{
		"deepseek-v4-flash":               {"DeepSeek V4 Flash", "chat", 256000},
		"deepseek-v4.1-flash":             {"DeepSeek V4.1 Flash", "chat", 256000},
		"deepseek-v4.1-flash-1m":          {"DeepSeek V4.1 Flash 1M", "chat", 1000000},
		"gemma-4-26b-a4b-it":              {"Gemma 4 26B A4B", "chat", 256000},
		"gemma-4-31b-it":                  {"Gemma 4 31B", "chat", 256000},
		"glm-5.3":                         {"GLM-5.3", "chat", 230000},
		"glm-5.3-flash":                   {"GLM-5.3 Flash", "chat", 256000},
		"glm-5.3-flash-1m":                {"GLM-5.3 Flash 1M", "chat", 1000000},
		"nemotron-3.5-super-vl-120b-a12b": {"Nemotron 3.5 Super VL 120B A12B", "chat", 1000000},
		"qwen3.8-27b-fp8":                 {"Qwen 3.8 27B", "chat", 256000},
		"pplx-decider-v1.1":               {"Perplexity Decider v1.1 (27B)", "systemone", 8000},
	}
	if len(policy.Models) != len(want) {
		t.Fatalf("IndieRouter packaged model count = %d, want exact public catalog count %d", len(policy.Models), len(want))
	}
	for id, expected := range want {
		model, ok := policy.Models[id]
		if !ok {
			t.Errorf("IndieRouter catalog omitted public model %q", id)
			continue
		}
		if model.DisplayName != expected.name || len(model.Endpoints) != 1 || model.Endpoints[0] != expected.endpoint || model.Context != expected.context {
			t.Errorf("IndieRouter model %q metadata = name %q endpoints %v context %d; want %q/%s/%d", id, model.DisplayName, model.Endpoints, model.Context, expected.name, expected.endpoint, expected.context)
		}
		if model.Publisher != "" {
			t.Errorf("IndieRouter model %q has an unverified upstream publisher label %q", id, model.Publisher)
		}
		if model.Eligible == nil || *model.Eligible {
			t.Errorf("IndieRouter model %q must remain explicitly disabled", id)
		}
		if model.Pricing != nil || model.Licence != nil {
			t.Errorf("IndieRouter model %q promotes public rates or an unreviewed license into a Vrooli price/rights decision", id)
		}
		if model.Provenance.Catalog.Source != "https://indierouter.ai/models/; official catalog lists this exact model ID and display association, category, and context window" || model.Provenance.Catalog.ObservedAt != "2026-11-23" {
			t.Errorf("IndieRouter model %q catalog provenance = %+v", id, model.Provenance.Catalog)
		}
		if !strings.Contains(model.Provenance.Terms.Source, "reseller agreement") || model.Provenance.Terms.ObservedAt != "2026-11-23" {
			t.Errorf("IndieRouter model %q lost the separate reseller-agreement caveat: %+v", id, model.Provenance.Terms)
		}
		joinedCaveats := strings.Join(model.Caveats, " ")
		if !strings.Contains(joinedCaveats, "model-specific license") || !strings.Contains(joinedCaveats, "account-scoped availability") {
			t.Errorf("IndieRouter model %q does not preserve model-rights/account-scope limitations: %v", id, model.Caveats)
		}
	}
}
