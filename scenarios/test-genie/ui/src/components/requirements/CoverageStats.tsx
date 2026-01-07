import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import { InfoTip } from "../cards/InfoTip";
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
      color: "text-white",
      help: {
        title: "Coverage Rate",
        description:
          "Percentage of requirements with declared status 'complete'. This tracks implementation progress independent of test results."
      }
    },
    {
      label: "Passed",
      value: passed,
      color: "text-emerald-400",
      help: {
        title: "Passed Requirements",
        description:
          "Requirements that are marked 'complete' AND have all validations passing. Both conditions must be met."
      }
    },
    {
      label: "Failed",
      value: failed,
      color: failed > 0 ? "text-red-400" : "text-slate-400",
      help: {
        title: "Failed Requirements",
        description:
          "Requirements where ANY validation has a 'failing' status, regardless of the declared status."
      }
    },
    {
      label: "Not Run",
      value: notRun,
      color: notRun > 0 ? "text-amber-400" : "text-slate-400",
      help: {
        title: "Not Run Requirements",
        description:
          "Requirements that are 'pending' or 'in_progress', or have no validations defined yet."
      }
    }
  ];

  return (
    <div
      className={cn("grid grid-cols-4 gap-3", className)}
      data-testid={selectors.requirements.coverageStats}
    >
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="rounded-xl border border-white/5 bg-black/30 p-3 text-center"
        >
          <div className="flex items-center justify-center gap-1">
            <p className={cn("text-2xl font-semibold", stat.color)}>
              {stat.value}
            </p>
            <InfoTip title={stat.help.title} description={stat.help.description} />
          </div>
          <p className="mt-1 text-xs text-slate-400">{stat.label}</p>
        </div>
      ))}
    </div>
  );
}
