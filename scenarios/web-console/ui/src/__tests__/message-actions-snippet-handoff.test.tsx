import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils";
import MessagesPane from "../components/MessagesPane";
import { createConversationSessionState, useConversationStore } from "../stores/useConversationStore";
import type { ConversationEvent } from "../api/conversation";

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }),
  loadOlderConversationPage: vi.fn().mockResolvedValue(false),
  loadConversationPageContaining: vi.fn().mockResolvedValue(false),
}));
vi.mock("../components/markdown", () => ({ MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div> }));
vi.mock("../hooks/useHandoffSuggestions", () => ({
  useHandoffSuggestions: () => ({ forEvent: () => [], dismiss: vi.fn() }),
}));
vi.mock("../hooks/useSnippets", () => ({
  useSnippets: () => ({ save: vi.fn(), snippets: [], status: "ready", error: null, reload: vi.fn(), remove: vi.fn(), touch: vi.fn() }),
}));

function message(id: string, sequence: number, role: ConversationEvent["role"]): ConversationEvent {
  return {
    id,
    sessionId: "session-1",
    sequence,
    source: "claude_hook",
    role,
    text: role === "user" ? "Useful operator request" : "Useful assistant answer",
    speechParagraphs: ["Message"],
    summarized: false,
    createdAt: "2026-08-28T00:00:00Z",
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

const baseProps = {
  sessionId: "session-1",
  onPlayFromHere: vi.fn(),
  onPlayEvent: vi.fn(),
  activeSpeakingEventId: null,
  isTtsSpeaking: false,
  summarizeLevel: "moderate" as const,
  selectedVersionForEvent: vi.fn(() => "active" as const),
  summarizingEventId: null,
  getSummarizeError: vi.fn(() => null),
  onClearSummarizeError: vi.fn(),
  onToggleSummarized: vi.fn(),
  onChangeLevel: vi.fn(),
  playbackState: {
    currentTime: 0,
    duration: null,
    isPaused: true,
    playbackRate: 1,
    volume: 1,
    isMuted: false,
    capabilities: { canPause: true, canSeek: false, canAdjustSpeed: true, canAdjustVolume: true },
  },
  onSetPlaybackRate: vi.fn(),
  onSetVolume: vi.fn(),
  onSetMuted: vi.fn(),
  playbackFocusRequest: null,
};

function seed(events: ConversationEvent[]) {
  useConversationStore.setState({
    sessions: {
      "session-1": createConversationSessionState({
        events,
        cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
        hydrated: true,
      }),
    },
    viewModes: {},
  });
}

describe("message snippet and handoff actions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IntersectionObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
  });

  it("keeps user save inline and assistant save in overflow, then opens the sheet with exact body", () => {
    seed([message("user-1", 1, "user"), message("assistant-1", 2, "assistant")]);
    render(<MessagesPane {...baseProps} />);

    expect(screen.getByTestId("msg-save-snippet-user-1")).toHaveAttribute("data-message-action-inline");
    expect(screen.queryByTestId("msg-save-snippet-assistant-1")).toBeNull();
    fireEvent.click(screen.getByTestId("msg-actions-more-assistant-1"));
    fireEvent.click(screen.getByTestId("msg-save-snippet-assistant-1"));
    expect(screen.getByTestId("snippet-save-sheet")).toBeInTheDocument();
    expect(screen.getByTestId("snippet-save-body")).toHaveValue("Useful assistant answer");
  });

  it("offers direct handoff without a capture rule and forwards exact source and payload", () => {
    const onHandoff = vi.fn();
    seed([message("assistant-1", 1, "assistant")]);
    render(<MessagesPane {...baseProps} onHandoff={onHandoff} />);

    fireEvent.click(screen.getByTestId("msg-actions-more-assistant-1"));
    fireEvent.click(screen.getByTestId("msg-handoff-assistant-1"));
    expect(onHandoff).toHaveBeenCalledWith("session-1", "Useful assistant answer");
  });

  it("keeps archive-safe save and composer actions but omits handoff", () => {
    seed([message("assistant-1", 1, "assistant")]);
    render(<MessagesPane {...baseProps} readOnly onSendToComposer={vi.fn()} />);

    fireEvent.click(screen.getByTestId("msg-actions-more-assistant-1"));
    expect(screen.getByTestId("msg-save-snippet-assistant-1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-send-to-composer-assistant-1")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-handoff-assistant-1")).toBeNull();
  });
});
