import { jsx as _jsx } from "react/jsx-runtime";
import { useState } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, within, createEvent } from "@testing-library/react";
import MessageJumpList from "../components/MessageJumpList";
function makeEvent(overrides) {
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
describe("MessageJumpList navigator", () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    beforeEach(() => {
        vi.clearAllMocks();
        vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
    });
    // ── Core rendering ─────────────────────────────────────────────────────────
    it("renders all messages as rows", () => {
        const events = [
            makeEvent({ id: "e1", sequence: 1, role: "user", text: "User question" }),
            makeEvent({ id: "e2", sequence: 2, text: "Assistant answer" }),
            makeEvent({ id: "e3", sequence: 3, text: "Follow up", source: "codex_tailer" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-jump-list")).toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-e1").getAttribute("data-role")).toBe("user");
        expect(screen.getByTestId("msg-jump-item-e2").getAttribute("data-role")).toBe("assistant");
        expect(screen.getByTestId("msg-jump-item-e3")).toBeInTheDocument();
    });
    it("clicking a row calls onSelect and onClose", () => {
        const events = [makeEvent({ id: "e1", sequence: 1, text: "Click me" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-jump-item-e1"));
        expect(onSelect).toHaveBeenCalledWith("e1");
        expect(onClose).toHaveBeenCalled();
    });
    it("highlights the focused event", () => {
        const events = [makeEvent({ id: "e1", sequence: 1 }), makeEvent({ id: "e2", sequence: 2 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "e2", onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-jump-item-e2").className).toContain("bg-wc-accent");
    });
    it("playing event has aria-current and data-glyph='playing'", () => {
        const events = [makeEvent({ id: "p", sequence: 1, ttsState: "playing", text: "Streaming reply" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "p", onSelect: onSelect, onClose: onClose }));
        const item = screen.getByTestId("msg-jump-item-p");
        expect(item.getAttribute("aria-current")).toBe("true");
        expect(item.getAttribute("data-glyph")).toBe("playing");
    });
    it("summarized events render an S badge; next badge appears on the queued-next event", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, summarized: true }),
            makeEvent({ id: "b", sequence: 2 }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "a", onSelect: onSelect, onClose: onClose, hasQueuedNext: true }));
        expect(screen.getByTestId("msg-jump-summarized-a")).toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-next-b")).toBeInTheDocument();
    });
    it("renders safe-area spacer and reserves safe-bottom on the scroll area", () => {
        const events = [makeEvent({ id: "e1", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-jump-safe-spacer").getAttribute("style") ?? "").toContain("safe-bottom");
        expect(screen.getByTestId("msg-jump-scroll").className).toContain("safe-bottom");
    });
    it("assistant rows keep a 44px min tap target; user rows 48px", () => {
        const events = [
            makeEvent({ id: "u", sequence: 1, role: "user", text: "Hi" }),
            makeEvent({ id: "a", sequence: 2 }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-jump-item-a").className).toContain("min-h-[44px]");
        expect(screen.getByTestId("msg-jump-item-u").className).toContain("min-h-[48px]");
    });
    // ── Search ───────────────────────────────────────────────────────────────
    it("renders search input and result count", () => {
        const events = [makeEvent({ id: "e1", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-nav-search")).toBeInTheDocument();
        expect(screen.getByTestId("msg-nav-count")).toBeInTheDocument();
    });
    it("typing a query filters rows and highlights matched text", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, text: "deploy the service" }),
            makeEvent({ id: "b", sequence: 2, text: "unrelated content" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "deploy" } });
        expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
        expect(screen.queryByTestId("msg-jump-item-b")).not.toBeInTheDocument();
        const highlight = within(screen.getByTestId("msg-jump-item-a")).getByText("deploy", { selector: '[data-match="true"]' });
        expect(highlight).toBeInTheDocument();
    });
    it("does not prevent the search input's mousedown default (so clicking focuses it)", () => {
        // The overlay wrapper preventDefaults mousedown to protect host focus; the
        // input must opt out or it can never be focused by click.
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        const input = screen.getByTestId("msg-nav-search");
        const ev = createEvent.mouseDown(input);
        fireEvent(input, ev);
        expect(ev.defaultPrevented).toBe(false);
    });
    it("clear button appears with a query and resets it", () => {
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.queryByTestId("msg-nav-clear")).toBeNull();
        fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "hello" } });
        expect(screen.getByTestId("msg-nav-clear")).toBeInTheDocument();
        fireEvent.click(screen.getByTestId("msg-nav-clear"));
        expect(screen.getByTestId("msg-nav-search").value).toBe("");
    });
    it("uses a controlled query when provided", () => {
        const onQueryChange = vi.fn();
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose, query: "hel", onQueryChange: onQueryChange }));
        expect(screen.getByTestId("msg-nav-search").value).toBe("hel");
        fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "hello" } });
        expect(onQueryChange).toHaveBeenCalledWith("hello");
    });
    // ── Primary chips ──────────────────────────────────────────────────────────
    it("User chip filters to user messages; Assistant chip to assistant messages", () => {
        const events = [
            makeEvent({ id: "u", sequence: 1, role: "user", text: "Q" }),
            makeEvent({ id: "a", sequence: 2, text: "A" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-chip-user"));
        expect(screen.getByTestId("msg-jump-item-u")).toBeInTheDocument();
        expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
        fireEvent.click(screen.getByTestId("msg-nav-chip-assistant"));
        expect(screen.queryByTestId("msg-jump-item-u")).not.toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
    });
    it("Failed chip shows only failed/rejected; All resets", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, ttsState: "played" }),
            makeEvent({ id: "b", sequence: 2, ttsState: "failed" }),
            makeEvent({ id: "c", sequence: 3, ttsState: "rejected" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-chip-failed"));
        expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-b")).toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-c")).toBeInTheDocument();
        fireEvent.click(screen.getByTestId("msg-nav-chip-all"));
        expect(screen.getByTestId("msg-jump-item-a")).toBeInTheDocument();
    });
    // ── Advanced panel ─────────────────────────────────────────────────────────
    it("More toggles the advanced panel with source/status/content/sort/group controls", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, source: "claude_hook" }),
            makeEvent({ id: "b", sequence: 2, source: "grok_tailer" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.queryByTestId("msg-nav-advanced")).toBeNull();
        fireEvent.click(screen.getByTestId("msg-nav-more"));
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
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-more"));
        fireEvent.click(screen.getByTestId("msg-nav-source-grok"));
        expect(screen.queryByTestId("msg-jump-item-a")).not.toBeInTheDocument();
        expect(screen.getByTestId("msg-jump-item-b")).toBeInTheDocument();
    });
    it("content filter narrows to code messages", () => {
        const events = [
            makeEvent({ id: "a", sequence: 1, text: "run `make test`" }),
            makeEvent({ id: "b", sequence: 2, text: "plain prose" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-more"));
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
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-more"));
        fireEvent.click(screen.getByTestId("msg-nav-group-flat"));
        fireEvent.click(screen.getByTestId("msg-nav-sort-newest"));
        const rows = screen.getAllByTestId(/^msg-jump-item-/);
        expect(rows.map((r) => r.getAttribute("data-testid"))).toEqual([
            "msg-jump-item-c",
            "msg-jump-item-b",
            "msg-jump-item-a",
        ]);
    });
    it("relevance sort is disabled until a query exists", () => {
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-more"));
        expect(screen.getByTestId("msg-nav-sort-relevance")).toBeDisabled();
        fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "hello" } });
        expect(screen.getByTestId("msg-nav-sort-relevance")).not.toBeDisabled();
    });
    it("by-role grouping renders rows under role headings without losing identity", () => {
        const events = [
            makeEvent({ id: "u1", sequence: 1, role: "user", text: "Q1" }),
            makeEvent({ id: "a1", sequence: 2, text: "A1" }),
        ];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-more"));
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
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.click(screen.getByTestId("msg-nav-chip-failed"));
        const list = screen.getByTestId("msg-jump-list");
        fireEvent.keyDown(list, { key: "ArrowDown" });
        fireEvent.keyDown(list, { key: "Enter" });
        expect(onSelect).toHaveBeenCalledWith("c");
    });
    it("Escape closes the navigator", () => {
        const events = [makeEvent({ id: "e1", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.keyDown(screen.getByTestId("msg-jump-list"), { key: "Escape" });
        expect(onClose).toHaveBeenCalled();
    });
    it("ArrowDown in the search input moves into the results", () => {
        const events = [makeEvent({ id: "a", sequence: 1 }), makeEvent({ id: "b", sequence: 2 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        const input = screen.getByTestId("msg-nav-search");
        fireEvent.keyDown(input, { key: "ArrowDown" });
        const list = screen.getByTestId("msg-jump-list");
        fireEvent.keyDown(list, { key: "Enter" });
        expect(onSelect).toHaveBeenCalledWith("a");
    });
    it("Escape in the search input clears a query before closing", () => {
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        const input = screen.getByTestId("msg-nav-search");
        fireEvent.change(input, { target: { value: "hello" } });
        fireEvent.keyDown(input, { key: "Escape" });
        expect(onClose).not.toHaveBeenCalled();
        expect(input.value).toBe("");
        fireEvent.keyDown(input, { key: "Escape" });
        expect(onClose).toHaveBeenCalled();
    });
    // ── Empty states ─────────────────────────────────────────────────────────────
    it("distinguishes empty states by reason", () => {
        const { rerender } = render(_jsx(MessageJumpList, { events: [], focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noMessages");
        const events = [makeEvent({ id: "a", sequence: 1, text: "hello" })];
        rerender(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        fireEvent.change(screen.getByTestId("msg-nav-search"), { target: { value: "zzz-no-match" } });
        expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noSearchResults");
        fireEvent.click(screen.getByTestId("msg-nav-clear"));
        fireEvent.click(screen.getByTestId("msg-nav-chip-failed"));
        expect(screen.getByTestId("msg-nav-empty").getAttribute("data-reason")).toBe("noFilterResults");
    });
    // ── Mode-specific copy + now-playing ───────────────────────────────────────
    it("uses jump-mode title by default and playback title in playback-select mode", () => {
        const events = [makeEvent({ id: "a", sequence: 1 })];
        const { rerender } = render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose }));
        expect(screen.getByText("messageJumpList.titleJump")).toBeInTheDocument();
        rerender(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose, mode: "playback-select" }));
        expect(screen.getByText("messageJumpList.titlePlayback")).toBeInTheDocument();
    });
    it("hides the now-playing header in jump mode without active playback", () => {
        const events = [makeEvent({ id: "a", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "a", onSelect: onSelect, onClose: onClose }));
        expect(screen.queryByTestId("msg-jump-now-playing")).toBeNull();
    });
    it("shows the now-playing header (idle) in playback-select mode", () => {
        const events = [makeEvent({ id: "a", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "a", onSelect: onSelect, onClose: onClose, mode: "playback-select" }));
        expect(screen.getByTestId("msg-jump-now-playing").getAttribute("data-state")).toBe("idle");
    });
    it("now-playing header shows scrub and play/pause when duration is set", () => {
        const onPause = vi.fn();
        const events = [makeEvent({ id: "a", sequence: 1, text: "Now playing" })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "a", onSelect: onSelect, onClose: onClose, currentTime: 5, duration: 20, isPaused: false, onPause: onPause }));
        expect(screen.getByTestId("msg-jump-now-playing").getAttribute("data-state")).toBe("playing");
        expect(screen.getByTestId("msg-jump-now-scrub")).toBeInTheDocument();
        fireEvent.click(screen.getByTestId("msg-jump-now-playpause"));
        expect(onPause).toHaveBeenCalled();
    });
    it("scrub bar uses summarized accent when isSummarized=true", () => {
        const events = [makeEvent({ id: "a", sequence: 1 })];
        render(_jsx(MessageJumpList, { events: events, focusedEventId: "a", onSelect: onSelect, onClose: onClose, currentTime: 1, duration: 10, isPaused: false, isSummarized: true }));
        expect(screen.getByTestId("msg-jump-now-scrub").className).toMatch(/amber-400/);
    });
});
// ── Export selection mode ─────────────────────────────────────────────────────
describe("MessageJumpList export selection", () => {
    const onSelect = vi.fn();
    const onClose = vi.fn();
    const onContinue = vi.fn();
    beforeEach(() => {
        vi.clearAllMocks();
        vi.stubGlobal("ResizeObserver", vi.fn().mockImplementation(() => ({ observe: vi.fn(), unobserve: vi.fn(), disconnect: vi.fn() })));
    });
    /**
     * Stateful harness standing in for MessagesPane: owns the selected-ID set
     * exactly the way the pane does so toggle/bulk interactions are observable.
     */
    function Harness({ events }) {
        const [selectedIds, setSelectedIds] = useState(new Set());
        const exportSelection = {
            selectedIds,
            onToggle: (id) => setSelectedIds((prev) => {
                const next = new Set(prev);
                if (next.has(id))
                    next.delete(id);
                else
                    next.add(id);
                return next;
            }),
            onSelectAll: () => setSelectedIds(new Set(events.map((e) => e.id))),
            onSelectVisible: (ids) => setSelectedIds(new Set(ids)),
            onClear: () => setSelectedIds(new Set()),
            onContinue,
        };
        return (_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose, exportSelection: exportSelection }));
    }
    const threeEvents = () => [
        makeEvent({ id: "a", sequence: 1, role: "user", text: "question about deploy" }),
        makeEvent({ id: "b", sequence: 2, text: "deploy answer" }),
        makeEvent({ id: "c", sequence: 3, text: "unrelated" }),
    ];
    it("shows a labelled Export action in every normal navigator session", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        const enter = screen.getByTestId("msg-export-enter");
        expect(enter).toBeInTheDocument();
        expect(enter.textContent).toContain("messageExport.exportAction");
    });
    it("hides the Export action in playback-select mode", () => {
        const events = threeEvents();
        render(_jsx(MessageJumpList, { events: events, focusedEventId: null, onSelect: onSelect, onClose: onClose, mode: "playback-select", exportSelection: {
                selectedIds: new Set(),
                onToggle: vi.fn(),
                onSelectAll: vi.fn(),
                onSelectVisible: vi.fn(),
                onClear: vi.fn(),
                onContinue,
            } }));
        expect(screen.queryByTestId("msg-export-enter")).toBeNull();
    });
    it("activating Export enters selection mode without closing the navigator", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.getByText("messageExport.selectionTitle")).toBeInTheDocument();
        expect(screen.getByTestId("msg-export-footer")).toBeInTheDocument();
    });
    it("selection-mode rows toggle a checkbox and never jump or close", () => {
        render(_jsx(Harness, { events: threeEvents() }));
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
        render(_jsx(Harness, { events: threeEvents() }));
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
        render(_jsx(Harness, { events: threeEvents() }));
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
        render(_jsx(Harness, { events: threeEvents() }));
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
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        expect(screen.getByTestId("msg-export-tokens").textContent).toContain("messageExport.approxTokens");
    });
    it("Cancel exits selection mode; reentering shows the retained selection", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        fireEvent.click(screen.getByTestId("msg-jump-item-a"));
        fireEvent.click(screen.getByTestId("msg-export-cancel"));
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.queryByTestId("msg-export-footer")).toBeNull();
        expect(screen.getByText("messageJumpList.titleJump")).toBeInTheDocument();
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
    });
    it("Escape steps back to the normal navigator before closing", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        const list = screen.getByTestId("msg-jump-list");
        fireEvent.keyDown(list, { key: "Escape" });
        expect(onClose).not.toHaveBeenCalled();
        expect(screen.queryByTestId("msg-export-footer")).toBeNull();
        fireEvent.keyDown(list, { key: "Escape" });
        expect(onClose).toHaveBeenCalled();
    });
    it("keyboard Enter toggles the active row in selection mode instead of jumping", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-export-enter"));
        const list = screen.getByTestId("msg-jump-list");
        fireEvent.keyDown(list, { key: "Enter" });
        expect(screen.getByTestId("msg-jump-item-a").getAttribute("aria-checked")).toBe("true");
        expect(onSelect).not.toHaveBeenCalled();
        expect(onClose).not.toHaveBeenCalled();
    });
    it("normal jump behavior is unchanged when export selection is not active", () => {
        render(_jsx(Harness, { events: threeEvents() }));
        fireEvent.click(screen.getByTestId("msg-jump-item-a"));
        expect(onSelect).toHaveBeenCalledWith("a");
        expect(onClose).toHaveBeenCalled();
    });
});
