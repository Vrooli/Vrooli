package bootstrap

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
	"audio-tools/internal/store"
)

type fixedProviderConfig struct{ config store.ProviderConfig }

func (f fixedProviderConfig) Get(context.Context) (store.ProviderConfig, error) { return f.config, nil }

func TestStartupAppliesSavedRoutingAndEnvironmentUpperBounds(t *testing.T) {
	coordinator := &chains.Coordinator{TTS: ttschain.NewChain(ttschain.Options{
		BYOK: ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{
			"openai-tts": &ttsmocks.FakeBYOK{IDStr: "openai-tts", Available: true},
		}),
		EnableBYOK: true,
	})}
	env := Env{EnableBYOK: true, EnableVrooli: true, EnableLocal: true}
	reader := fixedProviderConfig{config: store.ProviderConfig{BYOKEnabled: false, VrooliEnabled: true, LocalEnabled: true}}
	if err := applyStoredProviderRouting(context.Background(), reader, coordinator, env); err != nil {
		t.Fatalf("apply saved provider routing: %v", err)
	}
	if got := coordinator.Probe(context.Background()).TTS.BYOK; got {
		t.Fatal("saved BYOK disable was ignored at startup")
	}

	reader.config.BYOKEnabled = true
	env.EnableBYOK = false
	if err := applyStoredProviderRouting(context.Background(), reader, coordinator, env); err != nil {
		t.Fatalf("apply provider routing under environment cap: %v", err)
	}
	if got := coordinator.Probe(context.Background()).TTS.BYOK; got {
		t.Fatal("persisted preference bypassed the environment's BYOK upper bound")
	}

	env.EnableBYOK = true
	if err := applyStoredProviderRouting(context.Background(), reader, coordinator, env); err != nil {
		t.Fatalf("re-enable provider routing: %v", err)
	}
	if got := coordinator.Probe(context.Background()).TTS.BYOK; !got {
		t.Fatal("saved BYOK enable was not applied when the environment permits it")
	}
}

func TestBYOKCredentialCheckersAreGeneratedFromProviderCatalog(t *testing.T) {
	checkers, err := declaredBYOKCredentialCheckers(nil, nil)
	if err != nil {
		t.Fatalf("declaredBYOKCredentialCheckers: %v", err)
	}
	want := map[string]string{}
	for _, provider := range providercatalog.All() {
		if provider.Status != providercatalog.StatusActive || provider.Tier != providercatalog.TierBYOK || provider.Probe == nil {
			continue
		}
		if len(provider.Capabilities) != 1 {
			t.Fatalf("catalog provider %q has %d capabilities for one provider health row", provider.ID, len(provider.Capabilities))
		}
		want[provider.ID] = string(provider.Capabilities[0])
	}
	if len(checkers) != len(want) {
		t.Fatalf("generated %d BYOK checkers, want %d from active probed provider declarations", len(checkers), len(want))
	}
	for providerID, capability := range want {
		checker, ok := checkers[providerID].(*capabilities.BYOKCredentialChecker)
		if !ok {
			t.Fatalf("checker %q has type %T; want BYOKCredentialChecker", providerID, checkers[providerID])
		}
		if checker.ProviderID != providerID || checker.Capability != capability || checker.Probe == nil {
			t.Errorf("checker %q = %+v; want declared capability %q and a catalog-backed probe", providerID, checker, capability)
		}
	}
	if _, exists := checkers["openrouter"]; exists {
		t.Fatal("OpenRouter must not receive a direct HTTP health probe for a request-scoped BYOK key")
	}
}

func TestDeclaredVLLMBatchRouteNeedsConfiguredEndpointAndServedModel(t *testing.T) {
	if got := declaredVLLMBatchModels(""); len(got) != 0 {
		t.Fatalf("routes without endpoint = %v, want no opt-in route", got)
	}
	model, found := modelcatalog.Default().ByID("rumble-2-flash")
	if !found {
		t.Fatal("rumble-2-flash model declaration missing")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	routes := declaredVLLMBatchModels(srv.URL)
	provider, ok := routes[model.ID]
	serviceable, implementsServiceability := provider.(interface{ IsServiceable(context.Context) bool })
	if !ok || !implementsServiceability || !serviceable.IsServiceable(context.Background()) {
		t.Fatalf("configured route %q is absent or not serviceable: %+v", model.ID, routes)
	}
	wrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"another-model"}]}`)
	}))
	defer wrong.Close()
	provider, ok = declaredVLLMBatchModels(wrong.URL)[model.ID]
	serviceable, implementsServiceability = provider.(interface{ IsServiceable(context.Context) bool })
	if !ok || !implementsServiceability || serviceable.IsServiceable(context.Background()) {
		t.Fatal("a configured endpoint serving another model must remain unserviceable")
	}
}

func TestDeclaredBatchRoutesRequireDeclaredProtocolAdapter(t *testing.T) {
	model, ok := modelcatalog.Default().ByID("rumble-2-flash")
	if !ok {
		t.Fatal("Rumble model declaration missing")
	}
	undeclaredProtocol := model
	undeclaredProtocol.ID = "another-vllm-model"
	undeclaredProtocol.Name = "Another vLLM model"
	undeclaredProtocol.BatchAdapterID = ""
	catalog := modelcatalog.New(modelcatalog.Seed{Models: []modelcatalog.Model{model, undeclaredProtocol}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"Reson8/Rumble-2-Flash"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	routes := declaredBatchModels(catalog, srv.URL)
	if len(routes) != 1 || routes[model.ID] == nil {
		t.Fatalf("routes = %v; want only explicitly declared protocol model %q", routes, model.ID)
	}
	if _, exists := routes[undeclaredProtocol.ID]; exists {
		t.Fatalf("unconfigured vLLM model %q was routed through the Rumble protocol", undeclaredProtocol.ID)
	}
}

func TestLocalTTSBootstrapIdentityComesFromCatalog(t *testing.T) {
	for _, resource := range []string{"kokoro", "kale-tts"} {
		provider := declaredTTSProviderForResource(resource)
		if provider.ResourceSlug != resource || provider.Tier != providercatalog.TierLocal {
			t.Fatalf("resource %q resolved to unexpected provider: %+v", resource, provider)
		}
		if _, ok := providercatalog.ByID(provider.ID); !ok {
			t.Fatalf("resolved provider %q is absent from provider catalog", provider.ID)
		}
		adapterID, voiceMap := declaredTTSAdapter(provider.ID)
		if adapterID == "" || len(voiceMap) == 0 {
			t.Fatalf("provider %q has no catalog-backed adapter mapping", provider.ID)
		}
		role := "default"
		if resource == "kale-tts" {
			role = "fast"
		}
		modelID := declaredTTSModelID(provider.ID, role)
		model, ok := modelcatalog.Default().ByID(modelID)
		if !ok || model.Capability != modelcatalog.CapabilityTTS || !model.HasRole(role) {
			t.Fatalf("provider %q bootstrap model %q does not satisfy role %q", provider.ID, modelID, role)
		}
	}

	kokoro := declaredTTSProviderForResource("kokoro")
	aliases := providercatalog.RuntimeAliasesForProvider(kokoro.ID)
	if len(aliases) == 0 || aliases[0] != "kokoro-local" {
		t.Fatalf("Kokoro chain runtime identity = %v; expected catalog-declared alias", aliases)
	}
	kale := declaredTTSProviderForResource("kale-tts")
	if kale.Status != providercatalog.StatusPlanned {
		t.Fatalf("Kale provider status changed unexpectedly: %q", kale.Status)
	}
}
