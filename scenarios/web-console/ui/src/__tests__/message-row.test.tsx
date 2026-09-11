import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
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

function event(id: string, sequence: number, role: ConversationEvent["role"] = "assistant"): ConversationEvent {
  return {
    id,
    sessionId: "sess-1",
    sequence,
    source: "claude_hook",
    role,
    text: `Message ${String(sequence)}`,
    speechParagraphs: [`Message ${String(sequence)}`],
    summarized: false,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
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
  onSendToComposer: vi.fn(),
};

function seed(events: ConversationEvent[]) {
  useConversationStore.setState({
    sessions: {
      "sess-1": createConversationSessionState({ events, cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
    },
  });
}

function setPointer(coarse: boolean) {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches: coarse && (query.includes("pointer: coarse") || query.includes("hover: none")),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
}

describe("message row", () => {
  let originalMatchMedia: typeof window.matchMedia;

  beforeEach(() => {
    originalMatchMedia = window.matchMedia;
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    seed([event("u1", 1, "user"), event("a2", 2), event("a3", 3)]);
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("[REQ:P0-017c] shows the speaker, a time, and the text, with no actions at rest", () => {
    setPointer(false);
    render(<MessagesPane {...props} />);

    const row = screen.getByTestId("msg-card-a2");
    expect(within(row).getByTestId("msg-speaker-a2")).toHaveTextContent(strings.messagesPane.speaker.claude);
    expect(within(row).getByTestId("msg-time-a2")).toHaveAttribute("title", "#2");
    expect(within(row).getByTestId("msg-time-a2").textContent).not.toBe("");
    expect(row).toHaveTextContent("Message 2");
    expect(screen.getByTestId("msg-speaker-u1")).toHaveTextContent(strings.messagesPane.speaker.you);
    expect(screen.queryByTestId("msg-actions-inline")).toBeNull();
    expect(row.textContent).not.toContain("#2");
  });

  it("[REQ:P0-017c] reveals at most three 44px inline controls on hover or keyboard focus (fine pointer)", () => {
    setPointer(false);
    render(<MessagesPane {...props} />);

    const row = screen.getByTestId("msg-card-a2");
    fireEvent.mouseEnter(row);
    const cluster = screen.getByTestId("msg-actions-inline");
    const controls = cluster.querySelectorAll<HTMLElement>("[data-message-action-inline]");
    expect(controls.length).toBeGreaterThan(0);
    expect(controls.length).toBeLessThanOrEqual(3);
    for (const control of controls) {
      expect(control.className).toContain("h-11");
      expect(control.className).toContain("w-11");
    }
    fireEvent.mouseLeave(row);
    expect(screen.queryByTestId("msg-actions-inline")).toBeNull();

    fireEvent.focus(row);
    expect(screen.getByTestId("msg-actions-inline")).toBeInTheDocument();
  });

  it("[REQ:P0-017c] right-click opens the full action list, primary actions first", () => {
    setPointer(false);
    render(<MessagesPane {...props} />);

    fireEvent.contextMenu(screen.getByTestId("msg-card-a2"), { clientX: 40, clientY: 60 });
    const menu = screen.getByTestId("msg-actions-menu-a2");
    const items = within(menu).getAllByRole("menuitem");
    expect(items[0]).toHaveTextContent(strings.messageActions.copy);
    expect(items[1]).toHaveTextContent(strings.messageActions.readFromHere);
    expect(within(menu).getByTestId("msg-send-to-composer-a2")).toBeInTheDocument();
  });

  it("[REQ:P0-017c] a long-press on a touch device opens the action sheet with 44px rows", () => {
    setPointer(true);
    vi.useFakeTimers();
    render(<MessagesPane {...props} />);

    const row = screen.getByTestId("msg-card-a2");
    fireEvent.mouseEnter(row);
    expect(screen.queryByTestId("msg-actions-inline")).toBeNull();

    fireEvent.pointerDown(row, { pointerType: "touch", pointerId: 7, button: 0, clientX: 10, clientY: 10 });
    act(() => { vi.advanceTimersByTime(550); });
    act(() => { window.dispatchEvent(new PointerEvent("pointerup", { pointerId: 7, clientX: 10, clientY: 10 })); });

    const sheet = screen.getByTestId("msg-action-sheet");
    const rows = sheet.querySelectorAll<HTMLElement>("[data-action-row]");
    expect(rows[0]).toHaveAttribute("data-testid", "msg-copy-a2");
    expect(rows[1]).toHaveAttribute("data-testid", "msg-speak-from-a2");
    for (const actionRow of rows) expect(actionRow.className).toContain("min-h-11");
  });

  it("[REQ:P0-017h] Read from here starts playback from that row", () => {
    setPointer(false);
    const onPlayFromHere = vi.fn();
    render(<MessagesPane {...props} onPlayFromHere={onPlayFromHere} />);
    fireEvent.mouseEnter(screen.getByTestId("msg-card-a2"));
    fireEvent.click(screen.getByTestId("msg-speak-from-a2"));
    expect(onPlayFromHere).toHaveBeenCalledWith("a2");
  });

  it("[REQ:P0-017h] the speaking row carries data-speaking and the accent border", () => {
    render(<MessagesPane {...props} isTtsSpeaking activeSpeakingEventId="a2" />);
    const row = screen.getByTestId("msg-card-a2");
    expect(row).toHaveAttribute("data-speaking", "true");
    expect(row.className).toContain("border-l-wc-accent");
    expect(screen.getByTestId("msg-card-a3")).not.toHaveAttribute("data-speaking");
  });

  it("[REQ:P0-017c] a touch that moves (a scroll) never opens the sheet", () => {
    setPointer(true);
    vi.useFakeTimers();
    render(<MessagesPane {...props} />);

    const row = screen.getByTestId("msg-card-a2");
    fireEvent.pointerDown(row, { pointerType: "touch", pointerId: 8, button: 0, clientX: 10, clientY: 10 });
    act(() => { window.dispatchEvent(new PointerEvent("pointermove", { pointerId: 8, clientX: 10, clientY: 60 })); });
    act(() => { vi.advanceTimersByTime(550); });
    act(() => { window.dispatchEvent(new PointerEvent("pointerup", { pointerId: 8, clientX: 10, clientY: 60 })); });

    expect(screen.queryByTestId("msg-action-sheet")).toBeNull();
  });

  it("[REQ:P0-017c] j/k move between messages, Enter opens the focused message's actions, Escape closes", () => {
    setPointer(false);
    render(<MessagesPane {...props} />);
    const list = screen.getByTestId("messages-scroll");

    fireEvent.keyDown(list, { key: "j" });
    const first = document.querySelector("[data-focused='true']");
    expect(first).not.toBeNull();
    fireEvent.keyDown(list, { key: "j" });
    const second = document.querySelector("[data-focused='true']");
    expect(second).not.toBe(first);
    fireEvent.keyDown(list, { key: "k" });
    expect(document.querySelector("[data-focused='true']")).toBe(first);

    const focusedId = first?.getAttribute("data-testid")?.replace("msg-card-", "") ?? "";
    fireEvent.keyDown(list, { key: "Enter" });
    expect(screen.getByTestId(`msg-actions-menu-${focusedId}`)).toBeInTheDocument();
    fireEvent.keyDown(list, { key: "Escape" });
    expect(screen.queryByTestId(`msg-actions-menu-${focusedId}`)).toBeNull();
  });
});
