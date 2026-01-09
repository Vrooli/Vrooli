import { useCallback, useEffect, useMemo, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useParams, useNavigate } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  Check,
  Clock,
  Eye,
  Play,
  RefreshCw,
  Search,
  Square,
  Trash2,
  X,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
import { formatRelativeTime } from "../lib/utils";
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
import { InvestigateModal } from "../components/InvestigateModal";
import { RunDetail } from "../components/RunDetail";
import { useViewportSize } from "../hooks/useViewportSize";

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
  onGetEvents: (id: string) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<RunDiff>;
  onApproveRun: (id: string, req: ApproveFormData) => Promise<ApproveResult>;
  onRejectRun: (id: string, req: RejectFormData) => Promise<void>;
  onInvestigateRuns: (runIds: string[], customContext?: string, depth?: "quick" | "standard" | "deep") => Promise<Run>;
  onApplyInvestigation: (investigationRunId: string, customContext?: string) => Promise<Run>;
  onContinueRun: (id: string, message: string) => Promise<Run>;
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
  onGetEvents,
  onGetDiff,
  onApproveRun,
  onRejectRun,
  onInvestigateRuns,
  onApplyInvestigation,
  onContinueRun,
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
  const [events, setEvents] = useState<RunEvent[]>([]);
  const [diff, setDiff] = useState<RunDiff | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [diffLoading, setDiffLoading] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

  // Multi-select state for investigations
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedRunIds, setSelectedRunIds] = useState<Set<string>>(new Set());
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null);
  const [investigateModalOpen, setInvestigateModalOpen] = useState(false);
  const [investigateLoading, setInvestigateLoading] = useState(false);
  const [investigateError, setInvestigateError] = useState<string | null>(null);
  const [applyModalOpen, setApplyModalOpen] = useState(false);
  const [applyInvestigationId, setApplyInvestigationId] = useState<string | null>(null);
  const [applyLoading, setApplyLoading] = useState(false);
  const [applyError, setApplyError] = useState<string | null>(null);

  const getTaskTitle = useCallback(
    (taskId: string) => tasks.find((t) => t.id === taskId)?.title || "Unknown Task",
    [tasks]
  );

  const getTaskById = useCallback(
    (taskId: string) => tasks.find((t) => t.id === taskId) ?? null,
    [tasks]
  );

  const getProfileName = useCallback(
    (profileId?: string) =>
      profileId ? profiles.find((p) => p.id === profileId)?.name || "Unknown Profile" : "Unknown Profile",
    [profiles]
  );

  // Subscribe to WebSocket events for the selected run
  useEffect(() => {
    if (!selectedRun) return;
    wsSubscribe(selectedRun.id);
    return () => {
      wsUnsubscribe(selectedRun.id);
    };
  }, [selectedRun?.id, wsSubscribe, wsUnsubscribe]);

  // Handle WebSocket messages for real-time updates
  useEffect(() => {
    const handleMessage: MessageHandler = (message: WebSocketMessage) => {
      if (!selectedRun || message.runId !== selectedRun.id) return;

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
          break;
        }
      }
    };

    wsAddMessageHandler(handleMessage);
    return () => {
      wsRemoveMessageHandler(handleMessage);
    };
  }, [selectedRun?.id, wsAddMessageHandler, wsRemoveMessageHandler]);

  const loadRunDetails = useCallback(
    async (run: Run) => {
      setSelectedRun(run);
      setEvents([]);
      setDiff(null);

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
    [onGetEvents, onGetDiff]
  );

  // Sync selectedRun with the runs list when it updates
  useEffect(() => {
    if (!selectedRun) return;
    const updatedRun = runs.find((r) => r.id === selectedRun.id);
    if (updatedRun && updatedRun !== selectedRun) {
      setSelectedRun(updatedRun);
    }
  }, [runs, selectedRun]);

  // Load run from URL params when component mounts or runId changes
  useEffect(() => {
    if (!runId || runs.length === 0) return;
    if (selectedRun?.id === runId) return;
    const run = runs.find((r) => r.id === runId);
    if (run) {
      loadRunDetails(run);
    }
  }, [runId, runs, selectedRun?.id, loadRunDetails]);

  const handleStop = async (runId: string) => {
    if (!confirm("Are you sure you want to stop this run?")) return;
    try {
      await onStopRun(runId);
      onRefresh();
    } catch (err) {
      console.error("Failed to stop run:", err);
    }
  };

  const canDeleteRun = (run: Run | null): boolean => {
    if (!run) return false;
    return ![RunStatus.PENDING, RunStatus.STARTING, RunStatus.RUNNING].includes(run.status);
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

  // Multi-select handlers
  const handleRunCheckboxChange = (runId: string, index: number, shiftKey: boolean) => {
    setSelectedRunIds((prev) => {
      const next = new Set(prev);

      if (shiftKey && lastClickedIndex !== null) {
        const start = Math.min(lastClickedIndex, index);
        const end = Math.max(lastClickedIndex, index);
        for (let i = start; i <= end; i++) {
          next.add(filteredAndSortedRuns[i].id);
        }
      } else {
        if (next.has(runId)) {
          next.delete(runId);
        } else {
          next.add(runId);
        }
      }

      return next;
    });
    setLastClickedIndex(index);
  };

  const toggleSelectionMode = () => {
    if (selectionMode) {
      setSelectedRunIds(new Set());
      setLastClickedIndex(null);
    }
    setSelectionMode(!selectionMode);
  };

  const handleInvestigate = async (customContext: string, depth: "quick" | "standard" | "deep") => {
    setInvestigateLoading(true);
    setInvestigateError(null);
    try {
      const created = await onInvestigateRuns(Array.from(selectedRunIds), customContext || undefined, depth);
      setInvestigateModalOpen(false);
      setSelectedRunIds(new Set());
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
    setApplyInvestigationId(runId);
    setApplyModalOpen(true);
  };

  const handleApplyInvestigation = async (customContext: string) => {
    if (!applyInvestigationId) return;
    setApplyLoading(true);
    setApplyError(null);
    try {
      const created = await onApplyInvestigation(applyInvestigationId, customContext || undefined);
      setApplyModalOpen(false);
      setApplyInvestigationId(null);
      navigate(`/runs/${created.id}`);
    } catch (err) {
      setApplyError((err as Error).message);
    } finally {
      setApplyLoading(false);
    }
  };

  const filteredAndSortedRuns = useMemo(() => {
    let result = [...runs];

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
  }, [runs, statusFilter, searchQuery, sortBy, getTaskTitle, getProfileName]);

  useEffect(() => {
    if (!isDesktop) return;
    if (runId) return;
    if (filteredAndSortedRuns.length === 0) return;

    const selectedRunId = selectedRun?.id ?? null;
    const hasSelection =
      selectedRunId !== null &&
      filteredAndSortedRuns.some((run) => run.id === selectedRunId);

    if (!hasSelection) {
      const firstRun = filteredAndSortedRuns[0];
      navigate(`/runs/${firstRun.id}`, { replace: true });
      loadRunDetails(firstRun);
    }
  }, [filteredAndSortedRuns, isDesktop, loadRunDetails, navigate, runId, selectedRun?.id]);

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
                  handleRunCheckboxChange(run.id, index, e.nativeEvent instanceof MouseEvent && e.nativeEvent.shiftKey);
                }}
                onClick={(e) => e.stopPropagation()}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
            ) : undefined
          }
          icon={<RunStatusIcon status={run.status} />}
          actions={
            <div className="flex items-center gap-1">
              <Badge
                variant={
                  runStatusLabel(run.status) as
                    | "pending"
                    | "starting"
                    | "running"
                    | "needs_review"
                    | "complete"
                    | "failed"
                    | "cancelled"
                }
              >
                {runStatusLabel(run.status).replace("_", " ")}
              </Badge>
              {(run.status === RunStatus.RUNNING || run.status === RunStatus.STARTING) && (
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
              {canDeleteRun(run) && (
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
            {getProfileName(run.agentProfileId)} | {formatRelativeTime(run.createdAt)}
          </ListItemSubtitle>
        </ListItem>
      ))}
    </ListPanel>
  );

  const detailPanel = (
    <DetailPanel
      title="Run Details"
      hasSelection={!!selectedRun}
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
          onDelete={handleDelete}
          onContinue={async (message) => {
            await onContinueRun(selectedRun.id, message);
            // Reload events to show the new messages
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
          setSelectedRun(null);
          navigate("/runs");
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
        onSubmit={handleInvestigate}
        loading={investigateLoading}
        error={investigateError}
      />

      <InvestigateModal
        open={applyModalOpen}
        onOpenChange={(open) => {
          setApplyModalOpen(open);
          if (!open) {
            setApplyInvestigationId(null);
            setApplyError(null);
          }
        }}
        title="Apply Investigation Fixes"
        description="Apply recommendations from the selected investigation run."
        confirmLabel="Apply Fixes"
        onSubmit={handleApplyInvestigation}
        loading={applyLoading}
        error={applyError}
      />
    </>
  );
}

function runStatusLabel(status: RunStatus): string {
  switch (status) {
    case RunStatus.PENDING:
      return "pending";
    case RunStatus.STARTING:
      return "starting";
    case RunStatus.RUNNING:
      return "running";
    case RunStatus.NEEDS_REVIEW:
      return "needs_review";
    case RunStatus.COMPLETE:
      return "complete";
    case RunStatus.FAILED:
      return "failed";
    case RunStatus.CANCELLED:
      return "cancelled";
    default:
      return "pending";
  }
}

function RunStatusIcon({ status }: { status: RunStatus }) {
  switch (status) {
    case RunStatus.COMPLETE:
      return <Check className="h-5 w-5 text-success flex-shrink-0" />;
    case RunStatus.FAILED:
      return <XCircle className="h-5 w-5 text-destructive flex-shrink-0" />;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      return <Activity className="h-5 w-5 text-primary animate-pulse flex-shrink-0" />;
    case RunStatus.NEEDS_REVIEW:
      return <Clock className="h-5 w-5 text-warning flex-shrink-0" />;
    default:
      return <Clock className="h-5 w-5 text-muted-foreground flex-shrink-0" />;
  }
}
