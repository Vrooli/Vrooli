import { Activity, AlertTriangle, CheckCircle, Server } from "lucide-react";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Separator } from "./ui/separator";
import { ScrollArea } from "./ui/scroll-area";
import type { GraphStats } from "../hooks/useGraphData";
import type { DependencyGraphNode } from "../types";

interface StatsPanelProps {
  stats: GraphStats | null;
  apiHealthy: boolean | null;
  influentialNodes: Array<{ node: DependencyGraphNode; score: number }>;
}

export function StatsPanel({ stats, apiHealthy, influentialNodes }: StatsPanelProps) {
  return (
    <Card className="glass border border-border/40">
      <CardHeader className="space-y-1">
        <CardTitle className="flex items-center gap-2 text-base font-semibold">
          <Activity className="h-4 w-4 text-primary" /> System Telemetry
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid grid-cols-2 gap-3">
          <Metric label="Nodes" value={stats?.totalNodes ?? 0} />
          <Metric label="Edges" value={stats?.totalEdges ?? 0} />
          <Metric label="Scenarios" value={stats?.scenarioCount ?? 0} />
          <Metric label="Resources" value={stats?.resourceCount ?? 0} />
        </div>

        <div className="flex items-center gap-3 rounded-lg border border-border/40 bg-background/60 px-3 py-2 text-sm">
          <Server className="h-4 w-4" aria-hidden="true" />
          <div>
            <p className="font-medium">API Health</p>
            <p className="text-xs text-muted-foreground">Real-time availability signal</p>
          </div>
          <Badge
            className="ml-auto"
            variant={apiHealthy ? "default" : apiHealthy === null ? "secondary" : "outline"}
          >
            {apiHealthy === null ? "Checking" : apiHealthy ? "Healthy" : "Degraded"}
          </Badge>
        </div>

        <div className="space-y-2">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Graph Complexity
          </p>
          <div className="flex items-center gap-3 rounded-lg border border-border/40 bg-background/60 p-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
              <CheckCircle className="h-5 w-5 text-primary" aria-hidden="true" />
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Composite complexity score</p>
              <p className="text-lg font-semibold text-foreground">
                {stats?.complexityScore ?? "0"}
              </p>
            </div>
          </div>
        </div>

        <Separator className="bg-border/30" />

        <div className="space-y-2">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Highest Influence Nodes
          </p>
          {influentialNodes.length === 0 ? (
            <div className="flex items-center gap-3 rounded-lg border border-border/50 bg-background/60 p-3 text-xs text-muted-foreground">
              <AlertTriangle className="h-4 w-4" aria-hidden="true" />
              Awaiting data — run an analysis to populate insights.
            </div>
          ) : (
            <ScrollArea className="max-h-[220px] rounded-lg border border-border/40 bg-background/60">
              <ul className="divide-y divide-border/20">
                {influentialNodes.map(({ node, score }) => (
                  <li key={node.id} className="flex items-center gap-3 p-3 text-sm">
                    <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                      {node.type.slice(0, 1).toUpperCase()}
                    </div>
                    <div className="flex-1">
                      <p className="font-medium text-foreground">{node.label}</p>
                      <p className="text-xs text-muted-foreground">Impact score {score.toFixed(1)}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </ScrollArea>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-border/50 bg-background/60 p-3 text-left">
      <p className="text-xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold text-foreground">{value}</p>
    </div>
  );
}
