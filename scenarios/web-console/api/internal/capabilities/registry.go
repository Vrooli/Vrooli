// Package capabilities supplies web-console's concrete capability catalogue
// and checker wiring. The registry contract and state machine live in the
// shared capability-registry-go package so API and UI surfaces cannot drift.
package capabilities

import (
	"runtime"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	common_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	"github.com/vrooli/vrooli/scenarios/audio-tools/clients/go/audiotools"
)

type (
	DependencyKind = capabilityregistry.DependencyKind
	Status         = capabilityregistry.Status
	ActionKind     = capabilityregistry.ActionKind
	Def            = capabilityregistry.Def
	State          = capabilityregistry.State
	Checker        = capabilityregistry.Checker
	ResultChecker  = capabilityregistry.ResultChecker
	CheckResult    = capabilityregistry.CheckResult
	Registry       = capabilityregistry.Registry
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
	PlatformUnsupported       = capabilityregistry.PlatformUnsupported
)

var knownCatalogue = []Def{
	{
		ID: audiotools.CapabilitySlug, Name: "Audio Tools",
		Description:    "Shared audio capability scenario: STT, TTS, summarization, provider routing, BYOK/LPBS/local tiers, adoptable UI",
		DependencyKind: DependencyScenario, DependencySlug: audiotools.CapabilitySlug,
		ActionKind: ActionKindScenarioStart, ActionLabel: "Start Audio Tools", OperatorCommand: "vrooli scenario start audio-tools --json",
		Features: []string{
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_INPUT),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_STREAMING),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_OUTPUT),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_CACHE),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_PARAGRAPH_SPLIT),
			audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_AUDIO_PROVIDER_ROUTING),
		},
		Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional audio capability depends on the selected provider and host media path"},
	},
	{ID: "ollama", Name: "Ollama", Description: "Local model runtime used by optional AI features", DependencyKind: DependencyResource, DependencySlug: "ollama"},
	{ID: "openrouter", Name: "OpenRouter", Description: "Hosted model provider used by optional AI features", DependencyKind: DependencyResource, DependencySlug: "openrouter"},
	{ID: "session-backend-standard", Name: "Standard Terminal Sessions", Description: "Local PTY terminal sessions", DependencyKind: DependencyResource, DependencySlug: "session-backend-standard"},
	{ID: "session-backend-persistent", Name: "Persistent Terminal Sessions", Description: "tmux-backed terminal sessions that survive API restarts", DependencyKind: DependencyResource, DependencySlug: "session-backend-persistent"},
	{ID: "vrooli-bridge", Name: "Remote Terminals", Description: "Bridged terminal sessions on registered nodes", DependencyKind: DependencyScenario, DependencySlug: "vrooli-bridge", ActionKind: ActionKindScenarioStart, ActionLabel: "Start Bridge", OperatorCommand: "vrooli scenario start vrooli-bridge --json"},
}

// Known is the catalogue for the current host. Platform-specific backend
// verdicts are applied at construction time so the same registry contract is
// honest on Windows without making Linux callers pretend tmux exists there.
var Known = KnownForPlatform(runtime.GOOS)

func KnownForPlatform(goos string) []Def {
	defs := make([]Def, len(knownCatalogue))
	copy(defs, knownCatalogue)
	if goos != "windows" {
		return defs
	}
	for i := range defs {
		switch defs[i].ID {
		case "session-backend-persistent":
			defs[i].Platform = capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformUnsupported, Reason: "tmux is not available on this platform"}
		}
	}
	return defs
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}
