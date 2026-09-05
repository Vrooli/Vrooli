import { describe, expect, it, vi, beforeEach } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";

import { useFilePreviewController } from "../useFilePreviewController";
import type { DirectoryEntry, PreviewListing, PreviewModel } from "../types";

const { mockResolve, mockGetText, mockList } = vi.hoisted(() => ({
  mockResolve: vi.fn(),
  mockGetText: vi.fn(),
  mockList: vi.fn(),
}));

vi.mock("../../../api/filePreview", () => ({
  resolveFilePreview: mockResolve,
  getFilePreviewText: mockGetText,
  listDirectory: mockList,
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
    listingAvailable: false,
    blobUrl: "/blob",
    blobHref: "/blob",
    expiresMs: Date.now() + 60_000,
    warnings: [],
    ...overrides,
  };
}

function dirModel(path: string, previewId = "pv-dir"): PreviewModel {
  return model({
    previewId,
    kind: "directory",
    inputPath: path,
    resolvedPath: path,
    basename: path.split("/").pop() ?? path,
    mimeType: "inode/directory",
    canDownload: false,
    textContentAvailable: false,
    listingAvailable: true,
  });
}

function listing(overrides: Partial<PreviewListing> = {}): PreviewListing {
  return {
    resolvedPath: "/tmp/dir",
    parentPath: "/tmp",
    entries: [],
    totalEntries: 0,
    truncated: false,
    nextPageToken: "",
    effectiveSort: "dirs_first_name",
    sort: "dirs_first_name",
    showHidden: false,
    warnings: [],
    ...overrides,
  };
}

function dirEntry(name: string): DirectoryEntry {
  return {
    name,
    entryType: "file",
    kind: null,
    sizeBytes: 0,
    mtimeMs: 0,
    canPreview: true,
    symlinkTarget: "",
    symlinkBroken: false,
    mode: "-rw-r--r--",
    childCount: null,
  };
}

beforeEach(() => {
  mockResolve.mockReset();
  mockGetText.mockReset();
  mockList.mockReset();
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

describe("useFilePreviewController — directory navigation", () => {
  it("fetches a listing for a directory instead of text or bytes", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    expect(result.current.state.status).toBe("ready");
    expect(result.current.state.listing?.entries).toHaveLength(1);
    expect(mockGetText).not.toHaveBeenCalled();
    expect(mockList).toHaveBeenCalledWith("s", "pv-dir", { sort: "dirs_first_name", showHidden: false });
  });

  it("pushes a frame when navigating in, and restores it on back", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });
    expect(result.current.state.stack).toHaveLength(0);

    mockResolve.mockResolvedValueOnce(model({ resolvedPath: "/tmp/dir/a.md", previewId: "pv-file" }));
    mockGetText.mockResolvedValueOnce({
      resolvedPath: "/tmp/dir/a.md",
      kind: "code",
      mimeType: "text/plain",
      content: "hi",
      truncated: false,
    });
    await act(async () => {
      result.current.navigateTo("/tmp/dir/a.md");
    });
    await waitFor(() => expect(result.current.state.status).toBe("ready"));
    expect(result.current.state.stack).toHaveLength(1);
    expect(result.current.state.model?.resolvedPath).toBe("/tmp/dir/a.md");

    // Back restores the directory without re-resolving it.
    const resolveCalls = mockResolve.mock.calls.length;
    act(() => result.current.navigateBack());
    expect(result.current.state.stack).toHaveLength(0);
    expect(result.current.state.model?.resolvedPath).toBe("/tmp/dir");
    expect(result.current.state.listing?.entries).toHaveLength(1);
    expect(mockResolve.mock.calls).toHaveLength(resolveCalls);
  });

  it("re-resolves on back when the cached preview id has expired", async () => {
    mockResolve.mockResolvedValueOnce(
      { ...dirModel("/tmp/dir"), expiresMs: Date.now() - 1 },
    );
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(model({ resolvedPath: "/tmp/dir/a.md", previewId: "pv-file" }));
    mockGetText.mockResolvedValueOnce({
      resolvedPath: "/tmp/dir/a.md",
      kind: "code",
      mimeType: "text/plain",
      content: "hi",
      truncated: false,
    });
    await act(async () => {
      result.current.navigateTo("/tmp/dir/a.md");
    });
    await waitFor(() => expect(result.current.state.stack).toHaveLength(1));

    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir", "pv-dir-2"));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    await act(async () => {
      result.current.navigateBack();
    });
    await waitFor(() => expect(result.current.state.model?.previewId).toBe("pv-dir-2"));
    expect(result.current.state.stack).toHaveLength(0);
  });

  it("opening from a message clears the navigation history", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(listing());
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir/sub", "pv-sub"));
    mockList.mockResolvedValueOnce(listing({ resolvedPath: "/tmp/dir/sub" }));
    await act(async () => {
      result.current.navigateTo("/tmp/dir/sub");
    });
    await waitFor(() => expect(result.current.state.stack).toHaveLength(1));

    mockResolve.mockResolvedValueOnce(model({ resolvedPath: "/elsewhere.md" }));
    mockGetText.mockResolvedValueOnce({
      resolvedPath: "/elsewhere.md",
      kind: "code",
      mimeType: "text/plain",
      content: "x",
      truncated: false,
    });
    await act(async () => {
      await result.current.openPreview("/elsewhere.md", "message_link");
    });
    expect(result.current.state.stack).toHaveLength(0);
  });

  it("appends the next page to the entries already loaded", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 2, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("b")], totalEntries: 2 }));
    await act(async () => {
      result.current.loadMore();
    });
    await waitFor(() => expect(result.current.state.listing?.entries).toHaveLength(2));
    expect(result.current.state.listing?.entries.map((e) => e.name)).toEqual(["a", "b"]);
    expect(result.current.state.listing?.nextPageToken).toBe("");
    expect(mockList).toHaveBeenLastCalledWith("s", "pv-dir", {
      sort: "dirs_first_name",
      showHidden: false,
      pageToken: "tok",
    });
  });

  it("discards a page that lands after the user navigated away", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 2, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    // The page-2 request never settles until after the navigation lands.
    let releasePage: (value: PreviewListing) => void = () => {};
    mockList.mockImplementationOnce(
      () => new Promise<PreviewListing>((resolve) => { releasePage = resolve; }),
    );
    act(() => result.current.loadMore());

    mockResolve.mockResolvedValueOnce(dirModel("/tmp/other", "pv-other"));
    mockList.mockResolvedValueOnce(listing({ resolvedPath: "/tmp/other", entries: [dirEntry("z")], totalEntries: 1 }));
    await act(async () => {
      result.current.navigateTo("/tmp/other");
    });
    await waitFor(() => expect(result.current.state.model?.previewId).toBe("pv-other"));

    await act(async () => {
      releasePage(listing({ entries: [dirEntry("b")], totalEntries: 2 }));
    });

    // The stale page must not be appended to the directory now on screen.
    expect(result.current.state.listing?.entries.map((e) => e.name)).toEqual(["z"]);
  });

  // Changing the sort while a page fetch is in flight discards that fetch via
  // the request-id guard — which means the discarded handler never clears
  // loadingMore itself. If the flag stuck, "Load more" would stay disabled for
  // the rest of the session.
  it("clears loadingMore when a sort change discards an in-flight page", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 9, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    let releasePage: (value: PreviewListing) => void = () => {};
    mockList.mockImplementationOnce(
      () => new Promise<PreviewListing>((resolve) => { releasePage = resolve; }),
    );
    act(() => result.current.loadMore());
    expect(result.current.state.loadingMore).toBe(true);

    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("b")], totalEntries: 9, sort: "name", effectiveSort: "name" }),
    );
    await act(async () => {
      result.current.setListOptions({ sort: "name" });
    });
    await waitFor(() => expect(result.current.state.status).toBe("ready"));

    await act(async () => {
      releasePage(listing({ entries: [dirEntry("stale")], totalEntries: 9 }));
    });

    expect(result.current.state.loadingMore).toBe(false);
    expect(result.current.state.listing?.entries.map((e) => e.name)).toEqual(["b"]);
  });

  it("re-lists from page one when the sort changes", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("b")], totalEntries: 1, sort: "size_desc", effectiveSort: "size_desc" }),
    );
    await act(async () => {
      result.current.setListOptions({ sort: "size_desc" });
    });
    await waitFor(() => expect(result.current.state.status).toBe("ready"));
    expect(mockList).toHaveBeenLastCalledWith("s", "pv-dir", { sort: "size_desc", showHidden: false });
    expect(result.current.state.listing?.entries.map((e) => e.name)).toEqual(["b"]);
  });

  it("reopen preserves the chosen ordering and the history beneath it", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockResolvedValueOnce(listing({ sort: "mtime_desc", effectiveSort: "mtime_desc", showHidden: true }));
    const { result } = renderHook(() => useFilePreviewController("s"));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir", "pv-dir-2"));
    mockList.mockResolvedValueOnce(listing({ sort: "mtime_desc", effectiveSort: "mtime_desc", showHidden: true }));
    await act(async () => {
      result.current.reopen();
    });
    await waitFor(() => expect(result.current.state.model?.previewId).toBe("pv-dir-2"));
    expect(mockList).toHaveBeenLastCalledWith("s", "pv-dir-2", { sort: "mtime_desc", showHidden: true });
  });

  it("surfaces a listing error without leaving the viewer stuck loading", async () => {
    mockResolve.mockResolvedValueOnce(dirModel("/tmp/dir"));
    mockList.mockRejectedValueOnce(new Error("Directory changed while paging; reload the listing"));
    const { result } = renderHook(() => useFilePreviewController("s"));

    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    expect(result.current.state.status).toBe("error");
    expect(result.current.state.error).toMatch(/reload the listing/);
  });
});
