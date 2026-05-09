import { FileList } from "./components/FileList";
import { HistoryFileList } from "./components/HistoryFileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import { GitHistory } from "./components/GitHistory";
import { RelatedFilesPanel } from "./components/RelatedFilesPanel";
import { ScenarioReviewPanel } from "./components/ScenarioReviewPanel";
import type { LayoutSection } from "./components/LayoutSettingsModal";
import type { GroupingRule } from "./components/FileList";
import type { ViewingCommit } from "./components/HistoryModeHeader";
import type { ViewMode, FileViewMode, RepoStatus, RepoHistoryResponse, RepoHistoryEntry, DiffResponse, SyncStatusResponse, SaveFileContentResponse } from "./lib/api";
import type { SelectionEntry, ViewingFileBlame } from "./App.types";
import type { ScenarioReviewState, DeepPartial } from "./hooks/useScenarioReviewState";

export interface PanelDeps {
  // Repo
  repoId: string | null;

  // Queries
  statusData: RepoStatus | undefined;
  diffData: DiffResponse | undefined;
  diffIsLoading: boolean;
  diffError: Error | null;
  historyData: RepoHistoryResponse | undefined;
  historyIsLoading: boolean;
  historyError: Error | null;
  historyIsFetching: boolean;
  syncStatusData: SyncStatusResponse | undefined;
  approvedChangesData: { available: boolean; committableFiles: number; suggestedMessage?: string; warning?: string; files?: Array<{ status: string; relativePath: string }> } | undefined;

  // Selection state
  selectedFile: string | undefined;
  selectedFiles: SelectionEntry[];
  selectedKeySet: Set<string>;
  selectionKey: (entry: { path: string; staged: boolean }) => string;
  selectedIsStaged: boolean;
  selectedIsUntracked: boolean;
  approvedPendingSet: Set<string>;

  // View state
  viewMode: ViewMode;
  viewingCommit: ViewingCommit | null;
  isViewingAnyFile: boolean;
  fileViewMode: FileViewMode;
  groupingRules: GroupingRule[];
  groupingAvailable: boolean;
  changesCollapsed: boolean;
  historyCollapsed: boolean;
  commitCollapsed: boolean;
  showRelatedFiles: boolean;
  relatedFilesForPath: string | undefined;
  scrollToFile: string | undefined;
  viewingFileBlame: ViewingFileBlame | null;

  // History filters
  historySearch: string;
  historyScopeFilter: string | null;
  historyWorkingSetOnly: boolean;
  isHistoryFiltersOpen: boolean;
  historyHeight: number;
  historyLimit: number;
  historyMaxLimit: number;
  primaryPanel: LayoutSection;
  historyGrepPrefix: string | null;
  activeGroupFilter: { prefix: string; count: number } | null;
  workingSetPaths: string[];

  // Commit state
  commitMessage: string;
  commitError: string | undefined;
  canUseApprovedMessage: boolean;
  canAmend: boolean;
  amendDisabledReason: string | undefined;
  pushTargetRef: string | undefined;
  pushSourceBranch: string | undefined;

  // Mutation pending flags
  isStaging: boolean;
  isDiscarding: boolean;
  isIgnoring: boolean;
  isCommitting: boolean;
  isPushing: boolean;
  isUsingApprovedMessage: boolean;
  isDeleting: boolean;
  isSavingFile: boolean;
  confirmingDiscard: string | null;
  confirmingIgnore: string | null;

  // Mobile
  mobileSelectionMode: boolean;

  // Review
  reviewScenarioSlug: string;
  scenarioReview: {
    state: ScenarioReviewState;
    update: (partial: DeepPartial<ScenarioReviewState>) => void;
    switchScenario: (from: string, to: string) => ScenarioReviewState;
  };

  // Handlers
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  onDiscardFile: (path: string, untracked: boolean) => void;
  onIgnoreFile: (path: string, level?: "project" | "group", groupDir?: string) => void;
  onStageAll: () => void;
  onUnstageAll: () => void;
  onStagePaths: (paths: string[]) => void;
  onStageApproved: () => void;
  onDiscardPaths: (paths: string[], untracked: boolean) => void;
  onConfirmDiscard: (path: string | null) => void;
  onConfirmIgnore: (path: string | null) => void;
  onSelectHistoryFile: (path: string) => void;
  onSelectCommit: (entry: RepoHistoryEntry | null) => void;
  onContinueCommit: (message: string) => void;
  onCommit: (message: string, options: { conventional: boolean; amend: boolean; authorName?: string; authorEmail?: string }) => void;
  onCommitMessageChange: (msg: string) => void;
  onUseApprovedMessage: () => void;
  onPush: () => void;
  onLoadMoreHistory: () => void;
  onViewModeChange: (mode: ViewMode) => void;
  onShowRelatedFiles: (path: string) => void;
  onBackFromRelatedFiles: () => void;
  onScrollComplete: () => void;
  onDeletePath: (path: string, isDir: boolean) => void;
  onBlameFile: (path: string) => void;
  onExitBlameMode: () => void;
  onCycleViewMode: () => void;
  onSetChangesCollapsed: (fn: (prev: boolean) => boolean) => void;
  onSetHistoryCollapsed: (fn: (prev: boolean) => boolean) => void;
  onSetCommitCollapsed: (fn: (prev: boolean) => boolean) => void;
  onSetHistorySearch: (q: string) => void;
  onSetHistoryScopeFilter: (f: string | null) => void;
  onSetHistoryWorkingSetOnly: (v: boolean) => void;
  onOpenHistoryFilters: () => void;
  onCloseHistoryFilters: () => void;
  onFilterGroup: (prefix: string | null) => void;
  onClearGroupFilter: () => void;
  onOpenFileSearch: () => void;
  onSaveFileContent: (path: string, content: string, expectedHash?: string) => Promise<SaveFileContentResponse>;
  onSelectRelatedFile: (path: string) => void;
  onEnterSelectionMode: (path: string, staged: boolean) => void;
  onExitSelectionMode: () => void;
  onMobileSelectFile: (path: string, staged: boolean, mode: "toggle" | "range") => void;

  // Layout navigation (panel switching on select)
  onOpenReview: (slug?: string) => void;
  onSetPrimaryPanel: (panel: LayoutSection) => void;
  onSetMobileActivePanel: (panel: LayoutSection) => void;
  onSetReviewScenarioSlug: (slug: string) => void;
}

export function renderPanel(
  deps: PanelDeps,
  panel: LayoutSection,
  slot: "top" | "middle" | "bottom" | "main",
) {
  const isMain = slot === "main";
  const isHistoryMode = Boolean(deps.viewingCommit);

  switch (panel) {
    case "changes":
      if (deps.showRelatedFiles && deps.relatedFilesForPath) {
        return (
          <RelatedFilesPanel
            forPath={deps.relatedFilesForPath}
            onBack={deps.onBackFromRelatedFiles}
            onSelectFile={deps.onSelectRelatedFile}
            repoId={deps.repoId}
          />
        );
      }
      if (isHistoryMode && deps.viewingCommit) {
        return (
          <HistoryFileList
            viewingCommit={deps.viewingCommit}
            selectedFile={deps.selectedFile}
            onSelectFile={deps.onSelectHistoryFile}
            collapsed={deps.changesCollapsed}
            onToggleCollapse={() => deps.onSetChangesCollapsed((prev) => !prev)}
            fillHeight={isMain || !deps.changesCollapsed}
            onDeletePath={deps.onDeletePath}
          />
        );
      }
      return (
        <FileList
          files={deps.statusData?.files}
          fileStats={deps.statusData?.file_stats}
          selectedFiles={deps.selectedFiles}
          selectedKeySet={deps.selectedKeySet}
          selectionKey={deps.selectionKey}
          approvedChanges={
            deps.approvedChangesData
              ? {
                  available: deps.approvedChangesData.available,
                  committableFiles: deps.approvedChangesData.committableFiles,
                  warning: deps.approvedChangesData.warning,
                }
              : undefined
          }
          approvedPaths={deps.approvedPendingSet}
          onStageApproved={deps.onStageApproved}
          isStagingApproved={deps.isStaging}
          onSelectFile={(path, staged, event) => {
            deps.onSelectFile(path, staged, event);
            if (deps.primaryPanel === "review") deps.onSetPrimaryPanel("diff");
          }}
          onStageFile={deps.onStageFile}
          onUnstageFile={deps.onUnstageFile}
          onDiscardFile={deps.onDiscardFile}
          onIgnoreFile={deps.onIgnoreFile}
          onStageAll={deps.onStageAll}
          onUnstageAll={deps.onUnstageAll}
          isStaging={deps.isStaging}
          isDiscarding={deps.isDiscarding}
          isIgnoring={deps.isIgnoring}
          confirmingDiscard={deps.confirmingDiscard}
          onConfirmDiscard={deps.onConfirmDiscard}
          confirmingIgnore={deps.confirmingIgnore}
          onConfirmIgnore={deps.onConfirmIgnore}
          collapsed={deps.changesCollapsed}
          onToggleCollapse={() => deps.onSetChangesCollapsed((prev) => !prev)}
          fillHeight={isMain || !deps.changesCollapsed}
          fileViewMode={deps.fileViewMode}
          groupingRules={deps.groupingRules}
          groupingAvailable={deps.groupingAvailable}
          onCycleViewMode={deps.onCycleViewMode}
          onStagePaths={deps.onStagePaths}
          onDiscardPaths={deps.onDiscardPaths}
          scrollToFile={deps.scrollToFile}
          onScrollComplete={deps.onScrollComplete}
          onDeletePath={deps.onDeletePath}
          onBlameFile={deps.onBlameFile}
          repoId={deps.repoId}
          onOpenReview={(slug) => {
            if (deps.reviewScenarioSlug && slug !== deps.reviewScenarioSlug) {
              deps.scenarioReview.switchScenario(deps.reviewScenarioSlug, slug);
            }
            deps.onSetReviewScenarioSlug(slug);
            deps.onSetPrimaryPanel("review");
          }}
          mobileSelectionMode={deps.mobileSelectionMode}
          onEnterSelectionMode={deps.onEnterSelectionMode}
          onExitSelectionMode={deps.onExitSelectionMode}
          onMobileSelectFile={deps.onMobileSelectFile}
          fileHotspots={deps.statusData?.file_hotspots}
        />
      );

    case "history":
      return (
        <GitHistory
          lines={deps.historyData?.lines}
          entries={deps.historyData?.entries}
          isLoading={deps.historyIsLoading}
          error={deps.historyError}
          collapsed={deps.historyCollapsed}
          onToggleCollapse={() => deps.onSetHistoryCollapsed((prev) => !prev)}
          height={slot === "middle" ? deps.historyHeight : undefined}
          fillHeight={isMain}
          onLoadMore={deps.onLoadMoreHistory}
          isFetching={deps.historyIsFetching}
          hasMore={
            !deps.historyGrepPrefix &&
            (deps.historyData?.lines?.length ?? 0) >= deps.historyLimit &&
            deps.historyLimit < deps.historyMaxLimit
          }
          searchQuery={deps.historySearch}
          onSearchQueryChange={deps.onSetHistorySearch}
          scopeFilter={deps.historyScopeFilter}
          onScopeFilterChange={deps.onSetHistoryScopeFilter}
          groupingEnabled={deps.fileViewMode === "grouped"}
          groupingRules={deps.groupingRules}
          workingSetPaths={deps.workingSetPaths}
          workingSetOnly={deps.historyWorkingSetOnly}
          onWorkingSetOnlyChange={deps.onSetHistoryWorkingSetOnly}
          filtersOpen={deps.isHistoryFiltersOpen}
          onOpenFilters={deps.onOpenHistoryFilters}
          onCloseFilters={deps.onCloseHistoryFilters}
          selectedCommitHash={deps.viewingCommit?.hash}
          onSelectCommit={(entry) => {
            deps.onSelectCommit(entry);
            if (entry && deps.primaryPanel === "review") deps.onSetPrimaryPanel("diff");
          }}
          blameFilePath={deps.viewingFileBlame?.path}
          blameFileName={deps.viewingFileBlame?.filename}
          onExitBlameMode={deps.onExitBlameMode}
          onContinueCommit={deps.onContinueCommit}
          activeGroupFilter={deps.activeGroupFilter}
          onFilterGroup={deps.onFilterGroup}
          onClearGroupFilter={deps.onClearGroupFilter}
        />
      );

    case "commit":
      return (
        <CommitPanel
          stagedCount={deps.statusData?.summary.staged ?? 0}
          commitMessage={deps.commitMessage}
          onCommitMessageChange={deps.onCommitMessageChange}
          canUseApprovedMessage={deps.canUseApprovedMessage}
          onUseApprovedMessage={deps.onUseApprovedMessage}
          isUsingApprovedMessage={deps.isUsingApprovedMessage}
          onCommit={deps.onCommit}
          isCommitting={deps.isCommitting}
          commitError={deps.commitError}
          defaultAuthorName={deps.statusData?.author?.name}
          defaultAuthorEmail={deps.statusData?.author?.email}
          canAmend={deps.canAmend}
          amendDisabledReason={deps.amendDisabledReason}
          collapsed={deps.commitCollapsed}
          onToggleCollapse={() => deps.onSetCommitCollapsed((prev) => !prev)}
          fillHeight={isMain || !deps.commitCollapsed}
          onPush={deps.onPush}
          isPushing={deps.isPushing}
          canPush={deps.syncStatusData?.can_push ?? false}
          aheadCount={deps.syncStatusData?.ahead ?? 0}
          pushTarget={deps.pushTargetRef}
          sourceBranch={deps.pushSourceBranch}
          isHistoryMode={isHistoryMode}
          historyCommit={deps.viewingCommit}
        />
      );

    case "review":
      return (
        <ScenarioReviewPanel
          scenarioSlug={deps.reviewScenarioSlug}
          repoId={deps.repoId}
          fileStats={deps.statusData?.file_stats}
          onChangeScenario={(slug) => {
            if (deps.reviewScenarioSlug && slug !== deps.reviewScenarioSlug) {
              deps.scenarioReview.switchScenario(deps.reviewScenarioSlug, slug);
            }
            deps.onSetReviewScenarioSlug(slug);
          }}
          activeTab={deps.scenarioReview.state.activeTab}
          onActiveTabChange={(tab) => deps.scenarioReview.update({ activeTab: tab })}
          agentRunId={deps.scenarioReview.state.agentRunId}
          onAgentRunIdChange={(id) => deps.scenarioReview.update({ agentRunId: id })}
          scenarioState={deps.scenarioReview.state}
          onScenarioStateChange={deps.scenarioReview.update}
        />
      );

    case "diff":
    default:
      return (
        <div className="h-full min-h-0">
          <DiffViewer
            diff={deps.diffData}
            selectedFile={deps.selectedFile}
            isStaged={deps.selectedIsStaged}
            isUntracked={deps.selectedIsUntracked}
            isLoading={deps.diffIsLoading}
            error={deps.diffError}
            repoDir={deps.statusData?.repo_dir}
            viewMode={deps.viewMode}
            onViewModeChange={deps.onViewModeChange}
            isHistoryMode={isHistoryMode}
            commitHash={deps.viewingCommit?.hash}
            onShowRelatedFiles={deps.onShowRelatedFiles}
            onOpenSearch={deps.onOpenFileSearch}
            onOpenReview={() => deps.onSetPrimaryPanel("review")}
            isReadOnly={deps.isViewingAnyFile}
            onSaveFileContent={deps.onSaveFileContent}
            isSavingFile={deps.isSavingFile}
            onDeletePath={deps.onDeletePath}
            isDeleting={deps.isDeleting}
          />
        </div>
      );
  }
}
