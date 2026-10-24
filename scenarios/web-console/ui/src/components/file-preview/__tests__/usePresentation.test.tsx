import { act, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { usePresentation } from "../usePresentation";
import { loadPresentationFrame } from "../presentationFrame";
import { isPresentationEvent } from "../publicationData";
vi.mock("../presentationFrame", () => ({ loadPresentationFrame: vi.fn() }));
class PreviewWorker {
  static instances: PreviewWorker[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  postMessage = vi.fn(); terminate = vi.fn();
  constructor() { PreviewWorker.instances.push(this); }
}
function Harness({ href = "/blob", legacy = false }: { href?: string; legacy?: boolean }) {
  const state = usePresentation(href, 100, legacy, 0, vi.fn());
  return <><iframe ref={state.frame} title="Test slides" /><div data-testid="source">{state.source}</div><output>{state.error ?? (state.metadata ? "ready" : "loading")}</output></>;
}
const metadata = { width: 960, height: 540, warnings: false, slides: [{ title: "A slide", text: "Text", notes: "", hidden: false }] };
function frame() { return screen.getByTitle<HTMLIFrameElement>("Test slides").contentWindow; }
function message(data: object, source: Window | null = frame()) { act(() => { window.dispatchEvent(new MessageEvent("message", { source, data: { channel: "wc-presentation", ...data } })); }); }
beforeEach(() => { vi.mocked(loadPresentationFrame).mockReset().mockImplementation(() => new Promise(() => { /* pending host module */ })); PreviewWorker.instances = []; vi.stubGlobal("Worker", PreviewWorker); vi.useFakeTimers(); });
afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); vi.restoreAllMocks(); });
describe("presentation frame ownership", () => {
  it("ignores foreign/malformed messages and only accepts metadata from its own frame", () => {
    render(<Harness />); message({ type: "ready", metadata }, window); expect(screen.getByRole("status")).toHaveTextContent("loading");
    message({ type: "ready", metadata: { ...metadata, slides: [null] } }); expect(screen.getByRole("status")).toHaveTextContent("loading"); message({ type: "ready", metadata }); expect(screen.getByRole("status")).toHaveTextContent("ready");
    expect(isPresentationEvent({ type: "ready", metadata: { ...metadata, width: Infinity } })).toBe(false); expect(isPresentationEvent({ type: "slide", index: -1 })).toBe(false);
  });
  it("cleans up workers and deadlines on close and discards a late result from a previous file", () => {
    const scheduled = vi.spyOn(window, "setTimeout"), canceled = vi.spyOn(window, "clearTimeout");
    const { rerender, unmount } = render(<Harness />), old = PreviewWorker.instances[0]; if (!old) throw Error("Missing worker"); const late = old.onmessage;
    rerender(<Harness href="/new" />); expect(old.terminate).toHaveBeenCalled(); act(() => { late?.(new MessageEvent("message", { data: { error: "invalid" } })); }); expect(screen.getByRole("status")).toHaveTextContent("loading"); unmount(); expect(vi.mocked(loadPresentationFrame).mock.lastCall?.[0].aborted).toBe(true); expect(PreviewWorker.instances[1]?.terminate).toHaveBeenCalled(); scheduled.mock.calls.forEach((call, index) => { if (call[1] === 25000) expect(canceled).toHaveBeenCalledWith(scheduled.mock.results[index]?.value); });
  });
  it("returns legacy and deadline fallbacks without an unbounded parser", () => {
    const { unmount } = render(<Harness legacy />); expect(screen.getByRole("status")).toHaveTextContent("legacy"); expect(PreviewWorker.instances).toHaveLength(0); unmount(); render(<Harness />); act(() => { vi.advanceTimersByTime(25000); }); expect(screen.getByRole("status")).toHaveTextContent("unavailable"); expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
  });
  it("waits for its own frame handshake before transferring validated bytes", () => {
    render(<Harness />); const target = frame(), worker = PreviewWorker.instances[0]; if (!target || !worker) throw Error("Missing preview resources");
    const send = vi.spyOn(target, "postMessage"), bytes = new ArrayBuffer(8);
    act(() => { worker.onmessage?.(new MessageEvent("message", { data: { bytes, xml: {} } })); }); expect(send).not.toHaveBeenCalled();
    message({ type: "boot" }, window); expect(send).not.toHaveBeenCalled(); message({ type: "boot" });
    expect(send).toHaveBeenCalledWith({ channel: "wc-presentation", type: "open", bytes, xml: {} }, "*", [bytes]); expect(worker.terminate).toHaveBeenCalled();
  });
  it("loads the authenticated host module and ignores its late document after close", async () => {
    vi.mocked(loadPresentationFrame).mockResolvedValue("owned document");
    const { unmount } = render(<Harness />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByTestId("source")).toHaveTextContent("owned document");
    unmount();
    let finish: (value: string) => void = () => { throw Error("Missing loader"); };
    vi.mocked(loadPresentationFrame).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const { rerender } = render(<Harness />);
    const oldFinish = finish;
    rerender(<Harness href="/renewed" />);
    await act(async () => { oldFinish("stale document"); await Promise.resolve(); });
    expect(screen.getByTestId("source")).toBeEmptyDOMElement();
    await act(async () => { finish("renewed document"); await Promise.resolve(); });
    expect(screen.getByTestId("source")).toHaveTextContent("renewed document");
  });
  it("reports renderer download failure as unavailable and stops the parser", async () => {
    vi.mocked(loadPresentationFrame).mockRejectedValue(Error("Access login required"));
    render(<Harness />);
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByRole("status")).toHaveTextContent("unavailable");
    expect(vi.mocked(loadPresentationFrame).mock.lastCall?.[0].aborted).toBe(true);
    expect(PreviewWorker.instances[0]?.terminate).toHaveBeenCalled();
    act(() => { vi.advanceTimersByTime(25000); });
    expect(screen.getByRole("status")).toHaveTextContent("unavailable");
  });
  it("retains the processing limit when the runtime boots but parsing times out", () => {
    render(<Harness />); const worker = PreviewWorker.instances[0];
    act(() => { worker?.onmessage?.(new MessageEvent("message", { data: { bytes: new ArrayBuffer(8), xml: {} } })); });
    message({ type: "boot" });
    act(() => { vi.advanceTimersByTime(25000); });
    expect(screen.getByRole("status")).toHaveTextContent("limit");
  });
  it("does not load the renderer for legacy files", () => {
    render(<Harness legacy />);
    expect(loadPresentationFrame).not.toHaveBeenCalled();
  });

  it("discards a late host module and worker result after a terminal deadline", async () => {
    let finish: (value: string) => void = () => { throw Error("Missing loader"); };
    vi.mocked(loadPresentationFrame).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    render(<Harness />); const worker = PreviewWorker.instances[0];
    act(() => { vi.advanceTimersByTime(25000); });
    await act(async () => { finish("late document"); await Promise.resolve(); });
    act(() => { worker?.onmessage?.(new MessageEvent("message", { data: { error: "invalid" } })); });
    expect(screen.getByTestId("source")).toBeEmptyDOMElement();
    expect(screen.getByRole("status")).toHaveTextContent("unavailable");
  });

});
