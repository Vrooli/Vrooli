import { describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getVoiceStreamConfig: vi.fn(),
  getWakeWordConfig: vi.fn(),
  transcribeAudio: vi.fn(),
  transcribeAudioBypassFilter: vi.fn(),
  transcribeAudioWithRetry: vi.fn(),
}));

vi.mock("./api/voice", () => mocks);
vi.mock("./hooks/voice/WhisperProvider", () => ({ WhisperProvider: vi.fn() }));
vi.mock("./hooks/voice/PcmVoiceStreamProvider", () => ({ PcmVoiceStreamProvider: vi.fn() }));

import { PcmVoiceStreamProvider } from "./hooks/voice/PcmVoiceStreamProvider";
import { setVoiceConfigFromServer, _resetVoiceConfigForTesting } from "./hooks/useVoiceConfigStore";
import { voiceCoreServices } from "./voiceCoreServices";

describe("scenario voice-core service adapter", () => {
  it.each([
    ["sherpa-streaming", true],
    ["kyutai", false],
    ["", false],
  ])("uses the fast first batch only for resolved engine %s", (engineId, fastFirstWireBatch) => {
    setVoiceConfigFromServer({ engineId, vadSilenceMs: 1000, segmentSilenceMs: 1000, persistentMode: false, wakeWordEnabled: false });
    new voiceCoreServices.PcmVoiceStreamProvider();
    expect(vi.mocked(PcmVoiceStreamProvider)).toHaveBeenLastCalledWith({
      wireBatchSamples: 800,
      fastFirstWireBatch,
    });
    _resetVoiceConfigForTesting();
  });

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
