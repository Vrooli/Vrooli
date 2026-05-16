import { describe, expect, it } from "vitest";

import {
  AudioPlayerBar,
  EnableAudioBanner,
  MicReadinessIndicator,
  TtsSettingsPanel,
  VoiceCommandSuggestion,
  VoiceInputButton,
  VoiceRejectionBanner,
  VoiceSettingsPanel,
  createAudioToolsClient,
} from "./index";

describe("@audio-tools/embed surface", () => {
  it("exposes the documented component set", () => {
    expect(AudioPlayerBar).toBeTypeOf("function");
    expect(EnableAudioBanner).toBeTypeOf("function");
    expect(MicReadinessIndicator).toBeTypeOf("function");
    expect(TtsSettingsPanel).toBeTypeOf("function");
    expect(VoiceCommandSuggestion).toBeTypeOf("function");
    expect(VoiceInputButton).toBeTypeOf("function");
    expect(VoiceRejectionBanner).toBeTypeOf("function");
    expect(VoiceSettingsPanel).toBeTypeOf("function");
  });

  it("creates a Connect client when given a baseUrl", () => {
    const client = createAudioToolsClient({ baseUrl: "http://localhost:0" });
    expect(client.baseUrl).toBe("http://localhost:0");
    expect(client.stt).toBeDefined();
    expect(client.tts).toBeDefined();
    expect(client.summarize).toBeDefined();
  });

  it("falls back to a sentinel base URL when window.__AUDIO_TOOLS_URL__ is unset", () => {
    const original = (globalThis as { window?: { __AUDIO_TOOLS_URL__?: string } }).window?.__AUDIO_TOOLS_URL__;
    if (typeof window !== "undefined") {
      delete window.__AUDIO_TOOLS_URL__;
    }
    const client = createAudioToolsClient();
    expect(client.baseUrl).toBe("http://localhost:0");
    if (typeof original === "string" && typeof window !== "undefined") {
      window.__AUDIO_TOOLS_URL__ = original;
    }
  });
});
