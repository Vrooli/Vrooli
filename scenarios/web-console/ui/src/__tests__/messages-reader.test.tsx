import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
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

const LONG = "# Findings\n\nalpha one. beta two. alpha three.\n\n## More\n\n```ts\nconst alpha = 1;\n```\n";

function event(id: string, sequence: number, text: string): ConversationEvent {
  return {
    id, sessionId: "sess-1", sequence, source: "claude_hook", role: "assistant", text, speechParagraphs: [text],
    summarized: false, createdAt: new Date().toISOString(), deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  };
}

function seed(events: ConversationEvent[]) {
  useConversationStore.setState({
    sessions: {
      "sess-1": createConversationSessionState({
        events,
        cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
        hydrated: true,
      }),
    },
  });
}

const onPlayEvent = vi.fn();
const props = {
  sessionId: "sess-1",
  onPlayFromHere: vi.fn(),
  onPlayEvent,
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

describe("messages reader", () => {
  let scrollTop = 0;

  beforeEach(() => {
    vi.clearAllMocks();
    // jsdom has no layout, so no scrollIntoView; the reader calls it for the active match.
    Element.prototype.scrollIntoView = vi.fn();
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), disconnect: vi.fn() })));
    globalThis.fetch = vi.fn() as typeof fetch;
    useMessagesViewStore.setState({ viewModes: {}, positions: {}, readerFontSize: null, readers: {} });
    useWorkspaceStore.setState({ keyboardOpen: false });
    seed([event("short", 1, "A short reply."), event("long", 2, LONG)]);
    // The long reply's content box renders taller than the collapse threshold.
    Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
      configurable: true,
      get(this: HTMLElement) { return this.getAttribute("data-testid") === "msg-markdown-long" ? 1800 : 0; },
    });
    scrollTop = 0;
    Object.defineProperty(HTMLElement.prototype, "scrollTop", {
      configurable: true,
      get(this: HTMLElement) { return this.getAttribute("data-testid") === "messages-scroll" ? scrollTop : 0; },
      set(this: HTMLElement, value: number) { if (this.getAttribute("data-testid") === "messages-scroll") scrollTop = value; },
    });
  });

  afterEach(() => {
    const proto = HTMLElement.prototype as unknown as Record<string, unknown>;
    delete proto.scrollHeight;
    delete proto.scrollTop;
    vi.unstubAllGlobals();
  });

  it("[REQ:P0-017d] ends a tall row in an outline footer and never expands it inline", () => {
    render(<MessagesPane {...props} />);

    expect(screen.getByTestId("msg-open-reader-long")).toHaveTextContent(strings.messagesPane.readerFooter);
    expect(screen.queryByTestId("msg-open-reader-short")).toBeNull();
    expect(screen.queryByTestId(/^msg-collapse-/)).toBeNull();
  });

  it("[REQ:P0-017d] opens inside the pane, over a list it takes out of reach, with back, find, copy and More as its header and the speaker, time and number heading the text", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    // In the pane rather than a page overlay, so the composer below stays usable.
    const reader = within(screen.getByTestId("messages-pane-sess-1")).getByTestId("messages-reader");
    expect(reader).not.toHaveAttribute("role", "dialog");
    expect(reader).toHaveAttribute("aria-label", strings.messagesPane.speaker.claude);
    expect(screen.getByTestId("messages-scroll")).toHaveAttribute("inert");

    const header = within(reader).getByTestId("reader-header");
    for (const id of ["reader-back", "reader-find-input", "reader-copy", "reader-more"]) {
      expect(within(header).getByTestId(id)).toBeInTheDocument();
    }
    expect(header).not.toHaveTextContent(strings.messagesPane.speaker.claude);
    expect(within(reader).queryByTestId("reader-play")).toBeNull();

    const meta = within(reader).getByTestId("reader-meta");
    expect(meta).toHaveTextContent(strings.messagesPane.speaker.claude);
    expect(meta).toHaveTextContent("#2");
    expect(within(reader).getByTestId("messages-reader-body")).toHaveTextContent("alpha three.");
  });

  it("[REQ:P0-017d] finds in the message: highlights, counts, and steps through matches", async () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    fireEvent.change(screen.getByTestId("reader-find-input"), { target: { value: "alpha" } });
    const count = screen.getByTestId("reader-match-count");
    await waitFor(() => { expect(count).toHaveAttribute("data-total", "3"); });
    expect(count).toHaveAttribute("data-current", "1");
    const body = screen.getByTestId("messages-reader-body");
    expect(body.querySelectorAll("mark")).toHaveLength(3);
    expect(body.querySelector("mark[data-find-match='active']")?.textContent).toBe("alpha");

    fireEvent.click(screen.getByTestId("reader-find-next"));
    expect(count).toHaveAttribute("data-current", "2");
    fireEvent.click(screen.getByTestId("reader-find-prev"));
    fireEvent.click(screen.getByTestId("reader-find-prev"));
    expect(count).toHaveAttribute("data-current", "3");

    fireEvent.change(screen.getByTestId("reader-find-input"), { target: { value: "" } });
    await waitFor(() => { expect(body.querySelectorAll("mark")).toHaveLength(0); });
    expect(body).toHaveTextContent("alpha one. beta two. alpha three.");
  });

  it("[REQ:P0-017d] offers match stepping only once there is a query", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    expect(screen.queryByTestId("reader-find-prev")).toBeNull();
    expect(screen.queryByTestId("reader-find-next")).toBeNull();
    fireEvent.change(screen.getByTestId("reader-find-input"), { target: { value: "alpha" } });
    expect(screen.getByTestId("reader-find-prev")).toBeInTheDocument();
    expect(screen.getByTestId("reader-find-next")).toBeInTheDocument();
  });

  it("[REQ:P0-017d] steps to the previous and next reply", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    const meta = () => screen.getByTestId("reader-meta");

    expect(screen.getByTestId("reader-next-reply")).toBeDisabled();
    fireEvent.click(screen.getByTestId("reader-prev-reply"));
    expect(meta()).toHaveTextContent("#1");
    expect(screen.getByTestId("messages-reader-body")).toHaveTextContent("A short reply.");
    expect(screen.getByTestId("reader-prev-reply")).toBeDisabled();

    fireEvent.click(screen.getByTestId("reader-next-reply"));
    expect(meta()).toHaveTextContent("#2");
    expect(screen.getByTestId("messages-reader-body")).toHaveTextContent("alpha three.");
  });

  it("[REQ:P0-017d] marks the next-reply button when a reply arrives while reading, and clears it once read", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    expect(screen.getByTestId("reader-next-reply")).toBeDisabled();

    act(() => { seed([event("short", 1, "A short reply."), event("long", 2, LONG), event("fresh", 3, "A fresh reply.")]); });
    const next = screen.getByTestId("reader-next-reply");
    expect(next).toBeEnabled();
    expect(next).toHaveAttribute("data-new-count", "1");
    expect(next).toHaveAttribute("aria-label", strings.reader.nextReplyNew);

    fireEvent.click(next);
    expect(screen.getByTestId("messages-reader-body")).toHaveTextContent("A fresh reply.");
    expect(screen.getByTestId("reader-next-reply")).not.toHaveAttribute("data-new-count");
  });

  it("[REQ:P0-017d] resizes its text with the size buttons and a pinch, and keeps the size", () => {
    const { unmount } = render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    const body = () => screen.getByTestId("messages-reader-body");
    const start = parseFloat(body().style.fontSize);

    fireEvent.click(screen.getByTestId("reader-font-larger"));
    expect(parseFloat(body().style.fontSize)).toBeGreaterThan(start);
    fireEvent.click(screen.getByTestId("reader-font-smaller"));
    expect(parseFloat(body().style.fontSize)).toBe(start);

    // Two fingers spreading to twice their distance double the size.
    fireEvent.touchStart(body(), { touches: [{ identifier: 0, clientX: 0, clientY: 0 }, { identifier: 1, clientX: 0, clientY: 100 }] });
    fireEvent.touchMove(body(), { touches: [{ identifier: 0, clientX: 0, clientY: 0 }, { identifier: 1, clientX: 0, clientY: 200 }] });
    fireEvent.touchEnd(body(), { touches: [] });
    expect(parseFloat(body().style.fontSize)).toBe(start * 2);

    unmount();
    useMessagesViewStore.setState({ readers: {} });
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    expect(parseFloat(body().style.fontSize)).toBe(start * 2);
  });

  it("[REQ:P0-017d] hides its footer while the phone keyboard is up", () => {
    const touchPoints = Object.getOwnPropertyDescriptor(Navigator.prototype, "maxTouchPoints");
    Object.defineProperty(navigator, "maxTouchPoints", { configurable: true, value: 1 });
    try {
      useWorkspaceStore.setState({ keyboardOpen: true });
      render(<MessagesPane {...props} />);
      fireEvent.click(screen.getByTestId("msg-open-reader-long"));

      expect(screen.queryByTestId("reader-font-larger")).toBeNull();
      act(() => { useWorkspaceStore.setState({ keyboardOpen: false }); });
      expect(screen.getByTestId("reader-font-larger")).toBeInTheDocument();
    } finally {
      delete (navigator as unknown as Record<string, unknown>).maxTouchPoints;
      if (touchPoints) Object.defineProperty(Navigator.prototype, "maxTouchPoints", touchPoints);
    }
  });

  it("[REQ:P0-017c] on touch, an action that moves focus on (staging into the composer) keeps it after its sheet closes, and the reader stays open", async () => {
    const touchPoints = Object.getOwnPropertyDescriptor(Navigator.prototype, "maxTouchPoints");
    Object.defineProperty(navigator, "maxTouchPoints", { configurable: true, value: 1 });
    const bar = document.createElement("textarea");
    document.body.appendChild(bar);
    try {
      // A modal sheet makes the rest of the page inert, and focus on an inert
      // field silently fails (jsdom does not enforce that, so it is asserted).
      let barInertWhenStaged: boolean | null = null;
      const onSendToComposer = vi.fn(() => {
        barInertWhenStaged = bar.inert || bar.closest("[inert]") !== null;
        bar.focus();
      });
      render(<MessagesPane {...props} onSendToComposer={onSendToComposer} />);
      fireEvent.click(screen.getByTestId("msg-open-reader-long"));
      fireEvent.click(screen.getByTestId("reader-more"));
      fireEvent.click(within(screen.getByTestId("msg-action-sheet")).getByTestId("msg-send-to-composer-long"));

      expect(onSendToComposer).toHaveBeenCalledWith(LONG);
      expect(barInertWhenStaged).toBe(false);
      await waitFor(() => { expect(screen.queryByTestId("msg-action-sheet")).toBeNull(); });
      expect(document.activeElement).toBe(bar);
      expect(screen.getByTestId("messages-reader")).toBeInTheDocument();
    } finally {
      bar.remove();
      delete (navigator as unknown as Record<string, unknown>).maxTouchPoints;
      if (touchPoints) Object.defineProperty(Navigator.prototype, "maxTouchPoints", touchPoints);
    }
  });

  it("[REQ:P0-017c] the More button lists the message's other actions, play among them", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    fireEvent.click(screen.getByTestId("reader-more"));
    const menu = screen.getByTestId("msg-actions-menu-long");
    expect(within(menu).getByTestId("msg-render-toggle-long")).toBeInTheDocument();
    expect(within(menu).queryByTestId("msg-open-in-reader-long")).toBeNull();
    fireEvent.click(within(menu).getByTestId("msg-play-long"));
    expect(onPlayEvent).toHaveBeenCalledWith("long");
  });

  it("[REQ:P0-017d] copies the full text", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    fireEvent.click(screen.getByTestId("reader-copy"));
    expect(writeText).toHaveBeenCalledWith(LONG);
  });

  it("[REQ:P0-017d] Escape and the back button return to the list where it was, focus on the row", async () => {
    render(<MessagesPane {...props} />);
    scrollTop = 120;
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    fireEvent.keyDown(screen.getByTestId("messages-reader"), { key: "Escape" });
    await waitFor(() => { expect(screen.queryByTestId("messages-reader")).toBeNull(); });
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("msg-card-long")); });
    expect(screen.getByTestId("messages-scroll")).not.toHaveAttribute("inert");
    expect(scrollTop).toBe(120);

    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    fireEvent.click(screen.getByTestId("reader-back"));
    expect(screen.queryByTestId("messages-reader")).toBeNull();
  });

  it("[REQ:P0-017d] keeps the open reader when the pane is remounted, as a tab switch does", () => {
    const { unmount } = render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    unmount();

    render(<MessagesPane {...props} />);
    expect(screen.getByTestId("reader-meta")).toHaveTextContent("#2");
  });

  it("[REQ:P0-017d] the archive's reader is its own: it never opens the live pane's", () => {
    render(<MessagesPane {...props} readOnly />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    expect(screen.getByTestId("messages-reader")).toBeInTheDocument();
    expect(useMessagesViewStore.getState().readers).toEqual({});
  });
});
