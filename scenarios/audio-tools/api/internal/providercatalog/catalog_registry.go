package providercatalog

import (
	"sort"
	"strings"
)

func cloneProvider(p Provider) Provider {
	p.Capabilities = append([]Capability(nil), p.Capabilities...)
	p.Features = append([]string(nil), p.Features...)
	p.EngineIDs = append([]string(nil), p.EngineIDs...)
	p.RuntimeAliases = append([]string(nil), p.RuntimeAliases...)
	p.Actions = append([]Action(nil), p.Actions...)
	if p.Probe != nil {
		probe := *p.Probe
		p.Probe = &probe
	}
	if p.ConsentRequirement != nil {
		consent := *p.ConsentRequirement
		p.ConsentRequirement = &consent
	}
	if p.TTSRuntime != nil {
		runtime := *p.TTSRuntime
		if runtime.SharedRuntime != nil {
			shared := *runtime.SharedRuntime
			runtime.SharedRuntime = &shared
		}
		p.TTSRuntime = &runtime
	}
	return p
}

// Default returns the process-wide validated catalogue. A malformed seed is a
// build-time defect, not a runtime condition.
func Default() *Registry {
	return registry
}

var registry = func() *Registry {
	r, err := Load()
	if err != nil {
		panic(err)
	}
	return r
}()

// SchemaVersion returns the seed schema version.
func (r *Registry) SchemaVersion() string { return r.schemaVersion }

// All returns every declared provider in seed order.
func (r *Registry) All() []Provider {
	out := make([]Provider, len(r.providers))
	for i := range r.providers {
		out[i] = cloneProvider(r.providers[i])
	}
	return out
}

// Active returns the providers that are probed and rendered into
// capabilities.Known, in seed order. Planned, experimental, and watch entries
// are excluded.
func (r *Registry) Active() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status == StatusActive {
			out = append(out, cloneProvider(p))
		}
	}
	return out
}

// ByID returns a provider by stable provider ID.
func (r *Registry) ByID(id string) (Provider, bool) {
	p, ok := r.byID[id]
	if !ok {
		return Provider{}, false
	}
	return cloneProvider(p), true
}

// ByCapability returns active providers that declare the capability, in seed
// order. Planned/watch entries are never returned.
func (r *Registry) ByCapability(capability Capability) []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status == StatusActive && hasCapability(&p, capability) {
			out = append(out, cloneProvider(p))
		}
	}
	return out
}

// CapabilityForFeature maps a declared feature string to its catalogue
// capability.
func (r *Registry) CapabilityForFeature(feature string) (Capability, bool) {
	capability, ok := featureCapabilities[feature]
	return capability, ok
}

// TierForProviderID returns the routing tier for a provider ID, or TierNone for
// an unknown ID.
func (r *Registry) TierForProviderID(id string) Tier {
	if p, ok := r.byID[id]; ok {
		return p.Tier
	}
	return TierNone
}

// ResourceSlugForProviderID returns the `vrooli resource <verb> <slug>` slug for
// an active local resource provider. Non-local, planned, and unknown IDs return
// false.
func (r *Registry) ResourceSlugForProviderID(id string) (string, bool) {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive || p.Kind != KindLocalResource || strings.TrimSpace(p.ResourceSlug) == "" {
		return "", false
	}
	return p.ResourceSlug, true
}

// IsLocalProvider reports whether the provider ID is an active local resource
// that lifecycle actions may target.
func (r *Registry) IsLocalProvider(id string) bool {
	_, ok := r.ResourceSlugForProviderID(id)
	return ok
}

// SupportsPullModel reports whether the active provider advertises pull_model.
func (r *Registry) SupportsPullModel(id string) bool {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive {
		return false
	}
	for _, action := range p.Actions {
		if action == ActionPullModel {
			return true
		}
	}
	return false
}

// ActionsForProvider returns a copy of the lifecycle actions advertised by an
// active local resource provider.
func (r *Registry) ActionsForProvider(id string) []Action {
	p, ok := r.byID[id]
	if !ok || p.Status != StatusActive || p.Kind != KindLocalResource {
		return nil
	}
	return append([]Action(nil), p.Actions...)
}

// LocalProviderIDs returns the active local-resource providers that own a
// lifecycle row, in seed order. Providers that share a resource slug collapse
// to the first/primary row, so ListLocalProviders never emits a duplicate
// resource (sherpa-stt and speaker-verification both map to sherpa-onnx).
func (r *Registry) LocalProviderIDs() []string {
	seenSlugs := make(map[string]struct{})
	out := make([]string, 0, len(r.providers))
	for _, p := range r.providers {
		if p.Status != StatusActive || p.Kind != KindLocalResource || len(p.Actions) == 0 || strings.TrimSpace(p.ResourceSlug) == "" {
			continue
		}
		if _, dup := seenSlugs[p.ResourceSlug]; dup {
			continue
		}
		seenSlugs[p.ResourceSlug] = struct{}{}
		out = append(out, p.ID)
	}
	return out
}

// ProviderIDForEngine maps an STT engine ID (for example sherpa-streaming) to
// the stable provider ID used by health, lifecycle, settings, and usage (for
// example sherpa-stt).
func (r *Registry) ProviderIDForEngine(engineID string) (string, bool) {
	id, ok := r.providerByEngine[engineID]
	return id, ok
}

// EngineIDsForProvider returns the STT engine IDs declared by a provider.
func (r *Registry) EngineIDsForProvider(providerID string) []string {
	p, ok := r.byID[providerID]
	if !ok {
		return nil
	}
	return append([]string(nil), p.EngineIDs...)
}

// ProviderIDForRuntimeAlias maps a runtime identity that is not an engine ID
// (for example kokoro-local) to the stable provider ID it canonicalizes to.
func (r *Registry) ProviderIDForRuntimeAlias(raw string) (string, bool) {
	id, ok := r.providerByAlias[raw]
	return id, ok
}

// ProviderIDForRuntimeIdentity resolves any identity a runtime can report:
// canonical provider IDs, STT engine IDs, or declared adapter aliases. Keeping
// this join here prevents callers from maintaining their own ordered lookup
// branches as provider identity vocabularies evolve.
func (r *Registry) ProviderIDForRuntimeIdentity(raw string) (string, bool) {
	if _, ok := r.byID[raw]; ok {
		return raw, true
	}
	if id, ok := r.ProviderIDForEngine(raw); ok {
		return id, true
	}
	return r.ProviderIDForRuntimeAlias(raw)
}

// RuntimeAliasesForProvider returns the runtime identities declared by a
// provider.
func (r *Registry) RuntimeAliasesForProvider(providerID string) []string {
	p, ok := r.byID[providerID]
	if !ok {
		return nil
	}
	return append([]string(nil), p.RuntimeAliases...)
}

// ProbeFor returns the remote readiness probe fact for a provider. The probe
// carries the endpoint and header shape only, never a credential.
func (r *Registry) ProbeFor(providerID string) (Probe, bool) {
	p, ok := r.byID[providerID]
	if !ok || p.Probe == nil {
		return Probe{}, false
	}
	return *p.Probe, true
}

// ConsentRequirementFor returns the declared consent contract for a provider.
func (r *Registry) ConsentRequirementFor(providerID string) (ConsentRequirement, bool) {
	p, ok := r.byID[providerID]
	if !ok || p.ConsentRequirement == nil {
		return ConsentRequirement{}, false
	}
	return *p.ConsentRequirement, true
}

// All returns every provider from the default catalogue.
func All() []Provider { return registry.All() }

// Active returns the active providers from the default catalogue.
func Active() []Provider { return registry.Active() }

// ByID returns a provider from the default catalogue.
func ByID(id string) (Provider, bool) { return registry.ByID(id) }

// ByCapability returns active providers for a capability from the default
// catalogue.
func ByCapability(capability Capability) []Provider { return registry.ByCapability(capability) }

// CapabilityForFeature maps a feature using the default catalogue.
func CapabilityForFeature(feature string) (Capability, bool) {
	return registry.CapabilityForFeature(feature)
}

// TierForProviderID returns the routing tier using the default catalogue.
func TierForProviderID(id string) Tier { return registry.TierForProviderID(id) }

// ResourceSlugForProviderID returns the local resource slug using the default
// catalogue.
func ResourceSlugForProviderID(id string) (string, bool) {
	return registry.ResourceSlugForProviderID(id)
}

// IsLocalProvider reports whether the ID is an active local resource using the
// default catalogue.
func IsLocalProvider(id string) bool { return registry.IsLocalProvider(id) }

// SupportsPullModel reports pull_model support using the default catalogue.
func SupportsPullModel(id string) bool { return registry.SupportsPullModel(id) }

// ActionsForProvider returns lifecycle actions using the default catalogue.
func ActionsForProvider(id string) []Action { return registry.ActionsForProvider(id) }

// LocalProviderIDs returns the lifecycle-row provider IDs using the default
// catalogue.
func LocalProviderIDs() []string { return registry.LocalProviderIDs() }

// ProviderIDForEngine maps an STT engine ID using the default catalogue.
func ProviderIDForEngine(engineID string) (string, bool) {
	return registry.ProviderIDForEngine(engineID)
}

// EngineIDsForProvider returns a provider's engine IDs using the default
// catalogue.
func EngineIDsForProvider(providerID string) []string {
	return registry.EngineIDsForProvider(providerID)
}

// ProviderIDForRuntimeAlias maps a runtime identity using the default catalogue.
func ProviderIDForRuntimeAlias(raw string) (string, bool) {
	return registry.ProviderIDForRuntimeAlias(raw)
}

// ProviderIDForRuntimeIdentity resolves a canonical ID, engine ID, or runtime
// alias using the default catalogue.
func ProviderIDForRuntimeIdentity(raw string) (string, bool) {
	return registry.ProviderIDForRuntimeIdentity(raw)
}

// RuntimeAliasesForProvider returns a provider's runtime aliases using the
// default catalogue.
func RuntimeAliasesForProvider(providerID string) []string {
	return registry.RuntimeAliasesForProvider(providerID)
}

// ProbeFor returns a provider's readiness probe fact using the default
// catalogue.
func ProbeFor(providerID string) (Probe, bool) {
	return registry.ProbeFor(providerID)
}

// ConsentRequirementFor returns a provider's source-bound consent contract.
func ConsentRequirementFor(providerID string) (ConsentRequirement, bool) {
	return registry.ConsentRequirementFor(providerID)
}

// KnownFeatures returns every feature string the seed may use, sorted for
// deterministic generation.
func KnownFeatures() []string {
	out := make([]string, 0, len(featureCapabilities))
	for feature := range featureCapabilities {
		out = append(out, feature)
	}
	sort.Strings(out)
	return out
}
