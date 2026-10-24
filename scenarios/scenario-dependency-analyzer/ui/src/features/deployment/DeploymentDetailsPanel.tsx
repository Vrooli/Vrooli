import { AlertTriangle, Clock, Download, ExternalLink, Info } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { statusTone } from "../../theme/status";
import type { ScenarioDeploymentStatus, DeploymentTierOption } from "./deploymentStatus";
import { MetadataGapsPanel } from "./MetadataGapsPanel";

interface DeploymentDetailsPanelProps {
  onExportDag: (scenarioName: string) => void;
  onOpenCatalog: (scenarioName: string) => void;
  onScan: (scenarioName: string, apply?: boolean) => void;
  selectedScenario: ScenarioDeploymentStatus;
  targetTier: string;
  tierOptions: DeploymentTierOption[];
}

export function DeploymentDetailsPanel({
  onExportDag,
  onOpenCatalog,
  onScan,
  selectedScenario,
  targetTier,
  tierOptions
}: DeploymentDetailsPanelProps) {
  const selectedTierFitness = selectedScenario.lastReport?.aggregates?.[targetTier]?.fitness_score;
  const selectedTierBlockers = selectedScenario.lastReport?.aggregates?.[targetTier]?.blocking_dependencies || [];
  const selectedTierRequirements = selectedScenario.lastReport?.aggregates?.[targetTier]?.estimated_requirements;
  const selectedScenarioGaps = selectedScenario.lastReport?.metadata_gaps?.gaps_by_scenario?.[selectedScenario.scenario.name];
  const tierLabel = tierOptions.find((t) => t.value === targetTier)?.label;

  return (
    <Card className="border border-border/50 bg-background/50">
      <CardHeader className="flex flex-wrap items-center justify-between gap-3 pb-3">
        <div>
          <CardTitle className="text-base">
            Deployment details: {selectedScenario.scenario.display_name || selectedScenario.scenario.name}
          </CardTitle>
          <p className="text-xs text-muted-foreground">
            Focused on {tierLabel}. Tips below explain each field.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="outline" className="gap-2" onClick={() => onExportDag(selectedScenario.scenario.name)}>
            <Download className="h-4 w-4" />
            Export DAG JSON
          </Button>
          <Button
            size="sm"
            variant="outline"
            className="gap-2"
            onClick={() => window.open("/docs/deployment/guides/deployment-checklist.md", "_blank")}
          >
            <Info className="h-4 w-4" />
            Deployment checklist
          </Button>
          <Button
            size="sm"
            variant="secondary"
            className="gap-2"
            onClick={() => window.open("/docs/deployment/tiers/tier-2-desktop.md", "_blank")}
          >
            <ExternalLink className="h-4 w-4" />
            Tier guide
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <div className="rounded border border-border/40 bg-background/40 p-3">
            <p className="flex items-center gap-1 text-xs uppercase tracking-wide text-muted-foreground">
              <Info className="h-3.5 w-3.5" aria-hidden="true" />
              Fitness score
            </p>
            <p className="mt-1 text-2xl font-semibold">
              {selectedTierFitness !== undefined ? `${Math.round((selectedTierFitness || 0) * 100)}%` : "N/A"}
            </p>
            <p className="text-[11px] text-muted-foreground">
              Higher is better. <strong>Tip:</strong> below 70% usually means you need swaps (e.g., Postgres → SQLite) or more metadata.
            </p>
          </div>
          <div className="rounded border border-border/40 bg-background/40 p-3">
            <p className="flex items-center gap-1 text-xs uppercase tracking-wide text-muted-foreground">
              <AlertTriangle className={`h-3.5 w-3.5 ${statusTone("warning").text}`} aria-hidden="true" />
              Blockers ({targetTier})
            </p>
            <p className={`mt-1 text-2xl font-semibold ${statusTone("warning").text}`}>{selectedTierBlockers.length}</p>
            <p className="text-[11px] text-muted-foreground">
              These dependencies cannot run on this tier. Fix by adding alternatives in <code>deployment.dependencies</code> or swapping resources.
            </p>
          </div>
          <div className="rounded border border-border/40 bg-background/40 p-3">
            <p className="flex items-center gap-1 text-xs uppercase tracking-wide text-muted-foreground">
              <Clock className="h-3.5 w-3.5" aria-hidden="true" />
              Last scanned
            </p>
            <p className="mt-1 text-2xl font-semibold">
              {selectedScenario.scenario.last_scanned
                ? new Date(selectedScenario.scenario.last_scanned).toLocaleString()
                : "Never"}
            </p>
            <p className="text-[11px] text-muted-foreground">
              <strong>Tip:</strong> Re-scan after adding metadata or changing dependencies.
            </p>
          </div>
        </div>

        <div className="grid gap-3 md:grid-cols-2">
          <div className="space-y-2 rounded border border-border/40 bg-background/40 p-3">
            <div className="flex items-center gap-2">
              <Info className="h-4 w-4 text-primary" aria-hidden="true" />
              <p className="text-sm font-medium text-foreground">Requirements estimate</p>
            </div>
            {selectedTierRequirements ? (
              <ul className="space-y-1 text-xs text-muted-foreground">
                <li>RAM: {selectedTierRequirements.ram_mb} MB</li>
                <li>Disk: {selectedTierRequirements.disk_mb} MB</li>
                <li>CPU cores: {selectedTierRequirements.cpu_cores}</li>
                <li className="text-[11px] text-muted-foreground">
                  Use these when sizing desktop bundles; lower numbers are better for portability.
                </li>
              </ul>
            ) : (
              <p className="text-xs text-muted-foreground">
                No estimates yet. Add requirements to the tier metadata in <code>.vrooli/service.json</code> or re-run Scan &amp; Apply.
              </p>
            )}
          </div>

          <div className="space-y-2 rounded border border-border/40 bg-background/40 p-3">
            <div className="flex items-center gap-2">
              <AlertTriangle className={`h-4 w-4 ${statusTone("warning").text}`} aria-hidden="true" />
              <p className="text-sm font-medium text-foreground">Blocking dependencies</p>
            </div>
            {selectedTierBlockers.length > 0 ? (
              <ul className="list-inside list-disc space-y-1 text-xs text-muted-foreground">
                {selectedTierBlockers.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            ) : (
              <p className="text-xs text-muted-foreground">No blockers detected for this tier. Move on to packaging.</p>
            )}
            <p className="text-[11px] text-muted-foreground">
              <strong>How to fix:</strong> Add <code>alternatives</code> or mark unsupported tiers in <code>deployment.dependencies</code>, then rerun Scan &amp; Apply.
            </p>
          </div>
        </div>

        <div className="space-y-2 rounded border border-border/40 bg-background/40 p-3">
          <div className="flex items-center gap-2">
            <Info className="h-4 w-4 text-primary" aria-hidden="true" />
            <p className="text-sm font-medium text-foreground">Metadata gaps for this scenario</p>
          </div>
          {selectedScenarioGaps && selectedScenario.lastReport?.metadata_gaps ? (
            <div className="space-y-2 text-xs text-muted-foreground">
              <p className="text-[11px] text-muted-foreground">
                Fill these fields in <code>.vrooli/service.json</code>. Use the docs and the “Scan &amp; Apply” button to re-check.
              </p>
              <MetadataGapsPanel
                gaps={{
                  total_gaps: selectedScenario.lastReport.metadata_gaps.total_gaps,
                  scenarios_missing_all: selectedScenario.lastReport.metadata_gaps.scenarios_missing_all,
                  gaps_by_scenario: {
                    [selectedScenario.scenario.name]: selectedScenarioGaps
                  }
                }}
              />
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">No metadata gaps reported for this scenario.</p>
          )}
        </div>

        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="secondary" onClick={() => onScan(selectedScenario.scenario.name, false)}>
            Re-scan
          </Button>
          <Button size="sm" variant="secondary" onClick={() => onScan(selectedScenario.scenario.name, true)}>
            Scan & Apply (write metadata)
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => window.open("/docs/deployment/guides/dependency-swapping.md", "_blank")}
          >
            Read: Dependency swapping
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => onOpenCatalog(selectedScenario.scenario.name)}
          >
            Open in catalog view
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
