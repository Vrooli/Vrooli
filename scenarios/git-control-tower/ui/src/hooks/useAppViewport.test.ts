import { act, renderHook } from "@testing-library/react";
import { createElement, StrictMode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAppViewport } from "./useAppViewport";

const state = vi.hoisted(() => ({ viewport: {
  frameHeight: undefined as number | undefined,
  layoutWidth: 390, layoutHeight: 844, visibleWidth: 390, visibleHeight: 844,
  keyboardInset: 0, keyboardVisible: false, reachesScreenBottom: true,
  offsetLeft: 0, offsetTop: 0, scale: 1,
} }));
vi.mock("@vrooli/react-component-library/useViewportEnvironment/1", () => ({
  useViewportEnvironment: () => state.viewport,
}));
let focusedEditor: HTMLTextAreaElement | undefined;
let owner: HTMLDivElement | undefined;
let nextFrame = 0;
const frames = new Map<number, FrameRequestCallback>();
const setDocumentScroll = (x: number, y: number) => {
  Object.defineProperties(window, { scrollX: { configurable: true, value: x }, scrollY: { configurable: true, value: y } });
};
const flushFrame = () => act(() => {
  const pending = [...frames.values()];
  frames.clear();
  pending.forEach(callback => callback(0));
});
beforeEach(() => {
  vi.spyOn(window, "requestAnimationFrame").mockImplementation(callback => { frames.set(++nextFrame, callback); return nextFrame; });
  vi.spyOn(window, "cancelAnimationFrame").mockImplementation(id => { frames.delete(id); });
  vi.spyOn(window, "scrollTo").mockImplementation(() => setDocumentScroll(0, 0));
});
afterEach(() => {
  focusedEditor?.remove();
  owner?.remove();
  focusedEditor = undefined;
  owner = undefined;
  frames.clear();
  setDocumentScroll(0, 0);
  state.viewport = { ...state.viewport, layoutHeight: 844, keyboardVisible: false, reachesScreenBottom: true, visibleHeight: 844, frameHeight: undefined, offsetLeft: 0, offsetTop: 0, scale: 1 };
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
const focusEditor = () => {
  focusedEditor = document.createElement("textarea");
  document.body.append(focusedEditor);
  focusedEditor.focus();
  return focusedEditor;
};
function ownedEditor(editorTop: number, editorBottom: number) {
  owner = document.createElement("div");
  owner.dataset.gctScrollOwner = "commit";
  owner.style.overflowY = "auto";
  Object.defineProperties(owner, { clientHeight: { value: 230 }, scrollHeight: { value: 1000 } });
  owner.getBoundingClientRect = () => new DOMRect(0, 150 - window.scrollY, 390, 230);
  const editor = focusEditor();
  owner.append(editor);
  document.body.append(owner);
  editor.focus();
  const scroller = owner;
  editor.getBoundingClientRect = () => new DOMRect(0, editorTop - scroller.scrollTop - window.scrollY, 360, editorBottom - editorTop);
  return { scroller, editor };
}
const value = (name: string) => document.documentElement.style.getPropertyValue(name);

describe("mobile viewport ownership", () => {
  it("publishes one effective bottom inset for both navigation and library overlays", () => {
    const { rerender, unmount } = renderHook(useAppViewport);
    expect(value("--gct-app-height")).toBe("844px");
    expect(value("--rcl-safe-bottom")).toBe("env(safe-area-inset-bottom, 0px)");
    state.viewport = { ...state.viewport, reachesScreenBottom: false };
    rerender();
    expect(value("--rcl-safe-bottom")).toBe("0px");
    expect(value("--gct-app-height")).toBe("844px");
    unmount();
    expect(value("--rcl-safe-bottom")).toBe("");
  });

  it("uses the client-measured drawable canvas instead of the shortened installed viewport", () => {
    state.viewport = { ...state.viewport, layoutHeight: 793, visibleHeight: 793, frameHeight: 852 };
    renderHook(useAppViewport);
    expect(value("--gct-app-height")).toBe("852px");
    expect(value("--rcl-safe-bottom")).toBe("env(safe-area-inset-bottom, 0px)");
    state.viewport = { ...state.viewport, layoutHeight: 844 };
  });

  it("uses the keyboard viewport only while the keyboard is established", () => {
    const { rerender } = renderHook(useAppViewport);
    state.viewport = { ...state.viewport, keyboardVisible: true, visibleHeight: 510, reachesScreenBottom: false };
    rerender();
    expect(value("--gct-app-height")).toBe("510px");
    expect(value("--rcl-safe-bottom")).toBe("0px");
    state.viewport = { ...state.viewport, keyboardVisible: false };
    rerender();
    expect(value("--gct-app-height")).toBe("844px");
  });

  it("preserves the browser's pan while a focused editor is zoomed", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    focusedEditor = document.createElement("textarea");
    document.body.append(focusedEditor);
    focusedEditor.focus();
    state.viewport = { ...state.viewport, scale: 4 / 3, visibleHeight: 375, offsetTop: 250, offsetLeft: 12 };
    const { rerender } = renderHook(useAppViewport);
    flushFrame();
    expect(value("--gct-app-height")).toBe("844px");
    expect(scrollTo).not.toHaveBeenCalled();
    state.viewport = { ...state.viewport, offsetTop: 270 };
    rerender();
    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("recovers actual unzoomed document focus scroll even before keyboard detection", () => {
    focusEditor();
    setDocumentScroll(0, 344);
    state.viewport = { ...state.viewport, layoutHeight: 449, visibleHeight: 449, offsetTop: 344, keyboardVisible: false };
    renderHook(useAppViewport);
    flushFrame();
    expect(window.scrollTo).toHaveBeenCalledWith(0, 0);
    expect(window.scrollY).toBe(0);
    expect(value("--gct-app-height")).toBe("449px");
  });

  it("recovers a coalesced document scroll event without a new shared viewport snapshot", () => {
    focusEditor();
    renderHook(useAppViewport);
    flushFrame();
    setDocumentScroll(0, 213);
    window.dispatchEvent(new Event("scroll"));
    window.dispatchEvent(new Event("scroll"));
    expect(frames.size).toBe(1);
    flushFrame();
    expect(window.scrollTo).toHaveBeenCalledOnce();
    expect(window.scrollY).toBe(0);
  });

  it("does not reset a visual offset when the document is already at its origin", () => {
    state.viewport = { ...state.viewport, offsetTop: 344 };
    const { rerender } = renderHook(useAppViewport);
    flushFrame();
    state.viewport = { ...state.viewport, offsetTop: 213 };
    rerender();
    flushFrame();
    expect(window.scrollTo).not.toHaveBeenCalled();
  });

  it("ignores inner-scroll events instead of scheduling page recovery", () => {
    const { scroller } = ownedEditor(180, 230);
    renderHook(useAppViewport);
    flushFrame();
    scroller.dispatchEvent(new Event("scroll", { bubbles: true }));
    expect(frames.size).toBe(0);
    expect(window.scrollTo).not.toHaveBeenCalled();
  });

  it("uses current native zoom when a document scroll precedes shared viewport publication", () => {
    vi.stubGlobal("visualViewport", { scale: 4 / 3 });
    setDocumentScroll(31, 444);
    renderHook(useAppViewport);
    flushFrame();
    expect(window.scrollTo).not.toHaveBeenCalled();
    expect(window.scrollY).toBe(444);
  });

  it("reveals the focused editor inside its owned scroller after keyboard resize", () => {
    const { scroller, editor } = ownedEditor(570, 666);
    scroller.getBoundingClientRect = () => new DOMRect(0, 150 - window.scrollY, 390, state.viewport.keyboardVisible ? 230 : 625);
    const { rerender } = renderHook(useAppViewport);
    flushFrame();
    expect(scroller.scrollTop).toBe(0);
    setDocumentScroll(0, 344);
    state.viewport = { ...state.viewport, keyboardVisible: true, layoutHeight: 449, visibleHeight: 449, reachesScreenBottom: false };
    rerender();
    flushFrame();
    const rect = editor.getBoundingClientRect();
    expect(window.scrollY).toBe(0);
    expect(scroller.scrollTop).toBeGreaterThan(0);
    expect(rect.top).toBeGreaterThanOrEqual(scroller.getBoundingClientRect().top);
    expect(rect.bottom).toBeLessThanOrEqual(scroller.getBoundingClientRect().bottom);
    expect(rect.bottom).toBeLessThan(449);
  });

  it("reveals an owned editor on focus and leaves an already visible field still", () => {
    const { scroller, editor } = ownedEditor(570, 666);
    const { rerender } = renderHook(useAppViewport);
    flushFrame();
    const revealed = scroller.scrollTop;
    expect(revealed).toBeGreaterThan(0);
    editor.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    flushFrame();
    expect(scroller.scrollTop).toBe(revealed);
    state.viewport = { ...state.viewport, visibleHeight: 510, keyboardVisible: true };
    rerender();
    flushFrame();
    expect(scroller.scrollTop).toBe(revealed);
    expect(window.scrollTo).not.toHaveBeenCalled();
  });

  it("keeps a tall textarea's top visible without repeated scroll jitter", () => {
    const { scroller, editor } = ownedEditor(570, 970);
    renderHook(useAppViewport);
    flushFrame();
    expect(editor.getBoundingClientRect().top).toBeGreaterThanOrEqual(150);
    expect(editor.getBoundingClientRect().top).toBeLessThan(380);
    const revealed = scroller.scrollTop;
    editor.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    flushFrame();
    expect(scroller.scrollTop).toBe(revealed);
  });

  it("cleans up document/focus observers and pending recovery under StrictMode", () => {
    const { unmount } = renderHook(useAppViewport, { wrapper: ({ children }) => createElement(StrictMode, null, children) });
    expect(frames.size).toBe(1);
    unmount();
    expect(frames.size).toBe(0);
    setDocumentScroll(0, 344);
    window.dispatchEvent(new Event("scroll"));
    focusEditor();
    flushFrame();
    expect(window.scrollTo).not.toHaveBeenCalled();
    expect(frames.size).toBe(0);
  });

  it("shrinks an unzoomed keyboard canvas without resetting ordinary resize snapshots", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    state.viewport = { ...state.viewport, keyboardVisible: true, visibleHeight: 510, reachesScreenBottom: false };
    renderHook(useAppViewport);
    expect(value("--gct-app-height")).toBe("510px");
    expect(value("--rcl-safe-bottom")).toBe("0px");
    expect(scrollTo).not.toHaveBeenCalled();
  });

  it("corrects idle unzoomed overscroll and resumes correction after zoom ends", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => undefined);
    state.viewport = { ...state.viewport, scale: 1.5, offsetTop: 24 };
    setDocumentScroll(0, 24);
    const { rerender } = renderHook(useAppViewport);
    flushFrame();
    expect(scrollTo).not.toHaveBeenCalled();
    state.viewport = { ...state.viewport, scale: 1 };
    rerender();
    flushFrame();
    expect(scrollTo).toHaveBeenCalledOnce();
    expect(scrollTo).toHaveBeenCalledWith(0, 0);
  });
});
