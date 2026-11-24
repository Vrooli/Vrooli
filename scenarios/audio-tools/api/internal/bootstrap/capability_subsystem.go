package bootstrap

import (
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/httpc"
	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
	intsumm "audio-tools/internal/summarize"
)

type capabilitySubsystem struct {
	registry         *capabilities.Registry
	livenessCheckers map[string]capabilities.Checker
	doer             httpc.Doer
	summarizeConfig  intsumm.SummarizeConfig
}

type ttsCapabilityRoute struct {
	provider      providercatalog.Provider
	backend       string
	url           string
	path          string
	readinessKind string
	aggregate     bool
}

type healthCheckers struct {
	readiness, liveness capabilities.Checker
	aggregateLiveness   capabilities.Checker
}

type healthAdapter func(Env, httpc.Doer, httpc.Doer, capabilities.Checker, intsumm.SummarizeConfig) healthCheckers

// Adapter selection is by declared health profile. Provider IDs do not appear
// here, so providers can reuse a profile without adding another branch.
var healthAdapters = map[providercatalog.HealthProfile]healthAdapter{
	providercatalog.HealthProfileWhisperSilent: func(env Env, ready, live httpc.Doer, _ capabilities.Checker, _ intsumm.SummarizeConfig) healthCheckers {
		return healthCheckers{readiness: &capabilities.WhisperChecker{BaseURL: env.WhisperURL, Doer: ready}, liveness: &capabilities.ResourceChecker{URL: env.WhisperURL + "/", Doer: live}}
	},
	providercatalog.HealthProfileOllamaModelAware: func(env Env, ready, live httpc.Doer, _ capabilities.Checker, cfg intsumm.SummarizeConfig) healthCheckers {
		return healthCheckers{readiness: &capabilities.OllamaChecker{BaseURL: env.OllamaURL, Doer: ready, ModelFn: func() string { return cfg.Model }}, liveness: &capabilities.ResourceChecker{URL: env.OllamaURL + "/api/tags", Doer: live}}
	},
	providercatalog.HealthProfileRequestScoped: func(_ Env, _, _ httpc.Doer, checker capabilities.Checker, _ intsumm.SummarizeConfig) healthCheckers {
		return healthCheckers{readiness: checker, liveness: checker}
	},
	providercatalog.HealthProfileTranscode: func(_ Env, _, _ httpc.Doer, _ capabilities.Checker, _ intsumm.SummarizeConfig) healthCheckers {
		return healthCheckers{readiness: &capabilities.TranscodeChecker{}, liveness: &capabilities.TranscodeChecker{}}
	},
}

func resolveDeclaredHealth(providers []providercatalog.Provider, env Env, readinessDoer, livenessDoer httpc.Doer, byok map[string]capabilities.Checker, cfg intsumm.SummarizeConfig) (map[string]capabilities.Checker, map[string]capabilities.Checker, []capabilities.Checker, []capabilities.Checker) {
	readiness, liveness := map[string]capabilities.Checker{}, map[string]capabilities.Checker{}
	var readinessAggregate, livenessAggregate []capabilities.Checker
	for _, provider := range providers {
		if provider.HealthProfile == "" || provider.HealthProfile == providercatalog.HealthProfileNone {
			continue
		}
		if provider.HealthProfile == providercatalog.HealthProfileHTTP {
			checkers := declaredHTTPHealth(provider, env, readinessDoer, livenessDoer)
			storeHealthCheckers(provider, checkers, readiness, liveness, &readinessAggregate, &livenessAggregate)
			continue
		}
		adapter := healthAdapters[provider.HealthProfile]
		if adapter == nil {
			continue
		}
		checkers := adapter(env, readinessDoer, livenessDoer, byok[provider.ID], cfg)
		storeHealthCheckers(provider, checkers, readiness, liveness, &readinessAggregate, &livenessAggregate)
	}
	return readiness, liveness, readinessAggregate, livenessAggregate
}

func declaredHTTPHealth(provider providercatalog.Provider, env Env, ready, live httpc.Doer) healthCheckers {
	probe := provider.HealthProbe
	if probe == nil {
		return healthCheckers{}
	}
	base := healthBaseURL(env, probe.BaseURLField)
	if base == "" {
		return healthCheckers{}
	}
	checkers := healthCheckers{
		readiness: &capabilities.ResourceChecker{URL: base + probe.ReadinessPath, Doer: ready},
		liveness:  &capabilities.ResourceChecker{URL: base + probe.LivenessPath, Doer: live},
	}
	if probe.AggregateLivenessPath != "" {
		checkers.aggregateLiveness = &capabilities.ResourceChecker{URL: base + probe.AggregateLivenessPath, Doer: live}
	}
	return checkers
}

func healthBaseURL(env Env, field providercatalog.HealthBaseURLField) string {
	switch field {
	case providercatalog.HealthBaseURLWhisper:
		return env.WhisperURL
	case providercatalog.HealthBaseURLKyutai:
		return env.KyutaiURL
	case providercatalog.HealthBaseURLSherpa:
		return env.SherpaURL
	case providercatalog.HealthBaseURLOllama:
		return env.OllamaURL
	default:
		return ""
	}
}

func storeHealthCheckers(provider providercatalog.Provider, checkers healthCheckers, readiness, liveness map[string]capabilities.Checker, readinessAggregate, livenessAggregate *[]capabilities.Checker) {
	if checkers.readiness != nil {
		readiness[provider.ID] = checkers.readiness
	}
	if checkers.liveness != nil {
		liveness[provider.ID] = checkers.liveness
	}
	if provider.ReadinessAggregate && checkers.readiness != nil {
		*readinessAggregate = append(*readinessAggregate, checkers.readiness)
	}
	if provider.LivenessAggregate {
		aggregateChecker := checkers.aggregateLiveness
		if aggregateChecker == nil {
			aggregateChecker = checkers.liveness
		}
		if aggregateChecker != nil {
			*livenessAggregate = append(*livenessAggregate, aggregateChecker)
		}
	}
}

func buildCapabilitySubsystem(env Env, localTTSURL, selectedTTSBackend string, selectedProvider providercatalog.Provider, byokCheckers map[string]capabilities.Checker) (capabilitySubsystem, error) {
	doer := httpc.DefaultDoer()
	readinessDoer := httpc.ReadinessDoer()
	livenessDoer := httpc.LivenessDoer()
	summCfg := intsumm.DefaultSummarizeConfig()
	summCfg.Model = env.SummarizeDefaultModel
	defs := capabilities.KnownForPlatform("")
	providers := ttsRuntimeProviders(env, selectedTTSBackend, selectedProvider)
	appendDeclaredRuntimeDefs(&defs, providers)
	routes := declaredTTSCapabilityRoutes(providers, env, localTTSURL, selectedTTSBackend)
	ttsMaps := buildTTSCapabilityMaps(routes, readinessDoer, livenessDoer)
	capsCheckers := ttsMaps.readiness
	livenessCheckers := ttsMaps.liveness
	allProviders := providercatalog.All()
	declaredReadiness, declaredLiveness, readinessAggregate, livenessAggregate := resolveDeclaredHealth(allProviders, env, readinessDoer, livenessDoer, byokCheckers, summCfg)
	sharedCapabilityCheckers := declaredReadiness
	for id, checker := range declaredLiveness {
		livenessCheckers[id] = checker
	}
	for id, checker := range capsCheckers {
		sharedCapabilityCheckers[id] = checker
	}
	readinessAggregate = append(readinessAggregate, ttsReadinessCheckers(routes, readinessDoer)...)
	sharedCapabilityCheckers["audio-tools"] = capabilities.AggregateChecker{Checkers: readinessAggregate}
	// These are non-TTS health rows. TTS rows and the aggregate are assembled
	// from the local provider declarations above.
	livenessAggregate = append(livenessAggregate, ttsLivenessCheckers(routes, livenessDoer)...)
	livenessCheckers["audio-tools"] = capabilities.AggregateChecker{Checkers: livenessAggregate}
	capsRegistry := capabilities.NewRegistry(defs, sharedCapabilityCheckers, 30*time.Second)
	capsRegistry.SetLivenessCheckers(livenessCheckers)
	return capabilitySubsystem{registry: capsRegistry, livenessCheckers: livenessCheckers, doer: doer, summarizeConfig: summCfg}, nil
}

func ttsRuntimeProviders(env Env, selectedBackend string, selected providercatalog.Provider) []providercatalog.Provider {
	if env.TTSBackend == "auto" {
		var eligible []providercatalog.Provider
		models := modelcatalog.Default()
		for _, provider := range providercatalog.All() {
			if autoTTSProviderEligible(provider, models) {
				eligible = append(eligible, provider)
			}
		}
		return eligible
	}
	providers := make([]providercatalog.Provider, 0)
	for _, provider := range providercatalog.All() {
		if provider.Tier == providercatalog.TierLocal && provider.TTSRuntime != nil && provider.Status == providercatalog.StatusActive {
			providers = append(providers, provider)
		}
	}
	if selected.ID != "" {
		found := false
		for _, provider := range providers {
			if provider.ID == selected.ID {
				found = true
			}
		}
		if !found {
			providers = append(providers, selected)
		}
	}
	return providers
}

func declaredTTSCapabilityRoutes(providers []providercatalog.Provider, env Env, selectedURL, selectedBackend string) []ttsCapabilityRoute {
	urls := map[string]string{"KokoroURL": env.KokoroURL, "KaleURL": env.KaleURL, "SherpaURL": env.SherpaURL}
	routes := make([]ttsCapabilityRoute, 0, len(providers))
	for _, provider := range providers {
		if provider.TTSRuntime == nil {
			continue
		}
		runtime := provider.TTSRuntime
		backend, field, path := runtime.BackendID, runtime.URLField, runtime.HealthPath
		sharedSelected := runtime.SharedRuntime != nil && runtime.SharedRuntime.BackendID == selectedBackend && env.TTSBackend != "auto"
		if sharedSelected {
			backend, field, path = runtime.SharedRuntime.BackendID, runtime.SharedRuntime.URLField, runtime.SharedRuntime.HealthPath
		}
		baseURL := urls[field]
		if sharedSelected {
			baseURL = selectedURL
		}
		readinessKind := runtime.ReadinessKind
		if sharedSelected {
			readinessKind = "health"
		}
		routes = append(routes, ttsCapabilityRoute{provider: provider, backend: backend, url: baseURL, path: path, readinessKind: readinessKind, aggregate: env.TTSBackend == "auto" || backend == selectedBackend})
	}
	return routes
}

func buildTTSCapabilityMaps(routes []ttsCapabilityRoute, readinessDoer, livenessDoer httpc.Doer) struct {
	readiness, liveness map[string]capabilities.Checker
} {
	result := struct {
		readiness, liveness map[string]capabilities.Checker
	}{map[string]capabilities.Checker{}, map[string]capabilities.Checker{}}
	for _, route := range routes {
		if route.aggregate {
			var readiness capabilities.Checker
			if route.readinessKind == "synthesize" {
				readiness = &capabilities.KokoroChecker{BaseURL: route.url, Doer: readinessDoer}
			} else {
				readiness = &capabilities.ResourceChecker{URL: route.url + route.path, Doer: readinessDoer}
			}
			result.readiness[route.provider.ID] = readiness
		}
		result.liveness[route.provider.ID] = &capabilities.ResourceChecker{URL: route.url + route.path, Doer: livenessDoer}
	}
	return result
}

func ttsReadinessCheckers(routes []ttsCapabilityRoute, doer httpc.Doer) []capabilities.Checker {
	checkers := make([]capabilities.Checker, 0, len(routes))
	for _, route := range routes {
		if !route.aggregate {
			continue
		}
		if route.readinessKind == "synthesize" {
			checkers = append(checkers, &capabilities.KokoroChecker{BaseURL: route.url, Doer: doer})
		} else {
			checkers = append(checkers, &capabilities.ResourceChecker{URL: route.url + route.path, Doer: doer})
		}
	}
	return checkers
}

func ttsLivenessCheckers(routes []ttsCapabilityRoute, doer httpc.Doer) []capabilities.Checker {
	checkers := make([]capabilities.Checker, 0, len(routes))
	for _, route := range routes {
		if !route.aggregate {
			continue
		}
		checkers = append(checkers, &capabilities.ResourceChecker{URL: route.url + route.path, Doer: doer})
	}
	return checkers
}

func appendDeclaredRuntimeDefs(defs *[]capabilities.Def, providers []providercatalog.Provider) {
	for _, provider := range providers {
		if hasCapability(*defs, provider.ID) {
			continue
		}
		*defs = append(*defs, capabilities.Def{
			ID: provider.ID, Name: provider.Name, Description: provider.Description,
			DependencyKind: capabilities.DependencyKind(provider.DependencyKind), DependencySlug: provider.DependencySlug,
			Features:    append([]string(nil), provider.Features...),
			Platform:    capabilities.PlatformVerdict{Support: capabilities.PlatformSupport(provider.Platform.Support), Reason: provider.Platform.Reason},
			Criticality: capabilities.Criticality(provider.Criticality), ActionKind: capabilities.ActionKind(provider.DefActionKind), OperatorCommand: provider.OperatorCommand,
		})
	}
}
