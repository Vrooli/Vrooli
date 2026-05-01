import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { RepoFilesStatus } from "../lib/api";
import { renderWithQueryClient } from "../test-utils";
import { FileList } from "./FileList";
import type { FileListProps } from "./FileListTypes";

vi.mock("../hooks", () => ({
  useIsMobile: () => false,
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
