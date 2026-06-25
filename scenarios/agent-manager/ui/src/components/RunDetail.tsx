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
import { ApprovalState, ModelPreset, RunMode, RunPhase, RunStatus, TaskStatus } from "../types";

const modelPresetLabel = (preset?: ModelPreset) => {
  switch (preset) {
    case ModelPreset.FAST:
      return "Fast";
    case ModelPreset.CHEAP:
      return "Cheap";
    case ModelPreset.SMART:
      return "Smart";
    default:
      return "";
  }
};

import { MarkdownRenderer } from "./markdown";
import { ModelCostComparison } from "./ModelCostComparison";
import { RunTimeline } from "./RunTimeline";
import { FallbackTimeline } from "./runs/FallbackTimeline";
import { ContextAttachmentModal } from "./ContextAttachmentModal";
import { DiffViewer } from "./DiffViewer";
import { ReviewModal } from "./ReviewModal";

type TabId = "task" | "timeline" | "diff" | "cost";

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
              fallbackChain={(run.resolvedConfig?.fallbackRunnerTypes ?? []).map(String)}
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
function RunModelBadge({
  requested,
  actual,
  fallbackChain,
}: {
  requested: string;
  actual: string;
  fallbackChain?: string[];
}) {
  if (!requested && !actual) {
    return null;
  }
  const display = actual || requested;
  const degraded = Boolean(requested && actual && requested !== actual);
  const label = display === "" ? "runner default" : display;
  if (degraded) {
    const chainLine =
      fallbackChain && fallbackChain.length > 0
        ? ` Fallback chain: ${fallbackChain.join(" → ")}.`
        : "";
    return (
      <Badge
        variant="destructive"
        title={`Ran on fallback model "${actual || "runner default"}" after the requested "${requested}" failed.${chainLine} See the run's Fallback Timeline for the per-attempt reason.`}
      >
        model: {label} (fallback)
      </Badge>
    );
  }
  return (
    <Badge variant="secondary" title={`Model executed: ${label}`}>
      model: {label}
    </Badge>
  );
}

// Helper functions and sub-components
function runStatusLabel(status: RunStatus, approvalState?: ApprovalState): string {
  if (approvalState === ApprovalState.REJECTED) return "rejected";
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
    case RunStatus.PARKED:
      return "parked";
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

const STATUS_DOT_COLORS: Record<string, string> = {
  pending: "bg-muted-foreground",
  cancelled: "bg-muted-foreground",
  starting: "bg-primary",
  running: "bg-primary",
  complete: "bg-success",
  needs_review: "bg-warning",
  failed: "bg-destructive",
  rejected: "bg-orange-500",
};

interface RunDetailsContentProps {
  run: Run;
  taskTitle: string;
  profileName: string;
  durationMs: number | null;
  costTotals: CostTotals;
}

function RunDetailsContent({ run, taskTitle, profileName, durationMs, costTotals }: RunDetailsContentProps) {
  return (
    <>
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

        {run.status === RunStatus.PARKED && run.awaitHandle && (
          <div className="mt-3 rounded-md border border-amber-500/40 bg-amber-500/5 p-3 text-sm text-amber-700 dark:text-amber-400">
            <div className="flex items-start gap-2">
              <PauseCircle className="mt-0.5 h-4 w-4 flex-shrink-0" />
              <div className="space-y-1">
                <p className="font-medium">Parked — waiting, not hung</p>
                <p className="break-words text-xs">
                  Suspended (zero tokens) while agent-manager waits on{" "}
                  <code className="font-mono">
                    {run.awaitHandle.producer}
                    {run.awaitHandle.key ? `:${run.awaitHandle.key}` : ""}
                  </code>
                  . It resumes automatically with the result.
                  {run.awaitHandle.deadline
                    ? ` Resumes by ${new Date(timestampMs(run.awaitHandle.deadline)).toLocaleString()} at the latest.`
                    : ""}
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
          {run.resolvedConfig?.modelPreset !== undefined &&
          run.resolvedConfig.modelPreset !== ModelPreset.UNSPECIFIED ? (
            <div>
              <span className="text-muted-foreground">Model preset: </span>
              {modelPresetLabel(run.resolvedConfig.modelPreset)}
            </div>
          ) : null}
          {run.actualModel || run.resolvedConfig?.model ? (
            <div>
              <span className="text-muted-foreground">Model: </span>
              <code className="text-xs bg-muted px-1 py-0.5 rounded">
                {run.actualModel || run.resolvedConfig?.model}
              </code>
            </div>
          ) : null}
          {run.resolvedConfig?.maxTurns ? (
            <div>
              <span className="text-muted-foreground">Max turns: </span>
              {run.resolvedConfig.maxTurns}
            </div>
          ) : null}
          {run.resolvedConfig?.timeout ? (
            <div>
              <span className="text-muted-foreground">Timeout: </span>
              {formatDuration(Number(protoDurationMs(run.resolvedConfig.timeout)))}
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
    </>
  );
}

const STATUS_LEGEND: { label: string; color: string }[] = [
  { label: "Pending", color: "bg-muted-foreground" },
  { label: "Starting", color: "bg-primary" },
  { label: "Running", color: "bg-primary" },
  { label: "Needs review", color: "bg-warning" },
  { label: "Complete", color: "bg-success" },
  { label: "Rejected", color: "bg-orange-500" },
  { label: "Failed", color: "bg-destructive" },
  { label: "Cancelled", color: "bg-muted-foreground" },
];

interface StatusDotWithLegendProps {
  dotColor: string;
  statusLabel: string;
  legendOpen: boolean;
  setLegendOpen: (open: boolean) => void;
}

function StatusDotWithLegend({ dotColor, statusLabel, legendOpen, setLegendOpen }: StatusDotWithLegendProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!legendOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setLegendOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [legendOpen, setLegendOpen]);

  return (
    <div className="relative shrink-0" ref={ref}>
      <button
        type="button"
        className="flex items-center justify-center w-6 h-6 rounded-full hover:bg-muted/50 transition-colors"
        onClick={() => setLegendOpen(!legendOpen)}
        title={statusLabel.replace("_", " ")}
      >
        <span className={cn("w-2.5 h-2.5 rounded-full", dotColor)} />
      </button>
      {legendOpen && (
        <div className="absolute left-0 top-full z-50 mt-1 min-w-[140px] rounded-md border border-border bg-card p-2 shadow-lg">
          <p className="text-[10px] font-medium text-muted-foreground mb-1.5">Status</p>
          {STATUS_LEGEND.map(({ label, color }) => (
            <div
              key={label}
              className={cn(
                "flex items-center gap-2 px-1 py-0.5 text-xs rounded",
                statusLabel.replace("_", " ") === label.toLowerCase() && "bg-muted/50 font-medium"
              )}
            >
              <span className={cn("w-2 h-2 rounded-full shrink-0", color)} />
              <span>{label}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

interface MobileHeaderActionsProps {
  actions: Run["actions"];
  onInfoOpen: () => void;
  onStop: () => void;
  actionsMenuOpen: boolean;
  setActionsMenuOpen: (fn: (prev: boolean) => boolean) => void;
  actionsMenuRef: React.RefObject<HTMLDivElement>;
  onInvestigate: () => void;
  canApplyFixes: boolean;
  onApplyInvestigation: () => void;
  onRetry: () => void;
  onResumeFromFailure: () => void;
  onReview: () => void;
  canDeleteRun: boolean;
  onDelete: () => void;
  deleteLoading: boolean;
}

function MobileHeaderActions({
  actions,
  onInfoOpen,
  onStop,
  actionsMenuOpen,
  setActionsMenuOpen,
  actionsMenuRef,
  onInvestigate,
  canApplyFixes,
  onApplyInvestigation,
  onRetry,
  onResumeFromFailure,
  onReview,
  canDeleteRun,
  onDelete,
  deleteLoading,
}: MobileHeaderActionsProps) {
  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        className="h-7 w-7 p-0"
        onClick={onInfoOpen}
        title="Run details"
      >
        <Info className="h-4 w-4" />
      </Button>
      {actions?.canStop && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onStop}
          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
          title="Stop run"
        >
          <Square className="h-3.5 w-3.5" />
        </Button>
      )}
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
                onClick={() => { onInvestigate(); setActionsMenuOpen(() => false); }}
              >
                <Search className="h-3.5 w-3.5" /> Investigate
              </button>
            )}
            {canApplyFixes && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                onClick={() => { onApplyInvestigation(); setActionsMenuOpen(() => false); }}
              >
                <Wrench className="h-3.5 w-3.5" /> Apply Fixes
              </button>
            )}
            {actions?.canRetry && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                title="Fresh attempt, no prior context"
                onClick={() => { onRetry(); setActionsMenuOpen(() => false); }}
              >
                <RotateCcw className="h-3.5 w-3.5" /> Re-run
              </button>
            )}
            {actions?.canResumeFromFailure && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                title="Continue this task with the prior transcript + diff as context"
                onClick={() => { onResumeFromFailure(); setActionsMenuOpen(() => false); }}
              >
                <PlayCircle className="h-3.5 w-3.5" /> Resume
              </button>
            )}
            {(actions?.canReview || actions?.canApprove || actions?.canReject) && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-foreground hover:bg-muted/50 transition-colors"
                onClick={() => { onReview(); setActionsMenuOpen(() => false); }}
              >
                <Eye className="h-3.5 w-3.5" /> Review
              </button>
            )}
            {canDeleteRun && (
              <button
                type="button"
                className="flex w-full items-center gap-2 rounded px-3 py-2 text-sm text-destructive hover:bg-muted/50 transition-colors"
                disabled={deleteLoading}
                onClick={() => { onDelete(); setActionsMenuOpen(() => false); }}
              >
                <Trash2 className="h-3.5 w-3.5" /> Delete
              </button>
            )}
          </div>
        )}
      </div>
    </>
  );
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
