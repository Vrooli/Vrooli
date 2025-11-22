import { AlertTriangle, Loader2, RefreshCw, Sparkles } from "lucide-react";
import type {
  ScenarioDetailResponse,
  DependencyDiffEntry,
  ScenarioDependencyRecord,
} from "../types";
import { OptimizationPanel } from "./OptimizationPanel";
import { DeploymentInsightsPanel } from "./DeploymentInsightsPanel";
import { MetadataGapsPanel } from "./deployment/MetadataGapsPanel";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";

interface ScenarioDetailPanelProps {
  detail: ScenarioDetailResponse | null;
  loading: boolean;
  scanning: boolean;
  onScan: (options?: { apply?: boolean }) => void;
  optimizing: boolean;
  onOptimize: (options?: { apply?: boolean }) => void;
}

function DiffList({ title, entries, tone }: { title: string; entries: DependencyDiffEntry[]; tone: "warning" | "info" }) {
  if (!entries.length) return null;
  const color = tone === "warning" ? "bg-amber-500/20 text-amber-200" : "bg-primary/10 text-primary";
  return (
    <div>
      <p className="mb-1 text-xs uppercase text-muted-foreground">{title}</p>
      <div className="flex flex-wrap gap-1">
        {entries.map((entry) => (
          <Badge key={entry.name} variant="outline" className={color}>
            {entry.name}
          </Badge>
        ))}
      </div>
    </div>
  );
}

function formatTimestamp(timestamp?: string) {
  if (!timestamp) return null;
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return null;
  return date.toLocaleString();
}

function DependencySection({
  title,
  dependencies,
  emptyLabel,
}: {
  title: string;
  dependencies: ScenarioDependencyRecord[];
  emptyLabel: string;
}) {
  return (
    <div className="space-y-2 text-sm">
      <p className="text-xs uppercase text-muted-foreground">{title}</p>
      {dependencies.length === 0 ? (
        <p className="text-xs text-muted-foreground">{emptyLabel}</p>
      ) : (
        <div className="space-y-2">
          {dependencies.map((dependency) => (
            <div
              key={dependency.id ?? `${dependency.dependency_name}-${dependency.access_method}`}
              className="rounded-lg border border-border/40 bg-background/40 p-3"
            >
              <div className="flex items-center justify-between text-sm font-medium">
                <div>{dependency.dependency_name}</div>
                <Badge variant={dependency.required ? "default" : "secondary"} className="text-[10px] uppercase">
                  {dependency.required ? "required" : "optional"}
                </Badge>
              </div>
              {dependency.purpose && <p className="mt-1 text-xs text-muted-foreground">{dependency.purpose}</p>}
              <div className="mt-2 grid gap-1 text-xs text-muted-foreground md:grid-cols-2">
                {dependency.access_method && <span>Access: {dependency.access_method}</span>}
                {typeof dependency.configuration?.found_in_file === "string" && (
                  <span>File: {dependency.configuration?.found_in_file as string}</span>
                )}
                {typeof dependency.configuration?.pattern_type === "string" && (
                  <span>Pattern: {dependency.configuration?.pattern_type as string}</span>
                )}
                {dependency.discovered_at && <span>Detected: {formatTimestamp(dependency.discovered_at)}</span>}
                {dependency.last_verified && <span>Verified: {formatTimestamp(dependency.last_verified)}</span>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function ScenarioDetailPanel({ detail, loading, scanning, optimizing, onScan, onOptimize }: ScenarioDetailPanelProps) {
  if (loading) {
    return (
      <Card className="border border-border/40">
        <CardContent className="flex min-h-[360px] items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </CardContent>
      </Card>
    );
  }

  if (!detail) {
    return (
      <Card className="border border-border/40">
        <CardContent className="flex min-h-[360px] flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
          <AlertTriangle className="h-6 w-6 text-muted-foreground" />
          <p>Select a scenario to view its dependencies.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border border-border/40">
      <CardHeader className="space-y-2">
        <CardTitle className="text-lg">{detail.display_name}</CardTitle>
        <p className="text-sm text-muted-foreground">{detail.description}</p>
        <div className="text-xs text-muted-foreground">
          Last scan: {detail.last_scanned ? new Date(detail.last_scanned).toLocaleString() : "Not yet scanned"}
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex flex-wrap gap-3">
          <Button onClick={() => onScan()} disabled={scanning} className="gap-2">
            {scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            Scan now
          </Button>
          <Button
            variant="secondary"
            onClick={() => onScan({ apply: true })}
            disabled={scanning}
            className="gap-2"
          >
            {scanning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            Scan & apply
          </Button>
          <Button variant="outline" disabled={optimizing} onClick={() => onOptimize()} className="gap-2">
            {optimizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            Optimize
          </Button>
          <Button variant="outline" disabled={optimizing} onClick={() => onOptimize({ apply: true })} className="gap-2">
            {optimizing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            Optimize & apply
          </Button>
        </div>

        <Tabs defaultValue="dependencies" className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="dependencies">Dependencies</TabsTrigger>
            <TabsTrigger value="deployment">Deployment</TabsTrigger>
            <TabsTrigger value="optimization">Optimization</TabsTrigger>
          </TabsList>

          <TabsContent value="dependencies" className="mt-6 space-y-6">
            <div className="grid gap-4 md:grid-cols-2">
              <Card className="border border-border/40 bg-background/40">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Resource drift</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <DiffList title="Missing" entries={detail.resource_diff?.missing ?? []} tone="warning" />
                  <DiffList title="Declared only" entries={detail.resource_diff?.extra ?? []} tone="info" />
                </CardContent>
              </Card>
              <Card className="border border-border/40 bg-background/40">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Scenario drift</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-sm">
                  <DiffList title="Missing" entries={detail.scenario_diff?.missing ?? []} tone="warning" />
                  <DiffList title="Declared only" entries={detail.scenario_diff?.extra ?? []} tone="info" />
                </CardContent>
              </Card>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <DependencySection
                title="Resource dependencies"
                dependencies={detail.stored_dependencies?.resources ?? []}
                emptyLabel="No resources detected yet."
              />
              <DependencySection
                title="Scenario dependencies"
                dependencies={detail.stored_dependencies?.scenarios ?? []}
                emptyLabel="No scenario interactions detected yet."
              />
            </div>
          </TabsContent>

          <TabsContent value="deployment" className="mt-6 space-y-6">
            {detail.deployment_report ? (
              <>
                <DeploymentInsightsPanel report={detail.deployment_report} />
                {detail.deployment_report.metadata_gaps && (
                  <MetadataGapsPanel gaps={detail.deployment_report.metadata_gaps} />
                )}
              </>
            ) : (
              <Card className="border border-border/40 bg-background/40">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Deployment readiness</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-muted-foreground">
                    Run a scan to generate deployment metadata for this scenario.
                  </p>
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="optimization" className="mt-6 space-y-6">
            <OptimizationPanel recommendations={detail.optimization_recommendations ?? []} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  );
}
