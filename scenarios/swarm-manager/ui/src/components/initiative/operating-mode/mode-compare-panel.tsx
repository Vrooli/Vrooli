import { ArrowRight, Minus, Plus } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";
import { CapabilityList } from "./capability-list";
import { capabilityLabel, humanizeRunStrategy, humanizeTargetKind } from "./utils";

export interface ModeComparePanelProps {
  current: OperatingModeCatalogEntry;
  selected: OperatingModeCatalogEntry;
}

const CAPABILITY_FLAGS: ReadonlyArray<keyof OperatingModeCapabilities> = [
  "supportsPhases",
  "canStartPhases",
  "canCompleteItems",
  "canApplyBacklogSyncProposals",
  "requiresAcceptanceCriteria",
  "supportsArtifacts",
  "supportsHandoffs",
  "usesItemExecutionFlow",
];

interface Delta {
  flag: keyof OperatingModeCapabilities;
  kind: "added" | "removed";
}

function computeDeltas(current: OperatingModeCapabilities, selected: OperatingModeCapabilities): Delta[] {
  const out: Delta[] = [];
  for (const flag of CAPABILITY_FLAGS) {
    const cur = current[flag];
    const sel = selected[flag];
    if (cur === sel) continue;
    out.push({ flag, kind: sel ? "added" : "removed" });
  }
  return out;
}

export function ModeComparePanel({ current, selected }: ModeComparePanelProps) {
  const deltas = computeDeltas(current.capabilities, selected.capabilities);
  const sameMode = current.mode === selected.mode;

  return (
    <div
      className="rounded-lg border border-slate-800/80 bg-slate-900/40 p-4"
      data-testid={selectors.initiativeDetails.modePickerComparePanel}
    >
      <div className="grid gap-4 md:grid-cols-[1fr_auto_1fr]">
        <Column label="Currently" mode={current} />
        <div className="hidden items-center justify-center text-slate-500 md:flex">
          <ArrowRight className="h-5 w-5" aria-hidden="true" />
        </div>
        <Column label="Switching to" mode={selected} highlight={!sameMode} />
      </div>

      <div className="mt-4 border-t border-slate-800 pt-3">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-500">Changes</p>
        {sameMode || deltas.length === 0 ? (
          <p className="mt-2 text-sm italic text-slate-500">No capability changes.</p>
        ) : (
          <ul className="mt-2 space-y-1">
            {deltas.map((delta) => (
              <li key={delta.flag} className="flex items-center gap-2 text-sm">
                {delta.kind === "added" ? (
                  <Plus className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" />
                ) : (
                  <Minus className="h-3.5 w-3.5 text-amber-400" aria-hidden="true" />
                )}
                <span className={delta.kind === "added" ? "text-emerald-300" : "text-amber-300"}>
                  {delta.kind === "added" ? "Adds" : "Removes"} {capabilityLabel(delta.flag).toLowerCase()}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Column({
  label,
  mode,
  highlight,
}: {
  label: string;
  mode: OperatingModeCatalogEntry;
  highlight?: boolean;
}) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p className={`mt-1 text-sm font-semibold ${highlight ? "text-cyan-200" : "text-slate-100"}`}>
        {mode.label}
      </p>
      <p className="mt-1 text-[11px] text-slate-500">
        {humanizeTargetKind(mode.targetKind)} · {humanizeRunStrategy(mode.runStrategy)}
      </p>
      <CapabilityList capabilities={mode.capabilities} variant="compact" />
    </div>
  );
}
