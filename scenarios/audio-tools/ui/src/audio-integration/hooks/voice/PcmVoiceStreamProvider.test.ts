import { describe, expect, it, vi } from "vitest";
import { getDefaultVoiceTransport, registerVoiceTransport, type VoiceStreamConnection, type VoiceStreamHandlers } from "@vrooli/audio-capture-browser";

describe("scenario PCM provider transport adapter", () => {
  it("registers the browser stream adapter and retained-audio recovery operation", async () => {
    const connection: VoiceStreamConnection = {
      isOpen: () => true,
      sendAudio: vi.fn(),
      end: vi.fn(),
      close: vi.fn(),
    };
    const openStream = vi.fn(() => connection);
    const transcribeRetained = vi.fn(() => Promise.resolve("recovered transcript"));
    registerVoiceTransport({ retainedRoutePolicy: "exact-model", openStream, transcribeRetained });
    const transport = getDefaultVoiceTransport();
    expect(transport).not.toBeNull();

    const start = {
      language: "en-US", sessionId: "session-1", resumeToken: "resume-1",
      protocolVersion: 2 as const, generation: 0n, inputFormat: "pcm_s16le" as const, sampleRateHz: 16_000,
    };
    const handlers: VoiceStreamHandlers = { onReady: vi.fn(), onEvent: vi.fn(), onClose: vi.fn() };
    expect(transport?.openStream(start, handlers)).toBe(connection);
    expect(openStream).toHaveBeenCalledWith(start, handlers);

    const blob = new Blob(["pcm"], { type: "audio/webm" });
    const route = { providerTier: "local" as const, modelId: "whisper-large-v3-turbo" };
    await expect(transport?.transcribeRetained(blob, "en-US", route)).resolves.toBe("recovered transcript");
    expect(transcribeRetained).toHaveBeenCalledWith(blob, "en-US", route);
  });
});
