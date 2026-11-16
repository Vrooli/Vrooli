import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils/index.js";
import { previewModel } from "../../test-utils/previewFixtures.js";
import { NO_CAPABILITIES, type PreviewArchive, type PreviewSource } from "../../source/types.js";
import { useFilePreviewController } from "../../viewer/useFilePreviewController.js";
import { archiveRows } from "../archiveData.js";
import { DEFAULT_ARCHIVE_VIEW } from "../types.js";
import { strings } from "../../i18n/strings.js";
import { FileViewer } from "../../viewer/FileViewer.js";

// An archive-capable source whose operations are spies, so each test scripts
// the outcome of one call. The spies keep the session as their first argument.
const resolveFilePreview = vi.fn();
const listArchive = vi.fn();
const openArchiveEntry = vi.fn();
const getFilePreviewText = vi.fn();
const source: PreviewSource = {
  scope: "session",
  capabilities: { ...NO_CAPABILITIES, archives: true },
  resolve: async (path, options) => ({ outcome: "resolved", model: await resolveFilePreview("session", path, options?.context) }),
  readText: (model) => getFilePreviewText("session", model.previewId),
  listDirectory: () => Promise.reject(new Error("unused")),
  listArchive: (model, signal) => listArchive("session", model.previewId, signal),
  openArchiveEntry: (entry, signal) => openArchiveEntry("session", entry, signal),
};

const archive: PreviewArchive = { format: "ZIP", status: "ready", entries: [{ id: "0", name: "docs/hello.html", sizeBytes: 100, status: "ready" }, { id: "1", name: "docs/hello.html", sizeBytes: 90, status: "ready" }, { id: "2", name: "../escape", sizeBytes: 1, status: "unsafe" }, { id: "3", name: ".hidden", sizeBytes: 1, status: "ready" }, { id: "4", name: "linked", sizeBytes: 1, status: "link" }] };
function Harness({ close = vi.fn() }: { close?: () => void }) {
  const controller = useFilePreviewController(source);
  return <>
    <button onClick={() => { void controller.openPreview("/archive.zip"); }}>Open archive</button>
    <FileViewer source={source} controller={controller} onClose={close} />
  </>;
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
  resolveFilePreview.mockResolvedValue(previewModel({ previewId: "root", kind: "archive", resolvedPath: "/archive.zip", basename: "archive.zip", listingAvailable: false, textContentAvailable: false }));
  listArchive.mockResolvedValue(archive);
  openArchiveEntry.mockResolvedValue({ status: "ready", target: previewModel({ previewId: "member", archiveMember: true, kind: "code", resolvedPath: "/archive.zip!/docs/hello.html", basename: "hello.html", textContentAvailable: true }) });
  getFilePreviewText.mockResolvedValue({ resolvedPath: "virtual", kind: "code", mimeType: "text/plain", content: "<script>window.bad=true</script>", truncated: false });
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
    let finish: ((value: { status: string; target: ReturnType<typeof previewModel> | null }) => void) | undefined;
    openArchiveEntry.mockImplementationOnce(() => new Promise(resolve => {
      finish = resolve;
    }));
    fireEvent.click(entryAt(0));
    fireEvent.click(screen.getByTestId("file-preview-back"));
    await screen.findByTestId("file-preview-archive");
    expect((openArchiveEntry.mock.calls.at(-1)?.[2] as AbortSignal | undefined)?.aborted).toBe(true);
    await act(async () => {
      finish?.({ status: "ready", target: previewModel({ kind: "image" }) });
      await Promise.resolve();
    });
    expect(screen.getByTestId("file-preview-archive")).toBeInTheDocument();
  });
  it("retains original download when inspection exceeds a limit", async () => {
    listArchive.mockResolvedValue({ format: "", status: "limit", entries: [] });
    await open();
    expect(screen.getByText(strings.archivePreview.limit)).toBeInTheDocument();
    expect(screen.getByTestId("file-preview-download")).toBeInTheDocument();
  });
  it("retains original download when the archive request fails", async () => {
    listArchive.mockRejectedValue(new Error("Connection lost"));
    render(<Harness />);
    fireEvent.click(screen.getByText("Open archive"));
    await screen.findByText("Connection lost");
    expect(screen.getByTestId("file-preview-error-download")).toHaveAttribute("href", "/preview/blob");
  });
  it("sorts and reveals hidden entries locally and restores those options on Back", async () => {
    listArchive.mockResolvedValue({
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
    listArchive.mockResolvedValue({ format: "ZIP", status: "ready", entries: Array.from({ length: 230 }, (_, i) => ({ id: String(i), name: `file-${String(i)}.txt`, sizeBytes: i, status: "ready" })) });
    await open();
    expect(screen.getAllByTestId("archive-entry")).toHaveLength(100);
    fireEvent.click(screen.getByRole("button", { name: strings.viewer.directoryLoadMore }));
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
    getFilePreviewText.mockRejectedValueOnce(new Error("expired"));
    // Renderer error recovery remains tied to the member handle.
    fireEvent.click(screen.getByTestId("file-preview-back"));
    await screen.findByTestId("file-preview-archive");
    openArchiveEntry.mockResolvedValueOnce({ status: "expired", target: null });
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
