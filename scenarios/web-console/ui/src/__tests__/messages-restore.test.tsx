import { renderWithProviders as render } from "../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, screen } from "@testing-library/react";
import MessagesPane from "../components/MessagesPane";
import { strings } from "../consts/strings";
import { useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import type { ConversationEvent } from "../api/conversation";
import { makeConversationEvents } from "./fixtures/conversationFixture";
import {
  container,
  flushFrames,
  geo,
  growRow,
  installScrollGeometry,
  liveEvent,
  maxScrollTop,
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

const ALL = makeConversationEvents(2500, 7);

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

function rowTop(eventId: string): number {
  const row = document.querySelector<HTMLElement>(`[data-event-id="${eventId}"]`);
  if (!row) throw new Error(`row ${eventId} not rendered`);
  return parseFloat(row.style.top);
}

async function settle() {
  for (let i = 0; i < 3; i += 1) {
    await act(async () => { flushFrames(); await Promise.resolve(); });
  }
}

describe("Messages position restore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    installScrollGeometry();
    localStorage.clear();
    useMessagesViewStore.setState({ viewModes: {}, positions: {} });
    useConversationStore.setState({ sessions: {} });
    globalThis.fetch = vi.fn() as typeof fetch;
  });

  afterEach(() => {
    uninstallScrollGeometry();
  });

  it("[REQ:P0-017b] pages the saved message in before any scroll, then restores it with its offset", async () => {
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-100", topSequence: 100, offsetPx: 40, follow: false });
    seed(ALL.slice(2000)); // the loaded window is the tail: 2001..2500
    let writesWhenPaged = -1;
    mockLoadConversationPageContaining.mockImplementation(() => {
      writesWhenPaged = geo.writes.length;
      seed(ALL.slice(0, 500));
      return Promise.resolve(true);
    });

    render(<MessagesPane {...props} />);
    await settle();

    expect(mockLoadConversationPageContaining).toHaveBeenCalledWith("sess-1", 100);
    expect(writesWhenPaged).toBe(0);
    expect(geo.scrollTop).toBe(rowTop("fixture-100") + 40);
    expect(container()).toHaveAttribute("data-follow", "false");
  });

  it("[REQ:P0-017b] restores a place inside the loaded window without a request", async () => {
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-2100", topSequence: 2100, offsetPx: 12, follow: false });
    seed(ALL.slice(2000));

    render(<MessagesPane {...props} />);
    await settle();

    expect(mockLoadConversationPageContaining).not.toHaveBeenCalled();
    // The saved place is shown directly; the list never visits the end first.
    expect(geo.writes).not.toContain(maxScrollTop());
    expect(geo.scrollTop).toBe(rowTop("fixture-2100") + 12);
  });

  it("[REQ:P0-017b] a place near the end that does not fit yet is reached once the rows below measure", async () => {
    // The last message, 500 px into it: with estimated heights the list ends too soon.
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-2500", topSequence: 2500, offsetPx: 500, follow: false });
    seed(ALL.slice(2000));

    render(<MessagesPane {...props} />);
    await settle();
    expect(geo.scrollTop).toBeLessThan(rowTop("fixture-2500") + 500);

    growRow("fixture-2500", 2400);
    await settle();
    expect(geo.scrollTop).toBe(rowTop("fixture-2500") + 500);

    // Once the place is shown, later growth never moves what the reader sees:
    // a row above the viewport growing is compensated, so the message on
    // screen keeps its screen position.
    userScrollTo(geo.scrollTop - 100);
    const onScreen = rowTop("fixture-2500") - geo.scrollTop;
    growRow("fixture-2499", 3000);
    expect(rowTop("fixture-2500") - geo.scrollTop).toBe(onScreen);
  });

  it("[REQ:P0-017b] a place deeper into its row than the row's estimate is kept once the row measures", async () => {
    // fixture-2100's estimate is shorter than 450 px; its real height is 1200 px.
    geo.rowHeights.set("fixture-2100", 1200);
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-2100", topSequence: 2100, offsetPx: 450, follow: false });
    seed(ALL.slice(2000));

    render(<MessagesPane {...props} />);
    await settle();

    expect(rowTop("fixture-2100") - geo.scrollTop).toBe(-450);
  });

  it("[REQ:P0-017b] a user gesture drops a place still waiting to fit", async () => {
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-2500", topSequence: 2500, offsetPx: 500, follow: false });
    seed(ALL.slice(2000));

    render(<MessagesPane {...props} />);
    await settle();
    userScrollTo(1000);
    // The list grows (a new message arrives); the dropped place is not applied.
    act(() => { useConversationStore.getState().appendEvent(liveEvent(2501)); });
    await settle();

    expect(geo.scrollTop).toBe(1000);
  });

  it("[REQ:P0-017b] says it could not restore the place and goes to the end once", async () => {
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "gone", topSequence: 7, offsetPx: 0, follow: false });
    seed(ALL.slice(2000));
    mockLoadConversationPageContaining.mockResolvedValue(false);

    render(<MessagesPane {...props} />);
    await settle();

    expect(screen.getByText(strings.messagesPane.restoreFailed)).toBeInTheDocument();
    expect(geo.writes).toEqual([maxScrollTop()]);
    expect(container()).toHaveAttribute("data-follow", "true");
  });

  it("[REQ:P0-017b] opens at the end when nothing was saved or the list was following", async () => {
    useMessagesViewStore.getState().savePosition("sess-1", { topEventId: "fixture-100", topSequence: 100, offsetPx: 0, follow: true });
    seed(ALL);

    render(<MessagesPane {...props} />);
    await settle();

    expect(mockLoadConversationPageContaining).not.toHaveBeenCalled();
    expect(geo.writes).toEqual([maxScrollTop()]);
  });

  it("[REQ:P0-017b] saves the top visible message, its offset, and follow as the user scrolls and on unmount", async () => {
    seed(ALL);
    const view = render(<MessagesPane {...props} />);
    await settle();

    userScrollTo(50_000);
    await act(async () => { await new Promise((resolve) => { setTimeout(resolve, 300); }); });

    // Oracle: the first row whose bottom is below the viewport top, by the rows' own rects.
    const rows = [...document.querySelectorAll<HTMLElement>("[data-event-id]")];
    const top = rows.find((row) => row.getBoundingClientRect().bottom > 0);
    expect(top).toBeDefined();
    expect(useMessagesViewStore.getState().positions["sess-1"]).toMatchObject({
      topEventId: top?.dataset.eventId,
      topSequence: Number(top?.dataset.sequence),
      offsetPx: 50_000 - parseFloat(top?.style.top ?? "0"),
      follow: false,
    });

    // A scroll just before unmount is not lost to the throttle.
    userScrollTo(maxScrollTop());
    view.unmount();
    expect(useMessagesViewStore.getState().positions["sess-1"]?.follow).toBe(true);
  });
});
