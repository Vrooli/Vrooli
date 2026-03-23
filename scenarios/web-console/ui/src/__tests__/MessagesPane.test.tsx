import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore } from "../stores/useConversationStore";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import type { ConversationEvent } from "../lib/api";

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

  it("renders play and audio icons on each assistant message card", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ];
    seedEvents(events);
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

  it("active speaking event shows left border highlight", () => {
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

    // Active card has accent left border
    const activeCard = screen.getByTestId("msg-card-e2");
    expect(activeCard.className).toContain("border-l-");

    // Inactive card does not
    const inactiveCard = screen.getByTestId("msg-card-e1");
    expect(inactiveCard.className).not.toContain("border-l-");
  });

  it("play button on active card triggers onSpeakFromHere (stop is in global bar)", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(
      <MessagesPane
        {...defaultProps}
        activeSpeakingEventId="e1"
        isTtsSpeaking={true}
      />,
    );

    // The play button always shows — stop lives in AudioPlayerBar
    fireEvent.click(screen.getByTestId("msg-speak-from-e1"));
    expect(defaultProps.onSpeakFromHere).toHaveBeenCalledWith("e1");
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

    // User message has no audio controls
    expect(screen.queryByTestId("msg-speak-from-e1")).toBeNull();
    expect(screen.queryByTestId("msg-audio-e1")).toBeNull();
    // Assistant message has audio controls
    expect(screen.getByTestId("msg-speak-from-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-audio-e2")).toBeInTheDocument();
  });

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

    // Open popover
    fireEvent.click(screen.getByTestId("msg-audio-e1"));

    // Verify summarization toggle buttons
    expect(screen.getByTestId("msg-play-summarized-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-play-original-e1")).toBeInTheDocument();
  });

  // --- Feature 1: Control strip (search + navigation) ---

  it("renders control strip with search, up, and down buttons", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-control-strip")).toBeInTheDocument();
    expect(screen.getByTestId("messages-search-btn")).toBeInTheDocument();
    expect(screen.getByTestId("messages-nav-up")).toBeInTheDocument();
    expect(screen.getByTestId("messages-nav-down")).toBeInTheDocument();
  });

  it("clicking search button opens the search drawer", () => {
    seedEvents([makeEvent({ id: "e1", sequence: 1 })]);
    render(<MessagesPane {...defaultProps} />);

    // Search drawer should not be visible initially
    expect(screen.queryByTestId("messages-search-panel")).toBeNull();

    fireEvent.click(screen.getByTestId("messages-search-btn"));
    expect(screen.getByTestId("messages-search-panel")).toBeInTheDocument();
  });

  it("search highlights matching messages", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
      makeEvent({ id: "e2", sequence: 2, text: "Goodbye world" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Open search and type a query
    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "world" },
    });

    // Both messages contain "world" — check for <mark> elements
    const marks = document.querySelectorAll("mark");
    expect(marks.length).toBe(2);
  });

  it("down chevron without search scrolls to next user message", () => {
    const scrollIntoViewMock = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoViewMock;

    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "Question 1" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Answer 1" }),
      makeEvent({ id: "e3", sequence: 3, role: "user", text: "Question 2" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Click down — should scroll to first user message
    fireEvent.click(screen.getByTestId("messages-nav-down"));
    expect(scrollIntoViewMock).toHaveBeenCalledTimes(1);
  });

  it("disables nav chevrons when no events exist", () => {
    seedEvents([]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-nav-up")).toBeDisabled();
    expect(screen.getByTestId("messages-nav-down")).toBeDisabled();
  });

  it("enables nav chevrons when events exist", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, role: "assistant", text: "Answer" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    expect(screen.getByTestId("messages-nav-up")).not.toBeDisabled();
    expect(screen.getByTestId("messages-nav-down")).not.toBeDisabled();
  });

  it("clicking a message card highlights it with accent border", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Initially no accent border
    expect(screen.getByTestId("msg-card-e1").className).toContain("border-wc-default");

    // Click to focus
    fireEvent.click(screen.getByTestId("msg-card-e1"));
    expect(screen.getByTestId("msg-card-e1").className).toContain("border-wc-accent");
    expect(screen.getByTestId("msg-card-e2").className).toContain("border-wc-default");
  });

  it("chevron navigation starts from clicked message", () => {
    const scrollIntoViewMock = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoViewMock;

    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "First" }),
      makeEvent({ id: "e2", sequence: 2, text: "Second" }),
      makeEvent({ id: "e3", sequence: 3, text: "Third" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Click the second message to focus it
    fireEvent.click(screen.getByTestId("msg-card-e2"));

    // Press down — should move to e3 (next after e2)
    fireEvent.click(screen.getByTestId("messages-nav-down"));
    expect(screen.getByTestId("msg-card-e3").className).toContain("border-wc-accent");
    expect(screen.getByTestId("msg-card-e2").className).toContain("border-wc-default");
  });

  it("closing search clears highlights", () => {
    seedEvents([
      makeEvent({ id: "e1", sequence: 1, text: "Hello world" }),
    ]);
    render(<MessagesPane {...defaultProps} />);

    // Open search and type
    fireEvent.click(screen.getByTestId("messages-search-btn"));
    fireEvent.change(screen.getByTestId("messages-search-input"), {
      target: { value: "world" },
    });
    expect(document.querySelectorAll("mark").length).toBe(1);

    // Close search
    fireEvent.click(screen.getByTestId("messages-search-close"));
    expect(document.querySelectorAll("mark").length).toBe(0);
  });

  // --- Feature 2: Font size sync ---

  it("applies font size from workspace store to message text", () => {
    useWorkspaceStore.setState({
      panes: [{ sessionId: "sess-1", name: "test", headerColor: "transparent", themeId: "slate-ocean", fontSize: 20, groupId: null, supportsMessagesView: true }],
    });
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Sized text" })]);
    render(<MessagesPane {...defaultProps} />);

    const card = screen.getByTestId("msg-card-e1");
    // The text div is the last child of the article
    const textDiv = card.querySelector(".whitespace-pre-wrap") as HTMLElement;
    expect(textDiv.style.fontSize).toBe("20px");
  });

  it("falls back to default font size when pane has no custom size", () => {
    // No pane in workspace store for sess-1
    useWorkspaceStore.setState({ panes: [] });
    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Default size" })]);
    render(<MessagesPane {...defaultProps} />);

    const card = screen.getByTestId("msg-card-e1");
    const textDiv = card.querySelector(".whitespace-pre-wrap") as HTMLElement;
    expect(textDiv.style.fontSize).toBe("14px"); // TERMINAL_FONT_SIZE default
  });

  // --- Feature 4: Copy-to-clipboard ---

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
    Object.assign(navigator, {
      clipboard: { writeText: writeTextMock },
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-copy-e1"));
    expect(writeTextMock).toHaveBeenCalledWith("Copy me");
  });

  it("shows checkmark icon after copying", () => {
    const writeTextMock = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText: writeTextMock },
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "Copy me" })]);
    render(<MessagesPane {...defaultProps} />);

    fireEvent.click(screen.getByTestId("msg-copy-e1"));

    // After clicking, the button should contain a Check icon (green)
    const btn = screen.getByTestId("msg-copy-e1");
    const svg = btn.querySelector("svg");
    expect(svg?.classList.toString()).toContain("text-green-400");
  });

  // --- On-demand summarize error visibility ---

  it("shows error when on-demand summarize returns an error", async () => {
    mockSummarizeEvent.mockResolvedValue({
      summarized: false,
      error: "Summarization failed: ollama returned 404: model not found",
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(<MessagesPane {...defaultProps} />);

    // Open audio popover
    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    expect(screen.getByTestId("audio-popover-e1")).toBeInTheDocument();

    // Click "Summarize for playback"
    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    // Error should be visible in the popover
    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
      expect(screen.getByTestId("msg-summarize-error-e1").textContent).toContain("model not found");
    });
  });

  it("clears summarize error when retrying successfully", async () => {
    // First call fails
    mockSummarizeEvent.mockResolvedValueOnce({
      summarized: false,
      error: "Summarization failed: connection refused",
    });
    // Second call succeeds
    mockSummarizeEvent.mockResolvedValueOnce({
      summarized: true,
      speechParagraphs: ["Short summary"],
    });

    seedEvents([makeEvent({ id: "e1", sequence: 1, text: "A long assistant response" })]);
    render(<MessagesPane {...defaultProps} />);

    // Open popover and trigger first (failing) summarize
    fireEvent.click(screen.getByTestId("msg-audio-e1"));
    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    await waitFor(() => {
      expect(screen.getByTestId("msg-summarize-error-e1")).toBeInTheDocument();
    });

    // Popover stays open after error — retry directly
    fireEvent.click(screen.getByTestId("msg-request-summarize-e1"));

    // Error should clear on retry
    await waitFor(() => {
      expect(screen.queryByTestId("msg-summarize-error-e1")).toBeNull();
    });
  });
});
