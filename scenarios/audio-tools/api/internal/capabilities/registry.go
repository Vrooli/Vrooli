// Package capabilities provides audio-tools' scenario-specific catalogue and
// checker wiring over the shared capability-registry-go state machine.
package capabilities

import (
	"time"

	"github.com/vrooli/api-core/schedule"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type (
	DependencyKind  = capabilityregistry.DependencyKind
	Status          = capabilityregistry.Status
	PlatformSupport = capabilityregistry.PlatformSupport
	PlatformVerdict = capabilityregistry.PlatformVerdict
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
)

var Known = []Def{
	{ID: "audio-tools", Name: "Audio Tools", Description: "Shared audio capability scenario: STT, TTS, summarization, provider routing, BYOK/LPBS/local tiers, adoptable UI", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"voice-input", "voice-streaming", "voice-speaker-verification", "voice-enrollment", "voice-output", "tts-summarization", "tts-cache", "tts-paragraph-split", "audio-provider-routing"}, ActionKind: ActionKindScenarioStart, OperatorCommand: "vrooli scenario start audio-tools", Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "audio capability depends on the selected provider and host media path"}},
	{ID: "whisper-stt", Name: "Whisper STT", Description: "Local batch speech-to-text provider", DependencyKind: DependencyResource, DependencySlug: "whisper", Features: []string{"voice-input"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start whisper"},
	{ID: "kyutai-stt", Name: "Kyutai STT", Description: "Local streaming speech-to-text provider", DependencyKind: DependencyResource, DependencySlug: "kyutai-stt", Features: []string{"voice-streaming"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start kyutai-stt"},
	{ID: "kokoro-tts", Name: "Kokoro TTS", Description: "Local text-to-speech provider", DependencyKind: DependencyResource, DependencySlug: "kokoro", Features: []string{"voice-output"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start kokoro"},
	{ID: "speaker-verification", Name: "Speaker Verification", Description: "Local speaker enrollment and verification provider", DependencyKind: DependencyResource, DependencySlug: "speaker-verification", Features: []string{"voice-speaker-verification", "voice-enrollment"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start speaker-verification"},
	{ID: "ollama", Name: "Ollama Summarize", Description: "Local language-model summarization provider", DependencyKind: DependencyResource, DependencySlug: "ollama", Features: []string{"ai-command-generation"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "vrooli resource start ollama"},
	{ID: "openrouter", Name: "OpenRouter", Description: "BYOK cloud language-model provider", DependencyKind: DependencyResource, DependencySlug: "openrouter", Features: []string{"ai-command-generation"}, ActionKind: ActionKindOwnerGuidance, OperatorCommand: "audio-tools settings providers"},
	{ID: "audio-transcode", Name: "Audio Transcode", Description: "In-process audio format conversion capability", DependencyKind: DependencyScenario, DependencySlug: "audio-tools", Features: []string{"transcode"}, ActionKind: ActionKindOperatorCommand, OperatorCommand: "audio-tools health show"},
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}

func NewRegistryWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, clk schedule.Clock) *Registry {
	return capabilityregistry.NewWithClock(defs, checkers, cacheTTL, func() time.Time { return clk.Now() })
}
