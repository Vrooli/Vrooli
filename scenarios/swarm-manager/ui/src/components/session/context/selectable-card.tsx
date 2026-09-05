/**
 * PickModeRow — the shared pick-mode wrapper used by every context-entity
 * card when rendered inside the SessionContextPicker. It owns the checkbox
 * affordance, selected/disabled styling, the `contextRow` test selector, and
 * the toggle click. Cards pass their rich interior as `children`.
 */
import type { ReactNode } from "react";
import { Check } from "lucide-react";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import type { CardSelection } from "./selectable";

interface PickModeRowProps {
  selection: CardSelection;
  children: ReactNode;
  className?: string;
}

export function PickModeRow({ selection, children, className }: PickModeRowProps) {
  const { selected, disabled, disabledReason, onToggleSelect } = selection;
  return (
    <button
      type="button"
      disabled={disabled}
      title={disabled ? disabledReason : undefined}
      aria-pressed={selected}
      onClick={() => {
        if (!disabled) onToggleSelect?.();
      }}
      className={cn(
        "flex w-full items-start gap-2.5 rounded-md border px-2.5 py-2 text-left transition-colors",
        selected
          ? "border-cyan-400/50 bg-cyan-400/10 text-cyan-50"
          : "border-slate-800 bg-slate-950/45 text-slate-200 hover:border-slate-700 hover:bg-slate-800/55",
        disabled && "cursor-not-allowed opacity-50 hover:border-slate-800 hover:bg-slate-950/45",
        className,
      )}
      data-testid={selectors.agentSessions.contextRow}
    >
      <span
        className={cn(
          "mt-0.5 flex h-[1.125rem] w-[1.125rem] shrink-0 items-center justify-center rounded border transition-colors",
          selected ? "border-cyan-300 bg-cyan-300 text-slate-950" : "border-slate-600 bg-slate-900",
        )}
      >
        {selected && <Check className="h-3.5 w-3.5" />}
      </span>
      <div className="min-w-0 flex-1">{children}</div>
    </button>
  );
}
