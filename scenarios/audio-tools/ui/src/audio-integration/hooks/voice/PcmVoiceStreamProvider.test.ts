import { describe, expect, it, vi } from "vitest";
import { getDefaultVoiceTransport, registerVoiceTransport } from "@vrooli/audio-capture-browser";

const mocks = vi.hoisted(() => ({
  buildVoiceStreamWsUrl: vi.fn((language: string, sessionId: string, resumeToken: string) =>
    `${language}:${sessionId}:${resumeToken}`,
  ),
  transcribeAudioWithRetry: vi.fn(async () => "recovered transcript"),
}));

describe("scenario PCM provider transport adapter", () => {
  it("forwards URL construction and retained-audio recovery to the host API", async () => {
    registerVoiceTransport({
      buildStreamUrl: mocks.buildVoiceStreamWsUrl,
      transcribeRetained: mocks.transcribeAudioWithRetry,
    });
    const transport = getDefaultVoiceTransport();
    expect(transport).not.toBeNull();

    expect(transport?.buildStreamUrl("en-US", "session-1", "resume-1")).toBe("en-US:session-1:resume-1");
    const blob = new Blob(["pcm"], { type: "audio/webm" });
    await expect(transport?.transcribeRetained(blob, "en-US")).resolves.toBe("recovered transcript");

    expect(mocks.buildVoiceStreamWsUrl).toHaveBeenCalledWith("en-US", "session-1", "resume-1");
    expect(mocks.transcribeAudioWithRetry).toHaveBeenCalledWith(blob, "en-US");
  });
});
