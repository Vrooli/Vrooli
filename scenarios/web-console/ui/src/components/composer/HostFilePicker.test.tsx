import { renderWithProviders as render } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import HostFilePicker from "./HostFilePicker";
import { listDirectory, resolveFilePreview, type DirectoryEntry, type PreviewListing, type PreviewModel } from "../../api/filePreview";
import { useConversationStore } from "../../stores/useConversationStore";
import type { ConversationEvent } from "../../api/conversation";

vi.mock("../../api/filePreview", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/filePreview")>();
  return { ...actual, resolveFilePreview: vi.fn(), listDirectory: vi.fn() };
});

const CWD = "/home/me/Vrooli/scenarios/web-console";

function model(resolvedPath: string, kind: PreviewModel["kind"], extra: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: `pv:${resolvedPath}`, inputPath: resolvedPath, resolvedPath, basename: resolvedPath.split("/").at(-1) ?? "",
    resolutionBasis: "absolute", kind, mimeType: "", sizeBytes: 10, canPreview: true, canDownload: true, supportsRange: false,
    textContentAvailable: false, listingAvailable: kind === "directory", blobUrl: "", blobHref: "", expiresMs: Date.now() + 60_000,
    warnings: [], candidates: [], searchBases: [], ...extra,
  };
}

function entry(name: string, entryType: DirectoryEntry["entryType"], sizeBytes = 0): DirectoryEntry {
  return { name, entryType, kind: entryType === "directory" ? "directory" : "markdown", sizeBytes, mtimeMs: 0, canPreview: true, symlinkTarget: "", symlinkBroken: false, mode: "", childCount: null };
}

function listing(resolvedPath: string, entries: DirectoryEntry[]): PreviewListing {
  return { resolvedPath, parentPath: "/home/me/Vrooli/scenarios", entries, totalEntries: entries.length, truncated: false, nextPageToken: "", effectiveSort: "dirs_first_name", sort: "dirs_first_name", showHidden: false, warnings: [] };
}

const resolveMock = vi.mocked(resolveFilePreview);
const listMock = vi.mocked(listDirectory);

beforeEach(() => {
  // The executor turns a fixture throw into a rejection, as the real client does.
  resolveMock.mockImplementation((_session, path) => new Promise((resolve) => { resolve(resolveFixture(path)); }));
  listMock.mockImplementation((_session, previewId) => Promise.resolve(previewId === `pv:${CWD}`
    ? listing(CWD, [entry("docs", "directory"), entry("PRD.md", "file", 18_000), entry("README.md", "file", 6_000), entry("shot.png", "file", 4_000)])
    : listing(`${CWD}/docs`, [entry("PLAN.md", "file", 900)])));
});

function resolveFixture(path: string): PreviewModel {
  // "." matches both the session cwd and the project root.
  if (path === ".") return model("", "directory", { candidates: [
    { resolvedPath: "/home/me/Vrooli", resolutionBasis: "project_root", kind: "directory", canPreview: true, basename: "Vrooli" },
    { resolvedPath: CWD, resolutionBasis: "session_cwd", kind: "directory", canPreview: true, basename: "web-console" },
  ] });
  if (path === CWD || path === `${CWD}/docs`) return model(path, "directory");
  if (path === "docs/PLAN.md") return model(`${CWD}/docs/PLAN.md`, "markdown");
  if (path === `${CWD}/shot.png`) return model(path, "image");
  if (path === "/tmp/ui/") return model("/tmp/ui", "directory");
  if (path === "/tmp/ui") return model("/tmp/ui", "directory");
  if (path === "missing.md") return model("", "text", { unresolved: true });
  throw new Error(`unexpected path ${path}`);
}

afterEach(() => {
  vi.clearAllMocks();
  useConversationStore.getState().clearSession("s1");
});

function row(name: string): HTMLElement {
  const el = screen.getAllByTestId("directory-entry").find((node) => node.textContent.includes(name));
  if (!el) throw new Error(`no row ${name}`);
  return el;
}

describe("HostFilePicker", () => {
  it("opens in the session's working folder, not the project root", async () => {
    render(<HostFilePicker open sessionId="s1" onClose={vi.fn()} onAttach={vi.fn()} />);
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(4));
    expect(resolveMock).toHaveBeenCalledWith("s1", CWD, undefined);
  });

  it("toggles files instead of opening them, selects folders by checkbox, and attaches by absolute path", async () => {
    const onAttach = vi.fn();
    const onClose = vi.fn();
    render(<HostFilePicker open sessionId="s1" onClose={onClose} onAttach={onAttach} />);
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(4));
    expect(screen.getByTestId("host-picker-attach")).toHaveProperty("disabled", true);

    fireEvent.click(row("PRD.md"));
    expect(row("PRD.md").getAttribute("aria-checked")).toBe("true");
    // Tapping a file never leaves the picker.
    expect(resolveMock).not.toHaveBeenCalledWith("s1", `${CWD}/PRD.md`, undefined);

    const docsRow = row("docs").closest("li");
    if (!docsRow) throw new Error("no docs li");
    fireEvent.click(within(docsRow).getByTestId("directory-entry-select"));
    expect(screen.getByTestId("host-picker-summary").textContent).toBe("composer.hostPickerSelected");

    // Tapping a selected file again deselects it.
    fireEvent.click(row("README.md"));
    fireEvent.click(row("README.md"));
    expect(row("README.md").getAttribute("aria-checked")).toBe("false");

    fireEvent.click(screen.getByTestId("host-picker-attach"));
    expect(onAttach).toHaveBeenCalledWith([
      { path: `${CWD}/PRD.md`, isDirectory: false, sizeBytes: 18_000 },
      { path: `${CWD}/docs`, isDirectory: true, sizeBytes: 0 },
    ]);
    expect(onClose).toHaveBeenCalled();
  });

  it("keeps selections while walking into a folder", async () => {
    const onAttach = vi.fn();
    render(<HostFilePicker open sessionId="s1" onClose={vi.fn()} onAttach={onAttach} />);
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(4));
    fireEvent.click(row("PRD.md"));
    fireEvent.click(row("docs"));
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(1));
    fireEvent.click(row("PLAN.md"));
    fireEvent.click(screen.getByTestId("host-picker-attach"));
    const picks = (onAttach.mock.calls[0]?.[0] ?? []) as { path: string }[];
    expect(picks.map((p) => p.path)).toEqual([`${CWD}/PRD.md`, `${CWD}/docs/PLAN.md`]);
  });

  it("shows host thumbnails for images the host can render, and the kind icon otherwise", async () => {
    render(<HostFilePicker open sessionId="s1" onClose={vi.fn()} onAttach={vi.fn()} />);
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(4));
    const thumbs = screen.getAllByTestId("directory-entry-thumbnail");
    expect(thumbs).toHaveLength(1);
    const src = thumbs[0]?.querySelector("img")?.getAttribute("src") ?? "";
    // The folder's own preview id is the capability; the child is named, never a path.
    expect(src).toContain(`/api/v1/sessions/s1/file-previews/${encodeURIComponent(`pv:${CWD}`)}/thumbnail?entry=shot.png`);
  });

  it("previews a file over the picker and can select it from the preview", async () => {
    const onAttach = vi.fn();
    render(<HostFilePicker open sessionId="s1" onClose={vi.fn()} onAttach={onAttach} />);
    await waitFor(() => expect(screen.getAllByTestId("directory-entry")).toHaveLength(4));
    // Folders open on tap, so only file rows carry a Preview button.
    expect(screen.getAllByTestId("directory-entry-preview")).toHaveLength(3);

    const shotRow = row("shot.png").closest("li");
    if (!shotRow) throw new Error("no shot.png li");
    fireEvent.click(within(shotRow).getByTestId("directory-entry-preview"));
    await waitFor(() => expect(screen.getByTestId("host-picker-preview-select")).toBeTruthy());
    expect(row("shot.png").getAttribute("aria-checked")).toBe("false");

    fireEvent.click(screen.getByTestId("host-picker-preview-select"));
    expect(screen.getByTestId("host-picker-preview-select").getAttribute("aria-pressed")).toBe("true");
    expect(row("shot.png").getAttribute("aria-checked")).toBe("true");

    fireEvent.click(screen.getByTestId("host-picker-attach"));
    expect(onAttach).toHaveBeenCalledWith([{ path: `${CWD}/shot.png`, isDirectory: false, sizeBytes: 10 }]);
  });

  it("lists files mentioned in the session that resolve to exactly one place", async () => {
    const event = (text: string, sequence: number): ConversationEvent => ({
      id: `e${String(sequence)}`, sessionId: "s1", source: "test", role: "assistant", text, speechParagraphs: [], summarized: false,
      createdAt: "2026-10-29T00:00:00Z", sequence, deliveryState: "", ttsState: "", consumptionState: "",
    });
    useConversationStore.getState().hydrateSession("s1", [event("Plan is in `docs/PLAN.md`.", 1), event("Also `missing.md` and /tmp/ui/", 2)], { lastSeenSequence: 2, lastListenedSequence: 0 } as never);

    const onAttach = vi.fn();
    render(<HostFilePicker open sessionId="s1" onClose={vi.fn()} onAttach={onAttach} />);
    fireEvent.click(screen.getByTestId("host-picker-tab-mentioned"));
    await waitFor(() => expect(screen.getAllByTestId("host-picker-mentioned-entry")).toHaveLength(2));
    const entries = screen.getAllByTestId("host-picker-mentioned-entry");
    expect(entries.map((el) => el.textContent)).toEqual([expect.stringContaining("/tmp/ui"), expect.stringContaining(`${CWD}/docs/PLAN.md`)]);

    const plan = entries[1];
    if (!plan) throw new Error("no PLAN.md entry");
    fireEvent.click(plan);
    fireEvent.click(screen.getByTestId("host-picker-attach"));
    expect(onAttach).toHaveBeenCalledWith([{ path: `${CWD}/docs/PLAN.md`, isDirectory: false, sizeBytes: 10 }]);
  });
});
