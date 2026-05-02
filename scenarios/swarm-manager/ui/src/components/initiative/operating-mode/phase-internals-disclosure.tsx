/**
 * PhaseInternalsDisclosure
 *
 * Collapsible "Internals" panel that exposes the registry-level metadata for a
 * phase (catalog/skill IDs, activity/lock purposes, trigger copy, metrics flags,
 * result bindings). Uses native <details>/<summary> for free keyboard handling
 * and a Tailwind-styled chevron.
 */

import { useState } from "react";
import { BookOpen, ChevronRight } from "lucide-react";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { StatusChip } from "../../ui/status-chip";
import type { OperatingModeCatalogPhase } from "../../../types/operating-mode";
import { SkillViewerDialog } from "./skill-viewer-dialog";

interface PhaseInternalsDisclosureProps {
  phase: OperatingModeCatalogPhase;
  defaultOpen?: boolean;
}

const TOKEN_CHIP_COLORS = {
  background: "bg-slate-800/80",
  border: "border-slate-700",
  text: "text-slate-200",
};

const METRICS_REPLAN_COLORS = {
  background: "bg-cyan-500/10",
  border: "border-cyan-500/30",
  text: "text-cyan-300",
  dot: "bg-cyan-400",
};

const METRICS_ACCEPTANCE_COLORS = {
  background: "bg-violet-500/10",
  border: "border-violet-500/30",
  text: "text-violet-300",
  dot: "bg-violet-400",
};

function TokenRow({ label, value }: { label: string; value: string }) {
  if (!value) return null;
  return (
    <div className="flex flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3">
      <dt className="shrink-0 text-[11px] uppercase tracking-wide text-slate-500 sm:w-32">{label}</dt>
      <dd>
        <code className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-200">
          {value}
        </code>
      </dd>
    </div>
  );
}

export function PhaseInternalsDisclosure({ phase, defaultOpen }: PhaseInternalsDisclosureProps) {
  const hasMetricFlags = phase.samplesReplanRate || phase.samplesAcceptanceRate;
  const hasBindings = (phase.resultBindings?.length ?? 0) > 0;
  const [skillDialogOpen, setSkillDialogOpen] = useState(false);

  return (
    <details
      className="group rounded-md border border-slate-800 bg-slate-900/40"
      open={defaultOpen}
      data-testid={`phase-internals-${phase.phase}`}
    >
      <summary
        className={cn(
          "flex cursor-pointer list-none items-center gap-1.5 px-3 py-2 text-xs font-medium text-slate-300",
          "outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
          "hover:text-slate-100",
        )}
      >
        <ChevronRight
          className="h-3.5 w-3.5 shrink-0 transition-transform group-open:rotate-90"
          aria-hidden
        />
        <span>Internals</span>
      </summary>
      <div className="space-y-3 border-t border-slate-800 px-3 py-2.5 text-xs text-slate-300">
        {phase.trigger && (
          <p className="italic text-slate-400">&ldquo;{phase.trigger}&rdquo;</p>
        )}
        <dl className="space-y-1.5">
          <TokenRow label="Catalog ID" value={phase.catalogId} />
          <SkillTokenRow skillId={phase.skillId} onOpen={() => setSkillDialogOpen(true)} />
          <TokenRow label="Activity purpose" value={phase.activityPurpose} />
          <TokenRow label="Lock purpose" value={phase.lockPurpose} />
        </dl>
        {hasMetricFlags && (
          <div className="flex flex-wrap gap-1.5">
            {phase.samplesReplanRate && (
              <StatusChip label="samples replan rate" colors={METRICS_REPLAN_COLORS} leadingDot />
            )}
            {phase.samplesAcceptanceRate && (
              <StatusChip label="samples acceptance rate" colors={METRICS_ACCEPTANCE_COLORS} leadingDot />
            )}
          </div>
        )}
        {hasBindings && (
          <div>
            <p className="mb-1 text-[11px] uppercase tracking-wide text-slate-500">Result bindings</p>
            <ul className="space-y-1">
              {phase.resultBindings?.map((binding) => (
                <li
                  key={`${binding.kind}-${binding.artifact.path}`}
                  className="flex flex-wrap items-center gap-2"
                >
                  <code className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-200">
                    {binding.artifact.path}
                  </code>
                  <StatusChip label={binding.kind} colors={TOKEN_CHIP_COLORS} />
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
      {skillDialogOpen && (
        <SkillViewerDialog
          isOpen={skillDialogOpen}
          onClose={() => setSkillDialogOpen(false)}
          skillId={phase.skillId}
        />
      )}
    </details>
  );
}

function SkillTokenRow({ skillId, onOpen }: { skillId: string; onOpen: () => void }) {
  if (!skillId) return null;
  return (
    <div className="flex flex-col gap-0.5 sm:flex-row sm:items-center sm:gap-3">
      <dt className="shrink-0 text-[11px] uppercase tracking-wide text-slate-500 sm:w-32">Skill ID</dt>
      <dd>
        <button
          type="button"
          onClick={onOpen}
          className={cn(
            "group/skill inline-flex items-center gap-1.5 rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-200 transition-colors",
            "hover:bg-cyan-500/20 hover:text-cyan-200",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50",
          )}
          data-testid={selectors.initiativeDetails.phaseSkillIdButton}
          title={`Open skill template for ${skillId}`}
        >
          <BookOpen
            className="h-3 w-3 shrink-0 text-slate-500 transition-colors group-hover/skill:text-cyan-300"
            aria-hidden
          />
          <code className="font-mono">{skillId}</code>
        </button>
      </dd>
    </div>
  );
}
