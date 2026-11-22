import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Sparkles, X } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { useIsMobile } from "../hooks";
import type { StagedSuggestionHold } from "../features/commit-message/messageSession";
import type { QueuedStageSelection } from "../features/commit-message/stageQueue";

interface StageDuringSuggestionModalProps {
  hold: StagedSuggestionHold | null;
  paths: string[];
  queuedConflicts?: Pick<QueuedStageSelection, "id" | "paths">[];
  onConfirm: () => void;
  isQueueRecovery?: boolean;
  onQueue?: () => void;
  queuedSelections?: number;
  onCancel: () => void;
}

/** Staging changes the exact selection a commit suggestion describes. */
export function StageDuringSuggestionModal({ hold, paths, queuedConflicts = [], onConfirm, isQueueRecovery = false, onQueue, queuedSelections = 0, onCancel }: StageDuringSuggestionModalProps) {
  const isMobile = useIsMobile();
  const uniquePaths = [...new Set(paths)];
  if ((!hold && queuedConflicts.length === 0) || uniquePaths.length === 0) return null;

  const drafting = hold?.state === "drafting";
  const noun = uniquePaths.length === 1 ? "file" : "files";
  const requestedPaths = new Set(uniquePaths);
  const queuedPathOverlap = queuedConflicts.some(selection => selection.paths.some(path => requestedPaths.has(path)));
  const described = hold?.paths.length ? `${hold.paths.length} staged ${hold.paths.length === 1 ? "file" : "files"}` : "the current staged selection";
  const title = isQueueRecovery ? "Review current file versions" : hold ? drafting ? "A commit suggestion is being drafted" : "A commit suggestion is ready" : "Queued suggestions use these files";
  const stageGuidance = isQueueRecovery
    ? `${hold ? `The current suggestion describes ${described}. ` : ""}Applying these worktree edits updates the staged file contents${hold ? " and may make that suggestion outdated" : ""}. GCT clears the saved queue entry after the full selection is staged.`
    : hold
      ? onQueue
        ? "Staging these files changes the suggestion’s file set. Queue them to keep the current draft unchanged; you can draft a message for them later."
        : "Staging these files changes the file set described by the current suggestion. Review the paths before continuing."
      : "Staging these files now would change or consume an existing queued selection.";
  const content = (
    <>
      <div className="flex items-center gap-3 text-amber-300 mb-4">
        <Sparkles className="h-5 w-5 flex-shrink-0" />
        <span className="font-semibold">Stage {uniquePaths.length} {noun}?</span>
      </div>
      <p className="text-sm text-slate-400 mb-4">
        {isQueueRecovery ? stageGuidance : hold ? <>{drafting ? "The drafting agent is" : "The suggestion is"} describing {described}. {stageGuidance}</> : stageGuidance} {queuedSelections ? `${queuedSelections} selection${queuedSelections === 1 ? " is" : "s are"} in the follow-up queue.` : ""}
      </p>
      {queuedConflicts.length > 0 && <div className="mb-4 rounded-lg border border-amber-800/70 bg-amber-950/30 p-3 text-xs text-amber-100">
        <p className="font-semibold">Staging now removes {queuedConflicts.length} overlapping queued {queuedConflicts.length === 1 ? "selection" : "selections"}.</p>
        <ul className="mt-2 space-y-1">{queuedConflicts.map((selection, index) => <li key={selection.id} className="break-all">{index + 1}. Existing selection: {selection.paths.join(", ")}</li>)}</ul>
      </div>}
      <details className="text-xs text-slate-300">
        <summary className="min-h-9 cursor-pointer py-2">Review {uniquePaths.length} file paths</summary>
        <ul className="max-h-40 overflow-y-auto space-y-1 rounded-lg border border-slate-700/70 bg-slate-950/40 p-3" aria-label="Files to stage">
          {uniquePaths.map(file => <li key={file} className="break-all font-mono">{file}</li>)}
        </ul>
      </details>
      {onQueue && queuedPathOverlap && <p role="status" className="text-xs text-amber-200">Some files are already in the queued selections above. Remove or resolve that overlap before queueing these files.</p>}
    </>
  );

  if (isMobile) {
    return (
      <FullPageDrawer
        open
        onClose={onCancel}
        title={title}
        ariaLabel={title}
        closeLabel="Close"
        dismissAffordance="close"
        contentClassName="px-4 py-6 space-y-6"
        footer={
          <div className="flex w-full flex-col gap-2 sm:flex-row">
            <Button
              variant="secondary"
              size="sm"
              onClick={onCancel}
              className="h-12 w-full sm:flex-1"
              data-testid="stage-during-suggestion-cancel"
              style={{ background: "transparent" }}
            >
              Cancel
            </Button>
            {onQueue && <Button
              variant="secondary"
              size="sm"
              onClick={onQueue}
              disabled={queuedPathOverlap}
              className="h-12 w-full sm:flex-1"
              data-testid="stage-during-suggestion-queue"
              style={{ background: "transparent" }}
            >Queue files</Button>}
            <Button size="sm" onClick={onConfirm} className="h-12 w-full sm:flex-1" data-testid="stage-during-suggestion-confirm">
              {queuedConflicts.length ? `Stage + remove ${queuedConflicts.length}` : isQueueRecovery ? `Stage ${uniquePaths.length} current versions` : `Stage ${uniquePaths.length}`}
            </Button>
          </div>
        }
      >
        {content}
      </FullPageDrawer>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4" role="dialog" aria-modal="true" aria-label={title} onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">{title}</h2>
          <button type="button" className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300" onClick={onCancel} aria-label="Close"><X className="h-4 w-4" /></button>
        </div>
        <div className="px-4 py-4">{content}</div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button
            variant="secondary"
            size="sm"
            onClick={onCancel}
            data-testid="stage-during-suggestion-cancel"
            style={{ background: "transparent" }}
          >Cancel</Button>
          {onQueue && <Button
            variant="secondary"
            size="sm"
            onClick={onQueue}
            data-testid="stage-during-suggestion-queue"
            style={{ background: "transparent" }}
          >Queue files</Button>}
          <Button size="sm" onClick={onConfirm} data-testid="stage-during-suggestion-confirm">{queuedConflicts.length ? `Stage + remove ${queuedConflicts.length}` : isQueueRecovery ? `Stage ${uniquePaths.length} current versions` : `Stage ${uniquePaths.length}`}</Button>
        </div>
      </div>
    </div>
  );
}
