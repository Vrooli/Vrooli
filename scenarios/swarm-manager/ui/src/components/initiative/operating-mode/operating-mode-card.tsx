import { cn } from "../../../lib/utils";
import { Card } from "../../ui/card";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { humanizeRunStrategy, humanizeTargetKind, modeLabel } from "./utils";
import { PickModeRow } from "../../session/context/selectable-card";
import type { CardSelection } from "../../session/context/selectable";

export interface OperatingModeCardProps {
  mode: OperatingModeCatalogEntry;
  selected?: boolean;
  onClick?: () => void;
  compact?: boolean;
  className?: string;
  "data-testid"?: string;
  /** Picker pick-mode contract. When set, renders inside PickModeRow. */
  selection?: CardSelection;
}

export function OperatingModeCard({
  mode,
  selected,
  onClick,
  compact,
  className,
  "data-testid": testId,
  selection,
}: OperatingModeCardProps) {
  const interactive = Boolean(onClick);
  // Cards in the picker live in an evenly-sized grid, so they keep a uniform
  // tight shape regardless of selection state. Decision-support detail
  // (full description, best-for / not-for / tradeoffs callouts) renders in
  // a full-width detail block below the grid where it can breathe — see
  // `mode-picker-dialog.tsx`.
  const descriptionClamp = compact ? "line-clamp-1" : "line-clamp-2";

  const body = (
    <>
      <div className="flex items-start justify-between gap-2">
        {/* Label routes through the member-item-strategy mapping so the
            legacy item-level entry presents as the workflow strategy. */}
        <p className="line-clamp-2 text-sm font-medium leading-snug text-slate-100">
          {modeLabel(mode.mode, mode.label)}
        </p>
        <span className="shrink-0 rounded-full bg-slate-700/60 px-2 py-0.5 text-[10px] font-medium text-slate-300">
          {mode.usageCount} init.
        </span>
      </div>
      {mode.description && (
        <p className={cn("mt-1.5 text-xs text-slate-400", descriptionClamp)}>
          {mode.description}
        </p>
      )}
      <p className="mt-1.5 text-[11px] text-slate-500">
        {humanizeTargetKind(mode.targetKind)}
        {mode.runStrategy ? ` · ${humanizeRunStrategy(mode.runStrategy)}` : " · workflow strategy"}
        {mode.default ? " · default" : ""}
      </p>
    </>
  );

  if (selection?.selectionMode) {
    return <PickModeRow selection={selection}>{body}</PickModeRow>;
  }

  const cardClassName = cn(
    "block w-full text-left",
    selected && "border-cyan-400/60 ring-2 ring-cyan-400/40",
    className,
  );

  if (interactive) {
    return (
      <button
        type="button"
        onClick={onClick}
        data-testid={testId}
        aria-pressed={selected}
        className="block w-full text-left"
      >
        <Card padding="sm" interactive className={cardClassName}>
          {body}
        </Card>
      </button>
    );
  }

  return (
    <Card padding="sm" className={cardClassName} data-testid={testId}>
      {body}
    </Card>
  );
}
