import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { readPreviewBytes } from "../readPreviewBytes";
import { MAX_HTML_PREVIEW_BYTES, useHtmlPreview } from "../useHtmlPreview";
import type { PreviewModel, PreviewTextContent } from "../types";

vi.mock("../readPreviewBytes", () => ({ readPreviewBytes: vi.fn() }));
const read = vi.mocked(readPreviewBytes);
const model = (overrides: Partial<PreviewModel> = {}) => ({
  previewId: "large-html", blobHref: "/blob/large-html", sizeBytes: 3_759_455,
  resolvedPath: "/report.html", kind: "code", mimeType: "text/html", ...overrides,
}) as PreviewModel;
const bytes = (s: string) => new TextEncoder().encode(s).buffer;

afterEach(() => { vi.resetAllMocks(); });

describe("large HTML loading", () => {
  it("loads complete HTML when inline text is unavailable", async () => {
    const content = "<h1>Research review</h1><script>window.ready = true</script>";
    read.mockResolvedValueOnce(bytes(content));
    const { result } = renderHook(() => useHtmlPreview(model(), null));
    expect(result.current.content).toBeNull();
    await waitFor(() => { expect(result.current.content).toBe(content); });
    expect(read).toHaveBeenCalledWith("/blob/large-html", MAX_HTML_PREVIEW_BYTES, expect.any(AbortSignal));
  });

  it("preserves the supplied text without a second request", () => {
    const text = { content: "<h1>Small report</h1>" } as PreviewTextContent;
    const { result } = renderHook(() => useHtmlPreview(model(), text));
    expect(result.current.content).toBe(text.content);
    expect(read).not.toHaveBeenCalled();
  });

  it("does not fetch files above the bounded HTML budget", () => {
    const { result } = renderHook(() => useHtmlPreview(model({ sizeBytes: MAX_HTML_PREVIEW_BYTES + 1 }), null));
    expect(result.current.error).toBe("limit");
    expect(read).not.toHaveBeenCalled();
  });

  it.each([new RangeError("bytes"), new Error("fetch"), new TypeError("invalid UTF-8")])("reports load failures without reporting an empty file: %s", async (error) => {
    read.mockRejectedValueOnce(error);
    const { result } = renderHook(() => useHtmlPreview(model(), null));
    await waitFor(() => { expect(result.current.error).toBe(error instanceof RangeError ? "limit" : "fetch"); });
    expect(result.current.content).toBeNull();
  });

  it("rejects invalid UTF-8 and NUL bytes instead of rendering a corrupt document", async () => {
    read.mockResolvedValueOnce(new Uint8Array([0xff]).buffer);
    const first = renderHook(() => useHtmlPreview(model(), null));
    await waitFor(() => { expect(first.result.current.error).toBe("fetch"); });
    first.unmount();
    read.mockResolvedValueOnce(bytes("<h1>\0</h1>"));
    const second = renderHook(() => useHtmlPreview(model(), null));
    await waitFor(() => { expect(second.result.current.error).toBe("fetch"); });
  });

  it("aborts an obsolete request and does not flash its document on navigation", async () => {
    let finish: (b: ArrayBuffer) => void = () => {};
    read.mockReturnValueOnce(new Promise<ArrayBuffer>((resolve) => { finish = resolve; }));
    const { result, rerender } = renderHook(({ href }) => useHtmlPreview(model({ blobHref: href }), null), { initialProps: { href: "/old" } });
    const signal = read.mock.calls[0]?.[2];
    read.mockResolvedValueOnce(bytes("<h1>New</h1>"));
    rerender({ href: "/new" });
    expect(signal?.aborted).toBe(true);
    await waitFor(() => { expect(result.current.content).toBe("<h1>New</h1>"); });
    await act(async () => { finish(bytes("<h1>Old</h1>")); await Promise.resolve(); });
    expect(result.current.content).toBe("<h1>New</h1>");
  });

  it("distinguishes a genuinely empty byte response from missing content", async () => {
    read.mockResolvedValueOnce(bytes(""));
    const { result } = renderHook(() => useHtmlPreview(model({ sizeBytes: 0 }), null));
    await waitFor(() => { expect(result.current.content).toBe(""); });
    expect(result.current.error).toBeNull();
  });
});
