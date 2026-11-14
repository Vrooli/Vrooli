import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { i18n } from "../i18n";
import { useConversationStore, createConversationSessionState, type OutboxSend } from "../stores/useConversationStore";
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

function event(sequence: number, overrides: Partial<ConversationEvent> = {}): ConversationEvent {
  return {
    id: `e${String(sequence)}`, sessionId: "sess-1", sequence, source: "claude_hook", role: "assistant",
    text: `Message ${String(sequence)}`, speechParagraphs: [`Message ${String(sequence)}`], summarized: false,
    createdAt: "2026-09-11T07:59:00Z", deliveryState: "received", ttsState: "idle", consumptionState: "seen",
    ...overrides,
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
};

function addEcho(send: OutboxSend | string = "run the tests") {
  const payload: OutboxSend = typeof send === "string" ? { text: send } : send;
  act(() => { useConversationStore.getState().addEcho("sess-1", payload); });
}

function workingSince(iso: string): SessionActivityView {
  return { state: "working", source: "screen", confidence: 1, since: iso, harness: "claude" };
}

describe("echo rows", () => {
  beforeEach(async () => {
    // The delivery labels are the contract here, so this file renders English.
    await i18n.changeLanguage("en");
    vi.useFakeTimers({ now: new Date("2026-09-11T08:00:00Z"), toFake: ["Date", "setTimeout", "clearTimeout", "setInterval", "clearInterval"] });
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useSessionActivityStore.setState({ activities: {} });
    useConversationStore.setState({
      sessions: {
        "sess-1": createConversationSessionState({ events: [event(1), event(2)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
      echoes: {},
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("[REQ:P0-017f] shows a staged send the moment it is acknowledged, never as an event", () => {
    render(<MessagesPane {...props} />);
    addEcho();

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "staged");
    expect(row).toHaveTextContent("run the tests");
    expect(row).toHaveTextContent("typed into terminal, not submitted");
    expect(row.closest("[data-event-id]")).toBeNull();
    const events = useConversationStore.getState().sessions["sess-1"]?.events ?? [];
    expect(events.some((e) => e.text === "run the tests")).toBe(false);
  });

  it("[REQ:P0-017f] a staged send offers Enter and the terminal, and Enter marks it submitted", () => {
    const onPressEnter = vi.fn();
    const onOpenTerminal = vi.fn();
    render(<MessagesPane {...props} onPressEnter={onPressEnter} onOpenTerminal={onOpenTerminal} />);
    addEcho();

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "staged");
    fireEvent.click(screen.getByTestId("msg-echo-open-terminal"));
    expect(onOpenTerminal).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("msg-echo-enter"));
    expect(onPressEnter).toHaveBeenCalledTimes(1);
    // Pressing Enter resolves the staged state locally — no more "Press Enter".
    expect(screen.getByTestId("msg-echo-row")).toHaveAttribute("data-state", "submitted");
    expect(screen.queryByTestId("msg-echo-enter")).toBeNull();
  });

  it("coalesces two staged sends into one row — a two-line message, not two", () => {
    render(<MessagesPane {...props} />);
    addEcho("line one");
    addEcho("line two");

    const rows = screen.getAllByTestId("msg-echo-row");
    expect(rows).toHaveLength(1);
    expect(rows[0]).toHaveTextContent("line one");
    expect(rows[0]).toHaveTextContent("line two");
  });

  it("shows an attachment count without hunting for the path on screen", () => {
    render(<MessagesPane {...props} />);
    addEcho({ text: "describe this", payload: "describe this /tmp/u.png /tmp/v.png", attachmentCount: 2 });

    expect(screen.getByTestId("msg-echo-attachments")).toHaveTextContent("+2 attachments");
    expect(screen.getByTestId("msg-echo-row")).toHaveAttribute("data-state", "staged");
  });

  it("stops saying 'Press Enter' once the agent is seen to start working", () => {
    render(<MessagesPane {...props} />);
    addEcho();
    expect(screen.getByTestId("msg-echo-row")).toHaveAttribute("data-state", "staged");

    // The agent began a turn after the send was staged → it was submitted.
    act(() => { useSessionActivityStore.getState().apply("sess-1", workingSince("2026-09-11T08:00:05Z")); });

    expect(screen.getByTestId("msg-echo-row")).toHaveAttribute("data-state", "submitted");
    expect(screen.queryByTestId("msg-echo-enter")).toBeNull();
  });

  it("shows a send submitted while the agent was already working as queued", () => {
    render(<MessagesPane {...props} />);
    addEcho({ text: "next thing", submitted: true, agentBusy: true });

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "queued");
    expect(row).toHaveTextContent("queued behind the current turn");
  });

  it("[REQ:P0-017f] is replaced by the real row when the harness records the user turn", () => {
    render(<MessagesPane {...props} />);
    addEcho();

    act(() => {
      useConversationStore.getState().appendEvent(event(3, { id: "u3", role: "user", text: "run the tests", createdAt: "2026-09-11T08:00:04Z" }));
    });

    expect(screen.queryByTestId("msg-echo-row")).toBeNull();
    expect(useConversationStore.getState().echoes["sess-1"] ?? []).toHaveLength(0);
  });

  it("clears an attachment send even though the harness recorded a chip, not the path", () => {
    render(<MessagesPane {...props} />);
    addEcho({ text: "describe this screenshot", payload: "describe this screenshot /tmp/u.png", attachmentCount: 1 });

    act(() => {
      useConversationStore.getState().appendEvent(event(3, { id: "u3", role: "user", text: "describe this screenshot [Image #1]", createdAt: "2026-09-11T08:00:04Z" }));
    });

    expect(screen.queryByTestId("msg-echo-row")).toBeNull();
  });

  it("[REQ:P0-017f] a merged page carrying the user turn also resolves the echo", () => {
    render(<MessagesPane {...props} />);
    addEcho();

    act(() => {
      useConversationStore.getState().mergeEvents("sess-1", [event(3, { id: "u3", role: "user", text: "run the tests\r", createdAt: "2026-09-11T08:00:04Z" })]);
    });

    expect(screen.queryByTestId("msg-echo-row")).toBeNull();
  });
});
