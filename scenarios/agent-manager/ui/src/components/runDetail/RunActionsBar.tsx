import type React from "react";
import { Eye, MoreVertical, PlayCircle, RotateCcw, Search, Square, Trash2, Wrench } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import type { Run } from "../../types";

interface RunActionsBarProps {
  run: Run;
  isDesktop: boolean;
  deleteLoading: boolean;
  actionsMenuOpen: boolean;
  setActionsMenuOpen: React.Dispatch<React.SetStateAction<boolean>>;
  actionsMenuRef: React.RefObject<HTMLDivElement>;
  onStop: (run: Run) => Promise<void>;
  onRetry: (run: Run) => Promise<Run>;
  onResumeFromFailure: (run: Run) => void;
  onInvestigate: (runId: string) => void;
  onApplyInvestigation: (runId: string) => void;
  onDelete: (run: Run) => void;
  onReview: () => void;
}

/** Stop is always primary; the rest are inline on desktop and behind an ellipsis menu below it. */
export function RunActionsBar({
  run, isDesktop, deleteLoading, actionsMenuOpen, setActionsMenuOpen, actionsMenuRef,
  onStop, onRetry, onResumeFromFailure, onInvestigate, onApplyInvestigation, onDelete, onReview,
}: RunActionsBarProps) {
  const actions = run.actions;
  const canDeleteRun = actions?.canDelete ?? false;
  const canApplyFixes = actions?.canApplyInvestigation ?? false;
  return (
      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
        {actions?.canStop && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onStop(run)}
            className="gap-1 h-11 px-2 sm:h-7 text-destructive hover:text-destructive"
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
                className="gap-1 h-11 px-2 sm:h-7"
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
                className="gap-1 h-11 px-2 sm:h-7"
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
                className="gap-1 h-11 px-2 sm:h-7"
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
                className="gap-1 h-11 px-2 sm:h-7"
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
                onClick={() => onReview()}
                className="gap-1 h-11 px-2 sm:h-7"
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
                className="gap-1 h-11 px-2 sm:h-7 text-destructive hover:text-destructive"
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
              className="h-11 w-11 p-0 sm:h-7 sm:w-7"
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
                    onClick={() => { onReview(); setActionsMenuOpen(false); }}
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
  );
}
