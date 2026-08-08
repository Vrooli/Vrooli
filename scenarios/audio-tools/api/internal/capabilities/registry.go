// Package capabilities provides audio-tools' scenario-specific catalogue and
// checker wiring over the shared capability-registry-go state machine.
package capabilities

import (
	"time"

	"audio-tools/internal/clock"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type DependencyKind = capabilityregistry.DependencyKind
type Status = capabilityregistry.Status
type PlatformSupport = capabilityregistry.PlatformSupport
type PlatformVerdict = capabilityregistry.PlatformVerdict
type Def = capabilityregistry.Def
type State = capabilityregistry.State
type Checker = capabilityregistry.Checker
type ResultChecker = capabilityregistry.ResultChecker
type CheckResult = capabilityregistry.CheckResult
type Registry = capabilityregistry.Registry

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
)

var Known = []Def{
	{ID: "whisper-stt", Name: "Whisper STT", Description: "Speech-to-text transcription via Whisper", DependencyKind: DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input", "voice-streaming"}},
	{ID: "kyutai-stt", Name: "Kyutai Streaming STT", Description: "Real-time speech-to-text via Kyutai", DependencyKind: DependencyResource, DependencySlug: "kyutai-stt", Features: []string{"voice-input", "voice-streaming-realtime"}},
	{ID: "speaker-verification", Name: "Speaker Verification", Description: "Local speaker verification for enrolled voice filtering", DependencyKind: DependencyResource, DependencySlug: "speaker-verification", Features: []string{"voice-speaker-verification", "voice-enrollment"}},
	{ID: "kokoro-tts", Name: "Kokoro TTS", Description: "Text-to-speech synthesis via Kokoro", DependencyKind: DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}},
	{ID: "ollama", Name: "Ollama", Description: "Local LLM inference for AI command generation", DependencyKind: DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}},
	{ID: "openrouter", Name: "OpenRouter", Description: "Cloud LLM API for AI command generation", DependencyKind: DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}},
	{ID: "audio-tools", Name: "Audio Tools", Description: "Shared audio capability scenario: STT, TTS, summarization, provider routing, BYOK/LPBS/local tiers, adoptable UI", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-streaming", "voice-speaker-verification", "voice-enrollment", "voice-output", "tts-summarization", "tts-cache", "tts-paragraph-split", "audio-provider-routing"}, ActionKind: ActionKindScenarioStart, OperatorCommand: "vrooli scenario start audio-tools"},
	{ID: "landing-page-business-suite", Name: "Landing Page Business Suite", Description: "Optional LPBS audio routing tier", DependencyKind: DependencyScenario, DependencySlug: "landing-page-business-suite", Features: []string{"audio-provider-routing"}, ActionKind: ActionKindScenarioStart, OperatorCommand: "vrooli scenario start landing-page-business-suite"},
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}

func NewRegistryWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, clk clock.Clock) *Registry {
	return capabilityregistry.NewWithClock(defs, checkers, cacheTTL, func() time.Time { return clk.Now() })
}
