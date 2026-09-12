import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { i18n } from "../i18n";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { useSessionActivityStore } from "../stores/useSessionActivityStore";
import type { PendingPromptView, SessionActivityView } from "../api/sessionActivity";
import type { ConversationEvent } from "../api/conversation";

const { answerPrompt } = vi.hoisted(() => ({ answerPrompt: vi.fn() }));

vi.mock("../api/promptAnswer", () => ({ answerPrompt }));

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }),
  loadOlderConversationPage: vi.fn().mockResolvedValue(false),
  loadConversationPageContaining: vi.fn().mockResolvedValue(false),
}));

vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div>{content}</div>,
}));

vi.mock("../hooks/useHandoffSuggestions", () => ({
  useHandoffSuggestions: () => ({ forEvent: () => [], dismiss: vi.fn() }),
}));

function event(sequence: number): ConversationEvent {
  return {
    id: `e${String(sequence)}`, sessionId: "sess-1", sequence, source: "claude_hook", role: "assistant",
    text: `Message ${String(sequence)}`, speechParagraphs: [`Message ${String(sequence)}`], summarized: false,
    createdAt: "2026-09-11T07:59:00Z", deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  };
}

const props = {
  sessionId: "sess-1",
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
  playbackFocusRequest: null,
  onOpenTerminal: vi.fn(),
};

function waitingOn(prompt: Partial<PendingPromptView> = {}): SessionActivityView {
  return {
    state: "waiting", source: "screen", confidence: 0.85, since: "2026-09-11T08:01:00Z", harness: "claude",
    prompt: {
      kind: "permission",
      text: "Do you want to proceed?",
      options: [{ key: "1", label: "Yes", selected: true }, { key: "2", label: "No", selected: false }],
      answerable: true,
      hash: "h1",
      cancellable: false,
      ...prompt,
    },
  };
}

function setActivity(activity: SessionActivityView | null) {
  useSessionActivityStore.setState({ activities: activity ? { "sess-1": activity } : {} });
}

function echoTexts(): string[] {
  return (useConversationStore.getState().echoes["sess-1"] ?? []).map((echo) => echo.text);
}

describe("session state slot answering", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en");
    vi.useFakeTimers({ now: new Date("2026-09-11T08:01:12Z"), toFake: ["Date", "setInterval", "clearInterval"] });
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    answerPrompt.mockReset();
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({
      sessions: {
        "sess-1": createConversationSessionState({ events: [event(1), event(2)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
      echoes: {},
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    setActivity(null);
  });

  it("[REQ:P0-017i] an answerable prompt's options are buttons that send, and the answer shows as an echo row", async () => {
    answerPrompt.mockResolvedValue({ delivery: "keystrokes", answer: "Yes" });
    setActivity(waitingOn());
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "waiting-answerable");
    const yes = within(slot).getByTestId("state-slot-option-1");
    expect(yes.tagName).toBe("BUTTON");
    expect(yes).toHaveTextContent("Send 1");
    await act(async () => { fireEvent.click(yes); await Promise.resolve(); });

    expect(answerPrompt).toHaveBeenCalledWith("sess-1", { optionKey: "1", promptHash: "h1", cancel: false });
    expect(echoTexts()).toContain("Yes");
    // Until the next activity arrives the prompt cannot be answered twice.
    expect(within(slot).getByTestId("state-slot-option-2")).toBeDisabled();
  });

  it("[REQ:P0-017i] a read-only prompt shows its options without buttons", () => {
    setActivity(waitingOn({ answerable: false }));
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "waiting-rendered");
    expect(within(slot).getByTestId("state-slot-option-1").tagName).toBe("SPAN");
    expect(within(slot).queryByTestId("state-slot-cancel")).toBeNull();
  });

  it("[REQ:P0-017i] a cancel button is offered only when the prompt allows Escape", async () => {
    answerPrompt.mockResolvedValue({ delivery: "keystrokes", answer: "" });
    setActivity(waitingOn({ cancellable: true }));
    const { unmount } = render(<MessagesPane {...props} />);
    await act(async () => { fireEvent.click(screen.getByTestId("state-slot-cancel")); await Promise.resolve(); });
    expect(answerPrompt).toHaveBeenCalledWith("sess-1", { promptHash: "h1", cancel: true });
    unmount();

    setActivity(waitingOn({ cancellable: false, hash: "h2" }));
    render(<MessagesPane {...props} />);
    expect(screen.queryByTestId("state-slot-cancel")).toBeNull();
  });

  it("[REQ:P0-017i] a refused answer says why and leaves the options usable", async () => {
    answerPrompt.mockRejectedValue(new Error("the prompt changed"));
    setActivity(waitingOn());
    render(<MessagesPane {...props} />);
    await act(async () => { fireEvent.click(screen.getByTestId("state-slot-option-1")); await Promise.resolve(); });

    expect(screen.getByTestId("state-slot-answer-error")).toHaveTextContent("the prompt changed");
    expect(screen.getByTestId("state-slot-option-1")).not.toBeDisabled();
    expect(echoTexts()).toHaveLength(0);
  });
});
