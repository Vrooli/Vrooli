import { useMemo, useState, useRef, useEffect, useCallback } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  Clock,
  Copy,
  DollarSign,
  Eye,
  FileCode,
  File,
  FolderOpen,
  Link2,
  MessageSquare,
  MoreVertical,
  RotateCcw,
  Search,
  Square,
  StickyNote,
  Tag,
  Terminal,
  Trash2,
  Wrench,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { formatUsdFixed } from "../lib/currency";
import { cn, formatDuration, runnerTypeLabel } from "../lib/utils";
import { useCollapsiblePanel } from "../hooks/useCollapsiblePanel";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useViewportSize } from "../hooks/useViewportSize";
import { formatRelativeTimeShort, formatStandardDateTime } from "../lib/dateTime";
import type {
  ApproveFormData,
  ContextAttachmentData,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
  Task,
} from "../types";
import { RunEventType, RunMode, RunPhase, RunStatus, TaskStatus } from "../types";

import { MarkdownRenderer } from "./markdown";
import { CodeBlock } from "./markdown/components/CodeBlock";
import { ModelCostComparison } from "./ModelCostComparison";
import { ChatInterface } from "./ChatInterface";
import { ContextAttachmentModal } from "./ContextAttachmentModal";
import { DiffViewer } from "./DiffViewer";
import { ReviewModal } from "./ReviewModal";

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
  onStop: (run: Run) => Promise<void>;
  onDelete: (run: Run) => Promise<void>;
  onContinue: (message: string, attachmentIds?: string[]) => Promise<void>;
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
  onStop,
  onDelete,
  onContinue,
  onDeleteMessage,
  deleteLoading,
}: RunDetailProps) {
  const [activeTab, setActiveTab] = useState<"task" | "events" | "diff" | "messages" | "cost">("messages");
  const [showReviewModal, setShowReviewModal] = useState(false);
  const [eventFilter, setEventFilter] = useState<"all" | "errors" | "messages" | "tools" | "status">("all");
  const [eventsAutoScroll, setEventsAutoScroll] = useState(true);
  const [idCopied, setIdCopied] = useState(false);
  const [actionsMenuOpen, setActionsMenuOpen] = useState(false);
  const actionsMenuRef = useRef<HTMLDivElement>(null);
  const { isDesktop } = useViewportSize();

  const { isCollapsed: isDetailsCollapsed, toggle: toggleDetailsCollapsed } = useCollapsiblePanel({
    storageKey: "run.details",
    persistKey: "agm.runDetailsCollapsed",
    defaultCollapsed: false,
  });

  // Details section resize state
  const DETAILS_MIN_HEIGHT = 200;
  const TABS_MIN_HEIGHT = 200;
  const {
    size: detailsHeight,
    handleResizeStart,
    containerRef,
  } = useResizablePanel({
    storageKey: "run.details",
    persistKey: "agm.runDetailsHeight",
    axis: "vertical",
    defaultSize: 350,
    minSize: DETAILS_MIN_HEIGHT,
    minOtherSize: TABS_MIN_HEIGHT,
  });
  const eventsScrollRef = useRef<HTMLDivElement | null>(null);

  // Close actions menu on outside click
  useEffect(() => {
    if (!actionsMenuOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (actionsMenuRef.current && !actionsMenuRef.current.contains(e.target as Node)) {
        setActionsMenuOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [actionsMenuOpen]);

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

  return (
    <div className="h-full flex flex-col" ref={containerRef}>
      {/* Details Section (collapsible) */}
      <div
        className="flex-shrink-0 flex flex-col border-b border-border"
        style={isDetailsCollapsed || !isDesktop ? undefined : { height: detailsHeight }}
      >
        {/* Header - always visible, clickable to toggle */}
        <div
          className="flex items-center justify-between px-4 py-2 border-b border-border cursor-pointer hover:bg-muted/30 transition-colors"
          onClick={toggleDetailsCollapsed}
        >
          <div className="flex items-center gap-2">
            {isDetailsCollapsed ? (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            )}
            <span className="font-semibold text-sm">Details</span>
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
            <button
              type="button"
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                navigator.clipboard.writeText(run.id);
                setIdCopied(true);
                setTimeout(() => setIdCopied(false), 1500);
              }}
              title={`Copy run ID: ${run.id}`}
            >
              <code className="font-mono">{run.id.slice(0, 8)}</code>
              {idCopied ? <Check className="h-3 w-3 text-emerald-500" /> : <Copy className="h-3 w-3" />}
            </button>
          </div>
          {/* Action buttons — stop is always primary, rest are inline on desktop / ellipsis on mobile */}
          <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
            {actions?.canStop && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onStop(run)}
                className="gap-1 h-7 px-2 text-destructive hover:text-destructive"
                title="Stop run"
              >
                <Square className="h-3 w-3" />
                <span className="hidden sm:inline">Stop</span>
              </Button>
            )}
            {isDesktop ? (
              <>
                {actions?.canInvestigate && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onInvestigate(run.id)}
                    className="gap-1 h-7 px-2"
                    title="Investigate"
                  >
                    <Search className="h-3 w-3" />
                    <span className="hidden lg:inline">Investigate</span>
                  </Button>
                )}
                {canApplyFixes && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onApplyInvestigation(run.id)}
                    className="gap-1 h-7 px-2"
                    title="Apply Fixes"
                  >
                    <Wrench className="h-3 w-3" />
                    <span className="hidden lg:inline">Apply Fixes</span>
                  </Button>
                )}
                {actions?.canRetry && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRetry(run)}
                    className="gap-1 h-7 px-2"
                    title="Re-run"
                  >
                    <RotateCcw className="h-3 w-3" />
                    <span className="hidden lg:inline">Re-run</span>
                  </Button>
                )}
                {(actions?.canReview || actions?.canApprove || actions?.canReject) && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowReviewModal(true)}
                    className="gap-1 h-7 px-2"
                    title="Review changes"
                  >
                    <Eye className="h-3 w-3" />
                    <span className="hidden lg:inline">Review</span>
                  </Button>
                )}
                {canDeleteRun && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onDelete(run)}
                    className="gap-1 h-7 px-2 text-destructive hover:text-destructive"
                    disabled={deleteLoading}
                    title="Delete run"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                )}
              </>
            ) : (
              <div className="relative" ref={actionsMenuRef}>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setActionsMenuOpen((prev) => !prev)}
                  className="h-7 w-7 p-0"
                  aria-label="Run actions"
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
                {actionsMenuOpen && (
                  <div className="absolute right-0 top-full z-50 mt-1 min-w-[160px] rounded-md border border-border bg-card p-1 shadow-lg">
                    {actions?.canInvestigate && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                        onClick={() => { onInvestigate(run.id); setActionsMenuOpen(false); }}
                      >
                        <Search className="h-3.5 w-3.5" /> Investigate
                      </button>
                    )}
                    {canApplyFixes && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                        onClick={() => { onApplyInvestigation(run.id); setActionsMenuOpen(false); }}
                      >
                        <Wrench className="h-3.5 w-3.5" /> Apply Fixes
                      </button>
                    )}
                    {actions?.canRetry && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                        onClick={() => { onRetry(run); setActionsMenuOpen(false); }}
                      >
                        <RotateCcw className="h-3.5 w-3.5" /> Re-run
                      </button>
                    )}
                    {(actions?.canReview || actions?.canApprove || actions?.canReject) && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                        onClick={() => { setShowReviewModal(true); setActionsMenuOpen(false); }}
                      >
                        <Eye className="h-3.5 w-3.5" /> Review
                      </button>
                    )}
                    {canDeleteRun && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-destructive hover:bg-muted/50 transition-colors"
                        disabled={deleteLoading}
                        onClick={() => { onDelete(run); setActionsMenuOpen(false); }}
                      >
                        <Trash2 className="h-3.5 w-3.5" /> Delete
                      </button>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Content - hidden when collapsed */}
        {!isDetailsCollapsed && (
          <div className="flex-1 min-h-0 overflow-y-auto p-3 space-y-3 sm:p-4 sm:space-y-4">
            {/* Run Overview */}
            <div className="rounded-lg border border-border bg-card/50 p-3 sm:p-4">
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground">Run Overview</p>
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
                {run.resolvedConfig?.features?.enableBrowser && (
                  <div className="col-span-2 flex flex-wrap gap-1">
                    <Badge variant="outline">Browser</Badge>
                  </div>
                )}
                {run.resolvedConfig?.extraFlags && Object.entries(run.resolvedConfig.extraFlags).map(([rt, list]) =>
                  list.flags?.map((flag, i) => (
                    <Badge key={`${rt}-${i}`} variant="outline">{rt}: {flag}</Badge>
                  ))
                )}
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
            <div className="grid grid-cols-3 gap-2 text-sm">
              <div className="rounded border border-border px-3 py-2">
                <div className="text-xs text-muted-foreground">Duration</div>
                <div className="font-semibold">{durationMs !== null ? formatDuration(durationMs) : "—"}</div>
              </div>
              <div className="rounded border border-border px-3 py-2">
                <div className="text-xs text-muted-foreground">Cost</div>
                <div className="font-semibold">{costTotals.totalCostUsd ? formatCurrency(costTotals.totalCostUsd) : "$0.0000"}</div>
              </div>
              <div className="rounded border border-border px-3 py-2">
                <div className="text-xs text-muted-foreground">Changes</div>
                <div className="font-semibold">{run.changedFiles > 0 ? `${run.changedFiles} files` : "None"}</div>
              </div>
            </div>
          </div>
        )}

      </div>

      {/* Resize handle - only when expanded, desktop only */}
      {!isDetailsCollapsed && isDesktop && (
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
                "px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "task"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("task")}
            >
              <span className="hidden sm:inline mr-2"><File className="h-4 w-4 inline" /></span>
              Task
            </button>
            <button
              className={cn(
                "px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "events"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("events")}
            >
              <span className="hidden sm:inline mr-2"><Terminal className="h-4 w-4 inline" /></span>
              Events ({events.length})
            </button>
            <button
              className={cn(
                "px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "messages"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("messages")}
            >
              <span className="hidden sm:inline mr-2"><MessageSquare className="h-4 w-4 inline" /></span>
              Messages
            </button>
            <button
              className={cn(
                "px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "diff"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("diff")}
            >
              <span className="hidden sm:inline mr-2"><FileCode className="h-4 w-4 inline" /></span>
              Diff
            </button>
            <button
              className={cn(
                "px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                activeTab === "cost"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("cost")}
            >
              <span className="hidden sm:inline mr-2"><DollarSign className="h-4 w-4 inline" /></span>
              Cost
            </button>
          </div>

          {/* Tab content - fills remaining space, scrollable for most tabs but not Messages */}
          <div
            ref={eventsScrollRef}
            onScroll={activeTab === "events" ? handleEventsScroll : undefined}
            className={cn(
              "flex-1 min-h-0",
              activeTab === "messages" ? "flex flex-col" : "overflow-y-auto p-3 sm:p-4"
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
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-1.5">
                  {eventCounts.errors > 0 && (
                    <button
                      type="button"
                      onClick={() => setEventFilter("errors")}
                      className={cn(
                        "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors",
                        eventFilter === "errors"
                          ? "bg-destructive text-destructive-foreground"
                          : "bg-destructive/10 text-destructive hover:bg-destructive/20"
                      )}
                    >
                      Errors <span className="font-semibold">{eventCounts.errors}</span>
                    </button>
                  )}
                  {(["all", "messages", "tools", "status"] as const).map((filter) => (
                    <button
                      key={filter}
                      type="button"
                      onClick={() => setEventFilter(filter)}
                      className={cn(
                        "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors",
                        eventFilter === filter
                          ? "bg-primary text-primary-foreground"
                          : "bg-muted text-muted-foreground hover:bg-muted/80 hover:text-foreground"
                      )}
                    >
                      {filter === "all" ? "All" : filter}
                      <span className="opacity-70">{eventCounts[filter]}</span>
                    </button>
                  ))}
                </div>
                <div className="space-y-0.5">
                  {filteredEvents.map((event) => (
                    <EventItem key={event.id} event={event} />
                  ))}
                </div>
              </div>
            )
          ) : activeTab === "messages" ? (
            <div className="flex-1 min-h-0 p-3 sm:p-4">
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
            <div className="space-y-3 sm:space-y-4">
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 text-sm">
                <div className="rounded border border-border px-3 py-2">
                  <div className="text-xs text-muted-foreground">Total Cost</div>
                  <div className="font-semibold">{formatCurrency(costTotals.totalCostUsd)}</div>
                </div>
                <div className="rounded border border-border px-3 py-2">
                  <div className="text-xs text-muted-foreground">Total Tokens</div>
                  <div className="font-semibold">{totalTokens.toLocaleString()}</div>
                </div>
                <div className="rounded border border-border px-3 py-2">
                  <div className="text-xs text-muted-foreground">Output Tokens</div>
                  <div className="font-semibold">{costTotals.outputTokens.toLocaleString()}</div>
                </div>
                <div className="rounded border border-border px-3 py-2">
                  <div className="text-xs text-muted-foreground">Requests</div>
                  <div className="font-semibold">{(costTotals.webSearchRequests + costTotals.serverToolUseRequests).toLocaleString()}</div>
                </div>
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

      <ReviewModal
        open={showReviewModal}
        onOpenChange={setShowReviewModal}
        run={run}
        diff={diff}
        diffLoading={diffLoading}
        onApprove={onApprove}
        onReject={onReject}
      />
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
          <h4 className="text-xs font-medium text-muted-foreground">Scope</h4>
          <div className="flex items-center gap-2">
            <FolderOpen className="h-4 w-4 text-muted-foreground" />
            <code className="text-xs bg-muted px-2 py-1 rounded">{task.scopePath}</code>
          </div>
        </div>

        {task.projectRoot && (
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground">Project Root</h4>
            <code className="text-xs bg-muted px-2 py-1 rounded">{task.projectRoot}</code>
          </div>
        )}

        {task.contextAttachments && task.contextAttachments.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-xs font-medium text-muted-foreground">
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
  const payloadValue = payload.value as Record<string, unknown> | undefined;

  const getIcon = () => {
    switch (event.eventType) {
      case RunEventType.LOG:
        return <Terminal className="h-3.5 w-3.5" />;
      case RunEventType.MESSAGE:
        return <MessageSquare className="h-3.5 w-3.5" />;
      case RunEventType.TOOL_CALL:
      case RunEventType.TOOL_RESULT:
        return <Wrench className="h-3.5 w-3.5" />;
      case RunEventType.STATUS:
        return <Activity className="h-3.5 w-3.5" />;
      case RunEventType.ERROR:
        return <AlertCircle className="h-3.5 w-3.5" />;
      default:
        return <ChevronRight className="h-3.5 w-3.5" />;
    }
  };

  const getAccentColor = () => {
    switch (event.eventType) {
      case RunEventType.ERROR:
        return "border-l-destructive text-destructive";
      case RunEventType.STATUS:
        return "border-l-primary text-primary";
      case RunEventType.TOOL_CALL:
      case RunEventType.TOOL_RESULT:
        return "border-l-warning text-warning";
      case RunEventType.MESSAGE:
        return "border-l-success text-success";
      default:
        return "border-l-muted-foreground text-muted-foreground";
    }
  };

  const getSummary = () => {
    const v = payloadValue ?? {};
    switch (payload.case) {
      case "log": {
        const msg = String(v.message ?? "Log entry");
        // Strip "phase: " prefix — the badge already shows type
        return msg.replace(/^phase:\s*/i, "");
      }
      case "message":
        return String(v.role ?? "unknown") + ": " + String(v.content ?? "").slice(0, 120);
      case "toolCall":
        return String(v.toolName ?? "unknown tool");
      case "toolResult":
        return (v.success ? "OK" : "Failed") + " — " + String(v.toolName ?? "");
      case "status":
        return (
          runStatusLabel((v.oldStatus as RunStatus) ?? RunStatus.UNSPECIFIED) +
          " -> " +
          runStatusLabel((v.newStatus as RunStatus) ?? RunStatus.UNSPECIFIED)
        );
      case "metric":
        return `${String(v.name ?? "metric")}: ${v.value ?? 0}`;
      case "artifact":
        return v.path ? String(v.path) : "Artifact";
      case "progress":
        return `Progress ${v.percentComplete ?? 0}%`;
      case "cost":
        return "Cost update";
      case "rateLimit":
        return "Rate limit";
      case "error":
        return String(v.message ?? v.code ?? "Error occurred");
      default:
        return runEventTypeLabel(event.eventType);
    }
  };

  const accentClasses = getAccentColor();
  const iconColor = accentClasses.split(" ").slice(1).join(" ");

  return (
    <div
      className={cn(
        "border-l-2 rounded-r-md bg-card/50 text-xs transition-colors hover:bg-muted/30",
        accentClasses.split(" ")[0]
      )}
    >
      <div
        className="flex items-center gap-2 px-3 py-1.5 cursor-pointer"
        onClick={() => setExpanded(!expanded)}
      >
        <span className={cn("shrink-0", iconColor)}>{getIcon()}</span>
        <span className="text-[10px] font-medium text-muted-foreground shrink-0">
          {runEventTypeLabel(event.eventType).replace("_", " ")}
        </span>
        <span className="flex-1 truncate text-sm text-foreground">
          {getSummary()}
        </span>
        <span className="text-[10px] text-muted-foreground shrink-0">
          {formatRelativeTimeShort(event.timestamp)}
        </span>
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 text-muted-foreground shrink-0" />
        )}
      </div>
      {expanded && (
        <div className="px-3 pb-2 pt-1 space-y-1 border-t border-border/50">
          <div className="text-[10px] text-muted-foreground">
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
    const p = event.data.value as Record<string, unknown> | undefined;
    if (!p) continue;
    totals.events += 1;
    totals.inputTokens += Number(p.inputTokens ?? 0);
    totals.outputTokens += Number(p.outputTokens ?? 0);
    totals.cacheCreationTokens += Number(p.cacheCreationTokens ?? 0);
    totals.cacheReadTokens += Number(p.cacheReadTokens ?? 0);
    totals.totalCostUsd += Number(p.totalCostUsd ?? 0);
    totals.webSearchRequests += Number(p.webSearchRequests ?? 0);
    totals.serverToolUseRequests += Number(p.serverToolUseRequests ?? 0);
    if (p.model) models.add(String(p.model));
    if (p.serviceTier) serviceTiers.add(String(p.serviceTier));
  }

  totals.models = Array.from(models);
  totals.serviceTiers = Array.from(serviceTiers);
  return totals;
}

function formatCurrency(value: number): string {
  return formatUsdFixed(value, 4, { useGrouping: false });
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

