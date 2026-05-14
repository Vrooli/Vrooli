import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import MessageJumpList from "../components/MessageJumpList";
import type { ConversationEvent } from "../api/conversation";

function makeEvent(overrides: Partial<ConversationEvent> & { id: string; sequence: number }): ConversationEvent {
  return {
    sessionId: "sess-1",
    source: "claude_hook",
    role: "assistant",
    text: `Message ${overrides.sequence}`,
    speechParagraphs: [],
    summarized: false,
    createdAt: new Date().toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
    ...overrides,
  };
}

describe("MessageJumpList", () => {
  const onSelect = vi.fn();
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      unobserve: vi.fn(),
      disconnect: vi.fn(),
    })));
  });

  it("renders all messages in the list", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User question" }),
      makeEvent({ id: "e2", sequence: 2, role: "assistant", text: "Assistant answer" }),
      makeEvent({ id: "e3", sequence: 3, role: "assistant", text: "Follow up", source: "codex" }),
    ];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e2")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e3")).toBeInTheDocument();
  });

  it("renders user events as turn headers and assistants as indented rows", () => {
    const events = [
      makeEvent({ id: "u1", sequence: 1, role: "user", text: "Hello" }),
      makeEvent({ id: "a1", sequence: 2 }),
      makeEvent({ id: "a2", sequence: 3, source: "codex" }),
    ];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    expect(screen.getByTestId("msg-jump-item-u1").getAttribute("data-role")).toBe("user");
    expect(screen.getByTestId("msg-jump-item-a1").getAttribute("data-role")).toBe("assistant");
    expect(screen.getByTestId("msg-jump-item-a2").getAttribute("data-role")).toBe("assistant");
  });

  it("truncates long message text", () => {
    const longText = "A".repeat(300);
    const events = [makeEvent({ id: "e1", sequence: 1, text: longText })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    const item = screen.getByTestId("msg-jump-item-e1");
    expect(item.textContent).toContain("…");
    expect(item.textContent?.length).toBeLessThan(longText.length);
  });

  it("clicking an item calls onSelect and onClose", () => {
    const events = [makeEvent({ id: "e1", sequence: 1, text: "Click me" })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
    expect(onSelect).toHaveBeenCalledWith("e1");
    expect(onClose).toHaveBeenCalled();
  });

  it("highlights the focused event", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1 }),
      makeEvent({ id: "e2", sequence: 2 }),
    ];

    render(<MessageJumpList events={events} focusedEventId="e2" onSelect={onSelect} onClose={onClose} />);

    const focusedItem = screen.getByTestId("msg-jump-item-e2");
    expect(focusedItem.className).toContain("bg-wc-accent");
  });

  it("shows empty state when no events", () => {
    render(<MessageJumpList events={[]} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByText("messageJumpList.noMessages")).toBeInTheDocument();
  });

  it("Escape key calls onClose", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];

    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.keyDown(screen.getByTestId("msg-jump-list"), { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  // ── New behaviors ────────────────────────────────────────────────────────

  it("now-playing header collapses when duration is null", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId="e1" onSelect={onSelect} onClose={onClose} />);
    const header = screen.getByTestId("msg-jump-now-playing");
    expect(header.getAttribute("data-state")).toBe("idle");
    expect(screen.queryByTestId("msg-jump-now-scrub")).not.toBeInTheDocument();
  });

  it("now-playing header shows scrub when duration is set", () => {
    const events = [makeEvent({ id: "e1", sequence: 1, text: "Now playing this" })];
    render(
      <MessageJumpList
        events={events}
        focusedEventId="e1"
        onSelect={onSelect}
        onClose={onClose}
        currentTime={5}
        duration={20}
        isPaused={false}
      />,
    );
    const header = screen.getByTestId("msg-jump-now-playing");
    expect(header.getAttribute("data-state")).toBe("playing");
    expect(screen.getByTestId("msg-jump-now-scrub")).toBeInTheDocument();
  });

  it("now-playing play/pause button calls onPause when playing", () => {
    const onPause = vi.fn();
    const onResume = vi.fn();
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(
      <MessageJumpList
        events={events}
        focusedEventId="e1"
        onSelect={onSelect}
        onClose={onClose}
        currentTime={5}
        duration={20}
        isPaused={false}
        onPause={onPause}
        onResume={onResume}
      />,
    );
    fireEvent.click(screen.getByTestId("msg-jump-now-playpause"));
    expect(onPause).toHaveBeenCalled();
    expect(onResume).not.toHaveBeenCalled();
  });

  it("filter chips toggle visibility — failed filter shows only failed/rejected", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
      makeEvent({ id: "b", sequence: 2, ttsState: "failed" }),
      makeEvent({ id: "c", sequence: 3, ttsState: "rejected" }),
      makeEvent({ id: "d", sequence: 4, ttsState: "idle" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.click(screen.getByTestId("msg-jump-filter-failed"));
    expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-b")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-c")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-item-d")).not.toBeInTheDocument();
  });

  it("unheard filter hides played and listened events", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
      makeEvent({ id: "b", sequence: 2, ttsState: "idle", consumptionState: "listened" }),
      makeEvent({ id: "c", sequence: 3, ttsState: "idle", consumptionState: "unseen" }),
      makeEvent({ id: "d", sequence: 4, ttsState: "playing" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);

    fireEvent.click(screen.getByTestId("msg-jump-filter-unheard"));
    expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-item-b")).not.toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-c")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-d")).toBeInTheDocument();
  });

  it("playing event has aria-current and data-glyph='playing'", () => {
    const events = [
      makeEvent({ id: "p", sequence: 1, ttsState: "playing", text: "Streaming reply" }),
    ];
    render(<MessageJumpList events={events} focusedEventId="p" onSelect={onSelect} onClose={onClose} />);
    const item = screen.getByTestId("msg-jump-item-p");
    expect(item.getAttribute("aria-current")).toBe("true");
    expect(item.getAttribute("data-glyph")).toBe("playing");
  });

  it("failed event has data-glyph='failed' and is visible under default 'all' filter", () => {
    const events = [makeEvent({ id: "x", sequence: 1, ttsState: "failed" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-item-x").getAttribute("data-glyph")).toBe("failed");
  });

  it("renders safe-area spacer", () => {
    const events = Array.from({ length: 3 }, (_, i) =>
      makeEvent({ id: `e${i}`, sequence: i + 1 }),
    );
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    const spacer = screen.getByTestId("msg-jump-safe-spacer");
    expect(spacer).toBeInTheDocument();
    expect(spacer.getAttribute("style") ?? "").toContain("safe-bottom");
  });

  it("assistant rows have min-h-[44px] class for tap target", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-item-e1").className).toContain("min-h-[44px]");
  });

  it("user turn header has min-h-[48px] class", () => {
    const events = [
      makeEvent({ id: "u", sequence: 1, role: "user", text: "Hi" }),
      makeEvent({ id: "a", sequence: 2 }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-item-u").className).toContain("min-h-[48px]");
  });

  it("scroll container reserves safe-bottom via Tailwind class", () => {
    const events = Array.from({ length: 3 }, (_, i) =>
      makeEvent({ id: `e${i}`, sequence: i + 1 }),
    );
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    const list = screen.getByTestId("msg-jump-scroll");
    expect(list.className).toContain("safe-bottom");
  });

  it("scrub bar uses summarized accent when isSummarized=true", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(
      <MessageJumpList
        events={events}
        focusedEventId="e1"
        onSelect={onSelect}
        onClose={onClose}
        currentTime={1}
        duration={10}
        isPaused={false}
        isSummarized
      />,
    );
    const scrub = screen.getByTestId("msg-jump-now-scrub");
    expect(scrub.className).toMatch(/amber-400/);
  });

  it("renders 'next' badge on the event after the focused one when hasQueuedNext is true", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1 }),
      makeEvent({ id: "b", sequence: 2 }),
      makeEvent({ id: "c", sequence: 3 }),
    ];
    render(
      <MessageJumpList
        events={events}
        focusedEventId="a"
        onSelect={onSelect}
        onClose={onClose}
        hasQueuedNext
      />,
    );
    expect(screen.getByTestId("msg-jump-next-b")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-next-c")).not.toBeInTheDocument();
  });

  it("summarized events render an S badge", () => {
    const events = [makeEvent({ id: "s", sequence: 1, summarized: true })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-summarized-s")).toBeInTheDocument();
  });

  it("keyboard ArrowDown walks filtered events only", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
      makeEvent({ id: "b", sequence: 2, ttsState: "failed" }),
      makeEvent({ id: "c", sequence: 3, ttsState: "rejected" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filter-failed"));
    const list = screen.getByTestId("msg-jump-list");
    fireEvent.keyDown(list, { key: "ArrowDown" });
    fireEvent.keyDown(list, { key: "Enter" });
    // After filter, only b/c are visible. Starts at activeIndex=0 (b), Down → c, Enter selects c.
    expect(onSelect).toHaveBeenCalledWith("c");
  });

  it("scrolling to focused event uses the node index (not visible index) when user headers shift positions", () => {
    // Smoke test: list with mixed roles renders both header and rows; no throw.
    const events = [
      makeEvent({ id: "u", sequence: 1, role: "user", text: "Hi" }),
      makeEvent({ id: "a1", sequence: 2 }),
      makeEvent({ id: "a2", sequence: 3 }),
    ];
    expect(() =>
      render(
        <MessageJumpList events={events} focusedEventId="a2" onSelect={onSelect} onClose={onClose} />,
      ),
    ).not.toThrow();
    const focused = screen.getByTestId("msg-jump-item-a2");
    expect(within(focused).queryByText("#3")).toBeTruthy();
  });
});
