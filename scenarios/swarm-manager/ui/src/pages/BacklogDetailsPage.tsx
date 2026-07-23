/**
 * Backlog Details Page — layout shell and tab routing.
 * Data fetching: useBacklogDetailData, UI handlers: useBacklogHandlers.
 * UI state: backlog-detail-ui-store, computed values: BacklogDetailContext.
 * [REQ:REQ-P0-004]
 */

import { useCallback, useState, useMemo, useRef, useEffect } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { Archive, ArchiveRestore, CheckSquare, CircleHelp, ClipboardList, Files, GitPullRequestArrow, Network, RefreshCw, RotateCcw, Sparkles } from "lucide-react";
import { CompactTabBar, type CompactTabItem } from "../components/ui/compact-tab-bar";
import { Button } from "../components/ui/button";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { PlanPanel } from "../components/backlog/plan-panel";
import { useUrlState } from "../hooks/use-url-state";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { BacklogFileWorkspace } from "../components/backlog/backlog-file-workspace";
import { BacklogDetailsPanel } from "../components/backlog/backlog-details-panel";
import { BacklogNotesPanel } from "../components/backlog/backlog-notes-panel";
import { buildBacklogActionMenuItems } from "../components/backlog/backlog-action-buttons";
import type { ActionMenuItem } from "../components/ui/action-menu";
import { WorkFeedList, type WorkFeedEntry } from "../components/backlog/activity-surface/work-feed-list";
import { ActiveWorkCard } from "../components/backlog/activity-surface/active-work-card";
import { ReviewDecisionCard } from "../components/backlog/activity-surface/review-decision-card";
import { RelatedTab } from "../components/related/RelatedTab";
import { BacklogScenariosPanel } from "../components/backlog/backlog-scenarios-panel";
import { BacklogDialogs } from "../components/backlog/backlog-dialogs";
import { useAttachToSessionAction } from "../components/session/context/useAttachToSessionAction";
import { ProposalSessionsPanel } from "../components/session/ProposalSessionsPanel";
import { proposalSessionService } from "../services/proposal-session-service";
import { backlogOption } from "../components/session/context/session-context-refs";
import { OperationalTargetsPanel } from "../components/backlog/operational-targets-panel";
import { BulkActionToolbar } from "../components/backlog/bulk-action-toolbar";
import { useBacklogDetailData } from "../hooks/useBacklogDetailData";
import { useEmbeddedServiceUrl } from "../hooks/useEmbeddedServiceUrl";
import { useRuntimeConfig } from "../hooks/useRuntimeConfig";
import { useStorePolling } from "../hooks/useStorePolling";
import { useBacklogHandlers } from "../hooks/useBacklogHandlers";
import { findBacklogFileByPath } from "../lib/workshop-files";
import { isAgentActivityBlocking, isAgentActivityExecuting } from "../lib/agent-activity-utils";
import { selectors } from "../consts/selectors";
import { BACKLOG_KIND_LABELS, BACKLOG_KINDS } from "../types";
import type { BacklogFile, BacklogKind } from "../types";
import {
  selectLatestActivityForBacklog,
  useAgentActivitiesStore,
  useBacklogDetailUIStore,
  useBacklogStore,
} from "../stores";
import { BACKLOG_LENSES } from "../components/detail/lens-options";
import { backlogService } from "../services/backlog-service";
import { autoFilerService } from "../services/auto-filer-service";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { EvidenceRequestPanel } from "../components/backlog/evidence-request-panel";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMutation } from "@tanstack/react-query";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { formatRelativeTime } from "../lib";
import { routeTargetToNodeId } from "../app/routes/route-paths";
import { useAppBack } from "../app/routes/useAppBack";
import { BacklogDetailProvider } from "../contexts/BacklogDetailContext";
import { FileServiceProvider } from "../contexts/FileServiceContext";
import { createBacklogFileServiceAdapter } from "../services/backlog/backlog-file-service-adapter";
import { nextActionDetailTab } from "../lib/backlog-next-action";

const DEFAULT_PREVIEW_FILE_PATH = "spec.json";
const AGENT_RUN_REFRESH_MS = 6000;
const LIFECYCLE_RESET_SCOPES: Array<[string, string]> = [
  ["review", "Review rounds"],
  ["handoff_executions", "Handoff data and executions"],
  ["plan_unbind", "Plan binding"],
];
type DetailsTab = "info" | "prompt" | "decide" | "files" | "activity" | "related";

export function BacklogDetailsPage() {
  // --- Navigation / selection ---
  const goBack = useAppBack();
  const params = useParams<{ kind: string; name: string }>();
  const kind = params.kind;
  const name = params.name;
  const nodeId = routeTargetToNodeId({ entityType: "backlog", kind, name });
  const closeDetail = goBack;
  const backlogKind = BACKLOG_KINDS.includes(kind as BacklogKind) ? (kind as BacklogKind) : null;
  const [searchParams, setSearchParams] = useSearchParams();

  // --- Global stores ---
  const upsertItem = useBacklogStore((s) => s.upsertItem);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
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

  const agentRunIsBusy = isAgentActivityExecuting(latestAgentActivity?.status);
  const agentRunIsBlocking = isAgentActivityBlocking(latestAgentActivity?.status);

  // --- UI state store ---
  const uiStore = useBacklogDetailUIStore();
  const { getDeleteConfirmLevel } = useRuntimeConfig();
  const queryClient = useQueryClient();

  // --- Data hook ---
  const data = useBacklogDetailData({
    backlogKind,
    name,
    agentRunIsExecuting: agentRunIsBusy,
    agentRunIsBlocking,
  });
  const {
    item, isLoadingItem, itemError, refetchItem, spawnedItems,
    files, isLoadingFiles, filesError, refetchFiles,
    executionHistory, reviewRounds, nextAction,
    archiveTargets,
    depRelations, itemActions, targetScenarios,
    isLocked, isTerminal,
    deleteError, archiveError, isArchiving, isUpdating,
    isBatchReviewing, isFileActionPending,
  } = data;

  // --- Local UI state (URL-synced or needs render) ---
  const [activeTab, setActiveTab] = useUrlState<DetailsTab>("tab", "info", {
    validate: (v): v is DetailsTab => ["info", "prompt", "decide", "files", "activity", "related"].includes(v),
  });
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);
  const [dismissSuggestionPending, setDismissSuggestionPending] = useState(false);
  const [dismissSuggestionError, setDismissSuggestionError] = useState<string | null>(null);
  const [recreateOpen, setRecreateOpen] = useState(false);
  const [resetArtifactsOpen, setResetArtifactsOpen] = useState(false);
  const [resetScope, setResetScope] = useState<string[]>(["review"]);
  const [lifecyclePending, setLifecyclePending] = useState(false);
  const [lifecycleError, setLifecycleError] = useState<string | undefined>();
  const planAuthorMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog item is required to author a plan.");
      return backlogService.startPlanAuthor(backlogKind, name);
    },
    onSuccess: () => {
      setActiveTab("activity");
      void refreshActivities(true);
    },
  });
  const { data: proposalSessions = [] } = useQuery({
    queryKey: ["proposal-sessions", "backlog_item", backlogKind && name ? `${backlogKind}/${name}` : ""],
    queryFn: () => proposalSessionService.list({ type: "backlog_item", ref: `${backlogKind}/${name}` }),
    enabled: Boolean(backlogKind && name),
    refetchInterval: 15_000,
  });
  const { data: workFeed = [], isLoading: isLoadingWorkFeed, error: workFeedError } = useQuery({
    queryKey: ["work-feed", backlogKind, name],
    queryFn: async () => (await defaultApiClient.get<{ items: WorkFeedEntry[] }>(API_ENDPOINTS.backlogWorkFeed(backlogKind!, name!))).items,
    enabled: activeTab === "activity" && Boolean(backlogKind && name),
    refetchInterval: agentRunIsBlocking ? 6_000 : 20_000,
  });
  const proposalCount = useMemo(
    () => proposalSessions.reduce((count, session) => count + (session.proposals ?? []).filter((proposal) => proposal.kind === "mutation_list" || proposal.kind === "no_change_recommendation").length, 0),
    [proposalSessions],
  );
  const activeExecution = useMemo(
    () => executionHistory?.find((execution) => ["starting", "running", "needs_review"].includes(execution.status)) ?? null,
    [executionHistory],
  );
  const { url: agentManagerUiUrl } = useEmbeddedServiceUrl("agent-manager");
  const handleDismissSuggestion = useCallback(async () => {
    if (!item) return;
    setDismissSuggestionPending(true);
    setDismissSuggestionError(null);
    try {
      const archived = await autoFilerService.dismissSuggestion(item.kind, item.name);
      upsertItem(archived);
      refetchItem();
    } catch (error) {
      setDismissSuggestionError(error instanceof Error ? error.message : "Unable to dismiss suggestion");
    } finally {
      setDismissSuggestionPending(false);
    }
  }, [item, refetchItem, upsertItem]);
  const recreateItem = useCallback(async () => {
    if (!backlogKind || !name) return;
    setLifecyclePending(true);
    setLifecycleError(undefined);
    try {
      await defaultApiClient.post(API_ENDPOINTS.backlogRecreate(backlogKind, name), {});
      setRecreateOpen(false);
      data.invalidateItem();
    } catch (error) {
      setLifecycleError(error instanceof Error ? error.message : "Unable to recreate item.");
    } finally {
      setLifecyclePending(false);
    }
  }, [backlogKind, data, name]);
  const resetArtifacts = useCallback(async () => {
    if (!backlogKind || !name || resetScope.length === 0) return;
    setLifecyclePending(true);
    setLifecycleError(undefined);
    try {
      await defaultApiClient.post(API_ENDPOINTS.backlogResetArtifacts(backlogKind, name), { scope: resetScope });
      setResetArtifactsOpen(false);
      data.invalidateItem();
    } catch (error) {
      setLifecycleError(error instanceof Error ? error.message : "Unable to reset artifacts.");
    } finally {
      setLifecyclePending(false);
    }
  }, [backlogKind, data, name, resetScope]);

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
  });

  // --- Computed labels ---
  const agentRunningLabel = useMemo(() => {
    if (!agentRunIsBusy || !latestAgentActivity) return "Agent running\u2026";
    switch (latestAgentActivity.purpose) {
      case "workshop": return "Running Plan Workshop review\u2026";
      case "finalize": return "Finalizing historical workshop\u2026";
      case "research": return "Running research\u2026";
      case "initialize": return "Initializing historical workshop\u2026";
      case "process": return "Processing\u2026";
      default: return "Agent running\u2026";
    }
  }, [agentRunIsBusy, latestAgentActivity]);

  const attachToSession = useAttachToSessionAction(item ? backlogOption(item) : null);

  // --- Effects ---

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

  // Reset UI store on mount to clear stale dialog state (e.g. showDelete)
  // that persists in the global Zustand store across unmount/remount cycles.
  useEffect(() => {
    uiStore.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount only
  }, []);

  // Reset tab and UI state when navigating between backlog items while mounted.
  const prevItemRef = useRef(`${backlogKind}/${name}`);
  useEffect(() => {
    const key = `${backlogKind}/${name}`;
    if (key !== prevItemRef.current) {
      prevItemRef.current = key;
      setActiveTab("info");
      uiStore.reset();
    }
  }, [backlogKind, name, setActiveTab, uiStore]);

  const reconciliationItemKeyRef = useRef<string>("");
  const prevBlockingRef = useRef(false);
  useEffect(() => {
    if (!backlogKind || !name) return;

    const itemKey = `${backlogKind}/${name}`;
    if (reconciliationItemKeyRef.current !== itemKey) {
      reconciliationItemKeyRef.current = itemKey;
      prevBlockingRef.current = agentRunIsBlocking;
      return;
    }

    const blockingEnded = prevBlockingRef.current && !agentRunIsBlocking;
    prevBlockingRef.current = agentRunIsBlocking;
    if (!blockingEnded) return;

    void Promise.allSettled([
      queryClient.refetchQueries({ queryKey: ["backlog", backlogKind, name] }),
      queryClient.refetchQueries({ queryKey: ["backlog", backlogKind, name, "files"] }),
      queryClient.refetchQueries({ queryKey: ["executions", backlogKind, name] }),
      queryClient.refetchQueries({ queryKey: ["review-rounds", backlogKind, name] }),
    ]);
  }, [agentRunIsBlocking, backlogKind, name, queryClient]);

  const fileService = useMemo(
    () => backlogKind && name ? createBacklogFileServiceAdapter(backlogKind, name) : null,
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
    backlogKind: backlogKind,
    name: name ?? "",
    item,
    itemActions,
    isLocked,
    isTerminal,
    agentRunIsActive: agentRunIsBlocking,
    latestAgentActivity,
    agentRunningLabel,
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
      onSaveDescription={data.updateDescription}
      isSavingDescription={data.isUpdatingDescription}
      descriptionSaveError={data.updateDescriptionError}
    />
  ) : null;

  const notesPanel = <BacklogNotesPanel />;

  const menuActions: ActionMenuItem[] = item && itemActions ? [
    attachToSession.actionItem,
    ...buildBacklogActionMenuItems(
      {
        itemActions,
        isLocked,
        isTerminal,
        agentRunningLabel,
      },
      {
        isUpdating,
        onStartRun: uiStore.openRunModal,
        onEdit: uiStore.openEdit,
        onFollowUp: () => uiStore.setFollowUpTarget(executionHistory?.[0] ?? null),
        onRetry: () => {
          void backlogService.retry(item.kind, item.name).then(async () => {
            data.refetchItem();
            await queryClient.invalidateQueries({
              queryKey: ["execution-history", item.kind, item.name],
            });
          });
        },
        onArchive: () => handlers.handleArchiveItem(),
        onDelete: () => {
          if (getDeleteConfirmLevel("backlog") === "none") {
            handlers.handleDeleteConfirm();
          } else {
            uiStore.openDelete();
          }
        },
      },
    ),
    {
      label: "Recreate item",
      description: "Archive this item and create a fresh successor with its lineage intact.",
      icon: <RotateCcw />,
      onSelect: () => setRecreateOpen(true),
      disabled: lifecyclePending || agentRunIsBlocking,
    },
    {
      label: "Reset derived artifacts",
      description: "Remove selected generated work while preserving the canonical specification.",
      icon: <RefreshCw />,
      onSelect: () => setResetArtifactsOpen(true),
      disabled: lifecyclePending || agentRunIsBlocking,
    },
  ] : [];

  const suggestionActions = item?.status === "suggested" && item.archivedAt == null ? (
    <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/[0.04] px-3 py-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium text-cyan-200">Auto-filer suggestion</h3>
          <p className="mt-1 text-xs text-slate-400">Accept to add it to the backlog, or dismiss to archive and remember the finding.</p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            onClick={() => data.updateStatus("backlog")}
            disabled={data.isUpdatingStatus}
          >
            <CheckSquare className="mr-1 h-3.5 w-3.5" />
            Accept
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => void handleDismissSuggestion()}
            disabled={dismissSuggestionPending}
          >
            <Archive className="mr-1 h-3.5 w-3.5" />
            {dismissSuggestionPending ? "Dismissing..." : "Dismiss"}
          </Button>
        </div>
      </div>
      {dismissSuggestionError ? (
        <p className="mt-2 text-xs text-amber-300">{dismissSuggestionError}</p>
      ) : null}
    </div>
  ) : null;

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

  const tabItems: CompactTabItem<DetailsTab>[] = [
    { value: "info", label: "Info", icon: CircleHelp },
    { value: "prompt", label: "Plan", icon: Sparkles },
    {
      value: "decide",
      label: "Decide",
      icon: GitPullRequestArrow,
      count: proposalCount || undefined,
    },
    { value: "files", label: "Files", icon: Files },
    {
      value: "activity",
      label: "Activity",
      icon: ClipboardList,
      badge: agentRunIsBusy ? (
        <span className="relative ml-1 flex h-2 w-2">
          <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
          <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
        </span>
      ) : undefined,
    },
    { value: "related", label: "Related", icon: Network },
  ];

  const tabBar = item ? (
    <div className="border-t border-slate-800/50" data-testid={selectors.backlogDetails.tabRow}>
      <CompactTabBar
        items={tabItems}
        activeValue={activeTab}
        onValueChange={setActiveTab}
        aria-label="Backlog item sections"
        className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3"
        tabTestIdPrefix="backlog-details-tab"
      />
    </div>
  ) : null;

  const handleNextAction = () => {
    if (nextAction) {
      const targetTab = nextActionDetailTab(nextAction);
      if (targetTab) {
        setActiveTab(targetTab);
        return;
      }
    }
    switch (nextAction?.id) {
      case "run":
        uiStore.openRunModal();
        break;
      case "accept_suggestion":
        data.updateStatus("backlog");
        break;
      case "archive":
        handlers.handleArchiveItem();
        break;
      case "retry":
        if (item) void backlogService.retry(item.kind, item.name).then(() => data.refetchItem());
        break;
      default:
        break;
    }
  };

  return (
    <BacklogDetailProvider value={contextValue}>
      <DetailPageLayout
        header={
          <DetailPageHeader
            entityType={backlogKind ? BACKLOG_KIND_LABELS[backlogKind] : "Backlog"}
            title={item?.title ?? name ?? "Loading..."}
            status={item?.status}
            nodeId={nodeId}
            lenses={BACKLOG_LENSES}
            menuActions={menuActions}
            primaryAction={nextAction && nextAction.id !== "none" ? (
              <Button size="sm" onClick={handleNextAction} disabled={!nextAction.enabled} title={nextAction.reason}>
                {nextAction.compactLabel}
              </Button>
            ) : undefined}
            onStatusChange={!isLocked ? (newStatus) => data.updateStatus(newStatus) : undefined}
            statusChangePending={data.isUpdatingStatus}
            tabBar={tabBar}
            showLenses={activeTab === "info"}
          />
        }
      >
        {attachToSession.sheet}
        <div className="space-y-0 lg:space-y-6" data-testid={selectors.backlogDetails.page}>
          {isPageLoading && (
            <PageLoadingState label="Loading backlog details..." variant="detail" testId="backlog-details-loading-state" />
          )}
          {pageError && (
            <ErrorState error={pageError} title="Unable to load backlog item" onRetry={refetchItem} />
          )}
          {item && !pageError && (
            <>
              {item.archivedAt != null && (
                <div className="mb-3 flex items-center justify-between rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
                  <div className="flex items-center gap-2 text-sm text-amber-300">
                    <Archive className="h-4 w-4" />
                    <span>Archived {formatRelativeTime(item.archivedAt)}</span>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    className="border-amber-500/30 text-amber-300 hover:bg-amber-500/10"
                    onClick={handlers.handleUnarchiveItem}
                    disabled={isArchiving}
                  >
                    <ArchiveRestore className="mr-2 h-4 w-4" />
                    {isArchiving ? "Restoring..." : "Unarchive"}
                  </Button>
                </div>
              )}
              {(deleteError || archiveError) && (
                <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                  {deleteError || archiveError}
                </div>
              )}
              {activeTab === "info" && (
                <div className="flex-1 space-y-0 overflow-y-auto pb-4 lg:pt-3">
                  {suggestionActions}
                  {detailsPanel}
                  <BacklogScenariosPanel targetScenarios={targetScenarios} />
                  {notesPanel}
                  {operationalTargetsSection}
                </div>
              )}
              {activeTab === "prompt" && (
                <PlanPanel
                  backlogKind={backlogKind}
                  backlogName={name}
                  className="flex-1 overflow-y-auto lg:mt-3 lg:min-h-[500px]"
                  onAuthorPlan={() => planAuthorMutation.mutate()}
                  authorPlanPending={planAuthorMutation.isPending}
                  authorPlanError={planAuthorMutation.error instanceof Error ? planAuthorMutation.error.message : null}
                />
              )}
              {activeTab === "decide" && backlogKind && name && item && (
                <ProposalSessionsPanel target={{ type: "backlog_item", ref: `${backlogKind}/${name}`, name: item.title || name }} />
              )}
              {activeTab === "files" && fileWorkspaceElement}
              {activeTab === "related" && backlogKind && name && <RelatedTab target={{ kind: "backlog", backlogKind, name }} enabled />}
              {activeTab === "activity" && (
                <div className="flex-1 space-y-0 overflow-y-auto pb-4 lg:pt-3">
                  {activeExecution ? <ActiveWorkCard executionId={activeExecution.executionId} agentManagerUrl={agentManagerUiUrl} onStopped={() => { void queryClient.invalidateQueries({ queryKey: ["executions", backlogKind, name] }); void queryClient.invalidateQueries({ queryKey: ["work-feed", backlogKind, name] }); }} /> : null}
                  {!activeExecution && (item.status === "review_pending" || item.status === "in_review") ? <ReviewDecisionCard kind={backlogKind} name={name} round={reviewRounds.at(-1)} onDecided={() => { void data.invalidateItem(); void queryClient.invalidateQueries({ queryKey: ["review-rounds", backlogKind, name] }); void queryClient.invalidateQueries({ queryKey: ["work-feed", backlogKind, name] }); }} onSendBack={() => { const round = reviewRounds.at(-1); const context = ["Address the review feedback before continuing.", round?.agent_assessment ? `Review note: ${round.agent_assessment}` : "", round?.disposition ? `Disposition: ${round.disposition.kind} — ${round.disposition.rationale}` : ""].filter(Boolean).join("\n\n"); uiStore.setFollowUpContext(context); uiStore.setFollowUpTarget(executionHistory?.[0] ?? null); }} /> : null}
                  <WorkFeedList entries={workFeed} loading={isLoadingWorkFeed} error={workFeedError instanceof Error ? workFeedError : null} />
                </div>
              )}
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
            upsertItem={upsertItem}
          />
          <ConfirmDialog
            isOpen={recreateOpen}
            onClose={() => setRecreateOpen(false)}
            onConfirm={() => void recreateItem()}
            title="Recreate backlog item"
            description={`Archives "${item?.title || name}" and creates a fresh backlog clone. Metadata, membership, dependencies, and lineage are retained; derived work starts fresh.`}
            confirmationText={item?.name}
            confirmLabel="Recreate item"
            isLoading={lifecyclePending}
            errorMessage={lifecycleError}
          />
          <ConfirmDialog
            isOpen={resetArtifactsOpen}
            onClose={() => setResetArtifactsOpen(false)}
            onConfirm={() => void resetArtifacts()}
            title="Reset derived artifacts"
            description="Deletes only the selected derived artifacts. The canonical item specification is kept."
            confirmLabel="Reset selected artifacts"
            isLoading={lifecyclePending}
            errorMessage={lifecycleError}
            sidePanel={<div className="space-y-2 p-4"><p className="text-sm font-medium text-slate-100">Choose what to remove</p>{LIFECYCLE_RESET_SCOPES.map(([value, label]) => <label key={value} className="flex items-center gap-2 text-sm text-slate-300"><input type="checkbox" checked={resetScope.includes(value)} onChange={() => setResetScope((current) => current.includes(value) ? current.filter((entry) => entry !== value) : [...current, value])} />{label}</label>)}</div>}
          />
        </div>
      </DetailPageLayout>
    </BacklogDetailProvider>
  );
}
