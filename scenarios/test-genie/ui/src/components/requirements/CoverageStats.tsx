import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { RequirementsCoverage } from "../../hooks/useRequirements";

interface CoverageStatsProps {
  coverage: RequirementsCoverage;
  className?: string;
}

export function CoverageStats({ coverage, className }: CoverageStatsProps) {
  const { total, passed, failed, notRun, completionRate } = coverage;

  const stats = [
    {
      label: "Coverage",
      value: total > 0 ? `${Math.round(completionRate)}%` : "—",
      color: "text-white"
    },
    {
      label: "Passed",
      value: passed,
      color: "text-emerald-400"
    },
    {
      label: "Failed",
      value: failed,
      color: failed > 0 ? "text-red-400" : "text-slate-400"
    },
    {
      label: "Not Run",
      value: notRun,
      color: notRun > 0 ? "text-amber-400" : "text-slate-400"
    }
  ];

  return (
    <div className={cn("grid grid-cols-4 gap-3", className)} data-testid={selectors.requirements.coverageStats}>
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="rounded-xl border border-white/5 bg-black/30 p-3 text-center"
        >
          <p className={cn("text-2xl font-semibold", stat.color)}>
            {stat.value}
          </p>
          <p className="mt-1 text-xs text-slate-400">{stat.label}</p>
        </div>
      ))}
    </div>
  );
}
