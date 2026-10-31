import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useStructuredData } from "../useStructuredData";
import type { StructureResult } from "../structuredData";

class TestWorker {
  static instances: TestWorker[] = [];
  onmessage: ((event: MessageEvent<StructureResult>) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  postMessage = vi.fn(); terminate = vi.fn();
  constructor() { TestWorker.instances.push(this); }
}
beforeEach(() => { TestWorker.instances = []; vi.stubGlobal("Worker", TestWorker); });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); vi.useRealTimers(); });
function worker() { const current = TestWorker.instances.at(-1); if (!current) throw new Error("Missing worker"); return current; }
describe("configuration parser lifecycle", () => {
  it("parses off the UI thread and terminates the worker after its result", () => {
    const { result } = renderHook(() => useStructuredData('{"x":1}', true, false));
    const parser = worker(); expect(parser.postMessage).toHaveBeenCalledWith({ source: '{"x":1}', json: true });
    expect(result.current).toBeNull();
    act(() => { parser.onmessage?.(new MessageEvent("message", { data: { documents: [] } })); });
    expect(result.current).toEqual({ documents: [] }); expect(parser.terminate).toHaveBeenCalled();
  });
  it("discards stale results when the file changes or the drawer closes", () => {
    const { result, rerender, unmount } = renderHook(({ text }) => useStructuredData(text, true, false), { initialProps: { text: "old" } });
    const old = worker(); rerender({ text: "new" });
    expect(old.terminate).toHaveBeenCalled();
    act(() => { old.onmessage?.(new MessageEvent("message", { data: { error: "invalid" } })); });
    expect(result.current).toBeNull(); const current = worker(); unmount(); expect(current.terminate).toHaveBeenCalled();
  });
  it("keeps truncated content in source and terminates work that exceeds its deadline", () => {
    vi.useFakeTimers();
    const { result, rerender } = renderHook(({ truncated }) => useStructuredData("{}", true, truncated), { initialProps: { truncated: true } });
    expect(result.current).toEqual({ error: "limit" }); expect(TestWorker.instances).toHaveLength(0);
    rerender({ truncated: false }); const parser = worker();
    act(() => { vi.advanceTimersByTime(4000); });
    expect(result.current).toEqual({ error: "limit" }); expect(parser.terminate).toHaveBeenCalled();
  });
  it("uses source when a worker cannot start or fails during parsing", () => {
    const { result } = renderHook(() => useStructuredData("{}", false, false));
    const error = new ErrorEvent("error", { cancelable: true }); act(() => { worker().onerror?.(error); });
    expect(error.defaultPrevented).toBe(true); expect(result.current).toEqual({ error: "invalid" });
    vi.stubGlobal("Worker", undefined);
    const unavailable = renderHook(() => useStructuredData("{}", true, false));
    expect(unavailable.result.current).toEqual({ error: "unsupported" });
  });
});
