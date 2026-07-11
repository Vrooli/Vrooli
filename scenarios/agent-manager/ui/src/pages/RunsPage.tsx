import type React from "react";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  Ban,
  Check,
  CheckSquare,
  Clock,
  Eye,
  PlayCircle,
  RefreshCw,
  Search,
  Square,
  Trash2,
  X,
  XCircle,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { formatStandardRelativeTime } from "../lib/dateTime";
import type {
  AgentProfile,
  ApproveFormData,
  ApproveResult,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
  Task,
} from "../types";
import { ApprovalState, RunStatus } from "../types";
import type { UseRunEventStoreReturn } from "../hooks/useRunEventStore";
import { ApplyInvestigationModal } from "../components/ApplyInvestigationModal";
import { InvestigateModal } from "../components/InvestigateModal";
import { ResumeFromFailureModal } from "../components/ResumeFromFailureModal";
import { RunDetail } from "../components/RunDetail";
import { useViewportSize } from "../hooks/useViewportSize";
import { useRunsPageState } from "../hooks/useRunsPageState";
import { useSelectedRunController } from "../hooks/useSelectedRunController";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { BoundedList, ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";

interface RunsPageProps {
  runs: Run[];
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onStopRun: (id: string) => Promise<void>;
  onDeleteRun: (id: string) => Promise<void>;
  onRetryRun: (run: Run) => Promise<Run>;
  onGetRun: (id: string) => Promise<Run>;
  onGetEvents: (id: string, options?: { afterSequence?: bigint }) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<RunDiff>;
  onGetTask: (id: string) => Promise<Task>;
  onApproveRun: (id: string, req: ApproveFormData) => Promise<ApproveResult>;
  onRejectRun: (id: string, req: RejectFormData) => Promise<void>;
  onPartialApproveRun: (id: string, fileIds: string[], actor?: string, commitMsg?: string) => Promise<ApproveResult>;
  onInvestigateRuns: (
    runIds: string[],
    customContext?: string,
    depth?: "quick" | "standard" | "deep",
    projectRoot?: string,
    scopePaths?: string[],
    attachmentIds?: string[],
    overrides?: { roleRef?: string }
  ) => Promise<Run>;
  onApplyInvestigation: (
    investigationRunId: string,
    customContext?: string,
    attachmentIds?: string[],
    overrides?: { roleRef?: string }
  ) => Promise<Run>;
  onResumeFromFailedRun: (runId: string, customContext?: string, attachmentIds?: string[]) => Promise<Run>;
  onContinueRun: (id: string, message: string, attachmentIds?: string[]) => Promise<Run>;
  onDeleteRunMessage: (runId: string, eventId: string) => Promise<void>;
  onRefresh: () => void;
  runEventStore: UseRunEventStoreReturn;
  wsSubscribe: (runId: string) => void;
  wsUnsubscribe: (runId: string) => void;
}

const STATUS_FILTER_OPTIONS = [
  { value: String(RunStatus.PENDING), label: "Pending" },
  { value: String(RunStatus.STARTING), label: "Starting" },
  { value: String(RunStatus.RUNNING), label: "Running" },
  { value: String(RunStatus.NEEDS_REVIEW), label: "Needs Review" },
  { value: String(RunStatus.COMPLETE), label: "Complete" },
  { value: String(RunStatus.FAILED), label: "Failed" },
  { value: String(RunStatus.CANCELLED), label: "Cancelled" },
];

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
];

const VALID_TABS = new Set(["task", "timeline", "diff", "cost"]);

interface RunListRowProps {
  run: Run;
  index: number;
  selected: boolean;
  highlighted: boolean;
  selectionMode: boolean;
  taskTitle: string;
  profileName: string;
  onSelect: (run: Run) => void;
  onCheckboxChange: (runId: string, index: number, shiftKey: boolean) => void;
  onStop: (runId: string) => void;
  onResume: (run: Run) => void;
  onDelete: (run: Run) => void;
}

const RunListRow = memo(function RunListRow({
  run,
  index,
  selected,
  highlighted,
  selectionMode,
  taskTitle,
  profileName,
  onSelect,
  onCheckboxChange,
  onStop,
  onResume,
  onDelete,
}: RunListRowProps) {
  return (
    <ListItem
      selected={selected}
      highlighted={highlighted}
      onClick={() => onSelect(run)}
      checkbox={
        selectionMode ? (
          <input
            type="checkbox"
            checked={highlighted}
            onChange={(e) => {
              e.stopPropagation();
              onCheckboxChange(
                run.id,
                index,
                e.nativeEvent instanceof MouseEvent && e.nativeEvent.shiftKey
              );
            }}
            onClick={(e) => e.stopPropagation()}
            className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
          />
        ) : undefined
      }
      icon={<RunStatusIcon status={run.status} approvalState={run.approvalState} />}
      actions={
        <div className="flex items-center gap-1">
          {run.actions?.canStop && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              aria-label={`Stop run ${taskTitle}`}
              onClick={(e) => {
                e.stopPropagation();
                onStop(run.id);
              }}
            >
              <Square className="h-3.5 w-3.5" />
            </Button>
          )}
          {run.actions?.canResumeFromFailure && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              aria-label={`Resume run ${taskTitle} from failure`}
              title="Resume: continue this task with the prior transcript + diff as context"
              onClick={(e) => {
                e.stopPropagation();
                onResume(run);
              }}
            >
              <PlayCircle className="h-3.5 w-3.5" />
            </Button>
          )}
          {run.actions?.canDelete && (
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
              aria-label={`Delete run ${taskTitle}`}
              onClick={(e) => {
                e.stopPropagation();
                onDelete(run);
              }}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      }
    >
      <ListItemTitle>{taskTitle}</ListItemTitle>
      <ListItemSubtitle>
        {profileName} | {formatStandardRelativeTime(run.createdAt)}
      </ListItemSubtitle>
    </ListItem>
  );
});

export function RunsPage({
  runs,
  tasks,
  profiles,
  loading,
  error,
  onStopRun,
  onDeleteRun,
  onRetryRun,
  onGetRun,
  onGetEvents,
  onGetDiff,
  onGetTask,
  onApproveRun,
  onRejectRun,
  onPartialApproveRun,
  onInvestigateRuns,
  onApplyInvestigation,
  onResumeFromFailedRun,
  onContinueRun,
  onDeleteRunMessage,
  onRefresh,
  runEventStore,
  wsSubscribe,
  wsUnsubscribe,
}: RunsPageProps) {
  const { runId } = useParams<{ runId?: string }>();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { isDesktop } = useViewportSize();
  const isDeselectingRef = useRef(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleteConfirmRun, setDeleteConfirmRun] = useState<Run | null>(null);
  const [mobileHeaderLeft, setMobileHeaderLeft] = useState<React.ReactNode>(null);
  const [mobileHeaderRight, setMobileHeaderRight] = useState<React.ReactNode>(null);

  const {
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    sortBy,
    setSortBy,
    selectionMode,
    setSelectionMode,
    selectedRunIds,
    setSelectedRunIds,
    investigateModalOpen,
    setInvestigateModalOpen,
    investigateLoading,
    setInvestigateLoading,
    investigateError,
    setInvestigateError,
    toggleSelectionMode,
    clearSelection,
    handleRunCheckboxChange,
  } = useRunsPageState();

  const [applyModalOpen, setApplyModalOpen] = useState(false);
  const [applyInvestigationRun, setApplyInvestigationRun] = useState<Run | null>(null);
  const [applyLoading, setApplyLoading] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);

  const [resumeModalOpen, setResumeModalOpen] = useState(false);
  const [resumeTargetRun, setResumeTargetRun] = useState<Run | null>(null);
  const [resumeLoading, setResumeLoading] = useState(false);
  const [resumeError, setResumeError] = useState<string | null>(null);

  const profileNameById = useMemo(
    () => new Map(profiles.map((profile) => [profile.id, profile.name])),
    [profiles]
  );
  const getProfileName = useCallback(
    (profileId?: string) =>
      profileId ? profileNameById.get(profileId) || "Unknown Profile" : "Unknown Profile",
    [profileNameById]
  );

  const applyInvestigationRunId = applyInvestigationRun?.id ?? null;
  const {
    selectedRun,
    setSelectedRun,
    selectedRunId,
    diff,
    events,
    eventsLoading,
    diffLoading,
    resolvedRuns,
    getTaskById,
    getTaskTitle,
    loadRunDetails,
  } = useSelectedRunController({
    runs,
    tasks,
    routeRunId: runId,
    isDeselectingRef,
    onGetRun,
    onGetEvents,
    onGetDiff,
    onGetTask,
    runEventStore,
    wsSubscribe,
    wsUnsubscribe,
  });

  // Subscribe to WebSocket events for the investigation run when Apply modal is open
  // This ensures we get updates when recommendation extraction completes
  useEffect(() => {
    if (!applyInvestigationRunId || !applyModalOpen) return;
    runEventStore.actions.subscribeRun(applyInvestigationRunId);
    wsSubscribe(applyInvestigationRunId);
    return () => {
      runEventStore.actions.unsubscribeRun(applyInvestigationRunId);
      wsUnsubscribe(applyInvestigationRunId);
    };
  }, [applyInvestigationRunId, applyModalOpen, runEventStore.actions, wsSubscribe, wsUnsubscribe]);

  useEffect(() => {
    if (!applyInvestigationRunId) return;
    const snapshot = runEventStore.state.runsById[applyInvestigationRunId];
    if (snapshot) {
      setApplyInvestigationRun((prev) => (prev ? { ...prev, ...snapshot } as Run : prev));
    }
  }, [applyInvestigationRunId, runEventStore.state.runsById]);

  useEffect(() => {
    if (!applyInvestigationRun) return;
    const updatedRun = resolvedRuns.find((r) => r.id === applyInvestigationRun.id);
    if (updatedRun && updatedRun !== applyInvestigationRun) {
      setApplyInvestigationRun(updatedRun);
    }
  }, [resolvedRuns, applyInvestigationRun]);

  // Clear deselecting guard only after the router has actually processed
  // the navigation (runId becomes undefined). This prevents the URL-sync
  // effect from re-selecting the run due to a stale runId param.
  useEffect(() => {
    if (isDeselectingRef.current && !runId) {
      isDeselectingRef.current = false;
    }
  }, [runId]);

  const handleStop = useCallback(async (runId: string) => {
    if (!confirm("Are you sure you want to stop this run?")) return;
    try {
      await onStopRun(runId);
      onRefresh();
    } catch (err) {
      console.error("Failed to stop run:", err);
    }
  }, [onRefresh, onStopRun]);

  const handleDeleteRequest = useCallback((run: Run) => {
    setDeleteError(null);
    setDeleteConfirmRun(run);
  }, []);

  const handleDeleteConfirm = async () => {
    if (!deleteConfirmRun) return;
    const run = deleteConfirmRun;
    setDeleteLoading(true);
    setDeleteError(null);
    try {
      await onDeleteRun(run.id);
      setDeleteConfirmRun(null);
      if (selectedRun?.id === run.id) {
        setSelectedRun(null);
        navigate("/runs");
      }
    } catch (err) {
      console.error("Failed to delete run:", err);
      setDeleteError((err as Error).message || "Failed to delete run");
    } finally {
      setDeleteLoading(false);
    }
  };

  const handleApprove = async (id: string, req: ApproveFormData) => {
    await onApproveRun(id, req);
    setSelectedRun(null);
    onRefresh();
  };

  const handleReject = async (id: string, req: RejectFormData) => {
    await onRejectRun(id, req);
    setSelectedRun(null);
    onRefresh();
  };

  const handlePartialApprove = async (id: string, fileIds: string[], actor?: string, commitMsg?: string) => {
    const result = await onPartialApproveRun(id, fileIds, actor, commitMsg);
    // Reload to reflect updated approval state
    if (result.remaining === 0) {
      setSelectedRun(null);
    }
    onRefresh();
    return result;
  };

  const handleRetry = async (run: Run) => {
    const newRun = await onRetryRun(run);
    runEventStore.actions.runSnapshotLoaded(newRun);
    loadRunDetails(newRun);
    return newRun;
  };

  const handleInvestigate = async (
    customContext: string,
    depth: "quick" | "standard" | "deep",
    _context?: unknown, // ignored - context flags handled server-side
    projectRoot?: string,
    scopePaths?: string[],
    attachmentIds?: string[],
    overrides?: { roleRef?: string }
  ) => {
    setInvestigateLoading(true);
    setInvestigateError(null);
    try {
      const created = await onInvestigateRuns(
        Array.from(selectedRunIds),
        customContext || undefined,
        depth,
        projectRoot,
        scopePaths,
        attachmentIds,
        overrides
      );
      setInvestigateModalOpen(false);
      clearSelection();
      setSelectionMode(false);
      runEventStore.actions.runSnapshotLoaded(created);
      navigate(`/runs/${created.id}`);
    } catch (err) {
      setInvestigateError((err as Error).message);
    } finally {
      setInvestigateLoading(false);
    }
  };

  const handleInvestigateFromDetail = (runId: string) => {
    setSelectedRunIds(new Set([runId]));
    setInvestigateModalOpen(true);
  };

  const handleApplyInvestigationFromDetail = (runId: string) => {
    // Find the run object from the runs list
    const run = runs.find((r) => r.id === runId);
    if (run) {
      setApplyInvestigationRun(run);
      setApplyModalOpen(true);
    }
  };

  const handleApplyInvestigation = async (customContext: string, attachmentIds?: string[]) => {
    if (!applyInvestigationRun) return;
    setApplyLoading(true);
    setApplyError(null);
    try {
      const created = await onApplyInvestigation(
        applyInvestigationRun.id,
        customContext || undefined,
        attachmentIds
      );
      setApplyModalOpen(false);
      setApplyInvestigationRun(null);
      runEventStore.actions.runSnapshotLoaded(created);
      navigate(`/runs/${created.id}`);
    } catch (err) {
      setApplyError((err as Error).message);
    } finally {
      setApplyLoading(false);
    }
  };

  const handleResumeRequest = useCallback((run: Run) => {
    setResumeError(null);
    setResumeTargetRun(run);
    setResumeModalOpen(true);
  }, []);

  const handleResumeFromFailure = async (customContext: string, attachmentIds?: string[]) => {
    if (!resumeTargetRun) return;
    setResumeLoading(true);
    setResumeError(null);
    try {
      const created = await onResumeFromFailedRun(
        resumeTargetRun.id,
        customContext || undefined,
        attachmentIds
      );
      setResumeModalOpen(false);
      setResumeTargetRun(null);
      runEventStore.actions.runSnapshotLoaded(created);
      onRefresh();
      navigate(`/runs/${created.id}`);
    } catch (err) {
      setResumeError((err as Error).message);
    } finally {
      setResumeLoading(false);
    }
  };

  const filteredAndSortedRuns = useMemo(() => {
    let result = [...resolvedRuns];

    if (statusFilter !== "all") {
      const statusValue = Number(statusFilter) as RunStatus;
      result = result.filter((r) => r.status === statusValue);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((r) => {
        const taskTitle = getTaskTitle(r.taskId).toLowerCase();
        const profileName = getProfileName(r.agentProfileId).toLowerCase();
        return taskTitle.includes(query) || profileName.includes(query) || r.id.toLowerCase().includes(query);
      });
    }

    result.sort((a, b) => {
      const aTime = a.createdAt ? timestampMs(a.createdAt) : 0;
      const bTime = b.createdAt ? timestampMs(b.createdAt) : 0;
      return sortBy === "newest" ? bTime - aTime : aTime - bTime;
    });

    return result;
  }, [resolvedRuns, statusFilter, searchQuery, sortBy, getTaskTitle, getProfileName]);

  const getRunKey = useCallback((run: Run) => run.id, []);
  const handleSelectRun = useCallback(
    (run: Run) => {
      navigate(`/runs/${run.id}`);
      loadRunDetails(run);
    },
    [loadRunDetails, navigate]
  );
  const handleRunRowCheckboxChange = useCallback(
    (runId: string, index: number, shiftKey: boolean) => {
      handleRunCheckboxChange(runId, index, shiftKey, filteredAndSortedRuns);
    },
    [filteredAndSortedRuns, handleRunCheckboxChange]
  );
  const renderRunRow = useCallback(
    (run: Run, index: number) => (
      <RunListRow
        run={run}
        index={index}
        selected={selectedRun?.id === run.id}
        highlighted={selectedRunIds.has(run.id)}
        selectionMode={selectionMode}
        taskTitle={getTaskTitle(run.taskId)}
        profileName={getProfileName(run.agentProfileId)}
        onSelect={handleSelectRun}
        onCheckboxChange={handleRunRowCheckboxChange}
        onStop={handleStop}
        onResume={handleResumeRequest}
        onDelete={handleDeleteRequest}
      />
    ),
    [
      getProfileName,
      getTaskTitle,
      handleDeleteRequest,
      handleResumeRequest,
      handleRunRowCheckboxChange,
      handleSelectRun,
      handleStop,
      selectedRun?.id,
      selectedRunIds,
      selectionMode,
    ]
  );

  useEffect(() => {
    if (!isDesktop) return;
    if (runId) return;
    if (filteredAndSortedRuns.length === 0) return;

    const hasSelection =
      selectedRunId !== null &&
      filteredAndSortedRuns.some((run) => run.id === selectedRunId);

    if (!hasSelection) {
      const firstRun = filteredAndSortedRuns[0];
      if (firstRun) {
        navigate(`/runs/${firstRun.id}`, { replace: true });
        loadRunDetails(firstRun);
      }
    }
  }, [filteredAndSortedRuns, isDesktop, loadRunDetails, navigate, runId, selectedRunId]);

  const filters: FilterConfig[] = [
    {
      id: "status",
      label: "Filter by status",
      value: statusFilter,
      options: STATUS_FILTER_OPTIONS,
      onChange: setStatusFilter,
      allLabel: "All Status",
    },
  ];

  const listPanel = (
    <ListPanel
      title="All Runs"
      count={filteredAndSortedRuns.length}
      loading={loading}
      headerActions={
        <div className="flex gap-2">
          {selectedRunIds.size > 0 && (
            <Button
              variant="default"
              size="sm"
              onClick={() => setInvestigateModalOpen(true)}
              className="gap-1"
            >
              <Search className="h-4 w-4" />
              <span className="hidden sm:inline">Investigate</span> ({selectedRunIds.size})
            </Button>
          )}
          {filteredAndSortedRuns.length > 0 && (
            <Button
              variant={selectionMode ? "default" : "outline"}
              size="sm"
              onClick={toggleSelectionMode}
              aria-label={selectionMode ? "Exit selection mode" : "Enter selection mode"}
              className="w-9 px-0"
            >
              {selectionMode ? (
                <X className="h-4 w-4" />
              ) : (
                <CheckSquare className="h-4 w-4" />
              )}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={onRefresh} aria-label="Refresh" className="w-9 px-0">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      }
      toolbar={
        <SearchToolbar
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="Search runs..."
          filters={filters}
          sortOptions={SORT_OPTIONS}
          currentSort={sortBy}
          onSortChange={setSortBy}
        />
      }
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Activity className="h-12 w-12 mb-3 opacity-50" />
          <p className="font-medium">
            {runs.length === 0 ? "No Runs Yet" : "No Matching Runs"}
          </p>
          <p className="text-sm text-center mt-1">
            {runs.length === 0
              ? "Start a run from the Tasks tab"
              : "Try adjusting your filters"}
          </p>
        </div>
      }
    >
      <BoundedList
        items={filteredAndSortedRuns}
        getKey={getRunKey}
        renderItem={renderRunRow}
      />
    </ListPanel>
  );

  // Determine initial tab: query param > status-based default > "timeline"
  const tabParam = searchParams.get("tab");
  const initialTab = useMemo(() => {
    if (tabParam && VALID_TABS.has(tabParam)) return tabParam as "task" | "timeline" | "diff" | "cost";
    if (selectedRun?.status === RunStatus.NEEDS_REVIEW) return "diff" as const;
    return "timeline" as const;
  }, [tabParam, selectedRun?.status]);

  const detailPanel = (
    <DetailPanel
      title="Run Details"
      hasSelection={!!selectedRun}
      hideHeader
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Eye className="h-12 w-12 mb-3 opacity-50" />
          <p className="text-sm">Select a run to view details</p>
        </div>
      }
    >
      {selectedRun && (
          <RunDetail
            run={selectedRun}
            initialTab={initialTab}
            events={events}
            diff={diff}
            eventsLoading={eventsLoading}
            diffLoading={diffLoading}
            task={getTaskById(selectedRun.taskId)}
            taskTitle={getTaskTitle(selectedRun.taskId)}
            profileName={getProfileName(selectedRun.agentProfileId)}
            onApprove={(req) => handleApprove(selectedRun.id, req)}
            onReject={(req) => handleReject(selectedRun.id, req)}
            onPartialApprove={(fileIds, actor, commitMsg) => handlePartialApprove(selectedRun.id, fileIds, actor, commitMsg)}
            onRetry={handleRetry}
            onResumeFromFailure={handleResumeRequest}
            onInvestigate={handleInvestigateFromDetail}
            onApplyInvestigation={handleApplyInvestigationFromDetail}
            onStop={async (r) => handleStop(r.id)}
            onDelete={handleDeleteRequest}
            onContinue={async (message, attachmentIds) => {
              const updatedRun = await onContinueRun(selectedRun.id, message, attachmentIds);
              runEventStore.actions.runSnapshotLoaded(updatedRun);
              const newEvents = await onGetEvents(selectedRun.id, {
                afterSequence: runEventStore.state.lastSequenceByRunId[selectedRun.id],
              });
              runEventStore.actions.eventsGapFilled(selectedRun.id, newEvents);
            }}
            onDeleteMessage={async (eventId) => {
              await onDeleteRunMessage(selectedRun.id, eventId);
              const newEvents = await onGetEvents(selectedRun.id, {
                afterSequence: runEventStore.state.lastSequenceByRunId[selectedRun.id],
              });
              runEventStore.actions.eventsGapFilled(selectedRun.id, newEvents);
            }}
            deleteLoading={deleteLoading}
            onMobileHeaderLeft={setMobileHeaderLeft}
            onMobileHeaderRight={setMobileHeaderRight}
          />
        )}
      </DetailPanel>
  );

  const headerContent = error ? (
    <Card className="border-destructive/50 bg-destructive/10">
      <CardContent className="flex items-center gap-3 py-4">
        <AlertCircle className="h-4 w-4 text-destructive" />
        <p className="text-sm text-destructive">{error}</p>
      </CardContent>
    </Card>
  ) : null;

  return (
    <>
      <MasterDetailLayout
        storageKey="runs"
        headerContent={headerContent}
        listPanel={listPanel}
        detailPanel={detailPanel}
        selectedId={selectedRun?.id ?? null}
        onDeselect={() => {
          isDeselectingRef.current = true;
          setSelectedRun(null);
          navigate("/runs");
        }}
        detailTitle={selectedRun ? getTaskTitle(selectedRun.taskId) : "Run Details"}
        detailHeaderLeft={selectedRun ? mobileHeaderLeft : undefined}
        detailHeaderRight={selectedRun ? mobileHeaderRight : undefined}
      />

      {/* Investigation Modal */}
      <InvestigateModal
        open={investigateModalOpen}
        onOpenChange={(open) => {
          setInvestigateModalOpen(open);
          if (!open) {
            setInvestigateError(null);
          }
        }}
        title={`Investigate ${selectedRunIds.size} Run${selectedRunIds.size !== 1 ? "s" : ""}`}
        description="Analyze the selected runs to identify issues and recommendations."
        confirmLabel="Start Investigation"
        defaultProjectRoot={(() => {
          // Get project root from the first selected run's task
          const firstRunId = Array.from(selectedRunIds)[0];
          const firstRun = runs.find((r) => r.id === firstRunId);
          if (firstRun) {
            const task = tasks.find((t) => t.id === firstRun.taskId);
            return task?.projectRoot || "";
          }
          return "";
        })()}
        defaultScopePaths={(() => {
          // Get scope paths from the first selected run's task
          const firstRunId = Array.from(selectedRunIds)[0];
          const firstRun = runs.find((r) => r.id === firstRunId);
          if (firstRun) {
            const task = tasks.find((t) => t.id === firstRun.taskId);
            // scopePath may be colon-separated or single
            const scopePath = task?.scopePath || "";
            return scopePath ? scopePath.split(":").filter(Boolean) : [];
          }
          return [];
        })()}
        onSubmit={handleInvestigate}
        loading={investigateLoading}
        error={investigateError}
      />

      <ApplyInvestigationModal
        open={applyModalOpen}
        onOpenChange={(open) => {
          setApplyModalOpen(open);
          if (!open) {
            setApplyInvestigationRun(null);
            setApplyError(null);
          }
        }}
        investigationRun={applyInvestigationRun}
        onSubmit={handleApplyInvestigation}
        loading={applyLoading}
        error={applyError}
      />

      <ResumeFromFailureModal
        open={resumeModalOpen}
        onOpenChange={(open) => {
          setResumeModalOpen(open);
          if (!open) {
            setResumeTargetRun(null);
            setResumeError(null);
          }
        }}
        failedRun={resumeTargetRun}
        onSubmit={handleResumeFromFailure}
        loading={resumeLoading}
        error={resumeError}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={!!deleteConfirmRun}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteConfirmRun(null);
            setDeleteError(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Run</DialogTitle>
            <DialogDescription>
              Delete this run? This removes its history and events. This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          {deleteError && (
            <div className="mx-4 sm:mx-6 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {deleteError}
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setDeleteConfirmRun(null);
                setDeleteError(null);
              }}
              disabled={deleteLoading}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteConfirm}
              disabled={deleteLoading}
            >
              {deleteLoading ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function runStatusTooltip(status: RunStatus): string {
  switch (status) {
    case RunStatus.COMPLETE:
      return "Complete";
    case RunStatus.FAILED:
      return "Failed";
    case RunStatus.RUNNING:
      return "Running";
    case RunStatus.STARTING:
      return "Starting";
    case RunStatus.NEEDS_REVIEW:
      return "Needs Review";
    case RunStatus.CANCELLED:
      return "Cancelled";
    case RunStatus.PENDING:
      return "Pending";
    default:
      return "Pending";
  }
}

function RunStatusIcon({ status, approvalState }: { status: RunStatus; approvalState?: ApprovalState }) {
  // Rejected runs get distinct orange icon
  if (approvalState === ApprovalState.REJECTED) {
    return <span className="flex-shrink-0" title="Rejected"><Ban className="h-5 w-5 text-orange-500" /></span>;
  }

  const tooltip = runStatusTooltip(status);
  let icon: JSX.Element;
  switch (status) {
    case RunStatus.COMPLETE:
      icon = <Check className="h-5 w-5 text-success" />;
      break;
    case RunStatus.FAILED:
      icon = <XCircle className="h-5 w-5 text-destructive" />;
      break;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      icon = <Activity className="h-5 w-5 text-primary animate-pulse" />;
      break;
    case RunStatus.NEEDS_REVIEW:
      icon = <Clock className="h-5 w-5 text-warning" />;
      break;
    default:
      icon = <Clock className="h-5 w-5 text-muted-foreground" />;
      break;
  }
  return <span className="flex-shrink-0" title={tooltip}>{icon}</span>;
}
