import { useMemo, useState, useRef, useEffect, useCallback } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  Clock,
  DollarSign,
  ExternalLink,
  FileCode,
  File,
  FolderOpen,
  Link2,
  MessageSquare,
  RotateCcw,
  Search,
  StickyNote,
  Tag,
  Terminal,
  Trash2,
  Wrench,
  X,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { buildSandboxReviewUrl, cn, formatDate, formatDuration, runnerTypeLabel } from "../lib/utils";
import type {
  ApproveFormData,
  ContextAttachmentData,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
  Task,
} from "../types";
import { ApprovalState, RunEventType, RunMode, RunPhase, RunStatus, TaskStatus } from "../types";
import { KPICard } from "../features/stats/components/kpi/KPICard";
import { MarkdownRenderer } from "./markdown";
import { CodeBlock } from "./markdown/components/CodeBlock";
import { ModelCostComparison } from "./ModelCostComparison";
import { ChatInterface } from "./ChatInterface";
import { ContextAttachmentModal } from "./ContextAttachmentModal";

interface RunDetailProps {
  run: Run;
  events: RunEvent[];
  diff: RunDiff | null;
  eventsLoading: boolean;
  diffLoading: boolean;
  task?: Task | null;
  taskTitle: string;
  profileName: string;
  onApprove: (req: ApproveFormData) => Promise<void>;
  onReject: (req: RejectFormData) => Promise<void>;
  onRetry: (run: Run) => Promise<Run>;
  onInvestigate: (runId: string) => void;
  onApplyInvestigation: (runId: string) => void;
  onDelete: (run: Run) => Promise<void>;
  onContinue: (message: string) => Promise<void>;
  onDeleteMessage: (eventId: string) => Promise<void>;
  deleteLoading: boolean;
}

export function RunDetail({
  run,
  events,
  diff,
  eventsLoading,
  diffLoading,
  task,
  taskTitle,
  profileName,
  onApprove,
  onReject,
  onRetry,
  onInvestigate,
  onApplyInvestigation,
  onDelete,
  onContinue,
  onDeleteMessage,
  deleteLoading,
}: RunDetailProps) {
  const [activeTab, setActiveTab] = useState<"task" | "events" | "diff" | "messages" | "cost">("events");
  const [approvalForm, setApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [rejectForm, setRejectForm] = useState({ actor: "", reason: "" });
  const [showRejectConfirm, setShowRejectConfirm] = useState(false);
  const [showApprovalForm, setShowApprovalForm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [eventFilter, setEventFilter] = useState<"all" | "errors" | "messages" | "tools" | "status">("all");
  const [eventsAutoScroll, setEventsAutoScroll] = useState(true);

  // Collapsed state for details section (persisted in localStorage)
  const [isDetailsCollapsed, setIsDetailsCollapsed] = useState(() => {
    if (typeof window === "undefined") return false;
    return localStorage.getItem("agm.runDetailsCollapsed") === "true";
  });

  // Persist collapsed state to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("agm.runDetailsCollapsed", String(isDetailsCollapsed));
  }, [isDetailsCollapsed]);

  // Details section resize state
  const DETAILS_MIN_HEIGHT = 200;
  const TABS_MIN_HEIGHT = 200;
  const [detailsHeight, setDetailsHeight] = useState(() => {
    if (typeof window === "undefined") return 350;
    const stored = Number(localStorage.getItem("agm.runDetailsHeight"));
    return Number.isFinite(stored) && stored > 0 ? stored : 350;
  });
  const [isResizing, setIsResizing] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const eventsScrollRef = useRef<HTMLDivElement | null>(null);
  const resizeRef = useRef<{ top: number; height: number } | null>(null);

  // Resize handler
  const handleResizeStart = (event: React.MouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    if (!containerRef.current) return;

    const rect = containerRef.current.getBoundingClientRect();
    resizeRef.current = {
      top: rect.top,
      height: rect.height,
    };
    setIsResizing(true);
  };

  // Resize mouse events
  useEffect(() => {
    if (!isResizing) return;

    const handleMove = (event: MouseEvent) => {
      if (!resizeRef.current) return;
      const nextHeight = event.clientY - resizeRef.current.top;
      const maxHeight = resizeRef.current.height - TABS_MIN_HEIGHT;
      const clampedHeight = Math.max(DETAILS_MIN_HEIGHT, Math.min(maxHeight, nextHeight));
      setDetailsHeight(clampedHeight);
    };

    const handleUp = () => {
      setIsResizing(false);
      resizeRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "row-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);

    return () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing]);

  // Persist details height to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem("agm.runDetailsHeight", String(detailsHeight));
  }, [detailsHeight]);

  // Constrain details height when container shrinks
  useEffect(() => {
    if (!containerRef.current || typeof ResizeObserver === "undefined") return;

    const clamp = () => {
      if (!containerRef.current) return;
      const height = containerRef.current.clientHeight;
      const maxDetails = Math.max(DETAILS_MIN_HEIGHT, height - TABS_MIN_HEIGHT);
      if (detailsHeight > maxDetails) {
        setDetailsHeight(maxDetails);
      }
    };

    clamp();
    const observer = new ResizeObserver(clamp);
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [detailsHeight]);

  const costTotals = getCostTotals(events);
  const durationMs =
    run.endedAt && run.startedAt ? timestampMs(run.endedAt) - timestampMs(run.startedAt) : null;
  const totalTokens =
    costTotals.inputTokens +
    costTotals.outputTokens +
    costTotals.cacheCreationTokens +
    costTotals.cacheReadTokens;

  const actions = run.actions;
  const canDeleteRun = actions?.canDelete ?? false;
  const canApplyFixes = actions?.canApplyInvestigation ?? false;
  const runVariant = getRunVariant(run.status);

  const eventCounts = useMemo(() => {
    const counts = {
      all: events.length,
      errors: 0,
      messages: 0,
      tools: 0,
      status: 0,
    };
    for (const event of events) {
      if (event.eventType === RunEventType.ERROR) counts.errors += 1;
      if (event.eventType === RunEventType.MESSAGE) counts.messages += 1;
      if (event.eventType === RunEventType.TOOL_CALL || event.eventType === RunEventType.TOOL_RESULT) counts.tools += 1;
      if (event.eventType === RunEventType.STATUS) counts.status += 1;
    }
    return counts;
  }, [events]);

  const filteredEvents = useMemo(() => {
    switch (eventFilter) {
      case "errors":
        return events.filter((event) => event.eventType === RunEventType.ERROR);
      case "messages":
        return events.filter((event) => event.eventType === RunEventType.MESSAGE);
      case "tools":
        return events.filter(
          (event) => event.eventType === RunEventType.TOOL_CALL || event.eventType === RunEventType.TOOL_RESULT
        );
      case "status":
        return events.filter((event) => event.eventType === RunEventType.STATUS);
      default:
        return events;
    }
  }, [eventFilter, events]);

  const handleEventsScroll = useCallback(() => {
    const container = eventsScrollRef.current;
    if (!container) return;
    const distanceToBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
    setEventsAutoScroll(distanceToBottom <= 24);
  }, []);

  useEffect(() => {
    if (activeTab !== "events" || !eventsAutoScroll) return;
    const container = eventsScrollRef.current;
    if (!container) return;
    container.scrollTop = container.scrollHeight;
  }, [activeTab, eventsAutoScroll, filteredEvents]);

  useEffect(() => {
    if (activeTab !== "events") return;
    const container = eventsScrollRef.current;
    if (!container) return;
    container.scrollTop = container.scrollHeight;
    setEventsAutoScroll(true);
  }, [activeTab]);

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
    <div className="h-full flex flex-col" ref={containerRef}>
      {/* Details Section (collapsible) */}
      <div
        className="flex-shrink-0 flex flex-col border-b border-border"
        style={isDetailsCollapsed ? undefined : { height: detailsHeight }}
      >
        {/* Header - always visible, clickable to toggle */}
        <div
          className="flex items-center justify-between px-4 py-3 border-b border-border cursor-pointer hover:bg-muted/30 transition-colors"
          onClick={() => setIsDetailsCollapsed(!isDetailsCollapsed)}
        >
          <div className="flex items-center gap-2">
            {isDetailsCollapsed ? (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            )}
            <span className="font-semibold text-sm">Details</span>
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
            {run.approvalState !== ApprovalState.NONE && (
              <Badge variant="outline">{approvalStateLabel(run.approvalState)}</Badge>
            )}
            {/* Show task title summary when collapsed */}
            {isDetailsCollapsed && (
              <span className="text-xs text-muted-foreground truncate max-w-[200px]">
                {taskTitle}
              </span>
            )}
          </div>
        </div>

        {/* Content - hidden when collapsed */}
        {!isDetailsCollapsed && (
          <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
            {/* Run Overview */}
            <div className="rounded-lg border border-border bg-card/50 p-4">
              <div className="space-y-1">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Run Overview</p>
                <h3 className="text-lg font-semibold">{taskTitle}</h3>
                <p className="text-sm text-muted-foreground">{profileName}</p>
              </div>

              {run.errorMsg && (
                <div className="mt-3 rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
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

              <div className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
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
                {run.resolvedConfig?.runnerType ? (
                  <div>
                    <span className="text-muted-foreground">Runner: </span>
                    {runnerTypeLabel(run.resolvedConfig.runnerType)}
                  </div>
                ) : null}
                {run.resolvedConfig?.fallbackRunnerTypes?.length ? (
                  <div>
                    <span className="text-muted-foreground">Fallbacks: </span>
                    {run.resolvedConfig.fallbackRunnerTypes
                      .map((runnerType) => runnerTypeLabel(runnerType))
                      .join(", ")}
                  </div>
                ) : null}
                {run.sandboxId && (
                  <div>
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
                {durationMs !== null && (
                  <div>
                    <span className="text-muted-foreground">Duration: </span>
                    {formatDuration(durationMs)}
                  </div>
                )}
                <div>
                  <span className="text-muted-foreground">Tag: </span>
                  {run.tag ? (
                    <code className="text-xs bg-muted px-1 py-0.5 rounded">{run.tag}</code>
                  ) : (
                    <span className="text-muted-foreground">None</span>
                  )}
                </div>
              </div>
            </div>

            {/* Highlights */}
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <KPICard
                title="Status"
                value={runStatusLabel(run.status).replace("_", " ")}
                subtitle={runPhaseLabel(run.phase).replace("_", " ")}
                icon={<Activity className="h-4 w-4" />}
                variant={runVariant}
              />
              <KPICard
                title="Duration"
                value={durationMs !== null ? formatDuration(durationMs) : "In progress"}
                subtitle={run.startedAt ? formatDate(run.startedAt) : "Not started"}
                icon={<Clock className="h-4 w-4" />}
              />
              <KPICard
                title="Total Cost"
                value={costTotals.totalCostUsd ? formatCurrency(costTotals.totalCostUsd) : "$0.0000"}
                subtitle={`${totalTokens.toLocaleString()} tokens`}
                icon={<DollarSign className="h-4 w-4" />}
              />
              <KPICard
                title="Changes"
                value={run.changedFiles > 0 ? `${run.changedFiles} files` : "No changes"}
                subtitle={diff?.files?.length ? `${diff.files.length} files in diff` : "Diff pending"}
                icon={<FileCode className="h-4 w-4" />}
              />
            </div>
          </div>
        )}

        {/* Actions bar - ALWAYS VISIBLE */}
        <div
          className="px-4 py-3 border-t border-border flex flex-wrap items-center gap-2"
          onClick={(e) => e.stopPropagation()}
        >
          {actions?.canInvestigate && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onInvestigate(run.id)}
              className="gap-1"
            >
              <Search className="h-3 w-3" />
              Investigate
            </Button>
          )}
          {canApplyFixes && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => onApplyInvestigation(run.id)}
              className="gap-1"
            >
              <Wrench className="h-3 w-3" />
              Apply Fixes
            </Button>
          )}
          {actions?.canRetry && (
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

          {/* Review button - opens sandbox in workspace-sandbox UI */}
          {actions?.canReview && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const url = buildSandboxReviewUrl(run.sandboxId!);
                window.open(url, "_blank", "noopener,noreferrer");
              }}
              className="gap-1"
            >
              <ExternalLink className="h-3 w-3" />
              Review
            </Button>
          )}

          {/* Approval buttons - for NEEDS_REVIEW runs */}
          {(actions?.canApprove || actions?.canReject) && (
            <>
              <Button
                variant="success"
                size="sm"
                onClick={() => {
                  setShowApprovalForm(true);
                  setShowRejectConfirm(false);
                }}
                disabled={submitting || !actions?.canApprove}
                className="gap-1"
              >
                <Check className="h-3 w-3" />
                Approve
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => {
                  setShowRejectConfirm(true);
                  setShowApprovalForm(false);
                  setRejectForm({
                    actor: approvalForm.actor,
                    reason: "",
                  });
                }}
                disabled={submitting || !actions?.canReject}
                className="gap-1"
              >
                <X className="h-3 w-3" />
                Reject
              </Button>
            </>
          )}

          {/* Delete button - right aligned */}
          {canDeleteRun && (
            <div className="ml-auto">
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
            </div>
          )}
        </div>

        {/* Inline approval form - shown when approve clicked */}
        {showApprovalForm && (
          <div className="px-4 py-3 border-t border-border bg-muted/30 space-y-3">
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
                size="sm"
                onClick={handleApprove}
                disabled={submitting}
                className="gap-2"
              >
                <Check className="h-4 w-4" />
                Confirm Approval
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowApprovalForm(false)}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        {/* Inline rejection form - shown when reject clicked */}
        {showRejectConfirm && (
          <div className="px-4 py-3 border-t border-border bg-destructive/5 space-y-3">
            <div className="grid gap-3 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="rejectActor">Your Name (optional)</Label>
                <Input
                  id="rejectActor"
                  value={rejectForm.actor}
                  onChange={(e) =>
                    setRejectForm({
                      ...rejectForm,
                      actor: e.target.value,
                    })
                  }
                  placeholder="Leave blank for anonymous"
                />
              </div>
              <div className="space-y-2">
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
              </div>
            </div>
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

      {/* Resize handle - only when expanded */}
      {!isDetailsCollapsed && (
        <div
          className="h-1.5 bg-border hover:bg-primary/50 cursor-row-resize flex-shrink-0"
          onMouseDown={handleResizeStart}
        />
      )}

      {/* Tabs Section - fills remaining space */}
      <div className="flex-1 min-h-0 flex flex-col">
        <div className="h-full rounded-lg border border-border bg-card/50 flex flex-col overflow-hidden">
          <div className="flex border-b border-border overflow-x-auto overflow-y-hidden shrink-0">
            <button
              className={cn(
                "px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "task"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("task")}
            >
              <File className="h-4 w-4 inline mr-2" />
              Task
            </button>
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
                activeTab === "messages"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("messages")}
            >
              <MessageSquare className="h-4 w-4 inline mr-2" />
              Messages
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

          {/* Tab content - fills remaining space, scrollable for most tabs but not Messages */}
          <div
            ref={eventsScrollRef}
            onScroll={activeTab === "events" ? handleEventsScroll : undefined}
            className={cn(
              "flex-1 min-h-0",
              activeTab === "messages" ? "flex flex-col" : "overflow-y-auto p-4"
            )}
          >
            {activeTab === "task" ? (
            task ? (
              <TaskSummary task={task} />
            ) : (
              <div className="py-8 text-center text-muted-foreground">
                Task details unavailable for {run.taskId}
              </div>
            )
          ) : activeTab === "events" ? (
            eventsLoading ? (
              <div className="py-8 text-center text-muted-foreground">
                Loading events...
              </div>
            ) : events.length === 0 ? (
              <div className="py-8 text-center text-muted-foreground">
                No events recorded
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    variant={eventFilter === "errors" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setEventFilter("errors")}
                    className="gap-2"
                    disabled={eventCounts.errors === 0}
                  >
                    Jump to errors
                    <Badge variant="secondary">{eventCounts.errors}</Badge>
                  </Button>
                  {(["all", "messages", "tools", "status"] as const).map((filter) => (
                    <Button
                      key={filter}
                      variant={eventFilter === filter ? "default" : "outline"}
                      size="sm"
                      onClick={() => setEventFilter(filter)}
                      className="gap-2"
                    >
                      {filter === "all" ? "All" : filter}
                      <Badge variant="secondary">
                        {eventCounts[filter]}
                      </Badge>
                    </Button>
                  ))}
                </div>
                <div className="space-y-2">
                  {filteredEvents.map((event) => (
                    <EventItem key={event.id} event={event} />
                  ))}
                </div>
              </div>
            )
          ) : activeTab === "messages" ? (
            <div className="flex-1 min-h-0 p-4">
              <ChatInterface
                run={run}
                events={events}
                eventsLoading={eventsLoading}
                onContinue={onContinue}
                onDeleteMessage={onDeleteMessage}
              />
            </div>
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
            <div className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <KPICard
                  title="Total Cost"
                  value={formatCurrency(costTotals.totalCostUsd)}
                  subtitle={`${costTotals.events} events`}
                  icon={<DollarSign className="h-4 w-4" />}
                />
                <KPICard
                  title="Total Tokens"
                  value={totalTokens.toLocaleString()}
                  subtitle={`${costTotals.inputTokens.toLocaleString()} input`}
                  icon={<Terminal className="h-4 w-4" />}
                />
                <KPICard
                  title="Output Tokens"
                  value={costTotals.outputTokens.toLocaleString()}
                  subtitle={`${costTotals.cacheReadTokens.toLocaleString()} cache read`}
                  icon={<MessageSquare className="h-4 w-4" />}
                />
                <KPICard
                  title="Requests"
                  value={(costTotals.webSearchRequests + costTotals.serverToolUseRequests).toLocaleString()}
                  subtitle={`${costTotals.serverToolUseRequests} tool calls`}
                  icon={<Wrench className="h-4 w-4" />}
                />
              </div>
              <CostBreakdown totals={costTotals} />
              <ModelCostComparison
                costTotals={costTotals}
                actualModel={costTotals.models[0] || run.resolvedConfig?.model || ""}
              />
            </div>
          )}
        </div>
        </div>
      </div>
    </div>
  );
}

// Helper functions and sub-components
function getRunVariant(status: RunStatus): "default" | "success" | "warning" | "error" {
  switch (status) {
    case RunStatus.COMPLETE:
      return "success";
    case RunStatus.NEEDS_REVIEW:
      return "warning";
    case RunStatus.FAILED:
      return "error";
    default:
      return "default";
  }
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

function taskStatusLabel(status: TaskStatus): string {
  switch (status) {
    case TaskStatus.QUEUED:
      return "queued";
    case TaskStatus.RUNNING:
      return "running";
    case TaskStatus.NEEDS_REVIEW:
      return "needs_review";
    case TaskStatus.APPROVED:
      return "approved";
    case TaskStatus.REJECTED:
      return "rejected";
    case TaskStatus.FAILED:
      return "failed";
    case TaskStatus.CANCELLED:
      return "cancelled";
    default:
      return "queued";
  }
}

function TaskSummary({ task }: { task: Task }) {
  const [selectedAttachment, setSelectedAttachment] = useState<ContextAttachmentData | null>(null);

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center gap-2 flex-wrap">
          <h3 className="text-lg font-semibold">{task.title}</h3>
          <Badge
            variant={
              taskStatusLabel(task.status) as
                | "queued"
                | "running"
                | "needs_review"
                | "approved"
                | "rejected"
                | "failed"
                | "cancelled"
            }
          >
            {taskStatusLabel(task.status).replace("_", " ")}
          </Badge>
        </div>
        {task.description ? (
          <div className="text-sm text-muted-foreground">
            <MarkdownRenderer content={task.description} />
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">No description provided</p>
        )}
      </div>

      <div className="space-y-3 text-sm">
        <div className="space-y-2">
          <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Scope</h4>
          <div className="flex items-center gap-2">
            <FolderOpen className="h-4 w-4 text-muted-foreground" />
            <code className="text-xs bg-muted px-2 py-1 rounded">{task.scopePath}</code>
          </div>
        </div>

        {task.projectRoot && (
          <div className="space-y-2">
            <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Project Root</h4>
            <code className="text-xs bg-muted px-2 py-1 rounded">{task.projectRoot}</code>
          </div>
        )}

        {task.contextAttachments && task.contextAttachments.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              Context Attachments
            </h4>
            <div className="space-y-2">
              {task.contextAttachments.map((att, index) => (
                <div
                  key={`${att.key || att.label || att.type}-${index}`}
                  className="flex items-start gap-2 p-2 bg-muted rounded-md text-sm cursor-pointer hover:bg-muted/70 transition-colors"
                  onClick={() => setSelectedAttachment(att as ContextAttachmentData)}
                >
                  {att.type === "file" && <File className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  {att.type === "link" && <Link2 className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  {att.type === "note" && <StickyNote className="h-4 w-4 text-muted-foreground shrink-0 mt-0.5" />}
                  <div className="flex-1 min-w-0 space-y-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      {att.key && (
                        <code className="text-[10px] bg-background px-1 py-0.5 rounded">
                          {att.key}
                        </code>
                      )}
                      {att.label && <span className="font-medium">{att.label}</span>}
                      {!att.key && !att.label && (
                        <span className="text-muted-foreground capitalize">{att.type}</span>
                      )}
                    </div>
                    {att.path && <p className="text-xs text-muted-foreground truncate">{att.path}</p>}
                    {att.url && <p className="text-xs text-muted-foreground truncate">{att.url}</p>}
                    {att.content && att.type === "note" && (
                      <p className="text-xs text-muted-foreground line-clamp-2">{att.content}</p>
                    )}
                    {att.tags && att.tags.length > 0 && (
                      <div className="flex gap-1 flex-wrap">
                        {att.tags.map((tag, i) => (
                          <Badge key={`${tag}-${i}`} variant="outline" className="text-[10px] gap-1 py-0">
                            <Tag className="h-2.5 w-2.5" />
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      <ContextAttachmentModal
        attachment={selectedAttachment}
        open={selectedAttachment !== null}
        onOpenChange={(open) => !open && setSelectedAttachment(null)}
      />
    </div>
  );
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

  const getBadgeVariant = () => {
    switch (event.eventType) {
      case RunEventType.ERROR:
        return "destructive";
      case RunEventType.STATUS:
        return "secondary";
      case RunEventType.TOOL_CALL:
      case RunEventType.TOOL_RESULT:
        return "warning";
      case RunEventType.MESSAGE:
        return "success";
      default:
        return "outline";
    }
  };

  const getCardStyles = () => {
    switch (event.eventType) {
      case RunEventType.ERROR:
        return "border-destructive/40 bg-destructive/5";
      case RunEventType.STATUS:
        return "border-primary/30 bg-primary/5";
      case RunEventType.TOOL_CALL:
      case RunEventType.TOOL_RESULT:
        return "border-warning/30 bg-warning/5";
      case RunEventType.MESSAGE:
        return "border-success/30 bg-success/5";
      default:
        return "border-border bg-card/50";
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
        "rounded-lg border p-3 text-xs transition-colors",
        getCardStyles()
      )}
    >
      <div
        className="flex items-start gap-3 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <div className={cn("mt-0.5 flex h-8 w-8 items-center justify-center rounded-md bg-muted/50")}>
          {getIcon()}
        </div>
        <div className="flex-1 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={getBadgeVariant()}>
              {runEventTypeLabel(event.eventType).replace("_", " ")}
            </Badge>
            <span className="text-[10px] uppercase tracking-wide text-muted-foreground">
              {formatDate(event.timestamp)}
            </span>
          </div>
          <div className="text-sm font-medium text-foreground">
            {getSummary()}
          </div>
        </div>
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-4 w-4 text-muted-foreground" />
        )}
      </div>
      {expanded && (
        <div className="mt-2 space-y-2">
          <div className="text-[10px] uppercase tracking-wide text-muted-foreground">
            Payload
          </div>
          <CodeBlock code={JSON.stringify(payloadValue, null, 2)} language="json" />
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
    <div className="space-y-4">
      <div className="rounded-lg border border-border bg-card/50 p-4 space-y-3">
        <div className="flex gap-4 text-xs">
          <span className="text-success">+{totals.additions}</span>
          <span className="text-destructive">-{totals.deletions}</span>
          <span className="text-muted-foreground">{fileCount} files</span>
        </div>

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
      </div>

      {diff.content && (
        <div className="rounded-lg border border-border bg-card/50 p-4">
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
        </div>
      )}
    </div>
  );
}
