import { useState } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  DollarSign,
  FileCode,
  MessageSquare,
  RotateCcw,
  Terminal,
  Trash2,
  Wrench,
  X,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { ScrollArea } from "./ui/scroll-area";
import { Textarea } from "./ui/textarea";
import { cn, formatDate, formatDuration, runnerTypeLabel } from "../lib/utils";
import type {
  ApproveFormData,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
} from "../types";
import { ApprovalState, RunEventType, RunMode, RunPhase, RunStatus } from "../types";

interface RunDetailProps {
  run: Run;
  events: RunEvent[];
  diff: RunDiff | null;
  eventsLoading: boolean;
  diffLoading: boolean;
  taskTitle: string;
  profileName: string;
  onApprove: (req: ApproveFormData) => Promise<void>;
  onReject: (req: RejectFormData) => Promise<void>;
  onRetry: (run: Run) => Promise<Run>;
  onDelete: (run: Run) => Promise<void>;
  deleteLoading: boolean;
}

export function RunDetail({
  run,
  events,
  diff,
  eventsLoading,
  diffLoading,
  taskTitle,
  profileName,
  onApprove,
  onReject,
  onRetry,
  onDelete,
  deleteLoading,
}: RunDetailProps) {
  const [activeTab, setActiveTab] = useState<"events" | "diff" | "response" | "cost">("events");
  const [approvalForm, setApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [rejectForm, setRejectForm] = useState({ actor: "", reason: "" });
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const finalResponse = getFinalResponse(events);
  const costTotals = getCostTotals(events);

  const canDeleteRun = ![RunStatus.PENDING, RunStatus.STARTING, RunStatus.RUNNING].includes(run.status);

  const handleApprove = async () => {
    setSubmitting(true);
    try {
      await onApprove({
        actor: approvalForm.actor.trim() || undefined,
        commitMsg: approvalForm.commitMsg || undefined,
      });
      setApprovalForm({ actor: "", commitMsg: "" });
      setShowRejectConfirm(false);
    } catch (err) {
      console.error("Failed to approve run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleReject = async () => {
    setSubmitting(true);
    try {
      await onReject({
        actor: rejectForm.actor.trim() || undefined,
        reason: rejectForm.reason || undefined,
      });
      setRejectForm({ actor: "", reason: "" });
      setShowRejectConfirm(false);
    } catch (err) {
      console.error("Failed to reject run:", err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Run Summary */}
      <div className="space-y-2">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <h3 className="font-semibold">{taskTitle}</h3>
          <div className="flex items-center gap-2 flex-wrap">
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
            {run.approvalState !== ApprovalState.NONE && (
              <Badge variant="outline">
                {approvalStateLabel(run.approvalState)}
              </Badge>
            )}
            {([RunStatus.FAILED, RunStatus.CANCELLED, RunStatus.COMPLETE].includes(run.status) ||
              [ApprovalState.APPROVED, ApprovalState.REJECTED].includes(run.approvalState)) && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => onRetry(run)}
                className="gap-1"
              >
                <RotateCcw className="h-3 w-3" />
                Re-run
              </Button>
            )}
            {canDeleteRun && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete(run)}
                className="gap-1 text-destructive hover:text-destructive"
                disabled={deleteLoading}
              >
                <Trash2 className="h-3 w-3" />
                Delete
              </Button>
            )}
          </div>
        </div>
        {run.errorMsg && (
          <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
            <div className="flex items-start gap-2">
              <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
              <div className="space-y-1">
                <p className="font-medium">Failure reason</p>
                <p className="break-words text-xs text-destructive/90">
                  {run.errorMsg}
                </p>
              </div>
            </div>
          </div>
        )}
        <div className="grid grid-cols-2 gap-2 text-sm">
          <div>
            <span className="text-muted-foreground">Profile: </span>
            {profileName}
          </div>
          <div>
            <span className="text-muted-foreground">Mode: </span>
            {runModeLabel(run.runMode)}
          </div>
          <div>
            <span className="text-muted-foreground">Phase: </span>
            {runPhaseLabel(run.phase).replace("_", " ")}
          </div>
          <div>
            <span className="text-muted-foreground">Progress: </span>
            {run.progressPercent}%
          </div>
          {run.resolvedConfig?.fallbackRunnerTypes?.length ? (
            <div className="col-span-2">
              <span className="text-muted-foreground">Fallbacks: </span>
              {run.resolvedConfig.fallbackRunnerTypes
                .map((runnerType) => runnerTypeLabel(runnerType))
                .join(", ")}
            </div>
          ) : null}
          {run.sandboxId && (
            <div className="col-span-2">
              <span className="text-muted-foreground">Sandbox: </span>
              <code className="text-xs bg-muted px-1 py-0.5 rounded">
                {run.sandboxId}
              </code>
            </div>
          )}
          {run.changedFiles > 0 && (
            <div>
              <span className="text-muted-foreground">Files: </span>
              {run.changedFiles} changed
            </div>
          )}
          {run.endedAt && run.startedAt && (
            <div>
              <span className="text-muted-foreground">Duration: </span>
              {formatDuration(
                timestampMs(run.endedAt) - timestampMs(run.startedAt)
              )}
            </div>
          )}
        </div>
      </div>

      {/* Tab Selector */}
      <div className="flex border-b border-border overflow-x-auto">
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
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
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
            activeTab === "response"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("response")}
        >
          <MessageSquare className="h-4 w-4 inline mr-2" />
          Response
        </button>
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
            activeTab === "diff"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("diff")}
        >
          <FileCode className="h-4 w-4 inline mr-2" />
          Diff
        </button>
        <button
          className={cn(
            "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
            activeTab === "cost"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          )}
          onClick={() => setActiveTab("cost")}
        >
          <DollarSign className="h-4 w-4 inline mr-2" />
          Cost
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
        ) : activeTab === "response" ? (
          eventsLoading ? (
            <div className="py-8 text-center text-muted-foreground">
              Loading response...
            </div>
          ) : !finalResponse ? (
            <div className="py-8 text-center text-muted-foreground">
              No response available
            </div>
          ) : (
            <div className="space-y-2">
              <div className="rounded border border-border bg-muted/40 p-3 text-xs whitespace-pre-wrap">
                {finalResponse.content}
              </div>
              <div className="text-[10px] text-muted-foreground">
                {formatDate(finalResponse.timestamp)}
              </div>
            </div>
          )
        ) : activeTab === "diff" ? (
          diffLoading ? (
            <div className="py-8 text-center text-muted-foreground">
              Loading diff...
            </div>
          ) : !diff ? (
            <div className="py-8 text-center text-muted-foreground">
              No diff available
            </div>
          ) : (
            <DiffViewer diff={diff} />
          )
        ) : eventsLoading ? (
          <div className="py-8 text-center text-muted-foreground">
            Loading cost...
          </div>
        ) : costTotals.events === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            No cost data available
          </div>
        ) : (
          <CostBreakdown totals={costTotals} />
        )}
      </ScrollArea>

      {/* Approval Actions */}
      {run.status === RunStatus.NEEDS_REVIEW && (
        <div className="space-y-4 pt-4 border-t border-border">
          <h4 className="font-semibold text-sm">Review Actions</h4>
          <div className="grid gap-3 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="approveActor">Your Name (optional)</Label>
              <Input
                id="approveActor"
                value={approvalForm.actor}
                onChange={(e) =>
                  setApprovalForm({
                    ...approvalForm,
                    actor: e.target.value,
                  })
                }
                placeholder="Leave blank to approve anonymously"
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
              disabled={submitting}
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
                setShowRejectConfirm(true);
              }}
              disabled={submitting}
              className="gap-2"
            >
              <X className="h-4 w-4" />
              Reject
            </Button>
          </div>

          {showRejectConfirm && (
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
                  onClick={() => {
                    setRejectForm({ actor: "", reason: "" });
                    setShowRejectConfirm(false);
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Helper functions and sub-components

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

function approvalStateLabel(state: ApprovalState): string {
  switch (state) {
    case ApprovalState.APPROVED:
      return "approved";
    case ApprovalState.REJECTED:
      return "rejected";
    case ApprovalState.PARTIALLY_APPROVED:
      return "partial approval";
    case ApprovalState.PENDING:
      return "pending review";
    default:
      return "no approval";
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

type CostTotals = {
  inputTokens: number;
  outputTokens: number;
  cacheCreationTokens: number;
  cacheReadTokens: number;
  totalCostUsd: number;
  webSearchRequests: number;
  serverToolUseRequests: number;
  models: string[];
  serviceTiers: string[];
  events: number;
};

type FinalResponse = {
  content: string;
  timestamp: Date;
};

function getFinalResponse(events: RunEvent[]): FinalResponse | null {
  const messageEvents = events.filter((event) => event.data.case === "message");
  if (messageEvents.length === 0) return null;

  const assistantMessages = messageEvents.filter((event) => {
    const payload = event.data.value as any;
    return (payload.role || "").toLowerCase() === "assistant";
  });
  const targetEvents = assistantMessages.length > 0 ? assistantMessages : messageEvents;
  const lastEvent = targetEvents[targetEvents.length - 1];
  const payload = lastEvent.data.value as any;

  return {
    content: payload.content || "",
    timestamp: lastEvent.timestamp ? new Date(timestampMs(lastEvent.timestamp)) : new Date(),
  };
}

function getCostTotals(events: RunEvent[]): CostTotals {
  const models = new Set<string>();
  const serviceTiers = new Set<string>();
  const totals: CostTotals = {
    inputTokens: 0,
    outputTokens: 0,
    cacheCreationTokens: 0,
    cacheReadTokens: 0,
    totalCostUsd: 0,
    webSearchRequests: 0,
    serverToolUseRequests: 0,
    models: [],
    serviceTiers: [],
    events: 0,
  };

  for (const event of events) {
    if (event.data.case !== "cost") continue;
    const payload = event.data.value as any;
    totals.events += 1;
    totals.inputTokens += payload.inputTokens || 0;
    totals.outputTokens += payload.outputTokens || 0;
    totals.cacheCreationTokens += payload.cacheCreationTokens || 0;
    totals.cacheReadTokens += payload.cacheReadTokens || 0;
    totals.totalCostUsd += payload.totalCostUsd || 0;
    totals.webSearchRequests += payload.webSearchRequests || 0;
    totals.serverToolUseRequests += payload.serverToolUseRequests || 0;
    if (payload.model) models.add(payload.model);
    if (payload.serviceTier) serviceTiers.add(payload.serviceTier);
  }

  totals.models = Array.from(models);
  totals.serviceTiers = Array.from(serviceTiers);
  return totals;
}

function formatCurrency(value: number): string {
  return `$${value.toFixed(4)}`;
}

function CostBreakdown({ totals }: { totals: CostTotals }) {
  const totalTokens =
    totals.inputTokens +
    totals.outputTokens +
    totals.cacheCreationTokens +
    totals.cacheReadTokens;

  return (
    <div className="space-y-3 text-xs">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="rounded border border-border p-3">
          <div className="text-muted-foreground">Total cost</div>
          <div className="text-lg font-semibold">{formatCurrency(totals.totalCostUsd)}</div>
        </div>
        <div className="rounded border border-border p-3">
          <div className="text-muted-foreground">Total tokens</div>
          <div className="text-lg font-semibold">{totalTokens.toLocaleString()}</div>
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-2">
        <div className="rounded border border-border p-3 space-y-1">
          <div className="text-muted-foreground">Token breakdown</div>
          <div>Input: {totals.inputTokens.toLocaleString()}</div>
          <div>Output: {totals.outputTokens.toLocaleString()}</div>
          <div>Cache creation: {totals.cacheCreationTokens.toLocaleString()}</div>
          <div>Cache read: {totals.cacheReadTokens.toLocaleString()}</div>
        </div>
        <div className="rounded border border-border p-3 space-y-1">
          <div className="text-muted-foreground">Request breakdown</div>
          <div>Web search: {totals.webSearchRequests.toLocaleString()}</div>
          <div>Server tool use: {totals.serverToolUseRequests.toLocaleString()}</div>
          <div>Cost events: {totals.events.toLocaleString()}</div>
        </div>
      </div>
      {(totals.models.length > 0 || totals.serviceTiers.length > 0) && (
        <div className="rounded border border-border p-3 space-y-1">
          <div className="text-muted-foreground">Usage context</div>
          {totals.models.length > 0 && <div>Models: {totals.models.join(", ")}</div>}
          {totals.serviceTiers.length > 0 && (
            <div>Service tiers: {totals.serviceTiers.join(", ")}</div>
          )}
        </div>
      )}
    </div>
  );
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
