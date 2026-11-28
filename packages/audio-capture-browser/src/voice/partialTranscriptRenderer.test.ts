import { describe, expect, it, vi } from "vitest";
import { PartialTranscriptRenderer, type AnimationFrameScheduler } from "./partialTranscriptRenderer";
import { IDLE_VOICE_ACTIVITY } from "./activity";
import type { VoiceInputState } from "./types";

function state(partialTranscript = "", fallbackNotice: string | null = null): VoiceInputState {
  return {
    supported: true,
    backend: "whisper",
    voiceState: "recording",
    error: null,
    audioLevel: 0,
    voiceActivity: IDLE_VOICE_ACTIVITY,
    fallbackNotice,
    streamingDegradationNotice: null,
    partialTranscript,
    voiceMode: "one-shot",
    segments: [],
    commandSuggestion: null,
    rejectedAudio: null,
    speakerVerificationEnabled: false,
    speakerProfileConfigured: false,
    turnDiagnostic: null,
    wakeWordConfigured: false,
    passiveListeningActive: false,
    staleLiveMicLease: false,
  };
}

function fakeScheduler() {
  const callbacks: Array<() => void> = [];
  const cancelled: number[] = [];
  const scheduler: AnimationFrameScheduler = {
    request: (callback) => {
      callbacks.push(callback);
      return callbacks.length;
    },
    cancel: (handle) => cancelled.push(handle),
  };
  return { scheduler, callbacks, cancelled };
}

describe("PartialTranscriptRenderer", () => {
  it("applies only the newest partial once per frame and clears fallback notice", () => {
    const { scheduler, callbacks } = fakeScheduler();
    let current = state("", "stream reconnecting");
    const updateState = vi.fn((update: (s: VoiceInputState) => VoiceInputState) => {
      current = update(current);
    });
    const renderer = new PartialTranscriptRenderer(updateState, scheduler);

    renderer.schedule("hel");
    renderer.schedule("hello");
    expect(updateState).not.toHaveBeenCalled();
    expect(callbacks).toHaveLength(1);

    callbacks[0]();
    expect(updateState).toHaveBeenCalledTimes(1);
    expect(current.partialTranscript).toBe("hello");
    expect(current.fallbackNotice).toBeNull();
  });

  it("cancels a queued partial so it cannot repaint after turn completion", () => {
    const { scheduler, callbacks, cancelled } = fakeScheduler();
    const updateState = vi.fn();
    const renderer = new PartialTranscriptRenderer(updateState, scheduler);
    renderer.schedule("stale partial");
    renderer.cancel();
    callbacks[0]();

    expect(cancelled).toEqual([1]);
    expect(updateState).not.toHaveBeenCalled();
  });

  it("updates immediately where animation frames are unavailable", () => {
    let current = state();
    const renderer = new PartialTranscriptRenderer((update) => {
      current = update(current);
    }, null);

    renderer.schedule("interim");
    expect(current.partialTranscript).toBe("interim");
  });
});
