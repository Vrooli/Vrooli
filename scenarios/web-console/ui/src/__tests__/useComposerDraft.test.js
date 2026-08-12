import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useComposerDraft } from "../hooks/useComposerDraft";
const DEBOUNCE_MS = 300;
function makeTextarea(value, caret) {
    const el = document.createElement("textarea");
    el.value = value;
    const pos = caret ?? value.length;
    el.setSelectionRange(pos, pos);
    return el;
}
describe("useComposerDraft", () => {
    beforeEach(() => {
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
    });
    it("seeds getValue/initialValue from persisted draft", () => {
        window.localStorage.setItem("wc-mobile-draft-sess-seed", "persisted text");
        const { result } = renderHook(() => useComposerDraft("sess-seed"));
        expect(result.current.getValue()).toBe("persisted text");
        expect(result.current.initialValue).toBe("persisted text");
    });
    it("setValue updates getValue and persists (debounced)", () => {
        vi.useFakeTimers();
        try {
            const { result } = renderHook(() => useComposerDraft("sess-set"));
            act(() => result.current.setValue("hello world"));
            expect(result.current.getValue()).toBe("hello world");
            act(() => vi.advanceTimersByTime(DEBOUNCE_MS));
            expect(window.localStorage.getItem("wc-mobile-draft-sess-set")).toBe("hello world");
        }
        finally {
            vi.useRealTimers();
        }
    });
    it("reset clears value and persistence", () => {
        vi.useFakeTimers();
        try {
            const { result } = renderHook(() => useComposerDraft("sess-reset"));
            act(() => result.current.setValue("abc"));
            act(() => vi.advanceTimersByTime(DEBOUNCE_MS));
            expect(window.localStorage.getItem("wc-mobile-draft-sess-reset")).toBe("abc");
            act(() => result.current.reset());
            expect(result.current.getValue()).toBe("");
            expect(window.localStorage.getItem("wc-mobile-draft-sess-reset")).toBeNull();
        }
        finally {
            vi.useRealTimers();
        }
    });
    it("appendAtCaret appends to end with a leading space (no focused textarea)", () => {
        const { result } = renderHook(() => useComposerDraft("sess-append"));
        act(() => result.current.setValue("Hello."));
        act(() => result.current.appendAtCaret("World."));
        expect(result.current.getValue()).toBe("Hello. World.");
    });
    it("appendAtCaret inserts at last-known caret from trackSelection", () => {
        const { result } = renderHook(() => useComposerDraft("sess-caret"));
        const el = makeTextarea("abc xyz", 3);
        act(() => result.current.handleChange(el)); // sets value + selection to 3
        act(() => result.current.appendAtCaret("MID"));
        // Leading space added after "abc"; next char is a space so no trailing space.
        expect(result.current.getValue()).toBe("abc MID xyz");
    });
    it("handleChange notifies subscribers with reason 'input'", () => {
        const { result } = renderHook(() => useComposerDraft("sess-sub"));
        const changes = [];
        let unsub = () => { };
        act(() => {
            unsub = result.current.subscribe((c) => changes.push(c));
        });
        const el = makeTextarea("typed");
        act(() => result.current.handleChange(el));
        expect(changes.at(-1)).toMatchObject({ value: "typed", reason: "input" });
        act(() => unsub());
    });
    it("subscribers stop receiving after unsubscribe", () => {
        const { result } = renderHook(() => useComposerDraft("sess-unsub"));
        const changes = [];
        let unsub = () => { };
        act(() => {
            unsub = result.current.subscribe((c) => changes.push(c));
        });
        act(() => result.current.setValue("one"));
        act(() => unsub());
        act(() => result.current.setValue("two"));
        expect(changes.map((c) => c.value)).toEqual(["one"]);
    });
    it("reloads the draft when the session id changes and notifies with reason 'set'", () => {
        window.localStorage.setItem("wc-mobile-draft-sess-b", "draft for B");
        const { result, rerender } = renderHook(({ id }) => useComposerDraft(id), {
            initialProps: { id: "sess-a" },
        });
        const changes = [];
        act(() => {
            result.current.subscribe((c) => changes.push(c));
        });
        rerender({ id: "sess-b" });
        expect(result.current.getValue()).toBe("draft for B");
        expect(changes.at(-1)).toMatchObject({ value: "draft for B", reason: "set" });
    });
});
