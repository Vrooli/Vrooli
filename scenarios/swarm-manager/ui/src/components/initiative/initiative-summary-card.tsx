/**
 * InitiativeSummaryCard — the compact initiative card shared by the sidebar
 * InitiativesTab and the SessionContextPicker.
 *
 * - Sidebar mode (no `selection`): a button that opens the initiative, with an
 *   optional bulk-selection checkbox. Behavior is identical to the previous
 *   inlined markup.
 * - Pick mode (`selection.selectionMode`): renders inside PickModeRow.
 */
import { memo } from "react";
import { Archive } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib/format-utils";
import type { InitiativeWithRollup } from "../../types";
import { NoteIndicator } from "../ui/note-indicator";
import { RollupProgressBar, rollupTotal } from "../ui/rollup-progress-bar";
import { PickModeRow } from "../session/context/selectable-card";
import type { CardSelection } from "../session/context/selectable";

const STATUS_COLORS: Record<string, string> = {
  active: "bg-cyan-500/20 text-cyan-300",
  completed: "bg-green-500/20 text-green-300",
};

export interface InitiativeSummaryCardProps {
  item: InitiativeWithRollup;
  // Sidebar mode
  onOpen?: () => void;
  batchMode?: boolean;
  batchSelected?: boolean;
  onBatchToggle?: () => void;
  // Picker pick mode
  selection?: CardSelection;
}

function InitiativeCardBody({ item }: { item: InitiativeWithRollup }) {
  const { initiative, rollup } = item;
  const deps = (initiative as { dependsOn?: string[] }).dependsOn ?? [];
  return (
    <>
      {(initiative as { archivedAt?: string }).archivedAt != null && (
        <div className="mb-1.5 flex items-center gap-1.5 rounded border border-amber-500/20 bg-amber-500/5 px-2 py-1 text-[11px] text-amber-400/80">
          <Archive className="h-3 w-3 shrink-0" />
          Archived
        </div>
      )}
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
          {initiative.title || initiative.name}
        </p>
        <div className="flex shrink-0 items-center gap-1.5">
          <NoteIndicator note={initiative.note} />
          <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[initiative.status] ?? "bg-slate-700/60 text-slate-300")}>
            {initiative.status}
          </span>
        </div>
      </div>
      {rollupTotal(rollup) > 0 && (
        <>
          <RollupProgressBar rollup={rollup} barHeight="h-1.5" className="mt-2" />
          <div className="mt-1 flex flex-wrap gap-2 text-[11px]">
            <span className="text-emerald-400">{rollup.completed} done</span>
            <span className="text-purple-400">{rollup.inProgress} active</span>
            {rollup.failed > 0 && <span className="text-red-400">{rollup.failed} failed</span>}
            <span className="text-slate-500">{rollup.pending} pending</span>
          </div>
        </>
      )}
      {deps.length > 0 && (
        <p className="mt-1 text-[11px] text-slate-400">
          <span className="text-slate-500">Depends on:</span> {deps.join(", ")}
        </p>
      )}
      <p className="mt-1 text-[11px] text-slate-500">{formatRelativeTime(initiative.updated)}</p>
    </>
  );
}

function InitiativeSummaryCardImpl({
  item,
  onOpen,
  batchMode = false,
  batchSelected = false,
  onBatchToggle,
  selection,
}: InitiativeSummaryCardProps) {
  if (selection?.selectionMode) {
    return (
      <PickModeRow selection={selection}>
        <InitiativeCardBody item={item} />
      </PickModeRow>
    );
  }

  const { initiative } = item;
  return (
    <button
      type="button"
      onClick={() => onOpen?.()}
      className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-initiative-item"
    >
      <div className="flex items-start gap-2">
        {batchMode && (
          <input
            type="checkbox"
            aria-label={`${batchSelected ? "Deselect" : "Select"} ${initiative.title || initiative.name}`}
            checked={batchSelected}
            onClick={(event) => event.stopPropagation()}
            onChange={(event) => {
              event.stopPropagation();
              onBatchToggle?.();
            }}
            className="mt-0.5"
          />
        )}
        <div className="min-w-0 flex-1">
          <InitiativeCardBody item={item} />
        </div>
      </div>
    </button>
  );
}

export const InitiativeSummaryCard = memo(InitiativeSummaryCardImpl);
