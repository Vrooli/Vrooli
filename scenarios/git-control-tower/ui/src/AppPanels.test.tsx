import { fireEvent, render, screen } from "@testing-library/react";
import type { MouseEvent } from "react";
import { describe, expect, it, vi } from "vitest";
import type { PanelDeps } from "./AppPanels";
import { renderPanel } from "./AppPanels";
import { renderMobilePanel } from "./AppMobilePanels";
import type {
  DiffResponse,
  RepoHistoryEntry,
  RepoHistoryResponse,
  RepoStatus,
  SaveFileContentResponse,
  SyncStatusResponse,
} from "./lib/api";
import { DEFAULT_STATE } from "./hooks/useScenarioReviewState";

interface FileListMockProps {
  onSelectFile: (path: string, staged: boolean, event: MouseEvent<HTMLButtonElement>) => void;
  onOpenReview: (slug: string) => void;
}

interface RelatedFilesPanelMockProps {
  onSelectFile: (path: string) => void;
}

interface HistoryFileListMockProps {
  onSelectFile: (path: string) => void;
}

interface GitHistoryMockProps {
  onSelectCommit: (entry: RepoHistoryEntry | null) => void;
}

interface DiffViewerMockProps {
  onOpenReview: () => void;
  onShowRelatedFiles?: (path: string) => void;
}

interface ScenarioReviewPanelMockProps {
  scenarioSlug: string;
  onChangeScenario: (slug: string) => void;
}

vi.mock("./components/FileList", () => ({
  FileList: ({ onSelectFile, onOpenReview }: FileListMockProps) => (
    <section data-testid="file-list">
      <button type="button" onClick={(event) => onSelectFile("src/app.ts", false, event)}>
        Select change
      </button>
      <button type="button" onClick={() => onOpenReview("workspace-sandbox")}>
        Open scenario review
      </button>
    </section>
  ),
}));

vi.mock("./components/HistoryFileList", () => ({
  HistoryFileList: ({ onSelectFile }: HistoryFileListMockProps) => (
    <button type="button" onClick={() => onSelectFile("src/history.ts")}>
      Select history file
    </button>
  ),
}));

vi.mock("./components/DiffViewer", () => ({
  DiffViewer: ({ onOpenReview, onShowRelatedFiles }: DiffViewerMockProps) => (
    <section data-testid="diff-viewer">
      <button type="button" onClick={onOpenReview}>
        Open review from diff
      </button>
      {onShowRelatedFiles ? (
        <button type="button" onClick={() => onShowRelatedFiles("src/app.ts")}>
          Show related files
        </button>
      ) : null}
    </section>
  ),
}));

vi.mock("./components/CommitPanel", () => ({
  CommitPanel: () => <section data-testid="commit-panel" />,
}));

vi.mock("./components/GitHistory", () => ({
  GitHistory: ({ onSelectCommit }: GitHistoryMockProps) => (
    <button
      type="button"
      onClick={() => onSelectCommit({ hash: "abc123", subject: "feat: test", files: ["src/app.ts"] })}
    >
      Select commit
    </button>
  ),
}));

vi.mock("./components/RelatedFilesPanel", () => ({
  RelatedFilesPanel: ({ onSelectFile }: RelatedFilesPanelMockProps) => (
    <button type="button" onClick={() => onSelectFile("src/related.ts")}>
      Select related file
    </button>
  ),
}));

vi.mock("./components/ScenarioReviewPanel", () => ({
  ScenarioReviewPanel: ({ scenarioSlug, onChangeScenario }: ScenarioReviewPanelMockProps) => (
    <section data-testid="scenario-review-panel">
      <span>{scenarioSlug}</span>
      <button type="button" onClick={() => onChangeScenario("workspace-sandbox")}>
        Change scenario
      </button>
    </section>
  ),
}));

function repoStatus(): RepoStatus {
  return {
    repo_dir: "/work/repo",
    branch: { head: "main" },
    files: {
      conflicts: [],
      staged: [],
      unstaged: ["src/app.ts"],
      untracked: [],
    },
    summary: {
      staged: 0,
      unstaged: 1,
      untracked: 0,
      conflicts: 0,
    },
    author: {},
    timestamp: "2026-05-01T00:00:00Z",
  };
}

function diffResponse(): DiffResponse {
  return {
    repo_dir: "/work/repo",
    path: "src/app.ts",
    staged: false,
    has_diff: true,
    stats: { additions: 1, deletions: 0, files: 1 },
    timestamp: "2026-05-01T00:00:00Z",
  };
}

function historyResponse(): RepoHistoryResponse {
  return {
    repo_dir: "/work/repo",
    lines: ["abc123 feat: test"],
    entries: [{ hash: "abc123", subject: "feat: test", files: ["src/app.ts"] }],
    limit: 50,
    timestamp: "2026-05-01T00:00:00Z",
  };
}

function syncStatus(): SyncStatusResponse {
  return {
    branch: "main",
    ahead: 1,
    behind: 0,
    has_upstream: true,
    can_push: true,
    can_pull: false,
    needs_push: true,
    needs_pull: false,
    has_uncommitted_changes: true,
    fetched: true,
    timestamp: "2026-05-01T00:00:00Z",
  };
}

function saveResponse(path: string): SaveFileContentResponse {
  return {
    success: true,
    path,
    content_hash: "hash",
    bytes_written: 10,
    timestamp: "2026-05-01T00:00:00Z",
  };
}

function baseDeps(overrides: Partial<PanelDeps> = {}): PanelDeps {
  const deps: PanelDeps = {
    repoId: "repo-1",
    statusData: repoStatus(),
    diffData: diffResponse(),
    diffIsLoading: false,
    diffError: null,
    historyData: historyResponse(),
    historyIsLoading: false,
    historyError: null,
    historyIsFetching: false,
    syncStatusData: syncStatus(),
    approvedChangesData: { available: false, committableFiles: 0 },
    selectedFile: "src/app.ts",
    selectedFiles: [],
    selectedKeySet: new Set<string>(),
    selectionKey: ({ path, staged }) => `${staged ? "1" : "0"}:${path}`,
    selectedIsStaged: false,
    selectedIsUntracked: false,
    approvedPendingSet: new Set<string>(),
    viewMode: "diff",
    viewingCommit: null,
    isViewingAnyFile: false,
    fileViewMode: "flat",
    groupingRules: [],
    groupingAvailable: false,
    changesCollapsed: false,
    historyCollapsed: false,
    commitCollapsed: false,
    showRelatedFiles: false,
    relatedFilesForPath: undefined,
    scrollToFile: undefined,
    viewingFileBlame: null,
    historySearch: "",
    historyScopeFilter: null,
    historyWorkingSetOnly: false,
    isHistoryFiltersOpen: false,
    historyHeight: 240,
    historyLimit: 50,
    historyMaxLimit: 200,
    primaryPanel: "diff",
    historyGrepPrefix: null,
    activeGroupFilter: null,
    workingSetPaths: [],
    commitMessage: "",
    commitError: undefined,
    canUseApprovedMessage: false,
    canAmend: false,
    amendDisabledReason: undefined,
    pushTargetRef: "origin/main",
    pushSourceBranch: "main",
    isStaging: false,
    isDiscarding: false,
    isIgnoring: false,
    isCommitting: false,
    isPushing: false,
    isUsingApprovedMessage: false,
    isDeleting: false,
    isSavingFile: false,
    confirmingDiscard: null,
    confirmingIgnore: null,
    mobileSelectionMode: false,
    reviewScenarioSlug: "git-control-tower",
    scenarioReview: {
      state: DEFAULT_STATE,
      update: vi.fn(),
      switchScenario: vi.fn(() => DEFAULT_STATE),
    },
    onSelectFile: vi.fn(),
    onStageFile: vi.fn(),
    onUnstageFile: vi.fn(),
    onDiscardFile: vi.fn(),
    onIgnoreFile: vi.fn(),
    onStageAll: vi.fn(),
    onUnstageAll: vi.fn(),
    onStagePaths: vi.fn(),
    onStageApproved: vi.fn(),
    onDiscardPaths: vi.fn(),
    onConfirmDiscard: vi.fn(),
    onConfirmIgnore: vi.fn(),
    onSelectHistoryFile: vi.fn(),
    onSelectCommit: vi.fn(),
    onContinueCommit: vi.fn(),
    onCommit: vi.fn(),
    onCommitMessageChange: vi.fn(),
    onUseApprovedMessage: vi.fn(),
    onPush: vi.fn(),
    onLoadMoreHistory: vi.fn(),
    onViewModeChange: vi.fn(),
    onShowRelatedFiles: vi.fn(),
    onBackFromRelatedFiles: vi.fn(),
    onScrollComplete: vi.fn(),
    onDeletePath: vi.fn(),
    onBlameFile: vi.fn(),
    onExitBlameMode: vi.fn(),
    onCycleViewMode: vi.fn(),
    onSetChangesCollapsed: vi.fn(),
    onSetHistoryCollapsed: vi.fn(),
    onSetCommitCollapsed: vi.fn(),
    onSetHistorySearch: vi.fn(),
    onSetHistoryScopeFilter: vi.fn(),
    onSetHistoryWorkingSetOnly: vi.fn(),
    onOpenHistoryFilters: vi.fn(),
    onCloseHistoryFilters: vi.fn(),
    onFilterGroup: vi.fn(),
    onClearGroupFilter: vi.fn(),
    onOpenFileSearch: vi.fn(),
    onSaveFileContent: vi.fn(async (path: string) => saveResponse(path)),
    onSelectRelatedFile: vi.fn(),
    onEnterSelectionMode: vi.fn(),
    onExitSelectionMode: vi.fn(),
    onMobileSelectFile: vi.fn(),
    onOpenReview: vi.fn(),
    onSetPrimaryPanel: vi.fn(),
    onSetMobileActivePanel: vi.fn(),
    onSetReviewScenarioSlug: vi.fn(),
  };

  return { ...deps, ...overrides };
}

describe("renderPanel", () => {
  it("returns from review to diff when a working-tree file is selected", () => {
    const deps = baseDeps({ primaryPanel: "review" });

    render(renderPanel(deps, "changes", "main"));
    fireEvent.click(screen.getByRole("button", { name: "Select change" }));

    expect(deps.onSelectFile).toHaveBeenCalledWith(
      "src/app.ts",
      false,
      expect.objectContaining({ type: "click" }),
    );
    expect(deps.onSetPrimaryPanel).toHaveBeenCalledWith("diff");
  });

  it("opens review from file list and preserves per-scenario review state", () => {
    const deps = baseDeps();

    render(renderPanel(deps, "changes", "main"));
    fireEvent.click(screen.getByRole("button", { name: "Open scenario review" }));

    expect(deps.scenarioReview.switchScenario).toHaveBeenCalledWith(
      "git-control-tower",
      "workspace-sandbox",
    );
    expect(deps.onSetReviewScenarioSlug).toHaveBeenCalledWith("workspace-sandbox");
    expect(deps.onSetPrimaryPanel).toHaveBeenCalledWith("review");
  });

  it("returns from review to diff when a history commit is selected", () => {
    const deps = baseDeps({ primaryPanel: "review" });

    render(renderPanel(deps, "history", "middle"));
    fireEvent.click(screen.getByRole("button", { name: "Select commit" }));

    expect(deps.onSelectCommit).toHaveBeenCalledWith({
      hash: "abc123",
      subject: "feat: test",
      files: ["src/app.ts"],
    });
    expect(deps.onSetPrimaryPanel).toHaveBeenCalledWith("diff");
  });
});

describe("renderMobilePanel", () => {
  it("opens selected files in the mobile diff panel", () => {
    const deps = baseDeps();

    render(renderMobilePanel(deps, "changes"));
    fireEvent.click(screen.getByRole("button", { name: "Select change" }));

    expect(deps.onSelectFile).toHaveBeenCalledWith(
      "src/app.ts",
      false,
      expect.objectContaining({ type: "click" }),
    );
    expect(deps.onSetMobileActivePanel).toHaveBeenCalledWith("diff");
  });

  it("routes related-file selection through the mobile changes panel", () => {
    const deps = baseDeps({
      showRelatedFiles: true,
      relatedFilesForPath: "src/app.ts",
    });

    render(renderMobilePanel(deps, "changes"));
    fireEvent.click(screen.getByRole("button", { name: "Select related file" }));

    expect(deps.onSelectRelatedFile).toHaveBeenCalledWith("src/related.ts");
    expect(deps.onSetMobileActivePanel).toHaveBeenCalledWith("diff");
  });

  it("opens the mobile review panel from diff actions", () => {
    const deps = baseDeps();

    render(renderMobilePanel(deps, "diff"));
    fireEvent.click(screen.getByRole("button", { name: "Open review from diff" }));

    expect(deps.onSetMobileActivePanel).toHaveBeenCalledWith("review");
  });
});
