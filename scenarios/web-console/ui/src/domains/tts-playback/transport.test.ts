import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useTtsPlaybackController } from "./useTtsPlaybackController";
import { publishTransport, type PlaybackTransport } from "./transport";

vi.mock("../../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration")>();
  return {
    ...actual,
    getTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
    updateTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
  };
});

vi.mock("../../api/conversation", () => ({
  summarizeEvent: vi.fn(),
  updateConversationCursor: vi.fn().mockResolvedValue({ lastSeenSequence: 0, lastListenedSequence: 0 }),
}));

function transport(overrides: Partial<PlaybackTransport> = {}): PlaybackTransport {
  return {
    currentTime: 3,
    duration: 10,
    playbackRate: 1,
    volume: 0.8,
    isMuted: false,
    isPaused: false,
    capabilities: { canPause: true, canSeek: true, canAdjustSpeed: true, canAdjustVolume: true },
    ...overrides,
  };
}

function renderController(activePaneId: string) {
  return renderHook(() => useTtsPlaybackController({
    conversationSessions: {},
    activePaneId,
    autoTtsEnabled: false,
    audioState: { isSpeaking: true, isPaused: false },
    setViewMode: vi.fn(),
    speakText: vi.fn().mockResolvedValue(undefined),
    stopPlayback: vi.fn(),
    applySummarizeResult: vi.fn(),
  }));
}

describe("playback transport", () => {
  afterEach(() => {
    vi.useRealTimers();
    publishTransport("s1", null);
    publishTransport("s2", null);
  });

  it("[REQ:P0-017h] delivers the active pane's transport on each provider event", () => {
    const { result } = renderController("s1");
    const received = vi.fn();
    const unsubscribe = result.current.subscribeTransport(received);
    act(() => { publishTransport("s1", transport({ currentTime: 4 })); });
    act(() => { publishTransport("s1", transport({ currentTime: 4.25, playbackRate: 1.5, isPaused: true, isMuted: true })); });
    expect(received).toHaveBeenCalledTimes(2);
    expect(received).toHaveBeenLastCalledWith(expect.objectContaining({
      currentTime: 4.25,
      duration: 10,
      playbackRate: 1.5,
      volume: 0.8,
      isMuted: true,
      isPaused: true,
    }));
    unsubscribe();
  });

  it("[REQ:P0-017h] nothing arrives on a timer", () => {
    vi.useFakeTimers();
    const { result } = renderController("s1");
    const received = vi.fn();
    const unsubscribe = result.current.subscribeTransport(received);
    act(() => { vi.advanceTimersByTime(5_000); });
    expect(received).not.toHaveBeenCalled();
    act(() => { publishTransport("s1", transport()); });
    act(() => { vi.advanceTimersByTime(5_000); });
    expect(received).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("another pane's transport is not delivered", () => {
    const { result } = renderController("s1");
    const received = vi.fn();
    const unsubscribe = result.current.subscribeTransport(received);
    act(() => { publishTransport("s2", transport()); });
    expect(received).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("[REQ:P0-017h] unsubscribing stops delivery", () => {
    const { result } = renderController("s1");
    const received = vi.fn();
    const unsubscribe = result.current.subscribeTransport(received);
    act(() => { publishTransport("s1", transport()); });
    unsubscribe();
    act(() => { publishTransport("s1", transport({ currentTime: 9 })); });
    expect(received).toHaveBeenCalledTimes(1);
  });
});
