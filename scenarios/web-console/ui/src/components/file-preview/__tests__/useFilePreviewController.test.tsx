import { describe, expect, it, vi, beforeEach } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

import { useFilePreviewController } from "../useFilePreviewController";
import type { PreviewModel } from "../types";

const { mockResolve, mockGetText } = vi.hoisted(() => ({
  mockResolve: vi.fn(),
  mockGetText: vi.fn(),
}));

vi.mock("../../../api/filePreview", () => ({
  resolveFilePreview: mockResolve,
  getFilePreviewText: mockGetText,
}));

function model(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "pv-1",
    inputPath: "/tmp/a",
    resolvedPath: "/tmp/a",
    basename: "a",
    resolutionBasis: "absolute",
    kind: "code",
    mimeType: "text/plain",
    sizeBytes: 10,
    canPreview: true,
    canDownload: true,
    supportsRange: false,
    textContentAvailable: true,
    blobUrl: "/blob",
    blobHref: "/blob",
    warnings: [],
    ...overrides,
  };
}

beforeEach(() => {
  mockResolve.mockReset();
  mockGetText.mockReset();
});

describe("useFilePreviewController", () => {
  it("walks resolving → loadingText → ready for text kinds", async () => {
    mockResolve.mockResolvedValueOnce(model());
    mockGetText.mockResolvedValueOnce({
      resolvedPath: "/tmp/a",
      kind: "code",
      mimeType: "text/plain",
      content: "hi",
      truncated: false,
    });
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/a", "message_link");
    });

    expect(result.current.state.status).toBe("ready");
    expect(result.current.state.text?.content).toBe("hi");
    expect(mockGetText).toHaveBeenCalledWith("s", "pv-1");
  });

  it("goes straight to ready for blob kinds without fetching text", async () => {
    mockResolve.mockResolvedValueOnce(model({ kind: "image", textContentAvailable: false }));
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/a.png");
    });

    expect(result.current.state.status).toBe("ready");
    expect(mockGetText).not.toHaveBeenCalled();
  });

  it("marks non-previewable, non-text files unsupported", async () => {
    mockResolve.mockResolvedValueOnce(model({ kind: "unsupported", canPreview: false, textContentAvailable: false }));
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/a.bin");
    });

    expect(result.current.state.status).toBe("unsupported");
  });

  it("surfaces resolve errors", async () => {
    mockResolve.mockRejectedValueOnce(new Error("nope"));
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/missing");
    });

    expect(result.current.state.status).toBe("error");
    expect(result.current.state.error).toBe("nope");
  });

  it("close() resets to idle", async () => {
    mockResolve.mockResolvedValueOnce(model({ kind: "image", textContentAvailable: false }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/a.png");
    });
    act(() => result.current.close());
    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.open).toBe(false);
  });

  it("reportError transitions a ready preview to error", async () => {
    mockResolve.mockResolvedValueOnce(model({ kind: "image", textContentAvailable: false }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/a.png");
    });
    act(() => result.current.reportError("blob failed"));
    await waitFor(() => expect(result.current.state.status).toBe("error"));
    expect(result.current.state.error).toBe("blob failed");
  });
});
