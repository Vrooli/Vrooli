import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { apiBaseMock, mockFetchSuccess, mockFetchError } from "../test-utils";

vi.mock("@vrooli/api-base", () => apiBaseMock());

import { useImageUpload } from "../hooks/useImageUpload";

describe("useImageUpload", () => {
  const savedFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = savedFetch;
    vi.restoreAllMocks();
  });

  it("uploads file and injects path into terminal", async () => {
    mockFetchSuccess({ path: "/tmp/web-console-uploads/sess-1/test.png" });
    const sendInput = vi.fn().mockReturnValue(true);

    const { result } = renderHook(() => useImageUpload("sess-1", sendInput));

    expect(result.current.uploading).toBe(false);
    expect(result.current.error).toBeNull();

    const file = new File(["fake png"], "test.png", { type: "image/png" });
    await act(async () => {
      await result.current.uploadAndInject(file);
    });

    expect(sendInput).toHaveBeenCalledWith("/tmp/web-console-uploads/sess-1/test.png\n");
    expect(result.current.uploading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("sets error on upload failure", async () => {
    mockFetchError(400, {
      error: "Only image files are accepted",
      code: "invalid_upload_type",
      category: "validation",
      recovery: "Upload an image file",
    });
    const sendInput = vi.fn().mockReturnValue(true);

    const { result } = renderHook(() => useImageUpload("sess-1", sendInput));

    const file = new File(["not an image"], "script.sh", { type: "text/plain" });
    await act(async () => {
      await result.current.uploadAndInject(file);
    });

    expect(sendInput).not.toHaveBeenCalled();
    expect(result.current.error).toBe("Only image files are accepted");
    expect(result.current.uploading).toBe(false);
  });

  it("shows uploading state during upload", async () => {
    let resolveUpload!: (value: Response) => void;
    globalThis.fetch = vi.fn().mockReturnValue(
      new Promise<Response>((resolve) => {
        resolveUpload = resolve;
      }),
    ) as typeof fetch;
    const sendInput = vi.fn().mockReturnValue(true);

    const { result } = renderHook(() => useImageUpload("sess-1", sendInput));

    const file = new File(["data"], "img.png", { type: "image/png" });
    let uploadPromise: Promise<void>;
    act(() => {
      uploadPromise = result.current.uploadAndInject(file);
    });

    // While uploading
    expect(result.current.uploading).toBe(true);

    // Resolve
    await act(async () => {
      resolveUpload({
        ok: true,
        json: () => Promise.resolve({ path: "/tmp/test.png" }),
      } as Response);
      await uploadPromise;
    });

    expect(result.current.uploading).toBe(false);
  });
});
