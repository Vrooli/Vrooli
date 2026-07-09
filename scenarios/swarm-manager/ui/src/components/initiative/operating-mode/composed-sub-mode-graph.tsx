/**
 * ComposedSubModeGraph
 *
 * Renders a delegated phase's sub-mode (`executed_by`) composed graph inline:
 * the sub-mode's phases, in order, marked as delegated. This makes the two-layer
 * composed structure legible at the exact point it applies without re-deriving
 * routing — the backend engine remains the single source of truth for how the
 * composed flow advances (EXECUTION-MODES.md, D3). The sub-mode entry is read
 * from the mode catalog for display only.
 */

import { StatusChip } from "../../ui/status-chip";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { humanizeTargetKind } from "./utils";

const START_COLORS = {
  background: "bg-emerald-500/10",
  border: "border-emerald-500/30",
  text: "text-emerald-300",
};

const TERMINAL_COLORS = {
  background: "bg-violet-500/10",
  border: "border-violet-500/30",
  text: "text-violet-300",
};

export function ComposedSubModeGraph({
  subModeId,
  subMode,
}: {
  subModeId: string;
  subMode?: OperatingModeCatalogEntry;
}) {
  return (
    <section
      className="rounded-md border border-cyan-500/20 bg-cyan-500/5 p-3"
      data-testid="composed-sub-mode-graph"
      data-sub-mode={subModeId}
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <p className="text-xs font-medium text-cyan-200">
          Executed by sub-mode <code className="font-mono text-cyan-100">{subModeId}</code>
        </p>
        {subMode && (
          <span className="text-[11px] text-slate-400">target: {humanizeTargetKind(subMode.targetKind)}</span>
        )}
      </div>
      <p className="mt-0.5 text-[11px] leading-relaxed text-slate-400">
        The sub-mode's loop runs as this phase, one level deep. Routing stays in the backend; this is the composed
        graph inline.
      </p>
      {subMode ? (
        <ol className="mt-2 space-y-1.5">
          {subMode.phases.map((phase) => (
            <li
              key={phase.phase}
              className="flex flex-wrap items-center gap-1.5 text-xs text-slate-300"
              data-testid={`composed-phase-${phase.phase}`}
            >
              <span className="text-slate-200">{phase.label || phase.title || phase.phase}</span>
              <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-300">
                {phase.phase}
              </code>
              {phase.isStart && <StatusChip label="start" colors={START_COLORS} />}
              {phase.isTerminal && <StatusChip label="terminal" colors={TERMINAL_COLORS} />}
              {phase.classification && (
                <span className="text-[11px] text-cyan-300/90">
                  classifies <code className="font-mono">{phase.classification.field}</code>
                </span>
              )}
            </li>
          ))}
        </ol>
      ) : (
        <p className="mt-2 text-[11px] italic text-slate-500">
          Sub-mode <code className="font-mono">{subModeId}</code> is not in the catalog yet.
        </p>
      )}
    </section>
  );
}
