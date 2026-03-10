import type { ReactNode } from "react";

interface ConfigPanelProps {
  expanded: boolean;
  onToggle: () => void;
  selectedRuleCount: number;
  totalRuleCount: number;
  allScenarios: boolean;
  selectedScenarioCount: number;
  runControls: ReactNode;
  children: ReactNode;
}

export function ConfigPanel({
  expanded,
  onToggle,
  selectedRuleCount,
  totalRuleCount,
  allScenarios,
  selectedScenarioCount,
  runControls,
  children
}: ConfigPanelProps) {
  const scenarioLabel = allScenarios
    ? "All scenarios"
    : `${selectedScenarioCount} scenario${selectedScenarioCount !== 1 ? "s" : ""} selected`;

  const ruleLabel = `${selectedRuleCount} of ${totalRuleCount} rules selected`;

  return (
    <section className="mt-8">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between rounded-xl border border-white/10 bg-white/5 px-4 py-3 backdrop-blur">
        <button
          type="button"
          onClick={onToggle}
          className="flex items-center gap-2 text-sm text-slate-300 hover:text-slate-100 transition-colors"
        >
          <span className="text-xs text-slate-400">{expanded ? "▾" : "▸"}</span>
          <span>{ruleLabel} · {scenarioLabel}</span>
        </button>
        <div className="flex-shrink-0">{runControls}</div>
      </div>

      {expanded && <div className="mt-4 grid gap-6 lg:grid-cols-[280px_1fr]">{children}</div>}
    </section>
  );
}
