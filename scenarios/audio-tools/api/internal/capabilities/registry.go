// Package capabilities provides audio-tools' scenario-specific catalogue and
// checker wiring over the shared capability-registry-go state machine.
package capabilities

import (
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// CapabilityServiceability is the scenario-facing projection of provider
// states. It keeps provider evidence while exposing the shared any-available
// rule used by both health surfaces.
type CapabilityServiceability struct {
	Capability           diagv1.Capability
	Providers            []State
	Serviceable          bool
	UnavailableProviders []string
}

// Serviceability groups catalogue states by their declared feature mapping.
func Serviceability(states []State) []CapabilityServiceability {
	groups := capabilityregistry.RollupByCapability(states, func(state State) []string {
		if state.ID == "audio-tools" {
			return nil
		}
		var names []string
		for _, feature := range state.Features {
			if capability, ok := CapabilityForFeature(feature); ok {
				names = append(names, capability.String())
			}
		}
		return names
	})
	out := make([]CapabilityServiceability, 0, len(groups))
	for _, group := range groups {
		capabilityValue, ok := diagv1.Capability_value[group.Name]
		if !ok {
			continue
		}
		capability := diagv1.Capability(capabilityValue)
		out = append(out, CapabilityServiceability{
			Capability:           capability,
			Providers:            group.Providers,
			Serviceable:          group.Serviceable,
			UnavailableProviders: group.UnavailableProviders,
		})
	}
	return out
}

// RequiredCapability reports whether the audio service must have at least
// one provider for this capability.
func RequiredCapability(capability diagv1.Capability) bool {
	return capability == diagv1.Capability_CAPABILITY_STT ||
		capability == diagv1.Capability_CAPABILITY_TTS ||
		capability == diagv1.Capability_CAPABILITY_TRANSCODE
}

// RequiredFailures returns actionable descriptions for required capabilities
// without an available provider.
func RequiredFailures(states []State) []string {
	var failures []string
	for _, group := range Serviceability(states) {
		if RequiredCapability(group.Capability) && !group.Serviceable {
			name := group.Capability.String()
			if len(group.UnavailableProviders) > 0 {
				name += " (unavailable: " + strings.Join(group.UnavailableProviders, ", ") + ")"
			}
			failures = append(failures, name)
		}
	}
	return failures
}

type (
	DependencyKind  = capabilityregistry.DependencyKind
	Status          = capabilityregistry.Status
	PlatformSupport = capabilityregistry.PlatformSupport
	PlatformVerdict = capabilityregistry.PlatformVerdict
	Criticality     = capabilityregistry.Criticality
	Def             = capabilityregistry.Def
	State           = capabilityregistry.State
	Checker         = capabilityregistry.Checker
	ResultChecker   = capabilityregistry.ResultChecker
	CheckResult     = capabilityregistry.CheckResult
	Registry        = capabilityregistry.Registry
)

const (
	DependencyScenario        = capabilityregistry.DependencyScenario
	DependencyResource        = capabilityregistry.DependencyResource
	StatusAvailable           = capabilityregistry.StatusAvailable
	StatusUnavailable         = capabilityregistry.StatusUnavailable
	StatusUnknown             = capabilityregistry.StatusUnknown
	ActionKindNone            = capabilityregistry.ActionKindNone
	ActionKindOperatorCommand = capabilityregistry.ActionKindOperatorCommand
	ActionKindScenarioStart   = capabilityregistry.ActionKindScenarioStart
	ActionKindScenarioRestart = capabilityregistry.ActionKindScenarioRestart
	ActionKindOwnerGuidance   = capabilityregistry.ActionKindOwnerGuidance
	PlatformSupported         = capabilityregistry.PlatformSupported
	PlatformDegraded          = capabilityregistry.PlatformDegraded
	PlatformUnsupported       = capabilityregistry.PlatformUnsupported
	CriticalityRequired       = capabilityregistry.CriticalityRequired
	CriticalityOptional       = capabilityregistry.CriticalityOptional
)

var Known = []Def{
	{ID: "audio-tools", Name: "Audio Tools", Description: "Shared audio capability scenario: STT, TTS, summarization, provider routing, BYOK/LPBS/local tiers, adoptable UI", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-streaming", "voice-speaker-verification", "voice-enrollment", "voice-output", "tts-summarization", "tts-cache", "tts-paragraph-split", "audio-provider-routing"}, Criticality: CriticalityOptional, ActionKind: ActionKindScenarioStart, OperatorCommand: "vrooli scenario start audio-tools", Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "audio capability depends on the selected provider and host media path"}},
	{ID: "whisper-stt", Name: "Whisper STT", Description: "Local batch speech-to-text provider", DependencyKind: DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input"}, Criticality: CriticalityOptional, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start whisper"},
	{ID: "kyutai-stt", Name: "Kyutai STT", Description: "Local streaming speech-to-text provider", DependencyKind: DependencyResource, DependencySlug: "kyutai-stt", Features: []string{"voice-streaming"}, Criticality: CriticalityOptional, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start kyutai-stt"},
	{ID: "kokoro-tts", Name: "Kokoro TTS", Description: "Local text-to-speech provider", DependencyKind: DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}, Criticality: CriticalityOptional, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start kokoro"},
	{ID: "speaker-verification", Name: "Speaker Verification", Description: "Local speaker enrollment and verification provider", DependencyKind: DependencyResource, DependencySlug: "sherpa-onnx", Features: []string{"voice-speaker-verification", "voice-enrollment"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "vrooli resource status sherpa-onnx --json"},
	{ID: "ollama", Name: "Ollama Summarize", Description: "Local language-model summarization provider", DependencyKind: DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}, Criticality: CriticalityOptional, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start ollama"},
	{ID: "openrouter", Name: "OpenRouter", Description: "BYOK cloud language-model provider", DependencyKind: DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "openai-whisper", Name: "OpenAI Whisper (BYOK)", Description: "BYOK cloud speech-to-text provider", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "deepgram", Name: "Deepgram (BYOK)", Description: "BYOK cloud streaming speech-to-text provider", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-streaming"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "openai-tts", Name: "OpenAI TTS (BYOK)", Description: "BYOK cloud text-to-speech provider", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-output"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "elevenlabs", Name: "ElevenLabs (BYOK)", Description: "BYOK cloud text-to-speech provider", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-output"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "browser-stt", Name: "Browser Speech Input", Description: "Browser Web Speech API last-resort speech-to-text; server transcript quality stages do not apply", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-streaming"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "browser-tts", Name: "Browser Speech Output", Description: "Browser speech synthesis last-resort text-to-speech; voice quality is browser-dependent", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-output"}, Criticality: CriticalityOptional, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "audio-transcode", Name: "Audio Transcode", Description: "In-process audio format conversion capability", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"transcode"}, Criticality: CriticalityRequired, ActionKind: ActionKindOperatorCommand, OperatorCommand: "audio-tools health show"},
}

// KnownForPlatform returns the catalogue with resource-backed platform
// verdicts applied. Keeping this construction in the catalogue makes the
// platform contract testable without relying on the process host's GOOS and
// ensures unsupported local providers are marked by design before probing.
func KnownForPlatform(goos string) []Def {
	defs := append([]Def(nil), Known...)
	resolver := NewResourcePlatformResolver(ResourcesFS(), goos)
	for i := range defs {
		if defs[i].DependencyKind == DependencyResource {
			defs[i].Platform = resolver.Resolve(defs[i].DependencySlug)
		}
	}
	return defs
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}

func NewRegistryWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, clk schedule.Clock) *Registry {
	return capabilityregistry.NewWithClock(defs, checkers, cacheTTL, func() time.Time { return clk.Now() })
}
