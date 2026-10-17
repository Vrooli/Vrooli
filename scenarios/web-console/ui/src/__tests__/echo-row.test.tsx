import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { i18n } from "../i18n";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { useSessionActivityStore } from "../stores/useSessionActivityStore";
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

function addEcho(text = "run the tests") {
  act(() => { useConversationStore.getState().addEcho("sess-1", text); });
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

  it("[REQ:P0-017f] shows a dimmed echo the moment a send is acknowledged, never as an event", () => {
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve(null)} />);
    addEcho();

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "sending");
    expect(row).toHaveTextContent("run the tests");
    expect(row).toHaveTextContent("sending…");
    expect(row.closest("[data-event-id]")).toBeNull();
    const events = useConversationStore.getState().sessions["sess-1"]?.events ?? [];
    expect(events.some((e) => e.text === "run the tests")).toBe(false);
  });

  it("[REQ:P0-017f] offers Enter and the terminal when the text sits unsubmitted on screen", async () => {
    const onPressEnter = vi.fn();
    const onOpenTerminal = vi.fn();
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve("❯ run the tests")} onPressEnter={onPressEnter} onOpenTerminal={onOpenTerminal} />);
    addEcho();

    await act(async () => { await vi.advanceTimersByTimeAsync(1000); });

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "unsubmitted");
    expect(row).toHaveTextContent("typed into terminal, not submitted");
    fireEvent.click(screen.getByTestId("msg-echo-enter"));
    expect(onPressEnter).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId("msg-echo-open-terminal"));
    expect(onOpenTerminal).toHaveBeenCalledTimes(1);
  });

  it("[REQ:P0-017f] finds a send the terminal soft-wrapped across screen rows", async () => {
    // A phone pane is 43 columns wide: after the prompt, the line wraps
    // mid-word (seen live on a throwaway session).
    const wrapped = "matthalloran8@swarminator:~/Vrooli$ echo wc\n-echo-probe\n\n[wc-eab735,0] \"swarminator\" 08:49 11-Sep-26";
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve(wrapped)} onPressEnter={vi.fn()} />);
    addEcho("echo wc-echo-probe");

    await act(async () => { await vi.advanceTimersByTimeAsync(1000); });

    expect(screen.getByTestId("msg-echo-row")).toHaveAttribute("data-state", "unsubmitted");
    expect(screen.getByTestId("msg-echo-enter")).toBeInTheDocument();
  });

  it("[REQ:P0-017f] is replaced by the real row when the harness records the user turn", () => {
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve(null)} />);
    addEcho();

    act(() => {
      useConversationStore.getState().appendEvent(event(3, { id: "u3", role: "user", text: "run the tests", createdAt: "2026-09-11T08:00:04Z" }));
    });

    expect(screen.queryByTestId("msg-echo-row")).toBeNull();
    expect(useConversationStore.getState().echoes["sess-1"] ?? []).toHaveLength(0);
  });

  it("[REQ:P0-017f] a merged page carrying the user turn also resolves the echo", () => {
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve(null)} />);
    addEcho();

    act(() => {
      useConversationStore.getState().mergeEvents("sess-1", [event(3, { id: "u3", role: "user", text: "run the tests\r", createdAt: "2026-09-11T08:00:04Z" })]);
    });

    expect(screen.queryByTestId("msg-echo-row")).toBeNull();
  });

  it("[REQ:P0-017f] says the send was not seen when neither happens within 60 s", async () => {
    render(<MessagesPane {...props} getTerminalText={() => Promise.resolve("unrelated output")} onOpenTerminal={vi.fn()} />);
    addEcho();

    await act(async () => { await vi.advanceTimersByTimeAsync(60_000); });

    const row = screen.getByTestId("msg-echo-row");
    expect(row).toHaveAttribute("data-state", "not-seen");
    expect(row).toHaveTextContent("sent · not seen on screen");
    expect(screen.getByTestId("msg-echo-open-terminal")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-echo-enter")).toBeNull();
  });
});
