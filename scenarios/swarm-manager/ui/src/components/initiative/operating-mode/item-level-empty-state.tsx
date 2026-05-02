import { ArrowRightLeft, CheckCircle2, Loader2, ListChecks } from "lucide-react";
import { Button } from "../../ui/button";
import { selectors } from "../../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../../types";
import type { OperatingModeWorkspace } from "../../../types/operating-mode";

export interface ItemLevelEmptyStateProps {
  initiative: Initiative;
  rollup?: InitiativeRollup;
  workspace?: OperatingModeWorkspace;
  onSwitchClick: () => void;
}

export function ItemLevelEmptyState({
  initiative,
  rollup,
  workspace,
  onSwitchClick,
}: ItemLevelEmptyStateProps) {
  const itemCount = (initiative.items ?? []).length;
  const completed = rollup?.completed ?? 0;
  // The operating-mode workspace lock is set when an initiative-scoped agent
  // round is running. For item-level mode that's never the case (item-level
  // rounds run through the existing item-execution flow, not the workspace),
  // so prefer the rollup's in-progress count, falling back to "agent running"
  // only when the lock is set without rollup data available.
  const inFlight = rollup?.inProgress ?? (workspace?.lock ? 1 : 0);

  return (
    <div className="space-y-4" data-testid={selectors.initiativeDetails.itemLevelEmptyState}>
      <ul className="space-y-1.5 text-sm text-slate-300">
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>Each backlog item runs through the existing execution flow.</span>
        </li>
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>Agents only read initiative state — they do not write to phase artifacts.</span>
        </li>
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>Switch to a phase-capable mode to coordinate work across items.</span>
        </li>
      </ul>

      <dl className="grid grid-cols-3 gap-3">
        <Stat icon={ListChecks} label="Items" value={itemCount} />
        <Stat icon={CheckCircle2} label="Completed" value={completed} valueClassName="text-emerald-300" />
        <Stat icon={Loader2} label="In flight" value={inFlight} valueClassName="text-cyan-300" />
      </dl>

      <div className="flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={onSwitchClick}
          data-testid={selectors.initiativeDetails.itemLevelEmptyStateSwitchButton}
        >
          <ArrowRightLeft className="mr-1.5 h-4 w-4" />
          Switch to phase-capable mode
        </Button>
      </div>
    </div>
  );
}

function Stat({
  icon: Icon,
  label,
  value,
  valueClassName,
}: {
  icon: typeof ListChecks;
  label: string;
  value: number;
  valueClassName?: string;
}) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-3 text-center">
      <Icon className="mx-auto h-4 w-4 text-slate-500" aria-hidden="true" />
      <dt className="mt-1 text-[11px] uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={`mt-1 text-xl font-semibold ${valueClassName ?? "text-slate-100"}`}>{value}</dd>
    </div>
  );
}
