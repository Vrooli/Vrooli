// features.ts — capability / feature identifiers for the audio-tools dependency.
//
// Mirror of clients/go/audiotools/features.go. Consumer scenarios
// register a dependency-capability under AUDIO_TOOLS_CAPABILITY_SLUG and
// list the feature slugs they wire into their UI. The slugs are the
// wire-level identifiers carried on web-console's CapabilitiesService
// `repeated string features` surface; they must agree byte-for-byte
// across the API registry and every UI consumer.
//
// The web-console UI must not import foreign-scenario audio-tools proto
// types. The dependency-capability surface carries feature slugs as strings,
// so this local enum gives UI call sites stable names without crossing the
// scenario API boundary.
// HOST DIFFERENCE: this avoids a foreign scenario proto; swarm's dependency
// registration is wired to the canonical audio-tools enum.
//
// Adding a new feature: add the source value in audio-tools, update the
// server-side capability registration, then add the slug entry below.
//
/** Scenario-level capability id used when registering audio-tools as a dependency-capability. Matches the scenario slug. */
export const AUDIO_TOOLS_CAPABILITY_SLUG = "audio-tools";

export enum AudioToolsFeature {
  UNSPECIFIED = 0,
  VOICE_INPUT = 1,
  VOICE_STREAMING = 2,
  VOICE_SPEAKER_VERIFICATION = 3,
  VOICE_ENROLLMENT = 4,
  VOICE_OUTPUT = 5,
  TTS_SUMMARIZATION = 6,
  TTS_CACHE = 7,
  TTS_PARAGRAPH_SPLIT = 8,
  AUDIO_PROVIDER_ROUTING = 9,
}

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

(function assertCoverage() {
  for (const key of Object.keys(AudioToolsFeature)) {
    const numeric = Number(AudioToolsFeature[key as keyof typeof AudioToolsFeature]);
    if (!Number.isFinite(numeric)) continue;
    if (numeric === AudioToolsFeature.UNSPECIFIED) continue;
    if (!FEATURE_SLUGS[numeric as AudioToolsFeature]) {
      throw new Error(
        `audio-integration/features.ts: missing slug for AudioToolsFeature.${key} (${numeric}). ` +
          "Add the entry to FEATURE_SLUGS and to clients/go/audiotools/features.go.",
      );
    }
  }
})();

/** Returns the wire-slug for an AudioToolsFeature enum value. Empty string for UNSPECIFIED. */
export function featureSlug(f: AudioToolsFeature): string {
  return FEATURE_SLUGS[f] ?? "";
}

/** All registered feature slugs in stable (alphabetical) order. */
export function allFeatureSlugs(): string[] {
  return Object.values(FEATURE_SLUGS).filter((s) => s !== "").sort();
}
