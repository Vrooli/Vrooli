import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { RepoFilesStatus } from "../lib/api";
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
    approvedChanges: { available: true, committableFiles: 1 },
    approvedPaths: new Set(["src/changed.ts"]),
    onStageApproved: vi.fn(),
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
    expect(screen.getByText("approved")).toBeInTheDocument();
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
      approvedChanges: undefined,
    });

    expect(screen.getByTestId("empty-state")).toBeInTheDocument();
    expect(screen.getByText("No changes detected")).toBeInTheDocument();
    expect(screen.getByText("Working directory is clean")).toBeInTheDocument();
    expect(screen.queryByTestId("stage-all-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("unstage-all-button")).not.toBeInTheDocument();
  });
});
