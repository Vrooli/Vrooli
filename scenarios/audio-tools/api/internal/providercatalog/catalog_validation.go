package providercatalog

import (
	"fmt"
	"strings"
)

func (r *Registry) validate() error {
	if len(r.providers) == 0 {
		return fmt.Errorf("providercatalog: seed declares no providers")
	}
	state := validationState{make(map[string]struct{}, len(r.providers)), make(map[string]string), make(map[string]string), make(map[int]string)}
	for i := range r.providers {
		p := &r.providers[i]
		if err := validateProviderIdentity(p, i, state.seenIDs); err != nil {
			return err
		}
		if err := validateProviderFields(p); err != nil {
			return err
		}
		if err := validateProviderHealth(p); err != nil {
			return err
		}
		if err := validateProviderStatusAndVocabulary(p); err != nil {
			return err
		}
		if err := validateProviderActions(p); err != nil {
			return err
		}
		if err := validateProviderEngines(p, state.seenEngineIDs); err != nil {
			return err
		}
		if err := validateProviderAdapter(p, state.seenAdapters); err != nil {
			return err
		}
		if err := validateProviderTTSRuntime(p, state.seenTTSAutoPriorities); err != nil {
			return err
		}
		if err := validateProviderVoiceAliases(p); err != nil {
			return err
		}
		if err := validateProbe(p); err != nil {
			return err
		}
		if err := validateConsentRequirement(p); err != nil {
			return err
		}
	}
	if err := validateRuntimeAliases(r.providers, state.seenIDs, state.seenEngineIDs); err != nil {
		return err
	}
	for i := range r.providers {
		if err := validateBYOKAdapter(&r.providers[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateBYOKAdapter(p *Provider) error {
	if p.Status == StatusActive && p.Tier == TierBYOK {
		if strings.TrimSpace(p.BYOKAdapterID) == "" || p.BYOKAdapterID != strings.TrimSpace(p.BYOKAdapterID) {
			return fmt.Errorf("providercatalog: active BYOK provider %q must declare byok_adapter_id", p.ID)
		}
		if p.Kind != KindBYOKAPI && p.Kind != KindGateway {
			return fmt.Errorf("providercatalog: provider %q declares BYOK tier with non-credential kind %q", p.ID, p.Kind)
		}
	}
	if p.BYOKAdapterID != "" && (p.Tier != TierBYOK || (p.Kind != KindBYOKAPI && p.Kind != KindGateway)) {
		return fmt.Errorf("providercatalog: provider %q declares byok_adapter_id without BYOK credential routing", p.ID)
	}
	return nil
}

func validateProviderHealth(p *Provider) error {
	if p.HealthProfile != "" && !p.HealthProfile.Valid() {
		return fmt.Errorf("providercatalog: provider %q has unknown health_profile %q", p.ID, p.HealthProfile)
	}
	if p.HealthProfile == HealthProfileNone {
		if p.CheckerKind != CheckerNone {
			return fmt.Errorf("providercatalog: provider %q health_profile %q requires checker_kind none", p.ID, p.HealthProfile)
		}
		if p.ReadinessAggregate || p.LivenessAggregate {
			return fmt.Errorf("providercatalog: provider %q with health_profile %q cannot join health aggregates", p.ID, p.HealthProfile)
		}
	}
	if p.HealthProfile == "" && (p.ReadinessAggregate || p.LivenessAggregate) {
		return fmt.Errorf("providercatalog: provider %q declares health aggregate membership without health_profile", p.ID)
	}
	if p.HealthProfile == HealthProfileHTTP {
		if p.HealthProbe == nil {
			return fmt.Errorf("providercatalog: provider %q health_profile %q requires health_probe", p.ID, p.HealthProfile)
		}
		if !p.HealthProbe.BaseURLField.Valid() {
			return fmt.Errorf("providercatalog: provider %q health_probe has unsupported base_url_field %q", p.ID, p.HealthProbe.BaseURLField)
		}
		for name, path := range map[string]string{
			"readiness_path": p.HealthProbe.ReadinessPath,
			"liveness_path":  p.HealthProbe.LivenessPath,
		} {
			if !validHealthPath(path) {
				return fmt.Errorf("providercatalog: provider %q health_probe %s must be a clean absolute path", p.ID, name)
			}
		}
		if p.HealthProbe.AggregateLivenessPath != "" && !validHealthPath(p.HealthProbe.AggregateLivenessPath) {
			return fmt.Errorf("providercatalog: provider %q health_probe aggregate_liveness_path must be a clean absolute path", p.ID)
		}
	} else if p.HealthProbe != nil {
		return fmt.Errorf("providercatalog: provider %q declares health_probe without health_profile %q", p.ID, HealthProfileHTTP)
	}
	if p.CheckerKind == CheckerNone && p.Status == StatusActive && (p.Kind == KindLocalResource) && p.HealthProfile != HealthProfileNone {
		return fmt.Errorf("providercatalog: active local provider %q without a checker must declare health_profile %q", p.ID, HealthProfileNone)
	}
	if (p.ReadinessAggregate || p.LivenessAggregate) && p.Status != StatusActive {
		return fmt.Errorf("providercatalog: non-active provider %q must not join health aggregates", p.ID)
	}
	return nil
}

func validHealthPath(path string) bool {
	return strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") && !strings.Contains(path, "..") && !strings.ContainsAny(path, "?#\\") && strings.TrimSpace(path) == path
}

func validateConsentRequirement(p *Provider) error {
	requirement := p.ConsentRequirement
	if p.Kind == KindFullDuplex {
		if requirement == nil {
			return fmt.Errorf("providercatalog: full-duplex provider %q must declare consent_requirement", p.ID)
		}
		source := string(requirement.Source)
		if strings.TrimSpace(source) == "" || source != strings.TrimSpace(source) {
			return fmt.Errorf("providercatalog: full-duplex provider %q has blank or invalid consent source", p.ID)
		}
		if !requirement.RevalidateActiveSessions {
			return fmt.Errorf("providercatalog: full-duplex provider %q must revalidate consent for active sessions", p.ID)
		}
		return nil
	}
	if requirement != nil {
		return fmt.Errorf("providercatalog: provider %q kind %q must not declare consent_requirement", p.ID, p.Kind)
	}
	return nil
}

type validationState struct {
	seenIDs               map[string]struct{}
	seenEngineIDs         map[string]string
	seenAdapters          map[string]string
	seenTTSAutoPriorities map[int]string
}

func validateProviderIdentity(p *Provider, i int, seenIDs map[string]struct{}) error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("providercatalog: provider with empty id at index %d", i)
	}
	if _, dup := seenIDs[p.ID]; dup {
		return fmt.Errorf("providercatalog: duplicate provider id %q", p.ID)
	}
	seenIDs[p.ID] = struct{}{}
	return nil
}

func validateProviderFields(p *Provider) error {
	if err := validateProviderCoreFields(p); err != nil {
		return err
	}
	return validateProviderPlatformFields(p)
}

func validateProviderCoreFields(p *Provider) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("providercatalog: provider %q has no name", p.ID)
	}
	if strings.TrimSpace(p.Description) == "" {
		return fmt.Errorf("providercatalog: provider %q has no description", p.ID)
	}
	if !p.Kind.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid kind %q", p.ID, p.Kind)
	}
	if !p.Tier.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid tier %q", p.ID, p.Tier)
	}
	if want := tierForKind(p.Kind); p.Tier != want {
		return fmt.Errorf("providercatalog: provider %q kind %q requires tier %q, got %q", p.ID, p.Kind, want, p.Tier)
	}
	if !p.Status.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid status %q", p.ID, p.Status)
	}
	if !p.DependencyKind.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid dependency_kind %q", p.ID, p.DependencyKind)
	}
	if strings.TrimSpace(p.DependencySlug) == "" {
		return fmt.Errorf("providercatalog: provider %q has no dependency_slug", p.ID)
	}
	return nil
}

func validateProviderPlatformFields(p *Provider) error {
	if !p.Platform.Support.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid platform support %q", p.ID, p.Platform.Support)
	}
	if p.Platform.Support != PlatformSupported && strings.TrimSpace(p.Platform.Reason) == "" {
		return fmt.Errorf("providercatalog: provider %q platform %q requires a reason", p.ID, p.Platform.Support)
	}
	if !p.CheckerKind.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid checker_kind %q", p.ID, p.CheckerKind)
	}
	if !p.Criticality.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid criticality %q", p.ID, p.Criticality)
	}
	if !p.DefActionKind.valid() {
		return fmt.Errorf("providercatalog: provider %q has invalid def_action_kind %q", p.ID, p.DefActionKind)
	}
	if strings.TrimSpace(p.Presentation.LabelKey) == "" || strings.TrimSpace(p.Presentation.DescriptionKey) == "" {
		return fmt.Errorf("providercatalog: provider %q has incomplete presentation keys", p.ID)
	}
	return nil
}

func validateProviderStatusAndVocabulary(p *Provider) error {
	if p.Status == StatusActive {
		if len(p.Features) == 0 {
			return fmt.Errorf("providercatalog: active provider %q declares no features", p.ID)
		}
		if len(p.Capabilities) == 0 {
			return fmt.Errorf("providercatalog: active provider %q declares no capabilities", p.ID)
		}
		if p.Kind == KindLocalResource && strings.TrimSpace(p.ResourceSlug) == "" {
			return fmt.Errorf("providercatalog: active local provider %q has no resource_slug", p.ID)
		}
	} else {
		if p.CheckerKind != CheckerNone {
			return fmt.Errorf("providercatalog: %s provider %q must not be probed (checker_kind=%q)", p.Status, p.ID, p.CheckerKind)
		}
		if len(p.Actions) > 0 {
			return fmt.Errorf("providercatalog: %s provider %q must not advertise lifecycle actions", p.Status, p.ID)
		}
	}
	for _, capability := range p.Capabilities {
		if !capability.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid capability %q", p.ID, capability)
		}
	}
	for _, feature := range p.Features {
		if _, ok := featureCapabilities[feature]; !ok {
			return fmt.Errorf("providercatalog: provider %q declares unknown feature %q", p.ID, feature)
		}
	}
	return nil
}

func validateProviderActions(p *Provider) error {
	for _, action := range p.Actions {
		if !action.valid() {
			return fmt.Errorf("providercatalog: provider %q has invalid action %q", p.ID, action)
		}
		if p.Kind != KindLocalResource {
			return fmt.Errorf("providercatalog: provider %q kind %q must not advertise lifecycle actions", p.ID, p.Kind)
		}
	}
	return nil
}

func validateProviderEngines(p *Provider, seenEngineIDs map[string]string) error {
	for _, engineID := range p.EngineIDs {
		if strings.TrimSpace(engineID) == "" {
			return fmt.Errorf("providercatalog: provider %q has an empty engine_id", p.ID)
		}
		if !hasCapability(p, CapabilitySTT) || p.Kind != KindLocalResource {
			return fmt.Errorf("providercatalog: provider %q declares engine_id %q but is not a local_resource STT provider", p.ID, engineID)
		}
		if owner, dup := seenEngineIDs[engineID]; dup {
			return fmt.Errorf("providercatalog: engine_id %q is declared by both %q and %q", engineID, owner, p.ID)
		}
		seenEngineIDs[engineID] = p.ID
	}
	if p.Status == StatusActive && p.Kind == KindLocalResource && hasCapability(p, CapabilitySTT) && len(p.EngineIDs) == 0 {
		return fmt.Errorf("providercatalog: active local STT provider %q declares no engine_ids", p.ID)
	}
	return nil
}

func validateProviderAdapter(p *Provider, seenAdapters map[string]string) error {
	if p.AdapterID != "" {
		if !hasCapability(p, CapabilityTTS) {
			return fmt.Errorf("providercatalog: provider %q declares adapter_id %q but does not provide TTS", p.ID, p.AdapterID)
		}
		if owner, dup := seenAdapters[p.AdapterID]; dup {
			return fmt.Errorf("providercatalog: adapter_id %q is declared by both %q and %q", p.AdapterID, owner, p.ID)
		}
		seenAdapters[p.AdapterID] = p.ID
	}
	return nil
}

func validateProviderTTSRuntime(p *Provider, seenTTSAutoPriorities map[int]string) error {
	runtime := p.TTSRuntime
	if runtime == nil {
		return validateRequiredLocalTTSRuntime(p)
	}
	if !hasCapability(p, CapabilityTTS) || p.Kind != KindLocalResource || p.AdapterID == "" {
		return fmt.Errorf("providercatalog: provider %q declares tts_runtime without a local TTS adapter", p.ID)
	}
	if err := validateLocalTTSRuntime(p.ID, runtime); err != nil {
		return err
	}
	if runtime.SharedRuntime != nil {
		if err := validateSharedTTSRuntime(p.ID, runtime, runtime.SharedRuntime); err != nil {
			return err
		}
	}
	if runtime.AutoPriority > 0 {
		if owner, duplicate := seenTTSAutoPriorities[runtime.AutoPriority]; duplicate {
			return fmt.Errorf("providercatalog: TTS auto_priority %d is shared by %q and %q", runtime.AutoPriority, owner, p.ID)
		}
		seenTTSAutoPriorities[runtime.AutoPriority] = p.ID
	}
	return nil
}

func validateRequiredLocalTTSRuntime(p *Provider) error {
	if p.Status == StatusActive && p.Kind == KindLocalResource && hasCapability(p, CapabilityTTS) {
		return fmt.Errorf("providercatalog: active local TTS provider %q declares no tts_runtime", p.ID)
	}
	return nil
}

func validateLocalTTSRuntime(providerID string, runtime *TTSRuntime) error {
	if strings.TrimSpace(runtime.BackendID) == "" || strings.TrimSpace(runtime.URLField) == "" || strings.TrimSpace(runtime.HealthPath) == "" || !strings.HasPrefix(runtime.HealthPath, "/") || strings.TrimSpace(runtime.ModelRole) == "" || strings.TrimSpace(runtime.CacheBackend) == "" {
		return fmt.Errorf("providercatalog: provider %q has incomplete tts_runtime", providerID)
	}
	if runtime.URLField != "KokoroURL" && runtime.URLField != "KaleURL" {
		return fmt.Errorf("providercatalog: provider %q has unsupported tts_runtime url_field %q", providerID, runtime.URLField)
	}
	return validateTTSSpeedMode(providerID, runtime.SpeedMode)
}

func validateSharedTTSRuntime(providerID string, runtime *TTSRuntime, shared *TTSSharedRuntime) error {
	if strings.TrimSpace(shared.BackendID) == "" || shared.BackendID == runtime.BackendID || strings.TrimSpace(shared.URLField) == "" || strings.TrimSpace(shared.HealthPath) == "" || !strings.HasPrefix(shared.HealthPath, "/") || strings.TrimSpace(shared.CachePrefix) == "" || strings.TrimSpace(shared.ModelRole) == "" {
		return fmt.Errorf("providercatalog: provider %q has incomplete shared TTS runtime", providerID)
	}
	if shared.URLField != "SherpaURL" {
		return fmt.Errorf("providercatalog: provider %q has unsupported shared TTS url_field %q", providerID, shared.URLField)
	}
	return validateTTSSpeedMode(providerID, shared.SpeedMode)
}

func validateTTSSpeedMode(providerID, mode string) error {
	if mode == "" {
		return fmt.Errorf("providercatalog: provider %q has incomplete TTS runtime: speed_mode is required", providerID)
	}
	if mode == "native" || mode == "postprocess" {
		return nil
	}
	return fmt.Errorf("providercatalog: provider %q has unsupported TTS speed_mode %q", providerID, mode)
}

func validateProviderVoiceAliases(p *Provider) error {
	if len(p.VoiceAliases) > 0 && p.AdapterID == "" {
		return fmt.Errorf("providercatalog: provider %q declares voice_aliases without adapter_id", p.ID)
	}
	for canonical, backend := range p.VoiceAliases {
		if !strings.HasPrefix(canonical, "voice.") || strings.TrimSpace(canonical) != canonical || strings.TrimSpace(backend) == "" {
			return fmt.Errorf("providercatalog: provider %q has invalid voice_alias %q=%q", p.ID, canonical, backend)
		}
	}
	return nil
}

func validateRuntimeAliases(providers []Provider, seenIDs map[string]struct{}, seenEngineIDs map[string]string) error {
	seenAliases := make(map[string]string)
	for i := range providers {
		p := &providers[i]
		for _, alias := range p.RuntimeAliases {
			if alias == "" || alias != strings.TrimSpace(alias) {
				return fmt.Errorf("providercatalog: provider %q has a blank runtime_alias", p.ID)
			}
			if alias == p.ID {
				return fmt.Errorf("providercatalog: provider %q declares its own id as a runtime_alias", p.ID)
			}
			if owner, dup := seenIDs[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by %q but collides with provider id %q", alias, p.ID, owner)
			}
			if owner, dup := seenEngineIDs[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by %q but collides with an engine_id of %q", alias, p.ID, owner)
			}
			if owner, dup := seenAliases[alias]; dup {
				return fmt.Errorf("providercatalog: runtime_alias %q is declared by both %q and %q", alias, owner, p.ID)
			}
			seenAliases[alias] = p.ID
		}
	}
	return nil
}

// validateProbe keeps the readiness-probe fact bound to remote providers and
// requires an active remote provider to state either the probe or why it has
// none. The URL must be https because the probe sends a credential in Header.
func validateProbe(p *Provider) error {
	remote := p.Kind == KindBYOKAPI || p.Kind == KindGateway || p.Kind == KindFullDuplex
	if p.Probe != nil {
		if err := validateDeclaredProbe(p, remote); err != nil {
			return err
		}
	}
	if err := validateProbeAbsenceReason(p, remote); err != nil {
		return err
	}
	if p.Status == StatusActive && remote && p.Probe == nil && p.ProbeAbsentReason == "" {
		return fmt.Errorf("providercatalog: active remote provider %q must declare a probe or a probe_absent_reason", p.ID)
	}
	return nil
}

func validateDeclaredProbe(p *Provider, remote bool) error {
	if !remote {
		return fmt.Errorf("providercatalog: provider %q kind %q is not remote and must not declare a probe", p.ID, p.Kind)
	}
	if p.Tier == TierBYOK && len(p.Capabilities) != 1 {
		return fmt.Errorf("providercatalog: BYOK provider %q with a credential probe must declare exactly one capability for provider-level health", p.ID)
	}
	if !strings.HasPrefix(p.Probe.URL, "https://") {
		return fmt.Errorf("providercatalog: provider %q probe url %q must be https", p.ID, p.Probe.URL)
	}
	if strings.TrimSpace(p.Probe.Header) == "" {
		return fmt.Errorf("providercatalog: provider %q probe has no header", p.ID)
	}
	if p.ProbeAbsentReason != "" {
		return fmt.Errorf("providercatalog: provider %q declares both a probe and a probe_absent_reason", p.ID)
	}
	return nil
}

func validateProbeAbsenceReason(p *Provider, remote bool) error {
	if p.ProbeAbsentReason == "" {
		return nil
	}
	if !remote {
		return fmt.Errorf("providercatalog: provider %q kind %q is not remote and must not declare a probe_absent_reason", p.ID, p.Kind)
	}
	if strings.TrimSpace(p.ProbeAbsentReason) == "" {
		return fmt.Errorf("providercatalog: provider %q has a blank probe_absent_reason", p.ID)
	}
	return nil
}

func tierForKind(kind Kind) Tier {
	switch kind {
	case KindLocalResource, KindLocalInProc:
		return TierLocal
	case KindBYOKAPI, KindFullDuplex:
		return TierBYOK
	case KindGateway:
		return TierVrooli
	case KindBrowser:
		return TierBrowser
	default:
		return TierNone
	}
}

func hasCapability(p *Provider, capability Capability) bool {
	for _, declared := range p.Capabilities {
		if declared == capability {
			return true
		}
	}
	return false
}

// featureCapabilities is the closed feature vocabulary. Every feature a
// provider may declare must appear here, and every entry maps to a known
// capability so the generated capabilities helper can never emit an
// unspecified row for a declared feature.
//
// Some scenario features (tts-cache, tts-paragraph-split, audio-provider-routing)
// are grouped with the capability they serve rather than treated as separate
// capabilities. The audio-tools rollup is excluded from serviceability grouping
// by ID, so these groupings do not add or remove a health row.
var featureCapabilities = map[string]Capability{
	"voice-input":                CapabilitySTT,
	"voice-streaming":            CapabilitySTT,
	"voice-streaming-realtime":   CapabilitySTT,
	"voice-speaker-verification": CapabilitySpeaker,
	"voice-enrollment":           CapabilitySpeaker,
	"voice-output":               CapabilityTTS,
	"voice-cloning":              CapabilityTTS,
	"tts-summarization":          CapabilitySummarize,
	"tts-cache":                  CapabilityTTS,
	"tts-paragraph-split":        CapabilityTTS,
	"audio-provider-routing":     CapabilitySTT,
	"ai-command-generation":      CapabilitySummarize,
	"transcode":                  CapabilityTranscode,
}
