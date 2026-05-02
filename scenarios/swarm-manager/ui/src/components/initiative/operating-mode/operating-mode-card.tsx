import { cn } from "../../../lib/utils";
import { Card } from "../../ui/card";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { humanizeRunStrategy, humanizeScopeKind } from "./utils";

export interface OperatingModeCardProps {
  mode: OperatingModeCatalogEntry;
  selected?: boolean;
  onClick?: () => void;
  compact?: boolean;
  className?: string;
  "data-testid"?: string;
}

export function OperatingModeCard({
  mode,
  selected,
  onClick,
  compact,
  className,
  "data-testid": testId,
}: OperatingModeCardProps) {
  const interactive = Boolean(onClick);
  const descriptionClamp = compact ? "line-clamp-1" : "line-clamp-2";

  const body = (
    <>
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-sm font-medium leading-snug text-slate-100">
          {mode.label}
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
        {humanizeScopeKind(mode.scopeKind)} · {humanizeRunStrategy(mode.runStrategy)}
        {mode.default ? " · default" : ""}
      </p>
    </>
  );

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
