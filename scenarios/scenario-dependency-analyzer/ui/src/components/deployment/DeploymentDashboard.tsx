import { useState, useEffect, useMemo } from "react";
import { RefreshCw, Loader2, AlertTriangle, CheckCircle2, Clock, TrendingUp, TrendingDown } from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import type { ScenarioSummary, DeploymentAnalysisReport } from "../../types";
import { MetadataGapsPanel } from "./MetadataGapsPanel";
import { RecommendedFlowPanel } from "./RecommendedFlowPanel";

interface DeploymentDashboardProps {
  scenarios: ScenarioSummary[];
  loading: boolean;
  onRefresh: () => void;
  onScanScenario: (scenarioName: string, apply?: boolean) => void;
  onSelectScenario: (scenarioName: string) => void;
}

interface ScenarioDeploymentStatus {
  scenario: ScenarioSummary;
  report: DeploymentAnalysisReport | null;
  loading: boolean;
  status: "ready" | "issues" | "not-scanned" | "critical";
  tierFitness: { best: number; worst: number } | null;
  blockersCount: number;
  missingMetadataCount: number;
}

async function fetchDeploymentReport(scenarioName: string): Promise<DeploymentAnalysisReport | null> {
  try {
    const apiPort = import.meta.env.VITE_API_PORT || "20400";
    const response = await fetch(`http://localhost:${apiPort}/api/v1/scenarios/${scenarioName}/deployment`);
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
      return <Badge className="bg-green-500/20 text-green-100 border-green-500/50"><CheckCircle2 className="mr-1 h-3 w-3" />Ready</Badge>;
    case "critical":
      return <Badge variant="outline" className="bg-red-500/20 text-red-100 border-red-500/50"><AlertTriangle className="mr-1 h-3 w-3" />Critical</Badge>;
    case "issues":
      return <Badge variant="outline" className="bg-amber-500/20 text-amber-100 border-amber-500/50"><AlertTriangle className="mr-1 h-3 w-3" />Issues</Badge>;
    default:
      return <Badge variant="secondary"><Clock className="mr-1 h-3 w-3" />Not Scanned</Badge>;
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
      <span>{Math.round(tierFitness.best * 100)}% / {Math.round(tierFitness.worst * 100)}%</span>
    </div>
  );
}

export function DeploymentDashboard({ scenarios, loading, onRefresh, onScanScenario, onSelectScenario }: DeploymentDashboardProps) {
  const [statuses, setStatuses] = useState<Map<string, ScenarioDeploymentStatus>>(new Map());
  const [loadingReports, setLoadingReports] = useState(false);

  // Fetch deployment reports for all scenarios
  useEffect(() => {
    if (scenarios.length === 0) return;

    const loadReports = async () => {
      setLoadingReports(true);
      const newStatuses = new Map<string, ScenarioDeploymentStatus>();

      // Fetch reports in parallel (with reasonable limit)
      const batchSize = 5;
      for (let i = 0; i < scenarios.length; i += batchSize) {
        const batch = scenarios.slice(i, i + batchSize);
        const reports = await Promise.all(
          batch.map(scenario => fetchDeploymentReport(scenario.name))
        );

        batch.forEach((scenario, idx) => {
          const report = reports[idx];

          // Determine status
          let status: "ready" | "issues" | "not-scanned" | "critical" = "not-scanned";
          let blockersCount = 0;
          let missingMetadataCount = 0;
          let tierFitness: { best: number; worst: number } | null = null;

          if (report) {
            // Check metadata gaps
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

            // Calculate tier fitness
            if (report.aggregates && Object.keys(report.aggregates).length > 0) {
              const fitnessScores = Object.values(report.aggregates)
                .map(agg => agg.fitness_score)
                .filter(score => typeof score === "number" && !isNaN(score));

              if (fitnessScores.length > 0) {
                tierFitness = {
                  best: Math.max(...fitnessScores),
                  worst: Math.min(...fitnessScores)
                };
              }

              // Count blockers
              Object.values(report.aggregates).forEach(agg => {
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
            missingMetadataCount
          });
        });
      }

      setStatuses(newStatuses);
      setLoadingReports(false);
    };

    void loadReports();
  }, [scenarios]);

  // Aggregate metadata gaps across all scenarios
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

        gaps.missing_tiers?.forEach(tier => missingTiersSet.add(tier));
        gaps.recommendations?.forEach(rec => {
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
      recommendations: recommendations.slice(0, 5) // Limit recommendations
    };
  }, [statuses]);

  const statusArray = Array.from(statuses.values());
  const readyCount = statusArray.filter(s => s.status === "ready").length;
  const issuesCount = statusArray.filter(s => s.status === "issues").length;
  const criticalCount = statusArray.filter(s => s.status === "critical").length;
  const notScannedCount = statusArray.filter(s => s.status === "not-scanned").length;

  return (
    <div className="space-y-6">
      {/* Recommended Flow Guide */}
      <RecommendedFlowPanel
        onScanAll={() => {
          // Trigger scan for all scenarios with issues
          statusArray
            .filter(s => s.status !== "ready")
            .forEach(s => onScanScenario(s.scenario.name, false));
        }}
        onExportDAG={() => {
          // Show CLI hint for DAG export
          alert(
            'To export a DAG, use the CLI:\n\n' +
            'scenario-dependency-analyzer dag export <scenario> --recursive\n\n' +
            'Example:\n' +
            'scenario-dependency-analyzer dag export ecosystem-manager --recursive'
          );
        }}
      />

      {/* Summary Stats */}
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

      {/* Metadata Gaps Panel */}
      {aggregatedGaps && <MetadataGapsPanel gaps={aggregatedGaps} />}

      {/* Scenarios Table */}
      <Card className="border border-border/40 bg-background/40">
        <CardHeader className="flex flex-row items-center justify-between pb-3">
          <CardTitle className="text-base">Scenario Deployment Status</CardTitle>
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={loading || loadingReports}
            className="gap-2"
          >
            {loading || loadingReports ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </Button>
        </CardHeader>
        <CardContent>
          {loading || loadingReports ? (
            <div className="flex min-h-[200px] items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : (
            <div className="space-y-2">
              {statusArray.map((status) => (
                <div
                  key={status.scenario.name}
                  className="flex cursor-pointer items-center justify-between rounded-lg border border-border/40 bg-background/40 p-3 transition-colors hover:bg-background/60"
                  onClick={() => onSelectScenario(status.scenario.name)}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <p className="font-medium text-foreground">{status.scenario.display_name || status.scenario.name}</p>
                      {getStatusBadge(status.status)}
                    </div>
                    {status.scenario.description && (
                      <p className="mt-1 truncate text-xs text-muted-foreground">{status.scenario.description}</p>
                    )}
                    <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <span>Last scan: {status.scenario.last_scanned ? new Date(status.scenario.last_scanned).toLocaleString() : "Never"}</span>
                      <span>Tier fitness: {getTierFitnessBadge(status.tierFitness)}</span>
                      {status.blockersCount > 0 && (
                        <span className="text-red-400">Blockers: {status.blockersCount}</span>
                      )}
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
                        onScanScenario(status.scenario.name, false);
                      }}
                      className="h-8 text-xs"
                    >
                      Scan
                    </Button>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onScanScenario(status.scenario.name, true);
                      }}
                      className="h-8 text-xs"
                    >
                      Scan & Apply
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
