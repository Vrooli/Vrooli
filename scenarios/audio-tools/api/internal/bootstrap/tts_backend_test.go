package bootstrap

import (
	"context"
	"testing"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
)

func TestResolveTTSBackendAutoUsesEligiblePriorityAndPreservesExplicitChoice(t *testing.T) {
	for _, tc := range []struct {
		name, requested string
		want            string
	}{
		{name: "auto prefers conditional Kale", requested: "auto", want: "kale-onnx"},
		{name: "explicit Kokoro", requested: "kokoro", want: "kokoro"},
		{name: "explicit Kale evaluation", requested: "kale-onnx", want: "kale-onnx"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTTSBackend(tc.requested, declaredAutoTTSProviders())
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("resolved backend = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDeclaredQualifiedDefaultTTSProviderUsesProviderAndModelDeclarations(t *testing.T) {
	providers := providercatalog.All()
	models := modelcatalog.Default()
	got := declaredQualifiedDefaultTTSProvider(providers, models)
	if got.TTSRuntime == nil || got.TTSRuntime.ModelRole != "default" {
		t.Fatalf("selected provider %q does not declare the default TTS role", got.ID)
	}
	found := false
	for _, model := range models.ByProvider(got.ID) {
		if model.Capability == modelcatalog.CapabilityTTS && model.HasRole(got.TTSRuntime.ModelRole) && model.DefaultEligible() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("selected provider %q has no eligible declared default model", got.ID)
	}
	if got.ResourceSlug == "" {
		t.Fatalf("selected provider %q has no resource for bootstrap", got.ID)
	}
}

func TestResolveTTSBackendAutoFailsWhenNoProviderIsEligible(t *testing.T) {
	if _, err := resolveTTSBackend("auto", nil); err == nil {
		t.Fatal("auto TTS should fail closed when no qualified provider is declared")
	}
}

func TestAutoTTSEligibilityAllowsOnlyExplicitConditionalCandidate(t *testing.T) {
	models := modelcatalog.Default()
	kokoro, ok := providercatalog.ByID("kokoro-tts")
	if !ok || !autoTTSProviderEligible(kokoro, models) {
		t.Fatal("active Kokoro with an available, commercially usable default model should be auto-eligible")
	}
	kale, ok := providercatalog.ByID("kale-tts")
	if !ok {
		t.Fatal("Kale declaration is missing")
	}
	runtimeCopy := *kale.TTSRuntime
	kale.TTSRuntime = &runtimeCopy
	if !autoTTSProviderEligible(kale, models) {
		t.Fatal("explicitly conditional planned Kale with a commercial candidate model should be tried when available")
	}
	providers := declaredAutoTTSProviders()
	if len(providers) < 2 || providers[0].ID != "kale-tts" || providers[1].ID != "kokoro-tts" {
		t.Fatalf("conditional auto order = %v, want Kale before Kokoro", providerIDs(providers))
	}

	// The opt-in is required: an otherwise identical planned candidate is
	// excluded unless its provider declaration explicitly enables conditional auto.
	runtimeCopy = *kale.TTSRuntime
	runtimeCopy.ConditionalAuto = false
	kale.TTSRuntime = &runtimeCopy
	if autoTTSProviderEligible(kale, models) {
		t.Fatal("unflagged planned provider with candidate model must remain excluded")
	}
	runtimeCopy.ConditionalAuto = true
	kale.TTSRuntime = &runtimeCopy
	kale.Status = providercatalog.StatusActive
	if autoTTSProviderEligible(kale, models) {
		t.Fatal("conditional auto must not promote a provider into active status")
	}
	kale.Status = providercatalog.StatusPlanned

	model := models.ByProvider(kale.ID)[0]
	model.Commercial = modelcatalog.CommercialNo
	conditionalNonCommercial := modelcatalog.New(modelcatalog.Seed{Models: []modelcatalog.Model{model}})
	if autoTTSProviderEligible(kale, conditionalNonCommercial) {
		t.Fatal("conditional auto must exclude non-commercial models")
	}
	model.Readiness = modelcatalog.ReadinessAvailable
	model.Commercial = modelcatalog.CommercialYes
	model.Roles = append(model.Roles, "default")
	qualifiedDefault := modelcatalog.New(modelcatalog.Seed{Models: []modelcatalog.Model{model}})
	runtimeCopy.ModelRole = "default"
	kale.TTSRuntime = &runtimeCopy
	if autoTTSProviderEligible(kale, qualifiedDefault) {
		t.Fatal("conditional auto must not bypass the planned-provider/default-role gate")
	}
}

func TestSortAutoTTSProvidersUsesDeclaredPriority(t *testing.T) {
	providers := declaredAutoTTSProviders()
	if len(providers) != 2 || providers[0].ID != "kale-tts" || providers[1].ID != "kokoro-tts" {
		t.Fatalf("auto providers = %v, want conditional Kale followed by qualified Kokoro", providerIDs(providers))
	}
	first, second := providers[0], providers[0]
	firstRuntime, secondRuntime := *first.TTSRuntime, *second.TTSRuntime
	first.ID, firstRuntime.AutoPriority = "first", 20
	second.ID, secondRuntime.AutoPriority = "second", 10
	first.TTSRuntime, second.TTSRuntime = &firstRuntime, &secondRuntime
	ordered := []providercatalog.Provider{first, second}
	sortAutoTTSProviders(ordered)
	if ordered[0].ID != "second" || ordered[1].ID != "first" {
		t.Fatalf("priority order = %q, %q; want second, first", ordered[0].ID, ordered[1].ID)
	}
}

func providerIDs(providers []providercatalog.Provider) []string {
	ids := make([]string, len(providers))
	for i, provider := range providers {
		ids[i] = provider.ID
	}
	return ids
}

func TestDeclaredTTSProviderResolvesRuntimeIdentityFromBackendDeclaration(t *testing.T) {
	for _, provider := range declaredAutoTTSProviders() {
		got, err := declaredTTSProviderForBackend(provider.TTSRuntime.BackendID)
		if err != nil {
			t.Fatalf("backend %q: %v", provider.TTSRuntime.BackendID, err)
		}
		if got.ID != provider.ID || got.TTSRuntime.ModelRole != provider.TTSRuntime.ModelRole {
			t.Errorf("backend %q resolved to provider %q, want %q", provider.TTSRuntime.BackendID, got.ID, provider.ID)
		}
		modelID := declaredTTSModelID(provider.ID, provider.TTSRuntime.ModelRole)
		if modelID == "" {
			t.Errorf("backend %q has no declared model identity", provider.TTSRuntime.BackendID)
		}
		adapterID, voices := declaredTTSAdapter(provider.ID)
		if adapterID != provider.AdapterID || len(voices) == 0 {
			t.Errorf("backend %q identity adapter=%q declared=%q voices=%v", provider.TTSRuntime.BackendID, adapterID, provider.AdapterID, voices)
		}
		if provider.TTSRuntime.CacheBackend == "" {
			t.Errorf("backend %q has no declared cache namespace", provider.TTSRuntime.BackendID)
		}
		if got := declaredTTSSpeedMode(provider, provider.TTSRuntime.BackendID); got != provider.TTSRuntime.SpeedMode {
			t.Errorf("backend %q speed mode = %q, want declared %q", provider.TTSRuntime.BackendID, got, provider.TTSRuntime.SpeedMode)
		}
	}
	if _, err := declaredTTSProviderForBackend("undeclared-backend"); err == nil {
		t.Fatal("unknown backend should fail closed")
	}
}

func TestDeclaredSherpaRuntimeKeepsKokoroIdentityAndNamespacedCache(t *testing.T) {
	provider, err := declaredTTSProviderForBackend("sherpa-onnx")
	if err != nil {
		t.Fatal(err)
	}
	if provider.ID != "kokoro-tts" || provider.TTSRuntime.SharedRuntime == nil {
		t.Fatalf("Sherpa runtime provider = %#v, want Kokoro with a declared shared runtime", provider)
	}
	runtime := provider.TTSRuntime.SharedRuntime
	if runtime.URLField != "SherpaURL" || runtime.HealthPath != "/healthz/tts" {
		t.Fatalf("shared runtime endpoint = %#v", runtime)
	}
	if got := declaredTTSCacheBackend(provider, "sherpa-onnx"); got != "sherpa-onnx:kokoro" {
		t.Fatalf("Sherpa cache namespace = %q", got)
	}
	if got := declaredTTSModelIDForBackend(provider, "sherpa-onnx"); got != "kokoro-82m-v1.0-sherpa-int8" {
		t.Fatalf("Sherpa provider model identity = %q", got)
	}
	if got := declaredTTSRuntimeModelIDForBackend(provider, "sherpa-onnx"); got != "kokoro" {
		t.Fatalf("Sherpa protocol model = %q", got)
	}
	if got := declaredTTSSpeedMode(provider, "sherpa-onnx"); got != "native" {
		t.Fatalf("Sherpa speed mode = %q, want shared-runtime declaration", got)
	}
	url, err := declaredTTSBackendURL(provider, "sherpa-onnx", map[string]string{"SherpaURL": "http://sherpa.test"})
	if err != nil || url != "http://sherpa.test" {
		t.Fatalf("Sherpa URL = %q, err=%v", url, err)
	}
}

func TestDeclaredTTSHealthURLUsesProviderRuntimeHealthPath(t *testing.T) {
	provider, err := declaredTTSProviderForBackend("kale-onnx")
	if err != nil {
		t.Fatal(err)
	}
	got, err := declaredTTSHealthURL(provider, "kale-onnx", "http://kale.test")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://kale.test/healthz" {
		t.Fatalf("Kale health URL = %q, want the declared runtime endpoint", got)
	}
	if _, err := declaredTTSHealthURL(provider, "wrong-backend", "http://kale.test"); err == nil {
		t.Fatal("health URL resolution should fail for a backend not owned by the provider")
	}
}

func TestDeclaredTTSURLRejectsUnknownEndpointField(t *testing.T) {
	provider := providercatalog.Provider{ID: "bad-tts", TTSRuntime: &providercatalog.TTSRuntime{URLField: "ARBITRARY_ENV"}}
	if _, err := declaredTTSURL(provider, map[string]string{"ARBITRARY_ENV": "http://example.test"}); err == nil {
		t.Fatal("expected unsupported endpoint field to fail closed")
	}
}

func TestLoadDefaultsTTSBackendToAuto(t *testing.T) {
	t.Setenv("AUDIO_TTS_BACKEND", "")
	if got := Load().TTSBackend; got != "auto" {
		t.Fatalf("default TTS backend = %q, want auto", got)
	}
}

func TestResolveTTSStartupPlanPreservesConfiguredLPBSBaseURL(t *testing.T) {
	const configured = "https://lpbs.example.test"
	_, env, err := resolveTTSStartupPlan(context.Background(), Env{
		TTSBackend:  "kokoro",
		LPBSBaseURL: configured,
	})
	if err != nil {
		t.Fatal(err)
	}
	if env.LPBSBaseURL != configured {
		t.Fatalf("LPBS base URL = %q, want configured override %q", env.LPBSBaseURL, configured)
	}
}
