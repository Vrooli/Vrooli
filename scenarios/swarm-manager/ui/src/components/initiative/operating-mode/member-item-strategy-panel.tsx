/**
 * MemberItemStrategyPanel — the member-item workflow strategy explainer.
 *
 * Formerly `ItemLevelEmptyState` ("How Item-Level Works"). In the declarative
 * model there is no "item-level mode": items run their own workflows and the
 * initiative provides strategy configuration (scheduling across member
 * items). This panel presents that vocabulary; the persisted wire value stays
 * `item-level` until Phase 8 (see lib/member-item-strategy.ts).
 */

import { ArrowRightLeft, CheckCircle2, Loader2, ListChecks } from "lucide-react";
import { Button } from "../../ui/button";
import { selectors } from "../../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../../types";
import type { OperatingModeWorkspace } from "../../../types/operating-mode";

export interface MemberItemStrategyPanelProps {
  initiative: Initiative;
  rollup?: InitiativeRollup;
  workspace?: OperatingModeWorkspace;
  onSwitchClick: () => void;
}

export function MemberItemStrategyPanel({
  initiative,
  rollup,
  workspace,
  onSwitchClick,
}: MemberItemStrategyPanelProps) {
  const itemCount = (initiative.items ?? []).length;
  const completed = rollup?.completed ?? 0;
  // The operating-mode workspace lock is set when an initiative-scoped agent
  // round is running. Under the member-item strategy that's never the case
  // (member items run their own workflows, not the initiative workspace), so
  // prefer the rollup's in-progress count, falling back to "agent running"
  // only when the lock is set without rollup data available.
  const inFlight = rollup?.inProgress ?? (workspace?.lock ? 1 : 0);

  return (
    <div className="space-y-4" data-testid={selectors.initiativeDetails.memberItemStrategyPanel}>
      <ul className="space-y-1.5 text-sm text-slate-300">
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>Each member item runs its own workflow through the standard execution flow.</span>
        </li>
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>
            The initiative provides strategy configuration — it schedules work across items and
            does not run a methodology loop of its own.
          </span>
        </li>
        <li className="flex gap-2">
          <span className="text-slate-500">•</span>
          <span>Switch to an operating mode to coordinate work across items with a phase loop.</span>
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
          data-testid={selectors.initiativeDetails.memberItemStrategyPanelSwitchButton}
        >
          <ArrowRightLeft className="mr-1.5 h-4 w-4" />
          Switch to an operating mode
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
