import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getVoiceStreamConfig: vi.fn(),
  getWakeWordConfig: vi.fn(),
  transcribeAudio: vi.fn(),
  transcribeAudioBypassFilter: vi.fn(),
  transcribeAudioWithRetry: vi.fn(),
  buildVoiceStreamWsUrl: vi.fn(),
}));

vi.mock("./api/voice", () => mocks);
vi.mock("./hooks/voice/WhisperProvider", () => ({ WhisperProvider: class {} }));
vi.mock("./hooks/voice/PcmVoiceStreamProvider", () => ({ PcmVoiceStreamProvider: class {} }));

import { voiceCoreServices } from "./voiceCoreServices";

describe("scenario voice-core service adapter", () => {
  it("normalizes the host wake-word response for the shared core", async () => {
    mocks.getWakeWordConfig.mockResolvedValueOnce({
      configured: true,
      template: {
        label: "Hey Vrooli",
        updatedAt: "2026-08-03T19:00:00Z",
        samples: [{ audio: new Uint8Array([1, 2, 3]) }],
      },
    });

    await expect(voiceCoreServices.getWakeWordConfig()).resolves.toEqual({
      configured: true,
      template: {
        label: "Hey Vrooli",
        updatedAt: "2026-08-03T19:00:00Z",
        samples: [{ audio: new Uint8Array([1, 2, 3]) }],
      },
    });
  });
});
