// features.ts — canonical capability / feature identifiers for audio-tools.
//
// Mirror of clients/go/audiotools/features.go. Consumer scenarios
// register a dependency-capability under AUDIO_TOOLS_CAPABILITY_SLUG and
// list the feature slugs they wire into their UI. The slugs are the
// wire-level identifiers carried on web-console's CapabilitiesService
// `repeated string features` surface; they must agree byte-for-byte
// across the API registry and every UI consumer.
//
// Drift safety: the slug map is keyed by the generated AudioToolsFeature
// proto enum. Adding a new value to the enum without updating the map
// triggers a runtime throw at module load (see assertCoverage below) so
// the bug is caught before any UI render path uses a stale slug.
//
// Adding a new feature: add the enum value in
// packages/proto/schemas/audio-tools/v1/common/common.proto, run
// `make generate`, then add the slug entry below AND in
// clients/go/audiotools/features.go.
//
// IMPORTANT: this file lives verbatim in audio-tools/ui (canonical) and
// is copied into each consumer scenario. Update the source here, then
// copy to web-console/ui and swarm-manager/ui.

import { AudioToolsFeature } from "@vrooli/proto-types/audio-tools/v1/common/common_pb";

/** Scenario-level capability id used when registering audio-tools as a dependency-capability. Matches the scenario slug. */
export const AUDIO_TOOLS_CAPABILITY_SLUG = "audio-tools";

const FEATURE_SLUGS: Record<AudioToolsFeature, string> = {
  [AudioToolsFeature.UNSPECIFIED]: "",
  [AudioToolsFeature.VOICE_INPUT]: "voice-input",
  [AudioToolsFeature.VOICE_STREAMING]: "voice-streaming",
  [AudioToolsFeature.VOICE_SPEAKER_VERIFICATION]: "voice-speaker-verification",
  [AudioToolsFeature.VOICE_ENROLLMENT]: "voice-enrollment",
  [AudioToolsFeature.VOICE_OUTPUT]: "voice-output",
  [AudioToolsFeature.TTS_SUMMARIZATION]: "tts-summarization",
  [AudioToolsFeature.TTS_CACHE]: "tts-cache",
  [AudioToolsFeature.TTS_PARAGRAPH_SPLIT]: "tts-paragraph-split",
  [AudioToolsFeature.AUDIO_PROVIDER_ROUTING]: "audio-provider-routing",
};

// The numeric members of the AudioToolsFeature enum. Object.values of a
// numeric enum yields both the reverse-mapped names (strings) and the
// numeric values; the type guard keeps only the latter as enum values.
const FEATURE_VALUES = Object.values(AudioToolsFeature).filter(
  (v): v is AudioToolsFeature => typeof v === "number",
);

(function assertCoverage() {
  // FEATURE_SLUGS is typed Record<AudioToolsFeature, string>, so the compiler
  // already enforces an entry per enum member; this runtime guard catches a
  // hand-edited empty slug and points at the Go mirror that must also change.
  for (const value of FEATURE_VALUES) {
    if (value === AudioToolsFeature.UNSPECIFIED) continue;
    if (!FEATURE_SLUGS[value]) {
      throw new Error(
        `audio-integration/features.ts: missing slug for AudioToolsFeature.${AudioToolsFeature[value]} (${value}). ` +
          "Add the entry to FEATURE_SLUGS and to clients/go/audiotools/features.go.",
      );
    }
  }
})();

/** Returns the wire-slug for an AudioToolsFeature enum value. Empty string for UNSPECIFIED. */
export function featureSlug(f: AudioToolsFeature): string {
  return FEATURE_SLUGS[f];
}

/** All registered feature slugs in stable (alphabetical) order. */
export function allFeatureSlugs(): string[] {
  return Object.values(FEATURE_SLUGS).filter((s) => s !== "").sort();
}

export { AudioToolsFeature };
// HOST DIFFERENCE: audio-tools maps its generated feature enum for its local API surface.
