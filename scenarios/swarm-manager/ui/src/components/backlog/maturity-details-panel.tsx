import { ArrowRight } from "lucide-react";
import { Card } from "../ui/card";
import type { MaturityIndicatorData, MaturityInput, PhaseState } from "../../lib/maturity";

const PHASE_META: Array<{
  key: "clarify" | "suggest" | "enhance";
  label: string;
  detail: (input: MaturityInput) => string;
}> = [
  {
    key: "clarify",
    label: "Clarify",
    detail: (i) =>
      i.questionsTotal === 0
        ? "No questions yet"
        : `${i.questionsAnswered}/${i.questionsTotal} answered`,
  },
  {
    key: "suggest",
    label: "Suggest",
    detail: (i) =>
      i.suggestionsTotal === 0
        ? "No suggestions yet"
        : `${i.suggestionsDecided}/${i.suggestionsTotal} decided`,
  },
  {
    key: "enhance",
    label: "Enhance",
    detail: (i) => {
      if (!i.hasEnhanceSummary) return "Not yet run";
      const stale = i.questionsNewOrUpdated + i.suggestionsNewOrUpdated;
      if (stale > 0) return `${stale} change${stale === 1 ? "" : "s"} pending`;
      return "Up to date";
    },
  },
];

const BAR_COLORS: Record<PhaseState, string> = {
  empty: "bg-slate-700",
  "in-progress": "bg-amber-500",
  complete: "bg-emerald-500",
};

const TEXT_COLORS: Record<PhaseState, string> = {
  empty: "text-slate-500",
  "in-progress": "text-amber-300",
  complete: "text-emerald-400",
};

interface MaturityDetailsPanelProps {
  maturity: MaturityIndicatorData;
  input: MaturityInput;
}

export function MaturityDetailsPanel({ maturity, input }: MaturityDetailsPanelProps) {
  const allEmpty = PHASE_META.every((p) => maturity.phases[p.key] === "empty");
  if (allEmpty) return null;

  return (
    <Card padding="sm">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-slate-300">Refinement Progress</span>
        {maturity.enhanceRound > 0 && (
          <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] text-slate-400">
            Round {maturity.enhanceRound}
          </span>
        )}
      </div>

      <div className="mt-3 space-y-2">
        {PHASE_META.map((phase) => {
          const state = maturity.phases[phase.key];
          return (
            <div key={phase.key} className="flex items-center gap-3">
              <div className="w-3 flex-shrink-0">
                <div className={`h-1.5 w-full rounded-sm ${BAR_COLORS[state]}`} />
              </div>
              <span className={`w-16 text-xs font-medium ${TEXT_COLORS[state]}`}>
                {phase.label}
              </span>
              <span className="text-xs text-slate-500">
                {phase.detail(input)}
              </span>
            </div>
          );
        })}
      </div>

      {maturity.nextNudge && (
        <div className="mt-3 flex items-start gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/5 px-3 py-2">
          <ArrowRight className="mt-0.5 h-3 w-3 flex-shrink-0 text-cyan-400" />
          <span className="text-xs text-cyan-300">{maturity.nextNudge}</span>
        </div>
      )}
    </Card>
  );
}
