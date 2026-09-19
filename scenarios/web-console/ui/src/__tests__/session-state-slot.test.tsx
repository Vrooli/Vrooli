import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { i18n } from "../i18n";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { useSessionActivityStore } from "../stores/useSessionActivityStore";
import type { SessionActivityView } from "../api/sessionActivity";
import type { ConversationEvent } from "../api/conversation";

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

const onOpenTerminal = vi.fn();

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
  onOpenTerminal,
};

function setActivity(activity: SessionActivityView | null) {
  useSessionActivityStore.setState({ activities: activity ? { "sess-1": activity } : {} });
}

describe("session state slot", () => {
  beforeEach(async () => {
    // The slot's copy is the contract here ("Working · 1 m 12 s"), so this file
    // renders real English instead of the default key-echo locale.
    await i18n.changeLanguage("en");
    vi.useFakeTimers({ now: new Date("2026-09-11T08:01:12Z"), toFake: ["Date", "setInterval", "clearInterval"] });
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    onOpenTerminal.mockClear();
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({
      sessions: {
        "sess-1": createConversationSessionState({ events: [event(1), event(2)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    setActivity(null);
  });

  it("[REQ:P0-017e] shows a working strip timed from when work began", () => {
    setActivity({ state: "working", source: "screen", confidence: 0.85, since: "2026-09-11T08:00:00Z", lastOutputAt: "2026-09-11T08:01:08Z", harness: "claude" });
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "working");
    expect(slot).toHaveTextContent("Working · 1 m 12 s");
    expect(slot).toHaveTextContent("last output 4 s ago");

    act(() => { vi.advanceTimersByTime(3000); });
    expect(slot).toHaveTextContent("Working · 1 m 15 s");
  });

  it("[REQ:P0-017e] shows a level-1 card that hands the user to the terminal", () => {
    setActivity({ state: "waiting", source: "screen", confidence: 0.85, since: "2026-09-11T08:00:58Z", harness: "claude", prompt: { kind: "unknown", text: "", options: [], answerable: false } });
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "waiting-detected");
    expect(slot).toHaveTextContent("Claude is asking you something");
    expect(slot).toHaveTextContent("detected 14 s ago · terminal scrape");
    fireEvent.click(within(slot).getByTestId("state-slot-open-terminal"));
    expect(onOpenTerminal).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017e] lays out a level-2 card with the prompt and read-only options", () => {
    setActivity({
      state: "waiting", source: "hook", confidence: 0.9, since: "2026-09-11T08:01:00Z", harness: "claude",
      prompt: { kind: "permission", text: "Do you want to proceed?", options: [{ key: "1", label: "Yes", selected: true }, { key: "2", label: "No", selected: false }], answerable: false },
    });
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "waiting-rendered");
    expect(slot).toHaveTextContent("Do you want to proceed?");
    expect(within(slot).getByText("Yes")).toBeInTheDocument();
    expect(within(slot).getByText("No")).toBeInTheDocument();
    expect(within(slot).getByTestId("state-slot-answer-in-terminal")).toBeInTheDocument();
  });

  it("[REQ:P0-017i] a level-2 card shows the question, every option with the selected one marked, the free-text hint, and Answer in terminal", () => {
    setActivity({
      state: "waiting", source: "screen", confidence: 0.85, since: "2026-09-11T08:01:00Z", harness: "claude",
      prompt: {
        kind: "question",
        text: "Empty dir\nShould I rmdir it?",
        options: [
          { key: "1", label: "rmdir and write file (Recommended)", selected: true },
          { key: "2", label: "Leave it, stop", selected: false },
          { key: "3", label: "Type something.", selected: false },
        ],
        answerable: false,
        freeTextHint: "Type something.",
      },
    });
    render(<MessagesPane {...props} />);

    const slot = screen.getByTestId("messages-state-slot");
    expect(slot).toHaveAttribute("data-kind", "waiting-rendered");
    expect(slot).toHaveTextContent("Should I rmdir it?");
    for (const key of ["1", "2", "3"]) expect(within(slot).getByTestId(`state-slot-option-${key}`)).toBeInTheDocument();
    expect(within(slot).getByTestId("state-slot-option-1")).toHaveAttribute("aria-current", "true");
    expect(within(slot).getByTestId("state-slot-option-2")).not.toHaveAttribute("aria-current");
    expect(within(slot).getByTestId("state-slot-free-text")).toHaveTextContent("Type something.");
    // The level-1 card is not rendered alongside it.
    expect(within(slot).queryByTestId("state-slot-open-terminal")).toBeNull();
    fireEvent.click(within(slot).getByTestId("state-slot-answer-in-terminal"));
    expect(onOpenTerminal).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017e] shows nothing for idle, unknown, or a low-confidence reading", () => {
    setActivity({ state: "idle", source: "screen", confidence: 0.75, since: "2026-09-11T08:00:00Z", harness: "claude" });
    const { rerender } = render(<MessagesPane {...props} />);
    expect(screen.queryByTestId("messages-state-slot")).toBeNull();

    act(() => { setActivity({ state: "working", source: "output_clock", confidence: 0.5, since: "2026-09-11T08:00:00Z", harness: "" }); });
    rerender(<MessagesPane {...props} />);
    expect(screen.queryByTestId("messages-state-slot")).toBeNull();
  });

  it("[REQ:P0-017e] renders exactly one slot, outside the virtual list, without scroll anchoring", () => {
    setActivity({ state: "working", source: "screen", confidence: 0.85, since: "2026-09-11T08:00:00Z", harness: "claude" });
    render(<MessagesPane {...props} />);

    const slots = screen.getAllByTestId("messages-state-slot");
    expect(slots).toHaveLength(1);
    expect(slots[0]?.className).toContain("[overflow-anchor:none]");
    expect(slots[0]?.closest("[data-event-id]")).toBeNull();
    expect(screen.getByTestId("messages-scroll")).toContainElement(slots[0] ?? null);
  });
});
