import type React from "react";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useParams, useNavigate, useSearchParams } from "react-router-dom";
import {
  Activity,
  Ban,
  Check,
  CheckSquare,
  Clock,
  Eye,
  PlayCircle,
  RefreshCw,
  Search,
  Square,
  X,
  XCircle,
} from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { useConfirmAction } from "../components/feedback/useConfirmAction";
import { runHeading, runOrigin } from "../lib/runPresentation";
import { useSubAgentRuns, type RunOrigin } from "../hooks/useApi";
import { formatCount } from "../lib/format";
import { formatStandardRelativeTime } from "../lib/dateTime";
import type {
  AgentProfile,
  ApproveResult,
  Run,
  RunDiff,
  RunEvent,
  Task,
  TypedInvestigation,
} from "../types";
import { ApprovalState, ExecutionMode, RunStatus } from "../types";
import type { UseRunEventStoreReturn } from "../hooks/useRunEventStore";
import { ApplyInvestigationModal } from "../components/ApplyInvestigationModal";
import { InvestigateModal } from "../components/InvestigateModal";
import { ResumeFromFailureModal } from "../components/ResumeFromFailureModal";
import { RunDetail } from "../components/RunDetail";
import { isRunTab, type RunTabId } from "../components/runTabs";
import { useViewportSize } from "../hooks/useViewportSize";
import { useRunsPageState } from "../hooks/useRunsPageState";
import { useSelectedRunController } from "../hooks/useSelectedRunController";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { BoundedList, ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";
import { ConversationSearchResults } from "../features/conversation-search/ConversationSearchResults";
import {
  DEFAULT_CONVERSATION_FILTERS,
  type ConversationSearchFiltersState,
} from "../features/conversation-search/useConversationSearch";
import type { ConversationSearchHit } from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";

interface RunsPageProps {
  runs: Run[];
  /** Who started the listed runs; the list is filtered by the API. */
  origin: RunOrigin;
  onOriginChange: (origin: RunOrigin) => void;
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
  onPartialApproveRun: (id: string, fileIds: string[], actor?: string, commitMsg?: string) => Promise<ApproveResult>;
  onStartInvestigation: (
    runIds: string[],
    customContext?: string,
    depth?: "quick" | "standard" | "deep",
  ) => Promise<TypedInvestigation>;
  onApplyInvestigation: (
    investigationRunId: string,
    selected: string[],
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

const ORIGIN_CHIPS: ReadonlyArray<{ value: RunOrigin; label: string }> = [
  { value: "", label: "All" },
  { value: "agent_manager", label: "Agent Manager" },
  { value: "operator_sessions", label: "Operator sessions" },
  { value: "workflows", label: "Workflows" },
];

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
];


interface RunListRowProps {
  run: Run;
  index: number;
  selected: boolean;
  highlighted: boolean;
  selectionMode: boolean;
  taskTitle: string;
  profileName?: string;
  /** The run open in the detail pane, so an expanded sub-agent can show as selected. */
  selectedRunId?: string;
  onSelect: (run: Run) => void;
  onCheckboxChange: (runId: string, index: number, shiftKey: boolean) => void;
  onStop: (run: Run) => void;
  onResume: (run: Run) => void;
}

/** "▸ N sub-agents" under a run; opening it lists them in place. Nothing renders for a run without sub-agents. */
function SubAgents({ parent, selectedRunId, onSelect }: { parent: Run; selectedRunId?: string; onSelect: (run: Run) => void }) {
  const [expanded, setExpanded] = useState(false);
  const children = useSubAgentRuns(parent.id, expanded);
  const count = parent.subAgentCount;
  if (count === 0) return null;
  return (
    <div onClick={(event) => event.stopPropagation()}>
      <button
        type="button"
        aria-expanded={expanded}
        data-testid="sub-agents-toggle"
        onClick={() => setExpanded((open) => !open)}
        className="mt-1 inline-flex min-h-11 items-center text-xs font-medium text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-6"
      >
        {expanded ? "▾" : "▸"} {formatCount(count)} {count === 1 ? "sub-agent" : "sub-agents"}
      </button>
      {expanded && (
        <ul className="ms-2 border-s border-border ps-3">
          {children.loading && !children.data?.length ? <li className="py-1 text-xs text-muted-foreground">Loading sub-agents…</li> : null}
          {children.error ? <li className="py-1 text-xs text-destructive">Could not load sub-agents: {children.error}</li> : null}
          {(children.data ?? []).map((child) => (
            <li key={child.id}>
              <button
                type="button"
                data-testid="sub-agent-row"
                aria-current={selectedRunId === child.id ? "true" : undefined}
                onClick={() => onSelect(child)}
                className={`flex min-h-11 w-full items-center gap-2 rounded px-2 text-start text-sm hover:bg-muted/50 sm:min-h-8 ${selectedRunId === child.id ? "bg-muted" : ""}`}
              >
                <RunStatusIcon status={child.status} approvalState={child.approvalState} />
                <span className="min-w-0 flex-1 truncate">{runHeading(child).title}</span>
                <span className="shrink-0 text-xs text-muted-foreground">{formatStandardRelativeTime(child.createdAt)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

const RunListRow = memo(function RunListRow({
  run,
  index,
  selected,
  highlighted,
  selectionMode,
  taskTitle,
  profileName,
  selectedRunId,
  onSelect,
  onCheckboxChange,
  onStop,
  onResume,
}: RunListRowProps) {
  const { title: runTitle, reference } = runHeading(run, taskTitle);
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
              className="h-11 w-11 sm:h-8 sm:w-8"
              aria-label={`Stop run ${runTitle}`}
              onClick={(e) => {
                e.stopPropagation();
                onStop(run);
              }}
            >
              <Square className="h-3.5 w-3.5" />
            </Button>
          )}
          {run.actions?.canResumeFromFailure && (
            <Button
              variant="ghost"
              size="icon"
              className="h-11 w-11 sm:h-8 sm:w-8"
                aria-label={`Resume run ${runTitle} from failure`}
              title="Resume: continue this task with the prior transcript + diff as context"
              onClick={(e) => {
                e.stopPropagation();
                onResume(run);
              }}
            >
              <PlayCircle className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      }
    >
      <ListItemTitle>
        {runTitle}
        {run.executionMode === ExecutionMode.INTERACTIVE && (
          <span
            data-testid="interactive-badge"
            className="ms-2 inline-flex items-center rounded bg-primary/15 px-1.5 py-0.5 align-middle text-[10px] font-medium uppercase tracking-wide text-primary"
          >
            interactive
          </span>
        )}
      </ListItemTitle>
      <ListItemSubtitle>
        {runOrigin(run, profileName)} · {formatStandardRelativeTime(run.createdAt)}
        <span className="ms-2 font-mono text-[11px] opacity-70">{reference}</span>
      </ListItemSubtitle>
      <SubAgents parent={run} selectedRunId={selectedRunId} onSelect={onSelect} />
    </ListItem>
  );
});

export function RunsPage({
  runs,
  origin,
  onOriginChange,
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
  onPartialApproveRun,
  onStartInvestigation,
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
  const [mobileHeaderLeft, setMobileHeaderLeft] = useState<React.ReactNode>(null);
  const [mobileHeaderRight, setMobileHeaderRight] = useState<React.ReactNode>(null);
  const [conversationFilters, setConversationFilters] = useState<ConversationSearchFiltersState>(DEFAULT_CONVERSATION_FILTERS);
  const [conversationResultCount, setConversationResultCount] = useState(0);

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
    (profileId?: string) => (profileId ? profileNameById.get(profileId) : undefined),
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

  const runActions = useConfirmAction("run-action");
  const confirmRunAction = runActions.confirm;
  const handleStopRequest = useCallback((run: Run) => {
    confirmRunAction({
      title: "Stop run?",
      description: "The agent is interrupted now. Work it already wrote stays in its sandbox for review.",
      confirmLabel: "Stop run",
      busyLabel: "Stopping…",
      subject: runHeading(run).title,
      action: async () => {
        await onStopRun(run.id);
        onRefresh();
      },
    });
  }, [confirmRunAction, onRefresh, onStopRun]);

  const handleDeleteRequest = useCallback((run: Run) => {
    confirmRunAction({
      title: "Delete run?",
      description: "This removes the run's history and events. It cannot be undone.",
      confirmLabel: "Delete run",
      busyLabel: "Deleting…",
      subject: runHeading(run).title,
      action: async () => {
        await onDeleteRun(run.id);
        if (selectedRunId === run.id) {
          setSelectedRun(null);
          navigate("/runs");
        }
      },
    });
  }, [confirmRunAction, navigate, onDeleteRun, selectedRunId, setSelectedRun]);

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
    _context?: unknown // typed lifecycle owns bounded evidence selection
  ) => {
    setInvestigateLoading(true);
    setInvestigateError(null);
    try {
      const created = await onStartInvestigation(
        Array.from(selectedRunIds),
        customContext || undefined,
        depth,
      );
      setInvestigateModalOpen(false);
      clearSelection();
      setSelectionMode(false);
      navigate(`/investigations?runId=${encodeURIComponent(created.investigationId)}`);
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

  const handleApplyInvestigation = async (selected: string[], customContext: string, attachmentIds?: string[]) => {
    if (!applyInvestigationRun) return;
    setApplyLoading(true);
    setApplyError(null);
    try {
      const created = await onApplyInvestigation(
        applyInvestigationRun.id,
        selected,
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
    const modelFilter = searchParams.get("model") ?? "";
    const runnerFilter = searchParams.get("runnerType") ?? "";
    const profileFilter = searchParams.get("profileId") ?? "";
    const tagFilter = searchParams.get("tagPrefix") ?? "";
    const workloadFilter = searchParams.get("workloadKey") ?? "";
    const errorFilter = searchParams.get("errorCode") ?? "";
    const fromFilter = searchParams.get("from");
    const toFilter = searchParams.get("to");

    if (statusFilter !== "all") {
      const statusValue = Number(statusFilter) as RunStatus;
      result = result.filter((r) => r.status === statusValue);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter((r) => {
        const taskTitle = getTaskTitle(r.taskId);
        return [r.label, taskTitle, runHeading(r, taskTitle).title, runOrigin(r, getProfileName(r.agentProfileId)), r.id]
          .some((value) => value.toLowerCase().includes(query));
      });
    }

    result = result.filter((run) => {
      const created = run.createdAt ? timestampMs(run.createdAt) : 0;
      const runner = run.resolvedConfig?.runnerType ?? "";
      const model = run.actualModel || run.requestedModel || "";
      const workload = (run as Run & { workloadKey?: string }).workloadKey ?? "";
      return (!modelFilter || model === modelFilter) &&
        (!runnerFilter || runner === runnerFilter) &&
        (!profileFilter || run.agentProfileId === profileFilter) &&
        (!tagFilter || run.tag.startsWith(tagFilter)) &&
        (!workloadFilter || workload === workloadFilter) &&
        (!errorFilter || run.errorMsg.toLowerCase().includes(errorFilter.toLowerCase())) &&
        (!fromFilter || created >= Date.parse(fromFilter)) &&
        (!toFilter || created <= Date.parse(toFilter));
    });

    result.sort((a, b) => {
      const aTime = a.createdAt ? timestampMs(a.createdAt) : 0;
      const bTime = b.createdAt ? timestampMs(b.createdAt) : 0;
      return sortBy === "newest" ? bTime - aTime : aTime - bTime;
    });

    return result;
  }, [resolvedRuns, statusFilter, searchQuery, sortBy, searchParams, getTaskTitle, getProfileName]);

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
        selectedRunId={selectedRun?.id}
        onSelect={handleSelectRun}
        onCheckboxChange={handleRunRowCheckboxChange}
        onStop={handleStopRequest}
        onResume={handleResumeRequest}
      />
    ),
    [
      getProfileName,
      getTaskTitle,
      handleResumeRequest,
      handleRunRowCheckboxChange,
      handleSelectRun,
      handleStopRequest,
      selectedRun?.id,
      selectedRunIds,
      selectionMode,
    ]
  );

  useEffect(() => {
    if (!isDesktop) return;
    if (runId) return;
    if (filteredAndSortedRuns.length === 0) return;

    // A sub-agent open in the detail pane is not in the top-level list; leave it selected.
    const hasSelection = selectedRunId !== null;

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
  const activeUrlFilters = ["from", "to", "model", "runnerType", "profileId", "tagPrefix", "workloadKey", "toolName", "errorCode"]
    .map((key) => [key, searchParams.get(key)] as const)
    .filter((entry): entry is readonly [string, string] => Boolean(entry[1]));
  const clearUrlFilter = (key: string) => {
    const params = new URLSearchParams(searchParams);
    params.delete(key);
    navigate(`/runs?${params.toString()}`, { replace: true });
  };

  const conversationSearchActive = searchQuery.trim().length > 0;
  const handleOpenConversationHit = useCallback((hit: ConversationSearchHit) => {
    const params = new URLSearchParams({
      tab: "timeline",
      event: hit.eventId,
      sequence: hit.eventSequence.toString(),
      q: searchQuery,
    });
    navigate(`/runs/${encodeURIComponent(hit.runId)}?${params.toString()}`);
  }, [navigate, searchQuery]);

  const runsListPanel = (
    <ListPanel
      title="All Runs"
      count={filteredAndSortedRuns.length}
      loading={loading}
      error={error}
      onRetry={onRefresh}
      headerActions={
        <div className="flex gap-2">
          {selectedRunIds.size > 0 && (
            <Button
              variant="primary"
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
              variant={selectionMode ? "primary" : "secondary"}
              size="sm"
              onClick={toggleSelectionMode}
              aria-label={selectionMode ? "Exit selection mode" : "Enter selection mode"}
              className="h-11 w-11 px-0 sm:h-9 sm:w-9"
            >
              {selectionMode ? (
                <X className="h-4 w-4" />
              ) : (
                <CheckSquare className="h-4 w-4" />
              )}
            </Button>
          )}
          <Button variant="secondary" size="sm" onClick={onRefresh} aria-label="Refresh" className="h-11 w-11 px-0 sm:h-9 sm:w-9">
            <RefreshCw className="h-4 w-4" />
          </Button>
        </div>
      }
      toolbar={
        <div className="space-y-2">
          <div role="group" aria-label="Who started the run" className="flex flex-wrap gap-1.5">
            {ORIGIN_CHIPS.map((chip) => (
              <button
                key={chip.value}
                type="button"
                aria-pressed={origin === chip.value}
                data-testid={`runs-origin-${chip.value || "all"}`}
                onClick={() => onOriginChange(chip.value)}
                className={`inline-flex min-h-11 min-w-11 items-center justify-center rounded-full border px-3 text-xs font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-8 ${origin === chip.value ? "border-primary bg-primary/10 text-primary" : "border-border text-muted-foreground hover:text-foreground"}`}
              >
                {chip.label}
              </button>
            ))}
          </div>
          <SearchToolbar
            searchValue={searchQuery}
            onSearchChange={setSearchQuery}
            searchPlaceholder="Search all conversation history..."
            filters={filters}
            sortOptions={SORT_OPTIONS}
            currentSort={sortBy}
            onSortChange={setSortBy}
          />
          {activeUrlFilters.length > 0 && <div className="flex flex-wrap gap-1.5" aria-label="Active URL filters">{activeUrlFilters.map(([key, value]) => <button key={key} type="button" onClick={() => clearUrlFilter(key)} className="rounded-full border border-primary/30 bg-primary/5 px-2 py-0.5 text-xs text-muted-foreground hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">{key}: {value} ×</button>)}</div>}
        </div>
      }
      empty={
        <EmptyState
          density="compact"
          icon={<Activity aria-hidden="true" />}
          title={runs.length === 0 ? "No Runs Yet" : "No Matching Runs"}
          description={runs.length === 0 ? "Start a run from the Tasks tab" : "Try adjusting your filters"}
        />
      }
    >
      <BoundedList
        items={filteredAndSortedRuns}
        getKey={getRunKey}
        renderItem={renderRunRow}
      />
    </ListPanel>
  );

  const listPanel = conversationSearchActive ? (
    <div className="flex h-full min-w-0 flex-col overflow-hidden bg-card">
      <div className="shrink-0 border-b border-border px-4 py-3">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Conversation history ({conversationResultCount})</h2>
          <Button type="button" variant="ghost" size="sm" onClick={() => setSearchQuery("")}>Back to recent runs</Button>
        </div>
        <div className="mt-3">
          <SearchToolbar
            searchValue={searchQuery}
            onSearchChange={setSearchQuery}
            searchPlaceholder="Search all conversation history..."
          />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-3">
        <ConversationSearchResults
          query={searchQuery}
          filters={conversationFilters}
          onFiltersChange={setConversationFilters}
          onOpenHit={handleOpenConversationHit}
          onResultCount={setConversationResultCount}
        />
      </div>
    </div>
  ) : runsListPanel;

  // Determine initial tab: query param > status-based default > "timeline"
  const tabParam = searchParams.get("tab");
  const initialTab = useMemo<RunTabId>(() => {
    if (isRunTab(tabParam)) return tabParam;
    if (selectedRun?.status === RunStatus.NEEDS_REVIEW) return "diff";
    return "timeline";
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
            focusEventId={searchParams.get("event") ?? undefined}
            focusSequence={searchParams.get("sequence") ?? undefined}
            events={events}
            diff={diff}
            eventsLoading={eventsLoading}
            diffLoading={diffLoading}
            task={getTaskById(selectedRun.taskId)}
            taskTitle={getTaskTitle(selectedRun.taskId)}
            profileName={runOrigin(selectedRun, getProfileName(selectedRun.agentProfileId))}
            onPartialApprove={(fileIds, actor, commitMsg) => handlePartialApprove(selectedRun.id, fileIds, actor, commitMsg)}
            onRetry={handleRetry}
            onResumeFromFailure={handleResumeRequest}
            onInvestigate={handleInvestigateFromDetail}
            onApplyInvestigation={handleApplyInvestigationFromDetail}
            onStop={(r) => { handleStopRequest(r); return Promise.resolve(); }}
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
            deleteLoading={runActions.pending?.confirmLabel === "Delete run"}
            onMobileHeaderLeft={setMobileHeaderLeft}
            onMobileHeaderRight={setMobileHeaderRight}
          />
        )}
      </DetailPanel>
  );


  return (
    <>
      <MasterDetailLayout
        storageKey="runs"
        listPanel={listPanel}
        detailPanel={detailPanel}
        selectedId={selectedRun?.id ?? null}
        onDeselect={() => {
          isDeselectingRef.current = true;
          setSelectedRun(null);
          navigate("/runs");
        }}
        detailTitle={selectedRun ? runHeading(selectedRun, getTaskTitle(selectedRun.taskId)).title : "Run Details"}
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
        description="Start a bounded, read-only diagnosis. Evidence coverage and applicability stay visible with the result."
        confirmLabel="Start Investigation"
        mode="typed"
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
        onViewRun={(runId) => {
          setApplyModalOpen(false);
          setApplyInvestigationRun(null);
          setApplyError(null);
          navigate(`/runs/${runId}`);
        }}
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

      {runActions.dialog}
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
