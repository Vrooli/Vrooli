import { describe, expect, it, vi, beforeEach } from "vitest";
import { act, waitFor } from "@testing-library/react";

import { renderHookWithProviders as renderHook } from "../../test-utils/index.js";
import { NO_CAPABILITIES, type DirectoryEntry, type PreviewListing, type PreviewModel, type PreviewSource, type ResolveOutcome } from "../../source/types.js";
import { strings } from "../../i18n/strings.js";
import { useFilePreviewController } from "../useFilePreviewController.js";

const { mockResolve, mockGetText, mockList } = vi.hoisted(() => ({
  mockResolve: vi.fn(),
  mockGetText: vi.fn(),
  mockList: vi.fn(),
}));

// A source whose operations are the spies above, so each test scripts the
// outcome of one call. `locate` is on, as it is for Web Console's sessions.
function source(scope: string): PreviewSource {
  return {
    scope,
    capabilities: { ...NO_CAPABILITIES, locate: true },
    resolve: (path, options) => mockResolve(path, options?.context) as Promise<ResolveOutcome>,
    readText: (model) => mockGetText(model) as ReturnType<PreviewSource["readText"]>,
    listDirectory: (model, options) => mockList(model, options) as Promise<PreviewListing>,
  };
}

const resolved = (model: PreviewModel): ResolveOutcome => ({ outcome: "resolved", model });

function model(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "pv-1",
    inputPath: "/tmp/a",
    resolvedPath: "/tmp/a",
    basename: "a",
    kind: "code",
    mimeType: "text/plain",
    sizeBytes: 10,
    canPreview: true,
    canDownload: true,
    textContentAvailable: true,
    listingAvailable: false,
    blobHref: "/blob",
    expiresMs: Date.now() + 60_000,
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
  it("opens both report links on the same file, preserving literal path hashes and line links", async () => {
    mockResolve.mockResolvedValue(resolved(model()));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    for (const anchor of ["review-2", "project-O5", "project-O5"]) {
      await act(async () => { await result.current.openPreview(`/tmp/report%23draft.html#${anchor}`, "message_link"); });
      expect(mockResolve).toHaveBeenLastCalledWith("/tmp/report%23draft.html", "message_link");
      expect(result.current.state).toMatchObject({ status: "ready", anchor });
    }
    for (const literal of ["/tmp/report.html%23draft", "/tmp/report.html:3", "/tmp/note#draft.txt"]) {
      await act(async () => { await result.current.openPreview(literal, "message_link"); });
      expect(mockResolve).toHaveBeenLastCalledWith(literal, "message_link");
      expect(result.current.state.anchor).toBeUndefined();
    }
  });

  it("keeps HTML in history when a clicked file hits closed lifecycle admission and can retry", async () => {
    const report = model({ previewId: "report", resolvedPath: "/research/REVIEW.html", basename: "REVIEW.html" });
    const content = { resolvedPath: report.resolvedPath, kind: "code", mimeType: "text/html", content: '<a href="script.md">Script</a>', truncated: false };
    mockResolve.mockResolvedValueOnce(resolved(report));
    mockGetText.mockResolvedValueOnce(content);
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => { await result.current.openPreview(`${report.resolvedPath}#sources`, "message_link"); });
    mockResolve.mockRejectedValueOnce(new Error("service unavailable: lifecycle admission closed"));
    act(() => { result.current.navigateTo("/research/script.md"); });
    await waitFor(() => { expect(result.current.state.status).toBe("error"); });
    expect(result.current.state.error).toContain("lifecycle admission closed");
    expect(result.current.state.stack[0]?.text?.content).toBe(content.content);
    mockResolve.mockResolvedValueOnce(resolved(model({ resolvedPath: "/research/script.md", kind: "markdown" })));
    mockGetText.mockResolvedValueOnce({ ...content, kind: "markdown", content: "# Script" });
    act(() => { result.current.reopen(); });
    await waitFor(() => { expect(result.current.state.status).toBe("ready"); });
    expect(result.current.state.model?.kind).toBe("markdown");
    act(() => { result.current.navigateBack(); });
    expect(result.current.state.model?.previewId).toBe("report");
    expect(result.current.state.anchor).toBe("sources");
    expect(result.current.state.text?.content).toBe(content.content);
  });
  it("walks resolving → loadingText → ready for text kinds", async () => {
    mockResolve.mockResolvedValueOnce(resolved(model()));
    mockGetText.mockResolvedValueOnce({
      resolvedPath: "/tmp/a",
      kind: "code",
      mimeType: "text/plain",
      content: "hi",
      truncated: false,
    });
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/a", "message_link");
    });

    expect(result.current.state.status).toBe("ready");
    expect(result.current.state.text?.content).toBe("hi");
    expect(mockGetText).toHaveBeenCalledWith(expect.objectContaining({ previewId: "pv-1" }));
  });

  it("goes straight to ready for blob kinds without fetching text", async () => {
    mockResolve.mockResolvedValueOnce(resolved(model({ kind: "image", textContentAvailable: false })));
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/a.png");
    });

    expect(result.current.state.status).toBe("ready");
    expect(mockGetText).not.toHaveBeenCalled();
  });

  it("marks non-previewable, non-text files unsupported", async () => {
    mockResolve.mockResolvedValueOnce(resolved(model({ kind: "unsupported", canPreview: false, textContentAvailable: false })));
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/a.bin");
    });

    expect(result.current.state.status).toBe("unsupported");
  });

  it("surfaces resolve errors", async () => {
    mockResolve.mockRejectedValueOnce(new Error("nope"));
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/missing");
    });

    expect(result.current.state.status).toBe("error");
    expect(result.current.state.error).toBe("nope");
  });

  it("close() resets to idle", async () => {
    mockResolve.mockResolvedValueOnce(resolved(model({ kind: "image", textContentAvailable: false })));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/a.png");
    });
    act(() => result.current.close());
    expect(result.current.state.status).toBe("idle");
    expect(result.current.state.open).toBe(false);
  });

  it("reportError transitions a ready preview to error", async () => {
    mockResolve.mockResolvedValueOnce(resolved(model({ kind: "image", textContentAvailable: false })));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
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
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    expect(result.current.state.status).toBe("ready");
    expect(result.current.state.listing?.entries).toHaveLength(1);
    expect(mockGetText).not.toHaveBeenCalled();
    expect(mockList).toHaveBeenCalledWith(expect.objectContaining({ previewId: "pv-dir" }), expect.objectContaining({ sort: "dirs_first_name", showHidden: false }));
  });

  it("pushes a frame when navigating in, and restores it on back", async () => {
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });
    expect(result.current.state.stack).toHaveLength(0);

    mockResolve.mockResolvedValueOnce(resolved(model({ resolvedPath: "/tmp/dir/a.md", previewId: "pv-file" })));
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
    mockResolve.mockResolvedValueOnce(resolved(
      { ...dirModel("/tmp/dir"), expiresMs: Date.now() - 1 },
    ));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(resolved(model({ resolvedPath: "/tmp/dir/a.md", previewId: "pv-file" })));
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

    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir", "pv-dir-2")));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a.md")], totalEntries: 1 }));
    await act(async () => {
      result.current.navigateBack();
    });
    await waitFor(() => expect(result.current.state.model?.previewId).toBe("pv-dir-2"));
    expect(result.current.state.stack).toHaveLength(0);
  });

  it("opening from a message clears the navigation history", async () => {
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(listing());
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir/sub", "pv-sub")));
    mockList.mockResolvedValueOnce(listing({ resolvedPath: "/tmp/dir/sub" }));
    await act(async () => {
      result.current.navigateTo("/tmp/dir/sub");
    });
    await waitFor(() => expect(result.current.state.stack).toHaveLength(1));

    mockResolve.mockResolvedValueOnce(resolved(model({ resolvedPath: "/elsewhere.md" })));
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
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 2, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController(source("s")));
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
    expect(mockList).toHaveBeenLastCalledWith(
      expect.objectContaining({ previewId: "pv-dir" }),
      expect.objectContaining({ sort: "dirs_first_name", showHidden: false, pageToken: "tok" }),
    );
  });

  it("discards a page that lands after the user navigated away", async () => {
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 2, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    // The page-2 request never settles until after the navigation lands.
    let releasePage: (value: PreviewListing) => void = () => {};
    mockList.mockImplementationOnce(
      () => new Promise<PreviewListing>((resolve) => { releasePage = resolve; }),
    );
    act(() => result.current.loadMore());

    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/other", "pv-other")));
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
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(
      listing({ entries: [dirEntry("a")], totalEntries: 9, nextPageToken: "tok" }),
    );
    const { result } = renderHook(() => useFilePreviewController(source("s")));
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
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(listing({ entries: [dirEntry("a")], totalEntries: 1 }));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
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
    expect(mockList).toHaveBeenLastCalledWith(expect.objectContaining({ previewId: "pv-dir" }), expect.objectContaining({ sort: "size_desc", showHidden: false }));
    expect(result.current.state.listing?.entries.map((e) => e.name)).toEqual(["b"]);
  });

  it("reopen preserves the chosen ordering and the history beneath it", async () => {
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockResolvedValueOnce(listing({ sort: "mtime_desc", effectiveSort: "mtime_desc", showHidden: true }));
    const { result } = renderHook(() => useFilePreviewController(source("s")));
    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir", "pv-dir-2")));
    mockList.mockResolvedValueOnce(listing({ sort: "mtime_desc", effectiveSort: "mtime_desc", showHidden: true }));
    await act(async () => {
      result.current.reopen();
    });
    await waitFor(() => expect(result.current.state.model?.previewId).toBe("pv-dir-2"));
    expect(mockList).toHaveBeenLastCalledWith(expect.objectContaining({ previewId: "pv-dir-2" }), expect.objectContaining({ sort: "mtime_desc", showHidden: true }));
  });

  it("surfaces a listing error without leaving the viewer stuck loading", async () => {
    mockResolve.mockResolvedValueOnce(resolved(dirModel("/tmp/dir")));
    mockList.mockRejectedValueOnce(new Error("Directory changed while paging; reload the listing"));
    const { result } = renderHook(() => useFilePreviewController(source("s")));

    await act(async () => {
      await result.current.openPreview("/tmp/dir");
    });

    expect(result.current.state.status).toBe("error");
    expect(result.current.state.error).toMatch(/reload the listing/);
  });
});

describe("useFilePreviewController relative-path resolution", () => {
  beforeEach(() => {
    mockResolve.mockReset();
    mockList.mockReset();
  });

  it("enters the picker when a relative path matches more than one base", async () => {
    mockResolve.mockResolvedValue({
      outcome: "ambiguous",
      inputPath: "shared.txt",
      candidates: [
        { resolvedPath: "/cwd/shared.txt", resolutionBasis: "session_cwd", kind: "text", canPreview: true, basename: "shared.txt" },
        { resolvedPath: "/root/shared.txt", resolutionBasis: "project_root", kind: "text", canPreview: true, basename: "shared.txt" },
      ],
    } satisfies ResolveOutcome);

    const { result } = renderHook(() => useFilePreviewController(source("s1")));
    await act(async () => { await result.current.openPreview("shared.txt"); });

    expect(result.current.state.status).toBe("choosing");
    expect(result.current.state.candidates).toHaveLength(2);
  });

  it("opens the first candidate when the source cannot ask the user to choose", async () => {
    mockResolve.mockImplementation((path: string) => Promise.resolve(path === "shared.txt"
      ? { outcome: "ambiguous", inputPath: path, candidates: [{ resolvedPath: "/cwd/shared.txt", resolutionBasis: "session_cwd", kind: "text", canPreview: true, basename: "shared.txt" }, { resolvedPath: "/root/shared.txt", resolutionBasis: "project_root", kind: "text", canPreview: true, basename: "shared.txt" }] } satisfies ResolveOutcome
      : resolved(model({ resolvedPath: path }))));
    mockGetText.mockResolvedValue({ resolvedPath: "/cwd/shared.txt", kind: "text", mimeType: "text/plain", content: "x", truncated: false });

    const { result } = renderHook(() => useFilePreviewController({ ...source("s1"), capabilities: NO_CAPABILITIES }));
    await act(async () => { await result.current.openPreview("shared.txt"); });

    expect(result.current.state.status).toBe("ready");
    expect(result.current.state.model?.resolvedPath).toBe("/cwd/shared.txt");
  });

  it("reports an unresolved path as unavailable when the source cannot locate", async () => {
    mockResolve.mockResolvedValue({
      outcome: "unresolved", inputPath: "missing.md",
      searchBases: [{ resolvedPath: "/proj", resolutionBasis: "project_root", kind: "directory", canPreview: true, basename: "proj" }],
    } satisfies ResolveOutcome);

    const { result } = renderHook(() => useFilePreviewController({ ...source("s1"), capabilities: NO_CAPABILITIES }));
    await act(async () => { await result.current.openPreview("missing.md"); });

    expect(result.current.state.status).toBe("error");
    expect(result.current.state.error).toBe(strings.viewer.openFailed);
  });

  it("opens a folder browser and locates the match for an unresolved relative path", async () => {
    mockResolve.mockImplementation((path: string) => {
      if (path === "src/App.tsx") {
        return Promise.resolve({
          outcome: "unresolved", inputPath: "src/App.tsx",
          searchBases: [{ resolvedPath: "/proj", resolutionBasis: "project_root", kind: "directory", canPreview: true, basename: "proj" }],
        } satisfies ResolveOutcome);
      }
      if (path === "/proj") return Promise.resolve(resolved(dirModel("/proj")));
      if (path === "/proj/src/App.tsx") return Promise.resolve(resolved(model({ inputPath: "/proj/src/App.tsx", resolvedPath: "/proj/src/App.tsx" })));
      return Promise.reject(new Error("not found"));
    });
    mockList.mockResolvedValue(listing());

    const { result } = renderHook(() => useFilePreviewController(source("s1")));
    await act(async () => { await result.current.openPreview("src/App.tsx"); });

    await waitFor(() => { expect(result.current.state.status).toBe("ready"); });
    expect(result.current.state.model?.kind).toBe("directory");
    expect(result.current.state.search?.relPath).toBe("src/App.tsx");
    await waitFor(() => { expect(result.current.state.search?.foundPath).toBe("/proj/src/App.tsx"); });
  });
});
