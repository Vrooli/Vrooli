/**
 * PhaseCard
 *
 * Detail card for a single operating-mode phase. Used inside the Phases section
 * of the Operating Mode Details Page (in both list and graph views — the graph
 * view scrolls/highlights the matching card on node click).
 *
 * Layout:
 *   header  : title + snake_case ID chip + start/terminal markers
 *   body    : purpose paragraph + chip cluster (profile, writes-repo, criteria)
 *             + the shared <PhaseViewer> in Contract source (Instructions /
 *             Reads / Emits / Transition) + output artifacts list
 *   footer  : <PhaseInternalsDisclosure> (catalog/skill IDs, trigger, metrics)
 */

import { cn } from "../../../lib/utils";
import { StatusChip } from "../../ui/status-chip";
import type {
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
} from "../../../types/operating-mode";
import { PhaseInternalsDisclosure } from "./phase-internals-disclosure";
import { PhaseViewer } from "./phase-viewer";
import { ComposedSubModeGraph } from "./composed-sub-mode-graph";
import { contractPhaseView } from "./phase-view";
import { phaseCardDomId } from "./utils";

interface PhaseCardProps {
  phase: OperatingModeCatalogPhase;
  transitions?: OperatingModePhaseTransition[];
  highlighted?: boolean;
  defaultInternalsOpen?: boolean;
  /**
   * Lookup of sub-mode catalog entries keyed by mode id, used to render a
   * delegated phase's composed graph inline. The backend stays the routing
   * source of truth; this only reads each sub-mode's own catalog for display.
   */
  subModes?: Record<string, OperatingModeCatalogEntry>;
}

const START_COLORS = {
  background: "bg-emerald-500/10",
  border: "border-emerald-500/30",
  text: "text-emerald-300",
  dot: "bg-emerald-400",
};

const TERMINAL_COLORS = {
  background: "bg-violet-500/10",
  border: "border-violet-500/30",
  text: "text-violet-300",
  dot: "bg-violet-400",
};

const WRITES_REPO_COLORS = {
  background: "bg-emerald-500/10",
  border: "border-emerald-500/30",
  text: "text-emerald-300",
};

const READ_ONLY_COLORS = {
  background: "bg-slate-800/80",
  border: "border-slate-700",
  text: "text-slate-300",
};

const CRITERIA_COLORS = {
  background: "bg-amber-500/10",
  border: "border-amber-500/30",
  text: "text-amber-300",
};

const DELEGATED_COLORS = {
  background: "bg-cyan-500/10",
  border: "border-cyan-500/30",
  text: "text-cyan-300",
  dot: "bg-cyan-400",
};

export function PhaseCard({ phase, transitions = [], highlighted, defaultInternalsOpen, subModes }: PhaseCardProps) {
  const writesRepoLabel = phase.writesRepo ? "writes repo" : "read-only";
  const writesRepoColors = phase.writesRepo ? WRITES_REPO_COLORS : READ_ONLY_COLORS;
  const headline = phase.label || phase.title || phase.phase;
  const delegated = phase.executedBy ?? "";

  return (
    <article
      id={phaseCardDomId(phase.phase)}
      data-testid="phase-card"
      data-phase={phase.phase}
      className={cn(
        "rounded-lg border bg-slate-900/40 p-4 transition-colors",
        highlighted ? "border-cyan-500/60 ring-2 ring-cyan-500/40" : "border-slate-800",
      )}
    >
      <header className="flex flex-wrap items-center gap-2">
        <h3 className="text-base font-semibold text-slate-100">{headline}</h3>
        <code className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-300">
          {phase.phase}
        </code>
        {phase.isStart && <StatusChip label="start" colors={START_COLORS} leadingDot />}
        {phase.isTerminal && <StatusChip label="terminal" colors={TERMINAL_COLORS} leadingDot />}
        {delegated && <StatusChip label="delegated" colors={DELEGATED_COLORS} leadingDot />}
      </header>

      {phase.purpose && (
        <p className="mt-2 text-sm leading-relaxed text-slate-300">{phase.purpose}</p>
      )}

      {!delegated && (
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          <StatusChip label={writesRepoLabel} colors={writesRepoColors} />
          {phase.requiresCriteria && <StatusChip label="requires criteria" colors={CRITERIA_COLORS} />}
        </div>
      )}

      {delegated && (
        <div className="mt-3">
          <ComposedSubModeGraph subModeId={delegated} subMode={subModes?.[delegated]} />
        </div>
      )}

      <div className="mt-4">
        <PhaseViewer view={contractPhaseView(phase, transitions)} hideHeader />
      </div>

      {phase.outputArtifacts && phase.outputArtifacts.length > 0 && (
        <div className="mt-3">
          <p className="mb-1 text-[11px] uppercase tracking-wide text-slate-500">Output artifacts</p>
          <ul className="space-y-1">
            {phase.outputArtifacts.map((artifact) => (
              <li
                key={artifact.path}
                className="flex flex-wrap items-center gap-2 text-xs text-slate-300"
              >
                <span
                  aria-hidden
                  className={cn(
                    "inline-block h-1.5 w-1.5 shrink-0 rounded-full",
                    artifact.required ? "bg-amber-400" : "bg-slate-600",
                  )}
                />
                <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-200">
                  {artifact.path}
                </code>
                {artifact.contentType && (
                  <span className="text-[11px] text-slate-500">{artifact.contentType}</span>
                )}
                {artifact.required && (
                  <span className="text-[11px] font-medium text-amber-400">required</span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mt-3">
        <PhaseInternalsDisclosure phase={phase} defaultOpen={defaultInternalsOpen} />
      </div>
    </article>
  );
}
