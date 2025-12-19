import { useCallback, useEffect, useState } from "react";
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
  ApprovalResult,
  ApproveRequest,
  DiffResult,
  RejectRequest,
  Run,
  RunEvent,
  Task,
} from "../types";

interface RunsPageProps {
  runs: Run[];
  tasks: Task[];
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onStopRun: (id: string) => Promise<void>;
  onGetEvents: (id: string) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<DiffResult>;
  onApproveRun: (id: string, req: ApproveRequest) => Promise<ApprovalResult>;
  onRejectRun: (id: string, req: RejectRequest) => Promise<void>;
  onRefresh: () => void;
}

export function RunsPage({
  runs,
  tasks,
  profiles,
  loading,
  error,
  onStopRun,
  onGetEvents,
  onGetDiff,
  onApproveRun,
  onRejectRun,
  onRefresh,
}: RunsPageProps) {
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const [events, setEvents] = useState<RunEvent[]>([]);
  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [diffLoading, setDiffLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<"events" | "diff">("events");
  const [approvalForm, setApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [rejectForm, setRejectForm] = useState({ actor: "", reason: "" });
  const [submitting, setSubmitting] = useState(false);

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
        run.status === "needs_review" ||
        run.status === "complete" ||
        run.approvalState !== "none"
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

  // Auto-refresh events for running runs
  useEffect(() => {
    if (
      !selectedRun ||
      (selectedRun.status !== "running" && selectedRun.status !== "starting")
    ) {
      return;
    }

    const interval = setInterval(async () => {
      try {
        const evts = await onGetEvents(selectedRun.id);
        setEvents(evts || []);
      } catch (err) {
        console.error("Failed to refresh events:", err);
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [selectedRun, onGetEvents]);

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

  const sortedRuns = [...runs].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const getTaskTitle = (taskId: string) =>
    tasks.find((t) => t.id === taskId)?.title || "Unknown Task";
  const getProfileName = (profileId: string) =>
    profiles.find((p) => p.id === profileId)?.name || "Unknown Profile";

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
                      onClick={() => loadRunDetails(run)}
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
                            run.status as
                              | "running"
                              | "complete"
                              | "failed"
                              | "pending"
                              | "needs_review"
                          }
                        >
                          {run.status.replace("_", " ")}
                        </Badge>
                        {(run.status === "running" || run.status === "starting") && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
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
                    <Badge
                      variant={
                        selectedRun.status as
                          | "running"
                          | "complete"
                          | "failed"
                          | "pending"
                      }
                    >
                      {selectedRun.status.replace("_", " ")}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-sm">
                    <div>
                      <span className="text-muted-foreground">Profile: </span>
                      {getProfileName(selectedRun.agentProfileId)}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Mode: </span>
                      {selectedRun.runMode}
                    </div>
                    <div>
                      <span className="text-muted-foreground">Phase: </span>
                      {selectedRun.phase.replace("_", " ")}
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
                          new Date(selectedRun.endedAt).getTime() -
                            new Date(selectedRun.startedAt).getTime()
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
                {selectedRun.status === "needs_review" && (
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

function RunStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "complete":
    case "approved":
      return <Check className="h-5 w-5 text-success" />;
    case "failed":
    case "rejected":
      return <XCircle className="h-5 w-5 text-destructive" />;
    case "running":
    case "starting":
      return <Activity className="h-5 w-5 text-primary animate-pulse" />;
    case "needs_review":
      return <Clock className="h-5 w-5 text-warning" />;
    default:
      return <Clock className="h-5 w-5 text-muted-foreground" />;
  }
}

function EventItem({ event }: { event: RunEvent }) {
  const [expanded, setExpanded] = useState(false);
  const data = event.data;

  // Check if this is a sandbox conflict error
  const hasConflicts = event.eventType === "error" &&
    data.details?.conflicts &&
    data.details.conflicts.length > 0;

  const getIcon = () => {
    switch (event.eventType) {
      case "log":
        return <Terminal className="h-4 w-4" />;
      case "message":
        return <MessageSquare className="h-4 w-4" />;
      case "tool_call":
      case "tool_result":
        return <Wrench className="h-4 w-4" />;
      case "status":
        return <Activity className="h-4 w-4" />;
      case "error":
        return <AlertCircle className="h-4 w-4 text-destructive" />;
      default:
        return <ChevronRight className="h-4 w-4" />;
    }
  };

  const getSummary = () => {
    switch (event.eventType) {
      case "log":
        return data.message || "Log entry";
      case "message":
        return (data.role || "unknown") + ": " + (data.content?.slice(0, 100) || "");
      case "tool_call":
        return "Called " + (data.toolName || "unknown tool");
      case "tool_result":
        return (data.success ? "Success" : "Failed") + ": " + (data.toolName || "");
      case "status":
        return (data.oldStatus || "?") + " -> " + (data.newStatus || "?");
      case "error":
        if (hasConflicts) {
          return `Sandbox conflict: ${data.details!.conflicts!.length} conflicting sandbox(es)`;
        }
        return data.message || data.code || "Error occurred";
      default:
        return event.eventType;
    }
  };

  return (
    <div
      className={cn(
        "rounded border border-border p-2 text-xs",
        event.eventType === "error" && "border-destructive/50 bg-destructive/5"
      )}
    >
      <div
        className="flex items-center gap-2 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        {getIcon()}
        <span className="flex-1 truncate">{getSummary()}</span>
        <span className="text-muted-foreground">
          {new Date(event.timestamp).toLocaleTimeString()}
        </span>
        {expanded ? (
          <ChevronDown className="h-3 w-3" />
        ) : (
          <ChevronRight className="h-3 w-3" />
        )}
      </div>
      {expanded && (
        <div className="mt-2 space-y-2">
          {/* Show conflict details in a user-friendly format */}
          {hasConflicts && (
            <div className="p-2 bg-destructive/10 rounded border border-destructive/20">
              <div className="font-medium text-destructive mb-2">
                Conflicting Sandboxes
              </div>
              <div className="space-y-2">
                {data.details!.conflicts!.map((conflict) => (
                  <div
                    key={conflict.sandboxId}
                    className="p-2 bg-background rounded border border-border"
                  >
                    <div className="flex items-center gap-2">
                      <Badge variant="outline" className="text-[10px] font-mono">
                        {conflict.sandboxId.slice(0, 8)}...
                      </Badge>
                      <span className="text-muted-foreground">
                        {conflict.conflictType === "new_contains_existing"
                          ? "Your scope contains this sandbox"
                          : "This sandbox contains your scope"}
                      </span>
                    </div>
                    <div className="mt-1 font-mono text-[10px] text-muted-foreground truncate">
                      {conflict.scope}
                    </div>
                  </div>
                ))}
              </div>
              <div className="mt-2 text-[10px] text-muted-foreground">
                To resolve: Delete or stop the conflicting sandbox(es) using the workspace-sandbox CLI:
                <code className="block mt-1 p-1 bg-muted rounded font-mono">
                  workspace-sandbox delete {data.details!.conflicts![0]?.sandboxId || "<sandbox-id>"}
                </code>
              </div>
            </div>
          )}
          {/* Show raw JSON data */}
          <pre className="p-2 bg-muted rounded text-[10px] overflow-x-auto">
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function DiffViewer({ diff }: { diff: DiffResult }) {
  return (
    <div className="space-y-3">
      {/* Stats */}
      <div className="flex gap-4 text-xs">
        <span className="text-success">+{diff.stats?.additions || 0}</span>
        <span className="text-destructive">-{diff.stats?.deletions || 0}</span>
        <span className="text-muted-foreground">
          {diff.stats?.filesChanged || 0} files
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
                  file.status === "added"
                    ? "success"
                    : file.status === "deleted"
                    ? "destructive"
                    : "secondary"
                }
                className="text-[10px] px-1"
              >
                {file.status}
              </Badge>
              <span className="font-mono truncate">{file.path}</span>
              <span className="text-success">+{file.additions}</span>
              <span className="text-destructive">-{file.deletions}</span>
            </div>
          ))}
        </div>
      )}

      {/* Unified Diff */}
      {diff.unified && (
        <pre className="text-[10px] font-mono bg-muted/50 rounded p-3 overflow-x-auto whitespace-pre-wrap">
          {diff.unified.split("\n").map((line, i) => (
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
