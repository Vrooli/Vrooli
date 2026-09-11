import { renderWithProviders as render, setDesktopViewport, setMobileViewport } from "../test-utils";
import { useState } from "react";
import { strings } from "../consts/strings";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, within, createEvent } from "@testing-library/react";
import MessageJumpList, { type MessageExportSelection, type NavigatorSearch } from "../components/MessageJumpList";
import type { ConversationEvent, ConversationSearchMatch } from "../api/conversation";

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

function hit(event: ConversationEvent, match: string): ConversationSearchMatch {
  const start = event.text.indexOf(match);
  return {
    eventId: event.id, sequence: event.sequence, excerpt: event.text,
    ranges: start >= 0 ? [{ start, end: start + match.length }] : [],
    role: event.role, createdAt: event.createdAt,
  };
}

/** Server search as MessagesPane provides it; hits default to none. */
function search(overrides: Partial<NavigatorSearch> = {}): NavigatorSearch {
  return {
    hits: [],
    truncated: false,
    options: { mode: "text", caseSensitive: false, wholeWord: false },
    onOptionsChange: vi.fn(),
    ...overrides,
  };
}

describe("MessageJumpList navigator", () => {
  const onSelect = vi.fn();
  const onClose = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    setDesktopViewport();
    vi.stubGlobal(
      "ResizeObserver",
      vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })),
    );
  });

  // ── Core rendering ─────────────────────────────────────────────────────────

  it("renders all messages as rows", () => {
    const events = [
      makeEvent({ id: "e1", sequence: 1, role: "user", text: "User question" }),
      makeEvent({ id: "e2", sequence: 2, text: "Assistant answer" }),
      makeEvent({ id: "e3", sequence: 3, text: "Follow up", source: "codex_tailer" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-e1").getAttribute("data-role")).toBe("user");
    expect(screen.getByTestId("msg-jump-item-e2").getAttribute("data-role")).toBe("assistant");
    expect(screen.getByTestId("msg-jump-item-e3")).toBeInTheDocument();
  });

  it("clicking a row calls onSelect and onClose", () => {
    const events = [makeEvent({ id: "e1", sequence: 1, text: "Click me" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
    expect(onSelect).toHaveBeenCalledWith("e1");
    expect(onClose).toHaveBeenCalled();
  });

  it("highlights the focused event", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 }), makeEvent({ id: "e2", sequence: 2 })];
    render(<MessageJumpList events={events} focusedEventId="e2" onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-item-e2").className).toContain("bg-wc-accent");
  });

  it("playing event has aria-current and data-glyph='playing'", () => {
    const events = [makeEvent({ id: "p", sequence: 1, ttsState: "playing", text: "Streaming reply" })];
    render(<MessageJumpList events={events} focusedEventId="p" onSelect={onSelect} onClose={onClose} />);
    const item = screen.getByTestId("msg-jump-item-p");
    expect(item.getAttribute("aria-current")).toBe("true");
    expect(item.getAttribute("data-glyph")).toBe("playing");
  });

  it("summarized events render an S badge", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, summarized: true }),
      makeEvent({ id: "b", sequence: 2 }),
    ];
    render(<MessageJumpList events={events} focusedEventId="a" onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-summarized-a")).toBeInTheDocument();
  });

  it("[REQ:P0-017g] keeps one bottom padding: no spacer, no safe-area padding on the scroller, a body that shrinks", () => {
    setMobileViewport();
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.queryByTestId("msg-jump-safe-spacer")).toBeNull();
    const scroller = screen.getByTestId("msg-jump-scroll");
    expect(scroller.className).not.toContain("safe-bottom");
    // The sheet body is a shrinking column, so the scroller gives up height
    // and anything below it (the export footer) stays on screen.
    expect(scroller.closest(".min-h-0.flex-col")).not.toBeNull();
  });

  it("[REQ:P0-017g] a long list renders a window, and reaching its last row renders that row", () => {
    const events = Array.from({ length: 300 }, (_, index) => makeEvent({
      id: `e${String(index + 1)}`,
      sequence: index + 1,
      text: index % 3 === 0 ? `Message ${String(index + 1)} ${"with a much longer preview ".repeat(6)}` : `Message ${String(index + 1)}`,
    }));
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    // The first commit renders a window of rows, not all 300.
    expect(screen.getAllByTestId(/^msg-jump-item-/).length).toBeLessThan(100);
    // Scrolled to its end, the list renders its last row.
    const scroller = screen.getByTestId("msg-jump-scroll");
    scroller.scrollTop = 1_000_000;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("msg-jump-item-e300")).toBeInTheDocument();
  });

  it("[REQ:P0-017g] a list that arrives after an empty state still follows scrolling", () => {
    // A search opens empty and fills when results land; the list must track
    // scrolling from then on, not only when it was there from the start.
    const { rerender } = render(<MessageJumpList events={[]} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    const events = Array.from({ length: 300 }, (_, index) => makeEvent({ id: `e${String(index + 1)}`, sequence: index + 1, text: `Message ${String(index + 1)}` }));
    rerender(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    const scroller = screen.getByTestId("msg-jump-scroll");
    scroller.scrollTop = 1_000_000;
    fireEvent.scroll(scroller);
    expect(screen.getByTestId("msg-jump-item-e300")).toBeInTheDocument();
  });

  it("[REQ:P0-017g] rows read as speaker · time, like the message rows", () => {
    const events = [
      makeEvent({ id: "u", sequence: 7, role: "user", text: "Question" }),
      makeEvent({ id: "a", sequence: 8, source: "claude_hook", text: "Answer" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-speaker-u")).toHaveTextContent(strings.messageJumpList.roleYou);
    expect(screen.getByTestId("msg-jump-speaker-a")).toHaveTextContent(strings.messageJumpList.roleClaude);
    // The sequence number lives in the time's tooltip, not in the row.
    expect(screen.getByTestId("msg-jump-time-a")).toHaveAttribute("title", "#8");
    expect(screen.getByTestId("msg-jump-item-a")).not.toHaveTextContent("#8");
  });

  it("assistant rows keep a 44px min tap target; user rows 48px", () => {
    const events = [
      makeEvent({ id: "u", sequence: 1, role: "user", text: "Hi" }),
      makeEvent({ id: "a", sequence: 2 }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-jump-item-a").className).toContain("min-h-[44px]");
    expect(screen.getByTestId("msg-jump-item-u").className).toContain("min-h-[48px]");
  });

  // ── Search ───────────────────────────────────────────────────────────────

  const exportSelection = (): MessageExportSelection => ({
    selectedIds: new Set(),
    onToggle: vi.fn(),
    onSelectAll: vi.fn(),
    onSelectVisible: vi.fn(),
    onClear: vi.fn(),
    onContinue: vi.fn(),
  });

  it("[REQ:P0-017g] desktop: the search field is the header, focused on open, beside an Export icon; no title text", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} initialFocus="search" query="" onQueryChange={vi.fn()} search={search()} exportSelection={exportSelection()} />);

    const header = screen.getByTestId("msg-nav-header");
    expect(within(header).getByTestId("msg-nav-search")).toHaveFocus();
    expect(within(header).getByTestId("msg-export-enter")).toHaveAttribute("aria-label", "messageExport.exportAction");
    expect(screen.queryByText("messageJumpList.titleJump")).toBeNull();
  });

  it("[REQ:P0-017g] phone: the sheet's header is the search field and an Export icon", () => {
    setMobileViewport();
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} initialFocus="search" query="" onQueryChange={vi.fn()} search={search()} exportSelection={exportSelection()} />);

    const header = screen.getByTestId("msg-nav-header");
    expect(within(header).getByTestId("msg-nav-search")).toHaveFocus();
    expect(screen.getByTestId("msg-export-enter")).toHaveAttribute("aria-label", "messageExport.exportAction");
    expect(screen.getByRole("dialog", { name: "messageJumpList.titleJump" })).toBeInTheDocument();
    expect(screen.queryByText("messageJumpList.titleJump")).toBeNull();
  });

  it("renders search input and result count", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    expect(screen.getByTestId("msg-nav-search")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-count")).toBeInTheDocument();
  });

  it("[REQ:P0-017g] a query lists the server's hits with the server's highlights", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, text: "deploy the service" }),
      makeEvent({ id: "b", sequence: 2, text: "unrelated content" }),
    ];
    const hits = [hit(events[0] as ConversationEvent, "deploy")];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} query="deploy" onQueryChange={vi.fn()} search={search({ hits })} />);
    expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-item-b")).not.toBeInTheDocument();
    const highlight = within(screen.getByTestId("msg-jump-item-a")).getByText("deploy", { selector: '[data-match="true"]' });
    expect(highlight).toBeInTheDocument();
  });

  it("[REQ:P0-017g] offers Text, Regex, and Fuzzy modes with case and whole-word chips", () => {
    const onOptionsChange = vi.fn();
    render(<MessageJumpList events={[makeEvent({ id: "a", sequence: 1 })]} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search({ onOptionsChange })} />);
    const modes = screen.getByTestId("msg-jump-mode");
    expect(within(modes).getAllByRole("button").map((b) => b.getAttribute("data-mode"))).toEqual(["text", "regex", "fuzzy"]);
    fireEvent.click(within(modes).getByRole("button", { name: /regex/i }));
    expect(onOptionsChange).toHaveBeenLastCalledWith({ mode: "regex", caseSensitive: false, wholeWord: false });
    fireEvent.click(screen.getByTestId("msg-jump-case"));
    expect(onOptionsChange).toHaveBeenLastCalledWith({ mode: "text", caseSensitive: true, wholeWord: false });
    fireEvent.click(screen.getByTestId("msg-jump-whole-word"));
    expect(onOptionsChange).toHaveBeenLastCalledWith({ mode: "text", caseSensitive: false, wholeWord: true });
  });

  it("[REQ:P0-017g] shows the server's error for a query it cannot run", () => {
    render(<MessageJumpList events={[makeEvent({ id: "a", sequence: 1 })]} focusedEventId={null} onSelect={onSelect} onClose={onClose} query="(" onQueryChange={vi.fn()} search={search({ error: "invalid regular expression: missing closing )" })} />);
    expect(screen.getByTestId("msg-jump-error")).toHaveTextContent("invalid regular expression: missing closing )");
  });

  it("[REQ:P0-017g] says when the server stopped short", () => {
    const events = [makeEvent({ id: "a", sequence: 1, text: "needle" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} query="needle" onQueryChange={vi.fn()} search={search({ hits: [hit(events[0] as ConversationEvent, "needle")], truncated: true })} />);
    expect(screen.getByTestId("msg-jump-truncated")).toBeInTheDocument();
  });

  it("does not prevent the search input's mousedown default (so clicking focuses it)", () => {
    // The overlay wrapper preventDefaults mousedown to protect host focus; the
    // input must opt out or it can never be focused by click.
    const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    const input = screen.getByTestId("msg-nav-search");
    const ev = createEvent.mouseDown(input);
    fireEvent(input, ev);
    expect(ev.defaultPrevented).toBe(false);
  });

  it("clear button appears with a query and resets it", () => {
    const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    expect(screen.queryByTestId("msg-nav-clear")).toBeNull();
    fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "hello" } });
    expect(screen.getByTestId("msg-nav-clear")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("msg-nav-clear"));
    expect((screen.getByTestId("msg-nav-search") as HTMLInputElement).value).toBe("");
  });

  it("uses a controlled query when provided", () => {
    const onQueryChange = vi.fn();
    const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
    render(
      <MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} query="hel" onQueryChange={onQueryChange} search={search()} />,
    );
    expect((screen.getByTestId("msg-nav-search") as HTMLInputElement).value).toBe("hel");
    fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "hello" } });
    expect(onQueryChange).toHaveBeenCalledWith("hello");
  });

  // ── Primary chips ──────────────────────────────────────────────────────────

  it("User chip filters to user messages; Assistant chip to assistant messages", () => {
    const events = [
      makeEvent({ id: "u", sequence: 1, role: "user", text: "Q" }),
      makeEvent({ id: "a", sequence: 2, text: "A" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-nav-chip-user"));
    expect(screen.getByTestId("msg-jump-item-u")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("msg-nav-chip-assistant"));
    expect(screen.queryByTestId("msg-jump-item-u")).not.toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
  });

  it("Failed status shows only failed/rejected; All resets", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
      makeEvent({ id: "b", sequence: 2, ttsState: "failed" }),
      makeEvent({ id: "c", sequence: 3, ttsState: "rejected" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-status-failed"));
    expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-b")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-c")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("msg-nav-chip-all"));
    expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
  });

  // ── Advanced panel ─────────────────────────────────────────────────────────

  it("[REQ:P0-017g] filters stay folded behind one Filters button; the role control stays out", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, source: "claude_hook" }),
      makeEvent({ id: "b", sequence: 2, source: "grok_tailer" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.queryByTestId("msg-nav-advanced")).toBeNull();
    expect(screen.queryByTestId("msg-nav-status-failed")).toBeNull();
    expect(screen.queryByTestId("msg-nav-chip-failed")).toBeNull();
    expect(screen.getByTestId("msg-nav-chip-user")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-filters")).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    expect(screen.getByTestId("msg-jump-filters")).toHaveAttribute("aria-expanded", "true");
    const panel = screen.getByTestId("msg-nav-advanced");
    expect(panel).toBeInTheDocument();
    // Source chips only for present sources.
    expect(screen.getByTestId("msg-nav-source-claude")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-source-grok")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-nav-source-codex")).toBeNull();
    expect(screen.getByTestId("msg-nav-status-summarized")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-content-code")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-sort-newest")).toBeInTheDocument();
    expect(screen.getByTestId("msg-nav-group-flat")).toBeInTheDocument();
  });

  it("source filter narrows to a single runtime", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, source: "claude_hook" }),
      makeEvent({ id: "b", sequence: 2, source: "grok_tailer" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-source-grok"));
    expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-b")).toBeInTheDocument();
  });

  it("content filter narrows to code messages", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, text: "run `make test`" }),
      makeEvent({ id: "b", sequence: 2, text: "plain prose" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-content-code"));
    expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
    expect(screen.queryByTestId("msg-jump-item-b")).not.toBeInTheDocument();
  });

  it("newest sort reverses row order", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1 }),
      makeEvent({ id: "b", sequence: 2 }),
      makeEvent({ id: "c", sequence: 3 }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-group-flat"));
    fireEvent.click(screen.getByTestId("msg-nav-sort-newest"));
    const rows = screen.getAllByTestId(/^msg-jump-item-/);
    expect(rows.map((r) => r.getAttribute("data-testid"))).toEqual([
      "msg-jump-item-c",
      "msg-jump-item-b",
      "msg-jump-item-a",
    ]);
  });

  it("[REQ:P0-017g] the User chip also narrows the server search", () => {
    const onOptionsChange = vi.fn();
    render(<MessageJumpList events={[makeEvent({ id: "a", sequence: 1 })]} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search({ onOptionsChange })} />);
    fireEvent.click(screen.getByTestId("msg-nav-chip-user"));
    expect(onOptionsChange).toHaveBeenLastCalledWith({ mode: "text", caseSensitive: false, wholeWord: false, role: "user" });
  });

  it("by-role grouping renders rows under role headings without losing identity", () => {
    const events = [
      makeEvent({ id: "u1", sequence: 1, role: "user", text: "Q1" }),
      makeEvent({ id: "a1", sequence: 2, text: "A1" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-group-role"));
    expect(screen.getByTestId("msg-jump-item-u1")).toBeInTheDocument();
    expect(screen.getByTestId("msg-jump-item-a1")).toBeInTheDocument();
  });

  // ── Keyboard ────────────────────────────────────────────────────────────────

  it("ArrowDown then Enter selects the next visible (filtered) result", () => {
    const events = [
      makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
      makeEvent({ id: "b", sequence: 2, ttsState: "failed" }),
      makeEvent({ id: "c", sequence: 3, ttsState: "rejected" }),
    ];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-status-failed"));
    const list = screen.getByTestId("msg-jump-list");
    fireEvent.keyDown(list, { key: "ArrowDown" });
    fireEvent.keyDown(list, { key: "Enter" });
    expect(onSelect).toHaveBeenCalledWith("c");
  });

  it("Escape closes the navigator", () => {
    const events = [makeEvent({ id: "e1", sequence: 1 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    fireEvent.keyDown(screen.getByTestId("msg-jump-list"), { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  it("ArrowDown in the search input moves into the results", () => {
    const events = [makeEvent({ id: "a", sequence: 1 }), makeEvent({ id: "b", sequence: 2 })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    const input = screen.getByTestId("msg-nav-search");
    fireEvent.keyDown(input, { key: "ArrowDown" });
    const list = screen.getByTestId("msg-jump-list");
    fireEvent.keyDown(list, { key: "Enter" });
    expect(onSelect).toHaveBeenCalledWith("a");
  });

  it("Escape in the search input clears a query before closing", () => {
    const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
    render(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    const input = screen.getByTestId("msg-nav-search");
    fireEvent.change(input, { target: { value: "hello" } });
    fireEvent.keyDown(input, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
    expect((input as HTMLInputElement).value).toBe("");
    fireEvent.keyDown(input, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  // ── Empty states ─────────────────────────────────────────────────────────────

  it("distinguishes empty states by reason", () => {
    const { rerender } = render(<MessageJumpList events={[]} focusedEventId={null} onSelect={onSelect} onClose={onClose} />);
    expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noMessages");

    const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
    rerender(<MessageJumpList events={events} focusedEventId={null} onSelect={onSelect} onClose={onClose} search={search()} />);
    fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "zzz-no-match" } });
    expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noSearchResults");

    fireEvent.click(screen.getByTestId("msg-nav-clear"));
    fireEvent.click(screen.getByTestId("msg-jump-filters"));
    fireEvent.click(screen.getByTestId("msg-nav-status-failed"));
    expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noFilterResults");
  });
});

// ── Export selection mode ─────────────────────────────────────────────────────

describe("MessageJumpList export selection", () => {
  const onSelect = vi.fn();
  const onClose = vi.fn();
  const onContinue = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal(
      "ResizeObserver",
      vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })),
    );
  });

  /**
   * Stateful harness standing in for MessagesPane: owns the selected-ID set
   * exactly the way the pane does so toggle/bulk interactions are observable.
   */
  function Harness({ events }: { events: ConversationEvent[] }) {
    const [selectedIds, setSelectedIds] = useState<ReadonlySet<string>>(new Set());
    const exportSelection: MessageExportSelection = {
      selectedIds,
      onToggle: (id) =>
        setSelectedIds((prev) => {
          const next = new Set(prev);
          if (next.has(id)) next.delete(id);
          else next.add(id);
          return next;
        }),
      onSelectAll: () => setSelectedIds(new Set(events.map((e) => e.id))),
      onSelectVisible: (ids) => setSelectedIds(new Set(ids)),
      onClear: () => setSelectedIds(new Set()),
      onContinue,
    };
    // Like MessagesPane, the harness owns the query and answers it the way the
    // server does (whole-history hits with ranges).
    const [query, setQuery] = useState("");
    const hits = query ? events.filter((e) => e.text.includes(query)).map((e) => hit(e, query)) : [];
    return (
      <MessageJumpList
        events={events}
        focusedEventId={null}
        onSelect={onSelect}
        onClose={onClose}
        exportSelection={exportSelection}
        query={query}
        onQueryChange={setQuery}
        search={search({ hits })}
      />
    );
  }

  const threeEvents = () => [
    makeEvent({ id: "a", sequence: 1, role: "user", text: "question about deploy" }),
    makeEvent({ id: "b", sequence: 2, text: "deploy answer" }),
    makeEvent({ id: "c", sequence: 3, text: "unrelated" }),
  ];

  it("shows a labelled Export action in every normal navigator session", () => {
    render(<Harness events={threeEvents()} />);
    const enter = screen.getByTestId("msg-export-enter");
    expect(enter).toBeInTheDocument();
    expect(enter).toHaveAttribute("aria-label", "messageExport.exportAction");
  });

  it("activating Export enters selection mode without closing the navigator", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByText("messageExport.selectionTitle")).toBeInTheDocument();
    expect(screen.getByTestId("msg-export-footer")).toBeInTheDocument();
  });

  it("selection-mode rows toggle a checkbox and never jump or close", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    const row = screen.getByTestId("msg-jump-item-b");
    expect(row.getAttribute("role")).toBe("checkbox");
    expect(row.getAttribute("aria-checked")).toBe("false");
    fireEvent.click(row);
    expect(screen.getByTestId("msg-jump-item-b").getAttribute("aria-checked")).toBe("true");
    fireEvent.click(screen.getByTestId("msg-jump-item-b"));
    expect(screen.getByTestId("msg-jump-item-b").getAttribute("aria-checked")).toBe("false");
    expect(onSelect).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("retains selected IDs when a filter hides them and counts them as hidden", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    fireEvent.click(screen.getByTestId("msg-jump-item-a"));
    fireEvent.click(screen.getByTestId("msg-jump-item-b"));
    expect(screen.getByTestId("msg-export-count").textContent).toContain("messageExport.selectedCount");

    // Filter to assistant-only: user row "a" disappears but stays selected.
    fireEvent.click(screen.getByTestId("msg-nav-chip-assistant"));
    expect(screen.queryByTestId("msg-jump-item-a")).toBeNull();
    expect(screen.getByTestId("msg-export-hidden-hint")).toBeInTheDocument();

    // Back to all: the selection is still intact.
    fireEvent.click(screen.getByTestId("msg-nav-chip-all"));
    expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
    expect(screen.getByTestId("msg-jump-item-b").getAttribute("aria-checked")).toBe("true");
  });

  it("bulk actions select all, select visible results, and clear", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));

    fireEvent.click(screen.getByTestId("msg-export-select-all"));
    for (const id of ["a", "b", "c"]) {
      expect(screen.getByTestId(`msg-jump-item-${id}`).getAttribute("aria-checked")).toBe("true");
    }

    fireEvent.click(screen.getByTestId("msg-export-clear"));
    for (const id of ["a", "b", "c"]) {
      expect(screen.getByTestId(`msg-jump-item-${id}`).getAttribute("aria-checked")).toBe("false");
    }

    // Search narrows visible results; "visible" selects exactly those.
    fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "deploy" } });
    fireEvent.click(screen.getByTestId("msg-export-select-visible"));
    fireEvent.click(screen.getByTestId("msg-nav-clear"));
    expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
    expect(screen.getByTestId("msg-jump-item-b").getAttribute("aria-checked")).toBe("true");
    expect(screen.getByTestId("msg-jump-item-c").getAttribute("aria-checked")).toBe("false");
  });

  it("Continue is disabled at zero selection and fires the callback once selected", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    const cont = screen.getByTestId("msg-export-continue");
    expect(cont).toBeDisabled();
    fireEvent.click(cont);
    expect(onContinue).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("msg-jump-item-a"));
    expect(screen.getByTestId("msg-export-continue")).not.toBeDisabled();
    fireEvent.click(screen.getByTestId("msg-export-continue"));
    expect(onContinue).toHaveBeenCalledTimes(1);
  });

  it("shows the shared-formatter token estimate in the footer", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    expect(screen.getByTestId("msg-export-tokens").textContent).toContain("messageExport.approxTokens");
  });

  it("Cancel exits selection mode; reentering shows the retained selection", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    fireEvent.click(screen.getByTestId("msg-jump-item-a"));
    fireEvent.click(screen.getByTestId("msg-export-cancel"));
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByTestId("msg-export-footer")).toBeNull();
    expect(screen.queryByText("messageExport.selectionTitle")).toBeNull();

    fireEvent.click(screen.getByTestId("msg-export-enter"));
    expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
  });

  it("Escape steps back to the normal navigator before closing", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    const list = screen.getByTestId("msg-jump-list");
    fireEvent.keyDown(list, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.queryByTestId("msg-export-footer")).toBeNull();
    fireEvent.keyDown(list, { key: "Escape" });
    expect(onClose).toHaveBeenCalled();
  });

  it("keyboard Enter toggles the active row in selection mode instead of jumping", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-export-enter"));
    const list = screen.getByTestId("msg-jump-list");
    fireEvent.keyDown(list, { key: "Enter" });
    expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
    expect(onSelect).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("normal jump behavior is unchanged when export selection is not active", () => {
    render(<Harness events={threeEvents()} />);
    fireEvent.click(screen.getByTestId("msg-jump-item-a"));
    expect(onSelect).toHaveBeenCalledWith("a");
    expect(onClose).toHaveBeenCalled();
  });
});
