import { afterEach, describe, expect, it, vi } from "vitest";
import type { VoiceInputState, VoiceRejection } from "./types";
import { VoiceRejectionController } from "./rejectionController";

function harness() {
  let state = { rejectedAudio: null, error: null } as VoiceInputState;
  const audio = { blob: new Blob(["audio"]), mimeType: "audio/webm", durationMs: 250 };
  const provider = {
    getLastTurnAudio: vi.fn(() => audio),
    disposeLastTurn: vi.fn(),
  };
  const onTranscript = vi.fn();
  const controller = new VoiceRejectionController({
    getProvider: () => provider as never,
    setState: (update) => {
      state = typeof update === "function" ? update(state) : update;
    },
    getLanguage: () => "en-US",
    transcribeAudio: vi.fn(async () => " recovered "),
    transcribeAudioBypassFilter: vi.fn(async () => " recovered "),
    onTranscript,
  });
  return { controller, provider, onTranscript, get state() { return state; } };
}

afterEach(() => vi.useRealTimers());

describe("VoiceRejectionController", () => {
  it("surfaces retained audio and expires only the currently displayed rejection", () => {
    vi.useFakeTimers();
    const h = harness();
    h.controller.markPending(0.2, 0.7);
    h.controller.surfacePending();
    const first = h.state.rejectedAudio as VoiceRejection;
    expect(first.kind).toBe("retryable");
    expect(h.controller.hasPending).toBe(false);

    h.controller.surfaceEmptyTranscript();
    const second = h.state.rejectedAudio as VoiceRejection;
    expect(second.id).not.toBe(first.id);
    vi.advanceTimersByTime(5 * 60 * 1000);
    expect(h.state.rejectedAudio).toBeNull();
    expect(h.provider.disposeLastTurn).toHaveBeenCalledOnce();
  });

  it("does not deliver a retry result after the user dismisses its rejection", async () => {
    const h = harness();
    let resolveRetry!: (text: string) => void;
    const transcribeAudio = vi.fn(() => new Promise<string>((resolve) => { resolveRetry = resolve; }));
    const controller = new VoiceRejectionController({
      getProvider: () => h.provider as never,
      setState: (update) => {
        const current = h.state;
        Object.assign(current, typeof update === "function" ? update(current) : update);
      },
      getLanguage: () => "en",
      transcribeAudio,
      transcribeAudioBypassFilter: vi.fn(async () => ""),
      onTranscript: h.onTranscript,
    });
    controller.surfaceEmptyTranscript();
    const retry = controller.retry();
    controller.dismiss();
    resolveRetry("too late");
    await retry;
    expect(h.onTranscript).not.toHaveBeenCalled();
    expect(h.state.rejectedAudio).toBeNull();
  });
});
