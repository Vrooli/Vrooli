import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Filter,
  HelpCircle,
  Monitor,
  RefreshCw,
  Search,
  Upload,
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { Badge } from "../components/ui/badge";
import { Tip } from "../components/ui/tip";
import { TelemetryEntry } from "../components/TelemetryEntry";
import { listTelemetry, uploadTelemetry, type TelemetrySummary } from "../lib/api";

/** Known failure event types from the desktop runtime */
const FAILURE_EVENT_TYPES = [
  "dependency_unreachable",
  "swap_missing_asset",
  "asset_missing",
  "migration_failed",
  "api_unreachable",
  "secrets_missing",
  "health_failed",
] as const;

/** Aggregate stats computed from all telemetry entries */
interface AggregateStats {
  totalBundles: number;
  totalEvents: number;
  totalFailures: number;
  healthyBundles: number;
  failedBundles: number;
  failuresByType: Record<string, number>;
}

function computeAggregateStats(entries: TelemetrySummary[]): AggregateStats {
  const stats: AggregateStats = {
    totalBundles: entries.length,
    totalEvents: 0,
    totalFailures: 0,
    healthyBundles: 0,
    failedBundles: 0,
    failuresByType: {},
  };

  for (const entry of entries) {
    stats.totalEvents += entry.total_events ?? 0;

    const failureCounts = entry.failure_counts ?? {};
    let bundleHasFailures = false;

    for (const [eventType, count] of Object.entries(failureCounts)) {
      if (count && count > 0) {
        stats.totalFailures += count;
        stats.failuresByType[eventType] = (stats.failuresByType[eventType] ?? 0) + count;
        bundleHasFailures = true;
      }
    }

    if (bundleHasFailures) {
      stats.failedBundles++;
    } else {
      stats.healthyBundles++;
    }
  }

  return stats;
}

function StatCard({
  label,
  value,
  icon: Icon,
  variant = "default",
}: {
  label: string;
  value: string | number;
  icon: React.ElementType;
  variant?: "default" | "success" | "danger";
}) {
  const colorClasses = {
    default: "text-cyan-400 bg-cyan-500/10 border-cyan-500/20",
    success: "text-emerald-400 bg-emerald-500/10 border-emerald-500/20",
    danger: "text-red-400 bg-red-500/10 border-red-500/20",
  };

  return (
    <div className={`rounded-lg border p-4 ${colorClasses[variant]}`}>
      <div className="flex items-center gap-2 text-sm font-medium opacity-80">
        <Icon className="h-4 w-4" />
        {label}
      </div>
      <div className="mt-1 text-2xl font-bold">{value}</div>
    </div>
  );
}

function FailureBreakdownGrid({ failuresByType }: { failuresByType: Record<string, number> }) {
  const sortedFailures = useMemo(() => {
    return Object.entries(failuresByType)
      .filter(([, count]) => count > 0)
      .sort(([, a], [, b]) => b - a);
  }, [failuresByType]);

  if (sortedFailures.length === 0) {
    return (
      <div className="flex items-center gap-2 text-sm text-emerald-400">
        <CheckCircle2 className="h-4 w-4" />
        No failures detected across any desktop bundles
      </div>
    );
  }

  return (
    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
      {sortedFailures.map(([eventType, count]) => (
        <div
          key={eventType}
          className="flex items-center justify-between rounded-lg border border-red-500/20 bg-red-500/5 p-3"
        >
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4 text-red-400" />
            <span className="text-sm font-medium text-red-100">{eventType.replace(/_/g, " ")}</span>
          </div>
          <Badge variant="destructive">{count}</Badge>
        </div>
      ))}
    </div>
  );
}

export function BundleTelemetry() {
  const [showHelp, setShowHelp] = useState(false);
  const [searchFilter, setSearchFilter] = useState("");
  const [showFailuresOnly, setShowFailuresOnly] = useState(false);

  // Fetch all telemetry entries
  const {
    data: telemetry,
    isFetching,
    refetch,
    error,
  } = useQuery({
    queryKey: ["telemetry"],
    queryFn: listTelemetry,
    staleTime: 30_000,
  });

  // Upload mutation
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [scenarioName, setScenarioName] = useState("");
  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!selectedFile) throw new Error("Select a telemetry file first");
      return uploadTelemetry(scenarioName || undefined, selectedFile);
    },
    onSuccess: () => {
      setSelectedFile(null);
      setScenarioName("");
      void refetch();
    },
  });

  // Compute aggregate stats
  const stats = useMemo(() => computeAggregateStats(telemetry ?? []), [telemetry]);

  // Filter entries
  const filteredEntries = useMemo(() => {
    if (!telemetry) return [];

    return telemetry.filter((entry) => {
      // Search filter
      if (searchFilter) {
        const searchLower = searchFilter.toLowerCase();
        const matchesScenario = entry.scenario?.toLowerCase().includes(searchLower);
        const matchesPath = entry.path?.toLowerCase().includes(searchLower);
        if (!matchesScenario && !matchesPath) return false;
      }

      // Failures-only filter
      if (showFailuresOnly) {
        const failureCounts = entry.failure_counts ?? {};
        const hasFailures = Object.values(failureCounts).some((c) => c && c > 0);
        if (!hasFailures) return false;
      }

      return true;
    });
  }, [telemetry, searchFilter, showFailuresOnly]);

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-sm text-amber-100 flex gap-3 items-start">
        <HelpCircle className="h-4 w-4 mt-0.5" />
        <div className="space-y-1">
          <p className="font-semibold">Use after running a packaged app</p>
          <p className="text-amber-50/90">
            Telemetry comes from scenario-to-* builds (e.g., scenario-to-desktop). Upload deployment-telemetry.jsonl after running the packaged app to triage failures and swaps/secrets gaps.
          </p>
        </div>
      </div>

      {/* Page Header */}
      <div className="flex items-start justify-between gap-3 flex-wrap">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-3">
            <Monitor className="h-8 w-8 text-cyan-400" />
            Desktop Bundle Telemetry
          </h1>
          <p className="text-slate-400 mt-1">
            Monitor runtime telemetry from deployed desktop bundles. Track failures, health checks, and deployment events.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowHelp((v) => !v)} className="gap-2">
            <HelpCircle className="h-4 w-4" />
            {showHelp ? "Hide help" : "How this works"}
          </Button>
          <Button size="sm" onClick={() => refetch()} disabled={isFetching} className="gap-2">
            <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      {showHelp && (
        <Tip title="Desktop Bundle Telemetry">
          <p>
            Desktop bundles emit telemetry events during their lifecycle: app_start, ready, shutdown, and various failure events.
            This dashboard aggregates telemetry from all deployed bundles to help you identify and resolve issues.
          </p>
          <ul className="mt-2 list-disc list-inside text-sm text-slate-300 space-y-1">
            <li><strong>dependency_unreachable</strong> — A required service or API couldn't be reached</li>
            <li><strong>swap_missing_asset</strong> — A resource swap is missing bundled assets</li>
            <li><strong>asset_missing</strong> — Required asset file is missing or corrupted</li>
            <li><strong>migration_failed</strong> — Database migration failed during startup</li>
            <li><strong>api_unreachable</strong> — Runtime control API is unreachable</li>
            <li><strong>secrets_missing</strong> — Required secrets were not provided</li>
            <li><strong>health_failed</strong> — Service health check failed</li>
          </ul>
          <p className="mt-2 text-sm text-slate-400">
            Bundles auto-upload telemetry if configured with an upload URL. Use the upload form for manual imports.
          </p>
        </Tip>
      )}

      {/* Error state */}
      {error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-4 text-red-100">
          Failed to load telemetry: {(error as Error).message}
        </div>
      )}

      {/* Aggregate Stats */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Activity className="h-5 w-5" />
            Aggregate Statistics
          </CardTitle>
          <CardDescription>Overview of all desktop bundle telemetry</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Total Bundles" value={stats.totalBundles} icon={Monitor} />
            <StatCard label="Total Events" value={stats.totalEvents} icon={Activity} />
            <StatCard label="Healthy Bundles" value={stats.healthyBundles} icon={CheckCircle2} variant="success" />
            <StatCard
              label="Bundles with Failures"
              value={stats.failedBundles}
              icon={AlertTriangle}
              variant={stats.failedBundles > 0 ? "danger" : "default"}
            />
          </div>
        </CardContent>
      </Card>

      {/* Failure Breakdown */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5" />
            Failure Breakdown
          </CardTitle>
          <CardDescription>
            Aggregate failure counts by event type across all desktop bundles
          </CardDescription>
        </CardHeader>
        <CardContent>
          <FailureBreakdownGrid failuresByType={stats.failuresByType} />
          {stats.totalFailures > 0 && (
            <p className="mt-3 text-sm text-slate-400">
              Total failures: <span className="font-semibold text-red-400">{stats.totalFailures}</span>
            </p>
          )}
        </CardContent>
      </Card>

      {/* Upload Form */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Upload className="h-5 w-5" />
            Upload Telemetry
          </CardTitle>
          <CardDescription>
            Import deployment-telemetry.jsonl from a desktop bundle
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="scenario">Scenario name (optional)</Label>
              <Input
                id="scenario"
                placeholder="picker-wheel"
                value={scenarioName}
                onChange={(e) => setScenarioName(e.target.value)}
              />
              <p className="text-xs text-slate-400">
                Used for grouping telemetry. Auto-detected from file if not specified.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="telemetry-file">Telemetry file</Label>
              <Input
                id="telemetry-file"
                type="file"
                accept=".jsonl,.json,.txt"
                onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
              />
              <p className="text-xs text-slate-400">
                JSONL format: one JSON event per line
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Button
              size="sm"
              onClick={() => uploadMutation.mutate()}
              disabled={uploadMutation.isPending || !selectedFile}
              className="gap-2"
            >
              {uploadMutation.isPending ? (
                <>
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  Uploading...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4" />
                  Upload
                </>
              )}
            </Button>
            {selectedFile && (
              <span className="text-xs text-slate-400 truncate max-w-[200px]">{selectedFile.name}</span>
            )}
          </div>
          {uploadMutation.error && (
            <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-xs text-red-100">
              {(uploadMutation.error as Error).message}
            </div>
          )}
          {uploadMutation.isSuccess && (
            <div className="rounded border border-emerald-500/30 bg-emerald-500/10 p-2 text-xs text-emerald-100">
              Uploaded successfully. Saved to {uploadMutation.data?.path}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Telemetry List */}
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-3 flex-wrap">
            <div>
              <CardTitle>Bundle Telemetry Entries</CardTitle>
              <CardDescription>
                {filteredEntries.length} of {telemetry?.length ?? 0} entries
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="absolute left-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <Input
                  placeholder="Filter by scenario..."
                  value={searchFilter}
                  onChange={(e) => setSearchFilter(e.target.value)}
                  className="pl-8 w-48"
                />
              </div>
              <Button
                variant={showFailuresOnly ? "default" : "outline"}
                size="sm"
                onClick={() => setShowFailuresOnly((v) => !v)}
                className="gap-2"
              >
                <Filter className="h-4 w-4" />
                {showFailuresOnly ? "All" : "Failures only"}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isFetching && !telemetry && (
            <div className="flex items-center gap-2 text-slate-400">
              <RefreshCw className="h-4 w-4 animate-spin" />
              Loading telemetry...
            </div>
          )}

          {!isFetching && filteredEntries.length === 0 && (
            <div className="text-center py-8">
              <Monitor className="h-12 w-12 text-slate-600 mx-auto mb-4" />
              <p className="text-slate-400">
                {telemetry?.length === 0
                  ? "No telemetry ingested yet. Upload a deployment-telemetry.jsonl file to get started."
                  : "No entries match your filters."}
              </p>
            </div>
          )}

          <div className="space-y-3">
            {filteredEntries.map((entry) => (
              <TelemetryEntry key={entry.path} entry={entry} onRefresh={() => refetch()} />
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
