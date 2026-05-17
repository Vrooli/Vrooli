import { renderHook, act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useTtsPlaybackController } from "./useTtsPlaybackController";
import type { ConversationEvent } from "../../api/conversation";
import { useTtsPlaybackIntentStore } from "./store";

const { mockGetConfig, mockSummarizeEvent, mockUpdateConfig } = vi.hoisted(() => ({
  mockGetConfig: vi.fn(),
  mockSummarizeEvent: vi.fn(),
  mockUpdateConfig: vi.fn(),
}));

vi.mock("../../audio-integration", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../audio-integration")>();
  return {
    ...actual,
    getTTSSummarizeConfig: mockGetConfig,
    updateTTSSummarizeConfig: mockUpdateConfig,
  };
});

vi.mock("../../api/conversation", () => ({
  summarizeEvent: mockSummarizeEvent,
  updateConversationCursor: vi.fn().mockResolvedValue({ lastSeenSequence: 0, lastListenedSequence: 0 }),
}));

function makeEvent(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "sess-1",
    source: "claude_hook",
    role: "assistant",
    text: `Message ${overrides.sequence}`,
    speechParagraphs: [`Message ${overrides.sequence}`],
    summarized: false,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    ...overrides,
  };
}

describe("useTtsPlaybackController", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.localStorage.clear();
    useTtsPlaybackIntentStore.setState({
      playbackIntent: "continuous",
      selectedTarget: null,
    });
    mockGetConfig.mockResolvedValue({ level: "moderate" });
    mockUpdateConfig.mockResolvedValue({ level: "moderate" });
  });

  it("summarizes the next message when summarized playback is the active preference", async () => {
    const sessions = {
      "sess-1": {
        events: [
          makeEvent({
            id: "e1",
            sequence: 1,
            summarized: true,
            speechParagraphs: ["Summary 1"],
            originalSpeechParagraphs: ["Original 1"],
          }),
          makeEvent({
            id: "e2",
            sequence: 2,
            text: "Original 2",
            speechParagraphs: ["Original 2"],
          }),
        ],
      },
    };
    const speakText = vi.fn();
    const applySummarizeResult = vi.fn((sessionId: string, eventId: string, speechParagraphs: string[]) => {
      const session = sessions[sessionId as keyof typeof sessions];
      const event = session?.events.find((candidate) => candidate.id === eventId);
      if (event) {
        event.summarized = true;
        event.originalSpeechParagraphs = event.originalSpeechParagraphs ?? event.speechParagraphs;
        event.speechParagraphs = speechParagraphs;
      }
    });
    mockSummarizeEvent.mockResolvedValue({
      summarized: true,
      speechParagraphs: ["Summarized 2"],
    });

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult,
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.playEvent("sess-1", "e2");
    });

    await waitFor(() => {
      expect(mockSummarizeEvent).toHaveBeenCalledWith("sess-1", "e2");
    });
    await waitFor(() => {
      expect(speakText).toHaveBeenCalledWith(
        "sess-1",
        "Original 2",
        ["Summarized 2"],
        { eventId: "e2", version: "active", initiatedBy: "manual" },
      );
    });
  });

  it("surfaces loading state while playback prep is unresolved", async () => {
    const event = makeEvent({ id: "e-loading", sequence: 9, summarized: false });
    const sessions = { "sess-1": { events: [event] } };
    let releaseSummarize: ((value: { summarized: boolean; speechParagraphs: string[] }) => void) | null = null;
    mockSummarizeEvent.mockImplementation(() => new Promise((resolve) => {
      releaseSummarize = resolve;
    }));
    const speakText = vi.fn().mockResolvedValue("browser");

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.playEvent("sess-1", "e-loading");
    });

    await waitFor(() => {
      expect(result.current.loadingEventId).toBe("e-loading");
    });

    await act(async () => {
      releaseSummarize?.({ summarized: true, speechParagraphs: ["Summary"] });
      await Promise.resolve();
    });
  });

  it("normalizes unavailable summarize transport failures for the banner", async () => {
    const event = makeEvent({ id: "e-fail", sequence: 10, text: "Original" });
    const sessions = { "sess-1": { events: [event] } };
    const onSummarizeFailed = vi.fn();
    const speakText = vi.fn().mockResolvedValue("browser");
    mockSummarizeEvent.mockRejectedValue(new Error("[unavailable] HTTP 502"));

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed,
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.playEvent("sess-1", "e-fail");
    });

    await waitFor(() => {
      expect(onSummarizeFailed).toHaveBeenCalledWith(
        "sess-1",
        "e-fail",
        expect.stringContaining("audio-tools is unavailable"),
      );
    });
    expect(onSummarizeFailed.mock.calls[0]?.[2]).not.toContain("[unavailable] HTTP 502");
  });

  it("normalizes summarize timeout failures separately from availability", async () => {
    const event = makeEvent({ id: "e-timeout", sequence: 10, text: "Original" });
    const sessions = { "sess-1": { events: [event] } };
    const onSummarizeFailed = vi.fn();
    mockSummarizeEvent.mockRejectedValue(new Error("[deadline_exceeded] timeout"));

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText: vi.fn().mockResolvedValue("browser"),
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed,
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.playEvent("sess-1", "e-timeout");
    });

    await waitFor(() => {
      expect(onSummarizeFailed).toHaveBeenCalledWith(
        "sess-1",
        "e-timeout",
        expect.stringContaining("timed out"),
      );
    });
    expect(onSummarizeFailed.mock.calls[0]?.[2]).not.toContain("audio-tools is unavailable");
  });

  it("normalizes missing summarize model failures separately from availability", async () => {
    const event = makeEvent({ id: "e-missing-model", sequence: 10, text: "Original" });
    const sessions = { "sess-1": { events: [event] } };
    const onSummarizeFailed = vi.fn();
    mockSummarizeEvent.mockRejectedValue(new Error("[failed_precondition] summarize model is not installed"));

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText: vi.fn().mockResolvedValue("browser"),
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed,
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.playEvent("sess-1", "e-missing-model");
    });

    await waitFor(() => {
      expect(onSummarizeFailed).toHaveBeenCalledWith(
        "sess-1",
        "e-missing-model",
        expect.stringContaining("model is not installed"),
      );
    });
    expect(onSummarizeFailed.mock.calls[0]?.[2]).not.toContain("audio-tools is unavailable");
  });

  it("carries original-mode preference across messages", async () => {
    const sessions = {
      "sess-1": {
        events: [
          makeEvent({
            id: "e1",
            sequence: 1,
            summarized: true,
            speechParagraphs: ["Summary 1"],
            originalSpeechParagraphs: ["Original 1"],
          }),
          makeEvent({
            id: "e2",
            sequence: 2,
            summarized: true,
            speechParagraphs: ["Summary 2"],
            originalSpeechParagraphs: ["Original 2"],
          }),
        ],
      },
    };
    const speakText = vi.fn();

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.toggleVersion("sess-1", "e1", false);
    });

    act(() => {
      result.current.playEvent("sess-1", "e2");
    });

    await waitFor(() => {
      expect(speakText).toHaveBeenLastCalledWith(
        "sess-1",
        "Message 2",
        ["Original 2"],
        { eventId: "e2", version: "original", initiatedBy: "manual" },
      );
    });
  });

  it("blocks incoming assistant auto-play while user intent is paused", async () => {
    useTtsPlaybackIntentStore.setState({ playbackIntent: "paused" });
    const event = makeEvent({ id: "e-paused", sequence: 3 });
    const sessions = { "sess-1": { events: [event] } };
    const speakText = vi.fn().mockResolvedValue("browser");

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.handleIncomingEvent("sess-1", event, vi.fn());
    });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(speakText).not.toHaveBeenCalled();
    expect(useTtsPlaybackIntentStore.getState().selectedTarget).toBeNull();
  });

  it("auto-plays an incoming active-pane assistant event when intent is continuous", async () => {
    const event = makeEvent({ id: "e-live", sequence: 4, summarized: true });
    const sessions = { "sess-1": { events: [event] } };
    const speakText = vi.fn().mockResolvedValue("browser");
    const ack = vi.fn();

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "sess-1",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.handleIncomingEvent("sess-1", event, ack);
    });

    await waitFor(() => {
      expect(speakText).toHaveBeenCalledWith(
        "sess-1",
        event.text,
        event.speechParagraphs,
        { eventId: "e-live", version: "active", initiatedBy: "auto" },
      );
    });
    expect(ack).toHaveBeenCalledWith("playback_started");
  });

  it("does not replace replay target for inactive-pane incoming events", async () => {
    const priorTarget = { sessionId: "sess-1", eventId: "prior" };
    useTtsPlaybackIntentStore.setState({ selectedTarget: priorTarget });
    const event = makeEvent({ id: "inactive", sequence: 5 });
    const sessions = { "sess-1": { events: [event] } };
    const speakText = vi.fn().mockResolvedValue("browser");

    const { result } = renderHook(() => useTtsPlaybackController({
      conversationSessions: sessions,
      activePaneId: "other-pane",
      autoTtsEnabled: true,
      audioState: { playback: null, isSpeaking: false },
      setViewMode: vi.fn(),
      speakText,
      stopPlayback: vi.fn(),
      applySummarizeResult: vi.fn(),
      onSummarizeFailed: vi.fn(),
      onSummarizeSucceeded: vi.fn(),
    }));

    await waitFor(() => {
      expect(result.current.summarizeLevel).toBe("moderate");
    });

    act(() => {
      result.current.handleIncomingEvent("sess-1", event, vi.fn());
    });

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(speakText).not.toHaveBeenCalled();
    expect(useTtsPlaybackIntentStore.getState().selectedTarget).toEqual(priorTarget);
  });
});
