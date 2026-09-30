import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { useSessionActivityStore } from "../stores/useSessionActivityStore";
import type { ConversationEvent } from "../api/conversation";
import { makeConversationEvents } from "./fixtures/conversationFixture";
import {
  CLIENT_HEIGHT,
  container,
  flushFrames,
  geo,
  growRow,
  installScrollGeometry,
  liveEvent,
  maxScrollTop,
  remaining,
  uninstallScrollGeometry,
  userScrollTo,
} from "./fixtures/messagesScrollHarness";

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

vi.mock("../components/markdown", () => ({
  MarkdownRenderer: ({ content }: { content: string }) => <div data-testid="mock-markdown">{content}</div>,
}));

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

function append(sequence: number) {
  useConversationStore.getState().appendEvent(liveEvent(sequence));
}

describe("Messages scroll follow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    installScrollGeometry();
    localStorage.clear();
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({ sessions: {} });
    useSessionActivityStore.setState({ activities: {} });
    globalThis.fetch = vi.fn() as typeof fetch;
  });

  afterEach(() => {
    uninstallScrollGeometry();
  });

  function mount() {
    seed(makeConversationEvents(2500, 7));
    const view = render(<MessagesPane {...props} />);
    act(() => { flushFrames(); });
    return view;
  }

  it("[REQ:P0-017a] Test A: appending while not following leaves the viewport where the user put it", () => {
    mount();
    userScrollTo(maxScrollTop() - 1200);
    act(() => { flushFrames(); });
    const before = geo.scrollTop;
    geo.writes = [];

    act(() => {
      for (let sequence = 2501; sequence <= 2505; sequence += 1) append(sequence);
      flushFrames();
    });

    expect(geo.writes).toEqual([]);
    expect(geo.scrollTop).toBe(before);
    expect(screen.getByTestId("msg-new-pill")).toHaveAttribute("data-count", "5");
  });

  it("[REQ:P0-017a] Test B: appending while following keeps the list at its end", () => {
    mount();
    userScrollTo(maxScrollTop() - 100);
    act(() => { flushFrames(); });

    act(() => {
      append(2501);
      flushFrames();
    });

    expect(remaining()).toBe(0);
    expect(screen.queryByTestId("msg-new-pill")).not.toBeInTheDocument();
  });

  function showWorkingSlot() {
    useSessionActivityStore.getState().apply("sess-1", {
      state: "working", source: "screen", confidence: 0.85, since: new Date().toISOString(), harness: "claude",
    });
  }

  it("[REQ:P0-017e] the state slot appearing while following keeps the list at its end", () => {
    mount();
    act(() => {
      showWorkingSlot();
      flushFrames();
    });

    expect(screen.getByTestId("messages-state-slot")).toBeInTheDocument();
    expect(remaining()).toBe(0);
  });

  it("[REQ:P0-017e] the state slot appearing while not following moves nothing", () => {
    mount();
    userScrollTo(maxScrollTop() - 1200);
    act(() => { flushFrames(); });
    const before = geo.scrollTop;
    geo.writes = [];

    act(() => {
      showWorkingSlot();
      flushFrames();
    });

    expect(screen.getByTestId("messages-state-slot")).toBeInTheDocument();
    expect(geo.writes).toEqual([]);
    expect(geo.scrollTop).toBe(before);
  });

  it("[REQ:P0-017a] Test C: mount scrolls to the end once; growth follows only while following", () => {
    seed(makeConversationEvents(2500, 7));
    render(<MessagesPane {...props} />);
    act(() => { flushFrames(); });

    // Exactly one programmatic write lands the fresh open at the end.
    expect(geo.writes).toEqual([maxScrollTop()]);
    expect(remaining()).toBe(0);

    // A visible row measuring taller while following keeps the end in view
    // with one write, not a settle loop.
    geo.writes = [];
    growRow("fixture-2500", 900);
    expect(remaining()).toBe(0);
    expect(geo.writes).toHaveLength(1);

    // After the user scrolls away, the same kind of growth never moves them.
    userScrollTo(maxScrollTop() - 1600);
    act(() => { flushFrames(); });
    const held = geo.scrollTop;
    geo.writes = [];
    growRow("fixture-2499", 1400);
    expect(geo.writes).toEqual([]);
    expect(geo.scrollTop).toBe(held);
  });

  it("[REQ:P0-017a] Test D: a programmatic scroll never recomputes follow; a user gesture does", () => {
    seed(makeConversationEvents(2500, 7));
    render(<MessagesPane {...props} />);
    // Frames are NOT flushed: the mount scroll is still in flight.
    const el = container();
    expect(el).toHaveAttribute("data-follow", "true");

    geo.scrollTop = 0;
    fireEvent.scroll(el);
    expect(el).toHaveAttribute("data-follow", "true");

    userScrollTo(0);
    expect(el).toHaveAttribute("data-follow", "false");
  });

  it("[REQ:P0-017a] a user scroll during the mount scroll is honoured and never undone", () => {
    seed(makeConversationEvents(2500, 7));
    render(<MessagesPane {...props} />);
    // Mount scroll still in flight (frames pending) when the user flicks up.
    userScrollTo(maxScrollTop() - 3 * CLIENT_HEIGHT);
    const held = geo.scrollTop;
    geo.writes = [];

    act(() => {
      flushFrames();
      append(2501);
      flushFrames();
    });
    growRow("fixture-2500", 1200);

    expect(geo.writes).toEqual([]);
    expect(geo.scrollTop).toBe(held);
  });

  it("[REQ:P0-017a] jump to bottom resumes following", () => {
    mount();
    userScrollTo(0);
    act(() => { flushFrames(); });
    act(() => { append(2501); flushFrames(); });
    fireEvent.click(screen.getByTestId("msg-new-pill"));
    act(() => { flushFrames(); });
    expect(container()).toHaveAttribute("data-follow", "true");

    act(() => { append(2502); flushFrames(); });
    expect(remaining()).toBe(0);
  });
});
