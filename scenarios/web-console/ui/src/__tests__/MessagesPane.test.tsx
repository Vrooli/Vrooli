import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { ConversationEvent } from "../lib/api";
import type { TTSPlaybackState } from "../hooks/tts/types";

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: vi.fn().mockResolvedValue(undefined),
}));

// Mock the markdown renderer to avoid shiki/mermaid in jsdom
vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({
    content,
    onLinkClick,
  }: {
    content: string;
    onLinkClick?: (href: string, event: unknown) => void;
  }) => (
    <div data-testid="mock-markdown">
      {content.includes("[")
        ? (
          <a
            href="/tmp/example.ts:12"
            data-testid="mock-markdown-link"
            onClick={(event) => onLinkClick?.("/tmp/example.ts:12", event)}
          >
            example.ts
          </a>
        )
        : content}
    </div>
  ),
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

const defaultPlaybackState: TTSPlaybackState = {
  currentTime: 0,
  duration: null,
  isPaused: true,
  playbackRate: 1,
  volume: 1,
  isMuted: false,
  capabilities: {
    canPause: true,
    canSeek: false,
    canAdjustSpeed: true,
    canAdjustVolume: true,
  },
};

const defaultProps = {
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
  playbackState: defaultPlaybackState,
  onSetPlaybackRate: vi.fn(),
  onSetVolume: vi.fn(),
  onSetMuted: vi.fn(),
  playbackFocusRequest: null,
};

describe("MessagesPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useConversationStore.setState({ sessions: {}, viewModes: {} });
    globalThis.fetch = vi.fn() as typeof fetch;
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

  it("clicking 'read from here' calls onPlayFromHere with correct event ID", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
    expect(defaultProps.onPlayFromHere).toHaveBeenCalledWith("e1");
  });

  it("clicking audio button calls onPlayEvent and opens popover", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    expect(defaultProps.onPlayEvent).toHaveBeenCalledWith("e1");
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

  it("search updates row state without pushing query into markdown renderer props", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "world" },
    });

    const mdEl = screen.getByTestId("mock-markdown");
    expect(mdEl.getAttribute("data-search-query")).toBeNull();
  });

  it("opens the file viewer when clicking a file-like markdown link", async () => {
    (globalThis.fetch as unknown as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          input_path: "/tmp/example.ts:12",
          resolved_path: "/tmp/example.ts",
          line: 12,
          exists: true,
          resolution_basis: "absolute_allowed",
          category: "code",
          can_preview: true,
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          path: "/tmp/example.ts",
          line: 12,
          category: "code",
          content_type: "text/plain; charset=utf-8",
          content: "const x = 1;\n",
          truncated: false,
        }),
      });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[example.ts](/tmp/example.ts:12)" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("mock-markdown-link"));

    await waitFor(() => {
      expect(screen.getAllByText("example.ts").length).toBeGreaterThan(0);
      expect(screen.getByText("/tmp/example.ts")).toBeInTheDocument();
      expect(screen.getByText("line 12")).toBeInTheDocument();
      expect(screen.getByText("const x = 1;")).toBeInTheDocument();
    });
    expect(globalThis.fetch).toHaveBeenLastCalledWith(
      expect.stringContaining("path=%2Ftmp%2Fexample.ts%3A12"),
      expect.any(Object),
    );
  });

  it("shows a viewer error when file resolution fails", async () => {
    (globalThis.fetch as unknown as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: () => Promise.resolve({
          error: "Referenced file was not found",
          code: "file_reference_not_found",
          category: "validation",
        }),
      });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[missing.ts](missing.ts)" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("mock-markdown-link"));

    await waitFor(() => {
      expect(screen.getByText("File preview unavailable")).toBeInTheDocument();
      expect(screen.getByText("Referenced file was not found")).toBeInTheDocument();
    });
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
    const scrollToMock = vi.fn();
    Element.prototype.scrollTo = scrollToMock;

    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-nav-down"));
    expect(scrollToMock).toHaveBeenCalled();
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

  it("does not render the old summarized badge", () => {
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

    expect(screen.queryByTestId("msg-summarized-badge-e1")).toBeNull();
  });

  it("mode control dropdown shows Original + level options for summarized events", () => {
    seedEvents([
      makeEvent({
        id: "e1",
        sequence: 1,
        summarized: true,
        speechParagraphs: ["Short version"],
        originalSpeechParagraphs: ["Full original text"],
      }),
    ]);
    render(
      <MessagesPane
        {...defaultProps}
        selectedVersionForEvent={vi.fn(() => "active" as const)}
      />,
    );

    fireEvent.click(screen.getByTestId("msg-e1-mode-control"));
    expect(screen.getByTestId("msg-e1-mode-option-original")).toBeInTheDocument();
    expect(screen.getByTestId("msg-e1-mode-option-light")).toBeInTheDocument();
    expect(screen.getByTestId("msg-e1-mode-option-moderate")).toBeInTheDocument();
    expect(screen.getByTestId("msg-e1-mode-option-heavy")).toBeInTheDocument();
  });

  it("shows summarize error from the playback controller surface", async () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(
      <MessagesPane
        {...defaultProps}
        getSummarizeError={vi.fn(() => "Summarization failed: ollama returned 404: model not found")}
      />,
    );

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
      expect(screen.getByTestId("msg-summarize-error-e1").textContent).toContain("model not found");
    });
  });

  it("clears summarize error through the provided dismiss handler", async () => {
    const onClearSummarizeError = vi.fn();
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(
      <MessagesPane
        {...defaultProps}
        getSummarizeError={vi.fn(() => "Summarization failed: connection refused")}
        onClearSummarizeError={onClearSummarizeError}
      />,
    );

    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("msg-clear-summarize-error-e1"));
    expect(onClearSummarizeError).toHaveBeenCalledWith("e1");
  });

  it("applies a playback focus request by scrolling the targeted event into view", () => {
    const scrollToMock = vi.fn();
    Element.prototype.scrollTo = scrollToMock;

    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
    ]);
    render(
      <MessagesPane
        {...defaultProps}
        playbackFocusRequest={{ eventId: "e2", nonce: 1 }}
      />,
    );

    expect(scrollToMock).toHaveBeenCalled();
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
