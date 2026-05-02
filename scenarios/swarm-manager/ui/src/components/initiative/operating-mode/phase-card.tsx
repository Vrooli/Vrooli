/**
 * PhaseCard
 *
 * Detail card for a single operating-mode phase. Used inside the Phases section
 * of the Operating Mode Details Page (in both list and graph views — the graph
 * view scrolls/highlights the matching card on node click).
 *
 * Layout:
 *   header  : title + snake_case ID chip + start/terminal markers
 *   body    : purpose paragraph + chip cluster (profile, writes-repo, contract)
 *             + output artifacts list
 *   footer  : <PhaseInternalsDisclosure> (catalog/skill IDs, trigger, metrics)
 */

import { Info } from "lucide-react";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { StatusChip } from "../../ui/status-chip";
import type { OperatingModeCatalogPhase } from "../../../types/operating-mode";
import { PhaseInternalsDisclosure } from "./phase-internals-disclosure";
import { phaseCardDomId } from "./utils";

const PROFILE_KEY_HELP =
  "Agent-manager profile this phase runs under. Different profiles vary the model, tool access, and runtime budget the agent gets while executing the phase.";

interface PhaseCardProps {
  phase: OperatingModeCatalogPhase;
  highlighted?: boolean;
  defaultInternalsOpen?: boolean;
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

const PROFILE_COLORS = {
  background: "bg-slate-800/80",
  border: "border-slate-700",
  text: "text-slate-200",
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

const VERDICT_COLORS = {
  background: "bg-cyan-500/10",
  border: "border-cyan-500/30",
  text: "text-cyan-300",
};

const HANDOFF_COLORS = {
  background: "bg-violet-500/10",
  border: "border-violet-500/30",
  text: "text-violet-300",
};

const PROGRESS_COLORS = {
  background: "bg-cyan-500/10",
  border: "border-cyan-500/30",
  text: "text-cyan-300",
};

const STRUCTURED_COLORS = {
  background: "bg-indigo-500/10",
  border: "border-indigo-500/30",
  text: "text-indigo-300",
};

export function PhaseCard({ phase, highlighted, defaultInternalsOpen }: PhaseCardProps) {
  const writesRepoLabel = phase.writesRepo ? "writes repo" : "read-only";
  const writesRepoColors = phase.writesRepo ? WRITES_REPO_COLORS : READ_ONLY_COLORS;
  const headline = phase.title || phase.phase;
  const contract = phase.outputContract;

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
      </header>

      {phase.purpose && (
        <p className="mt-2 text-sm leading-relaxed text-slate-300">{phase.purpose}</p>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        <StatusChip label={phase.profileKey} colors={PROFILE_COLORS} title={PROFILE_KEY_HELP} />
        <details
          className="group/profile inline-flex items-center"
          data-testid={selectors.initiativeDetails.phaseCardProfileInfo}
        >
          <summary
            className={cn(
              "flex cursor-pointer list-none items-center rounded p-0.5 text-slate-500",
              "hover:text-cyan-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
            )}
            title="What's an agent profile?"
            aria-label="What is an agent profile?"
          >
            <Info className="h-3.5 w-3.5" aria-hidden />
          </summary>
          <p className="mt-1 max-w-xs basis-full rounded-md border border-slate-700/80 bg-slate-900/80 p-2 text-[11px] leading-snug text-slate-300">
            {PROFILE_KEY_HELP}
          </p>
        </details>
        <StatusChip label={writesRepoLabel} colors={writesRepoColors} />
        {phase.requiresCriteria && <StatusChip label="requires criteria" colors={CRITERIA_COLORS} />}
        {contract.requiresStructuredResult && (
          <StatusChip label="structured" colors={STRUCTURED_COLORS} title="Phase emits a structured result" />
        )}
        {contract.requiresVerdict && (
          <StatusChip label="verdict" colors={VERDICT_COLORS} title="Phase produces an acceptance verdict" />
        )}
        {contract.requiresHandoff && (
          <StatusChip label="handoff" colors={HANDOFF_COLORS} title="Phase emits a handoff packet" />
        )}
        {contract.requiresProgress && (
          <StatusChip label="progress" colors={PROGRESS_COLORS} title="Phase records progress decision" />
        )}
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
