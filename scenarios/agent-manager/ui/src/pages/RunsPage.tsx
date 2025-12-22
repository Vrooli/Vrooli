import { useCallback, useEffect, useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { useParams, useNavigate } from "react-router-dom";
import {
  Activity,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  Clock,
  Eye,
  FileCode,
  MessageSquare,
  Play,
  RefreshCw,
  RotateCcw,
  Square,
  Terminal,
  Wrench,
  X,
  XCircle,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ScrollArea } from "../components/ui/scroll-area";
import { Textarea } from "../components/ui/textarea";
import { cn, formatDate, formatDuration, formatRelativeTime } from "../lib/utils";
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
import { ApprovalState, RunEventType, RunMode, RunPhase, RunStatus } from "../types";
import type { MessageHandler, WebSocketMessage } from "../hooks/useWebSocket";

interface RunsPageProps {
  runs: Run[];
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onStopRun: (id: string) => Promise<void>;
  onRetryRun: (run: Run) => Promise<Run>;
  onGetEvents: (id: string) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<RunDiff>;
  onApproveRun: (id: string, req: ApproveFormData) => Promise<ApproveResult>;
  onRejectRun: (id: string, req: RejectFormData) => Promise<void>;
  onRefresh: () => void;
  wsSubscribe: (runId: string) => void;
  wsUnsubscribe: (runId: string) => void;
  wsAddMessageHandler: (handler: MessageHandler) => void;
  wsRemoveMessageHandler: (handler: MessageHandler) => void;
}

export function RunsPage({
  runs,
  tasks,
  profiles,
  loading,
  error,
  onStopRun,
  onRetryRun,
  onGetEvents,
  onGetDiff,
  onApproveRun,
  onRejectRun,
  onRefresh,
  wsSubscribe,
  wsUnsubscribe,
  wsAddMessageHandler,
  wsRemoveMessageHandler,
}: RunsPageProps) {
  const { runId } = useParams<{ runId?: string }>();
  const navigate = useNavigate();
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const [events, setEvents] = useState<RunEvent[]>([]);
  const [diff, setDiff] = useState<RunDiff | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [diffLoading, setDiffLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"events" | "diff">("events");
  const [approvalForm, setApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [rejectForm, setRejectForm] = useState({ actor: "", reason: "" });
  const [submitting, setSubmitting] = useState(false);

  // Subscribe to WebSocket events for the selected run
  useEffect(() => {
    if (!selectedRun) return;

    // Subscribe to this specific run's events
    wsSubscribe(selectedRun.id);

    return () => {
      // Unsubscribe when run is deselected or component unmounts
      wsUnsubscribe(selectedRun.id);
    };
  }, [selectedRun?.id, wsSubscribe, wsUnsubscribe]);

  // Handle WebSocket messages for real-time updates
  useEffect(() => {
    const handleMessage: MessageHandler = (message: WebSocketMessage) => {
      // Only handle messages for the selected run
      if (!selectedRun || message.runId !== selectedRun.id) return;

      switch (message.type) {
        case "run_event": {
          // Append new event to the list
          const newEvent = message.payload as RunEvent;
          setEvents((prev) => {
            // Avoid duplicates by checking sequence number
            if (prev.some((e) => e.id === newEvent.id || e.sequence === newEvent.sequence)) {
              return prev;
            }
            return [...prev, newEvent];
          });
          break;
        }
        case "run_status": {
          // Update the selected run's status
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

      // Load events
      setEventsLoading(true);
      try {
        const evts = await onGetEvents(run.id);
        setEvents(evts || []);
      } catch (err) {
        console.error("Failed to load events:", err);
      } finally {
        setEventsLoading(false);
      }

      // Load diff if run needs review or is complete
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

  // Note: Real-time updates are handled via WebSocket subscription above.
  // HTTP polling has been removed in favor of WebSocket-based updates.

  // Sync selectedRun with the runs list when it updates
  useEffect(() => {
    if (!selectedRun) return;

    const updatedRun = runs.find((r) => r.id === selectedRun.id);
    if (updatedRun && updatedRun !== selectedRun) {
      // Update selectedRun with latest data from the runs list
      setSelectedRun(updatedRun);
    }
  }, [runs, selectedRun]);

  // Load run from URL params when component mounts or runId changes
  useEffect(() => {
    if (!runId || runs.length === 0) return;

    // Check if we already have this run selected
    if (selectedRun?.id === runId) return;

    // Find the run in our list
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

  const handleApprove = async () => {
    if (!selectedRun || !approvalForm.actor) return;
    setSubmitting(true);
    try {
      await onApproveRun(selectedRun.id, {
        actor: approvalForm.actor,
        commitMsg: approvalForm.commitMsg || undefined,
      });
      setApprovalForm({ actor: "", commitMsg: "" });
      setSelectedRun(null);
      onRefresh();
    } catch (err) {
      console.error("Failed to approve run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    if (!selectedRun || !rejectForm.actor) return;
    setSubmitting(true);
    try {
      await onRejectRun(selectedRun.id, {
        actor: rejectForm.actor,
        reason: rejectForm.reason || undefined,
      });
      setRejectForm({ actor: "", reason: "" });
      setSelectedRun(null);
      onRefresh();
    } catch (err) {
      console.error("Failed to reject run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const sortedRuns = [...runs].sort((a, b) => {
    const aTime = a.createdAt ? timestampMs(a.createdAt) : 0;
    const bTime = b.createdAt ? timestampMs(b.createdAt) : 0;
    return bTime - aTime;
  });

  const getTaskTitle = (taskId: string) =>
    tasks.find((t) => t.id === taskId)?.title || "Unknown Task";
  const getProfileName = (profileId?: string) =>
    profileId ? profiles.find((p) => p.id === profileId)?.name || "Unknown Profile" : "Unknown Profile";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Runs</h2>
          <p className="text-sm text-muted-foreground">
            Monitor executions and review changes
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
          <RefreshCw className="h-4 w-4" />
          Refresh
        </Button>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Runs List */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Play className="h-5 w-5" />
              All Runs ({runs.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="py-8 text-center text-muted-foreground">
                Loading runs...
              </div>
            ) : sortedRuns.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Activity className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">No runs yet</p>
                <p className="text-xs">Start a run from the Tasks tab</p>
              </div>
            ) : (
              <ScrollArea className="h-[500px]">
                <div className="space-y-2">
                  {sortedRuns.map((run) => (
                    <div
                      key={run.id}
                      className={cn(
                        "flex items-center justify-between rounded-lg border p-3 cursor-pointer transition-colors",
                        selectedRun?.id === run.id
                          ? "border-primary bg-primary/5"
                          : "border-border hover:bg-muted/50"
                      )}
                      onClick={() => {
                        navigate(`/runs/${run.id}`);
                        loadRunDetails(run);
                      }}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          event.preventDefault();
                          navigate(`/runs/${run.id}`);
                          loadRunDetails(run);
                        }
                      }}
                    >
                      <div className="flex items-center gap-3">
                        <RunStatusIcon status={run.status} />
                        <div>
                          <p className="font-medium text-sm">
                            {getTaskTitle(run.taskId)}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {getProfileName(run.agentProfileId)} |{" "}
                            {formatRelativeTime(run.createdAt)}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
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
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            )}
          </CardContent>
        </Card>

        {/* Run Details */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Eye className="h-5 w-5" />
              Run Details
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!selectedRun ? (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <Eye className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">Select a run to view details</p>
              </div>
            ) : (
              <div className="space-y-4">
                {/* Run Summary */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <h3 className="font-semibold">
                      {getTaskTitle(selectedRun.taskId)}
                    </h3>
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={
                          runStatusLabel(selectedRun.status) as
                            | "pending"
                            | "starting"
                            | "running"
                            | "needs_review"
                            | "complete"
                            | "failed"
                            | "cancelled"
                        }
                      >
                        {runStatusLabel(selectedRun.status).replace("_", " ")}
                      </Badge>
                      {[
                        RunStatus.FAILED,
                        RunStatus.CANCELLED,
                        RunStatus.COMPLETE,
                      ].includes(selectedRun.status) || [
                        ApprovalState.APPROVED,
                        ApprovalState.REJECTED,
                      ].includes(selectedRun.approvalState) ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={async () => {
                            const newRun = await onRetryRun(selectedRun);
                            loadRunDetails(newRun);
                          }}
                          className="gap-1"
                        >
                          <RotateCcw className="h-3 w-3" />
                          Re-run
                        </Button>
                      ) : null}
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <span className="text-muted-foreground">Profile: </span>
                      {getProfileName(selectedRun.agentProfileId)}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Mode: </span>
                      {runModeLabel(selectedRun.runMode)}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Phase: </span>
                      {runPhaseLabel(selectedRun.phase).replace("_", " ")}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Progress: </span>
                      {selectedRun.progressPercent}%
                    </div>
                    {selectedRun.changedFiles > 0 && (
                      <div>
                        <span className="text-muted-foreground">Files: </span>
                        {selectedRun.changedFiles} changed
                      </div>
                    )}
                    {selectedRun.endedAt && selectedRun.startedAt && (
                      <div>
                        <span className="text-muted-foreground">Duration: </span>
                        {formatDuration(
                          timestampMs(selectedRun.endedAt) -
                            timestampMs(selectedRun.startedAt)
                        )}
                      </div>
                    )}
                  </div>
                </div>

                {/* Tab Selector */}
                <div className="flex border-b border-border">
                  <button
                    className={cn(
                      "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                      activeTab === "events"
                        ? "border-primary text-primary"
                        : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                    onClick={() => setActiveTab("events")}
                  >
                    <Terminal className="h-4 w-4 inline mr-2" />
                    Events ({events.length})
                  </button>
                  <button
                    className={cn(
                      "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors",
                      activeTab === "diff"
                        ? "border-primary text-primary"
                        : "border-transparent text-muted-foreground hover:text-foreground"
                    )}
                    onClick={() => setActiveTab("diff")}
                  >
                    <FileCode className="h-4 w-4 inline mr-2" />
                    Diff
                  </button>
                </div>

                {/* Content */}
                <ScrollArea className="h-[300px]">
                  {activeTab === "events" ? (
                    eventsLoading ? (
                      <div className="py-8 text-center text-muted-foreground">
                        Loading events...
                      </div>
                    ) : events.length === 0 ? (
                      <div className="py-8 text-center text-muted-foreground">
                        No events recorded
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {events.map((event) => (
                          <EventItem key={event.id} event={event} />
                        ))}
                      </div>
                    )
                  ) : diffLoading ? (
                    <div className="py-8 text-center text-muted-foreground">
                      Loading diff...
                    </div>
                  ) : !diff ? (
                    <div className="py-8 text-center text-muted-foreground">
                      No diff available
                    </div>
                  ) : (
                    <DiffViewer diff={diff} />
                  )}
                </ScrollArea>

                {/* Approval Actions */}
                {selectedRun.status === RunStatus.NEEDS_REVIEW && (
                  <div className="space-y-4 pt-4 border-t border-border">
                    <h4 className="font-semibold text-sm">Review Actions</h4>
                    <div className="grid gap-3 md:grid-cols-2">
                      <div className="space-y-2">
                        <Label htmlFor="approveActor">Your Name *</Label>
                        <Input
                          id="approveActor"
                          value={approvalForm.actor}
                          onChange={(e) =>
                            setApprovalForm({
                              ...approvalForm,
                              actor: e.target.value,
                            })
                          }
                          placeholder="e.g., John Doe"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="commitMsg">Commit Message</Label>
                        <Input
                          id="commitMsg"
                          value={approvalForm.commitMsg}
                          onChange={(e) =>
                            setApprovalForm({
                              ...approvalForm,
                              commitMsg: e.target.value,
                            })
                          }
                          placeholder="Optional custom message"
                        />
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="success"
                        onClick={handleApprove}
                        disabled={!approvalForm.actor || submitting}
                        className="gap-2"
                      >
                        <Check className="h-4 w-4" />
                        Approve & Apply
                      </Button>
                      <Button
                        variant="destructive"
                        onClick={() => {
                          setRejectForm({
                            actor: approvalForm.actor,
                            reason: "",
                          });
                        }}
                        disabled={!approvalForm.actor || submitting}
                        className="gap-2"
                      >
                        <X className="h-4 w-4" />
                        Reject
                      </Button>
                    </div>

                    {rejectForm.actor && (
                      <div className="space-y-2 p-3 bg-destructive/10 rounded-lg">
                        <Label htmlFor="rejectReason">Rejection Reason</Label>
                        <Textarea
                          id="rejectReason"
                          value={rejectForm.reason}
                          onChange={(e) =>
                            setRejectForm({
                              ...rejectForm,
                              reason: e.target.value,
                            })
                          }
                          placeholder="Why are you rejecting these changes?"
                          rows={2}
                        />
                        <div className="flex gap-2">
                          <Button
                            variant="destructive"
                            size="sm"
                            onClick={handleReject}
                            disabled={submitting}
                          >
                            Confirm Rejection
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setRejectForm({ actor: "", reason: "" })}
                          >
                            Cancel
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
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

function runModeLabel(mode: RunMode): string {
  switch (mode) {
    case RunMode.SANDBOXED:
      return "sandboxed";
    case RunMode.IN_PLACE:
      return "in_place";
    default:
      return "unspecified";
  }
}

function runPhaseLabel(phase: RunPhase): string {
  switch (phase) {
    case RunPhase.QUEUED:
      return "queued";
    case RunPhase.INITIALIZING:
      return "initializing";
    case RunPhase.SANDBOX_CREATING:
      return "sandbox_creating";
    case RunPhase.RUNNER_ACQUIRING:
      return "runner_acquiring";
    case RunPhase.EXECUTING:
      return "executing";
    case RunPhase.COLLECTING_RESULTS:
      return "collecting_results";
    case RunPhase.AWAITING_REVIEW:
      return "awaiting_review";
    case RunPhase.APPLYING:
      return "applying";
    case RunPhase.CLEANING_UP:
      return "cleaning_up";
    case RunPhase.COMPLETED:
      return "completed";
    default:
      return "queued";
  }
}

function RunStatusIcon({ status }: { status: RunStatus }) {
  switch (status) {
    case RunStatus.COMPLETE:
      return <Check className="h-5 w-5 text-success" />;
    case RunStatus.FAILED:
      return <XCircle className="h-5 w-5 text-destructive" />;
    case RunStatus.RUNNING:
    case RunStatus.STARTING:
      return <Activity className="h-5 w-5 text-primary animate-pulse" />;
    case RunStatus.NEEDS_REVIEW:
      return <Clock className="h-5 w-5 text-warning" />;
    default:
      return <Clock className="h-5 w-5 text-muted-foreground" />;
  }
}

function EventItem({ event }: { event: RunEvent }) {
  const [expanded, setExpanded] = useState(false);
  const payload = event.data;
  const payloadValue = payload.value as any;

  const getIcon = () => {
    switch (event.eventType) {
      case RunEventType.LOG:
        return <Terminal className="h-4 w-4" />;
      case RunEventType.MESSAGE:
        return <MessageSquare className="h-4 w-4" />;
      case RunEventType.TOOL_CALL:
      case RunEventType.TOOL_RESULT:
        return <Wrench className="h-4 w-4" />;
      case RunEventType.STATUS:
        return <Activity className="h-4 w-4" />;
      case RunEventType.ERROR:
        return <AlertCircle className="h-4 w-4 text-destructive" />;
      default:
        return <ChevronRight className="h-4 w-4" />;
    }
  };

  const getSummary = () => {
    switch (payload.case) {
      case "log":
        return payloadValue.message || "Log entry";
      case "message":
        return (payloadValue.role || "unknown") + ": " + (payloadValue.content?.slice(0, 100) || "");
      case "toolCall":
        return "Called " + (payloadValue.toolName || "unknown tool");
      case "toolResult":
        return (payloadValue.success ? "Success" : "Failed") + ": " + (payloadValue.toolName || "");
      case "status":
        return (
          runStatusLabel(payloadValue.oldStatus ?? RunStatus.UNSPECIFIED) +
          " -> " +
          runStatusLabel(payloadValue.newStatus ?? RunStatus.UNSPECIFIED)
        );
      case "metric":
        return `${payloadValue.name || "metric"}: ${payloadValue.value ?? 0}`;
      case "artifact":
        return payloadValue.path ? `Artifact: ${payloadValue.path}` : "Artifact";
      case "progress":
        return `Progress ${payloadValue.percentComplete ?? 0}%`;
      case "cost":
        return "Cost update";
      case "rateLimit":
        return "Rate limit";
      case "error":
        return payloadValue.message || payloadValue.code || "Error occurred";
      default:
        return runEventTypeLabel(event.eventType);
    }
  };

  return (
    <div
      className={cn(
        "rounded border border-border p-2 text-xs",
        event.eventType === RunEventType.ERROR && "border-destructive/50 bg-destructive/5"
      )}
    >
      <div
        className="flex items-center gap-2 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        {getIcon()}
        <span className="flex-1 truncate">{getSummary()}</span>
        <span className="text-muted-foreground">
          {formatDate(event.timestamp)}
        </span>
        {expanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
      </div>
      {expanded && (
        <div className="mt-2 space-y-2">
          <pre className="p-2 bg-muted rounded text-[10px] overflow-x-auto">
            {JSON.stringify(payloadValue, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function runEventTypeLabel(eventType: RunEventType): string {
  switch (eventType) {
    case RunEventType.LOG:
      return "log";
    case RunEventType.MESSAGE:
      return "message";
    case RunEventType.TOOL_CALL:
      return "tool_call";
    case RunEventType.TOOL_RESULT:
      return "tool_result";
    case RunEventType.STATUS:
      return "status";
    case RunEventType.METRIC:
      return "metric";
    case RunEventType.ARTIFACT:
      return "artifact";
    case RunEventType.ERROR:
      return "error";
    default:
      return "event";
  }
}

function DiffViewer({ diff }: { diff: RunDiff }) {
  const fileCount = diff.files?.length ?? 0;
  const totals = (diff.files ?? []).reduce(
    (acc, file) => {
      acc.additions += file.additions || 0;
      acc.deletions += file.deletions || 0;
      return acc;
    },
    { additions: 0, deletions: 0 }
  );

  return (
    <div className="space-y-3">
      {/* Stats */}
      <div className="flex gap-4 text-xs">
        <span className="text-success">+{totals.additions}</span>
        <span className="text-destructive">-{totals.deletions}</span>
        <span className="text-muted-foreground">
          {fileCount} files
        </span>
      </div>

      {/* File List */}
      {diff.files && diff.files.length > 0 && (
        <div className="space-y-1">
          {diff.files.map((file) => (
            <div
              key={file.path}
              className="flex items-center gap-2 text-xs rounded px-2 py-1 bg-muted/50"
            >
              <Badge
                variant={
                  file.changeType === "added"
                    ? "success"
                    : file.changeType === "deleted"
                    ? "destructive"
                    : "secondary"
                }
                className="text-[10px] px-1"
              >
                {file.changeType}
              </Badge>
              <span className="font-mono truncate">{file.path}</span>
              <span className="text-success">+{file.additions}</span>
              <span className="text-destructive">-{file.deletions}</span>
            </div>
          ))}
        </div>
      )}

      {/* Unified Diff */}
      {diff.content && (
        <pre className="text-[10px] font-mono bg-muted/50 rounded p-3 overflow-x-auto whitespace-pre-wrap">
          {diff.content.split("\n").map((line, i) => (
            <div
              key={i}
              className={cn(
                line.startsWith("+") && !line.startsWith("+++")
                  ? "diff-add"
                  : line.startsWith("-") && !line.startsWith("---")
                  ? "diff-remove"
                  : line.startsWith("@@")
                  ? "text-primary"
                  : "diff-context"
              )}
            >
              {line}
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}
