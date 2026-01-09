import { useState, useEffect, useMemo, useRef, useCallback } from "react";
import {
  RefreshCw,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  Clock,
  TrendingUp,
  TrendingDown,
  Info,
  Download,
  ExternalLink,
  LifeBuoy
} from "lucide-react";
import { buildApiUrl } from "@vrooli/api-base";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import type { ScenarioSummary, DeploymentAnalysisReport } from "../../types";
import { MetadataGapsPanel } from "./MetadataGapsPanel";
import { RecommendedFlowPanel } from "./RecommendedFlowPanel";
import { getApiBaseUrl } from "../../lib/utils";

interface DeploymentDashboardProps {
  scenarios: ScenarioSummary[];
  loading: boolean;
  onRefresh: () => void;
  onScanScenario: (scenarioName: string, apply?: boolean) => void | Promise<void>;
  onSelectScenario: (scenarioName: string, options?: { openCatalog?: boolean }) => void;
}

interface ScenarioDeploymentStatus {
  scenario: ScenarioSummary;
  report: DeploymentAnalysisReport | null;
  loading: boolean;
  status: "ready" | "issues" | "not-scanned" | "critical";
  tierFitness: { best: number; worst: number } | null;
  blockersCount: number;
  missingMetadataCount: number;
  lastReport?: DeploymentAnalysisReport | null;
}

async function fetchDeploymentReport(scenarioName: string): Promise<DeploymentAnalysisReport | null> {
  try {
    const baseUrl = getApiBaseUrl();
    const url = buildApiUrl(`/scenarios/${scenarioName}/deployment`, { baseUrl });
    const response = await fetch(url);
    if (!response.ok) {
      return null;
    }
    return await response.json() as DeploymentAnalysisReport;
  } catch (error) {
    console.error(`Failed to fetch deployment report for ${scenarioName}:`, error);
    return null;
  }
}

function getStatusBadge(status: string) {
  switch (status) {
    case "ready":
      return (
        <Badge className="border-green-500/50 bg-green-500/20 text-green-100">
          <CheckCircle2 className="mr-1 h-3 w-3" />
          Ready
        </Badge>
      );
    case "critical":
      return (
        <Badge variant="outline" className="border-red-500/50 bg-red-500/20 text-red-100">
          <AlertTriangle className="mr-1 h-3 w-3" />
          Critical
        </Badge>
      );
    case "issues":
      return (
        <Badge variant="outline" className="border-amber-500/50 bg-amber-500/20 text-amber-100">
          <AlertTriangle className="mr-1 h-3 w-3" />
          Issues
        </Badge>
      );
    default:
      return (
        <Badge variant="secondary">
          <Clock className="mr-1 h-3 w-3" />
          Not Scanned
        </Badge>
      );
  }
}

function getTierFitnessBadge(tierFitness: { best: number; worst: number } | null) {
  if (!tierFitness) {
    return <span className="text-xs text-muted-foreground">N/A</span>;
  }

  const avg = (tierFitness.best + tierFitness.worst) / 2;
  const Icon = avg >= 0.7 ? TrendingUp : TrendingDown;
  const color = avg >= 0.7 ? "text-green-400" : avg >= 0.5 ? "text-amber-400" : "text-red-400";

  return (
    <div className="flex items-center gap-2 text-xs">
      <Icon className={`h-3 w-3 ${color}`} />
      <span>
        {Math.round(tierFitness.best * 100)}% / {Math.round(tierFitness.worst * 100)}%
      </span>
    </div>
  );
}

export function DeploymentDashboard({ scenarios, loading, onRefresh, onScanScenario, onSelectScenario }: DeploymentDashboardProps) {
  const [statuses, setStatuses] = useState<Map<string, ScenarioDeploymentStatus>>(new Map());
  const [loadingReports, setLoadingReports] = useState(false);
  const [showDagHelp, setShowDagHelp] = useState(false);
  const [selectedScenario, setSelectedScenario] = useState<ScenarioDeploymentStatus | null>(null);
  const [targetTier, setTargetTier] = useState<string>("desktop");
  const [search, setSearch] = useState<string>("");
  const [apiError, setApiError] = useState<string | null>(null);
  const apiBase = useMemo(() => getApiBaseUrl(), []);
  const statusRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (scenarios.length === 0) return;

    const loadReports = async () => {
      setLoadingReports(true);
      const newStatuses = new Map<string, ScenarioDeploymentStatus>();
      const batchSize = 5;

      for (let i = 0; i < scenarios.length; i += batchSize) {
        const batch = scenarios.slice(i, i + batchSize);
        const reports = await Promise.all(
          batch.map(async (scenario) => {
            const report = await fetchDeploymentReport(scenario.name);
            if (!report) {
              setApiError(`API is unreachable. Is scenario-dependency-analyzer running? UI base: ${apiBase || "unknown"}`);
            }
            return report;
          })
        );

        batch.forEach((scenario, idx) => {
          const report = reports[idx];
          let status: "ready" | "issues" | "not-scanned" | "critical" = "not-scanned";
          let blockersCount = 0;
          let missingMetadataCount = 0;
          let tierFitness: { best: number; worst: number } | null = null;

          if (report) {
            const gaps = report.metadata_gaps;
            if (gaps) {
              missingMetadataCount = gaps.total_gaps;
              if (gaps.scenarios_missing_all > 0) {
                status = "critical";
              } else if (gaps.total_gaps > 0) {
                status = "issues";
              } else {
                status = "ready";
              }
            }

            if (report.aggregates && Object.keys(report.aggregates).length > 0) {
              const fitnessScores = Object.values(report.aggregates)
                .map((agg) => agg.fitness_score)
                .filter((score) => typeof score === "number" && !isNaN(score));

              if (fitnessScores.length > 0) {
                tierFitness = {
                  best: Math.max(...fitnessScores),
                  worst: Math.min(...fitnessScores)
                };
              }

              Object.values(report.aggregates).forEach((agg) => {
                if (agg.blocking_dependencies) {
                  blockersCount += agg.blocking_dependencies.length;
                }
              });

              if (blockersCount > 0 && status === "ready") {
                status = "issues";
              }
            }
          }

          newStatuses.set(scenario.name, {
            scenario,
            report,
            loading: false,
            status,
            tierFitness,
            blockersCount,
            missingMetadataCount,
            lastReport: report
          });
        });
      }

      setStatuses(newStatuses);
      setLoadingReports(false);
    };

    void loadReports();
  }, [scenarios]);

  const aggregatedGaps = useMemo(() => {
    const allGaps: Record<string, any> = {};
    let totalGaps = 0;
    let scenariosMissingAll = 0;
    const missingTiersSet = new Set<string>();
    const recommendations: string[] = [];

    statuses.forEach((status) => {
      if (status.report?.metadata_gaps) {
        const gaps = status.report.metadata_gaps;
        totalGaps += gaps.total_gaps;
        scenariosMissingAll += gaps.scenarios_missing_all;

        if (gaps.gaps_by_scenario) {
          Object.assign(allGaps, gaps.gaps_by_scenario);
        }

        gaps.missing_tiers?.forEach((tier) => missingTiersSet.add(tier));
        gaps.recommendations?.forEach((rec) => {
          if (!recommendations.includes(rec)) {
            recommendations.push(rec);
          }
        });
      }
    });

    if (totalGaps === 0) return null;

    return {
      total_gaps: totalGaps,
      scenarios_missing_all: scenariosMissingAll,
      gaps_by_scenario: allGaps,
      missing_tiers: Array.from(missingTiersSet),
      recommendations: recommendations.slice(0, 5)
    };
  }, [statuses]);

  const statusArray = Array.from(statuses.values());
  const readyCount = statusArray.filter((s) => s.status === "ready").length;
  const issuesCount = statusArray.filter((s) => s.status === "issues").length;
  const criticalCount = statusArray.filter((s) => s.status === "critical").length;
  const notScannedCount = statusArray.filter((s) => s.status === "not-scanned").length;

  const handleScanAllNonReady = () => {
    statusArray
      .filter((s) => s.status !== "ready")
      .forEach((s) => onScanScenario(s.scenario.name, false));
  };

  const filteredStatusArray = statusArray
    .filter((s) => (search ? s.scenario.name.toLowerCase().includes(search.toLowerCase()) || s.scenario.display_name.toLowerCase().includes(search.toLowerCase()) : true))
    .sort((a, b) => {
      const order = { critical: 0, issues: 1, "not-scanned": 2, ready: 3 };
      if (order[a.status] !== order[b.status]) return order[a.status] - order[b.status];
      return (b.blockersCount || 0) - (a.blockersCount || 0);
    });

  const handleSelectScenario = useCallback(
    (status: ScenarioDeploymentStatus) => {
      setSelectedScenario(status);
      onSelectScenario(status.scenario.name);
    },
    [onSelectScenario]
  );

  const handleScan = useCallback(
    async (scenarioName: string, apply?: boolean) => {
      // Mark row loading
      setStatuses((prev) => {
        const next = new Map(prev);
        const entry = next.get(scenarioName);
        if (entry) {
          next.set(scenarioName, { ...entry, loading: true });
        }
        return next;
      });

      try {
        await Promise.resolve(onScanScenario(scenarioName, apply));
        // Re-fetch just this scenario's deployment report so the row updates visibly
        const refreshed = await fetchDeploymentReport(scenarioName);
        if (!refreshed) {
          setApiError(`API is unreachable. Is scenario-dependency-analyzer running? UI base: ${apiBase || "unknown"}`);
        } else {
          setApiError(null);
        }
        setStatuses((prev) => {
          const next = new Map(prev);
          const entry = next.get(scenarioName);
          if (entry) {
            let status: "ready" | "issues" | "not-scanned" | "critical" = entry.status;
            let blockersCount = entry.blockersCount;
            let missingMetadataCount = entry.missingMetadataCount;
            let tierFitness = entry.tierFitness;

            if (refreshed) {
              const gaps = refreshed.metadata_gaps;
              blockersCount = 0;
              missingMetadataCount = gaps?.total_gaps || 0;
              if (gaps) {
                if (gaps.scenarios_missing_all > 0) {
                  status = "critical";
                } else if (gaps.total_gaps > 0) {
                  status = "issues";
                } else {
                  status = "ready";
                }
              }
              if (refreshed.aggregates && Object.keys(refreshed.aggregates).length > 0) {
                const fitnessScores = Object.values(refreshed.aggregates)
                  .map((agg) => agg.fitness_score)
                  .filter((score) => typeof score === "number" && !isNaN(score));
                if (fitnessScores.length > 0) {
                  tierFitness = {
                    best: Math.max(...fitnessScores),
                    worst: Math.min(...fitnessScores)
                  };
                }
                Object.values(refreshed.aggregates).forEach((agg) => {
                  if (agg.blocking_dependencies) {
                    blockersCount += agg.blocking_dependencies.length;
                  }
                });
                if (blockersCount > 0 && status === "ready") {
                  status = "issues";
                }
              }
            }

            next.set(scenarioName, {
              ...entry,
              report: refreshed,
              lastReport: refreshed,
              status,
              blockersCount,
              missingMetadataCount,
              tierFitness,
              loading: false
            });
          }
          return next;
        });
      } catch (error) {
        console.error("Scan failed", error);
        setApiError("Scan failed. Ensure the API is running (vrooli scenario run scenario-dependency-analyzer) and the UI points to the correct port.");
        setStatuses((prev) => {
          const next = new Map(prev);
          const entry = next.get(scenarioName);
          if (entry) {
            next.set(scenarioName, { ...entry, loading: false });
          }
          return next;
        });
      }
    },
    [onScanScenario]
  );

  const handleExportDag = useCallback(
    async (scenarioName?: string) => {
      const name = scenarioName || selectedScenario?.scenario.name;
      if (!name) {
        setShowDagHelp(true);
        return;
      }
      try {
        const apiPort = import.meta.env.VITE_API_PORT || "20400";
        const url = buildApiUrl(`/scenarios/${name}/dag/export?recursive=true`, { baseUrl: apiBase });
        const response = await fetch(url);
        if (!response.ok) {
          setShowDagHelp(true);
          return;
        }
        const data = await response.json();
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
        const downloadUrl = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = downloadUrl;
        link.download = `${name}-dag-export.json`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(downloadUrl);
      } catch (e) {
        console.error("Failed to export DAG", e);
        setShowDagHelp(true);
      }
    },
    [selectedScenario]
  );

  const tierOptions = [
    { value: "desktop", label: "Tier 2 · Desktop" },
    { value: "local_dev", label: "Tier 1 · Local / Dev" },
    { value: "mobile", label: "Tier 3 · Mobile" },
    { value: "saas", label: "Tier 4 · SaaS / Cloud" },
    { value: "enterprise", label: "Tier 5 · Enterprise" }
  ];

  const selectedTierFitness = selectedScenario?.lastReport?.aggregates?.[targetTier]?.fitness_score;
  const selectedTierBlockers = selectedScenario?.lastReport?.aggregates?.[targetTier]?.blocking_dependencies || [];
  const selectedTierRequirements = selectedScenario?.lastReport?.aggregates?.[targetTier]?.estimated_requirements;

  return (
    <div className="space-y-6">
      <Card className="border border-border/60 bg-background/50">
        <CardHeader className="pb-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <CardTitle className="text-lg">Deployment Readiness</CardTitle>
              <p className="text-xs text-muted-foreground">
                Goal: prepare scenarios for <strong>Tier 2 desktop</strong> (portable app with UI+API+resources) and beyond.
                Pick a target tier, scan, then fix blockers with the inline guide.
              </p>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Info className="h-3.5 w-3.5" aria-hidden="true" />
              <span>Need a refresher later? Hover the tips and help buttons for definitions.</span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {apiError ? (
            <div className="rounded border border-red-500/40 bg-red-500/10 p-3 text-xs text-red-100">
              <p className="font-medium">API unreachable</p>
              <p className="mt-1">
                {apiError} Start the scenario via <code>vrooli scenario run scenario-dependency-analyzer</code>. If you use a custom port,
                set <code>VITE_API_PORT</code> before running <code>npm run dev</code>, or configure proxy metadata to point the UI at the right base.
              </p>
            </div>
          ) : null}
          <div className="flex flex-wrap items-center gap-2">
            <label className="text-xs text-muted-foreground">Target tier</label>
            <div className="flex flex-wrap gap-2">
              {tierOptions.map((tier) => (
                <Button
                  key={tier.value}
                  size="sm"
                  variant={targetTier === tier.value ? "secondary" : "outline"}
                  className="h-8 text-xs"
                  onClick={() => setTargetTier(tier.value)}
                >
                  {tier.label}
                </Button>
              ))}
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <LifeBuoy className="h-3.5 w-3.5" aria-hidden="true" />
              <button
                className="underline-offset-2 hover:underline"
                onClick={() => window.open("/docs/deployment/tiers", "_blank")}
              >
                View tier definitions
              </button>
            </div>
          </div>
          <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
            <span className="font-medium text-foreground">What “ready” means:</span>
            <span>Deployment metadata exists for the target tier, no blocking dependencies, and tier fitness is acceptable.</span>
            <span className="text-primary">“Scan & Apply” writes inferred metadata to .vrooli/service.json.</span>
          </div>
        </CardContent>
      </Card>

      <RecommendedFlowPanel
        onScanAll={handleScanAllNonReady}
        onExportDAG={() => handleExportDag()}
        onJumpToStatus={() => statusRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })}
        dagHelp={showDagHelp ? "dag" : null}
        onOpenDocs={() => window.open("/docs/deployment", "_blank")}
        targetTierLabel={tierOptions.find((t) => t.value === targetTier)?.label}
      />

      <div className="grid gap-4 md:grid-cols-4">
        <Card className="border border-green-500/40 bg-green-500/5">
          <CardContent className="pt-6">
            <div className="text-2xl font-bold text-green-100">{readyCount}</div>
            <p className="text-xs text-muted-foreground">Ready for deployment</p>
          </CardContent>
        </Card>
        <Card className="border border-amber-500/40 bg-amber-500/5">
          <CardContent className="pt-6">
            <div className="text-2xl font-bold text-amber-100">{issuesCount}</div>
            <p className="text-xs text-muted-foreground">With issues</p>
          </CardContent>
        </Card>
        <Card className="border border-red-500/40 bg-red-500/5">
          <CardContent className="pt-6">
            <div className="text-2xl font-bold text-red-100">{criticalCount}</div>
            <p className="text-xs text-muted-foreground">Critical gaps</p>
          </CardContent>
        </Card>
        <Card className="border border-border/40 bg-background/40">
          <CardContent className="pt-6">
            <div className="text-2xl font-bold text-foreground">{notScannedCount}</div>
            <p className="text-xs text-muted-foreground">Not yet scanned</p>
          </CardContent>
        </Card>
      </div>

      <Card className="border border-border/40 bg-background/40">
        <CardContent className="py-4 text-xs text-muted-foreground">
          <div className="flex items-start gap-2">
            <Info className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
            <p className="font-medium text-foreground">How to read this page</p>
          </div>
          <p className="mt-1">
            Ready = metadata present, no blockers. Issues = gaps or blockers detected. Critical = missing all deployment
            metadata. Not scanned = run Scan to populate data. Fitness is best/worst tier score; blockers are dependencies
            that fail a tier. Scan &amp; Apply writes updates to service.json. Use the tier buttons above to see fitness
            and blockers for the platform you are targeting.
          </p>
        </CardContent>
      </Card>

      {aggregatedGaps && <MetadataGapsPanel gaps={aggregatedGaps} />}

      <Card className="border border-border/40 bg-background/40" ref={statusRef} id="deployment-status">
        <CardHeader className="flex flex-row items-center justify-between pb-1">
          <div>
            <CardTitle className="text-base">Scenario Deployment Status</CardTitle>
            <p className="text-xs text-muted-foreground">
              Click a row to open inline details (blockers, requirements, metadata gaps) for the selected tier.
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={loading || loadingReports}
            className="gap-2"
          >
            {loading || loadingReports ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
            Refresh
          </Button>
        </CardHeader>
        <CardContent>
          <div className="mb-3 flex flex-wrap items-center gap-3">
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search scenarios..."
              className="h-9 w-full max-w-xs rounded border border-border/60 bg-background/60 px-3 text-sm text-foreground placeholder:text-muted-foreground"
              aria-label="Search scenarios"
            />
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Info className="h-3.5 w-3.5" aria-hidden="true" />
              <span>Sorted by urgency: critical → issues → not scanned → ready.</span>
            </div>
          </div>
          {loading || loadingReports ? (
            <div className="flex min-h-[200px] items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : (
            <div className="space-y-2">
              {filteredStatusArray.map((status) => {
                const tierAgg = status.lastReport?.aggregates?.[targetTier];
                const tierFitness = tierAgg?.fitness_score;
                const blockers = tierAgg?.blocking_dependencies || [];
                return (
                  <div
                    key={status.scenario.name}
                    className={`flex cursor-pointer items-center justify-between rounded-lg border border-border/40 bg-background/40 p-3 transition-colors hover:bg-background/60 ${
                      selectedScenario?.scenario.name === status.scenario.name ? "ring-2 ring-primary/50" : ""
                    }`}
                    onClick={() => handleSelectScenario(status)}
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-medium text-foreground">
                          {status.scenario.display_name || status.scenario.name}
                        </p>
                        {getStatusBadge(status.status)}
                      </div>
                      {status.scenario.description && (
                        <p className="mt-1 truncate text-xs text-muted-foreground">{status.scenario.description}</p>
                      )}
                      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                        <span>
                          Last scan:{" "}
                          {status.scenario.last_scanned
                            ? new Date(status.scenario.last_scanned).toLocaleString()
                            : "Never"}
                        </span>
                        <span title="Best/worst across tiers">Tier fitness: {getTierFitnessBadge(status.tierFitness)}</span>
                        <span title={`Fitness for ${targetTier}`}>
                          {tierFitness !== undefined ? `${Math.round((tierFitness || 0) * 100)}% for ${targetTier}` : "No tier score"}
                        </span>
                        {blockers.length > 0 && <span className="text-red-400">Blockers ({targetTier}): {blockers.length}</span>}
                        {status.missingMetadataCount > 0 && (
                          <span className="text-amber-400">Missing metadata: {status.missingMetadataCount}</span>
                        )}
                      </div>
                    </div>
                    <div className="ml-4 flex gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleScan(status.scenario.name, false);
                        }}
                        className="h-8 text-xs"
                        disabled={status.loading}
                      >
                        {status.loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Scan"}
                      </Button>
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleScan(status.scenario.name, true);
                        }}
                        className="h-8 text-xs"
                        disabled={status.loading}
                      >
                        {status.loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Scan & Apply"}
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {selectedScenario && (
        <Card className="border border-border/50 bg-background/50">
          <CardHeader className="flex flex-wrap items-center justify-between gap-3 pb-3">
            <div>
              <CardTitle className="text-base">
                Deployment details: {selectedScenario.scenario.display_name || selectedScenario.scenario.name}
              </CardTitle>
              <p className="text-xs text-muted-foreground">
                Focused on {tierOptions.find((t) => t.value === targetTier)?.label}. Tips below explain each field.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" className="gap-2" onClick={() => handleExportDag(selectedScenario.scenario.name)}>
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
                onClick={() => window.open(`/docs/deployment/tiers/tier-2-desktop.md`, "_blank")}
              >
                <ExternalLink className="h-4 w-4" />
                Tier guide
              </Button>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 md:grid-cols-3">
              <div className="rounded border border-border/40 bg-background/40 p-3">
                <p className="text-xs uppercase tracking-wide text-muted-foreground flex items-center gap-1">
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
                <p className="text-xs uppercase tracking-wide text-muted-foreground flex items-center gap-1">
                  <AlertTriangle className="h-3.5 w-3.5 text-amber-400" aria-hidden="true" />
                  Blockers ({targetTier})
                </p>
                <p className="mt-1 text-2xl font-semibold text-amber-200">{selectedTierBlockers.length}</p>
                <p className="text-[11px] text-muted-foreground">
                  These dependencies cannot run on this tier. Fix by adding alternatives in <code>deployment.dependencies</code> or swapping resources.
                </p>
              </div>
              <div className="rounded border border-border/40 bg-background/40 p-3">
                <p className="text-xs uppercase tracking-wide text-muted-foreground flex items-center gap-1">
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
                  <AlertTriangle className="h-4 w-4 text-amber-400" aria-hidden="true" />
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
              {selectedScenario.lastReport?.metadata_gaps?.gaps_by_scenario?.[selectedScenario.scenario.name] ? (
                <div className="space-y-2 text-xs text-muted-foreground">
                  <p className="text-[11px] text-muted-foreground">
                    Fill these fields in <code>.vrooli/service.json</code>. Use the docs and the “Scan &amp; Apply” button to re-check.
                  </p>
                  <MetadataGapsPanel
                    gaps={{
                      total_gaps: selectedScenario.lastReport.metadata_gaps.total_gaps,
                      scenarios_missing_all: selectedScenario.lastReport.metadata_gaps.scenarios_missing_all,
                      gaps_by_scenario: {
                        [selectedScenario.scenario.name]:
                          selectedScenario.lastReport.metadata_gaps.gaps_by_scenario?.[selectedScenario.scenario.name]
                      }
                    }}
                  />
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">No metadata gaps reported for this scenario.</p>
              )}
            </div>

            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="secondary" onClick={() => handleScan(selectedScenario.scenario.name, false)}>
                Re-scan
              </Button>
              <Button size="sm" variant="secondary" onClick={() => handleScan(selectedScenario.scenario.name, true)}>
                Scan & Apply (write metadata)
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => window.open(`/docs/deployment/guides/dependency-swapping.md`, "_blank")}
              >
                Read: Dependency swapping
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => onSelectScenario(selectedScenario.scenario.name, { openCatalog: true })}
              >
                Open in catalog view
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
