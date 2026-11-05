// features.go — canonical capability / feature identifiers for audio-tools.
//
// Consumer scenarios (web-console, swarm-manager, ...) register a
// dependency-capability under CapabilitySlug and list the feature slugs
// they wire into their UI. The slugs are the wire-level identifiers
// carried on web-console's CapabilitiesService `repeated string features`
// surface; they must agree byte-for-byte across the API registry and
// every UI consumer.
//
// To prevent drift, callers MUST use:
//
//	audiotools.CapabilitySlug                              // "audio-tools"
//	audiotools.FeatureSlug(common_v1.AudioToolsFeature_*)  // "voice-input", ...
//	audiotools.AllFeatureSlugs()                           // every registered slug
//
// Adding a new feature: add the enum value in
// packages/proto/schemas/audio-tools/v1/common/common.proto, run
// `make generate`, then add the slug entry in `featureSlugs` below AND
// in audio-tools/ui/src/audio-integration/features.ts.

package audiotools

import (
	"fmt"
	"sort"

	common_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// CapabilitySlug is the scenario-level capability id used when registering
// audio-tools as a dependency capability. Matches the scenario slug.
const CapabilitySlug = "audio-tools"

// featureSlugs is the single source of truth for the AudioToolsFeature
// enum → wire-slug mapping. Keys MUST cover every value of the enum
// except _UNSPECIFIED; a missing key triggers a panic at startup via
// the init() check below.
var featureSlugs = map[common_v1.AudioToolsFeature]string{
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_INPUT:                "voice-input",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_STREAMING:            "voice-streaming",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_SPEAKER_VERIFICATION: "voice-speaker-verification",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_ENROLLMENT:           "voice-enrollment",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_VOICE_OUTPUT:               "voice-output",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_SUMMARIZATION:          "tts-summarization",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_CACHE:                  "tts-cache",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_TTS_PARAGRAPH_SPLIT:        "tts-paragraph-split",
	common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_AUDIO_PROVIDER_ROUTING:     "audio-provider-routing",
}

func init() {
	// Fail-fast if a generated enum value lacks a slug entry. This catches
	// the case where someone adds an enum value, regenerates, and forgets
	// to update featureSlugs (or features.ts).
	for v, name := range common_v1.AudioToolsFeature_name {
		feat := common_v1.AudioToolsFeature(v)
		if feat == common_v1.AudioToolsFeature_AUDIO_TOOLS_FEATURE_UNSPECIFIED {
			continue
		}
		if _, ok := featureSlugs[feat]; !ok {
			panic(fmt.Sprintf("audiotools: missing slug for AudioToolsFeature value %s (%d)", name, v))
		}
	}
}

// FeatureSlug returns the wire-slug for an AudioToolsFeature enum value.
// Returns the empty string for _UNSPECIFIED or any value not present in
// the mapping table.
func FeatureSlug(f common_v1.AudioToolsFeature) string {
	return featureSlugs[f]
}

// AllFeatureSlugs returns every registered feature slug in stable
// (alphabetical) order. Useful for consumer scenarios that want to
// register the full audio-tools feature surface.
func AllFeatureSlugs() []string {
	out := make([]string, 0, len(featureSlugs))
	for _, s := range featureSlugs {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
