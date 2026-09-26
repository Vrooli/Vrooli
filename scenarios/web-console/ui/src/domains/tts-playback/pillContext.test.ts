import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ConversationEvent } from "../../api/conversation";
import { useTtsPlaybackIntentStore } from "./store";
import { useTtsPlaybackController } from "./useTtsPlaybackController";

vi.mock("../../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration")>();
  return {
    ...actual,
    getTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
    updateTTSSummarizeConfig: vi.fn().mockResolvedValue({ level: "moderate" }),
  };
});

vi.mock("../../api/conversation", () => ({
  summarizeEvent: vi.fn().mockResolvedValue({ summarized: false }),
  updateConversationCursor: vi.fn().mockResolvedValue({ lastSeenSequence: 0, lastListenedSequence: 0 }),
}));

function makeEvent(id: string, sequence: number): ConversationEvent {
  return {
    id,
    sessionId: "s1",
    sequence,
    source: "claude_hook",
    role: "assistant",
    text: `Message ${String(sequence)}`,
    speechParagraphs: [`Message ${String(sequence)}`],
    summarized: true,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

function renderController(setViewMode = vi.fn()) {
  const sessions = { s1: { events: [makeEvent("e1", 1), makeEvent("e2", 2)] } };
  const hook = renderHook(() => useTtsPlaybackController({
    conversationSessions: sessions,
    activePaneId: "s1",
    autoTtsEnabled: false,
    audioState: { isSpeaking: false, isPaused: false },
    setViewMode,
    // Synthesis never finishes here: the pill must not wait for audio.
    speakText: vi.fn(() => new Promise<string | undefined>(() => {})),
    stopPlayback: vi.fn(),
    applySummarizeResult: vi.fn(),
  }));
  return { ...hook, setViewMode };
}

describe("playback pill context", () => {
  beforeEach(() => {
    useTtsPlaybackIntentStore.setState({ playbackIntent: "continuous", selectedTarget: null });
  });

  it("[REQ:P0-017h] Read from here makes the pill's context available before audio starts", async () => {
    const { result } = renderController();
    expect(result.current.buildPillContext("s1", false, { isSpeaking: false, isPaused: false })).toBeNull();
    act(() => { result.current.playFromHere("s1", "e1"); });
    await act(async () => { await Promise.resolve(); });
    expect(result.current.buildPillContext("s1", false, { isSpeaking: false, isPaused: false })).toMatchObject({
      event: { id: "e1" },
      queueIndex: 0,
      queueLength: 2,
    });
  });

  it("[REQ:P0-017h] a paused pill stays up with auto-TTS off, so it can be resumed", async () => {
    const sessions = { s1: { events: [makeEvent("e1", 1), makeEvent("e2", 2)] } };
    const { result, rerender } = renderHook(
      ({ audioState }: { audioState: { isSpeaking: boolean; isPaused: boolean } }) => useTtsPlaybackController({
        conversationSessions: sessions,
        activePaneId: "s1",
        autoTtsEnabled: false,
        audioState,
        setViewMode: vi.fn(),
        speakText: vi.fn(() => new Promise<string | undefined>(() => {})),
        stopPlayback: vi.fn(),
        applySummarizeResult: vi.fn(),
      }),
      { initialProps: { audioState: { isSpeaking: false, isPaused: false } } },
    );
    act(() => { result.current.playFromHere("s1", "e1"); });
    await act(async () => { await Promise.resolve(); });
    rerender({ audioState: { isSpeaking: true, isPaused: false } });
    rerender({ audioState: { isSpeaking: true, isPaused: true } });

    expect(result.current.buildPillContext("s1", false, { isSpeaking: false, isPaused: true })).toMatchObject({ event: { id: "e1" } });
  });

  it("[REQ:P0-017h] Jump to message focuses the speaking event in the Messages view", async () => {
    const { result, setViewMode } = renderController();
    act(() => { result.current.playFromHere("s1", "e1"); });
    await act(async () => { await Promise.resolve(); });
    act(() => { result.current.focusCurrentEvent("s1"); });
    expect(setViewMode).toHaveBeenCalledWith("s1", "messages");
    expect(result.current.focusRequest?.eventId).toBe("e1");
  });
});
