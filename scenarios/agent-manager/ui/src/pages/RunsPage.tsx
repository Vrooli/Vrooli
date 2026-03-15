import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useParams, useNavigate } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  Check,
  Clock,
  Eye,
  RefreshCw,
  Search,
  Square,
  Trash2,
  X,
  XCircle,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
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
import type { MessageHandler, WebSocketMessage } from "../hooks/useWebSocket";
import { ApplyInvestigationModal } from "../components/ApplyInvestigationModal";
import { InvestigateModal } from "../components/InvestigateModal";
import { RunDetail } from "../components/RunDetail";
import { useViewportSize } from "../hooks/useViewportSize";
import { useRunsPageState } from "../hooks/useRunsPageState";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";

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
  onGetEvents: (id: string) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<RunDiff>;
  onGetTask: (id: string) => Promise<Task>;
  onApproveRun: (id: string, req: ApproveFormData) => Promise<ApproveResult>;
  onRejectRun: (id: string, req: RejectFormData) => Promise<void>;
  onInvestigateRuns: (runIds: string[], customContext?: string, depth?: "quick" | "standard" | "deep", projectRoot?: string, scopePaths?: string[]) => Promise<Run>;
  onApplyInvestigation: (investigationRunId: string, customContext?: string) => Promise<Run>;
  onContinueRun: (id: string, message: string, attachmentIds?: string[]) => Promise<Run>;
  onDeleteRunMessage: (runId: string, eventId: string) => Promise<void>;
  onRefresh: () => void;
  wsSubscribe: (runId: string) => void;
  wsUnsubscribe: (runId: string) => void;
  wsAddMessageHandler: (handler: MessageHandler) => void;
  wsRemoveMessageHandler: (handler: MessageHandler) => void;
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
  onInvestigateRuns,
  onApplyInvestigation,
  onContinueRun,
  onDeleteRunMessage,
  onRefresh,
  wsSubscribe,
  wsUnsubscribe,
  wsAddMessageHandler,
  wsRemoveMessageHandler,
}: RunsPageProps) {
  const { runId } = useParams<{ runId?: string }>();
  const navigate = useNavigate();
  const { isDesktop } = useViewportSize();
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const isDeselectingRef = useRef(false);
  const [events, setEvents] = useState<RunEvent[]>([]);
  const [diff, setDiff] = useState<RunDiff | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [diffLoading, setDiffLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [runOverrides, setRunOverrides] = useState<Record<string, Run>>({});
  const [extraTasks, setExtraTasks] = useState<Record<string, Task>>({});

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

  const getTaskById = useCallback(
    (taskId: string) => extraTasks[taskId] ?? tasks.find((t) => t.id === taskId) ?? null,
    [extraTasks, tasks]
  );

  const getTaskTitle = useCallback(
    (taskId: string) => getTaskById(taskId)?.title || "Unknown Task",
    [getTaskById]
  );

  const getProfileName = useCallback(
    (profileId?: string) =>
      profileId ? profiles.find((p) => p.id === profileId)?.name || "Unknown Profile" : "Unknown Profile",
    [profiles]
  );

  const resolvedRuns = useMemo(
    () => runs.map((run) => runOverrides[run.id] ?? run),
    [runs, runOverrides]
  );

  const syncRunDetails = useCallback(
    async (runId: string) => {
      try {
        const latest = await onGetRun(runId);
        setRunOverrides((prev) => ({ ...prev, [runId]: latest }));
        setSelectedRun((prev) => (prev && prev.id === runId ? { ...prev, ...latest } : prev));

        if (!getTaskById(latest.taskId)) {
          const task = await onGetTask(latest.taskId);
          setExtraTasks((prev) => ({ ...prev, [task.id]: task }));
        }
      } catch (err) {
        console.error("Failed to sync run details:", err);
      }
    },
    [getTaskById, onGetRun, onGetTask]
  );

  // Extract IDs as stable primitives to avoid unnecessary effect re-runs
  const selectedRunId = selectedRun?.id ?? null;
  const applyInvestigationRunId = applyInvestigationRun?.id ?? null;

  // Subscribe to WebSocket events for the selected run
  useEffect(() => {
    if (!selectedRunId) return;
    wsSubscribe(selectedRunId);
    return () => {
      wsUnsubscribe(selectedRunId);
    };
  }, [selectedRunId, wsSubscribe, wsUnsubscribe]);

  // Subscribe to WebSocket events for the investigation run when Apply modal is open
  // This ensures we get updates when recommendation extraction completes
  useEffect(() => {
    if (!applyInvestigationRunId || !applyModalOpen) return;
    wsSubscribe(applyInvestigationRunId);
    return () => {
      wsUnsubscribe(applyInvestigationRunId);
    };
  }, [applyInvestigationRunId, applyModalOpen, wsSubscribe, wsUnsubscribe]);

  // Handle WebSocket messages for real-time updates
  useEffect(() => {
    const handleMessage: MessageHandler = (message: WebSocketMessage) => {
      // Handle messages for selectedRun
      if (selectedRunId && message.runId === selectedRunId) {
        switch (message.type) {
          case "run_event": {
            const newEvent = message.payload as RunEvent;
            setEvents((prev) => {
              if (prev.some((e) => e.id === newEvent.id || e.sequence === newEvent.sequence)) {
                return prev;
              }
              return [...prev, newEvent];
            });
            break;
          }
          case "run_status": {
            const statusUpdate = message.payload as Partial<Run>;
            setSelectedRun((prev) => (prev ? { ...prev, ...statusUpdate } : null));
            const runId = statusUpdate.id;
            if (runId) {
              setRunOverrides((prev) => {
                const existing = prev[runId];
                if (!existing) return prev;
                return { ...prev, [runId]: { ...existing, ...statusUpdate } as Run };
              });
            }
            // Unsubscribe once run reaches terminal state
            const isTerminal = statusUpdate.status !== undefined &&
              [RunStatus.COMPLETE, RunStatus.FAILED, RunStatus.CANCELLED, RunStatus.NEEDS_REVIEW].includes(statusUpdate.status);
            if (isTerminal && selectedRunId) {
              wsUnsubscribe(selectedRunId);
            }
            break;
          }
        }
      }

      // Handle messages for applyInvestigationRun (for recommendation extraction updates)
      if (applyInvestigationRunId && message.runId === applyInvestigationRunId) {
        if (message.type === "run_status") {
          const statusUpdate = message.payload as Partial<Run>;
          setApplyInvestigationRun((prev) => (prev ? { ...prev, ...statusUpdate } : null));
          // Unsubscribe once investigation run reaches terminal state
          const isTerminal = statusUpdate.status !== undefined &&
            [RunStatus.COMPLETE, RunStatus.FAILED, RunStatus.CANCELLED, RunStatus.NEEDS_REVIEW].includes(statusUpdate.status);
          if (isTerminal) {
            wsUnsubscribe(applyInvestigationRunId);
          }
        }
      }
    };

    wsAddMessageHandler(handleMessage);
    return () => {
      wsRemoveMessageHandler(handleMessage);
    };
  }, [selectedRunId, applyInvestigationRunId, wsAddMessageHandler, wsRemoveMessageHandler, wsUnsubscribe]);

  const loadRunDetails = useCallback(
    async (run: Run) => {
      setSelectedRun(run);
      setEvents([]);
      setDiff(null);
      void syncRunDetails(run.id);

      setEventsLoading(true);
      try {
        const evts = await onGetEvents(run.id);
        setEvents(evts || []);
      } catch (err) {
        console.error("Failed to load events:", err);
      } finally {
        setEventsLoading(false);
      }

      if (
        run.status === RunStatus.NEEDS_REVIEW ||
        run.status === RunStatus.COMPLETE ||
        run.approvalState !== ApprovalState.NONE
      ) {
        setDiffLoading(true);
        try {
          const diffResult = await onGetDiff(run.id);
          setDiff(diffResult);
        } catch (err) {
          console.error("Failed to load diff:", err);
        } finally {
          setDiffLoading(false);
        }
      }
    },
    [onGetEvents, onGetDiff, syncRunDetails]
  );

  // Sync selectedRun with the runs list when it updates
  useEffect(() => {
    if (!selectedRun) return;
    const updatedRun = resolvedRuns.find((r) => r.id === selectedRun.id);
    if (updatedRun && updatedRun !== selectedRun) {
      setSelectedRun(updatedRun);
    }
  }, [resolvedRuns, selectedRun]);

  // Sync applyInvestigationRun with the runs list when it updates
  // This ensures the modal sees WebSocket updates (e.g., recommendation extraction status)
  useEffect(() => {
    if (!applyInvestigationRun) return;
    const updatedRun = runs.find((r) => r.id === applyInvestigationRun.id);
    if (updatedRun && updatedRun !== applyInvestigationRun) {
      setApplyInvestigationRun(updatedRun);
    }
  }, [runs, applyInvestigationRun]);

  // Load run from URL params when component mounts or runId changes
  useEffect(() => {
    if (isDeselectingRef.current) return;
    if (!runId || resolvedRuns.length === 0) return;
    if (selectedRunId === runId) return;
    const run = resolvedRuns.find((r) => r.id === runId);
    if (run) {
      loadRunDetails(run);
    }
  }, [runId, resolvedRuns, selectedRunId, loadRunDetails]);

  const handleStop = async (runId: string) => {
    if (!confirm("Are you sure you want to stop this run?")) return;
    try {
      await onStopRun(runId);
      onRefresh();
    } catch (err) {
      console.error("Failed to stop run:", err);
    }
  };

  const handleDelete = async (run: Run) => {
    if (!confirm("Delete this run? This removes its history and events.")) return;
    setDeleteLoading(true);
    try {
      await onDeleteRun(run.id);
      if (selectedRun?.id === run.id) {
        setSelectedRun(null);
        navigate("/runs");
      }
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

  const handleRetry = async (run: Run) => {
    const newRun = await onRetryRun(run);
    loadRunDetails(newRun);
    return newRun;
  };

  const handleInvestigate = async (
    customContext: string,
    depth: "quick" | "standard" | "deep",
    _context?: unknown, // ignored - context flags handled server-side
    projectRoot?: string,
    scopePaths?: string[]
  ) => {
    setInvestigateLoading(true);
    setInvestigateError(null);
    try {
      const created = await onInvestigateRuns(
        Array.from(selectedRunIds),
        customContext || undefined,
        depth,
        projectRoot,
        scopePaths
      );
      setInvestigateModalOpen(false);
      clearSelection();
      setSelectionMode(false);
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

  const handleApplyInvestigation = async (customContext: string) => {
    if (!applyInvestigationRun) return;
    setApplyLoading(true);
    setApplyError(null);
    try {
      const created = await onApplyInvestigation(applyInvestigationRun.id, customContext || undefined);
      setApplyModalOpen(false);
      setApplyInvestigationRun(null);
      navigate(`/runs/${created.id}`);
    } catch (err) {
      setApplyError((err as Error).message);
    } finally {
      setApplyLoading(false);
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
              className="gap-1"
            >
              {selectionMode ? (
                <>
                  <X className="h-3 w-3" />
                  <span className="hidden sm:inline">Done</span>
                </>
              ) : (
                "Select"
              )}
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={onRefresh}>
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
      {filteredAndSortedRuns.map((run, index) => (
        <ListItem
          key={run.id}
          selected={selectedRun?.id === run.id}
          highlighted={selectedRunIds.has(run.id)}
          onClick={() => {
            navigate(`/runs/${run.id}`);
            loadRunDetails(run);
          }}
          checkbox={
            selectionMode ? (
              <input
                type="checkbox"
                checked={selectedRunIds.has(run.id)}
                onChange={(e) => {
                  e.stopPropagation();
                  handleRunCheckboxChange(
                    run.id,
                    index,
                    e.nativeEvent instanceof MouseEvent && e.nativeEvent.shiftKey,
                    filteredAndSortedRuns
                  );
                }}
                onClick={(e) => e.stopPropagation()}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
            ) : undefined
          }
          icon={<RunStatusIcon status={run.status} />}
          actions={
            <div className="flex items-center gap-1">
              {run.actions?.canStop && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  aria-label={`Stop run ${getTaskTitle(run.taskId)}`}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleStop(run.id);
                  }}
                >
                  <Square className="h-3 w-3" />
                </Button>
              )}
              {run.actions?.canDelete && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 text-destructive hover:text-destructive"
                  aria-label={`Delete run ${getTaskTitle(run.taskId)}`}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDelete(run);
                  }}
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              )}
            </div>
          }
        >
          <ListItemTitle>{getTaskTitle(run.taskId)}</ListItemTitle>
          <ListItemSubtitle>
            {getProfileName(run.agentProfileId)} | {formatStandardRelativeTime(run.createdAt)}
          </ListItemSubtitle>
        </ListItem>
      ))}
    </ListPanel>
  );

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
            events={events}
            diff={diff}
            eventsLoading={eventsLoading}
            diffLoading={diffLoading}
            task={getTaskById(selectedRun.taskId)}
            taskTitle={getTaskTitle(selectedRun.taskId)}
            profileName={getProfileName(selectedRun.agentProfileId)}
            onApprove={(req) => handleApprove(selectedRun.id, req)}
            onReject={(req) => handleReject(selectedRun.id, req)}
            onRetry={handleRetry}
            onInvestigate={handleInvestigateFromDetail}
            onApplyInvestigation={handleApplyInvestigationFromDetail}
            onStop={async (r) => handleStop(r.id)}
            onDelete={handleDelete}
            onContinue={async (message, attachmentIds) => {
              await onContinueRun(selectedRun.id, message, attachmentIds);
              // Reload events to show the new messages
              const newEvents = await onGetEvents(selectedRun.id);
              setEvents(newEvents);
            }}
            onDeleteMessage={async (eventId) => {
              await onDeleteRunMessage(selectedRun.id, eventId);
              const newEvents = await onGetEvents(selectedRun.id);
              setEvents(newEvents);
            }}
            deleteLoading={deleteLoading}
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
          // Clear after React processes the navigation
          requestAnimationFrame(() => { isDeselectingRef.current = false; });
        }}
        detailTitle={selectedRun ? getTaskTitle(selectedRun.taskId) : "Run Details"}
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

function RunStatusIcon({ status }: { status: RunStatus }) {
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
