import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { RepoFilesStatus, RepoFileStats } from "../lib/api";
import { renderWithQueryClient } from "../test-utils";
import { FileListFlatView } from "./FileListFlatView";
import { FileListGroupedView } from "./FileListGroupedView";
import { FileListMobileActions } from "./FileListMobileActions";
import type { FileCategory, GroupingRule, SelectedFileEntry } from "./FileListTypes";

const files: RepoFilesStatus = {
  conflicts: ["src/conflict.ts"],
  staged: ["src/staged.ts"],
  unstaged: ["src/changed.ts"],
  untracked: ["src/new.ts"],
  binary: ["src/new.ts"],
  statuses: {
    "src/conflict.ts": "UU",
    "src/staged.ts": "M",
    "src/changed.ts": "M",
    "src/new.ts": "?",
  },
};

const fileStats: RepoFileStats = {
  staged: {
    "src/staged.ts": { additions: 2, deletions: 1, files: 1 },
  },
  unstaged: {
    "src/conflict.ts": { additions: 5, deletions: 3, files: 1 },
    "src/changed.ts": { additions: 8, deletions: 2, files: 1 },
  },
  untracked: {
    "src/new.ts": { additions: 12, deletions: 0, files: 1 },
  },
};

const groupingRules: GroupingRule[] = [
  {
    id: "src",
    label: "Source",
    prefix: "src",
    mode: "prefix",
  },
];

function selectionKey(entry: SelectedFileEntry) {
  return `${entry.staged ? "1" : "0"}:${entry.path}`;
}

function baseViewProps() {
  return {
    files,
    fileStats,
    binarySet: new Set(files.binary),
    approvedPaths: new Set(["src/changed.ts"]),
    maxPathChars: 80,
    selectedFiles: [],
    selectedKeySet: new Set<string>(),
    selectionKey,
    onSelectFile: vi.fn(),
    onStageFile: vi.fn(),
    onUnstageFile: vi.fn(),
    isStaging: false,
    isDiscarding: false,
    isIgnoring: false,
    confirmingDiscard: null,
    onConfirmDiscard: vi.fn(),
    confirmingIgnore: null,
    onConfirmIgnore: vi.fn(),
    groupingRules,
    handleDiscardUnstaged: vi.fn(),
    handleDiscardUntracked: vi.fn(),
    handleIgnoreFile: vi.fn(),
    handleOpenMobileActions: vi.fn(),
    handleFileContextMenu: vi.fn(),
    mobileSelectionMode: false,
    handleLongPress: vi.fn(),
    handleMobileTap: vi.fn(),
    openFileMetrics: vi.fn(),
  };
}

describe("FileList view components", () => {
  it("routes flat-view section actions through the file-list seam", async () => {
    const user = userEvent.setup();
    const props = {
      ...baseViewProps(),
      openAggregateMetrics: vi.fn(),
    };

    renderWithQueryClient(<FileListFlatView {...props} />);

    expect(within(screenSection("conflicts")).getByText("src/conflict.ts")).toBeInTheDocument();
    expect(within(screenSection("staged")).getByText("src/staged.ts")).toBeInTheDocument();
    expect(within(screenSection("unstaged")).getByText("src/changed.ts")).toBeInTheDocument();

    await user.click(within(screenSection("conflicts")).getByLabelText("View change metrics"));
    expect(props.openAggregateMetrics).toHaveBeenCalledTimes(1);

    await user.click(within(screenSection("staged")).getByTestId("file-action-staged"));
    expect(props.onUnstageFile).toHaveBeenCalledWith("src/staged.ts");

    await user.click(within(screenSection("unstaged")).getByTestId("file-action-unstaged"));
    expect(props.onStageFile).toHaveBeenCalledWith("src/changed.ts");

    await user.click(within(screenSection("unstaged")).getByTestId("file-discard-unstaged"));
    expect(props.onConfirmDiscard).toHaveBeenCalledWith("src/changed.ts");
  });

  it("routes grouped-view aggregate review, stage, discard, and collapse actions", async () => {
    const user = userEvent.setup();
    const props = {
      ...baseViewProps(),
      groupedSections: [
        {
          id: "workspace",
          label: "workspace-sandbox",
          displayPrefix: "scenarios/workspace-sandbox",
          files: {
            conflicts: ["src/conflict.ts"],
            staged: ["src/staged.ts"],
            unstaged: ["src/changed.ts"],
            untracked: ["src/new.ts"],
          } satisfies Record<FileCategory, string[]>,
        },
      ],
      compactHeader: false,
      isMobile: false,
      collapsedGroups: new Set<string>(),
      confirmingGroup: null,
      onStagePaths: vi.fn(),
      onDiscardPaths: vi.fn(),
      onOpenReview: vi.fn(),
      onToggleGroupCollapse: vi.fn(),
      onSetConfirmingGroup: vi.fn(),
      openGroupMetrics: vi.fn(),
      openGroupCategoryMetrics: vi.fn(),
    };

    const { rerender } = renderWithQueryClient(<FileListGroupedView {...props} />);
    const group = screenGroup("workspace");

    await user.click(within(group).getByTitle("Open scenario review"));
    expect(props.onOpenReview).toHaveBeenCalledWith("workspace-sandbox");

    await user.click(within(group).getByRole("button", { name: "Stage All" }));
    expect(props.onStagePaths).toHaveBeenCalledWith([
      "src/changed.ts",
      "src/new.ts",
      "src/conflict.ts",
    ]);

    await user.click(within(group).getByRole("button", { name: "Discard All" }));
    expect(props.onSetConfirmingGroup).toHaveBeenCalledWith("workspace");

    rerender(<FileListGroupedView {...props} confirmingGroup="workspace" />);
    await user.click(within(screenGroup("workspace")).getByRole("button", { name: "Discard" }));
    expect(props.onDiscardPaths).toHaveBeenCalledWith(["src/changed.ts"], false);
    expect(props.onDiscardPaths).toHaveBeenCalledWith(["src/new.ts"], true);
    expect(props.onSetConfirmingGroup).toHaveBeenCalledWith(null);

    await user.click(within(screenGroup("workspace")).getByTestId("file-group-toggle-workspace"));
    expect(props.onToggleGroupCollapse).toHaveBeenCalledWith("workspace");
  });

  it("routes mobile action sheet commands for grouped and ungrouped files", async () => {
    const user = userEvent.setup();
    const groupedProps = {
      mobileActionFile: "src/changed.ts",
      mobileActionFileInfo: {
        path: "src/changed.ts",
        isStaged: false,
        isUnstaged: true,
        isUntracked: false,
        isConflict: false,
      },
      onClose: vi.fn(),
      onStageFile: vi.fn(),
      onUnstageFile: vi.fn(),
      onDiscardFile: vi.fn(),
      onIgnoreFile: vi.fn(),
      openFileMetrics: vi.fn(),
      groupingRules,
    };

    renderWithQueryClient(<FileListMobileActions {...groupedProps} />);

    await user.click(screen.getByRole("button", { name: /View Metrics/i }));
    expect(groupedProps.openFileMetrics).toHaveBeenCalledWith("src/changed.ts", "unstaged");
    expect(groupedProps.onClose).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: /^Stage/i }));
    expect(groupedProps.onStageFile).toHaveBeenCalledWith("src/changed.ts");

    await user.click(screen.getByRole("button", { name: /Ignore \(Source\)/i }));
    expect(groupedProps.onIgnoreFile).toHaveBeenCalledWith("src/changed.ts", "group", "src/");

    await user.click(screen.getByRole("button", { name: /Discard Changes/i }));
    expect(groupedProps.onDiscardFile).toHaveBeenCalledWith("src/changed.ts", false);
  });
});

function screenSection(category: FileCategory) {
  return screen.getByTestId(`file-section-${category}`);
}

function screenGroup(groupId: string) {
  return screen.getByTestId(`file-group-${groupId}`);
}
