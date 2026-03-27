import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { ConversationEvent } from "../lib/api";

// Mock the markdown renderer to avoid shiki/mermaid in jsdom
vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({ content, searchQuery, isSearchFocused }: { content: string; searchQuery?: string; isSearchFocused?: boolean }) => (
    <div data-testid="mock-markdown" data-search-query={searchQuery || ""} data-search-focused={String(!!isSearchFocused)}>
      {content}
    </div>
  ),
}));

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual<typeof import("../lib/api")>("../lib/api");
  return {
    ...actual,
    summarizeEvent: vi.fn(),
  };
});

import { summarizeEvent } from "../lib/api";
const mockSummarizeEvent = vi.mocked(summarizeEvent);

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

const defaultProps = {
  sessionId: "sess-1",
  onSpeakFromHere: vi.fn(),
  onSpeakOne: vi.fn(),
  activeSpeakingEventId: null,
  isTtsSpeaking: false,
};

describe("MessagesPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useConversationStore.setState({ sessions: {}, viewModes: {} });
    // Mock IntersectionObserver for auto-scroll sentinel
    const mockObserver = vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      unobserve: vi.fn(),
      disconnect: vi.fn(),
    }));
    vi.stubGlobal("IntersectionObserver", mockObserver);
    // Mock ResizeObserver for collapse measurement
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      unobserve: vi.fn(),
      disconnect: vi.fn(),
    })));
  });

  function seedEvents(events: ConversationEvent[]) {
    useConversationStore.setState({
      sessions: {
        "sess-1": {
          events,
          cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
          hydrated: true,
        },
      },
    });
  }

  // --- Core rendering ---

  it("renders play and audio icons on each assistant message", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-speak-from-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-audio-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-audio-e2")).toBeInTheDocument();
  });

  it("clicking 'read from here' calls onSpeakFromHere with correct event ID", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
    expect(defaultProps.onSpeakFromHere).toHaveBeenCalledWith("e1");
  });

  it("clicking audio button calls onSpeakOne and opens popover", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    expect(defaultProps.onSpeakOne).toHaveBeenCalledWith("e1", "Hello world", ["Hello world"], { version: "original" });
    expect(screen.getByTestId("audio-popover-e1")).toBeInTheDocument();
  });

  it("active speaking event shows TTS accent border", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ]);
    render(
      <MessagesPane
        {...defaultProps}
        activeSpeakingEventId="e2"
        isTtsSpeaking={true}
      />,
    );

    const activeCard = screen.getByTestId("msg-card-e2");
    expect(activeCard.className).toContain("border-l-wc-accent");

    const inactiveCard = screen.getByTestId("msg-card-e1");
    expect(inactiveCard.className).not.toContain("border-l-wc-accent");
  });

  it("empty state renders without speaker icons", () => {
    seedEvents([]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId(/msg-speak-/)).toBeNull();
    expect(screen.getByText(/No conversation events/)).toBeInTheDocument();
  });

  it("displays correct source labels", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, source: "claude_hook" }),
      makeEvent({ id: "e2", sequence: 2, source: "codex" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByText("Claude Code")).toBeInTheDocument();
    expect(screen.getByText("Codex")).toBeInTheDocument();
  });

  it("user messages have no TTS controls", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "My question" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "My answer" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("msg-speak-from-e1")).toBeNull();
    expect(screen.queryByTestId("msg-audio-e1")).toBeNull();
    expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-audio-e2")).toBeInTheDocument();
  });

  // --- Layout: full-width accent bars ---

  it("messages use accent bar layout with role-based colors", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    const userCard = screen.getByTestId("msg-card-e1");
    const assistantCard = screen.getByTestId("msg-card-e2");

    // Both have 3px left border
    expect(userCard.className).toContain("border-l-[3px]");
    expect(assistantCard.className).toContain("border-l-[3px]");

    // Different colors
    expect(userCard.className).toContain("border-l-sky");
    expect(assistantCard.className).toContain("border-l-emerald");
  });

  it("focused message gets accent background highlight", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Not focused initially
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("bg-wc-accent");
  });

  // --- Markdown rendering ---

  it("renders markdown content through MarkdownRenderer", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello World" })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-markdown-e1")).toBeInTheDocument();
    expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
    expect(screen.getByText("Hello World")).toBeInTheDocument();
  });

  it("passes search query to MarkdownRenderer", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "world" },
    });

    const mdEl = screen.getByTestId("mock-markdown");
    expect(mdEl.getAttribute("data-search-query")).toBe("world");
  });

  // --- Font size ---

  it("applies font size from workspace store to message content", () => {
    useWorkspaceStore.setState({
      panes: [{ sessionId: "sess-1", name: "test", headerColor: "transparent", themeId: "slate-ocean", fontSize: 20, groupId: null, supportsMessagesView: true }],
    });
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Sized text" })]);
    render(<MessagesPane {...defaultProps} />);

    const mdWrapper = screen.getByTestId("msg-markdown-e1");
    expect(mdWrapper.style.fontSize).toBe("20px");
  });

  // --- Control strip ---

  it("renders control strip with search, jump, and nav buttons", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-control-strip")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-btn")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-trigger")).toBeInTheDocument();
    expect(screen.getByTestId("messages-nav-up")).toBeInTheDocument();
    expect(screen.getByTestId("messages-nav-down")).toBeInTheDocument();
  });

  it("clicking search button opens the search drawer", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("messages-search-panel")).toBeNull();
    fireEvent.click(screen.getByTestId("messages-search-btn"));
    expect(screen.getByTestId("messages-search-panel")).toBeInTheDocument();
  });

  it("disables nav chevrons when no events exist", () => {
    seedEvents([]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-nav-up")).toBeDisabled();
    expect(screen.getByTestId("messages-nav-down")).toBeDisabled();
  });

  it("enables nav chevrons when events exist", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, role: "assistant", text: "Answer" })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-nav-up")).not.toBeDisabled();
    expect(screen.getByTestId("messages-nav-down")).not.toBeDisabled();
  });

  it("down chevron navigates to next message", () => {
    const scrollIntoViewMock = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoViewMock;

    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-nav-down"));
    expect(scrollIntoViewMock).toHaveBeenCalledTimes(1);
  });

  // --- Search ---

  it("non-matching messages are dimmed during search", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
      makeEvent({ id: "e2", sequence: 2, text: "Goodbye" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "Hello" },
    });

    // e1 matches, e2 does not
    expect(screen.getByTestId("msg-card-e2").className).toContain("opacity-40");
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("opacity-40");
  });

  it("closing search clears search state", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "world" },
    });

    fireEvent.click(screen.getByTestId("messages-search-close"));

    // Search panel should be gone
    expect(screen.queryByTestId("messages-search-panel")).toBeNull();
    // Messages should not be dimmed
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("opacity-40");
  });

  // --- Jump list ---

  it("jump trigger shows message count", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    const trigger = screen.getByTestId("msg-jump-trigger");
    expect(trigger.textContent).toContain("2");
  });

  it("clicking jump trigger opens jump list", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("msg-jump-list")).toBeNull();
    fireEvent.click(screen.getByTestId("msg-jump-trigger"));
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
  });

  it("jump list shows all messages with truncated text", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "Short user message" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "A very long assistant response that should be truncated in the jump list to save space" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-jump-trigger"));
    expect(screen.getByTestId("msg-jump-item-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e2")).toBeInTheDocument();
  });

  it("jump trigger is disabled when no events", () => {
    seedEvents([]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-jump-trigger")).toBeDisabled();
  });

  // --- Summarization ---

  it("summarized badge appears for summarized events", () => {
    seedEvents([
      makeEvent({
        id: "e1",
        sequence: 1,
        summarized: true,
        speechParagraphs: ["Short version"],
        originalSpeechParagraphs: ["Full original text that is much longer"],
      }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-summarized-badge-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-summarized-badge-e1").textContent).toBe("Summarized");
  });

  it("audio popover shows summarization toggle for summarized events", () => {
    seedEvents([
      makeEvent({
        id: "e1",
        sequence: 1,
        summarized: true,
        speechParagraphs: ["Short version"],
        originalSpeechParagraphs: ["Full original text"],
      }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    expect(screen.getByTestId("msg-play-summarized-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-play-original-e1")).toBeInTheDocument();
  });

  it("shows error when on-demand summarize returns an error", async () => {
    mockSummarizeEvent.mockResolvedValue({
      summarized: false,
      error: "Summarization failed: ollama returned 404: model not found",
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    expect(screen.getByTestId("audio-popover-e1")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
      expect(screen.getByTestId("msg-summarize-error-e1").textContent).toContain("model not found");
    });
  });

  it("clears summarize error when retrying successfully", async () => {
    mockSummarizeEvent.mockResolvedValueOnce({
      summarized: false,
      error: "Summarization failed: connection refused",
    });
    mockSummarizeEvent.mockResolvedValueOnce({
      summarized: true,
      speechParagraphs: ["Short summary"],
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    await waitFor(() => {
      expect(screen.queryByTestId("msg-summarize-error-e1")).toBeNull();
    });
  });

  // --- Copy-to-clipboard ---

  it("renders copy button on both user and assistant messages", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User msg" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant msg" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-copy-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-copy-e2")).toBeInTheDocument();
  });

  it("clicking copy writes message text to clipboard", () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: writeTextMock } });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-copy-e1"));
    expect(writeTextMock).toHaveBeenCalledWith("Copy me");
  });

  it("shows checkmark icon after copying", () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: writeTextMock } });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-copy-e1"));

    const btn = screen.getByTestId("msg-copy-e1");
    const svg = btn.querySelector("svg");
    expect(svg?.classList.toString()).toContain("text-green-400");
  });
});
