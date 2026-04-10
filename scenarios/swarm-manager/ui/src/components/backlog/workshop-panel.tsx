/**
 * Workshop panel for backlog detail views.
 *
 * Renders all workshop rounds, latest expanded by default, older collapsed.
 * Users can answer questions, decide on proposals, and trigger the next round.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { CheckCircle2, ChevronDown, ChevronRight, MoreHorizontal, Play, Trash2 } from "lucide-react";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { WorkshopItemCard } from "./workshop-item-card";
import { ReadinessDots } from "./readiness-dots";
import { buildWorkshopRoundContent, getPendingDecisionCount } from "../../lib/workshop-files";
import type { WorkshopRound, WorkshopItem, BacklogKind } from "../../types/domain";

/** Small dropdown menu for round-level actions. */
function RoundMenu({ onDelete, disabled }: { onDelete: () => void; disabled?: boolean }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (e: Event) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", handleClickOutside);
    return () => document.removeEventListener("pointerdown", handleClickOutside);
  }, [open]);

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        disabled={disabled}
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
        className="rounded p-1 text-slate-400 hover:text-slate-200 hover:bg-slate-700/50 transition-colors disabled:opacity-50"
        title="Round actions"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-10 mt-1 min-w-[160px] rounded-md border border-slate-700 bg-slate-900 py-1 shadow-lg">
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); setOpen(false); onDelete(); }}
            className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Delete round
          </button>
        </div>
      )}
    </div>
  );
}

interface WorkshopPanelProps {
  rounds: WorkshopRound[];
  backlogKind: BacklogKind;
  backlogName: string;
  disabled?: boolean;
  isSaving?: boolean;
  isRunningWorkshop?: boolean;
  onSaveRound?: (roundNumber: number, content: string) => void;
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  onRunWorkshop?: () => void;
  workshopActionLabel?: string;
  onDeleteRound?: (roundNumber: number) => void;
  isDeletingRound?: boolean;
  /**
   * Whether the workshop has been finalized (plan/conclusion generated from
   * workshop answers). When true, a "Finalized" badge is shown and the
   * "Next Round" button requires confirmation because running a new round
   * will invalidate the current plan/conclusion and require re-finalization.
   */
  isFinalized?: boolean;
  /** Human label for the deliverable produced by finalization (e.g. "Plan", "Conclusion"). */
  deliverableLabel?: string;
  /** Human-readable label shown on buttons while an agent is running (e.g. "Running workshop…"). */
  runningLabel?: string;
}

export function WorkshopPanel({
  rounds,
  backlogKind,
  backlogName,
  disabled,
  isSaving,
  isRunningWorkshop,
  onSaveRound,
  primaryActionLabel,
  onPrimaryAction,
  onRunWorkshop,
  workshopActionLabel = "Next Round",
  onDeleteRound,
  isDeletingRound,
  isFinalized,
  deliverableLabel = "Plan",
  runningLabel = "Running…",
}: WorkshopPanelProps) {
  // Confirmation dialog for running a new round after finalization.
  const [showPostFinalizeConfirm, setShowPostFinalizeConfirm] = useState(false);

  const [expandedRounds, setExpandedRounds] = useState<Set<number>>(() => {
    if (rounds.length === 0) return new Set();
    const last = rounds[rounds.length - 1];
    return new Set(last ? [last.round] : []);
  });
  const [localUpdates, setLocalUpdates] = useState<Map<string, WorkshopItem>>(new Map());
  const [deletedItems, setDeletedItems] = useState<Set<string>>(new Set());

  const toggleRound = useCallback((roundNum: number) => {
    setExpandedRounds((prev) => {
      const next = new Set(prev);
      if (next.has(roundNum)) {
        next.delete(roundNum);
      } else {
        next.add(roundNum);
      }
      return next;
    });
  }, []);

  const handleItemUpdate = useCallback((roundNum: number, updated: WorkshopItem) => {
    setLocalUpdates((prev) => {
      const next = new Map(prev);
      next.set(`${roundNum}:${updated.id}`, updated);
      return next;
    });
  }, []);

  const handleItemDelete = useCallback((roundNum: number, itemId: string) => {
    setDeletedItems((prev) => {
      const next = new Set(prev);
      next.add(`${roundNum}:${itemId}`);
      return next;
    });
  }, []);

  const getEffectiveItems = useCallback((round: WorkshopRound): WorkshopItem[] => {
    return round.items
      .filter((item) => !deletedItems.has(`${round.round}:${item.id}`))
      .map((item) => {
        const key = `${round.round}:${item.id}`;
        return localUpdates.get(key) ?? item;
      });
  }, [localUpdates, deletedItems]);

  const hasUnsavedChanges = localUpdates.size > 0 || deletedItems.size > 0;

  const handleSaveAll = useCallback(() => {
    for (const round of rounds) {
      const hasChangesInRound = (round.items ?? []).some((item) => {
        const key = `${round.round}:${item.id}`;
        return localUpdates.has(key) || deletedItems.has(key);
      });
      if (hasChangesInRound) {
        const effectiveItems = getEffectiveItems(round);
        const updatedRound: WorkshopRound = { ...round, items: effectiveItems };
        const content = buildWorkshopRoundContent(updatedRound);
        onSaveRound?.(round.round, content);
      }
    }
  }, [rounds, getEffectiveItems, onSaveRound, localUpdates, deletedItems]);

  // Auto-save: debounce changes and persist automatically.
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (!hasUnsavedChanges || !onSaveRound) return;
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(() => {
      handleSaveAll();
    }, 600);
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    };
  }, [hasUnsavedChanges, handleSaveAll, onSaveRound]);

  // Clear local edits once the parent finishes saving (isSaving transitions false).
  const prevIsSaving = useRef(isSaving);
  useEffect(() => {
    if (prevIsSaving.current && !isSaving) {
      setLocalUpdates(new Map());
      setDeletedItems(new Set());
    }
    prevIsSaving.current = isSaving;
  }, [isSaving]);

  /** Gate "Next Round" behind a confirmation if finalization already happened. */
  const handleWorkshopClick = useCallback(() => {
    if (isFinalized) {
      setShowPostFinalizeConfirm(true);
    } else {
      onRunWorkshop?.();
    }
  }, [isFinalized, onRunWorkshop]);

  if (rounds.length === 0) {
    return (
      <div className="rounded-lg border border-slate-700 bg-slate-800/50 p-4">
        <div className="text-center space-y-3">
          <p className="text-sm text-slate-400">No workshop rounds yet</p>
          <Button
            variant="outline"
            size="sm"
            disabled={disabled || isRunningWorkshop}
            onClick={onPrimaryAction ?? onRunWorkshop}
          >
            <Play className="mr-2 h-3.5 w-3.5" />
            {primaryActionLabel ?? "Start Workshop"}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {/* Confirmation dialog shown when user tries to run a new workshop round
          after finalization. A new round will invalidate the current deliverable
          and require another finalization pass. */}
      <ConfirmDialog
        isOpen={showPostFinalizeConfirm}
        onClose={() => setShowPostFinalizeConfirm(false)}
        onConfirm={() => {
          setShowPostFinalizeConfirm(false);
          onRunWorkshop?.();
        }}
        title="Start new workshop round?"
        description={`The ${deliverableLabel.toLowerCase()} has already been finalized from the current workshop answers. Running a new round will require re-finalization before this item can be executed.`}
        confirmLabel="Start Round"
      />
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-slate-200">
            Workshop Rounds ({rounds.length})
          </h3>
          {isFinalized && (
            <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-400">
              <CheckCircle2 className="h-3 w-3" />
              {deliverableLabel} finalized
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {onPrimaryAction && (
            <Button
              size="sm"
              disabled={disabled || isRunningWorkshop}
              onClick={onPrimaryAction}
            >
              <Play className="mr-2 h-3.5 w-3.5" />
              {isRunningWorkshop ? runningLabel : primaryActionLabel}
            </Button>
          )}
          {onRunWorkshop && (
            <Button
              variant="outline"
              size="sm"
              disabled={disabled || isRunningWorkshop}
              onClick={handleWorkshopClick}
            >
              <Play className="mr-2 h-3.5 w-3.5" />
              {isRunningWorkshop ? runningLabel : workshopActionLabel}
            </Button>
          )}
        </div>
      </div>

      {[...rounds].reverse().map((round) => {
        const isExpanded = expandedRounds.has(round.round);
        const effectiveItems = getEffectiveItems(round);
        const pendingDecisions = getPendingDecisionCount({ ...round, items: effectiveItems });
        // Find the previous round for delta comparison
        const roundIdx = rounds.findIndex((r) => r.round === round.round);
        const prevRound = roundIdx > 0 ? rounds[roundIdx - 1] : null;

        return (
          <div
            key={round.round}
            className="rounded-lg border border-slate-700 bg-slate-800/50"
          >
            <div className="flex items-center">
              <button
                type="button"
                className="flex flex-1 items-center justify-between px-4 py-3 text-left"
                onClick={() => toggleRound(round.round)}
              >
                <div className="flex items-center gap-3">
                  {isExpanded ? (
                    <ChevronDown className="h-4 w-4 text-slate-400" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-slate-400" />
                  )}
                  <span className="text-sm font-medium text-slate-200">
                    Round {round.round}
                  </span>
                  <ReadinessDots round={round} prevRound={prevRound} />
                </div>
                <div className="flex items-center gap-2 text-xs text-slate-500">
                  {pendingDecisions > 0 && (
                    <span className="text-amber-400">{pendingDecisions}D</span>
                  )}
                  <span>{(round.items ?? []).length} items</span>
                </div>
              </button>
              {onDeleteRound && !disabled && (
                <div className="pr-2">
                  <RoundMenu
                    onDelete={() => onDeleteRound(round.round)}
                    disabled={isDeletingRound}
                  />
                </div>
              )}
            </div>

            {isExpanded && (
              <div className="border-t border-slate-700 px-4 py-3 space-y-2">
                {round.plan_updates && (
                  <p className="text-xs text-slate-500 italic mb-2">
                    Plan updates: {round.plan_updates}
                  </p>
                )}
                {effectiveItems.map((item) => (
                  <WorkshopItemCard
                    key={item.id}
                    item={item}
                    disabled={disabled}
                    onUpdate={(updated) => handleItemUpdate(round.round, updated)}
                    onDelete={() => handleItemDelete(round.round, item.id)}
                    backlogKind={backlogKind}
                    backlogName={backlogName}
                    roundNumber={round.round}
                  />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
