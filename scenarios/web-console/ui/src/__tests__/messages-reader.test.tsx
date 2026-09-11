import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
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

const LONG = "# Findings\n\nalpha one. beta two. alpha three.\n\n## More\n\n```ts\nconst alpha = 1;\n```\n";

function event(id: string, sequence: number, text: string): ConversationEvent {
  return {
    id, sessionId: "sess-1", sequence, source: "claude_hook", role: "assistant", text, speechParagraphs: [text],
    summarized: false, createdAt: new Date().toISOString(), deliveryState: "received", ttsState: "idle", consumptionState: "seen",
  };
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
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({
      sessions: {
        "sess-1": createConversationSessionState({
          events: [event("short", 1, "A short reply."), event("long", 2, LONG)],
          cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
          hydrated: true,
        }),
      },
    });
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

  it("[REQ:P0-017d] opens the reader with speaker, time, sequence, and the full message", () => {
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    const reader = screen.getByTestId("messages-reader");
    expect(reader).toHaveTextContent(strings.messagesPane.speaker.claude);
    expect(reader).toHaveTextContent("#2");
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

  it("[REQ:P0-017d] copies the full text and plays the message", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<MessagesPane {...props} />);
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));

    fireEvent.click(screen.getByTestId("reader-copy"));
    expect(writeText).toHaveBeenCalledWith(LONG);
    fireEvent.click(screen.getByTestId("reader-play"));
    expect(onPlayEvent).toHaveBeenCalledWith("long");
  });

  it("[REQ:P0-017d] closing returns focus to the row and leaves the list where it was", async () => {
    render(<MessagesPane {...props} />);
    scrollTop = 120;
    fireEvent.click(screen.getByTestId("msg-open-reader-long"));
    expect(screen.getByTestId("messages-pane-sess-1")).toBeInTheDocument();

    fireEvent.keyDown(screen.getByTestId("messages-reader"), { key: "Escape" });
    await waitFor(() => { expect(screen.queryByTestId("messages-reader")).toBeNull(); });
    await waitFor(() => { expect(document.activeElement).toBe(screen.getByTestId("msg-card-long")); });
    expect(scrollTop).toBe(120);
  });
});
