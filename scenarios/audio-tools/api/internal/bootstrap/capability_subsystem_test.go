package bootstrap

import (
	"testing"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/httpc"
	"audio-tools/internal/providercatalog"
	intsumm "audio-tools/internal/summarize"
)

func TestDeclaredSyntheticTTSRuntimeBuildsHealthAndAggregateEntries(t *testing.T) {
	provider, ok := providercatalog.ByID("kale-tts")
	if !ok {
		t.Fatal("Kale declaration missing")
	}
	provider.ID = "synthetic-tts"
	provider.Name = "Synthetic TTS"
	provider.Description = "Synthetic declaration for registration coverage"
	provider.ResourceSlug = "synthetic-tts"
	provider.DependencySlug = "synthetic-tts"
	provider.TTSRuntime = &providercatalog.TTSRuntime{
		BackendID: "synthetic-runtime", URLField: "KaleURL", HealthPath: "/healthz",
		ModelRole: "fast", CacheBackend: "synthetic-model",
	}
	providers := []providercatalog.Provider{provider}
	env := Env{TTSBackend: "synthetic-runtime", KaleURL: "http://synthetic"}
	routes := declaredTTSCapabilityRoutes(providers, env, "http://selected", "synthetic-runtime")
	if len(routes) != 1 || routes[0].provider.ID != provider.ID || routes[0].url != "http://synthetic" || !routes[0].aggregate {
		t.Fatalf("synthetic routes = %#v", routes)
	}
	maps := buildTTSCapabilityMaps(routes, httpc.DefaultDoer(), httpc.DefaultDoer())
	if _, ok := maps.readiness[provider.ID]; !ok {
		t.Fatal("declared TTS readiness checker missing")
	}
	checker, ok := maps.liveness[provider.ID].(*capabilities.ResourceChecker)
	if !ok || checker.URL != "http://synthetic/healthz" {
		t.Fatalf("synthetic liveness checker = %#v", maps.liveness[provider.ID])
	}
	aggregate := ttsLivenessCheckers(routes, httpc.DefaultDoer())
	if len(aggregate) != 1 {
		t.Fatalf("synthetic aggregate entries = %d, want 1", len(aggregate))
	}
	aggregateChecker, ok := aggregate[0].(*capabilities.ResourceChecker)
	if !ok || aggregateChecker.URL != "http://synthetic/healthz" {
		t.Fatalf("synthetic aggregate checker = %#v", aggregate[0])
	}
	defs := []capabilities.Def{}
	appendDeclaredRuntimeDefs(&defs, providers)
	if len(defs) != 1 || defs[0].ID != provider.ID {
		t.Fatalf("synthetic capability definition = %#v", defs)
	}
}

func TestDeclaredHealthProfilesConstructAndAggregate(t *testing.T) {
	providers := providercatalog.All()
	profiles := map[providercatalog.HealthProfile]struct{}{}
	var wantReadinessAggregate, wantLivenessAggregate int
	for _, provider := range providers {
		if provider.HealthProfile == "" {
			continue
		}
		profiles[provider.HealthProfile] = struct{}{}
		if provider.HealthProfile == providercatalog.HealthProfileHTTP {
			if provider.HealthProbe == nil {
				t.Errorf("provider %q declares generic HTTP health without probe facts", provider.ID)
			}
		} else if healthAdapters[provider.HealthProfile] == nil {
			t.Errorf("provider %q declares unregistered health profile %q", provider.ID, provider.HealthProfile)
		}
		if provider.ReadinessAggregate {
			wantReadinessAggregate++
		}
		if provider.LivenessAggregate {
			wantLivenessAggregate++
		}
	}
	if len(profiles) != len(healthAdapters)+1 {
		t.Fatalf("declared health profiles = %d, executable strategies = %d", len(profiles), len(healthAdapters)+1)
	}
	env := Env{WhisperURL: "http://whisper", KyutaiURL: "http://kyutai", SherpaURL: "http://sherpa", OllamaURL: "http://ollama"}
	ready, live, readyAggregate, liveAggregate := resolveDeclaredHealth(providers, env, httpc.DefaultDoer(), httpc.DefaultDoer(), nil, intsumm.DefaultSummarizeConfig())
	for _, provider := range providers {
		if provider.HealthProfile == "" {
			continue
		}
		if provider.HealthProfile == providercatalog.HealthProfileRequestScoped {
			if ready[provider.ID] != nil || live[provider.ID] != nil {
				t.Errorf("request-scoped provider %q unexpectedly has ambient checkers", provider.ID)
			}
			continue
		}
		if ready[provider.ID] == nil || live[provider.ID] == nil {
			t.Errorf("declared provider %q did not construct both checker maps", provider.ID)
		}
	}
	if len(readyAggregate) != wantReadinessAggregate {
		t.Fatalf("readiness aggregate members = %d, declaration says %d", len(readyAggregate), wantReadinessAggregate)
	}
	if len(liveAggregate) != wantLivenessAggregate {
		t.Fatalf("liveness aggregate members = %d, declaration says %d", len(liveAggregate), wantLivenessAggregate)
	}

	// A newly declared provider reuses an existing profile and aggregate flags;
	// it constructs without adding a provider-specific adapter branch.
	var synthetic providercatalog.Provider
	for _, provider := range providers {
		if provider.HealthProfile == providercatalog.HealthProfileWhisperSilent {
			synthetic = provider
			break
		}
	}
	if synthetic.HealthProfile == "" {
		t.Fatal("whisper-silent profile missing")
	}
	synthetic.ID, synthetic.Name, synthetic.Description = "synthetic-health", "Synthetic", "Synthetic profile reuse"
	withSynthetic := append(append([]providercatalog.Provider(nil), providers...), synthetic)
	ready, live, readyAggregate, liveAggregate = resolveDeclaredHealth(withSynthetic, env, httpc.DefaultDoer(), httpc.DefaultDoer(), nil, intsumm.DefaultSummarizeConfig())
	if ready[synthetic.ID] == nil || live[synthetic.ID] == nil {
		t.Fatal("synthetic provider did not reuse its declared profile")
	}
	if len(readyAggregate) != wantReadinessAggregate+1 || len(liveAggregate) != wantLivenessAggregate+1 {
		t.Fatal("synthetic provider aggregate membership did not follow its declaration")
	}

	for _, provider := range providers {
		switch provider.HealthProfile {
		case providercatalog.HealthProfileWhisperSilent:
			if _, ok := ready[provider.ID].(*capabilities.WhisperChecker); !ok {
				t.Errorf("Whisper checker for %q = %T", provider.ID, ready[provider.ID])
			}
		case providercatalog.HealthProfileOllamaModelAware:
			if _, ok := ready[provider.ID].(*capabilities.OllamaChecker); !ok {
				t.Errorf("Ollama checker for %q = %T", provider.ID, ready[provider.ID])
			}
		case providercatalog.HealthProfileHTTP:
			checker, ok := live[provider.ID].(*capabilities.ResourceChecker)
			if ok && provider.HealthProbe != nil {
				want := healthBaseURL(env, provider.HealthProbe.BaseURLField) + provider.HealthProbe.LivenessPath
				if checker.URL != want {
					t.Errorf("declared HTTP liveness for %q = %q, want %q", provider.ID, checker.URL, want)
				}
			} else if !ok {
				t.Errorf("declared HTTP liveness for %q = %T", provider.ID, live[provider.ID])
			}
		case providercatalog.HealthProfileTranscode:
			if _, ok := live[provider.ID].(*capabilities.TranscodeChecker); !ok {
				t.Errorf("transcode checker for %q = %T", provider.ID, live[provider.ID])
			}
		}
	}
}

func TestBuildCapabilitySubsystemKeepsTTSCapabilityAndLivenessAligned(t *testing.T) {
	tests := []struct {
		name             string
		backend          string
		selectedBackend  string
		wantAggregate    int
		wantKaleLiveness bool
		wantKokoroLive   bool
		wantRouteIndex   int
		wantRouteURL     string
	}{
		{name: "shared sherpa runtime", backend: "sherpa-onnx", selectedBackend: "sherpa-onnx", wantAggregate: 4, wantKokoroLive: true, wantRouteIndex: 3, wantRouteURL: "http://selected-tts/healthz/tts"},
		{name: "explicit kale", backend: "kale-onnx", selectedBackend: "kale-onnx", wantAggregate: 4, wantKaleLiveness: true, wantKokoroLive: true, wantRouteIndex: 3, wantRouteURL: "http://kale/healthz"},
		{name: "auto retains both routes", backend: "auto", selectedBackend: "kokoro", wantAggregate: 5, wantKaleLiveness: true, wantKokoroLive: true, wantRouteIndex: 3, wantRouteURL: "http://kokoro/v1/audio/voices"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := declaredTTSProviderForBackend(tt.selectedBackend)
			if err != nil {
				t.Fatalf("declared provider: %v", err)
			}
			env := Env{
				TTSBackend: tt.backend,
				WhisperURL: "http://whisper",
				KyutaiURL:  "http://kyutai",
				SherpaURL:  "http://sherpa",
				KokoroURL:  "http://kokoro",
				KaleURL:    "http://kale",
				OllamaURL:  "http://ollama",
			}
			built, err := buildCapabilitySubsystem(env, "http://selected-tts", tt.selectedBackend, provider, map[string]capabilities.Checker{
				"openrouter": &capabilities.StaticChecker{Available: func() (bool, string) { return true, "configured" }},
			})
			if err != nil {
				t.Fatalf("build capabilities: %v", err)
			}
			aggregate, ok := built.livenessCheckers["audio-tools"].(capabilities.AggregateChecker)
			if !ok {
				t.Fatalf("audio-tools liveness checker has type %T", built.livenessCheckers["audio-tools"])
			}
			if got := len(aggregate.Checkers); got != tt.wantAggregate {
				t.Fatalf("audio-tools liveness aggregates %d checkers, want %d", got, tt.wantAggregate)
			}
			route, ok := aggregate.Checkers[tt.wantRouteIndex].(*capabilities.ResourceChecker)
			if !ok {
				t.Fatalf("selected TTS liveness checker has type %T", aggregate.Checkers[tt.wantRouteIndex])
			}
			if route.URL != tt.wantRouteURL {
				t.Fatalf("selected TTS liveness URL = %q, want %q", route.URL, tt.wantRouteURL)
			}
			_, hasKale := built.livenessCheckers["kale-tts"]
			if hasKale != tt.wantKaleLiveness {
				t.Fatalf("Kale liveness present = %v, want %v", hasKale, tt.wantKaleLiveness)
			}
			_, hasKokoro := built.livenessCheckers["kokoro-tts"]
			if hasKokoro != tt.wantKokoroLive {
				t.Fatalf("Kokoro liveness present = %v, want %v", hasKokoro, tt.wantKokoroLive)
			}
		})
	}
}
