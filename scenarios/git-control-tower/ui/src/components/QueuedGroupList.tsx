import { useState } from "react";
import { ChevronDown, ChevronRight, ListOrdered, LoaderCircle, Play, Trash2 } from "lucide-react";
import { StagedMessageMode } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { QueuedStageSuggestionStatus } from "./QueuedStageSuggestionStatus";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";
import type { StagedMessageSuggestion } from "../features/commit-message/api";

interface Props {
  repositoryId: string;
  selections: QueuedStageSelection[];
  stagedPaths?: readonly string[];
  worktreeChangedPaths?: readonly string[];
  recoveryPendingId?: string | null;
  /** Controlled expansion. Falls back to local state when omitted. */
  expandedId?: string | null;
  onExpandedChange?: (id: string | null) => void;
  /** Briefly highlighted after the blocker's Review action targets this group. */
  revealedId?: string | null;
  /** Queue entries whose agent start is in flight. */
  startingIds?: string[];
  onRemove?: (id: string) => void;
  onUse: (suggestion: StagedMessageSuggestion) => void;
  onRetry?: (selection: QueuedStageSelection) => void;
  onRefresh?: (selection: QueuedStageSelection) => void;
  onStartSuggestion?: (selection: QueuedStageSelection, mode: StagedMessageMode) => void;
  onStageNow?: (selection: QueuedStageSelection) => void;
  onFinishUnstage?: (selection: QueuedStageSelection) => void;
  onResolveCommit?: (id: string, committed: boolean) => void;
  /** Retry the stage head after a failed stage. */
  onRetryStage?: () => void;
}

const stateLabel: Record<QueuedStageSelection["state"], string> = {
  queued: "Queued",
  awaiting_unstage: "Unstage due",
  staged: "Staged",
  stage_failed: "Stage failed",
  stale: "Stale",
  commit_pending: "Commit pending",
};

function stateClass(state: QueuedStageSelection["state"]): string {
  if (state === "staged") return "border-emerald-700/60 bg-emerald-950/30 text-emerald-200";
  if (state === "queued") return "border-cyan-800/60 bg-cyan-950/30 text-cyan-200";
  if (state === "commit_pending") return "border-sky-800/60 bg-sky-950/30 text-sky-200";
  return "border-amber-800/60 bg-amber-950/30 text-amber-200";
}

function canStart(selection: QueuedStageSelection): boolean {
  return selection.state === "queued" && !selection.jobId && !selection.suggestionMode;
}

/**
 * Compact queue. One row per stage group; the verbose job card only mounts when
 * the operator asks for details, so ten groups do not bury Commit and Push.
 * The row itself is the expansion control: an inert row invited repeated
 * clicks because it looked like a control.
 */
export function QueuedGroupList({
  repositoryId,
  selections,
  stagedPaths,
  worktreeChangedPaths,
  recoveryPendingId,
  expandedId,
  onExpandedChange,
  revealedId,
  startingIds = [],
  onRemove,
  onUse,
  onRetry,
  onRefresh,
  onStartSuggestion,
  onStageNow,
  onFinishUnstage,
  onResolveCommit,
  onRetryStage,
}: Props) {
  const [localExpanded, setLocalExpanded] = useState<string | null>(() => selections.find(selection => selection.state === "staged")?.id ?? null);
  const expanded = expandedId === undefined ? localExpanded : expandedId;
  const setExpanded = (id: string | null) => {
    setLocalExpanded(id);
    onExpandedChange?.(id);
  };
  if (selections.length === 0) return null;
  const uniqueFiles = new Set(selections.flatMap(selection => selection.paths)).size;
  const next = selections.find(canStart);
  const startingNext = Boolean(next && startingIds.includes(next.id));

  return <section className="rounded-lg border border-slate-800 bg-slate-900/30" aria-label="Queued commit selections" data-testid="queued-stage-selections">
    <header className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-800 px-3 py-2">
      <div className="flex min-w-0 items-center gap-2">
        <ListOrdered className="h-4 w-4 shrink-0 text-cyan-300" aria-hidden="true" />
        <h3 className="text-sm font-semibold text-slate-100">Follow-up queue</h3>
        <span className="text-xs text-slate-400">{selections.length} group{selections.length === 1 ? "" : "s"} · {uniqueFiles} file{uniqueFiles === 1 ? "" : "s"}</span>
      </div>
      <div className="flex items-center gap-2">
        {next && onStartSuggestion && <button type="button" onClick={() => onStartSuggestion(next, StagedMessageMode.QUICK)} disabled={startingNext} className="inline-flex min-h-8 items-center gap-1.5 rounded-md border border-cyan-700/70 bg-cyan-950/30 px-2.5 text-xs font-medium text-cyan-100 hover:bg-cyan-950/60 disabled:cursor-wait disabled:opacity-60" data-testid="start-next-queued-group">
          {startingNext ? <LoaderCircle className="h-3.5 w-3.5 animate-spin" aria-hidden="true" /> : <Play className="h-3.5 w-3.5" aria-hidden="true" />}
          {startingNext ? "Starting…" : `Start next (${next.paths.length})`}
        </button>}
        <button type="button" onClick={() => setExpanded(null)} className={`text-xs text-slate-400 hover:text-slate-200 ${expanded === null ? "invisible" : ""}`} aria-label="Collapse queue details">Collapse</button>
      </div>
    </header>
    <ul className="divide-y divide-slate-800/80">
      {selections.map((selection, index) => {
        const open = expanded === selection.id;
        const starting = startingIds.includes(selection.id);
        const revealed = revealedId === selection.id;
        return <li key={selection.id} className={`min-w-0 transition-colors ${revealed ? "bg-sky-950/40 ring-1 ring-inset ring-sky-500/70" : ""}`} data-testid="queued-group-row" data-revealed={revealed ? "true" : undefined}>
          <div className="flex min-w-0 items-center gap-1 px-3 py-2">
            <button
              type="button"
              onClick={() => setExpanded(open ? null : selection.id)}
              aria-expanded={open}
              data-testid="queued-group-details"
              title={open ? "Hide queue details" : "Show queue details"}
              className="flex min-h-8 min-w-0 flex-1 items-center gap-2 rounded px-1 text-left hover:bg-slate-800/50"
            >
              <span className="w-4 shrink-0 text-center font-mono text-[11px] text-slate-500">{index + 1}</span>
              <span className={`shrink-0 rounded border px-1.5 py-0.5 text-[11px] font-medium ${stateClass(selection.state)}`}>{stateLabel[selection.state]}</span>
              <span className="shrink-0 text-xs text-slate-300">{selection.paths.length} file{selection.paths.length === 1 ? "" : "s"}</span>
              <span className="min-w-0 truncate text-xs text-slate-500">{starting ? "starting agent…" : selection.suggestionMode ? `${selection.suggestionMode} agent` : "no agent yet"}</span>
              {open ? <ChevronDown className="ml-auto h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" /> : <ChevronRight className="ml-auto h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" />}
            </button>
            <div className="flex shrink-0 items-center gap-1">
              {canStart(selection) && onStartSuggestion && <>
                <button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.QUICK)} disabled={starting} className="min-h-8 rounded-md px-2 text-xs text-cyan-200 hover:bg-slate-800 disabled:cursor-wait disabled:opacity-50" title="Start a quick draft for this group">Quick</button>
                <button type="button" onClick={() => onStartSuggestion(selection, StagedMessageMode.DEEP)} disabled={starting} className="min-h-8 rounded-md px-2 text-xs text-cyan-200 hover:bg-slate-800 disabled:cursor-wait disabled:opacity-50" title="Start a deep investigation for this group">Deep</button>
              </>}
              {selection.state === "staged" && onStageNow && <button type="button" onClick={() => onStageNow(selection)} className="min-h-8 rounded-md px-2 text-xs text-cyan-200 hover:bg-slate-800">Review</button>}
              {index === 0 && selection.state === "stage_failed" && onRetryStage && <button type="button" onClick={onRetryStage} className="min-h-8 rounded-md px-2 text-xs text-amber-200 hover:bg-slate-800">Retry stage</button>}
              {onRemove && selection.state !== "commit_pending" && <button type="button" onClick={() => onRemove(selection.id)} className="min-h-8 rounded-md px-1.5 text-slate-500 hover:bg-slate-800 hover:text-slate-200" aria-label={`Remove group ${index + 1}`} title={selection.state === "stale" ? "Remove this stale group from the queue" : "Remove this group from the queue"}><Trash2 className="h-3.5 w-3.5" aria-hidden="true" /></button>}
            </div>
          </div>
          {selection.stageError && !open && <p role="alert" className="mx-3 mb-2 break-words text-xs text-amber-200">{selection.stageError}</p>}
          {selection.state === "commit_pending" && onResolveCommit && <div className="mx-3 mb-2 flex flex-wrap items-center gap-3 text-xs"><span className="text-amber-100">Did the commit finish?</span><button type="button" className="text-cyan-200 underline" onClick={() => onResolveCommit(selection.id, true)}>I committed these files</button><button type="button" className="text-slate-300 underline" onClick={() => onResolveCommit(selection.id, false)}>The commit did not happen</button></div>}
          {open && <div className="px-3 pb-3"><QueuedStageSuggestionStatus repositoryId={repositoryId} selection={selection} isStaged={selection.state === "staged"} stagedPaths={stagedPaths} worktreeChangedPaths={worktreeChangedPaths} isQueueRecoveryBusy={recoveryPendingId === selection.id} onRemove={onRemove} onUse={onUse} onRetry={onRetry} onRefresh={onRefresh} onStartSuggestion={onStartSuggestion} onStageNow={onStageNow} onFinishUnstage={onFinishUnstage} onCollapse={() => setExpanded(null)} /></div>}
        </li>;
      })}
    </ul>
  </section>;
}
