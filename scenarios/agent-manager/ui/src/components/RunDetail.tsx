import type React from "react";
import { useState, useRef, useEffect, useCallback } from "react";
import { durationMs as protoDurationMs, timestampMs } from "@bufbuild/protobuf/wkt";
import {
  Activity,
  AlertCircle,
  Check,
  ChevronDown,
  ChevronRight,
  Copy,
  DollarSign,
  Eye,
  FileCode,
  File,
  FolderOpen,
  Info,
  Link2,
  MoreVertical,
  PauseCircle,
  PlayCircle,
  RotateCcw,
  Search,
  Square,
  StickyNote,
  Tag,
  Trash2,
  Wrench,
  ExternalLink,
  X,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { formatUsdFixed } from "../lib/currency";
import { cn, formatDuration, runnerTypeLabel } from "../lib/utils";
import { useCollapsiblePanel } from "../hooks/useCollapsiblePanel";
import { useRunReport } from "../hooks/useApi";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useViewportSize } from "../hooks/useViewportSize";
import type {
  ApproveFormData,
  ContextAttachmentData,
  RejectFormData,
  Run,
  RunDiff,
  RunEvent,
  Task,
} from "../types";
import { ApprovalState, ExecutionMode, RunMode, RunPhase, RunStatus, TaskStatus } from "../types";

import { MarkdownRenderer } from "./markdown";
import { ModelCostComparison } from "./ModelCostComparison";
import { RunTimeline } from "./RunTimeline";
import { FallbackTimeline } from "./runs/FallbackTimeline";
import { ContextAttachmentModal } from "./ContextAttachmentModal";
import { DiffViewer } from "./DiffViewer";
import { ReviewModal } from "./ReviewModal";

import {
  CostBreakdown,
  executionModeLabel,
  formatCurrency,
  getCostTotals,
  isInteractiveRun,
  MobileHeaderActions,
  RunDetailsContent,
  RunModelBadge,
  STATUS_DOT_COLORS,
  runModeLabel,
  runPhaseLabel,
  runStatusLabel,
  StatusDotWithLegend,
  TaskSummary,
  taskStatusLabel,
} from "./RunDetailParts";

type TabId = "task" | "timeline" | "diff" | "cost" | "report";

interface RunDetailProps {
  run: Run;
  events: RunEvent[];
  diff: RunDiff | null;
  eventsLoading: boolean;
  diffLoading: boolean;
  initialTab?: TabId;
  task?: Task | null;
  taskTitle: string;
  profileName: string;
  onApprove: (req: ApproveFormData) => Promise<void>;
  onReject: (req: RejectFormData) => Promise<void>;
  onPartialApprove?: (fileIds: string[], actor?: string, commitMsg?: string) => Promise<unknown>;
  onRetry: (run: Run) => Promise<Run>;
  onResumeFromFailure: (run: Run) => void;
  onInvestigate: (runId: string) => void;
  onApplyInvestigation: (runId: string) => void;
  onStop: (run: Run) => Promise<void>;
  onDelete: (run: Run) => void;
  onContinue: (message: string, attachmentIds?: string[]) => Promise<void>;
  onDeleteMessage: (eventId: string) => Promise<void>;
  deleteLoading: boolean;
  onMobileHeaderLeft?: (node: React.ReactNode | null) => void;
  onMobileHeaderRight?: (node: React.ReactNode | null) => void;
}

export function RunDetail({
  run,
  events,
  diff,
  eventsLoading,
  diffLoading,
  initialTab,
  task,
  taskTitle,
  profileName,
  onApprove,
  onReject,
  onPartialApprove,
  onRetry,
  onResumeFromFailure,
  onInvestigate,
  onApplyInvestigation,
  onStop,
  onDelete,
  onContinue,
  onDeleteMessage,
  deleteLoading,
  onMobileHeaderLeft,
  onMobileHeaderRight,
}: RunDetailProps) {
	const report = useRunReport(run.id);
  const [activeTab, setActiveTab] = useState<TabId>(initialTab ?? "timeline");

  // Reset tab when switching runs or when initialTab changes
  useEffect(() => {
    if (initialTab) setActiveTab(initialTab);
  }, [initialTab, run.id]);

  const [showReviewModal, setShowReviewModal] = useState(false);
  const [idCopied, setIdCopied] = useState(false);
  const [infoOpen, setInfoOpen] = useState(false);
  const [actionsMenuOpen, setActionsMenuOpen] = useState(false);
  const actionsMenuRef = useRef<HTMLDivElement>(null);
  const { isDesktop } = useViewportSize();

  // Inline review state
  const [inlineAction, setInlineAction] = useState<"none" | "approve" | "reject" | "partial">("none");
  const [inlineApprovalForm, setInlineApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [inlineRejectForm, setInlineRejectForm] = useState({ actor: "", reason: "" });
  const [inlineSubmitting, setInlineSubmitting] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());

  // Reset inline review state when run changes
  useEffect(() => {
    setInlineAction("none");
    setInlineApprovalForm({ actor: "", commitMsg: "" });
    setInlineRejectForm({ actor: "", reason: "" });
    setInlineSubmitting(false);
    setSelectedFiles(new Set());
  }, [run.id]);

  // Resolve workspace-sandbox URL via embedded proxy (standard cross-scenario pattern)
  const [sandboxUrl, setSandboxUrl] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    fetch(`/embedded/${encodeURIComponent("workspace-sandbox")}/external-url`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data: { url?: string } | null) => {
        if (!cancelled && data?.url) setSandboxUrl(data.url);
      })
      .catch(() => { /* workspace-sandbox not available */ });
    return () => { cancelled = true; };
  }, []);

  const openSandboxReview = useCallback(() => {
    if (!run.sandboxId || !sandboxUrl) return;
    const url = `${sandboxUrl}?sandbox=${run.sandboxId}&review=true`;
    window.open(url, "_blank", "noopener,noreferrer");
  }, [run.sandboxId, sandboxUrl]);

  const handleInlineConfirm = useCallback(async () => {
    setInlineSubmitting(true);
    try {
      if (inlineAction === "approve") {
        await onApprove({
          actor: inlineApprovalForm.actor.trim() || undefined,
          commitMsg: inlineApprovalForm.commitMsg || undefined,
        });
      } else if (inlineAction === "reject") {
        await onReject({
          actor: inlineRejectForm.actor.trim() || undefined,
          reason: inlineRejectForm.reason || undefined,
        });
      } else if (inlineAction === "partial" && onPartialApprove) {
        // Collect file IDs from selected files
        const files = diff?.files ?? [];
        const fileIds = files
          .filter((f) => selectedFiles.has(f.path) && f.id)
          .map((f) => f.id);
        if (fileIds.length > 0) {
          await onPartialApprove(
            fileIds,
            inlineApprovalForm.actor.trim() || undefined,
            inlineApprovalForm.commitMsg || undefined,
          );
          setSelectedFiles(new Set());
        }
      }
      setInlineAction("none");
    } catch (err) {
      console.error(`Failed to ${inlineAction} run:`, err);
    } finally {
      setInlineSubmitting(false);
    }
  }, [inlineAction, inlineApprovalForm, inlineRejectForm, onApprove, onReject, onPartialApprove, diff, selectedFiles]);

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

  // Close info dialog on Escape, preventing it from bubbling to DetailModal
  useEffect(() => {
    if (!infoOpen) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopImmediatePropagation();
        setInfoOpen(false);
      }
    };
    document.addEventListener("keydown", handleEscape, true); // capture phase
    return () => document.removeEventListener("keydown", handleEscape, true);
  }, [infoOpen]);

  // Build mobile header left (status dot) and push to parent
  const [statusLegendOpen, setStatusLegendOpen] = useState(false);
  useEffect(() => {
    if (isDesktop || !onMobileHeaderLeft) return;

    const statusLabel = runStatusLabel(run.status, run.approvalState);
    const dotColor = STATUS_DOT_COLORS[statusLabel] ?? "bg-muted-foreground";

    onMobileHeaderLeft(
      <StatusDotWithLegend
        dotColor={dotColor}
        statusLabel={statusLabel}
        legendOpen={statusLegendOpen}
        setLegendOpen={setStatusLegendOpen}
      />
    );

    return () => onMobileHeaderLeft(null);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDesktop, run.status, statusLegendOpen, onMobileHeaderLeft]);

  // Build mobile header right (info, stop, actions) and push to parent
  useEffect(() => {
    if (isDesktop || !onMobileHeaderRight) return;

    onMobileHeaderRight(
      <MobileHeaderActions
        actions={actions}
        onInfoOpen={() => setInfoOpen(true)}
        onStop={() => onStop(run)}
        actionsMenuOpen={actionsMenuOpen}
        setActionsMenuOpen={setActionsMenuOpen}
        actionsMenuRef={actionsMenuRef}
        onInvestigate={() => onInvestigate(run.id)}
        canApplyFixes={canApplyFixes}
        onApplyInvestigation={() => onApplyInvestigation(run.id)}
        onRetry={() => onRetry(run)}
        onResumeFromFailure={() => onResumeFromFailure(run)}
        onReview={() => setShowReviewModal(true)}
        canDeleteRun={canDeleteRun}
        onDelete={() => onDelete(run)}
        deleteLoading={deleteLoading}
      />
    );

    return () => onMobileHeaderRight(null);
  }, [isDesktop, run, actions, actionsMenuOpen, onMobileHeaderRight, onStop, onDelete, onInvestigate, onApplyInvestigation, onRetry, onResumeFromFailure, canApplyFixes, canDeleteRun, deleteLoading]);
  return (
    <div className="h-full flex flex-col" ref={containerRef}>
      {/* Details Section (collapsible) - hidden on mobile, shown via info dialog instead */}
      <div
        className="flex-shrink-0 flex-col border-b border-border hidden lg:flex"
        style={isDetailsCollapsed ? undefined : { height: detailsHeight }}
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
                runStatusLabel(run.status, run.approvalState) as
                  | "pending"
                  | "starting"
                  | "running"
                  | "needs_review"
                  | "complete"
                  | "failed"
                  | "cancelled"
              }
            >
              {runStatusLabel(run.status, run.approvalState).replace("_", " ")}
            </Badge>
            <RunModelBadge
              requested={run.requestedModel ?? ""}
              actual={run.actualModel ?? ""}
              fallbackChain={(run.resolvedConfig?.policySnapshot?.candidates ?? [])
                .slice(1)
                .map((candidate) => candidate.model || `${runnerTypeLabel(candidate.runnerType)} default`)}
            />
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
                    title="Re-run from scratch (fresh attempt, no prior context)"
                  >
                    <RotateCcw className="h-3 w-3" />
                    <span className="hidden lg:inline">Re-run</span>
                  </Button>
                )}
                {actions?.canResumeFromFailure && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onResumeFromFailure(run)}
                    className="gap-1 h-7 px-2"
                    title="Resume: continue this task with the prior transcript + diff as context"
                  >
                    <PlayCircle className="h-3 w-3" />
                    <span className="hidden lg:inline">Resume</span>
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
                    onClick={() => { onDelete(run); }}
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
                        title="Fresh attempt, no prior context"
                        onClick={() => { onRetry(run); setActionsMenuOpen(false); }}
                      >
                        <RotateCcw className="h-3.5 w-3.5" /> Re-run
                      </button>
                    )}
                    {actions?.canResumeFromFailure && (
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                        title="Continue this task with the prior transcript + diff as context"
                        onClick={() => { onResumeFromFailure(run); setActionsMenuOpen(false); }}
                      >
                        <PlayCircle className="h-3.5 w-3.5" /> Resume
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
            <RunDetailsContent
              run={run}
              taskTitle={taskTitle}
              profileName={profileName}
              durationMs={durationMs}
              costTotals={costTotals}
            />
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
                activeTab === "timeline"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )}
              onClick={() => setActiveTab("timeline")}
            >
              <span className="hidden sm:inline mr-2"><Activity className="h-4 w-4 inline" /></span>
              Timeline ({events.length})
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
            <button
              className={cn("px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap", activeTab === "report" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground")}
              onClick={() => setActiveTab("report")}
            >
              <span className="hidden sm:inline mr-2"><Info className="h-4 w-4 inline" /></span>
              Report
            </button>
          </div>

          {/* Tab content - fills remaining space, scrollable for most tabs but not Messages */}
          <div
            className={cn(
              "flex-1 min-h-0",
              activeTab === "timeline" ? "flex flex-col" : "overflow-y-auto p-3 sm:p-4"
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
          ) : activeTab === "timeline" ? (
            <div data-testid="run-detail-timeline-layout" className="flex h-full min-h-0 flex-col gap-3">
              <div className="shrink-0">
                <FallbackTimeline runId={run.id} />
              </div>
              <div className="min-h-0 flex-1">
                <RunTimeline
                  run={run}
                  events={events}
                  eventsLoading={eventsLoading}
                  onContinue={onContinue}
                  onDeleteMessage={onDeleteMessage}
                />
              </div>
            </div>
          ) : activeTab === "report" ? (
            report.loading ? <div className="py-8 text-center text-muted-foreground">Loading run report…</div> : report.error ? <div className="py-8 text-center text-destructive">Report unavailable: {report.error}</div> : report.data ? (
              <div className="space-y-3 text-sm" data-testid="run-report">
                <div><strong>Status:</strong> {report.data.status}{report.data.exit_code !== undefined ? ` (exit ${report.data.exit_code})` : ""}{report.data.error ? ` — ${report.data.error}` : ""}</div>
                <div><strong>Timing:</strong> duration={report.data.duration_ms ?? "unavailable"}ms; heartbeat gap={report.data.heartbeat_gap_ms ?? "unavailable"}ms; turns={report.data.turns}; tokens={report.data.tokens}; cost=${report.data.cost_usd}</div>
                <div><strong>Final output:</strong> {report.data.result.selection_status} ({report.data.result.selection_rule || "unavailable"}), candidates={report.data.result.candidate_count}</div>
                <div><strong>Structured:</strong> {report.data.result.structured_status || "unavailable"} {report.data.result.diagnostic_codes?.join(", ")}</div>
                <div><strong>Tools:</strong> project-owned={report.data.project_owned_tool_calls} external={report.data.external_tool_calls}; repeated={report.data.repeated_tool_calls}; files reread={report.data.files_read_more_than_once}</div>
                <div><strong>Model:</strong> requested={report.data.requested_model || "unavailable"} actual={report.data.actual_model || "unavailable"} fallbacks={report.data.fallback_count}</div>
                <div><strong>Diff:</strong> {report.data.diff.files} files, {report.data.diff.bytes} bytes ({report.data.diff.available.state})</div>
                <div><strong>Events:</strong> {report.data.events_availability.state}</div>
                <ul className="list-disc pl-5">{Object.entries(report.data.event_counts).sort(([left], [right]) => left.localeCompare(right)).map(([type, count]) => <li key={type}>{type}: {count}</li>)}</ul>
                <div><strong>Receipts:</strong> {report.data.receipts_availability.state} ({report.data.receipt_count})</div>
                <ul className="list-disc pl-5">{report.data.tools.map((tool) => <li key={tool.name}>{tool.name}: {tool.calls} calls, {tool.failures} failed, {tool.unresolved ?? 0} unresolved</li>)}</ul>
              </div>
            ) : <div className="py-8 text-center text-muted-foreground">Report unavailable</div>
          ) : activeTab === "diff" ? (
            diffLoading ? (
              <div className="py-8 text-center text-muted-foreground">
                <div className="h-8 w-8 mx-auto animate-spin rounded-full border-4 border-primary border-t-transparent" />
                <p className="mt-3 text-sm">Loading diff...</p>
              </div>
            ) : (
              <div className="space-y-4">
                {!diff ? (
                  <div className="py-8 flex flex-col items-center gap-4 text-center">
                    <FileCode className="h-10 w-10 text-muted-foreground/50" />
                    <div>
                      <p className="font-medium text-muted-foreground">No diff available</p>
                      <p className="text-sm text-muted-foreground/70 mt-1">
                        {run.sandboxId
                          ? "The diff hasn't been generated yet. You can review changes directly in the sandbox."
                          : "This run didn't use a sandbox, so no diff was collected."}
                      </p>
                    </div>
                    <div className="flex gap-2">
                      {run.sandboxId && sandboxUrl && (
                        <Button variant="outline" size="sm" className="gap-2" onClick={openSandboxReview}>
                          <ExternalLink className="h-3.5 w-3.5" />
                          Open in Workspace Sandbox
                        </Button>
                      )}
                    </div>
                  </div>
                ) : (
                  <DiffViewer
                    diff={diff}
                    selectable={inlineAction === "partial"}
                    selectedFiles={selectedFiles}
                    onFileSelectionChange={(path, selected) => {
                      setSelectedFiles((prev) => {
                        const next = new Set(prev);
                        if (selected) next.add(path);
                        else next.delete(path);
                        return next;
                      });
                    }}
                  />
                )}

                {/* Inline review controls */}
                {(actions?.canApprove || actions?.canReject) && (
                  <div className="border-t border-border pt-4 space-y-3">
                    <div className="flex items-center gap-2 flex-wrap">
                      {run.sandboxId && sandboxUrl && (
                        <Button variant="outline" size="sm" className="gap-1.5" onClick={openSandboxReview}>
                          <ExternalLink className="h-3.5 w-3.5" />
                          Open in Sandbox
                        </Button>
                      )}
                      <div className="flex-1" />
                      {actions?.canApprove && onPartialApprove && diff && (diff.files?.length ?? 0) > 1 && (
                        <Button
                          variant={inlineAction === "partial" ? "secondary" : "outline"}
                          size="sm"
                          className="gap-1.5"
                          onClick={() => {
                            if (inlineAction === "partial") {
                              setInlineAction("none");
                              setSelectedFiles(new Set());
                            } else {
                              setInlineAction("partial");
                            }
                          }}
                        >
                          <Check className="h-3.5 w-3.5" />
                          Partial Approve
                        </Button>
                      )}
                      {actions?.canApprove && (
                        <Button
                          variant={inlineAction === "approve" ? "success" : "outline"}
                          size="sm"
                          className="gap-1.5"
                          onClick={() => setInlineAction(inlineAction === "approve" ? "none" : "approve")}
                        >
                          <Check className="h-3.5 w-3.5" />
                          Approve All
                        </Button>
                      )}
                      {actions?.canReject && (
                        <Button
                          variant={inlineAction === "reject" ? "destructive" : "outline"}
                          size="sm"
                          className="gap-1.5"
                          onClick={() => setInlineAction(inlineAction === "reject" ? "none" : "reject")}
                        >
                          <X className="h-3.5 w-3.5" />
                          Reject
                        </Button>
                      )}
                    </div>

                    {inlineAction === "approve" && (
                      <div className="space-y-3 rounded-lg border border-success/30 bg-success/5 p-3">
                        <div className="space-y-2">
                          <Label htmlFor="inline-approve-actor">Your Name (optional)</Label>
                          <Input
                            id="inline-approve-actor"
                            value={inlineApprovalForm.actor}
                            onChange={(e) => setInlineApprovalForm({ ...inlineApprovalForm, actor: e.target.value })}
                            placeholder="Leave blank to approve anonymously"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="inline-approve-msg">Commit Message (optional)</Label>
                          <Input
                            id="inline-approve-msg"
                            value={inlineApprovalForm.commitMsg}
                            onChange={(e) => setInlineApprovalForm({ ...inlineApprovalForm, commitMsg: e.target.value })}
                            placeholder="Custom commit message"
                          />
                        </div>
                        <Button
                          variant="success"
                          size="sm"
                          onClick={handleInlineConfirm}
                          disabled={inlineSubmitting}
                          className="gap-2"
                        >
                          <Check className="h-4 w-4" />
                          {inlineSubmitting ? "Approving..." : "Confirm Approval"}
                        </Button>
                      </div>
                    )}

                    {inlineAction === "reject" && (
                      <div className="space-y-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
                        <div className="space-y-2">
                          <Label htmlFor="inline-reject-actor">Your Name (optional)</Label>
                          <Input
                            id="inline-reject-actor"
                            value={inlineRejectForm.actor}
                            onChange={(e) => setInlineRejectForm({ ...inlineRejectForm, actor: e.target.value })}
                            placeholder="Leave blank for anonymous"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="inline-reject-reason">Rejection Reason</Label>
                          <Textarea
                            id="inline-reject-reason"
                            value={inlineRejectForm.reason}
                            onChange={(e) => setInlineRejectForm({ ...inlineRejectForm, reason: e.target.value })}
                            placeholder="Why are you rejecting these changes?"
                            rows={3}
                          />
                        </div>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={handleInlineConfirm}
                          disabled={inlineSubmitting}
                          className="gap-2"
                        >
                          <X className="h-4 w-4" />
                          {inlineSubmitting ? "Rejecting..." : "Confirm Rejection"}
                        </Button>
                      </div>
                    )}

                    {inlineAction === "partial" && (
                      <div className="space-y-3 rounded-lg border border-warning/30 bg-warning/5 p-3">
                        <p className="text-sm text-muted-foreground">
                          Select files above, then approve the selected subset.
                        </p>
                        <div className="space-y-2">
                          <Label htmlFor="partial-approve-actor">Your Name (optional)</Label>
                          <Input
                            id="partial-approve-actor"
                            value={inlineApprovalForm.actor}
                            onChange={(e) => setInlineApprovalForm({ ...inlineApprovalForm, actor: e.target.value })}
                            placeholder="Leave blank to approve anonymously"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="partial-approve-msg">Commit Message (optional)</Label>
                          <Input
                            id="partial-approve-msg"
                            value={inlineApprovalForm.commitMsg}
                            onChange={(e) => setInlineApprovalForm({ ...inlineApprovalForm, commitMsg: e.target.value })}
                            placeholder="Custom commit message"
                          />
                        </div>
                        <Button
                          variant="success"
                          size="sm"
                          onClick={handleInlineConfirm}
                          disabled={inlineSubmitting || selectedFiles.size === 0}
                          className="gap-2"
                        >
                          <Check className="h-4 w-4" />
                          {inlineSubmitting
                            ? "Approving..."
                            : `Approve Selected (${selectedFiles.size} file${selectedFiles.size !== 1 ? "s" : ""})`}
                        </Button>
                      </div>
                    )}
                  </div>
                )}
              </div>
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

      {/* Mobile info dialog — shows run details behind info icon */}
      {infoOpen && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center lg:hidden">
          <div className="fixed inset-0 bg-black/40" onClick={() => setInfoOpen(false)} aria-hidden="true" />
          <div className="relative z-10 w-full max-w-md mx-4 max-h-[80vh] overflow-y-auto rounded-lg border border-border bg-card shadow-xl">
            <div className="flex items-center justify-between border-b border-border p-4">
              <h3 className="font-semibold">Run Details</h3>
              <button
                onClick={() => setInfoOpen(false)}
                className="rounded-sm p-1 opacity-70 hover:opacity-100 transition-opacity"
                aria-label="Close"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="p-4 space-y-4">
              {/* Run ID with copy */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">Run ID:</span>
                <button
                  type="button"
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                  onClick={() => {
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
              <RunDetailsContent
                run={run}
                taskTitle={taskTitle}
                profileName={profileName}
                durationMs={durationMs}
                costTotals={costTotals}
              />
            </div>
          </div>
        </div>
      )}

      <ReviewModal
        open={showReviewModal}
        onOpenChange={setShowReviewModal}
        run={run}
        diff={diff}
        diffLoading={diffLoading}
        onApprove={onApprove}
        onReject={onReject}
        onOpenSandbox={run.sandboxId && sandboxUrl ? openSandboxReview : undefined}
      />
    </div>
  );
}

// RunModelBadge renders the model the executor actually ran with. When the actual
// model differs from the originally requested one, a warning variant is shown to
// flag that the run degraded through the preset fallback chain. Both fields are
// optional — older runs persisted before provenance tracking appear unlabelled.
