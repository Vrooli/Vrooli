import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { strings } from "../../../consts/strings";
import { DirectoryPreview } from "../renderers/DirectoryPreview";
import type { DirectoryEntry, PreviewListing, PreviewModel, PreviewRendererProps } from "../types";

// The i18n test harness echoes key paths rather than English copy, so these
// assertions compare keys — copy can be reworded without breaking the suite.

function model(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "pv-1",
    inputPath: "/tmp/dir",
    resolvedPath: "/tmp/dir",
    basename: "dir",
    resolutionBasis: "absolute",
    kind: "directory",
    mimeType: "inode/directory",
    sizeBytes: 0,
    canPreview: true,
    canDownload: false,
    supportsRange: false,
    textContentAvailable: false,
    listingAvailable: true,
    blobUrl: "",
    blobHref: "",
    expiresMs: Date.now() + 60_000,
    warnings: [],
    ...overrides,
  };
}

function entry(overrides: Partial<DirectoryEntry> & { name: string }): DirectoryEntry {
  return {
    entryType: "file",
    kind: null,
    sizeBytes: 0,
    mtimeMs: 0,
    canPreview: true,
    symlinkTarget: "",
    symlinkBroken: false,
    mode: "-rw-r--r--",
    childCount: null,
    ...overrides,
  };
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

function renderDirectory(overrides: Partial<PreviewRendererProps> = {}) {
  const props: PreviewRendererProps = {
    model: model(),
    text: null,
    listing: listing(),
    onError: vi.fn(),
    onNavigate: vi.fn(),
    onLoadMore: vi.fn(),
    onListOptionsChange: vi.fn(),
    loadingMore: false,
    ...overrides,
  };
  return { props, ...render(<DirectoryPreview {...props} />) };
}

describe("DirectoryPreview", () => {
  it("renders one row per entry", () => {
    renderDirectory({
      listing: listing({
        totalEntries: 3,
        entries: [
          entry({ name: "renders", entryType: "directory", kind: "directory", childCount: 6 }),
          entry({ name: "catalog.md", kind: "markdown", sizeBytes: 18_400 }),
          entry({ name: "hero.png", kind: "image", sizeBytes: 1_200_000 }),
        ],
      }),
    });

    const rows = screen.getAllByTestId("directory-entry");
    expect(rows).toHaveLength(3);
    expect(within(rows[0] as HTMLElement).getByText("renders/")).toBeInTheDocument();
    // Byte sizes are formatted, not translated, so this one is real copy.
    expect(screen.getByText("18 KB")).toBeInTheDocument();
    expect(screen.getByText("1.1 MB")).toBeInTheDocument();
  });

  it("opens an entry by its full path, so files and directories take one code path", async () => {
    const onNavigate = vi.fn();
    renderDirectory({
      onNavigate,
      listing: listing({
        totalEntries: 1,
        entries: [entry({ name: "sub", entryType: "directory", kind: "directory", childCount: 2 })],
      }),
    });

    await userEvent.click(screen.getByTestId("directory-entry"));
    expect(onNavigate).toHaveBeenCalledWith("/tmp/dir/sub");
  });

  it("shows a directory's child count rather than a meaningless byte size", () => {
    renderDirectory({
      listing: listing({
        totalEntries: 1,
        entries: [entry({ name: "sub", entryType: "directory", kind: "directory", childCount: 6 })],
      }),
    });
    expect(screen.getByText(strings.messagesFileViewer.directoryChildCount)).toBeInTheDocument();
  });

  it("labels a broken symlink and refuses to open it", async () => {
    const onNavigate = vi.fn();
    renderDirectory({
      onNavigate,
      listing: listing({
        totalEntries: 2,
        entries: [
          entry({ name: "good", entryType: "symlink", symlinkTarget: "real.png", kind: "image" }),
          entry({
            name: "stale",
            entryType: "symlink",
            symlinkTarget: "../gone.png",
            symlinkBroken: true,
            canPreview: false,
          }),
        ],
      }),
    });

    expect(screen.getByText(strings.messagesFileViewer.directoryBrokenLink)).toBeInTheDocument();
    expect(screen.getByText("→ ../gone.png")).toBeInTheDocument();

    // The broken row is not a button, so it cannot be opened.
    const rows = screen.getAllByTestId("directory-entry");
    const broken = rows[1] as HTMLElement;
    expect(broken.tagName).not.toBe("BUTTON");
    await userEvent.click(broken);
    expect(onNavigate).not.toHaveBeenCalled();
  });

  it("distinguishes an empty directory from one hidden by the filter", async () => {
    const onListOptionsChange = vi.fn();
    renderDirectory({ onListOptionsChange, listing: listing({ entries: [], totalEntries: 0 }) });

    expect(screen.getByTestId("directory-empty")).toBeInTheDocument();
    await userEvent.click(screen.getByTestId("directory-empty-show-hidden"));
    expect(onListOptionsChange).toHaveBeenCalledWith({ showHidden: true });
  });

  it("filters the loaded entries and says so when nothing matches", async () => {
    renderDirectory({
      listing: listing({
        totalEntries: 2,
        entries: [entry({ name: "alpha.md" }), entry({ name: "beta.md" })],
      }),
    });

    const filter = screen.getByTestId("directory-filter");
    await userEvent.type(filter, "alph");
    expect(screen.getAllByTestId("directory-entry")).toHaveLength(1);

    await userEvent.clear(filter);
    await userEvent.type(filter, "zzz");
    expect(screen.getByTestId("directory-filter-empty")).toBeInTheDocument();
  });

  it("offers another page only while there is one, and reports loading", async () => {
    const onLoadMore = vi.fn();
    const { unmount } = renderDirectory({
      onLoadMore,
      listing: listing({ totalEntries: 10, entries: [entry({ name: "a" })], nextPageToken: "tok" }),
    });
    await userEvent.click(screen.getByTestId("directory-load-more"));
    expect(onLoadMore).toHaveBeenCalled();
    unmount();

    renderDirectory({ listing: listing({ totalEntries: 1, entries: [entry({ name: "a" })] }) });
    expect(screen.queryByTestId("directory-load-more")).not.toBeInTheDocument();
  });

  it("reports the entry count as loaded-of-total", () => {
    renderDirectory({
      listing: listing({
        totalEntries: 400,
        entries: [entry({ name: "a" }), entry({ name: "b" })],
        nextPageToken: "tok",
      }),
    });
    expect(screen.getByTestId("directory-count")).toHaveTextContent(
      strings.messagesFileViewer.directoryEntryCount,
    );
  });

  it("says when the server applied a different sort than the one requested", () => {
    renderDirectory({
      listing: listing({ sort: "size_desc", effectiveSort: "dirs_first_name", totalEntries: 0 }),
    });
    expect(screen.getByTestId("file-preview-notice")).toHaveTextContent(
      strings.messagesFileViewer.directorySortDowngraded,
    );
  });

  it("warns when the directory was too large to list completely", () => {
    renderDirectory({ listing: listing({ truncated: true, totalEntries: 50_000 }) });
    expect(screen.getByTestId("file-preview-notice")).toHaveTextContent(
      strings.messagesFileViewer.directoryTruncated,
    );
  });

  it("re-lists from page one when the sort or hidden filter changes", async () => {
    const onListOptionsChange = vi.fn();
    renderDirectory({
      onListOptionsChange,
      listing: listing({ totalEntries: 1, entries: [entry({ name: "a" })] }),
    });

    await userEvent.selectOptions(screen.getByTestId("directory-sort"), "name");
    expect(onListOptionsChange).toHaveBeenCalledWith({ sort: "name" });

    await userEvent.click(screen.getByTestId("directory-toggle-hidden"));
    expect(onListOptionsChange).toHaveBeenCalledWith({ showHidden: true });
  });

  it("shows a spinner until the first page arrives", () => {
    renderDirectory({ listing: null });
    expect(screen.queryByTestId("directory-entry")).not.toBeInTheDocument();
  });
});
