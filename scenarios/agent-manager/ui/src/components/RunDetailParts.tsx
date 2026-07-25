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
import { ApprovalState, ExecutionMode, RunMode, RunPhase, RunStatus, TaskStatus } from "../types";

import { MarkdownRenderer } from "./markdown";
import { ModelCostComparison } from "./ModelCostComparison";
import { RunTimeline } from "./RunTimeline";
import { FallbackTimeline } from "./runs/FallbackTimeline";
import { ContextAttachmentModal } from "./ContextAttachmentModal";
import { DiffViewer } from "./DiffViewer";
import { ReviewModal } from "./ReviewModal";



export function RunModelBadge({
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
export function runStatusLabel(status: RunStatus, approvalState?: ApprovalState): string {
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

export function runModeLabel(mode: RunMode): string {
  switch (mode) {
    case RunMode.SANDBOXED:
      return "sandboxed";
    case RunMode.IN_PLACE:
      return "in_place";
    default:
      return "unspecified";
  }
}

// executionModeLabel renders the CLI-driving substrate. UNSPECIFIED normalizes
// to codec_pipe (the default), matching the server's Normalized() contract.
export function executionModeLabel(mode: ExecutionMode): string {
  switch (mode) {
    case ExecutionMode.INTERACTIVE:
      return "interactive";
    case ExecutionMode.CODEC_PIPE:
    default:
      return "codec_pipe";
  }
}

// isInteractiveRun reports whether a run uses the interactive web-console
// substrate (drives the live-session link and the run-list badge).
export function isInteractiveRun(mode: ExecutionMode): boolean {
  return mode === ExecutionMode.INTERACTIVE;
}

export function runPhaseLabel(phase: RunPhase): string {
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

export function taskStatusLabel(status: TaskStatus): string {
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

export function TaskSummary({ task }: { task: Task }) {
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

export type CostTotals = {
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

export function getCostTotals(events: RunEvent[]): CostTotals {
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

export function formatCurrency(value: number): string {
  return formatUsdFixed(value, 4, { useGrouping: false });
}

export const STATUS_DOT_COLORS: Record<string, string> = {
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

export function RunDetailsContent({ run, taskTitle, profileName, durationMs, costTotals }: RunDetailsContentProps) {
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
            <span className="text-muted-foreground">Execution: </span>
            {executionModeLabel(run.executionMode)}
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
          {run.resolvedConfig?.roleRef ? (
            <div>
              <span className="text-muted-foreground">Role: </span>
              {run.resolvedConfig.roleRef}
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
          {run.resolvedConfig?.effort ? (
            <div>
              <span className="text-muted-foreground">Reasoning effort: </span>
              {run.resolvedConfig.effort}
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

      {/* Live web-console session — interactive runs only */}
      {isInteractiveRun(run.executionMode) && (
        <div className="flex flex-wrap items-center gap-3 rounded border border-border bg-muted/40 px-3 py-2 text-sm">
          <span className="font-medium">Live session</span>
          {run.webConsoleSessionUrl ? (
            <a
              href={run.webConsoleSessionUrl}
              target="_blank"
              rel="noreferrer"
              data-testid="open-live-session"
              className="inline-flex items-center gap-1 rounded bg-primary px-3 py-1 text-xs font-medium text-primary-foreground hover:opacity-90"
            >
              Open live terminal
              <ExternalLink className="h-3 w-3" />
            </a>
          ) : run.webConsoleSessionId ? (
            <span className="text-muted-foreground text-xs">
              Session <code className="bg-muted px-1 py-0.5 rounded">{run.webConsoleSessionId}</code> — open web-console and find it in the Programmatic tab.
            </span>
          ) : (
            <span className="text-muted-foreground text-xs">Session starting…</span>
          )}
        </div>
      )}

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

export function StatusDotWithLegend({ dotColor, statusLabel, legendOpen, setLegendOpen }: StatusDotWithLegendProps) {
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

export function MobileHeaderActions({
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

export function CostBreakdown({ totals }: { totals: CostTotals }) {
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

