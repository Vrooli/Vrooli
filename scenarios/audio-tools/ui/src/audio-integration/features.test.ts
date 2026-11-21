import { describe, it, expect } from "vitest";

import {
  AUDIO_TOOLS_CAPABILITY_SLUG,
  featureSlug,
  allFeatureSlugs,
  AudioToolsFeature,
} from "./features";

describe("AUDIO_TOOLS_CAPABILITY_SLUG", () => {
  it("equals 'audio-tools'", () => {
    expect(AUDIO_TOOLS_CAPABILITY_SLUG).toBe("audio-tools");
  });
});

describe("featureSlug", () => {
  it("returns empty string for UNSPECIFIED", () => {
    expect(featureSlug(AudioToolsFeature.UNSPECIFIED)).toBe("");
  });

  it("returns 'voice-input' for VOICE_INPUT", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_INPUT)).toBe("voice-input");
  });

  it("returns 'voice-streaming' for VOICE_STREAMING", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_STREAMING)).toBe("voice-streaming");
  });

  it("returns 'voice-speaker-verification' for VOICE_SPEAKER_VERIFICATION", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_SPEAKER_VERIFICATION)).toBe(
      "voice-speaker-verification",
    );
  });

  it("returns 'voice-enrollment' for VOICE_ENROLLMENT", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_ENROLLMENT)).toBe("voice-enrollment");
  });

  it("returns 'voice-output' for VOICE_OUTPUT", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_OUTPUT)).toBe("voice-output");
  });

  it("returns 'tts-summarization' for TTS_SUMMARIZATION", () => {
    expect(featureSlug(AudioToolsFeature.TTS_SUMMARIZATION)).toBe("tts-summarization");
  });

  it("returns 'tts-cache' for TTS_CACHE", () => {
    expect(featureSlug(AudioToolsFeature.TTS_CACHE)).toBe("tts-cache");
  });

  it("returns 'tts-paragraph-split' for TTS_PARAGRAPH_SPLIT", () => {
    expect(featureSlug(AudioToolsFeature.TTS_PARAGRAPH_SPLIT)).toBe("tts-paragraph-split");
  });

  it("returns 'audio-provider-routing' for AUDIO_PROVIDER_ROUTING", () => {
    expect(featureSlug(AudioToolsFeature.AUDIO_PROVIDER_ROUTING)).toBe("audio-provider-routing");
  });
});

describe("allFeatureSlugs", () => {
  it("returns all non-empty slugs in sorted order", () => {
    const slugs = allFeatureSlugs();
    // All entries should be non-empty strings
    expect(slugs.every((s) => typeof s === "string" && s.length > 0)).toBe(true);
    // Sorted
    const sorted = [...slugs].sort();
    expect(slugs).toEqual(sorted);
  });

  it("does not include the UNSPECIFIED empty slug", () => {
    expect(allFeatureSlugs()).not.toContain("");
  });

  it("includes the expected feature slugs", () => {
    const slugs = allFeatureSlugs();
    expect(slugs).toContain("voice-input");
    expect(slugs).toContain("voice-streaming");
    expect(slugs).toContain("voice-output");
    expect(slugs).toContain("audio-provider-routing");
  });
});
