import { useQuery } from "@tanstack/react-query";
import { RefreshCw, Settings, AlertCircle, ExternalLink } from "lucide-react";
import { Button } from "../components/ui/button";
import { ScoreClassificationBadge } from "../components/ScoreClassificationBadge";
import { HealthBadge } from "../components/HealthBadge";
import {
  fetchScores,
  fetchCollectorHealth,
  type ScenarioScore,
} from "../lib/api";

interface DashboardProps {
  onSelectScenario: (scenario: string) => void;
  onOpenConfig: () => void;
}

// [REQ:SCS-UI-001] Dashboard overview with all scenarios
export function Dashboard({ onSelectScenario, onOpenConfig }: DashboardProps) {
  const {
    data: scoresData,
    isLoading: scoresLoading,
    error: scoresError,
    refetch: refetchScores,
  } = useQuery({
    queryKey: ["scores"],
    queryFn: fetchScores,
    refetchInterval: 30000, // Refresh every 30s
  });

  const { data: healthData } = useQuery({
    queryKey: ["collector-health"],
    queryFn: fetchCollectorHealth,
    refetchInterval: 30000,
  });

  const unhealthyCollectors = healthData
    ? Object.entries(healthData.collectors).filter(
        ([, c]) => c.status !== "ok"
      )
    : [];

  // Sort scenarios by score descending
  const sortedScenarios = scoresData?.scenarios
    ? [...scoresData.scenarios].sort((a, b) => b.score - a.score)
    : [];

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6" data-testid="dashboard">
      <div className="max-w-6xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold">Scenario Health Overview</h1>
            <p className="text-slate-400 text-sm mt-1">
              {scoresData?.total ?? 0} scenarios tracked
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetchScores()}
              disabled={scoresLoading}
            >
              <RefreshCw
                className={`h-4 w-4 mr-2 ${scoresLoading ? "animate-spin" : ""}`}
              />
              Refresh
            </Button>
            <Button variant="outline" size="sm" onClick={onOpenConfig}>
              <Settings className="h-4 w-4 mr-2" />
              Config
            </Button>
          </div>
        </div>

        {/* Health Alert Banner */}
        {unhealthyCollectors.length > 0 && (
          <div
            className="mb-6 p-4 rounded-xl border border-amber-500/30 bg-amber-500/10"
            data-testid="health-alert-banner"
          >
            <div className="flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-amber-400 mt-0.5" />
              <div>
                <p className="font-medium text-amber-200">
                  {unhealthyCollectors.length} collector
                  {unhealthyCollectors.length > 1 ? "s" : ""} unhealthy
                </p>
                <div className="flex flex-wrap gap-2 mt-2">
                  {unhealthyCollectors.map(([name, collector]) => (
                    <HealthBadge
                      key={name}
                      status={collector.status}
                      label={name}
                    />
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Error State */}
        {scoresError && (
          <div className="mb-6 p-4 rounded-xl border border-red-500/30 bg-red-500/10">
            <div className="flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-red-400 mt-0.5" />
              <div>
                <p className="font-medium text-red-200">Failed to load scenarios</p>
                <p className="text-sm text-red-300/70 mt-1">
                  {scoresError instanceof Error
                    ? scoresError.message
                    : "Unknown error"}
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Scenarios Table */}
        <div className="rounded-xl border border-white/10 bg-white/5 overflow-hidden">
          <table className="w-full" data-testid="scenarios-table">
            <thead>
              <tr className="border-b border-white/10 bg-white/5">
                <th className="text-left px-4 py-3 text-sm font-medium text-slate-400">
                  Scenario
                </th>
                <th className="text-left px-4 py-3 text-sm font-medium text-slate-400">
                  Category
                </th>
                <th className="text-center px-4 py-3 text-sm font-medium text-slate-400">
                  Score
                </th>
                <th className="text-left px-4 py-3 text-sm font-medium text-slate-400">
                  Classification
                </th>
                <th className="text-right px-4 py-3 text-sm font-medium text-slate-400">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {scoresLoading && (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-slate-400">
                    <RefreshCw className="h-6 w-6 animate-spin mx-auto mb-2" />
                    Loading scenarios...
                  </td>
                </tr>
              )}
              {!scoresLoading && sortedScenarios.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-slate-400">
                    No scenarios found
                  </td>
                </tr>
              )}
              {sortedScenarios.map((scenario: ScenarioScore) => (
                <tr
                  key={scenario.scenario}
                  className="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
                  onClick={() => onSelectScenario(scenario.scenario)}
                  data-testid={`scenario-row-${scenario.scenario}`}
                >
                  <td className="px-4 py-3">
                    <span className="font-medium text-slate-200">
                      {scenario.scenario}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-sm text-slate-400">
                      {scenario.category}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-center">
                    <span
                      className={`font-bold text-lg ${
                        scenario.score >= 80
                          ? "text-emerald-400"
                          : scenario.score >= 50
                          ? "text-amber-400"
                          : "text-red-400"
                      }`}
                    >
                      {scenario.score}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <ScoreClassificationBadge
                      classification={scenario.classification}
                    />
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectScenario(scenario.scenario);
                      }}
                    >
                      <ExternalLink className="h-3.5 w-3.5 mr-1" />
                      Details
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Footer info */}
        {scoresData && (
          <p className="mt-4 text-xs text-slate-500 text-right">
            Last calculated:{" "}
            {new Date(scoresData.calculated_at).toLocaleString()}
          </p>
        )}
      </div>
    </div>
  );
}
