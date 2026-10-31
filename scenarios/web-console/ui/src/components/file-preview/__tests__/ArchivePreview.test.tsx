import { act, fireEvent, screen, waitFor } from "@testing-library/react";


import { beforeEach, describe, expect, it, vi } from "vitest";


import { renderWithProviders as render } from "../../../test-utils";


import { previewModel } from "../../../test-utils/filePreview";


import { DEFAULT_ARCHIVE_VIEW, listArchive, openArchiveEntry, resolveFilePreview, getFilePreviewText, type PreviewArchive } from "../../../api/filePreview";


import { useFilePreviewController } from "../useFilePreviewController";


import { archiveRows } from "../archiveData";


import { strings } from "../../../consts/strings";


import MessagesFileViewer from "../../MessagesFileViewer";


vi.mock("../../../api/filePreview", async original => ({ ...await original<typeof import("../../../api/filePreview")>(), resolveFilePreview: vi.fn(), listArchive: vi.fn(), openArchiveEntry: vi.fn(), getFilePreviewText: vi.fn() }));


const archive: PreviewArchive = { format: "ZIP", status: "ready", entries: [{ id: "0", name: "docs/hello.html", sizeBytes: 100, status: "ready" }, { id: "1", name: "docs/hello.html", sizeBytes: 90, status: "ready" }, { id: "2", name: "../escape", sizeBytes: 1, status: "unsafe" }, { id: "3", name: ".hidden", sizeBytes: 1, status: "ready" }, { id: "4", name: "linked", sizeBytes: 1, status: "link" }] };


function Harness({ close = vi.fn() }: {
  close?: () => void;
}) {
  const c = useFilePreviewController("session");

  return <><button onClick={() => {
    void c.openPreview("/archive.zip");

  }}>Open archive</button><MessagesFileViewer state={c.state} onArchiveFolder={c.navigateArchiveFolder} onArchiveEntry={c.navigateArchiveEntry} onArchiveViewChange={c.setArchiveView} onClose={close} onHandoff={vi.fn()} onRun={vi.fn()} onReopen={c.reopen} onRendererError={c.reportError} onNavigate={c.navigateTo} onNavigateBack={c.navigateBack} onLoadMore={c.loadMore} onListOptionsChange={c.setListOptions} /></>;

}
async function open() {
  render(<Harness />);

  fireEvent.click(screen.getByText("Open archive"));

  await screen.findByTestId("file-preview-archive");

}
beforeEach(() => {
  vi.clearAllMocks();

  vi.spyOn(window, "scrollTo").mockImplementation(() => { });

  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: vi.fn() });

  vi.mocked(resolveFilePreview).mockResolvedValue(previewModel({ previewId: "root", kind: "archive", resolvedPath: "/archive.zip", basename: "archive.zip", listingAvailable: false, textContentAvailable: false }));

  vi.mocked(listArchive).mockResolvedValue(archive);

  vi.mocked(openArchiveEntry).mockResolvedValue({ status: "ready", target: previewModel({ previewId: "member", archiveMember: true, kind: "code", resolvedPath: "/archive.zip!/docs/hello.html", basename: "hello.html", textContentAvailable: true }) });

  vi.mocked(getFilePreviewText).mockResolvedValue({ resolvedPath: "virtual", kind: "code", mimeType: "text/plain", content: "<script>window.bad=true</script>", truncated: false });

});


describe("archive inspection", () => {
  it("keeps unsafe names inert and preserves duplicate physical identities", () => {
    const root = archiveRows(archive, { ...DEFAULT_ARCHIVE_VIEW });

    expect(root.map(e => e.name)).toEqual(["docs", "../escape", "linked"]);

    const docs = archiveRows(archive, { ...DEFAULT_ARCHIVE_VIEW, folder: "docs" });

    expect(docs.map(e => e.id)).toEqual(["0", "1"]);

    expect(docs.every(e => e.collisions === 2)).toBe(true);

    expect(archiveRows(archive, { ...DEFAULT_ARCHIVE_VIEW, showHidden: true }).some(e => e.name === ".hidden")).toBe(true);

  });


  it("restores folder and query on Back without host resolution or dismissal", async () => {
    await open();

    const back = screen.getByTestId("file-preview-back");

    expect(back).toBeDisabled();

    expect(screen.getAllByTestId("archive-entry-blocked")).toHaveLength(2);

    fireEvent.click(screen.getByRole("button", { name: /docs/ }));

    fireEvent.change(screen.getByTestId("archive-filter"), { target: { value: "hello" } });

    fireEvent.click(entryAt(1));

    await screen.findByTestId("file-preview-source");

    expect(screen.queryByTestId("file-preview-html")).not.toBeInTheDocument();

    expect(screen.queryByTestId("run-file-viewer")).not.toBeInTheDocument();

    expect(screen.queryByTestId("handoff-file-viewer")).not.toBeInTheDocument();

    expect(openArchiveEntry).toHaveBeenCalledWith("session", { previewId: "root", entryId: "1" }, expect.any(AbortSignal));

    expect(resolveFilePreview).toHaveBeenCalledTimes(1);

    fireEvent.click(back);

    await screen.findByTestId("file-preview-archive");

    expect(screen.getByTestId("archive-filter")).toHaveValue("hello");

    expect(screen.getAllByTestId("archive-entry")).toHaveLength(2);

    fireEvent.click(back);

    await waitFor(() => {
      expect(back).toBeDisabled();

    });

    await waitFor(() => { expect(screen.getByTestId("file-preview-navigation")).toHaveFocus(); });
    expect(screen.getByTestId("messages-file-viewer-panel")).toHaveAttribute("data-state", "open");

  });


  it("discards a member result arriving after Back", async () => {
    await open();

    fireEvent.click(screen.getByRole("button", { name: /docs/ }));

    let finish: ((value: Awaited<ReturnType<typeof openArchiveEntry>>) => void) | undefined;

    vi.mocked(openArchiveEntry).mockImplementationOnce(() => new Promise(resolve => {
      finish = resolve;
    }));

    fireEvent.click(entryAt(0));

    fireEvent.click(screen.getByTestId("file-preview-back"));

    await screen.findByTestId("file-preview-archive");

    expect(vi.mocked(openArchiveEntry).mock.calls.at(-1)?.[2]?.aborted).toBe(true);

    await act(async () => {
      finish?.({ status: "ready", target: previewModel({ kind: "image" }) });

      await Promise.resolve();

    });

    expect(screen.getByTestId("file-preview-archive")).toBeInTheDocument();

  });


  it("retains original download when inspection exceeds a limit", async () => {
    vi.mocked(listArchive).mockResolvedValue({ format: "", status: "limit", entries: [] });

    await open();

    expect(screen.getByText(strings.archivePreview.limit)).toBeInTheDocument();

    expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();

  });


  it("retains original download when the archive request fails", async () => {
    vi.mocked(listArchive).mockRejectedValue(new Error("Connection lost"));

    render(<Harness />);

    fireEvent.click(screen.getByText("Open archive"));

    await screen.findByText("Connection lost");

    expect(screen.getByTestId("file-preview-error-download")).toHaveAttribute("href", "/preview/blob");

  });


  it("sorts and reveals hidden entries locally and restores those options on Back", async () => {
    vi.mocked(listArchive).mockResolvedValue({
      format: "ZIP", status: "ready", entries: [
        { id: "0", name: "b.txt", sizeBytes: 1, status: "ready" },
        { id: "1", name: "a.txt", sizeBytes: 3, status: "ready" },
        { id: "2", name: ".hidden", sizeBytes: 2, status: "ready" },
      ]
    });

    await open();

    expect(screen.getAllByTestId("archive-entry").map(e => e.textContent)).toEqual([
      expect.stringContaining("a.txt"), expect.stringContaining("b.txt"),
    ]);

    fireEvent.click(screen.getByRole("button", { name: strings.archivePreview.options }));

    fireEvent.change(screen.getByRole("combobox"), { target: { value: "size" } });

    fireEvent.click(screen.getByRole("checkbox"));

    fireEvent.keyDown(screen.getByTestId("inspection-options"), { key: "Escape" });

    await waitFor(() => {
      expect(screen.queryByTestId("inspection-options")).not.toBeInTheDocument();
    });

    expect(screen.getAllByTestId("archive-entry").map(e => e.textContent)).toEqual([
      expect.stringContaining("a.txt"), expect.stringContaining(".hidden"), expect.stringContaining("b.txt"),
    ]);

    fireEvent.click(entryAt(0));
    await screen.findByTestId("file-preview-source");

    fireEvent.click(screen.getByTestId("file-preview-back"));
    await screen.findByTestId("file-preview-archive");

    expect(screen.getAllByTestId("archive-entry")).toHaveLength(3);

    expect(listArchive).toHaveBeenCalledTimes(1);

  });


  it("bounds rendered rows while filtering the full index", async () => {
    vi.mocked(listArchive).mockResolvedValue({ format: "ZIP", status: "ready", entries: Array.from({ length: 230 }, (_, i) => ({ id: String(i), name: `file-${String(i)}.txt`, sizeBytes: i, status: "ready" })) });

    await open();
    expect(screen.getAllByTestId("archive-entry")).toHaveLength(100);

    fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.directoryLoadMore }));

    expect(screen.getAllByTestId("archive-entry")).toHaveLength(200);

    fireEvent.change(screen.getByTestId("archive-filter"), { target: { value: "file-229" } });

    expect(screen.getAllByTestId("archive-entry")).toHaveLength(1);

    expect(screen.getByTestId("archive-entry")).toHaveTextContent("file-229.txt");

  });


  it("uses virtual provenance when reopening a member", async () => {
    await open();

    fireEvent.click(screen.getByRole("button", { name: /docs/ }));

    fireEvent.click(entryAt(0));

    await screen.findByTestId("file-preview-source");

    vi.mocked(getFilePreviewText).mockRejectedValueOnce(new Error("expired"));

    // Renderer error recovery remains tied to the member handle.
    fireEvent.click(screen.getByTestId("file-preview-back"));

    await screen.findByTestId("file-preview-archive");

    vi.mocked(openArchiveEntry).mockResolvedValueOnce({ status: "expired", target: null });

    fireEvent.click(entryAt(0));

    await screen.findByText(strings.archivePreview.expired);

    fireEvent.click(screen.getByTestId("file-preview-reopen"));

    await waitFor(() => {
      expect(openArchiveEntry).toHaveBeenCalledTimes(3);

    });

    expect(resolveFilePreview).toHaveBeenCalledTimes(1);

  });


});



function entryAt(index: number): HTMLElement {
  const entry = screen.getAllByTestId("archive-entry")[index];


  if (!entry) throw new Error(`Missing archive entry ${String(index)}`);


  return entry;


}
