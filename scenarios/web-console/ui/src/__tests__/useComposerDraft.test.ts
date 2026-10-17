import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useComposerDraft, type ComposerDraftChange } from "../hooks/useComposerDraft";

const DEBOUNCE_MS = 300;

function makeTextarea(value: string, caret?: number): HTMLTextAreaElement {
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
    } catch {
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
    } finally {
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
    } finally {
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
    const changes: ComposerDraftChange[] = [];
    let unsub = () => {};
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
    const changes: ComposerDraftChange[] = [];
    let unsub = () => {};
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
    const changes: ComposerDraftChange[] = [];
    act(() => {
      result.current.subscribe((c) => changes.push(c));
    });
    rerender({ id: "sess-b" });
    expect(result.current.getValue()).toBe("draft for B");
    expect(changes.at(-1)).toMatchObject({ value: "draft for B", reason: "set" });
  });
});

/**
 * A draft belongs to the session it was typed in, so every write and clear must
 * target THAT session — not whichever session happens to be active when the
 * operation lands. Both race in practice: persistence is debounced, and a send
 * only clears once its stdin_ack settles, which can arrive after a switch.
 */
describe("useComposerDraft — session ownership of writes and clears", () => {
  beforeEach(() => {
    try {
      window.localStorage.clear();
    } catch {
      /* no-op */
    }
    vi.useFakeTimers();
  });

  const read = (id: string) => window.localStorage.getItem(`wc-mobile-draft-${id}`);

  function setup(initial = "sess-a") {
    const h = renderHook(({ id }) => useComposerDraft(id), { initialProps: { id: initial } });
    // Fake timers defer React's passive effects; act() forces the flush.
    const switchTo = (id: string) => act(() => { h.rerender({ id }); });
    return { ...h, switchTo };
  }

  it("keeps a draft written just before switching away", () => {
    const { result, switchTo } = setup();
    act(() => result.current.setValue("text in A"));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS - 200); });
    switchTo("sess-b");
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });
    expect(read("sess-a")).toBe("text in A");
  });

  it("does not let typing in the new session drop the previous session's pending write", () => {
    const { result, switchTo } = setup();
    act(() => result.current.setValue("text in A"));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS - 200); });
    switchTo("sess-b");
    act(() => result.current.setValue("text in B"));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });
    expect(read("sess-a")).toBe("text in A");
    expect(read("sess-b")).toBe("text in B");
  });

  it("does not let a send in the new session drop the previous session's pending write", () => {
    const { result, switchTo } = setup();
    act(() => result.current.setValue("text in A"));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS - 200); });
    switchTo("sess-b");
    act(() => result.current.reset());
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });
    expect(read("sess-a")).toBe("text in A");
  });

  it("restores each session's own draft when switching back and forth", () => {
    const { result, switchTo } = setup();
    act(() => result.current.setValue("text in A"));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS - 200); });
    switchTo("sess-b");
    act(() => result.current.setValue("text in B"));
    switchTo("sess-a");
    expect(result.current.getValue()).toBe("text in A");
    switchTo("sess-b");
    expect(result.current.getValue()).toBe("text in B");
  });

  it("clears the session a send came from when its ack settles after a switch", () => {
    window.localStorage.setItem("wc-mobile-draft-sess-b", "draft for B");
    const { result, switchTo } = setup();
    act(() => result.current.setValue("text in A"));
    const sentFrom = result.current.getSessionId();
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });

    switchTo("sess-b");
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });
    expect(result.current.getValue()).toBe("draft for B");

    // The late ack: finalizeSuccess() -> draft.reset(sentFrom)
    act(() => result.current.reset(sentFrom));
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });

    expect(read("sess-a")).toBeNull();
    expect(read("sess-b")).toBe("draft for B");
    expect(result.current.getValue()).toBe("draft for B");
  });

  it("still clears normally when the send settles in its own session", () => {
    const { result } = setup();
    act(() => result.current.setValue("text in A"));
    const sentFrom = result.current.getSessionId();
    act(() => { vi.advanceTimersByTime(DEBOUNCE_MS * 2); });
    act(() => result.current.reset(sentFrom));
    expect(result.current.getValue()).toBe("");
    expect(read("sess-a")).toBeNull();
  });
});
