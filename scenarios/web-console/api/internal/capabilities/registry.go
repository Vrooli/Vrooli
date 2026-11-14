// Package capabilities supplies web-console's concrete capability catalogue
// and checker wiring. The registry contract and state machine live in the
// shared capability-registry-go package so API and UI surfaces cannot drift.
package capabilities

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vrooli/cli-core/agentcatalog/harness"
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

var virtualCatalogue = []Def{
	{ID: "session-backend-standard", Name: "Standard Terminal Sessions", Description: "Local PTY terminal sessions", Uses: []capabilityregistry.Use{{ID: "terminal-local", Name: "Standard terminals", Description: "Open a terminal on this machine", Group: "terminals"}}, DependencyKind: DependencyResource, DependencySlug: "session-backend-standard"},
	{ID: "session-backend-persistent", Name: "Persistent Terminal Sessions", Description: "tmux-backed terminal sessions that survive API restarts", Uses: []capabilityregistry.Use{{ID: "terminal-persistent", Name: "Persistent terminals", Description: "Keep terminal sessions running when the app restarts", Group: "terminals"}}, DependencyKind: DependencyResource, DependencySlug: "session-backend-persistent"},
}

var audioFeatures = []string{
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_INPUT),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_STREAMING),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_OUTPUT),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_CACHE),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_PARAGRAPH_SPLIT),
	audiotools.FeatureSlug(common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_AUDIO_PROVIDER_ROUTING),
}

// Known is the catalogue for the current host. Platform-specific backend
// verdicts are applied at construction time so the same registry contract is
// honest on Windows without making Linux callers pretend tmux exists there.
var Known = KnownForPlatform(runtime.GOOS)

func KnownForPlatform(goos string) []Def {
	path := os.Getenv("WEB_CONSOLE_SERVICE_MANIFEST")
	if path == "" {
		path = findServiceManifest()
	}
	overlays := map[string]capabilityregistry.Overlay{
		"audio-tools":   {ID: "audio-tools", Features: append([]string(nil), audioFeatures...), Platform: capabilityregistry.PlatformVerdict{Support: capabilityregistry.PlatformDegraded, Reason: "optional audio capability depends on the selected provider and host media path"}, ActionKind: ActionKindScenarioStart, ActionLabel: "Start Audio Tools", OperatorCommand: "vrooli scenario start audio-tools --json"},
		"vrooli-bridge": {ID: "vrooli-bridge", Name: "Remote Terminals", Features: []string{"remote_terminal"}, ActionKind: ActionKindScenarioStart, ActionLabel: "Start Bridge", OperatorCommand: "vrooli scenario start vrooli-bridge --json"},
	}
	// An overlay must name a manifest dependency, so a harness this service does
	// not declare gets none rather than failing the whole catalogue.
	declared, _ := capabilityregistry.ProjectManifest(path, nil)
	for _, def := range declared {
		descriptor, ok := harness.Active().Lookup(def.DependencySlug)
		if !ok || def.DependencyKind != DependencyResource {
			continue
		}
		overlay := capabilityregistry.Overlay{ID: descriptor.Resource, Name: descriptor.DisplayName, ActionKind: ActionKindOperatorCommand, ActionLabel: "Install " + descriptor.DisplayName, OperatorCommand: "vrooli resource install " + descriptor.Resource + " --json"}
		if descriptor.ID != descriptor.Resource {
			overlay.IntegrationID = descriptor.ID
		}
		overlays[descriptor.Resource] = overlay
	}
	defs, _ := capabilityregistry.ProjectManifest(path, overlays)
	if os.Getenv("VROOLI_DESKTOP_MODE") == "true" {
		// Desktop harness actions use the native lifecycle owner in Manage.
		// A bundle cannot start repository scenarios or run host setup commands.
		for i := range defs {
			defs[i].ActionKind = ActionKindNone
			defs[i].ActionLabel = ""
			defs[i].OperatorCommand = ""
		}
	}
	defs = append(defs, cloneDefs(virtualCatalogue)...)
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

func findServiceManifest() string {
	if bundle := os.Getenv("BUNDLE_ROOT"); os.Getenv("VROOLI_DESKTOP_MODE") == "true" && bundle != "" {
		return filepath.Join(bundle, "catalog", "scenarios", "web-console", ".vrooli", "service.json")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join("..", ".vrooli", "service.json")
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, ".vrooli", "service.json")
		if filepath.Base(dir) == "web-console" {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Join("..", ".vrooli", "service.json")
		}
	}
}

func cloneDefs(defs []Def) []Def {
	out := make([]Def, len(defs))
	copy(out, defs)
	for i := range out {
		out[i].Features = append([]string(nil), defs[i].Features...)
	}
	return out
}

func NewRegistry(defs []Def, checkers map[string]Checker, cacheTTL time.Duration) *Registry {
	return capabilityregistry.New(defs, checkers, cacheTTL)
}
