import type { ApprovedDependencyFinding, DependencyUsageGroup } from "../../api/governance";
import { selectors } from "../../consts/selectors";
import { DecisionStatusBadge } from "./DecisionStatusBadge";

export function GovernanceFindingsTable({
  findings,
  onSelectDependency
}: {
  findings: ApprovedDependencyFinding[];
  onSelectDependency: (ecosystem: string, packageName: string) => void;
}) {
  if (findings.length === 0) {
    return <EmptyState title="No findings match the current filters." />;
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border/50 bg-card/70">
      <div className="hidden overflow-auto lg:block">
        <table className="w-full min-w-[900px] text-left text-sm" aria-label="Governance findings">
          <thead className="border-b border-border/50 bg-background/45 text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2 font-medium">Finding</th>
              <th className="px-3 py-2 font-medium">Dependency</th>
              <th className="px-3 py-2 font-medium">Scenario</th>
              <th className="px-3 py-2 font-medium">Observed</th>
              <th className="px-3 py-2 font-medium">Expected</th>
            </tr>
          </thead>
          <tbody>
            {findings.map((finding) => (
              <tr
                key={finding.id}
                className="border-b border-border/30 last:border-0"
                data-testid={selectors.governance.findingRow(finding.id || `${finding.ecosystem}-${finding.packageName}`)}
              >
                <td className="px-3 py-3 align-top">
                  <div className="flex flex-wrap items-center gap-2">
                    <DecisionStatusBadge value={finding.severity || "info"} />
                    <span className="font-medium">{finding.title || finding.findingClass}</span>
                  </div>
                  <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{finding.description}</p>
                </td>
                <td className="px-3 py-3 align-top">
                  <button
                    className="break-all text-left font-medium text-primary hover:underline"
                    onClick={() => onSelectDependency(finding.ecosystem, finding.packageName)}
                    type="button"
                  >
                    {finding.ecosystem}/{finding.packageName}
                  </button>
                  <p className="mt-1 text-xs text-muted-foreground">{finding.findingClass}</p>
                </td>
                <td className="px-3 py-3 align-top">{finding.scenario || "-"}</td>
                <td className="px-3 py-3 align-top">
                  <code className="break-all text-xs">{finding.observed || "-"}</code>
                </td>
                <td className="px-3 py-3 align-top">
                  <code className="break-all text-xs">{finding.expected || "-"}</code>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="divide-y divide-border/40 lg:hidden">
        {findings.map((finding) => (
          <button
            key={finding.id}
            className="block w-full p-3 text-left"
            onClick={() => onSelectDependency(finding.ecosystem, finding.packageName)}
            type="button"
          >
            <div className="flex flex-wrap items-center gap-2">
              <DecisionStatusBadge value={finding.severity || "info"} />
              <span className="font-medium">{finding.title || finding.findingClass}</span>
            </div>
            <p className="mt-1 break-all text-sm">{finding.ecosystem}/{finding.packageName}</p>
            <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{finding.description}</p>
          </button>
        ))}
      </div>
    </div>
  );
}

export function GovernanceDependenciesTable({
  groups,
  selectedKey,
  onSelect
}: {
  groups: DependencyUsageGroup[];
  selectedKey: string | null;
  onSelect: (group: DependencyUsageGroup) => void;
}) {
  if (groups.length === 0) {
    return <EmptyState title="No dependencies match the current filters." />;
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border/50 bg-card/70">
      <div className="hidden overflow-auto md:block">
        <table className="w-full min-w-[720px] text-left text-sm" aria-label="Dependency usage groups">
          <thead className="border-b border-border/50 bg-background/45 text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-3 py-2 font-medium">Dependency</th>
              <th className="px-3 py-2 font-medium">State</th>
              <th className="px-3 py-2 text-right font-medium">Scenarios</th>
              <th className="px-3 py-2 text-right font-medium">Usages</th>
              <th className="px-3 py-2 text-right font-medium">Findings</th>
            </tr>
          </thead>
          <tbody>
            {groups.map((group) => {
              const key = `${group.ecosystem}/${group.packageName}`;
              return (
                <tr
                  key={key}
                  className={selectedKey === key ? "bg-primary/10" : "border-b border-border/30 last:border-0"}
                  data-testid={selectors.governance.dependencyRow(group.ecosystem, group.packageName)}
                >
                  <td className="px-3 py-3">
                    <button className="break-all text-left font-medium text-primary hover:underline" onClick={() => onSelect(group)} type="button">
                      {key}
                    </button>
                  </td>
                  <td className="px-3 py-3"><DecisionStatusBadge value={group.state || group.highestSeverity || "info"} /></td>
                  <td className="px-3 py-3 text-right">{group.scenarioCount}</td>
                  <td className="px-3 py-3 text-right">{group.usageCount}</td>
                  <td className="px-3 py-3 text-right">{group.findingCount}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="divide-y divide-border/40 md:hidden">
        {groups.map((group) => {
          const key = `${group.ecosystem}/${group.packageName}`;
          return (
            <button key={key} className="block w-full p-3 text-left" onClick={() => onSelect(group)} type="button">
              <div className="flex items-start justify-between gap-3">
                <span className="break-all font-medium">{key}</span>
                <DecisionStatusBadge value={group.state || group.highestSeverity || "info"} />
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                {group.scenarioCount} scenarios, {group.usageCount} usages, {group.findingCount} findings
              </p>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function EmptyState({ title }: { title: string }) {
  return (
    <div className="rounded-lg border border-border/50 bg-card/70 p-8 text-center text-sm text-muted-foreground">
      {title}
    </div>
  );
}
