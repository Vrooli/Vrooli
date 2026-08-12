import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
import { createRef } from "react";
import MobileToolbar from "../components/MobileToolbar";
import { i18n } from "../i18n";
// Draft persistence wants a stable sessionId — pass a fixed one through props.
function renderToolbar(overrides = {}) {
    const onInput = overrides.onInput ?? vi.fn(() => ({ status: "sent", seq: 1 }));
    const settledSubs = new Set();
    const fireSettled = (seq, ok) => {
        for (const cb of settledSubs)
            cb(seq, ok);
    };
    const subscribeInputSettled = overrides.subscribeInputSettled ??
        vi.fn((cb) => {
            settledSubs.add(cb);
            return () => settledSubs.delete(cb);
        });
    const pendingSubs = new Set();
    let snapshot = [];
    const setSnapshot = (next) => {
        snapshot = next;
        for (const cb of pendingSubs)
            cb();
    };
    const subscribePendingInput = overrides.subscribePendingInput ??
        vi.fn((cb) => {
            pendingSubs.add(cb);
            return () => pendingSubs.delete(cb);
        });
    const getPendingInputSnapshot = overrides.getPendingInputSnapshot ?? vi.fn(() => snapshot);
    const utils = render(_jsx(MobileToolbar, { onInput: onInput, subscribeInputSettled: subscribeInputSettled, subscribePendingInput: subscribePendingInput, getPendingInputSnapshot: getPendingInputSnapshot, activeSessionId: overrides.activeSessionId ?? "sess-1", onFocusTerminal: overrides.onFocusTerminal ?? vi.fn(), ...overrides }));
    return { ...utils, onInput, fireSettled, setSnapshot };
}
describe("MobileToolbar — send/ack flow", () => {
    beforeEach(() => {
        // Reset draft persistence across tests so inputs start empty.
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
    });
    it("preserves draft during sending and clears on ok=true", () => {
        const { onInput, fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent", seq: 1 })) });
        const textarea = screen.getByTestId("mobile-command-input");
        fireEvent.change(textarea, { target: { value: "echo hi" } });
        expect(textarea.value).toBe("echo hi");
        fireEvent.click(screen.getByTestId("mobile-command-submit"));
        expect(onInput).toHaveBeenCalledWith("echo hi", "toolbar-submit");
        // Draft is kept visible during sending.
        expect(textarea.value).toBe("echo hi");
        expect(screen.getByTestId("send-status-sending")).toBeTruthy();
        act(() => fireSettled(1, true));
        // On success, draft clears and status switches to "sent".
        expect(textarea.value).toBe("");
    });
    it("restores editable draft and shows Send failed on ok=false", () => {
        const { fireSettled } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent", seq: 1 })) });
        const textarea = screen.getByTestId("mobile-command-input");
        fireEvent.change(textarea, { target: { value: "long payload" } });
        fireEvent.click(screen.getByTestId("mobile-command-submit"));
        expect(textarea.value).toBe("long payload");
        act(() => fireSettled(1, false));
        // Draft is still in the box so the user can retry.
        expect(textarea.value).toBe("long payload");
        expect(screen.getByTestId("send-status-failed")).toBeTruthy();
    });
    it("keeps queued status when onInput returns queued", () => {
        renderToolbar({ onInput: vi.fn(() => ({ status: "queued", reason: "not-ready" })) });
        const textarea = screen.getByTestId("mobile-command-input");
        fireEvent.change(textarea, { target: { value: "queued-cmd" } });
        fireEvent.click(screen.getByTestId("mobile-command-submit"));
        // Draft preserved (user still needs to retry).
        expect(textarea.value).toBe("queued-cmd");
        expect(screen.getByTestId("send-status-queued")).toBeTruthy();
    });
    it("renders N unsent pill when queue non-empty and hides when drained", async () => {
        // Opt into the real `en` locale so the `{{count}}` interpolation in
        // the unsent-pill heading renders an actual digit — cimode otherwise
        // returns the raw key path with the token unsubstituted.
        await i18n.changeLanguage("en");
        const { setSnapshot } = renderToolbar({ onInput: vi.fn(() => ({ status: "sent", seq: 1 })) });
        expect(screen.queryByTestId("pending-input-pill")).toBeNull();
        act(() => setSnapshot([
            { data: "ls", addedAt: Date.now() - 3000 },
            { data: "pwd", addedAt: Date.now() },
        ]));
        const pill = screen.getByTestId("pending-input-pill");
        expect(pill.textContent).toMatch(/2 unsent/);
        act(() => setSnapshot([]));
        expect(screen.queryByTestId("pending-input-pill")).toBeNull();
    });
});
describe("MobileToolbar — arrow hold-to-repeat", () => {
    beforeEach(() => {
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
        vi.useFakeTimers();
    });
    afterEach(() => {
        vi.useRealTimers();
    });
    // CSI escapes mirrored from toolbar-keys.ts so assertions stay legible.
    const ARROW_UP_INPUT = "\x1b[A";
    const ARROW_LEFT_INPUT = "\x1b[D";
    // Arrow labels render as Unicode glyphs that slugify() strips, so we
    // query by visible button text instead of by data-testid.
    const getArrow = (glyph) => screen.getByRole("button", { name: glyph });
    it("arrow fires once on pointerdown (no release needed)", () => {
        const { onInput } = renderToolbar();
        const up = getArrow("↑");
        act(() => {
            fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
        });
        expect(onInput).toHaveBeenCalledWith(ARROW_UP_INPUT, "toolbar-key");
        expect(onInput).toHaveBeenCalledTimes(1);
    });
    it("arrow repeats while held after the initial delay", () => {
        const { onInput } = renderToolbar();
        const up = getArrow("↑");
        act(() => {
            fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
        });
        expect(onInput).toHaveBeenCalledTimes(1);
        // Advance past the initial delay plus three repeat intervals.
        act(() => {
            vi.advanceTimersByTime(400 + 40 * 3);
        });
        expect(onInput).toHaveBeenCalledTimes(4);
        const mock = onInput;
        for (const call of mock.mock.calls) {
            expect(call).toEqual([ARROW_UP_INPUT, "toolbar-key"]);
        }
    });
    it("pointerup stops the repeat stream", () => {
        const { onInput } = renderToolbar();
        const left = getArrow("←");
        const mock = onInput;
        act(() => {
            fireEvent.pointerDown(left, { pointerType: "touch", button: 0 });
        });
        act(() => {
            vi.advanceTimersByTime(500);
        });
        const countAtRelease = mock.mock.calls.length;
        act(() => {
            fireEvent.pointerUp(left);
        });
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(onInput).toHaveBeenCalledTimes(countAtRelease);
        expect(mock.mock.calls[0]).toEqual([ARROW_LEFT_INPUT, "toolbar-key"]);
    });
    it("pointerleave (finger dragged off) stops repeats", () => {
        const { onInput } = renderToolbar();
        const up = getArrow("↑");
        const mock = onInput;
        act(() => {
            fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
        });
        act(() => {
            vi.advanceTimersByTime(500);
        });
        const countAtLeave = mock.mock.calls.length;
        act(() => {
            fireEvent.pointerLeave(up);
        });
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(onInput).toHaveBeenCalledTimes(countAtLeave);
    });
    it("quick tap on arrow fires exactly once (no phantom repeat after release)", () => {
        const { onInput } = renderToolbar();
        const up = getArrow("↑");
        act(() => {
            fireEvent.pointerDown(up, { pointerType: "touch", button: 0 });
        });
        act(() => {
            vi.advanceTimersByTime(50);
        });
        act(() => {
            fireEvent.pointerUp(up);
        });
        // A synthetic click may still follow on some browsers — verify it's ignored.
        act(() => {
            fireEvent.click(up);
        });
        act(() => {
            vi.advanceTimersByTime(2000);
        });
        expect(onInput).toHaveBeenCalledTimes(1);
    });
    it("non-arrow toolbar keys keep click semantics (no pointerdown fire)", () => {
        const { onInput } = renderToolbar();
        const esc = screen.getByTestId("toolbar-key-esc");
        act(() => {
            fireEvent.pointerDown(esc);
        });
        act(() => {
            vi.advanceTimersByTime(2000);
        });
        expect(onInput).not.toHaveBeenCalled();
        act(() => {
            fireEvent.click(esc);
        });
        expect(onInput).toHaveBeenCalledTimes(1);
        expect(onInput).toHaveBeenCalledWith("\x1b", "toolbar-key");
    });
});
describe("MobileToolbar — command textbox backspace", () => {
    let originalMaxTouchPoints;
    beforeEach(() => {
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
        // Pretend we're on a touch device — the bug only ever manifested there.
        originalMaxTouchPoints = navigator.maxTouchPoints;
        Object.defineProperty(navigator, "maxTouchPoints", {
            value: 1,
            configurable: true,
        });
    });
    afterEach(() => {
        Object.defineProperty(navigator, "maxTouchPoints", {
            value: originalMaxTouchPoints,
            configurable: true,
        });
    });
    // Backspace in a plain textarea is natively supported on every device — tap
    // deletes one char, hold repeats at the OS rate. The toolbar must NOT
    // intercept it (the old custom velocity-repeat belonged to the terminal,
    // whose xterm dependency lacks native key-repeat, and over-deleted here).
    it("leaves backspace to the browser (does not preventDefault the delete event)", () => {
        renderToolbar();
        const textarea = screen.getByTestId("mobile-command-input");
        fireEvent.change(textarea, { target: { value: "abcdef" } });
        textarea.focus();
        textarea.setSelectionRange(6, 6);
        const event = new InputEvent("beforeinput", {
            inputType: "deleteContentBackward",
            bubbles: true,
            cancelable: true,
        });
        act(() => {
            textarea.dispatchEvent(event);
        });
        // Not prevented → the browser performs its own native single-char delete.
        expect(event.defaultPrevented).toBe(false);
    });
});
describe("MobileToolbar — appendText (voice transcript insertion)", () => {
    beforeEach(() => {
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
    });
    function renderWithRef() {
        const ref = createRef();
        const utils = render(_jsx(MobileToolbar, { ref: ref, onInput: vi.fn(() => ({ status: "sent", seq: 1 })), activeSessionId: "sess-append", onFocusTerminal: vi.fn() }));
        const textarea = screen.getByTestId("mobile-command-input");
        return { ref, textarea, ...utils };
    }
    function getToolbarHandle(ref) {
        if (!ref.current)
            throw new Error("MobileToolbar ref was not attached");
        return ref.current;
    }
    // RAF used by appendText for caret restoration — flush it synchronously.
    async function flushRaf() {
        await act(async () => {
            await new Promise((r) => requestAnimationFrame(() => r()));
        });
    }
    it("appends to end with a leading space when prior text has no trailing whitespace", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "Hello." } });
        act(() => getToolbarHandle(ref).appendText("World."));
        await flushRaf();
        expect(textarea.value).toBe("Hello. World.");
    });
    it("does not double-space when prior text already ends in whitespace", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "Hello. " } });
        act(() => getToolbarHandle(ref).appendText("World."));
        await flushRaf();
        expect(textarea.value).toBe("Hello. World.");
    });
    it("inserts at the caret position rather than always at the end", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "abc xyz" } });
        // Move caret to between "abc" and " xyz" (index 3).
        textarea.focus();
        textarea.setSelectionRange(3, 3);
        fireEvent.select(textarea);
        act(() => getToolbarHandle(ref).appendText("MID"));
        await flushRaf();
        // Leading: prev char is "c" (non-ws) → add space. Trailing: next char is " " → no space.
        expect(textarea.value).toBe("abc MID xyz");
        // Caret lands at the end of the inserted text ("abc MID".length = 7).
        expect(textarea.selectionStart).toBe(7);
        expect(textarea.selectionEnd).toBe(7);
    });
    it("replaces the selected range when a selection is active", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "keep DROP keep" } });
        textarea.focus();
        textarea.setSelectionRange(5, 9); // selects "DROP"
        fireEvent.select(textarea);
        act(() => getToolbarHandle(ref).appendText("NEW"));
        await flushRaf();
        expect(textarea.value).toBe("keep NEW keep");
    });
    it("uses the last-known caret even after focus has moved away (e.g. to mic button)", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "abc xyz" } });
        textarea.focus();
        textarea.setSelectionRange(3, 3);
        fireEvent.select(textarea);
        // Simulate focus moving to the mic button — blur fires and we record selection.
        fireEvent.blur(textarea);
        act(() => getToolbarHandle(ref).appendText("MID"));
        await flushRaf();
        expect(textarea.value).toBe("abc MID xyz");
    });
    it("does not add a leading space when inserting at position 0", async () => {
        const { ref, textarea } = renderWithRef();
        fireEvent.change(textarea, { target: { value: "world" } });
        textarea.focus();
        textarea.setSelectionRange(0, 0);
        fireEvent.select(textarea);
        act(() => getToolbarHandle(ref).appendText("hello"));
        await flushRaf();
        // Trailing: next char "w" is non-ws → add trailing space.
        expect(textarea.value).toBe("hello world");
    });
});
