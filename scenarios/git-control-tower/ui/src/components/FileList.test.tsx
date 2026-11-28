import { fireEvent, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { RepoFilesStatus } from "../lib/api";
import type { RunAttribution } from "../lib/runAttribution";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { FileList } from "./FileList";
import type { FileListProps } from "./FileListTypes";

// These interaction tests exercise actions, not virtual viewport measurement.
// JSDOM has no layout; declare a visible viewport using the same adapter seam
// as history and proposal interaction tests, retaining all behavior assertions.
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getTotalSize: () => count * 64,
    getVirtualItems: () => Array.from({ length: count }, (_, index) => ({ index, start: index * 64, size: 64, key: index })),
    measureElement: vi.fn(), scrollToIndex: vi.fn(),
  }),
}));

const mobile = vi.hoisted(() => ({ enabled: false }));

vi.mock("../hooks", () => ({
  useIsMobile: () => mobile.enabled,
  useLongPress: () => ({}),
}));

function renderFileList(overrides: Partial<FileListProps> = {}) {
  const files: RepoFilesStatus = {
    conflicts: [],
    staged: ["src/committed.ts"],
    unstaged: ["src/changed.ts"],
    untracked: ["src/new.ts"],
    binary: ["src/new.ts"],
    statuses: {
      "src/committed.ts": "M",
      "src/changed.ts": "M",
      "src/new.ts": "?",
    },
  };

  const props: FileListProps = {
    files,
    selectedFiles: [],
    selectionKey: ({ path, staged }) => `${staged ? "1" : "0"}:${path}`,
    onSelectFile: vi.fn(),
    onStageFile: vi.fn(),
    onUnstageFile: vi.fn(),
    onDiscardFile: vi.fn(),
    onIgnoreFile: vi.fn(),
    onStageAll: vi.fn(),
    onUnstageAll: vi.fn(),
    isStaging: false,
    isDiscarding: false,
    isIgnoring: false,
    confirmingDiscard: null,
    onConfirmDiscard: vi.fn(),
    confirmingIgnore: null,
    onConfirmIgnore: vi.fn(),
    fillHeight: false,
    ...overrides,
  };

  return {
    props,
    user: userEvent.setup(),
    ...renderWithProviders(<FileList {...props} />),
  };
}

describe("FileList", () => {
  it("offers exact-basename bulk staging only when another changed match exists", async () => {
    const onStageFilesWithSameName = vi.fn();
    const { user } = renderFileList({
      files: {
        staged: ["services/one/go.mod"],
        unstaged: ["go.mod"],
        untracked: ["docs/go.mod.bak"],
        conflicts: [],
      },
      onStageFilesWithSameName,
    });

    fireEvent.contextMenu(screen.getByText("go.mod").closest("li") as HTMLLIElement, {
      clientX: 20,
      clientY: 20,
    });
    const action = screen.getByRole("menuitem", { name: "Stage all changed files named go.mod" });
    await user.click(action);
    expect(onStageFilesWithSameName).toHaveBeenCalledWith("go.mod");

    await user.click(screen.getByTestId("file-section-toggle-untracked"));
    fireEvent.contextMenu(screen.getByText("docs/go.mod.bak").closest("li") as HTMLLIElement, {
      clientX: 20,
      clientY: 20,
    });
    expect(screen.queryByRole("menuitem", { name: /Stage all changed files named go.mod.bak/ })).not.toBeInTheDocument();
  });

  it("keeps stage/unstage controls responsive while a stage is in flight", async () => {
    // Regression guard for the serialization complaint: staging is optimistic and the
    // batcher coalesces follow-up clicks, so an in-flight stage must not disable the
    // stage/unstage controls — the user can keep queueing groups.
    const { props, user } = renderFileList({ isStaging: true });

    const stageAll = screen.getByTestId("stage-all-button");
    const unstageAll = screen.getByTestId("unstage-all-button");
    expect(stageAll).toBeEnabled();
    expect(unstageAll).toBeEnabled();

    await user.click(stageAll);
    expect(props.onStageAll).toHaveBeenCalledTimes(1);
  });

  it("renders change sections and routes file actions to their callbacks", async () => {
    const { props, user } = renderFileList();

    expect(screen.getByTestId("file-list-panel")).toBeInTheDocument();
    expect(screen.getByText("Changes")).toBeInTheDocument();
    expect(screen.getByTestId("file-section-staged")).toBeInTheDocument();
    expect(screen.getByTestId("file-section-unstaged")).toBeInTheDocument();
    expect(screen.getByText("src/committed.ts")).toBeInTheDocument();
    expect(screen.getByText("src/changed.ts")).toBeInTheDocument();

    await user.click(screen.getByTestId("file-section-toggle-untracked"));
    expect(screen.getByText("src/new.ts")).toBeInTheDocument();
    expect(screen.queryByText("approved")).not.toBeInTheDocument();
    expect(screen.getByText("bin")).toBeInTheDocument();

    await user.click(screen.getByTestId("stage-all-button"));
    expect(props.onStageAll).toHaveBeenCalledTimes(1);

    await user.click(screen.getByTestId("unstage-all-button"));
    expect(props.onUnstageAll).toHaveBeenCalledTimes(1);

    const changedRow = screen.getByText("src/changed.ts").closest("li");
    expect(changedRow).not.toBeNull();
    fireEvent.click(changedRow as HTMLLIElement);
    expect(props.onSelectFile).toHaveBeenCalledWith(
      "src/changed.ts",
      false,
      expect.objectContaining({ type: "click" }),
    );

    await user.click(screen.getByTestId("file-action-unstaged"));
    expect(props.onStageFile).toHaveBeenCalledWith("src/changed.ts");

    await user.click(screen.getByTestId("file-discard-unstaged"));
    expect(props.onConfirmDiscard).toHaveBeenCalledWith("src/changed.ts");

    await user.click(screen.getByTestId("file-ignore-unstaged"));
    expect(props.onConfirmIgnore).toHaveBeenCalledWith("src/changed.ts");
  });

  it("keeps partially staged files in both grouped categories with the correct actions", async () => {
    const { props, user } = renderFileList({
      fileViewMode: "grouped",
      files: { staged: ["src/partial.ts"], unstaged: ["src/partial.ts"], untracked: [], conflicts: [] },
      resolvedGroups: [{ key: "partial", kind: "scenario", id: "partial", label: "partial", source: "contract", root: "src", files: ["src/partial.ts"] }],
    });
    await user.click(screen.getByTestId("file-group-open-partial"));
    expect(screen.getAllByText("partial.ts")).toHaveLength(2);
    await user.click(screen.getByTestId("file-action-staged"));
    expect(props.onUnstageFile).toHaveBeenCalledWith("src/partial.ts");
    await user.click(screen.getByTestId("file-action-unstaged"));
    expect(props.onStageFile).toHaveBeenCalledWith("src/partial.ts");
  });

  it("renders server-resolved groups as a group index, then opens their folders", async () => {
    const { user } = renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [{
        key: "contract:tool:compiler",
        kind: "tool",
        id: "compiler",
        label: "compiler",
        root: "",
        source: "contract",
        files: ["src/changed.ts", "src/new.ts"],
      }],
    });

    expect(screen.getByText("compiler")).toBeInTheDocument();
    expect(screen.getByLabelText("2 files")).toHaveTextContent("2");
    expect(screen.queryByText("changed.ts")).not.toBeInTheDocument();
    expect(screen.queryByText("Other")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("file-group-open-contract:tool:compiler"));
    expect(await screen.findByTestId("change-group-drill-header")).toBeInTheDocument();
    expect(screen.getByText("src")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Open folder src" }));
    expect(await screen.findByText("changed.ts")).toBeInTheDocument();
    await user.click(screen.getByTestId("file-section-toggle-untracked"));
    expect(await screen.findByText("new.ts")).toBeInTheDocument();
  });

  it("renders one kind band for each contiguous contract kind", () => {
    renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [
        { key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts"] },
        { key: "contract:resource:two", kind: "resource", id: "two", label: "two", root: "resources/two/", source: "contract", files: ["src/new.ts"] },
        { key: "contract:package:three", kind: "package", id: "three", label: "three", root: "packages/three/", source: "contract", files: ["src/committed.ts"] },
      ],
    });

    expect(screen.getByTestId("file-kind-band-scenario")).toHaveTextContent("Scenarios");
    expect(screen.getByTestId("file-kind-band-resource")).toHaveTextContent("Resources");
    expect(screen.getByTestId("file-kind-band-package")).toHaveTextContent("Packages");
  });

  it("suppresses redundant bands when all groups share one kind", () => {
    renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [
        { key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts"] },
        { key: "contract:scenario:two", kind: "scenario", id: "two", label: "two", root: "scenarios/two/", source: "contract", files: ["src/new.ts"] },
      ],
    });

    expect(screen.getAllByTestId("file-kind-band-scenario")).toHaveLength(1);
    expect(screen.queryByTestId("file-kind-band-resource")).not.toBeInTheDocument();
  });

  it("keeps paths for manual groups but omits them for contract groups", () => {
    renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [
        { key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts"] },
        { key: "manual:custom", kind: "", id: "", label: "Custom group", root: "custom/", source: "manual", files: ["src/new.ts"] },
      ],
    });

    expect(screen.getByText("custom")).toBeInTheDocument();
    expect(screen.queryByText("scenarios/one")).not.toBeInTheDocument();
  });

  it("uses full-bleed group rows instead of card containers", () => {
    renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [{ key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts"] }],
    });

    expect(screen.getByTestId("file-group-contract:scenario:one").className).not.toMatch(/rounded-lg|bg-slate-950\/40/);
  });

  it("spins only the touched row when its path is pending", () => {
    const { container } = renderFileList({
      pendingPaths: new Set(["src/changed.ts"]),
    });

    const changedRow = screen.getByText("src/changed.ts").closest("li");
    const stagedRow = screen.getByText("src/committed.ts").closest("li");
    expect(changedRow?.querySelector(".animate-spin")).not.toBeNull();
    expect(stagedRow?.querySelector(".animate-spin")).toBeNull();
    // Exactly one row shows the spinner.
    expect(container.querySelectorAll(".animate-spin").length).toBe(1);
  });

  it("hides global bulk stage/unstage buttons while in selection mode", () => {
    const { rerender, props } = renderFileList({ mobileSelectionMode: true });
    expect(screen.queryByTestId("stage-all-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("unstage-all-button")).not.toBeInTheDocument();

    // Leaving selection mode restores the bulk actions.
    rerender(<FileList {...props} mobileSelectionMode={false} />);
    expect(screen.getByTestId("stage-all-button")).toBeInTheDocument();
    expect(screen.getByTestId("unstage-all-button")).toBeInTheDocument();
  });

  it("persists subsection collapse across a remount and applies defaults on first load", async () => {
    const first = renderFileList();
    // Modified (unstaged) defaults to expanded, so its file is visible.
    expect(screen.getByText("src/changed.ts")).toBeInTheDocument();
    // Untracked defaults to collapsed, so its file is hidden until toggled.
    expect(screen.queryByText("src/new.ts")).not.toBeInTheDocument();

    // Collapse the Modified subsection.
    await first.user.click(screen.getByTestId("file-section-toggle-unstaged"));
    expect(screen.queryByText("src/changed.ts")).not.toBeInTheDocument();

    // Remount a fresh FileList: the collapse must persist via localStorage.
    first.unmount();
    renderFileList();
    expect(screen.queryByText("src/changed.ts")).not.toBeInTheDocument();
    expect(screen.getByTestId("file-section-staged")).toBeInTheDocument();
  });

  it("opens the owning group and untracked category before resolving a scroll target", async () => {
    const done = vi.fn();
    renderFileList({ fileViewMode: "grouped", scrollToFile: "src/new.ts", onScrollComplete: done,
      resolvedGroups: [{ key: "target-group", kind: "scenario", id: "target", label: "target", source: "contract", root: "src", files: ["src/new.ts"] }],
    });
    expect(await screen.findByText("new.ts")).toBeInTheDocument();
    expect(screen.getByTestId("change-group-drill-header")).toBeInTheDocument();
    expect(screen.getByTestId("file-section-toggle-untracked")).toHaveAttribute("aria-expanded", "true");
    await vi.waitFor(() => expect(done).toHaveBeenCalled());
  });

  it("saves the Changes scroll position to the store on scroll", () => {
    const store = { current: 0 } as React.MutableRefObject<number>;
    const { container } = renderFileList({ scrollTopStore: store });
    const scrollEl = container.querySelector("div.overflow-auto") as HTMLElement;
    expect(scrollEl).not.toBeNull();

    // jsdom has no layout, so pin scrollTop before dispatching the scroll event.
    Object.defineProperty(scrollEl, "scrollTop", { configurable: true, value: 240 });
    fireEvent.scroll(scrollEl);
    expect(store.current).toBe(240);
  });

  it("restores the owned Changes scroller after the panel remounts", () => {
    const store = { current: 240 } as React.MutableRefObject<number>;
    const frames: FrameRequestCallback[] = [];
    const originalRAF = window.requestAnimationFrame;
    const originalCancelRAF = window.cancelAnimationFrame;
    window.requestAnimationFrame = vi.fn((callback: FrameRequestCallback) => {
      frames.push(callback);
      return frames.length;
    });
    window.cancelAnimationFrame = vi.fn();

    try {
      const { container } = renderFileList({ scrollTopStore: store });
      const scrollEl = container.querySelector("div.overflow-auto") as HTMLDivElement;

      // Other components may queue frames too; flush until layout settles.
      for (let i = 0; i < 10 && frames.length; i += 1) frames.shift()?.(0);

      expect(scrollEl.scrollTop).toBe(240);
    } finally {
      window.requestAnimationFrame = originalRAF;
      window.cancelAnimationFrame = originalCancelRAF;
    }
  });

  it("keeps the mobile selection toolbar in the owned Changes scroller", () => {
    mobile.enabled = true;
    try {
      const { container } = renderFileList({
        mobileSelectionMode: true,
        selectedFiles: [{ path: "src/changed.ts", staged: false }],
      });
      const scrollRegion = screen.getByTestId("changes-scroll-region");
      const toolbar = screen.getByTestId("mobile-selection-toolbar");
      expect(scrollRegion).toContainElement(toolbar);
      expect(container.querySelector("[data-testid=mobile-selection-toolbar].sticky")).toBeNull();
    } finally {
      mobile.enabled = false;
    }
  });

  it("shows clean working tree copy when no files changed", () => {
    renderFileList({
      files: {
        conflicts: [],
        staged: [],
        unstaged: [],
        untracked: [],
      },
    });

    expect(screen.getByTestId("empty-state")).toBeInTheDocument();
    expect(screen.getByText("No changes detected")).toBeInTheDocument();
    expect(screen.getByText("Working directory is clean")).toBeInTheDocument();
    expect(screen.queryByTestId("stage-all-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("unstage-all-button")).not.toBeInTheDocument();
  });

  it("filters the Changes list to files with run attribution", async () => {
    const runIndex = new Map<string, RunAttribution>([
      ["src/changed.ts", { runId: "run-alpha", owner: "agent-a" }],
    ]);
    const { user } = renderFileList({ runIndex });

    await user.click(screen.getByRole("tab", { name: "Agents 1" }));
    expect(screen.getByText("src/changed.ts")).toBeInTheDocument();
    expect(screen.queryByText("src/committed.ts")).not.toBeInTheDocument();
  });

  it("opens the run sheet from an attributed run chip", async () => {
    const runIndex = new Map<string, RunAttribution>([
      ["src/changed.ts", { runId: "run-alpha", owner: "agent-a", appliedAt: "2026-08-30T12:00:00Z" }],
    ]);
    const { user } = renderFileList({ runIndex });

    await user.click(screen.getByTestId("run-chip-run-alph"));
    expect(screen.getByTestId("run-sheet")).toHaveTextContent("Auto-applied by the sandbox. Nothing here has been reviewed.");
    expect(screen.getByTestId("run-sheet")).toHaveTextContent("1 file in this run");
  });

  it("selects run files without invoking a staging callback", async () => {
    const runIndex = new Map<string, RunAttribution>([
      ["src/changed.ts", { runId: "run-alpha", owner: "agent-a" }],
    ]);
    const { props, user } = renderFileList({ runIndex });

    await user.click(screen.getByTestId("run-chip-run-alph"));
    await user.click(screen.getByRole("button", { name: /Select all 1/ }));
    expect(props.onSelectFile).toHaveBeenCalledWith("src/changed.ts", false, expect.anything());
    expect(props.onStageFile).not.toHaveBeenCalled();
  });

  it("shows one run dot per run on the group index", () => {
    const runIndex = new Map<string, RunAttribution>([
      ["src/changed.ts", { runId: "run-alpha", owner: "agent-a" }],
      ["src/new.ts", { runId: "run-beta", owner: "agent-b" }],
    ]);
    renderFileList({
      fileViewMode: "grouped",
      runIndex,
      resolvedGroups: [{ key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts", "src/new.ts"] }],
    });

    expect(screen.getByTestId("group-run-dots").children).toHaveLength(2);
    expect(screen.getByTestId("file-group-open-contract:scenario:one")).toBeInTheDocument();
  });

  it("shows run dots only for files included by the active group filter", async () => {
    const { user } = renderFileList({ fileViewMode: "grouped",
      runIndex: new Map([["src/changed.ts", { runId: "hidden-run", owner: "agent" }]]),
      resolvedGroups: [{ key: "filtered", id: "filtered", kind: "scenario", label: "filtered", source: "contract", files: ["src/committed.ts", "src/changed.ts"] }],
    });
    await user.click(screen.getByRole("tab", { name: "Staged" }));
    expect(screen.queryByTestId("group-run-dots")).not.toBeInTheDocument();
  });

  it("drills into a change group by folder and stages only the current folder scope", async () => {
    const onStagePaths = vi.fn();
    const groupedFiles = [
      "docs/README.md",
      "docs/agent-system/README.md",
      "docs/agent-system/decisions/authority.md",
      "docs/concepts/glossary.md",
    ];
    const { user } = renderFileList({
      files: { staged: [], unstaged: groupedFiles, untracked: [], conflicts: [] },
      fileViewMode: "grouped",
      resolvedGroups: [{ key: "contract:docs", id: "docs", kind: "docs", label: "docs", root: "docs/", source: "contract", files: groupedFiles }],
      onStagePaths,
    });

    const scroller = screen.getByTestId("changes-scroll-region");
    scroller.scrollTop = 120;
    fireEvent.scroll(scroller);
    await user.click(screen.getByTestId("file-group-open-contract:docs"));
    expect(scroller.scrollTop).toBe(0);
    expect(await screen.findByTestId("change-group-drill-header")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Conflicts" }));
    expect(await screen.findByTestId("change-group-empty")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "All" }));
    expect(screen.getByText("agent-system")).toBeInTheDocument();
    expect(screen.getByText("concepts")).toBeInTheDocument();
    expect(screen.getAllByTestId("file-item-unstaged").some((row) => row.textContent?.includes("README.md"))).toBe(true);

    const agentSystemRow = within(screen.getAllByTestId("change-folder-row")[0] as HTMLElement);
    const stageFolderButton = agentSystemRow.getByTestId("change-folder-stage");
    expect(stageFolderButton).toHaveAttribute("aria-label", "Stage 2 changed files in agent-system");
    await user.click(stageFolderButton);
    expect(onStagePaths).toHaveBeenCalledWith([
      "docs/agent-system/README.md",
      "docs/agent-system/decisions/authority.md",
    ]);

    await user.click(screen.getByRole("button", { name: "Open folder agent-system" }));
    expect(await screen.findByText("decisions")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Go to parent folder" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Go to parent folder" }));
    await user.click(screen.getByRole("button", { name: "Back to all change groups" }));
    expect(scroller.scrollTop).toBe(120);
  });

  it("stages a whole group from the group index", async () => {
    const onStagePaths = vi.fn();
    const { user } = renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [{
        key: "contract:scenario:one",
        kind: "scenario",
        id: "one",
        label: "one",
        root: "scenarios/one/",
        source: "contract",
        files: ["src/changed.ts", "src/new.ts"],
      }],
      onStagePaths,
    });

    const stageGroup = screen.getByTestId("file-group-stage-contract:scenario:one");
    expect(stageGroup).toHaveAttribute("aria-label", "Stage 2 files in one");
    await user.click(stageGroup);
    expect(onStagePaths).toHaveBeenCalledWith(["src/changed.ts", "src/new.ts"]);
  });

  it("stages a whole group from the group action menu", async () => {
    const onStagePaths = vi.fn();
    const { user } = renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [{
        key: "contract:scenario:one",
        kind: "scenario",
        id: "one",
        label: "one",
        root: "scenarios/one/",
        source: "contract",
        files: ["src/changed.ts", "src/new.ts"],
      }],
      onStagePaths,
    });

    await user.click(screen.getByTestId("file-group-actions-contract:scenario:one"));
    expect(screen.getByRole("menuitem", { name: /Stage 2 files in one/ })).toBeInTheDocument();
    await user.click(screen.getByTestId("group-context-stage"));
    expect(onStagePaths).toHaveBeenCalledWith(["src/changed.ts", "src/new.ts"]);
  });

  it("counts a partially staged file once in the group header", () => {
    renderFileList({
      fileViewMode: "grouped",
      files: { staged: ["src/partial.ts"], unstaged: ["src/partial.ts"], untracked: [], conflicts: [] },
      resolvedGroups: [{ key: "partial", kind: "scenario", id: "partial", label: "partial", source: "contract", files: ["src/partial.ts"] }],
    });
    const header = screen.getByTestId("file-group-header-partial");
    expect(within(header).getByLabelText("1 file")).toBeInTheDocument();
  });

  it("keeps stage and unstage in the file context menu", async () => {
    const { props, user } = renderFileList();

    fireEvent.contextMenu(screen.getByText("src/changed.ts").closest("li") as HTMLLIElement, { clientX: 10, clientY: 10 });
    await user.click(screen.getByRole("menuitem", { name: "Stage file" }));
    expect(props.onStageFile).toHaveBeenCalledWith("src/changed.ts");

    fireEvent.contextMenu(screen.getByText("src/committed.ts").closest("li") as HTMLLIElement, { clientX: 10, clientY: 10 });
    await user.click(screen.getByRole("menuitem", { name: "Unstage file" }));
    expect(props.onUnstageFile).toHaveBeenCalledWith("src/committed.ts");
  });

  it("shows per-file line totals on desktop rows", () => {
    renderFileList({
      fileStats: {
        staged: { "src/committed.ts": { additions: 2, deletions: 1, files: 1 } },
        unstaged: { "src/changed.ts": { additions: 14, deletions: 3, files: 1 } },
        untracked: {},
      },
    });
    const stats = screen.getAllByTestId("file-line-stats");
    expect(stats.some((row) => row.textContent === "+14−3")).toBe(true);
  });

  it("stages the desktop multi-file selection from the toolbar", async () => {
    const onStagePaths = vi.fn();
    const { user } = renderFileList({
      selectedFiles: [
        { path: "src/changed.ts", staged: false },
        { path: "src/new.ts", staged: false },
      ],
      onStagePaths,
    });
    expect(screen.getByTestId("changes-selection-toolbar")).toBeInTheDocument();
    await user.click(screen.getByTestId("selection-stage"));
    expect(onStagePaths).toHaveBeenCalledWith(["src/changed.ts", "src/new.ts"]);
  });

  it("recovers from a filter that matches no files", async () => {
    const { user } = renderFileList();
    await user.click(screen.getByRole("tab", { name: "Conflicts" }));
    expect(screen.getByTestId("filter-empty-state")).toHaveTextContent("No files in “Conflicts”");
    await user.click(screen.getByRole("button", { name: "Show all changes" }));
    expect(screen.getByText("src/changed.ts")).toBeInTheDocument();
  });

  it("exposes Reveal in file tree from the mobile action sheet", async () => {
    mobile.enabled = true;
    const onRevealInTree = vi.fn();
    try {
      const { user } = renderFileList({ onRevealInTree });
      await user.click(screen.getByTestId("file-item-unstaged-more-actions"));
      await user.click(screen.getByTestId("reveal-in-tree-action"));
      expect(onRevealInTree).toHaveBeenCalledWith("src/changed.ts");
    } finally {
      mobile.enabled = false;
    }
  });
});

it("keeps compact Changes tabs styled and keyboard selectable", () => {
 renderFileList();
 const all = screen.getByRole("tab", { name: "All" });
 expect(all.closest("[data-rcl-tabs]")).toHaveAttribute("data-rcl-tabs-density", "compact");
 expect(all.closest("[data-rcl-tabs]")).toHaveAttribute("data-rcl-tabs-variant", "underline");
 expect([...document.querySelectorAll("style[data-rcl-sheet]")].some(sheet => sheet.textContent?.includes('[data-rcl-tabs-density="compact"] [data-rcl-tab]'))).toBe(true);
 expect(all).toHaveAttribute("aria-selected", "true");
 fireEvent.keyDown(all, { key: "ArrowRight" });
 expect(screen.getByRole("tab", { name: /Agents/ })).toHaveAttribute("aria-selected", "true");
});
