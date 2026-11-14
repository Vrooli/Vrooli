import { describe, expect, it } from "vitest";
import { registerVoiceTransport } from "./index";
import { allFeatureSlugs, AudioToolsFeature, featureSlug } from "./features";

describe("audio capability registration", () => {
  it("keeps the feature enum mapped to stable sorted wire slugs", () => {
    expect(featureSlug(AudioToolsFeature.VOICE_STREAMING)).toBe("voice-streaming");
    expect(featureSlug(AudioToolsFeature.UNSPECIFIED)).toBe("");
    expect(allFeatureSlugs()).toEqual([...allFeatureSlugs()].sort());
    expect(allFeatureSlugs()).toContain("tts-summarization");
  });

  it("registers the web-console-owned voice transport adapter", () => {
    expect(() => registerVoiceTransport()).not.toThrow();
  });
});
