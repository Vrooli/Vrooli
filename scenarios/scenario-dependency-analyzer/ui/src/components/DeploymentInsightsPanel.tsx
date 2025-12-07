import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import type { DeploymentAnalysisReport, DeploymentDependencyNode, DeploymentTierAggregate } from "../types";

interface DeploymentInsightsPanelProps {
  report: DeploymentAnalysisReport;
}

const formatRequirement = (value?: number, label?: string) => {
  if (typeof value !== "number" || Number.isNaN(value)) return null;
  const rounded = Math.round(value * 10) / 10;
  return `${rounded}${label ?? ""}`;
};

const getBlockers = (aggregates?: Record<string, DeploymentTierAggregate>) => {
  if (!aggregates) return [];
  const blockers = new Set<string>();
  Object.values(aggregates).forEach((aggregate) => {
    aggregate.blocking_dependencies?.forEach((dep) => blockers.add(dep));
  });
  return Array.from(blockers);
};

const buildCatalogSummary = (nodes: DeploymentDependencyNode[]) => {
  const resourceDependencies = nodes.filter((node) => node.type === "resource");
  const scenarioDependencies = nodes.filter((node) => node.type === "scenario");
  return {
    resourceDependencies,
    scenarioDependencies
  };
};

export function DeploymentInsightsPanel({ report }: DeploymentInsightsPanelProps) {
  const tierEntries = Object.entries(report.aggregates ?? {}).sort(([a], [b]) => a.localeCompare(b));
  const blockers = getBlockers(report.aggregates);
  const manifest = report.bundle_manifest;
  const catalogSummary = buildCatalogSummary(report.dependencies ?? []);

  return (
    <div className="space-y-4">
      <Card className="border border-border/40 bg-background/40">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">Tier readiness</CardTitle>
          <p className="text-xs text-muted-foreground">Fitness and blockers per deployment tier</p>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          {tierEntries.length === 0 ? (
            <p className="text-xs text-muted-foreground">No tier metadata discovered yet.</p>
          ) : (
            tierEntries.map(([tierName, aggregate]) => (
              <div key={tierName} className="rounded-lg border border-border/40 bg-background/80 p-3">
                <div className="flex items-center justify-between">
                  <div className="font-medium text-foreground">{tierName}</div>
                  <Badge variant="outline" className="text-[10px] uppercase">
                    {Number.isFinite(aggregate.fitness_score)
                      ? `${Math.round(aggregate.fitness_score * 100)}%`
                      : "n/a"}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">
                  Dependencies: {aggregate.dependency_count} · Blockers:{" "}
                  {aggregate.blocking_dependencies && aggregate.blocking_dependencies.length > 0
                    ? aggregate.blocking_dependencies.join(", ")
                    : "none"}
                </p>
                <p className="text-xs text-muted-foreground">
                  RAM {formatRequirement(aggregate.estimated_requirements?.ram_mb, " MB") ?? "n/a"} · Disk{" "}
                  {formatRequirement(aggregate.estimated_requirements?.disk_mb, " MB") ?? "n/a"} · CPU{" "}
                  {formatRequirement(aggregate.estimated_requirements?.cpu_cores, " cores") ?? "n/a"}
                </p>
              </div>
            ))
          )}
          <div className="pt-1 text-xs text-muted-foreground">
            Generated {new Date(report.generated_at).toLocaleString()}
          </div>
        </CardContent>
      </Card>

      <Card className="border border-border/40 bg-background/40">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">Bundle manifest</CardTitle>
          <p className="text-xs text-muted-foreground">Files and dependencies required for packaging</p>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div>
            <p className="text-xs uppercase text-muted-foreground">Key files</p>
            {manifest && manifest.files.length > 0 ? (
              <ul className="mt-1 space-y-1">
                {manifest.files.slice(0, 5).map((file) => (
                  <li key={file.path} className="flex items-center justify-between rounded border border-border/20 px-2 py-1">
                    <span className="truncate text-xs">
                      {file.type}: {file.path}
                    </span>
                    <Badge variant={file.exists ? "secondary" : "outline"} className="text-[10px] uppercase">
                      {file.exists ? "present" : "missing"}
                    </Badge>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-xs text-muted-foreground">No bundle files recorded.</p>
            )}
          </div>

          <div>
            <p className="text-xs uppercase text-muted-foreground">Critical dependencies</p>
            {catalogSummary.resourceDependencies.length + catalogSummary.scenarioDependencies.length === 0 ? (
              <p className="text-xs text-muted-foreground">No dependencies recorded.</p>
            ) : (
              <div className="mt-1 grid gap-2 text-xs md:grid-cols-2">
                <div>
                  <p className="pb-1 text-[10px] uppercase tracking-wide text-muted-foreground">Resources</p>
                  <ul className="space-y-1">
                    {catalogSummary.resourceDependencies.slice(0, 4).map((node) => (
                      <li key={`resource-${node.name}`} className="rounded border border-border/20 px-2 py-1">
                        {node.name}
                      </li>
                    ))}
                  </ul>
                </div>
                <div>
                  <p className="pb-1 text-[10px] uppercase tracking-wide text-muted-foreground">Scenarios</p>
                  <ul className="space-y-1">
                    {catalogSummary.scenarioDependencies.slice(0, 4).map((node) => (
                      <li key={`scenario-${node.name}`} className="rounded border border-border/20 px-2 py-1">
                        {node.name}
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            )}
          </div>

          <div>
            <p className="text-xs uppercase text-muted-foreground">Blockers</p>
            {blockers.length === 0 ? (
              <p className="text-xs text-muted-foreground">No blocking dependencies detected.</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {blockers.map((blocker) => (
                  <Badge
                    key={blocker}
                    variant="secondary"
                    className="border-red-500/50 bg-red-500/10 text-[10px] uppercase text-red-100"
                  >
                    {blocker}
                  </Badge>
                ))}
              </div>
            )}
          </div>
          <div className="rounded-md border border-border/30 bg-background/70 p-3 text-xs text-muted-foreground">
            <p className="font-semibold text-foreground">Next step: export a bundle</p>
            <p className="mt-1">
              Run deployment-manager to export <code>bundle.json</code> from this analysis. Start it with{" "}
              <code className="rounded bg-black/30 px-1 py-0.5 text-[11px]">vrooli scenario start deployment-manager</code>, then use the bundle export flow
              (or <code>/api/v1/bundles/export</code>) to hand off to scenario-to-desktop.
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              <a
                href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/scenarios/scenario-to-desktop.md"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center rounded border border-border/40 px-2 py-1 text-[11px] font-medium text-foreground hover:border-border"
              >
                Hand off to scenario-to-desktop
              </a>
              <span className="inline-flex items-center rounded border border-border/40 px-2 py-1 text-[11px] font-medium text-foreground">
                Run <code className="mx-1 rounded bg-black/30 px-1 py-0.5 text-[11px]">vrooli scenario port deployment-manager UI_PORT</code> and open that URL.
              </span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
