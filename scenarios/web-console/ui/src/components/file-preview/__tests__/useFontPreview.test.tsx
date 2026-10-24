import { act, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { useFontPreview } from "../useFontPreview";
class PreviewWorker {
  static instances: PreviewWorker[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null; onerror: (() => void) | null = null;
  terminate = vi.fn(); postMessage = vi.fn(); constructor() { PreviewWorker.instances.push(this); }
}
class PreviewFont {
  static instances: PreviewFont[] = []; static resolves = true;
  resolve: (() => void) | undefined; reject: (() => void) | undefined;
  constructor(public family: string) { PreviewFont.instances.push(this); }
  load = vi.fn(() => PreviewFont.resolves ? Promise.resolve(this) : new Promise<this>((resolve, reject) => { this.resolve = () => { resolve(this); }; this.reject = () => { reject(Error("bad font")); }; }));
}
const add = vi.fn(), remove = vi.fn();
function Harness({ href = "/blob", size = 100, attempt = 0 }: { href?: string; size?: number; attempt?: number }) {
  const state = useFontPreview(href, size, attempt); return <output>{state.error || (state.loaded ? `ready:${state.loaded.family}` : "loading")}</output>;
}
const info = { format: "ttf", family: "Test", style: "Regular", glyphs: 1, characters: [65], axes: [], unitsPerEm: 1000 };
function message(data: object, index = 0) { act(() => { PreviewWorker.instances[index]?.onmessage?.(new MessageEvent("message", { data })); }); }
const ready = { type: "ready", bytes: new ArrayBuffer(1), info };
beforeEach(() => { vi.clearAllMocks(); PreviewWorker.instances = []; PreviewFont.instances = []; PreviewFont.resolves = true; vi.stubGlobal("Worker", PreviewWorker); vi.stubGlobal("FontFace", PreviewFont); Object.defineProperty(document, "fonts", { configurable: true, value: { add, delete: remove } }); vi.useFakeTimers(); });
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });
describe("font lifecycle ownership", () => {
  it("rejects oversized fonts and missing browser APIs without starting work", () => {
    const { rerender } = render(<Harness size={100000000} />); expect(screen.getByRole("status")).toHaveTextContent("limit"); expect(PreviewWorker.instances).toHaveLength(0);
    vi.stubGlobal("FontFace", undefined); rerender(<Harness size={100} />); expect(screen.getByRole("status")).toHaveTextContent("unavailable"); expect(PreviewWorker.instances).toHaveLength(0);
  });
  it("loads a private face and releases it on close", async () => {
    const { unmount } = render(<Harness />); message(ready); await act(async () => { await Promise.resolve(); }); expect(add).toHaveBeenCalledWith(PreviewFont.instances[0]); expect(screen.getByRole("status")).toHaveTextContent("ready:wc-preview-font-"); unmount(); expect(remove).toHaveBeenCalledWith(PreviewFont.instances[0]); expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
  });
  it("never registers a late native load after renewal", async () => {
    PreviewFont.resolves = false; const { rerender } = render(<Harness />); message(ready); rerender(<Harness href="/new" />);
    await act(async () => { PreviewFont.instances[0]?.resolve?.(); await Promise.resolve(); }); expect(add).not.toHaveBeenCalled(); expect(screen.getByRole("status")).toHaveTextContent("loading");
    message(ready,1); expect(PreviewFont.instances[1]?.family).not.toBe(PreviewFont.instances[0]?.family);
  });
  it("stops download timeouts and ignores late worker responses", async () => {
    render(<Harness />); const late = PreviewWorker.instances[0]?.onmessage; act(() => { vi.advanceTimersByTime(45000); }); expect(screen.getByRole("status")).toHaveTextContent("fetch");
    await act(async () => { late?.(new MessageEvent("message", { data: ready })); await Promise.resolve(); }); expect(PreviewFont.instances).toHaveLength(0); expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
  });
  it("distinguishes parser and native renderer deadlines", () => {
    const { rerender } = render(<Harness />); message({ type: "parsing" }); act(() => { vi.advanceTimersByTime(10000); }); expect(screen.getByRole("status")).toHaveTextContent("limit");
    PreviewFont.resolves = false; rerender(<Harness attempt={1} />); message(ready,1); act(() => { vi.advanceTimersByTime(15000); }); expect(screen.getByRole("status")).toHaveTextContent("unavailable"); expect(remove).toHaveBeenCalledWith(PreviewFont.instances[0]);
  });
  it("retains browser rejection as invalid and retries with fresh work", async () => {
    PreviewFont.resolves = false; const { rerender } = render(<Harness />); message(ready);
    await act(async () => { PreviewFont.instances[0]?.reject?.(); await Promise.resolve(); }); expect(screen.getByRole("status")).toHaveTextContent("invalid"); rerender(<Harness attempt={1} />); expect(screen.getByRole("status")).toHaveTextContent("loading"); expect(PreviewWorker.instances).toHaveLength(2);
  });
});
