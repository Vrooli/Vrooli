import type { MaturityIndicatorData, PhaseState } from "../../lib/maturity";

const PHASE_LABELS = { clarify: "C", suggest: "S", enhance: "E" } as const;
const PHASE_TOOLTIPS = { clarify: "Clarify", suggest: "Suggest", enhance: "Enhance" } as const;

const PHASE_COLORS: Record<PhaseState, string> = {
  empty: "bg-slate-700",
  "in-progress": "bg-amber-500",
  complete: "bg-emerald-500",
};

interface MaturityPhaseBarProps {
  maturity: MaturityIndicatorData;
  className?: string;
}

export function MaturityPhaseBar({ maturity, className }: MaturityPhaseBarProps) {
  const phases = ["clarify", "suggest", "enhance"] as const;
  const allEmpty = phases.every((p) => maturity.phases[p] === "empty");
  if (allEmpty) return null;

  return (
    <div className={className}>
      <div className="flex items-center gap-1.5">
        <div className="flex gap-0.5" title="Refinement: Clarify → Suggest → Enhance">
          {phases.map((phase) => (
            <div
              key={phase}
              className={`h-1.5 w-5 rounded-sm ${PHASE_COLORS[maturity.phases[phase]]}`}
              title={`${PHASE_TOOLTIPS[phase]}: ${maturity.phases[phase]}`}
            />
          ))}
        </div>
        {maturity.enhanceRound > 0 && (
          <span className="text-[10px] text-slate-500">R{maturity.enhanceRound}</span>
        )}
        {maturity.needsResynthesis > 0 && (
          <span className="text-[10px] text-amber-400">
            {maturity.needsResynthesis} unsynthesized
          </span>
        )}
      </div>
    </div>
  );
}
