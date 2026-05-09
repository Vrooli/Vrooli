import { FileList } from "./components/FileList";
import { HistoryFileList } from "./components/HistoryFileList";
import { DiffViewer } from "./components/DiffViewer";
import { CommitPanel } from "./components/CommitPanel";
import { GitHistory } from "./components/GitHistory";
import { RelatedFilesPanel } from "./components/RelatedFilesPanel";
import { ScenarioReviewPanel } from "./components/ScenarioReviewPanel";
import type { LayoutSection } from "./components/LayoutSettingsModal";
import type { PanelDeps } from "./AppPanels";

/** Mobile variant of renderPanel with mobile-specific behaviors (auto-navigate, collapsed=false, etc.) */
export function renderMobilePanel(
  deps: PanelDeps,
  panel: LayoutSection,
) {
  const isHistoryMode = Boolean(deps.viewingCommit);

  switch (panel) {
    case "changes":
      if (deps.showRelatedFiles && deps.relatedFilesForPath) {
        return (
          <RelatedFilesPanel
            forPath={deps.relatedFilesForPath}
            onBack={deps.onBackFromRelatedFiles}
            onSelectFile={(path) => {
              deps.onSelectRelatedFile(path);
              deps.onSetMobileActivePanel("diff");
            }}
            repoId={deps.repoId}
          />
        );
      }
      if (isHistoryMode && deps.viewingCommit) {
        return (
          <HistoryFileList
            viewingCommit={deps.viewingCommit}
            selectedFile={deps.selectedFile}
            onSelectFile={(path) => {
              deps.onSelectHistoryFile(path);
              deps.onSetMobileActivePanel("diff");
            }}
            collapsed={false}
            fillHeight={true}
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
            deps.onSetMobileActivePanel("diff");
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
          collapsed={false}
          fillHeight={true}
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
            deps.onSetReviewScenarioSlug(slug);
            deps.onSetMobileActivePanel("review");
          }}
          mobileSelectionMode={deps.mobileSelectionMode}
          onEnterSelectionMode={deps.onEnterSelectionMode}
          onExitSelectionMode={deps.onExitSelectionMode}
          onMobileSelectFile={deps.onMobileSelectFile}
          fileHotspots={deps.statusData?.file_hotspots}
        />
      );

    case "diff":
      return (
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
          onStage={deps.onStageFile}
          onUnstage={deps.onUnstageFile}
          onDiscard={deps.onDiscardFile}
          isStaging={deps.isStaging}
          isDiscarding={deps.isDiscarding}
          isHistoryMode={isHistoryMode}
          commitHash={deps.viewingCommit?.hash}
          onShowRelatedFiles={(path) => {
            deps.onShowRelatedFiles(path);
            deps.onSetMobileActivePanel("changes");
          }}
          onOpenSearch={deps.onOpenFileSearch}
          onOpenReview={() => deps.onSetMobileActivePanel("review")}
          isReadOnly={deps.isViewingAnyFile}
          onSaveFileContent={deps.onSaveFileContent}
          isSavingFile={deps.isSavingFile}
          onDeletePath={deps.onDeletePath}
          isDeleting={deps.isDeleting}
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
          collapsed={false}
          fillHeight={true}
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

    case "history":
      return (
        <GitHistory
          lines={deps.historyData?.lines}
          entries={deps.historyData?.entries}
          isLoading={deps.historyIsLoading}
          error={deps.historyError}
          collapsed={false}
          fillHeight={true}
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
            if (entry) {
              deps.onSetMobileActivePanel("changes");
            }
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
          isMobile
        />
      );
  }
}
