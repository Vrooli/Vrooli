/**
 * ScenarioSummaryCard — compact scenario card for the SessionContextPicker.
 *
 * There is no scenario sidebar tab, so this card is additive: it primarily
 * exists to give the picker a rich, type-appropriate row. It still implements
 * the shared `CardSelection` contract so it behaves like every other pick-mode
 * card, and renders a plain (non-interactive) summary when no selection is
 * supplied.
 */
import { memo } from "react";
import { cn } from "../../lib/utils";
import type { Scenario } from "../../types";
import { PickModeRow } from "../session/context/selectable-card";
import type { CardSelection } from "../session/context/selectable";

const STATUS_COLORS: Record<Scenario["status"], string> = {
  running: "bg-green-500/20 text-green-300",
  stopped: "bg-slate-700/60 text-slate-300",
  error: "bg-red-500/20 text-red-300",
  unknown: "bg-slate-700/40 text-slate-500",
};

export interface ScenarioSummaryCardProps {
  scenario: Scenario;
  selection?: CardSelection;
}

function ScenarioCardBody({ scenario }: { scenario: Scenario }) {
  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
          {scenario.displayName || scenario.name}
        </p>
        <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[scenario.status])}>
          {scenario.status}
        </span>
      </div>
      {scenario.description && (
        <p className="mt-1 line-clamp-2 text-[11px] text-slate-400">{scenario.description}</p>
      )}
      <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
        <span className="font-mono">{scenario.name}</span>
        {scenario.isGreenfield && <span className="text-emerald-400">greenfield</span>}
      </div>
    </>
  );
}

function ScenarioSummaryCardImpl({ scenario, selection }: ScenarioSummaryCardProps) {
  if (selection?.selectionMode) {
    return (
      <PickModeRow selection={selection}>
        <ScenarioCardBody scenario={scenario} />
      </PickModeRow>
    );
  }

  return (
    <div className="rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5">
      <ScenarioCardBody scenario={scenario} />
    </div>
  );
}

export const ScenarioSummaryCard = memo(ScenarioSummaryCardImpl);
