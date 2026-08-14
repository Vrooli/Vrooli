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
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}

func NewRegistryWithClock(defs []Def, checkers map[string]Checker, cacheTTL time.Duration, clk schedule.Clock) *Registry {
	return capabilityregistry.NewWithClock(defs, checkers, cacheTTL, func() time.Time { return clk.Now() })
}
