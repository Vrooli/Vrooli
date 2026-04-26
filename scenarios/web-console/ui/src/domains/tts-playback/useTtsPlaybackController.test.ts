import { renderHook, act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useTtsPlaybackController } from "./useTtsPlaybackController";
import type { ConversationEvent } from "../../lib/api";

const { mockGetConfig, mockSummarizeEvent, mockUpdateConfig } = vi.hoisted(() => ({
  mockGetConfig: vi.fn(),
  mockSummarizeEvent: vi.fn(),
  mockUpdateConfig: vi.fn(),
}));

vi.mock("../../lib/api", () => ({
  getTTSSummarizeConfig: mockGetConfig,
  summarizeEvent: mockSummarizeEvent,
  updateTTSSummarizeConfig: mockUpdateConfig,
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
      speakSequence: vi.fn().mockResolvedValue(undefined),
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
        { eventId: "e2", version: "active" },
      );
    });
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
      speakSequence: vi.fn().mockResolvedValue(undefined),
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
        { eventId: "e2", version: "original" },
      );
    });
  });
});
