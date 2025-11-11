import { AlertTriangle, Loader2, RefreshCw, Sparkles } from "lucide-react";
import type { ScenarioDetailResponse, DependencyDiffEntry } from "../types";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Badge } from "./ui/badge";

interface ScenarioDetailPanelProps {
  detail: ScenarioDetailResponse | null;
  loading: boolean;
  scanning: boolean;
  onScan: (options?: { apply?: boolean }) => void;
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

export function ScenarioDetailPanel({ detail, loading, scanning, onScan }: ScenarioDetailPanelProps) {
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
        </div>

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

        <div className="space-y-2 text-sm">
          <p className="text-xs uppercase text-muted-foreground">Detected scenario dependencies</p>
          <div className="flex flex-wrap gap-1">
            {(detail.stored_dependencies?.scenarios ?? []).map((dependency) => (
              <Badge key={`${dependency.dependency_name}-${dependency.access_method}`} variant="outline">
                {dependency.dependency_name}
              </Badge>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
