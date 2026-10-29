import type React from "react";
import { statusTone } from "../lib/display";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { useState, useRef, useEffect, useMemo } from "react";
import { timestampMs } from "@bufbuild/protobuf/wkt";
import { buildTimelineEntries } from "../lib/runTimeline";
import { Check, ChevronDown, ChevronRight, Copy } from "lucide-react";
import { runnerLabel } from "../lib/format";
import { cn } from "../lib/utils";
import { useCollapsiblePanel } from "../hooks/useCollapsiblePanel";
import { useClipboardFeedback } from "../hooks/useClipboardFeedback";
import { useDocumentEscape } from "../hooks/useDocumentEscape";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useViewportSize } from "../hooks/useViewportSize";
import type { Run, RunDiff, RunEvent, Task } from "../types";

import { ReviewModal } from "./ReviewModal";
import { RUN_TABS, type RunTabId } from "./runTabs";
import { DiffTab } from "./runDetail/DiffTab";
import { MobileInfoDialog } from "./runDetail/MobileInfoDialog";
import { ReportTab } from "./runDetail/ReportTab";
import { RunActionsBar } from "./runDetail/RunActionsBar";
import { TimelineTab } from "./runDetail/TimelineTab";
import { UsageTab } from "./runDetail/UsageTab";
import { useSandboxReview } from "./runDetail/useSandboxReview";

import {
  getCostTotals,
  MobileHeaderActions,
  RunDetailsContent,
  RunModelBadge,
  STATUS_DOT_COLORS,
  runStatusLabel,
  StatusDotWithLegend,
  TaskSummary,
} from "./RunDetailParts";

type TabId = RunTabId;

interface RunDetailProps {
  run: Run;
  events: RunEvent[];
  diff: RunDiff | null;
  eventsLoading: boolean;
  diffLoading: boolean;
  initialTab?: TabId;
  focusEventId?: string;
  focusSequence?: string;
  task?: Task | null;
  taskTitle: string;
  profileName: string;
  // Partial-approve (select specific diff files) is the one agent-manager-native
  // review action the Web Console inbox does NOT offer (the inbox is binary
  // approve/reject), so it is preserved here. Plain full-run approve/reject was
  // retired from this UI — a human does those only in the inbox.
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
  focusEventId,
  focusSequence,
  task,
  taskTitle,
  profileName,
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
  const { status: runIdCopyStatus, copyText: copyRunId } = useClipboardFeedback(1500);
  const [infoOpen, setInfoOpen] = useState(false);
  const [actionsMenuOpen, setActionsMenuOpen] = useState(false);
  const actionsMenuRef = useRef<HTMLDivElement>(null);
  const { isDesktop } = useViewportSize();

  const { sandboxUrl, openSandboxReview } = useSandboxReview(run.sandboxId);

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
  // The tab counts what the timeline lists (grouped entries), not the raw events behind them.
  const timelineEntryCount = useMemo(() => buildTimelineEntries(events).length, [events]);
  // A finished run reports its duration; one still running reports the time elapsed so far.
  const durationMs = run.startedAt ? (run.endedAt ? timestampMs(run.endedAt) : Date.now()) - timestampMs(run.startedAt) : null;
  const actions = run.actions;
  const canDeleteRun = actions?.canDelete ?? false;
  const canApplyFixes = actions?.canApplyInvestigation ?? false;

  // Consume Escape before an enclosing detail modal can act on it.
  useDocumentEscape(infoOpen, (event) => {
    event.stopImmediatePropagation();
    setInfoOpen(false);
  }, true);

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
            <StatusBadge tone={statusTone(runStatusLabel(run.status, run.approvalState))}>
              {runStatusLabel(run.status, run.approvalState).replace("_", " ")}
            </StatusBadge>
            <RunModelBadge
              requested={run.requestedModel ?? ""}
              actual={run.actualModel ?? ""}
              fallbackChain={(run.resolvedConfig?.policySnapshot?.candidates ?? [])
                .slice(1)
                .map((candidate) => candidate.model || `${runnerLabel(candidate.runnerType)} default`)}
            />
            <button
              type="button"
              className="flex min-h-11 items-center gap-1 px-1 text-xs text-muted-foreground hover:text-foreground transition-colors sm:min-h-8"
              onClick={(e) => {
                e.stopPropagation();
                void copyRunId(run.id);
              }}
              title={`Copy run ID: ${run.id}`}
            >
              <code className="font-mono">{run.id.slice(0, 8)}</code>
              {runIdCopyStatus === "copied" ? <Check className="h-3 w-3 text-emerald-500" /> : <Copy className="h-3 w-3" />}
            </button>
          </div>
            <RunActionsBar
              run={run}
              isDesktop={isDesktop}
              deleteLoading={deleteLoading}
              actionsMenuOpen={actionsMenuOpen}
              setActionsMenuOpen={setActionsMenuOpen}
              actionsMenuRef={actionsMenuRef}
              onStop={onStop}
              onRetry={onRetry}
              onResumeFromFailure={onResumeFromFailure}
              onInvestigate={onInvestigate}
              onApplyInvestigation={onApplyInvestigation}
              onDelete={onDelete}
              onReview={() => setShowReviewModal(true)}
            />
        </div>

        {/* Content - hidden when collapsed */}
        {!isDetailsCollapsed && (
          <div className="flex-1 min-h-0 overflow-y-auto p-3 space-y-3 sm:p-4 sm:space-y-4">
            <RunDetailsContent
              run={run}
              diff={diffLoading ? null : diff}
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
          <div role="tablist" aria-label="Run sections" className="flex border-b border-border overflow-x-auto overflow-y-hidden shrink-0">
            {RUN_TABS.map(({ id, label, Icon }) => (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={activeTab === id}
                className={cn(
                  "min-h-11 sm:min-h-0 px-3 py-1.5 sm:px-4 sm:py-2 text-sm font-medium border-b-2 -mb-px transition-colors whitespace-nowrap",
                  activeTab === id ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"
                )}
                onClick={() => setActiveTab(id)}
              >
                <span className="hidden sm:inline mr-2"><Icon className="h-4 w-4 inline" /></span>
                {id === "timeline" ? `${label} (${timelineEntryCount})` : label}
              </button>
            ))}
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
              <TimelineTab
                run={run}
                events={events}
                eventsLoading={eventsLoading}
                focusEventId={focusEventId}
                focusSequence={focusSequence}
                onContinue={onContinue}
                onDeleteMessage={onDeleteMessage}
              />
            ) : activeTab === "report" ? (
              <ReportTab runId={run.id} />
            ) : activeTab === "diff" ? (
              <DiffTab
                run={run}
                diff={diff}
                diffLoading={diffLoading}
                sandboxUrl={sandboxUrl}
                openSandboxReview={openSandboxReview}
                onPartialApprove={onPartialApprove}
              />
            ) : (
              <UsageTab run={run} costTotals={costTotals} eventsLoading={eventsLoading} />
            )}
          </div>
        </div>
      </div>

      {infoOpen && (
        <MobileInfoDialog
          run={run}
          diff={diffLoading ? null : diff}
          taskTitle={taskTitle}
          profileName={profileName}
          durationMs={durationMs}
          costTotals={costTotals}
          runIdCopyStatus={runIdCopyStatus}
          onCopyRunId={() => void copyRunId(run.id)}
          onClose={() => setInfoOpen(false)}
        />
      )}

      <ReviewModal
        open={showReviewModal}
        onOpenChange={setShowReviewModal}
        run={run}
        diff={diff}
        diffLoading={diffLoading}
        onOpenSandbox={run.sandboxId && sandboxUrl ? openSandboxReview : undefined}
      />
    </div>
  );
}

// RunModelBadge renders the model the executor actually ran with. When the actual
// model differs from the originally requested one, a warning variant is shown to
// flag that the run degraded through the preset fallback chain. Both fields are
// optional — older runs persisted before provenance tracking appear unlabelled.
