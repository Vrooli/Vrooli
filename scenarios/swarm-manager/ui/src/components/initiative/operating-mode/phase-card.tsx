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

import { useState } from "react";
import { BookOpen, Info } from "lucide-react";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { StatusChip } from "../../ui/status-chip";
import type {
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
} from "../../../types/operating-mode";
import { PhaseInternalsDisclosure } from "./phase-internals-disclosure";
import { PhaseProfilePopover } from "./phase-profile-popover";
import { SkillViewerDialog } from "./skill-viewer-dialog";
import {
  formatTransition,
  phaseEmitSchema,
  PHASE_READS,
  workedExampleForPhase,
} from "./phase-interpretability";
import { phaseCardDomId } from "./utils";

interface PhaseCardProps {
  phase: OperatingModeCatalogPhase;
  transitions?: OperatingModePhaseTransition[];
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

const REQUIRED_COLORS = {
  background: "bg-amber-500/10",
  border: "border-amber-500/30",
  text: "text-amber-300",
};

export function PhaseCard({ phase, transitions = [], highlighted, defaultInternalsOpen }: PhaseCardProps) {
  const writesRepoLabel = phase.writesRepo ? "writes repo" : "read-only";
  const writesRepoColors = phase.writesRepo ? WRITES_REPO_COLORS : READ_ONLY_COLORS;
  const headline = phase.label || phase.title || phase.phase;
  const [profilePopoverOpen, setProfilePopoverOpen] = useState(false);
  const [skillDialogOpen, setSkillDialogOpen] = useState(false);
  const emitSchema = phaseEmitSchema(phase);
  const workedExample = workedExampleForPhase(phase);

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
        <button
          type="button"
          onClick={() => setProfilePopoverOpen(true)}
          aria-label={`Explain agent profile ${phase.profileKey}`}
          data-testid={selectors.initiativeDetails.phaseCardProfileChip}
          className="group inline-flex items-center gap-1 rounded-full border border-slate-700 bg-slate-800/80 px-2 py-0.5 text-[11px] text-slate-200 transition-colors hover:border-cyan-500/60 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
        >
          <span className="font-medium">{phase.profileKey}</span>
          <Info className="h-3 w-3 text-slate-500 transition-colors group-hover:text-cyan-300" aria-hidden />
        </button>
        {profilePopoverOpen && (
          <PhaseProfilePopover
            profileKey={phase.profileKey}
            isOpen
            onClose={() => setProfilePopoverOpen(false)}
          />
        )}
        <StatusChip label={writesRepoLabel} colors={writesRepoColors} />
        {phase.requiresCriteria && <StatusChip label="requires criteria" colors={CRITERIA_COLORS} />}
      </div>

      {phase.skillId && (
        <button
          type="button"
          onClick={() => setSkillDialogOpen(true)}
          className={cn(
            "mt-3 flex w-full items-center justify-between gap-3 rounded-md border border-cyan-500/20 bg-cyan-500/10 px-3 py-2 text-left text-xs text-cyan-100 transition-colors",
            "hover:border-cyan-400/40 hover:bg-cyan-500/15",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
          )}
          data-testid="phase-agent-instructions"
        >
          <span className="min-w-0">
            <span className="block font-medium">Agent instructions</span>
            <code className="mt-0.5 block truncate font-mono text-[11px] text-cyan-200/80">
              {phase.skillId}
            </code>
          </span>
          <BookOpen className="h-4 w-4 shrink-0 text-cyan-300" aria-hidden />
        </button>
      )}

      <div className="mt-4 grid gap-3 lg:grid-cols-2">
        <section className="rounded-md border border-slate-800 bg-slate-950/40 p-3">
          <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Reads</h4>
          <ul className="mt-2 space-y-2">
            {PHASE_READS.map((read) => (
              <li key={read.key} className="text-xs text-slate-300">
                <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-100">
                  {read.key}
                </code>
                <span className="ml-2 font-medium text-slate-200">{read.label}</span>
                <p className="mt-1 leading-relaxed text-slate-500">{read.meaning}</p>
              </li>
            ))}
          </ul>
        </section>

        <section className="rounded-md border border-slate-800 bg-slate-950/40 p-3">
          <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Emits schema</h4>
          <ul className="mt-2 space-y-2">
            {emitSchema.map((emit) => (
              <li key={emit.field} className="text-xs text-slate-300">
                <div className="flex flex-wrap items-center gap-2">
                  <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-100">
                    {emit.label}
                  </code>
                  {emit.required && <StatusChip label="required" colors={REQUIRED_COLORS} />}
                </div>
                <p className="mt-1 leading-relaxed text-slate-500">{emit.meaning}</p>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <section className="mt-3 rounded-md border border-slate-800 bg-slate-950/40 p-3">
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Transitions</h4>
        {transitions.length > 0 ? (
          <ul className="mt-2 space-y-1">
            {transitions.map((transition) => (
              <li
                key={`${transition.from}-${transition.to}-${transition.label}`}
                className="text-xs text-slate-300"
              >
                {formatTransition(transition)}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-2 text-xs text-slate-500">No outgoing transition; this phase is terminal.</p>
        )}
      </section>

      <section className="mt-3 rounded-md border border-slate-800 bg-slate-950/40 p-3">
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">
          {workedExample.title}
        </h4>
        <pre className="mt-2 overflow-x-auto rounded bg-slate-950 p-2 text-[11px] leading-relaxed text-slate-300">
          {JSON.stringify({ operating_mode_result: workedExample.result }, null, 2)}
        </pre>
      </section>

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
      {skillDialogOpen && (
        <SkillViewerDialog
          isOpen={skillDialogOpen}
          onClose={() => setSkillDialogOpen(false)}
          skillId={phase.skillId}
        />
      )}
    </article>
  );
}
