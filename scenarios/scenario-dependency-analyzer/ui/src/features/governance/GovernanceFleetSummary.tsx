import { AlertTriangle, CheckCircle2, CircleSlash, Files, ShieldAlert } from "lucide-react";

import type { DependencyGovernanceSummary } from "../../api/governance";
import { selectors } from "../../consts/selectors";
import { prettyNumber } from "../../lib/utils";
import { DecisionStatusBadge } from "./DecisionStatusBadge";

const metrics = [
  { key: "scenarioCount", label: "Scenarios", icon: Files },
  { key: "dependencyCount", label: "Dependency groups", icon: CheckCircle2 },
  { key: "findingCount", label: "Findings", icon: AlertTriangle },
  { key: "errorCount", label: "Errors", icon: CircleSlash },
  { key: "warningCount", label: "Warnings", icon: ShieldAlert },
  { key: "unrecorded", label: "Need review", icon: AlertTriangle }
] as const;

export function GovernanceFleetSummary({
  summary,
  passed,
  guidance
}: {
  summary: DependencyGovernanceSummary | null | undefined;
  passed: boolean;
  guidance?: string;
}) {
  return (
    <section
      className="rounded-lg border border-border/50 bg-card/70 p-4 shadow-lg shadow-black/10"
      data-testid={selectors.governance.summary}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-xl font-semibold text-foreground">Dependency Governance</h1>
            <DecisionStatusBadge value={summary?.status || (passed ? "pass" : "warn")} />
          </div>
          <p className="mt-1 max-w-3xl text-sm text-muted-foreground">
            Fleet validation for approved, denied, scoped, expired, unrecorded, and vulnerable dependency decisions.
          </p>
        </div>
        <DecisionStatusBadge value={summary?.policyMode || "advisory"} />
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        {metrics.map(({ key, label, icon: Icon }) => (
          <div key={key} className="rounded-md border border-border/40 bg-background/35 p-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Icon className="h-4 w-4" aria-hidden="true" />
              {label}
            </div>
            <p className="mt-2 text-2xl font-semibold text-foreground">
              {prettyNumber(summary?.[key] ?? 0)}
            </p>
          </div>
        ))}
      </div>

      {guidance ? <p className="mt-3 text-xs leading-5 text-muted-foreground">{guidance}</p> : null}
    </section>
  );
}
