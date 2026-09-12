import { ExternalLink } from "lucide-react";

import type { DependencyUsageGroup } from "../../api/governance";
import { Button } from "../../components/ui/button";
import { DecisionStatusBadge } from "./DecisionStatusBadge";

export function DependencyUsagePanel({
  group,
  onStartDecision,
  onStartRemediation
}: {
  group: DependencyUsageGroup | null;
  onStartDecision: (group: DependencyUsageGroup, state: "approved" | "denied" | "deprecated" | "needs_review") => void;
  onStartRemediation: (group: DependencyUsageGroup) => void;
}) {
  if (!group) {
    return (
      <aside className="rounded-lg border border-border/50 bg-card/70 p-4">
        <h2 className="text-sm font-semibold">Dependency details</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Select a dependency to inspect usage, scenarios, and governance actions.
        </p>
      </aside>
    );
  }

  const cli = `scenario-dependency-analyzer deps approved usage ${group.ecosystem}/${group.packageName} --json`;

  return (
    <aside className="rounded-lg border border-border/50 bg-card/70 p-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <p className="text-xs uppercase text-muted-foreground">{group.ecosystem}</p>
          <h2 className="break-words text-lg font-semibold">{group.packageName}</h2>
        </div>
        <DecisionStatusBadge value={group.state || group.highestSeverity || "info"} />
      </div>

      <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
        <div>
          <dt className="text-xs text-muted-foreground">Scenarios</dt>
          <dd className="font-medium">{group.scenarioCount}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Usages</dt>
          <dd className="font-medium">{group.usageCount}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Findings</dt>
          <dd className="font-medium">{group.findingCount}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted-foreground">Highest severity</dt>
          <dd className="font-medium">{group.highestSeverity || "info"}</dd>
        </div>
      </dl>

      <div className="mt-4 flex flex-wrap gap-2">
        <Button size="sm" onClick={() => onStartDecision(group, "approved")}>Approve</Button>
        <Button size="sm" variant="outline" onClick={() => onStartDecision(group, "denied")}>Deny</Button>
        <Button size="sm" variant="secondary" onClick={() => onStartRemediation(group)}>
          Review vulnerability
        </Button>
      </div>

      <div className="mt-4">
        <p className="text-xs font-medium text-muted-foreground">Scenarios using this dependency</p>
        <div className="mt-2 flex max-h-28 flex-wrap gap-2 overflow-auto">
          {group.scenarios.map((scenario) => (
            <span key={scenario} className="rounded-md border border-border/40 bg-background/40 px-2 py-1 text-xs">
              {scenario}
            </span>
          ))}
        </div>
      </div>

      <div className="mt-4 rounded-md border border-border/40 bg-background/35 p-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
          CLI
        </div>
        <code className="mt-1 block break-all text-xs text-foreground">{cli}</code>
      </div>
    </aside>
  );
}
