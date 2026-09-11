import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { ConversationEvent } from "../api/conversation";
import { makeConversationEvents } from "./fixtures/conversationFixture";

const { mockLoadOlderConversationPage, mockLoadConversationPageContaining, mockRefreshConversationSession } = vi.hoisted(() => ({
  mockLoadOlderConversationPage: vi.fn().mockResolvedValue(false),
  mockLoadConversationPageContaining: vi.fn().mockResolvedValue(false),
  mockRefreshConversationSession: vi.fn().mockResolvedValue({ ok: true, addedEvents: 0 }),
}));

vi.mock("../hooks/useConversationSession", () => ({
  refreshConversationSession: mockRefreshConversationSession,
  loadOlderConversationPage: mockLoadOlderConversationPage,
  loadConversationPageContaining: mockLoadConversationPageContaining,
}));

const { mockResolveFilePreview, mockGetFilePreviewText } = vi.hoisted(() => ({
  mockResolveFilePreview: vi.fn(),
  mockGetFilePreviewText: vi.fn(),
}));

vi.mock("../api/filePreview", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/filePreview")>();
  return {
    ...actual,
    resolveFilePreview: mockResolveFilePreview,
    getFilePreviewText: mockGetFilePreviewText,
  };
});

// makeModel builds a PreviewModel fixture for the file-preview controller.
function makeModel(overrides: Record<string, unknown>) {
  return {
    previewId: "pv-1",
    inputPath: "/tmp/example.ts",
    resolvedPath: "/tmp/example.ts",
    basename: "example.ts",
    resolutionBasis: "absolute",
    kind: "code",
    mimeType: "text/plain; charset=utf-8",
    sizeBytes: 12,
    canPreview: true,
    canDownload: true,
    supportsRange: false,
    textContentAvailable: true,
    blobUrl: "/api/v1/sessions/sess-1/file-previews/pv-1/blob",
    blobHref: "/api/v1/sessions/sess-1/file-previews/pv-1/blob",
    warnings: [],
    ...overrides,
  };
}

// Mock the markdown renderer to avoid shiki/mermaid in jsdom
// Search runs on the server only; this stands in for it the way the server
// answers (whole-history hits with ranges), after the pane's 200 ms debounce.
vi.mock("../api/conversation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/conversation")>();
  return {
    ...actual,
    searchConversation: vi.fn((_sessionId: string, query: string) => Promise.resolve(
      query === "Hello"
        ? { matches: [{ eventId: "e1", sequence: 1, excerpt: "Hello world", ranges: [{ start: 0, end: 5 }], role: "assistant" as const, createdAt: "2026-09-11T08:00:00Z" }], truncated: false, totalMatches: 1 }
        : { matches: [], truncated: false, totalMatches: 0 },
    )),
  };
});

vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({
    content,
    onLinkClick,
    onMermaidOpen,
  }: {
    content: string;
    onLinkClick?: (href: string, event: unknown) => void;
    onMermaidOpen?: (code: string) => void;
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
      <button data-testid="mock-mermaid-open" type="button" onClick={() => onMermaidOpen?.("graph TD; A-->B")}>
        open diagram
      </button>
    </div>
  ),
}));

// Keep the real Mermaid viewer/drawer but avoid loading the mermaid bundle.
vi.mock("../components/markdown/hooks/useMermaidSvg", () => ({
  useMermaidSvg: () => ({ svgHtml: '<svg viewBox="0 0 100 80"></svg>', error: null, loading: false }),
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
  playbackFocusRequest: null,
};

/** Actions are hidden at rest: hovering (fine pointer) reveals the inline cluster. */
function inlineAction(eventId: string, testId: string): HTMLElement {
  fireEvent.mouseEnter(screen.getByTestId(`msg-card-${eventId}`));
  return screen.getByTestId(testId);
}

/** Every applicable action is in the row's action list, opened from the ⋯ control. */
function overflowAction(eventId: string, testId: string): HTMLElement {
  const existing = screen.queryByTestId(testId);
  if (existing && !existing.closest("[data-testid='msg-actions-inline']")) return existing;
  fireEvent.mouseEnter(screen.getByTestId(`msg-card-${eventId}`));
  fireEvent.click(screen.getByTestId(`msg-actions-more-${eventId}`));
  return screen.getByTestId(testId);
}

describe("MessagesPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockLoadOlderConversationPage.mockResolvedValue(false);
    mockLoadConversationPageContaining.mockResolvedValue(false);
    useConversationStore.setState({ sessions: {} });
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
        "sess-1": createConversationSessionState({
          events,
          cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
          hydrated: true,
        }),
      },
    });
  }

  // --- Core rendering ---

  it("renders a bounded first window for 2500 conversation events", () => {
    seedEvents(makeConversationEvents(2500, 42));
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getAllByTestId(/^msg-card-/).length).toBeLessThanOrEqual(60);
  });

  it("offers read-from-here on each assistant message and no per-message audio settings", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(inlineAction("e1", "msg-speak-from-e1")).toBeInTheDocument();
    expect(inlineAction("e2", "msg-speak-from-e2")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("msg-actions-more-e2"));
    expect(screen.queryByTestId("msg-audio-e2")).toBeNull();
    expect(screen.queryByTestId(/^audio-popover-/)).toBeNull();
  });

  it("read-only mode hides transcript-mutating controls and can stage a message", () => {
    const onSendToComposer = vi.fn();
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Reusable context" })]);
    render(<MessagesPane {...defaultProps} readOnly onSendToComposer={onSendToComposer} />);

    expect(inlineAction("e1", "msg-copy-e1")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-speak-from-e1")).not.toBeInTheDocument();
    expect(overflowAction("e1", "msg-render-toggle-e1")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-field")).toBeInTheDocument();

    fireEvent.click(overflowAction("e1", "msg-send-to-composer-e1"));
    expect(onSendToComposer).toHaveBeenCalledWith("Reusable context");
  });

  it("loads the page containing an archive hit before focusing it", async () => {
    mockLoadConversationPageContaining.mockImplementation(async () => {
      seedEvents([makeEvent({ id: "target", sequence: 42 })]);
      return true;
    });
    render(<MessagesPane {...defaultProps} readOnly focusEventId="target" focusSequence={42} />);

    await waitFor(() => expect(mockLoadConversationPageContaining).toHaveBeenCalledWith("sess-1", 42));
  });

  it("clicking 'read from here' calls onPlayFromHere with correct event ID", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(inlineAction("e1", "msg-speak-from-e1"));
    expect(defaultProps.onPlayFromHere).toHaveBeenCalledWith("e1");
  });

  it("shows loading feedback on the active message audio control", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world", speechParagraphs: ["Hello world"] })]);
    render(<MessagesPane {...defaultProps} loadingEventId="e1" />);

    expect(inlineAction("e1", "msg-speak-from-e1")).toBeDisabled();
    expect(screen.getByTestId("msg-audio-loading-e1")).toBeInTheDocument();
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
    // A seeded session with a healthy capture status and no events is the one
    // case that genuinely means "nothing said yet".
    expect(screen.getByTestId("messages-state-empty")).toBeInTheDocument();
    expect(screen.getByText(strings.messagesPane.state.emptyTitle)).toBeInTheDocument();
  });

  it("user messages have no TTS controls", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "My question" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "My answer" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.mouseEnter(screen.getByTestId("msg-card-e1"));
    expect(screen.queryByTestId("msg-speak-from-e1")).toBeNull();
    expect(inlineAction("e2", "msg-speak-from-e2")).toBeInTheDocument();
  });

  // --- Layout: speaker and time, no role colour bars ---

  it("names the speaker on every row instead of colour-coding roles", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("msg-speaker-e1")).toHaveTextContent(strings.messagesPane.speaker.you);
    expect(screen.getByTestId("msg-speaker-e2")).toHaveTextContent(strings.messagesPane.speaker.claude);
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("border-l-sky");
    expect(screen.getByTestId("msg-card-e2").className).not.toContain("border-l-emerald");
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

  it("toggles a message between markdown and plain text views", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "# Heading" })]);
    render(<MessagesPane {...defaultProps} />);

    // Markdown by default
    expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-plaintext-e1")).toBeNull();

    const toggle = overflowAction("e1", "msg-render-toggle-e1");
    expect(toggle.getAttribute("aria-pressed")).toBe("false");

    fireEvent.click(toggle);
    expect(screen.queryByTestId("mock-markdown")).toBeNull();
    const plain = screen.getByTestId("msg-plaintext-e1");
    expect(plain).toBeInTheDocument();
    expect(plain.textContent).toBe("# Heading");
    expect(overflowAction("e1", "msg-render-toggle-e1").getAttribute("aria-pressed")).toBe("true");

    fireEvent.click(screen.getByTestId("msg-render-toggle-e1"));
    expect(screen.getByTestId("mock-markdown")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-plaintext-e1")).toBeNull();
  });

  it("render mode toggle is independent per message", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "first" }),
      makeEvent({ id: "e2", sequence: 2, text: "second" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(overflowAction("e1", "msg-render-toggle-e1"));
    expect(screen.getByTestId("msg-plaintext-e1")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-plaintext-e2")).toBeNull();
  });

  it("opens the Mermaid viewer from a diagram open action and closes it", async () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "diagram here" })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull();

    fireEvent.click(screen.getAllByTestId("mock-mermaid-open")[0] as HTMLElement);

    const panel = screen.getByTestId("messages-mermaid-viewer-panel");
    expect(panel).toBeInTheDocument();
    expect(screen.getByText(strings.mermaid.viewerTitle)).toBeInTheDocument();
    expect(screen.getByLabelText(strings.mermaid.zoomIn)).toBeInTheDocument();
    expect(screen.getByLabelText(strings.mermaid.showSource)).toBeInTheDocument();

    fireEvent.pointerDown(screen.getByTestId("messages-mermaid-viewer-panel.backdrop"));
    await waitFor(() => expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull());
  });

  it("closes the Mermaid viewer with Escape", async () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "diagram here" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getAllByTestId("mock-mermaid-open")[0] as HTMLElement);
    expect(screen.getByTestId("messages-mermaid-viewer-panel")).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Escape" });
    await waitFor(() => expect(screen.queryByTestId("messages-mermaid-viewer-panel")).toBeNull());
  });

  it("search updates row state without pushing query into markdown renderer props", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Hello world" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-field"));
    fireEvent.change(screen.getByTestId("msg-nav-search"), {
      target: { value: "world" },
    });

    const mdEl = screen.getByTestId("mock-markdown");
    expect(mdEl.getAttribute("data-search-query")).toBeNull();
  });

  it("opens the file viewer when clicking a file-like markdown link", async () => {
    mockResolveFilePreview.mockResolvedValueOnce(
      makeModel({ inputPath: "/tmp/example.ts:12", line: 12, kind: "code" }),
    );
    mockGetFilePreviewText.mockResolvedValueOnce({
      resolvedPath: "/tmp/example.ts",
      kind: "code",
      mimeType: "text/plain; charset=utf-8",
      content: "const x = 1;\n",
      truncated: false,
      line: 12,
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[example.ts](/tmp/example.ts:12)" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("mock-markdown-link"));

    await waitFor(() => {
      expect(screen.getAllByText("example.ts").length).toBeGreaterThan(0);
      expect(screen.getByText("/tmp/example.ts")).toBeInTheDocument();
      expect(screen.getByText("messagesFileViewer.linePrefix")).toBeInTheDocument();
      expect(screen.getByText("const x = 1;")).toBeInTheDocument();
    });
    expect(screen.getByTestId("messages-file-viewer-panel").closest("[data-rcl-full-page-drawer]")).not.toBeNull();
    expect(mockResolveFilePreview).toHaveBeenCalledWith("sess-1", "/tmp/example.ts:12", "message_link");
    expect(mockGetFilePreviewText).toHaveBeenCalledWith("sess-1", "pv-1");
  });

  it("renders SVG file references as image previews", async () => {
    mockResolveFilePreview.mockResolvedValueOnce(
      makeModel({
        inputPath: "/tmp/logo.svg",
        resolvedPath: "/tmp/logo.svg",
        basename: "logo.svg",
        kind: "svg",
        mimeType: "image/svg+xml",
        textContentAvailable: false,
        supportsRange: true,
      }),
    );

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[logo](/tmp/logo.svg)" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("mock-markdown-link"));

    await waitFor(() => {
      expect(screen.getByRole("img", { name: "logo.svg" })).toBeInTheDocument();
    });
    expect(screen.getByText("/tmp/logo.svg")).toBeInTheDocument();
    expect(mockGetFilePreviewText).not.toHaveBeenCalled();
  });

  it("shows a viewer error when file resolution fails", async () => {
    mockResolveFilePreview.mockRejectedValueOnce(new Error("Referenced file was not found"));

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "[missing.ts](missing.ts)" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("mock-markdown-link"));

    await waitFor(() => {
      expect(screen.getByText("messagesFileViewer.unavailable")).toBeInTheDocument();
      expect(screen.getByText("Referenced file was not found")).toBeInTheDocument();
    });
  });

  // --- Font size ---

  it("applies font size from workspace store to message content", () => {
    useWorkspaceStore.setState({
      panes: [{ sessionId: "sess-1", name: "test", headerColor: "transparent", themeId: "slate-ocean", fontSize: 20, groupId: null, supportsMessagesView: true, manuallyUnread: false }],
    });
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Sized text" })]);
    render(<MessagesPane {...defaultProps} />);

    const mdWrapper = screen.getByTestId("msg-markdown-e1");
    expect(mdWrapper.style.fontSize).toBe("20px");
  });

  // --- Control strip ---

  it("renders the control strip with one search field", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-control-strip")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-field")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-trigger")).toBeNull();
  });

  it("renders an optional trailing action in the control strip", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(
      <MessagesPane
        {...defaultProps}
        toolbarTrailingAction={<button type="button" data-testid="messages-trailing-action">Toggle</button>}
      />,
    );

    expect(screen.getByTestId("messages-control-trailing")).toContainElement(
      screen.getByTestId("messages-trailing-action"),
    );
  });

  it("the search field opens the navigator focused on search", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("msg-jump-list")).toBeNull();
    fireEvent.click(screen.getByTestId("messages-search-field"));
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-search")).toBeInTheDocument();
  });

  // --- Search ---

  it("[REQ:P0-017g] non-matching messages are dimmed once the server answers", async () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
      makeEvent({ id: "e2", sequence: 2, text: "Goodbye" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-field"));
    fireEvent.change(screen.getByTestId("msg-nav-search"), {
      target: { value: "Hello" },
    });
    // Nothing is dimmed while the search is in flight: no row is judged
    // before the server has answered.
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("opacity-40");
    expect(screen.getByTestId("msg-card-e2").className).not.toContain("opacity-40");

    // e1 is a server hit, e2 is not — dimming applies to the message cards
    // behind the navigator overlay.
    await waitFor(() => { expect(screen.getByTestId("msg-card-e2").className).toContain("opacity-40"); });
    expect(screen.getByTestId("msg-card-e1").className).not.toContain("opacity-40");
  });

  it("clearing the navigator search removes message dimming", async () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
      makeEvent({ id: "e2", sequence: 2, text: "Goodbye" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-field"));
    fireEvent.change(screen.getByTestId("msg-nav-search"), {
      target: { value: "Hello" },
    });
    await waitFor(() => { expect(screen.getByTestId("msg-card-e2").className).toContain("opacity-40"); });

    // The navigator's clear button resets the lifted query.
    fireEvent.click(screen.getByTestId("msg-nav-clear"));
    expect(screen.getByTestId("msg-card-e2").className).not.toContain("opacity-40");
  });

  // --- Jump list ---

  it("the search field opens the navigator list", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.queryByTestId("msg-jump-list")).toBeNull();
    fireEvent.click(screen.getByTestId("messages-search-field"));
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
  });

  it("jump list shows all messages with truncated text", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "Short user message" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "A very long assistant response that should be truncated in the jump list to save space" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("messages-search-field"));
    expect(screen.getByTestId("msg-jump-item-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e2")).toBeInTheDocument();
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

    fireEvent.click(overflowAction("e1", "msg-e1-mode-control"));
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

    expect(inlineAction("e1", "msg-copy-e1")).toBeInTheDocument();
    expect(inlineAction("e2", "msg-copy-e2")).toBeInTheDocument();
  });

  it("clicking copy writes message text to clipboard", () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: writeTextMock } });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(inlineAction("e1", "msg-copy-e1"));
    expect(writeTextMock).toHaveBeenCalledWith("Copy me");
  });

  it("shows checkmark icon after copying", () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: writeTextMock } });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(inlineAction("e1", "msg-copy-e1"));

    const btn = screen.getByTestId("msg-copy-e1");
    const svg = btn.querySelector("svg");
    expect(svg?.classList.toString()).toContain("text-green-400");
  });

  // --- Scroll restore + jump-to-bottom ---

  describe("scroll follow + jump-to-bottom", () => {
    function seedManyEvents(n: number) {
      const events = Array.from({ length: n }, (_, i) => makeEvent({ id: `e${i + 1}`, sequence: i + 1 }));
      seedEvents(events);
      return events;
    }

    it("lands at the bottom on a fresh open", () => {
      const scrollToMock = vi.fn();
      Element.prototype.scrollTo = scrollToMock;
      Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
        configurable: true,
        get() { return 3000; },
      });

      seedManyEvents(60);
      render(<MessagesPane {...defaultProps} />);

      const wantedBottom = scrollToMock.mock.calls.some(
        ([arg]) => typeof arg === "object" && arg !== null && (arg as { top?: number }).top === 3000,
      );
      expect(wantedBottom).toBe(true);
    });

    it("jump-to-bottom button appears when not near bottom and there are no new messages", () => {
      // Force isNearBottom=false by stubbing scrollHeight/scrollTop/clientHeight so remaining > 200.
      Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
      Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
      Object.defineProperty(HTMLElement.prototype, "scrollTop", { configurable: true, get() { return 100; }, set() {} });

      seedManyEvents(40);
      render(<MessagesPane {...defaultProps} />);

      // A user gesture (wheel, then scroll) away from the bottom stops following.
      const container = screen.getByTestId("messages-scroll");
      fireEvent.wheel(container, { deltaY: -100 });
      fireEvent.scroll(container);

      expect(screen.getByTestId("msg-jump-bottom")).toBeInTheDocument();
    });

    it("jump-to-bottom button scrolls to bottom on click and disappears once near bottom", () => {
      const scrollToMock = vi.fn();
      Element.prototype.scrollTo = scrollToMock;
      let currentScrollTop = 100;
      Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 5000; } });
      Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
      Object.defineProperty(HTMLElement.prototype, "scrollTop", {
        configurable: true,
        get() { return currentScrollTop; },
        set(v: number) { currentScrollTop = v; },
      });

      seedManyEvents(40);
      render(<MessagesPane {...defaultProps} />);

      const container = screen.getByTestId("messages-scroll");
      fireEvent.wheel(container, { deltaY: -100 });
      fireEvent.scroll(container);

      const btn = screen.getByTestId("msg-jump-bottom");
      fireEvent.click(btn);

      expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ top: 5000 }));
    });

    it("does not follow new content after a user scroll moves away from the bottom", () => {
      const originalRaf = window.requestAnimationFrame;
      window.requestAnimationFrame = ((cb: FrameRequestCallback) => {
        cb(0);
        return 0;
      }) as typeof window.requestAnimationFrame;
      let currentScrollTop = 0;
      let currentScrollHeight = 5000;
      const scrollToMock = vi.fn((options?: ScrollToOptions | number) => {
        if (typeof options === "object" && options?.top != null) {
          currentScrollTop = options.top;
        }
      });
      Element.prototype.scrollTo = scrollToMock;
      Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return currentScrollHeight; } });
      Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
      Object.defineProperty(HTMLElement.prototype, "scrollTop", {
        configurable: true,
        get() { return currentScrollTop; },
        set(v: number) { currentScrollTop = v; },
      });

      try {
        seedManyEvents(80);
        render(<MessagesPane {...defaultProps} />);
        expect(scrollToMock).toHaveBeenCalledWith(expect.objectContaining({ top: 5000 }));

        scrollToMock.mockClear();
        currentScrollTop = 100;
        const container = screen.getByTestId("messages-scroll");
        fireEvent.wheel(container, { deltaY: -100 });
        fireEvent.scroll(container);

        currentScrollHeight = 7000;
        act(() => {
          seedManyEvents(100);
        });

        expect(scrollToMock).not.toHaveBeenCalledWith(expect.objectContaining({ top: 7000 }));
      } finally {
        window.requestAnimationFrame = originalRaf;
      }
    });

    it("keeps the visible message fixed while an older page is prepended", async () => {
      let scrollTop = 50;
      let prepended = false;
      Object.defineProperty(HTMLElement.prototype, "scrollHeight", { configurable: true, get() { return 20_000; } });
      Object.defineProperty(HTMLElement.prototype, "clientHeight", { configurable: true, get() { return 500; } });
      Object.defineProperty(HTMLElement.prototype, "scrollTop", {
        configurable: true,
        get() { return scrollTop; },
        set(value: number) { scrollTop = value; },
      });
      Element.prototype.scrollTo = ((options?: ScrollToOptions | number) => {
        if (typeof options === "object" && options?.top != null) scrollTop = options.top;
      }) as Element["scrollTo"];
      const originalGetBoundingClientRect = Element.prototype.getBoundingClientRect;
      Element.prototype.getBoundingClientRect = function (this: Element): DOMRect {
        const id = (this as HTMLElement).dataset?.eventId;
        if (id) {
          const sequence = Number(id.slice(1));
          const top = (prepended ? sequence * 10 : (sequence - 101) * 10);
          return { top, bottom: top + 8, left: 0, right: 0, width: 0, height: 8, x: 0, y: top, toJSON: () => ({}) } as DOMRect;
        }
        return { top: 0, bottom: 500, left: 0, right: 0, width: 0, height: 500, x: 0, y: 0, toJSON: () => ({}) } as DOMRect;
      };

      try {
        const current = Array.from({ length: 100 }, (_, index) => makeEvent({ id: `e${index + 101}`, sequence: index + 101 }));
        seedEvents(current);
        render(<MessagesPane {...defaultProps} />);
        scrollTop = 50;
        mockLoadOlderConversationPage.mockImplementation(async () => {
          prepended = true;
          seedEvents([
            ...Array.from({ length: 100 }, (_, index) => makeEvent({ id: `e${index + 1}`, sequence: index + 1 })),
            ...current,
          ]);
          return true;
        });

        const container = document.querySelector(".relative.min-h-0.flex-1.overflow-auto") as Element;
        await act(async () => fireEvent.scroll(container));

        // e101 was 0px from the viewport top before prepend. It is moved to
        // its new virtual index rather than leaving the user at the top of
        // the newly inserted page.
		expect(scrollTop).toBeGreaterThan(10 * 1000);
      } finally {
        Element.prototype.getBoundingClientRect = originalGetBoundingClientRect;
      }
    });
  });

  // --- Export selection flow (navigator → drawer → clipboard) ---

  describe("export selection flow", () => {
    function openExportSelection() {
      fireEvent.click(screen.getByTestId("messages-search-field"));
      fireEvent.click(screen.getByTestId("msg-export-enter"));
    }

    beforeEach(() => {
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: vi.fn().mockResolvedValue(undefined) },
        configurable: true,
      });
    });

    it("Continue opens the drawer with exactly the selected messages in order", () => {
      seedEvents([
        makeEvent({ id: "e1", sequence: 1, role: "user", text: "first question" }),
        makeEvent({ id: "e2", sequence: 2, text: "an answer" }),
        makeEvent({ id: "e3", sequence: 3, text: "unselected reply" }),
      ]);
      render(<MessagesPane {...defaultProps} />);
      openExportSelection();

      // Select out of order — the export must still be chronological.
      fireEvent.click(screen.getByTestId("msg-jump-item-e2"));
      fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
      fireEvent.click(screen.getByTestId("msg-export-continue"));

      const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
      expect(preview.indexOf("first question")).toBeGreaterThanOrEqual(0);
      expect(preview.indexOf("first question")).toBeLessThan(preview.indexOf("an answer"));
      expect(preview).not.toContain("unselected reply");
    });

    it("closing the drawer keeps the navigator selection for another pass", async () => {
      seedEvents([
        makeEvent({ id: "e1", sequence: 1, text: "keep me" }),
        makeEvent({ id: "e2", sequence: 2, text: "other" }),
      ]);
      render(<MessagesPane {...defaultProps} />);
      openExportSelection();
      fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
      fireEvent.click(screen.getByTestId("msg-export-continue"));
      expect(screen.getByTestId("msg-export-drawer")).toBeInTheDocument();

      fireEvent.pointerDown(screen.getByTestId("msg-export-drawer.backdrop"));
      await waitFor(() => expect(screen.queryByTestId("msg-export-drawer")).toBeNull());
      // Navigator is still open in selection mode with the selection intact.
      expect(screen.getByTestId("msg-jump-item-e1").getAttribute("aria-checked")).toBe("true");
      fireEvent.click(screen.getByTestId("msg-export-continue"));
      expect(screen.getByTestId("msg-export-preview").textContent).toContain("keep me");
    });

    it("drops selected IDs that no longer exist after a conversation refresh", () => {
      seedEvents([
        makeEvent({ id: "e1", sequence: 1, text: "stays" }),
        makeEvent({ id: "e2", sequence: 2, text: "goes away" }),
      ]);
      render(<MessagesPane {...defaultProps} />);
      openExportSelection();
      fireEvent.click(screen.getByTestId("msg-export-select-all"));
      expect(screen.getByTestId("msg-jump-item-e2").getAttribute("aria-checked")).toBe("true");

      act(() => {
        seedEvents([makeEvent({ id: "e1", sequence: 1, text: "stays" })]);
      });
      fireEvent.click(screen.getByTestId("msg-export-continue"));
      const preview = screen.getByTestId("msg-export-preview").textContent ?? "";
      expect(preview).toContain("stays");
      expect(preview).not.toContain("goes away");
    });

    it("existing jump behavior is preserved alongside the export entry point", () => {
      seedEvents([
        makeEvent({ id: "e1", sequence: 1, text: "target" }),
        makeEvent({ id: "e2", sequence: 2, text: "other" }),
      ]);
      render(<MessagesPane {...defaultProps} />);
      fireEvent.click(screen.getByTestId("messages-search-field"));
      expect(screen.getByTestId("msg-export-enter")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
      // Jump closes the navigator without entering selection mode.
      expect(screen.queryByTestId("msg-jump-list")).toBeNull();
      expect(screen.queryByTestId("msg-export-footer")).toBeNull();
    });
  });
});
