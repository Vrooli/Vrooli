import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { RepoFilesStatus } from "../lib/api";
import type { RunAttribution } from "../lib/runAttribution";
import { renderWithQueryClient } from "../test-utils";
import { FileList } from "./FileList";
import type { FileListProps } from "./FileListTypes";

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
    ...renderWithQueryClient(<FileList {...props} />),
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

  it("renders server-resolved groups without matching paths in the browser", () => {
    renderFileList({
      fileViewMode: "grouped",
      resolvedGroups: [{
        key: "contract:tool:compiler",
        kind: "tool",
        id: "compiler",
        label: "compiler",
        root: "internal/tools/compiler",
        source: "contract",
        files: ["src/changed.ts", "src/new.ts"],
      }],
    });

    expect(screen.getByText("compiler")).toBeInTheDocument();
    expect(screen.getByText("src/changed.ts")).toBeInTheDocument();
    expect(screen.queryByText("Other")).not.toBeInTheDocument();
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
        { key: "manual:custom", kind: "", id: "", label: "custom", root: "custom/", source: "manual", files: ["src/new.ts"] },
      ],
    });

    expect(screen.getByText("custom/")).toBeInTheDocument();
    expect(screen.queryByText("scenarios/one/")).not.toBeInTheDocument();
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
      scrollEl.scrollTo = vi.fn();

      frames.shift()?.(0);
      frames.shift()?.(0);

      expect(scrollEl.scrollTo).toHaveBeenCalledWith({ top: 240, behavior: "auto" });
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

    await user.click(screen.getByRole("tab", { name: "From agents 1" }));
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

  it("shows one run dot per run on a collapsed group", async () => {
    const runIndex = new Map<string, RunAttribution>([
      ["src/changed.ts", { runId: "run-alpha", owner: "agent-a" }],
      ["src/new.ts", { runId: "run-beta", owner: "agent-b" }],
    ]);
    const { user } = renderFileList({
      fileViewMode: "grouped",
      runIndex,
      resolvedGroups: [{ key: "contract:scenario:one", kind: "scenario", id: "one", label: "one", root: "scenarios/one/", source: "contract", files: ["src/changed.ts", "src/new.ts"] }],
    });

    await user.click(screen.getByTestId("file-group-toggle-contract:scenario:one"));
    expect(screen.getByTestId("group-run-dots").children).toHaveLength(2);
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
