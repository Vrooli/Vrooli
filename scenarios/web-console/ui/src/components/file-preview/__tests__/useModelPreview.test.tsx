import { useRef } from "react";
import { act, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { useModelPreview } from "../useModelPreview";
import { createModelStage } from "../modelStage";
vi.mock("../modelStage", () => ({ createModelStage: vi.fn() }));
class PreviewWorker {
  static instances: PreviewWorker[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null; onerror: (() => void) | null = null;
  terminate = vi.fn(); postMessage = vi.fn(); constructor() { PreviewWorker.instances.push(this); }
}
const stage = { options: vi.fn(), view: vi.fn(), zoom: vi.fn(), orbit: vi.fn(), dispose: vi.fn() };
function Harness({ href = "/blob", size = 100, attempt = 0 }: { href?: string; size?: number; attempt?: number }) {
  const ref = useRef<HTMLDivElement>(null), state = useModelPreview(ref, href, "glb", size, attempt);
  return <><div ref={ref} /><output>{state.error || (state.stats ? "ready" : "loading")}</output></>;
}
function message(data: object, index = 0) { act(() => { PreviewWorker.instances[index]?.onmessage?.(new MessageEvent("message", { data })); }); }
const prepared = { type: "ready", data: { format: "stl", positions: new Float32Array(9), partial: false } };
beforeEach(() => { PreviewWorker.instances = []; vi.stubGlobal("Worker", PreviewWorker); vi.mocked(createModelStage).mockReset().mockImplementation(() => new Promise(() => {})); vi.useFakeTimers(); });
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });
describe("3D preview resource ownership", () => {
  it("rejects oversized models without starting a parser", () => { render(<Harness size={100000000} />); expect(screen.getByRole("status")).toHaveTextContent("limit"); expect(PreviewWorker.instances).toHaveLength(0); });
  it("terminates timed-out work and ignores its late success", async () => {
    render(<Harness />); const late = PreviewWorker.instances[0]?.onmessage; act(() => { vi.advanceTimersByTime(45000); }); expect(screen.getByRole("status")).toHaveTextContent("fetch"); expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
    await act(async () => { late?.(new MessageEvent("message", { data: prepared })); await Promise.resolve(); }); expect(createModelStage).not.toHaveBeenCalled();
  });
  it("aborts and discards a renderer that finishes after source renewal", async () => {
    let finish: (value: typeof stage) => void = () => { throw Error("Missing stage"); };
    vi.mocked(createModelStage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const { rerender } = render(<Harness />); message(prepared); await act(async () => { await Promise.resolve(); });
    const signal = vi.mocked(createModelStage).mock.lastCall?.[2]; rerender(<Harness href="/new" />); expect(signal?.aborted).toBe(true);
    await act(async () => { finish(stage); await Promise.resolve(); }); expect(stage.dispose).toHaveBeenCalled(); expect(screen.getByRole("status")).toHaveTextContent("loading");
  });
  it("publishes geometry only from an active stage and disposes on close", async () => {
    vi.mocked(createModelStage).mockImplementation((_node, _data, _signal, report) => { report({ meshes: 1, triangles: 1, dimensions: [1, 1, 1] }); return Promise.resolve(stage); });
    const { unmount } = render(<Harness />); message(prepared); await act(async () => { await Promise.resolve(); }); expect(screen.getByRole("status")).toHaveTextContent("ready"); unmount(); expect(stage.dispose).toHaveBeenCalled(); expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
  });
  it("distinguishes a processing limit from a download timeout", () => {
    render(<Harness />); message({ type: "parsing" }); act(() => { vi.advanceTimersByTime(10000); }); expect(screen.getByRole("status")).toHaveTextContent("limit");
  });
  it("supports retry after parser failure with a fresh worker", () => {
    const { rerender } = render(<Harness />); message({ type: "error", error: "external" }); expect(screen.getByRole("status")).toHaveTextContent("external"); rerender(<Harness attempt={1} />); expect(screen.getByRole("status")).toHaveTextContent("loading"); expect(PreviewWorker.instances).toHaveLength(2);
  });
});
