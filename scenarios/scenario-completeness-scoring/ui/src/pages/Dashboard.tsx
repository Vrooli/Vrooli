import { useState, useMemo, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, Settings, AlertCircle, ExternalLink, Search, Activity, CheckCircle2, AlertTriangle, XCircle, Clock, TrendingDown, Zap, Sparkles } from "lucide-react";
import { Button } from "../components/ui/button";
import { ScoreClassificationBadge } from "../components/ScoreClassificationBadge";
import { HealthBadge } from "../components/HealthBadge";
import { ScoreLegend } from "../components/ScoreLegend";
import { useRecentScenarios } from "../hooks/useRecentScenarios";
import {
  fetchScores,
  fetchCollectorHealth,
  refreshAllScores,
  type ScenarioScore,
} from "../lib/api";

interface DashboardProps {
  onSelectScenario: (scenario: string) => void;
  onOpenConfig: () => void;
}

// [REQ:SCS-UI-001] Dashboard overview with all scenarios
export function Dashboard({ onSelectScenario, onOpenConfig }: DashboardProps) {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<string>("all");
  const { recentScenarios, markViewed, updateScenarioData } = useRecentScenarios();

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

  // Bulk refresh mutation for ops users
  const bulkRefreshMutation = useMutation({
    mutationFn: refreshAllScores,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scores"] });
    },
  });

  const { data: healthData } = useQuery({
    queryKey: ["collector-health"],
    queryFn: fetchCollectorHealth,
    refetchInterval: 30000,
  });

  // Update recent scenarios with fresh score data when loaded
  useEffect(() => {
    if (scoresData?.scenarios) {
      recentScenarios.forEach((recent) => {
        const match = scoresData.scenarios.find((s) => s.scenario === recent.name);
        if (match && (match.score !== recent.score || match.classification !== recent.classification)) {
          updateScenarioData(recent.name, match.score, match.classification);
        }
      });
    }
  }, [scoresData?.scenarios, recentScenarios, updateScenarioData]);

  const unhealthyCollectors = healthData
    ? Object.entries(healthData.collectors).filter(
        ([, c]) => c.status !== "ok"
      )
    : [];

  // Extract unique categories for filter dropdown
  const categories = useMemo(() => {
    if (!scoresData?.scenarios) return [];
    const cats = new Set(scoresData.scenarios.map((s) => s.category));
    return Array.from(cats).sort();
  }, [scoresData?.scenarios]);

  // Filter and sort scenarios
  const filteredScenarios = useMemo(() => {
    if (!scoresData?.scenarios) return [];
    return scoresData.scenarios
      .filter((s) => {
        const matchesSearch = searchQuery === "" ||
          s.scenario.toLowerCase().includes(searchQuery.toLowerCase()) ||
          s.category.toLowerCase().includes(searchQuery.toLowerCase());
        const matchesCategory = categoryFilter === "all" || s.category === categoryFilter;
        return matchesSearch && matchesCategory;
      })
      .sort((a, b) => b.score - a.score);
  }, [scoresData?.scenarios, searchQuery, categoryFilter]);

  // Identify scenarios that need attention (low score <50 or declining)
  const needsAttentionScenarios = useMemo(() => {
    if (!scoresData?.scenarios) return [];
    return scoresData.scenarios
      .filter((s) => s.score < 50)
      .sort((a, b) => a.score - b.score)
      .slice(0, 3);
  }, [scoresData?.scenarios]);

  // Health summary stats
  const healthSummary = healthData?.summary ?? { healthy: 0, degraded: 0, failed: 0, total: 0 };

  // Handle scenario selection with tracking
  const handleSelectScenario = (scenarioName: string) => {
    const scenario = scoresData?.scenarios.find((s) => s.scenario === scenarioName);
    markViewed(scenarioName, scenario?.score, scenario?.classification);
    onSelectScenario(scenarioName);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6" data-testid="dashboard">
      <div className="max-w-6xl mx-auto">
        {/* Header with purpose statement */}
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold">Scenario Health Overview</h1>
            <p className="text-slate-400 text-sm mt-1">
              Track scenario completeness scores and identify areas for improvement
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => bulkRefreshMutation.mutate()}
              disabled={bulkRefreshMutation.isPending}
              title="Recalculate scores for all scenarios"
              data-testid="bulk-refresh-button"
            >
              <Zap
                className={`h-4 w-4 mr-2 ${bulkRefreshMutation.isPending ? "animate-pulse" : ""}`}
              />
              {bulkRefreshMutation.isPending ? "Refreshing All..." : "Refresh All"}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetchScores()}
              disabled={scoresLoading}
              title="Reload cached scores"
            >
              <RefreshCw
                className={`h-4 w-4 mr-2 ${scoresLoading ? "animate-spin" : ""}`}
              />
              Reload
            </Button>
            <Button variant="outline" size="sm" onClick={onOpenConfig}>
              <Settings className="h-4 w-4 mr-2" />
              Config
            </Button>
          </div>
        </div>

        {/* Health Summary Bar - always visible for ops users */}
        <div className="flex items-center justify-between mb-6 p-3 rounded-lg bg-white/5 border border-white/10" data-testid="health-summary-bar">
          <div className="flex items-center gap-4">
            <Activity className="h-4 w-4 text-slate-400" />
            <span className="text-sm text-slate-400">Collectors:</span>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-1.5" title="Healthy collectors">
                <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                <span className="text-sm font-medium text-emerald-400">{healthSummary.healthy}</span>
              </div>
              {healthSummary.degraded > 0 && (
                <div className="flex items-center gap-1.5" title="Degraded collectors">
                  <AlertTriangle className="h-4 w-4 text-amber-400" />
                  <span className="text-sm font-medium text-amber-400">{healthSummary.degraded}</span>
                </div>
              )}
              {healthSummary.failed > 0 && (
                <div className="flex items-center gap-1.5" title="Failed collectors">
                  <XCircle className="h-4 w-4 text-red-400" />
                  <span className="text-sm font-medium text-red-400">{healthSummary.failed}</span>
                </div>
              )}
            </div>
            <span className="text-slate-600">|</span>
            <span className="text-sm text-slate-400">{scoresData?.total ?? 0} scenarios</span>
          </div>
          <ScoreLegend />
        </div>

        {/* First-time User Guidance - shown when no recent scenarios */}
        {recentScenarios.length === 0 && !scoresLoading && scoresData?.scenarios && scoresData.scenarios.length > 0 && (
          <div className="mb-6 p-4 rounded-xl border border-blue-500/20 bg-blue-500/5" data-testid="welcome-guidance">
            <div className="flex items-start gap-3">
              <Sparkles className="h-5 w-5 text-blue-400 mt-0.5" />
              <div>
                <h3 className="font-medium text-blue-200">Welcome to Scenario Completeness Scoring</h3>
                <p className="text-sm text-blue-300/70 mt-1">
                  This tool helps you understand how production-ready your scenarios are. Each scenario is scored from 0-100 across four dimensions: Quality, Coverage, Quantity, and UI.
                </p>
                <p className="text-sm text-blue-300/70 mt-2">
                  <strong>Get started:</strong> Click on any scenario below to see its detailed breakdown and recommendations for improvement.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Quick Access: Recent Scenarios and Needs Attention */}
        {(recentScenarios.length > 0 || needsAttentionScenarios.length > 0) && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6" data-testid="quick-access-section">
            {/* Recent Scenarios */}
            {recentScenarios.length > 0 && (
              <div className="rounded-xl border border-white/10 bg-white/5 p-4" data-testid="recent-scenarios">
                <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
                  <Clock className="h-4 w-4" />
                  Continue where you left off
                </h3>
                <div className="space-y-2">
                  {recentScenarios.map((recent) => (
                    <button
                      key={recent.name}
                      onClick={() => handleSelectScenario(recent.name)}
                      className="w-full flex items-center justify-between p-2 rounded-lg hover:bg-white/5 transition-colors text-left"
                    >
                      <span className="font-medium text-slate-200 truncate">
                        {recent.name}
                      </span>
                      {recent.score !== undefined && (
                        <span
                          className={`text-sm font-bold ${
                            recent.score >= 80
                              ? "text-emerald-400"
                              : recent.score >= 50
                              ? "text-amber-400"
                              : "text-red-400"
                          }`}
                        >
                          {recent.score}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Needs Attention */}
            {needsAttentionScenarios.length > 0 && (
              <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4" data-testid="needs-attention">
                <h3 className="text-sm font-medium text-red-400 mb-3 flex items-center gap-2">
                  <TrendingDown className="h-4 w-4" />
                  Needs attention
                </h3>
                <div className="space-y-2">
                  {needsAttentionScenarios.map((scenario) => (
                    <button
                      key={scenario.scenario}
                      onClick={() => handleSelectScenario(scenario.scenario)}
                      className="w-full flex items-center justify-between p-2 rounded-lg hover:bg-white/5 transition-colors text-left"
                    >
                      <span className="font-medium text-slate-200 truncate">
                        {scenario.scenario}
                      </span>
                      <span className="text-sm font-bold text-red-400">
                        {scenario.score}
                      </span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Search and Filter Bar */}
        <div className="flex items-center gap-3 mb-4" data-testid="search-filter-bar">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search scenarios..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 rounded-lg bg-white/5 border border-white/10 text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
              data-testid="search-input"
            />
          </div>
          <select
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
            className="px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/50"
            data-testid="category-filter"
          >
            <option value="all">All categories</option>
            {categories.map((cat) => (
              <option key={cat} value={cat}>{cat}</option>
            ))}
          </select>
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
              {!scoresLoading && filteredScenarios.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-slate-400">
                    {searchQuery || categoryFilter !== "all"
                      ? "No scenarios match your search"
                      : "No scenarios found"}
                  </td>
                </tr>
              )}
              {filteredScenarios.map((scenario: ScenarioScore) => (
                <tr
                  key={scenario.scenario}
                  className="border-b border-white/5 hover:bg-white/5 cursor-pointer transition-colors"
                  onClick={() => handleSelectScenario(scenario.scenario)}
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
                        handleSelectScenario(scenario.scenario);
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
