import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { readPreviewBytes } from "../readPreviewBytes";
import { inspectFontEnvelope, prepareFont } from "../prepareFont";
vi.mock("../readPreviewBytes");
vi.mock("../prepareFont");
const postMessage = vi.fn(), load = vi.fn();
const nativeFace = vi.fn(function () { return { load }; });
let receive: (event: MessageEvent<{ href: string }>) => Promise<void>;
beforeEach(async () => {
  vi.resetModules(); vi.clearAllMocks();
  vi.stubGlobal("self", { postMessage }); vi.stubGlobal("FontFace", nativeFace);
  vi.mocked(readPreviewBytes).mockResolvedValue(new ArrayBuffer(10)); vi.mocked(inspectFontEnvelope).mockReturnValue("woff2");
  load.mockResolvedValue({});
  await import("../font.worker");
  receive = (self as unknown as { onmessage: typeof receive }).onmessage;
});
afterEach(() => { vi.unstubAllGlobals(); });
describe("font worker admission", () => {
  it("validates the envelope and native font before metadata decompression", async () => {
    await receive(new MessageEvent("message", { data: { href: "/blob" } }));
    expect(vi.mocked(inspectFontEnvelope).mock.invocationCallOrder[0]).toBeLessThan(nativeFace.mock.invocationCallOrder[0] ?? 0);
    expect(load.mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(prepareFont).mock.invocationCallOrder[0] ?? 0);
    expect(postMessage).toHaveBeenLastCalledWith(expect.objectContaining({ type: "ready" }), expect.anything());
  });
  it("never admits native-rejected compressed fonts to the library", async () => {
    load.mockRejectedValue(new Error("bad compressed data")); await receive(new MessageEvent("message", { data: { href: "/blob" } }));
    expect(prepareFont).not.toHaveBeenCalled(); expect(postMessage).toHaveBeenLastCalledWith({ type: "error", error: "invalid" });
  });
  it("degrades gracefully when worker font validation is unavailable", async () => {
    vi.stubGlobal("FontFace", undefined); await receive(new MessageEvent("message", { data: { href: "/blob" } })); expect(prepareFont).not.toHaveBeenCalled(); expect(postMessage).toHaveBeenLastCalledWith({ type: "error", error: "unavailable" });
  });
  it("rejects expansion limits before native decoding", async () => {
    vi.mocked(inspectFontEnvelope).mockImplementationOnce(() => { throw new RangeError("expansion"); }); await receive(new MessageEvent("message", { data: { href: "/blob" } })); expect(nativeFace).not.toHaveBeenCalled(); expect(prepareFont).not.toHaveBeenCalled(); expect(postMessage).toHaveBeenLastCalledWith({ type: "error", error: "limit" });
  });
});
