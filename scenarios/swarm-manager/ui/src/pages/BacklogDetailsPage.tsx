/**
 * Backlog Details Page — layout shell and tab routing.
 * Data fetching: useBacklogDetailData, UI handlers: useBacklogHandlers.
 * UI state: backlog-detail-ui-store, computed values: BacklogDetailContext.
 * [REQ:REQ-P0-004]
 */

import { useState, useMemo, useRef, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { Activity, CircleHelp, ClipboardList, Files, Sparkles } from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { PlanPanel } from "../components/backlog/plan-panel";
import { useUrlState } from "../hooks/use-url-state";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { BacklogFileWorkspace } from "../components/backlog/backlog-file-workspace";
import { BacklogDetailsPanel } from "../components/backlog/backlog-details-panel";
import { BacklogNotesPanel } from "../components/backlog/backlog-notes-panel";
import { BacklogActionButtons } from "../components/backlog/backlog-action-buttons";
import { OutputTab } from "../components/backlog/output-tab";
import { ActivityTab } from "../components/backlog/activity-tab";
import { BacklogScenariosPanel } from "../components/backlog/backlog-scenarios-panel";
import { BacklogDesktopHeader } from "../components/backlog/backlog-desktop-header";
import { BacklogDialogs } from "../components/backlog/backlog-dialogs";
import { HeaderPrimaryAction } from "../components/backlog/header-primary-action";
import { OperationalTargetsPanel } from "../components/backlog/operational-targets-panel";
import { BulkActionToolbar } from "../components/backlog/bulk-action-toolbar";
import { useActivityTimeline } from "../hooks/useActivityTimeline";
import { useBacklogDetailData } from "../hooks/useBacklogDetailData";
import { useEmbeddedServiceUrl } from "../hooks/useEmbeddedServiceUrl";
import { useStorePolling } from "../hooks/useStorePolling";
import { useBacklogHandlers } from "../hooks/useBacklogHandlers";
import { findBacklogFileByPath } from "../lib/workshop-files";
import { selectors } from "../consts/selectors";
import { BACKLOG_KIND_ICONS, BACKLOG_KIND_LABELS, BACKLOG_KINDS } from "../types";
import type { BacklogFile, BacklogKind } from "../types";
import {
  selectLatestActivityForBacklog,
  useAgentActivitiesStore,
  useBacklogDetailUIStore,
  useBacklogStore,
  useDetailSelectionStore,
} from "../stores";
import { BACKLOG_LENSES } from "../components/detail/lens-options";
import { backlogService } from "../services/backlog-service";
import { reviewService } from "../services/review-service";
import { useReviewStore } from "../stores/review-store";
import { EvidenceRequestPanel } from "../components/backlog/evidence-request-panel";
import { useQueryClient } from "@tanstack/react-query";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { useDetailNavigation } from "../hooks/useDetailNavigation";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { BacklogDetailProvider } from "../contexts/BacklogDetailContext";
import { FileServiceProvider } from "../contexts/FileServiceContext";
import { createBacklogFileServiceAdapter } from "../services/backlog/backlog-file-service-adapter";

const DEFAULT_PREVIEW_FILE_PATH = "spec.json";
const AGENT_RUN_REFRESH_MS = 6000;
type DetailsTab = "info" | "prompt" | "files" | "output" | "activity";

export function BacklogDetailsPage() {
  // --- Navigation / selection ---
  const selection = useDetailSelectionStore((s) => s.selection);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);


  const nodeId = selectionToNodeId(selection);
  const { closeDetail } = useDetailNavigation();
  const kind = selection?.kind;
  const name = selection?.name;
  const backlogKind = BACKLOG_KINDS.includes(kind as BacklogKind) ? (kind as BacklogKind) : null;
  const [searchParams, setSearchParams] = useSearchParams();

  // --- Global stores ---
  const upsertItem = useBacklogStore((s) => s.upsertItem);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const stopRun = useAgentActivitiesStore((s) => s.stopRun);
  const latestAgentActivity = useAgentActivitiesStore((s) => {
    if (!backlogKind || !name) return null;
    return selectLatestActivityForBacklog(s, backlogKind, name);
  });

  useStorePolling({
    enabled: true,
    intervalMs: AGENT_RUN_REFRESH_MS,
    pollFn: () => void refreshActivities(true),
    immediate: true,
  });

  const agentRunIsActive = latestAgentActivity
    ? ["pending", "starting", "running", "needs_review"].includes(latestAgentActivity.status)
    : false;

  // --- UI state store ---
  const uiStore = useBacklogDetailUIStore();
  const queryClient = useQueryClient();

  // --- Data hook ---
  const data = useBacklogDetailData({ backlogKind, name, agentRunIsActive });
  const {
    item, isLoadingItem, itemError, refetchItem, spawnedItems,
    files, isLoadingFiles, filesError, refetchFiles,
    executionHistory, reviewRounds, isGatheringEvidence,
    workshopRounds, readinessData, archiveTargets,
    depRelations, itemActions, targetScenarios,
    deliverableLabel, workshopActionLabel,
    isWorkshopFinalized, isLocked, isTerminal, workshopBlockedDeps,
    deleteError, isUpdating, isRunningAgent, isSavingWorkshop,
    isBatchReviewing, isDeletingWorkshopRound, isFileActionPending,
  } = data;

  // --- Local UI state (URL-synced or needs render) ---
  const [activeTab, setActiveTab] = useUrlState<DetailsTab>("tab", "info", {
    validate: (v): v is DetailsTab => ["info", "prompt", "files", "output", "activity"].includes(v),
  });
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);
  const { url: agentManagerUiUrl } = useEmbeddedServiceUrl("agent-manager");

  // --- Handlers hook ---
  const handlers = useBacklogHandlers({
    data,
    backlogKind,
    name,
    setSelectedFile,
    setActiveTab,
    setSearchParams,
    selectedFile,
    closeDetail,
    refreshActivities,
  });

  // --- Computed labels ---
  const agentRunningLabel = useMemo(() => {
    if (!agentRunIsActive || !latestAgentActivity) return "Agent running\u2026";
    switch (latestAgentActivity.purpose) {
      case "workshop": return "Running workshop\u2026";
      case "finalize": return "Running finalize\u2026";
      case "research": return "Running research\u2026";
      case "initialize": return "Initializing workshop\u2026";
      case "process": return "Processing\u2026";
      default: return "Agent running\u2026";
    }
  }, [agentRunIsActive, latestAgentActivity]);

  const agentLabel = item?.kind === "idea" ? "Idea Agent" : "Workshop";

  // --- Effects ---

  // Activity timeline — fetches when the Output tab is active
  const timeline = useActivityTimeline({
    backlogKind: backlogKind ?? undefined,
    backlogName: name,
    enabled: activeTab === "output" || activeTab === "activity",
    agentRunIsActive,
  });

  // Auto-open follow-up dialog when navigated with ?action=followup
  const actionParam = searchParams.get("action");
  useEffect(() => {
    if (actionParam === "followup" && executionHistory && executionHistory.length > 0 && !uiStore.followUpTarget) {
      uiStore.setFollowUpTarget(executionHistory[0] ?? null);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.delete("action");
        return next;
      }, { replace: true });
    }
  }, [actionParam, executionHistory, uiStore.followUpTarget, setSearchParams, uiStore]);

  // Merged flagged items for the agent dialog
  const { agentDialogTargetIds: getAgentDialogTargetIds, agentDialogRequirementIds: getAgentDialogRequirementIds } = data;
  const agentDialogTargetIds = useMemo(
    () => getAgentDialogTargetIds(uiStore.selectedTargetIds),
    [getAgentDialogTargetIds, uiStore.selectedTargetIds],
  );
  const agentDialogRequirementIds = useMemo(
    () => getAgentDialogRequirementIds(uiStore.selectedRequirementIds),
    [getAgentDialogRequirementIds, uiStore.selectedRequirementIds],
  );

  // Sync selected file from URL param / file tree
  const selectedFileParam = searchParams.get("file");
  useEffect(() => {
    if (!files || files.length === 0) return;
    const requestedPath = selectedFileParam || DEFAULT_PREVIEW_FILE_PATH;
    const resolvedFile = findBacklogFileByPath(files, requestedPath);
    if (resolvedFile) {
      setSelectedFile((prev) => (prev?.path === resolvedFile.path ? prev : resolvedFile));
      if (!selectedFileParam) {
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          next.set("file", resolvedFile.path);
          return next;
        }, { replace: true });
      }
      return;
    }
    if (selectedFileParam) {
      const fallbackFile = findBacklogFileByPath(files, DEFAULT_PREVIEW_FILE_PATH);
      if (fallbackFile) {
        setSelectedFile((prev) => (prev?.path === fallbackFile.path ? prev : fallbackFile));
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          next.set("file", fallbackFile.path);
          return next;
        }, { replace: true });
        return;
      }
    }
    setSelectedFile(null);
  }, [files, selectedFileParam, setSearchParams]);

  // Reset UI state when navigating between backlog items
  const prevItemRef = useRef(`${backlogKind}/${name}`);
  useEffect(() => {
    const key = `${backlogKind}/${name}`;
    if (key !== prevItemRef.current) {
      prevItemRef.current = key;
      setActiveTab("info");
      uiStore.reset();
    }
  }, [backlogKind, name, setActiveTab, uiStore]);

  const fileService = useMemo(
    () => backlogKind && name ? createBacklogFileServiceAdapter(backlogKind as BacklogKind, name) : null,
    [backlogKind, name],
  );

  // --- Early returns ---
  if (!backlogKind || !name) {
    return (
      <div className="space-y-6" data-testid={selectors.backlogDetails.page}>
        <ErrorState error={new Error("Backlog kind and name are required")} title="Invalid URL" />
      </div>
    );
  }

  const isPageLoading = isLoadingItem && !item;
  const pageError = itemError;

  // --- Context value ---
  const contextValue = {
    backlogKind: backlogKind as BacklogKind,
    name: name ?? "",
    item,
    itemActions,
    isLocked,
    isTerminal,
    agentRunIsActive,
    latestAgentActivity,
    deliverableLabel,
    workshopActionLabel,
    agentRunningLabel,
    agentLabel,
    isWorkshopFinalized,
    workshopBlockedDeps,
    isRunningAgent,
  };

  // --- Shared element variables ---
  const operationalTargetsSection = archiveTargets?.has_archive ? (
    <>
      <OperationalTargetsPanel
        targets={archiveTargets.targets}
        requirements={archiveTargets.requirements}
        editable
        onCreateRequirement={handlers.handleCreateRequirement}
        onEditRequirement={handlers.handleEditRequirement}
        onDeleteRequirement={handlers.handleDeleteRequirement}
        onReorderRequirement={handlers.handleReorderRequirement}
        onCreateModule={handlers.handleCreateModule}
        onEditModule={handlers.handleEditModule}
        onDeleteModule={handlers.handleDeleteModule}
        onCreateTarget={handlers.handleCreateTarget}
        onEditTarget={handlers.handleEditTarget}
        onDeleteTarget={handlers.handleDeleteTarget}
        onReviewAction={handlers.handleReviewAction}
        reviewSaving={isBatchReviewing}
        reviewError={data.batchReviewError}
      />
      <BulkActionToolbar
        selectedCount={uiStore.selectedTargetIds.size + uiStore.selectedRequirementIds.size}
        onApproveSelected={handlers.handleBulkApprove}
        onFlagSelected={handlers.handleBulkFlag}
        onSendToAgent={uiStore.openAgent}
        onClearSelection={uiStore.clearSelections}
      />
    </>
  ) : null;

  const detailsPanel = item ? (
    <BacklogDetailsPanel
      item={item}
      depRelations={depRelations}
      spawnedItems={spawnedItems}
      isLocked={isLocked}
      onEditGlobs={uiStore.openGlob}
      onDepStatusChange={(dep, newStatus) =>
        data.updateDepStatus({ kind: dep.kind, depName: dep.name, newStatus })
      }
      onSaveNote={async (note) => {
        await backlogService.update(item.kind, item.name, { note });
        data.refetchItem();
      }}
    />
  ) : null;

  const notesPanel = (
    <BacklogNotesPanel
      readinessData={readinessData}
      workshopRounds={workshopRounds}
      isSavingWorkshop={isSavingWorkshop}
      isDeletingRound={isDeletingWorkshopRound}
      onStartRun={uiStore.openRunModal}
      onSaveRound={handlers.handleSaveRound}
      onRunWorkshop={handlers.handleRunWorkshop}
      onFinalizeWorkshop={handlers.handleFinalizeWorkshop}
      onInitializeWorkshop={() => handlers.handleAgentSubmit({
        mode: "initialize",
        prompt: "Initialize the first workshop round for this backlog item.",
      })}
      onDeleteRound={uiStore.setRoundToDelete}
    />
  );

  const mobileActionButtons = item && itemActions ? (
    <BacklogActionButtons
      item={item}
      isUpdating={isUpdating}
      onFinalizeWorkshop={handlers.handleFinalizeWorkshop}
      onStartRun={uiStore.openRunModal}
      onRunWorkshop={handlers.handleRunWorkshop}
      onEdit={uiStore.openEdit}
      onFollowUp={() => uiStore.setFollowUpTarget(executionHistory?.[0] ?? null)}
      onOpenAgentDialog={uiStore.openAgent}
      onArchive={() => handlers.handleUpdateItem({
        title: item.title, description: item.description,
        status: "archived", priority: item.priority, tags: item.tags,
      })}
      onStatusChange={(newStatus) => handlers.handleUpdateItem({
        title: item.title, description: item.description,
        status: newStatus, priority: item.priority, tags: item.tags,
      })}
      onResetWorkshop={uiStore.openWorkshopReset}
      hasWorkshopRounds={(workshopRounds?.length ?? 0) > 0}
      onDelete={uiStore.openDelete}
    />
  ) : undefined;

  const fileWorkspaceElement = fileService ? (
    <FileServiceProvider value={fileService}>
      <BacklogFileWorkspace
        files={files}
        isLoadingFiles={isLoadingFiles}
        filesError={filesError}
        selectedFile={selectedFile}
        isLocked={isLocked}
        onFileSelect={handlers.handleFileSelect}
        onRefetchFiles={refetchFiles}
        onUploadComplete={handlers.handleUploadComplete}
        fileActionPending={isFileActionPending}
        onFileAction={handlers.handleFileAction}
      />
    </FileServiceProvider>
  ) : null;

  const tabBar = item ? (
    <div className="border-t border-slate-800/50" data-testid={selectors.backlogDetails.tabRow}>
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as DetailsTab)}>
        <TabsList className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3">
          <TabsTrigger value="info" className="gap-2" data-testid={selectors.backlogDetails.tabInfo}>
            <CircleHelp className="h-4 w-4" />
            Info
          </TabsTrigger>
          <TabsTrigger value="prompt" className="gap-2" data-testid={selectors.backlogDetails.tabPrompt}>
            <Sparkles className="h-4 w-4" />
            {backlogKind === "research" ? "Conclusion" : "Plan"}
          </TabsTrigger>
          <TabsTrigger value="files" className="gap-2" data-testid={selectors.backlogDetails.tabFiles}>
            <Files className="h-4 w-4" />
            Files
          </TabsTrigger>
          <TabsTrigger value="output" className="gap-2" data-testid={selectors.backlogDetails.tabOutput}>
            <Activity className="h-4 w-4" />
            Output
          </TabsTrigger>
          <TabsTrigger value="activity" className="gap-2" data-testid={selectors.backlogDetails.tabActivity}>
            <ClipboardList className="h-4 w-4" />
            Activity
            {agentRunIsActive && (
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
              </span>
            )}
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  ) : null;

  return (
    <BacklogDetailProvider value={contextValue}>
      <DetailPageLayout
        header={
          <DetailPageHeader
            entityType={backlogKind ? BACKLOG_KIND_LABELS[backlogKind] : "backlog"}
            entityIcon={backlogKind ? BACKLOG_KIND_ICONS[backlogKind] : undefined}
            title={item?.title ?? name ?? "Loading..."}
            status={item?.status}
            nodeId={nodeId}
            lenses={BACKLOG_LENSES}
            actions={item ? <HeaderPrimaryAction className="shrink-0" onFinalizeWorkshop={handlers.handleFinalizeWorkshop} onRunWorkshop={handlers.handleRunWorkshop} /> : undefined}
            onStatusChange={!isLocked ? (newStatus) => data.updateStatus(newStatus) : undefined}
            statusChangePending={data.isUpdatingStatus}
            tabBar={tabBar}
          />
        }
        mobileActions={mobileActionButtons}
        mobileActionsTitle="Backlog Actions"
      >
        <div className="space-y-0 lg:space-y-6" data-testid={selectors.backlogDetails.page}>
          {isPageLoading && (
            <PageLoadingState label="Loading backlog details..." variant="detail" testId="backlog-details-loading-state" />
          )}
          {pageError && (
            <ErrorState error={pageError} title="Unable to load backlog item" onRetry={refetchItem} />
          )}
          {item && !pageError && (
            <>
              {/* Mobile tab content */}
              <div className="lg:hidden">
                {activeTab === "info" && (
                  <div className="flex-1 space-y-0 overflow-y-auto pb-4">
                    {deleteError && (
                      <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                        {deleteError}
                      </div>
                    )}
                    {detailsPanel}
                    <BacklogScenariosPanel targetScenarios={targetScenarios} />
                    {notesPanel}
                    {operationalTargetsSection}
                  </div>
                )}
                {activeTab === "prompt" && (
                  <PlanPanel backlogKind={backlogKind as BacklogKind} backlogName={name} className="flex-1 overflow-y-auto" />
                )}
                {activeTab === "files" && fileWorkspaceElement}
                {activeTab === "output" && (
                  <div className="flex-1 space-y-0 overflow-y-auto pb-4">
                    <OutputTab
                      executionHistory={executionHistory}
                      agentRunIsActive={agentRunIsActive}
                      latestAgentActivity={latestAgentActivity}
                      reviewRounds={reviewRounds}
                      isGatheringEvidence={isGatheringEvidence}
                      backlogKind={backlogKind ?? ""}
                      backlogName={name ?? ""}
                      onStopRun={(runId) => void stopRun(runId)}
                      onFollowUp={(exec) => uiStore.setFollowUpTarget(exec)}
                      onArchive={item ? () => handlers.handleUpdateItem({
                        title: item.title, description: item.description,
                        status: "archived", priority: item.priority, tags: item.tags,
                      }) : undefined}
                      onVerifyEvidence={(round, evidenceId, verified) => {
                        const execId = executionHistory?.[0]?.executionId;
                        void reviewService.verifyEvidence(backlogKind ?? "", name ?? "", round, evidenceId, verified, execId)
                          .then(() => queryClient.invalidateQueries({ queryKey: ["review-rounds", backlogKind, name] }));
                      }}
                      onRequestMoreEvidence={(round, evidenceId) => {
                        useReviewStore.getState().openRequestPanel(round, evidenceId);
                      }}
                    />
                  </div>
                )}
                {activeTab === "activity" && (
                  <div className="flex-1 space-y-0 overflow-y-auto pb-4">
                    <ActivityTab
                      timeline={timeline}
                      agentRunIsActive={agentRunIsActive}
                      latestAgentActivity={latestAgentActivity}
                      agentManagerUiUrl={agentManagerUiUrl}
                      onStopRun={(runId) => void stopRun(runId)}
                      onFollowUp={(exec) => uiStore.setFollowUpTarget(exec)}
                      onViewExecution={(exec) => selectExecution(exec.executionId)}
                    />
                  </div>
                )}
              </div>

              {/* Desktop content */}
              <div className="hidden space-y-6 lg:block">
                <BacklogDesktopHeader
                  item={item}
                  deleteError={deleteError}
                  primaryAction={<HeaderPrimaryAction onFinalizeWorkshop={handlers.handleFinalizeWorkshop} onRunWorkshop={handlers.handleRunWorkshop} />}
                  onEdit={uiStore.openEdit}
                  onDelete={uiStore.openDelete}
                  onResetWorkshop={uiStore.openWorkshopReset}
                  hasWorkshopRounds={(workshopRounds?.length ?? 0) > 0}
                  onOpenAgentDialog={uiStore.openAgent}
                  onStatusChange={(newStatus) => handlers.handleUpdateItem({
                    title: item.title, description: item.description,
                    status: newStatus, priority: item.priority, tags: item.tags,
                  })}
                />
                <div>
                  {activeTab === "info" && (
                    <div className="space-y-0 pt-3">
                      {detailsPanel}
                      <BacklogScenariosPanel targetScenarios={targetScenarios} />
                      {notesPanel}
                      {operationalTargetsSection}
                    </div>
                  )}
                  {activeTab === "prompt" && (
                    <PlanPanel backlogKind={backlogKind as BacklogKind} backlogName={name} className="mt-3 min-h-[500px] rounded-lg border border-slate-800 bg-slate-900/50" />
                  )}
                  {activeTab === "files" && fileWorkspaceElement}
                  {activeTab === "output" && (
                    <div className="space-y-0 pt-3">
                      <OutputTab
                        executionHistory={executionHistory}
                        agentRunIsActive={agentRunIsActive}
                        latestAgentActivity={latestAgentActivity}
                        reviewRounds={reviewRounds}
                        isGatheringEvidence={isGatheringEvidence}
                        backlogKind={backlogKind ?? ""}
                        backlogName={name ?? ""}
                        onStopRun={(runId) => void stopRun(runId)}
                        onFollowUp={(exec) => uiStore.setFollowUpTarget(exec)}
                        onArchive={item ? () => handlers.handleUpdateItem({
                          title: item.title, description: item.description,
                          status: "archived", priority: item.priority, tags: item.tags,
                        }) : undefined}
                        onVerifyEvidence={(round, evidenceId, verified) => {
                          const execId = executionHistory?.[0]?.executionId;
                          void reviewService.verifyEvidence(backlogKind ?? "", name ?? "", round, evidenceId, verified, execId)
                            .then(() => queryClient.invalidateQueries({ queryKey: ["review-rounds", backlogKind, name] }));
                        }}
                        onRequestMoreEvidence={(round, evidenceId) => {
                          useReviewStore.getState().openRequestPanel(round, evidenceId);
                        }}
                      />
                    </div>
                  )}
                  {activeTab === "activity" && (
                    <div className="space-y-0 pt-3">
                      <ActivityTab
                        timeline={timeline}
                        agentRunIsActive={agentRunIsActive}
                        latestAgentActivity={latestAgentActivity}
                        agentManagerUiUrl={agentManagerUiUrl}
                        onStopRun={(runId) => void stopRun(runId)}
                        onFollowUp={(exec) => uiStore.setFollowUpTarget(exec)}
                        onViewExecution={(exec) => selectExecution(exec.executionId)}
                      />
                    </div>
                  )}
                </div>
              </div>
            </>
          )}

          <EvidenceRequestPanel
            backlogKind={backlogKind ?? ""}
            backlogName={name ?? ""}
            reviewRounds={reviewRounds}
            onAction={() => void queryClient.invalidateQueries({ queryKey: ["review-rounds", backlogKind, name] })}
          />

          <BacklogDialogs
            data={data}
            handlers={handlers}
            files={files}
            readinessData={readinessData}
            agentDialogTargetIds={agentDialogTargetIds}
            agentDialogRequirementIds={agentDialogRequirementIds}
            upsertItem={upsertItem}
          />
        </div>
      </DetailPageLayout>
    </BacklogDetailProvider>
  );
}
