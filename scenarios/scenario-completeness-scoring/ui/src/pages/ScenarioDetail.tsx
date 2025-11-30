import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  RefreshCw,
  TrendingUp,
  Zap,
  BarChart3,
  Settings,
  Beaker,
  Check,
  AlertTriangle,
  Lightbulb,
} from "lucide-react";
import { Button } from "../components/ui/button";
import { ScoreBar } from "../components/ScoreBar";
import { TrendIndicator } from "../components/TrendIndicator";
import { ScoreClassificationBadge } from "../components/ScoreClassificationBadge";
import { Sparkline } from "../components/Sparkline";
import {
  fetchScoreDetail,
  fetchHistory,
  fetchTrends,
  calculateScore,
  type Recommendation,
} from "../lib/api";

interface ScenarioDetailProps {
  scenario: string;
  onBack: () => void;
  onOpenConfig?: () => void;
}

// Safe accessor for nested score properties
// ASSUMPTION: API responses may have missing or null nested fields
// HARDENED: Guards against undefined/null at any depth
const safeRate = (passRate: { rate?: number } | undefined | null): number => {
  return passRate?.rate ?? 0;
};

const safePoints = (metric: { points?: number } | undefined | null): number => {
  return metric?.points ?? 0;
};

const safeCount = (metric: { count?: number } | undefined | null): number => {
  return metric?.count ?? 0;
};

const safeRatio = (ratio: { ratio?: number } | undefined | null): number => {
  return ratio?.ratio ?? 0;
};

const safeAvgDepth = (depth: { avg_depth?: number } | undefined | null): number => {
  return depth?.avg_depth ?? 0;
};

const buildScoreDetails = (entries: Array<[string, number | undefined | null]>): Record<string, number> => {
  return entries.reduce((acc, [key, value]) => {
    if (typeof value === "number" && Number.isFinite(value)) {
      acc[key] = value;
    }
    return acc;
  }, {} as Record<string, number>);
};

// [REQ:SCS-UI-003] Scenario detail view with full breakdown
// [REQ:SCS-UI-007] What-if analysis interactive section
export function ScenarioDetail({ scenario, onBack, onOpenConfig }: ScenarioDetailProps) {
  const queryClient = useQueryClient();
  const [whatIfSelections, setWhatIfSelections] = useState<Record<string, boolean>>({});

  const {
    data: scoreData,
    isLoading: scoreLoading,
    error: scoreError,
  } = useQuery({
    queryKey: ["score-detail", scenario],
    queryFn: () => fetchScoreDetail(scenario),
  });

  const { data: historyData } = useQuery({
    queryKey: ["history", scenario],
    queryFn: () => fetchHistory(scenario, 20),
  });

  const { data: trendData } = useQuery({
    queryKey: ["trends", scenario],
    queryFn: () => fetchTrends(scenario),
  });

  const recalculateMutation = useMutation({
    mutationFn: () => calculateScore(scenario),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["score-detail", scenario] });
      queryClient.invalidateQueries({ queryKey: ["history", scenario] });
      queryClient.invalidateQueries({ queryKey: ["trends", scenario] });
    },
  });

  // Extract history scores for sparkline
  const historyScores =
    historyData?.snapshots?.map((s) => s.score).reverse() ?? [];

  if (scoreLoading) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-50 flex items-center justify-center">
        <RefreshCw className="h-8 w-8 animate-spin text-slate-400" />
      </div>
    );
  }

  if (scoreError || !scoreData) {
    return (
      <div className="min-h-screen bg-slate-950 text-slate-50 p-6">
        <div className="max-w-4xl mx-auto">
          <Button variant="outline" onClick={onBack}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div className="mt-8 p-6 rounded-xl border border-red-500/30 bg-red-500/10">
            <p className="text-red-300">
              Failed to load scenario: {scenario}
            </p>
            <p className="text-sm text-red-300/70 mt-1">
              {scoreError instanceof Error
                ? scoreError.message
                : "Unknown error"}
            </p>
          </div>
        </div>
      </div>
    );
  }

  const { breakdown, metrics, recommendations } = scoreData;

  // ASSUMPTION: breakdown fields may be missing or have undefined nested properties
  // HARDENED: Use safe accessors to prevent runtime crashes on malformed API responses
  const qualityDetails = buildScoreDetails([
    ["requirement_pass_rate", safeRate(breakdown?.quality?.requirement_pass_rate) * 100],
    ["target_pass_rate", safeRate(breakdown?.quality?.target_pass_rate) * 100],
    ["test_pass_rate", safeRate(breakdown?.quality?.test_pass_rate) * 100],
  ]);
  const coverageDetails = buildScoreDetails([
    ["test_coverage_ratio", safeRatio(breakdown?.coverage?.test_coverage_ratio) * 100],
    ["depth_avg", safeAvgDepth(breakdown?.coverage?.depth_score)],
    ["depth_points", safePoints(breakdown?.coverage?.depth_score)],
  ]);
  const quantityDetails = buildScoreDetails([
    ["requirements", safePoints(breakdown?.quantity?.requirements)],
    ["targets", safePoints(breakdown?.quantity?.targets)],
    ["tests", safePoints(breakdown?.quantity?.tests)],
  ]);
  const uiDetails = buildScoreDetails([
    ["template_points", safePoints(breakdown?.ui?.template_check)],
    ["component_complexity", safePoints(breakdown?.ui?.component_complexity)],
    ["api_integration", safePoints(breakdown?.ui?.api_integration)],
    ["routing", safePoints(breakdown?.ui?.routing)],
    ["code_volume", safePoints(breakdown?.ui?.code_volume)],
  ]);

  const penaltyValue = -(breakdown.validation_penalty ?? 0);
  const penaltyDetails = (() => {
    const details: Record<string, number> = {};
    if (penaltyValue < 0) {
      details.validation_penalty = penaltyValue;
    }
    const analysis = scoreData.validation_analysis;
    if (analysis) {
      if ((analysis.total_penalty ?? 0) > 0) {
        details.analysis_penalty = -(analysis.total_penalty ?? 0);
      }
      if ((analysis.issue_count ?? 0) > 0) {
        details.issue_count = analysis.issue_count ?? 0;
      }
    }
    return details;
  })();
  const hasPenaltyEntries = penaltyValue < 0 || Object.keys(penaltyDetails).length > 0;

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6" data-testid="scenario-detail">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button variant="outline" size="sm" onClick={onBack}>
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold">{scenario}</h1>
              <p className="text-slate-400 text-sm">
                Category: {scoreData.category}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <ScoreClassificationBadge
              classification={breakdown.classification}
              score={breakdown.score}
            />
            {trendData && (
              <TrendIndicator
                trend={trendData.analysis.trend}
                delta={trendData.analysis.delta}
              />
            )}
            {onOpenConfig && (
              <Button variant="outline" size="sm" onClick={onOpenConfig} data-testid="configure-button">
                <Settings className="h-4 w-4 mr-2" />
                Configure
              </Button>
            )}
          </div>
        </div>

        {/* Partial Data Banner - [REQ:SCS-CORE-004] Show when score is based on partial data */}
        {scoreData?.partial_result && !scoreData.partial_result.is_complete && (
          <div className="mb-6 p-4 rounded-xl border border-yellow-500/30 bg-yellow-500/10" data-testid="partial-data-banner">
            <div className="flex items-start gap-3">
              <AlertTriangle className="h-5 w-5 text-yellow-400 mt-0.5" />
              <div className="flex-1">
                <p className="font-medium text-yellow-200">Score Based on Partial Data</p>
                <p className="text-sm text-yellow-300/70 mt-1">
                  {scoreData.partial_result.message}
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {scoreData.partial_result.missing.map((collector) => (
                    <span
                      key={collector}
                      className="text-xs px-2 py-1 rounded bg-yellow-500/20 text-yellow-400"
                    >
                      {collector} unavailable
                    </span>
                  ))}
                </div>
                <p className="text-xs text-yellow-300/50 mt-2">
                  Confidence: {Math.round(scoreData.partial_result.confidence * 100)}% — Some dimensions may be incomplete.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Score Overview Card */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
          {/* Main Score */}
          <div className="md:col-span-2 rounded-xl border border-white/10 bg-white/5 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold flex items-center gap-2">
                <BarChart3 className="h-5 w-5 text-slate-400" />
                Score Breakdown
              </h2>
              <Button
                variant="outline"
                size="sm"
                onClick={() => recalculateMutation.mutate()}
                disabled={recalculateMutation.isPending}
              >
                <RefreshCw
                  className={`h-4 w-4 mr-2 ${
                    recalculateMutation.isPending ? "animate-spin" : ""
                  }`}
                />
                Recalculate
              </Button>
            </div>

            <p className="text-xs text-slate-500 mb-4">Click on any dimension to see detailed breakdown</p>
            <div className="space-y-4">
              <ScoreBar
                label="Quality"
                value={breakdown.quality.score}
                max={breakdown.quality.max}
                details={qualityDetails}
              />
              <ScoreBar
                label="Coverage"
                value={breakdown.coverage.score}
                max={breakdown.coverage.max}
                details={coverageDetails}
              />
              <ScoreBar
                label="Quantity"
                value={breakdown.quantity.score}
                max={breakdown.quantity.max}
                details={quantityDetails}
              />
              <ScoreBar
                label="UI"
                value={breakdown.ui.score}
                max={breakdown.ui.max}
                details={uiDetails}
              />
              {hasPenaltyEntries && (
                <ScoreBar
                  label="Penalties"
                  value={penaltyValue}
                  max={0}
                  details={penaltyDetails}
                />
              )}
            </div>

            <div className="mt-6 pt-4 border-t border-white/10 flex items-center justify-between">
              <span className="text-slate-400">Total Score</span>
              <span className="text-3xl font-bold">{breakdown.score}</span>
            </div>
          </div>

          {/* History Sparkline */}
          <div className="rounded-xl border border-white/10 bg-white/5 p-6">
            <h3 className="text-sm font-medium text-slate-400 mb-3 flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              History (last 20)
            </h3>
            <div className="flex items-center justify-center h-24">
              <Sparkline data={historyScores} width={180} height={64} />
            </div>
            {historyData && historyData.count > 0 && (
              <p className="text-xs text-slate-500 text-center mt-3">
                {historyData.total} total snapshots
              </p>
            )}
          </div>
        </div>

        {/* Metrics Summary */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <MetricCard
            label="Requirements"
            passing={metrics.requirements.passing}
            total={metrics.requirements.total}
          />
          <MetricCard
            label="Targets"
            passing={metrics.targets.passing}
            total={metrics.targets.total}
          />
          <MetricCard
            label="Tests"
            passing={metrics.tests.passing}
            total={metrics.tests.total}
          />
        </div>

        {/* Two column layout for recommendations and what-if */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Recommendations */}
          {recommendations && recommendations.length > 0 && (
            <div className="rounded-xl border border-white/10 bg-white/5 p-6" data-testid="recommendations-section">
              <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                <Zap className="h-5 w-5 text-amber-400" />
                Top Recommendations
              </h2>
              <div className="space-y-3">
                {recommendations.slice(0, 5).map((rec: Recommendation, i: number) => (
                  <div
                    key={i}
                    className="flex items-start justify-between p-3 rounded-lg bg-white/5"
                  >
                    <div className="flex items-start gap-3">
                      <span className="flex items-center justify-center w-6 h-6 rounded-full bg-slate-700 text-xs font-medium">
                        {i + 1}
                      </span>
                      <div>
                        <p className="text-sm text-slate-200">{rec.description}</p>
                        <p className="text-xs text-slate-500 mt-0.5">
                          Priority: {rec.priority}
                        </p>
                      </div>
                    </div>
                    <span className="text-emerald-400 font-medium text-sm">
                      +{rec.impact ?? 0} pts
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* What-If Analysis */}
          <WhatIfAnalysis
            currentScore={breakdown.score}
            recommendations={recommendations ?? []}
            selections={whatIfSelections}
            onSelectionChange={setWhatIfSelections}
            onOpenConfig={onOpenConfig}
          />
        </div>

        {/* Footer */}
        <p className="mt-6 text-xs text-slate-500 text-right">
          Last calculated: {new Date(scoreData.calculated_at).toLocaleString()}
        </p>
      </div>
    </div>
  );
}

function MetricCard({
  label,
  passing,
  total,
}: {
  label: string;
  passing: number;
  total: number;
}) {
  const rate = total > 0 ? (passing / total) * 100 : 0;
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4">
      <p className="text-xs text-slate-400">{label}</p>
      <p className="text-2xl font-bold mt-1">
        {passing}/{total}
      </p>
      <p className="text-xs text-slate-500">{rate.toFixed(0)}% passing</p>
    </div>
  );
}

// [REQ:SCS-UI-007] Interactive what-if analysis component
function WhatIfAnalysis({
  currentScore,
  recommendations,
  selections,
  onSelectionChange,
  onOpenConfig,
}: {
  currentScore: number;
  recommendations: Recommendation[];
  selections: Record<string, boolean>;
  onSelectionChange: (s: Record<string, boolean>) => void;
  onOpenConfig?: () => void;
}) {
  // Calculate projected score based on selections
  const projectedScore = useMemo(() => {
    const selectedImpact = recommendations.reduce((sum, rec, i) => {
      const key = `rec-${i}`;
      return selections[key] ? sum + (rec.impact ?? 0) : sum;
    }, 0);
    return Math.min(100, currentScore + selectedImpact);
  }, [currentScore, recommendations, selections]);

  const selectedCount = Object.values(selections).filter(Boolean).length;
  const delta = projectedScore - currentScore;

  const getClassification = (score: number) => {
    if (score >= 90) return "Production Ready";
    if (score >= 80) return "Nearly Ready";
    if (score >= 65) return "Mostly Complete";
    if (score >= 50) return "Functional Incomplete";
    if (score >= 25) return "Foundation Laid";
    return "Initial Concept";
  };

  const toggleSelection = (key: string) => {
    onSelectionChange({
      ...selections,
      [key]: !selections[key],
    });
  };

  const selectAll = () => {
    const newSelections: Record<string, boolean> = {};
    recommendations.slice(0, 5).forEach((_, i) => {
      newSelections[`rec-${i}`] = true;
    });
    onSelectionChange(newSelections);
  };

  const clearAll = () => {
    onSelectionChange({});
  };

  if (!recommendations || recommendations.length === 0) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-6" data-testid="what-if-section">
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Beaker className="h-5 w-5 text-blue-400" />
          What-If Analysis
        </h2>
        <p className="text-sm text-slate-400">No recommendations available for simulation</p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-6" data-testid="what-if-section">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <Beaker className="h-5 w-5 text-blue-400" />
          What-If Analysis
        </h2>
        <div className="flex gap-2">
          <button
            onClick={selectAll}
            className="text-xs text-blue-400 hover:text-blue-300"
            type="button"
          >
            Select all
          </button>
          <span className="text-slate-600">|</span>
          <button
            onClick={clearAll}
            className="text-xs text-slate-400 hover:text-slate-300"
            type="button"
          >
            Clear
          </button>
        </div>
      </div>

      <p className="text-xs text-slate-500 mb-4">
        Select improvements to see projected score
      </p>

      {/* Checkboxes for recommendations */}
      <div className="space-y-2 mb-4">
        {recommendations.slice(0, 5).map((rec, i) => {
          const key = `rec-${i}`;
          const isSelected = selections[key] ?? false;
          return (
            <label
              key={key}
              className="flex items-center gap-3 p-2 rounded-lg hover:bg-white/5 cursor-pointer"
            >
              <input
                type="checkbox"
                checked={isSelected}
                onChange={() => toggleSelection(key)}
                className="w-4 h-4 rounded border-slate-600 bg-slate-700 text-blue-500 focus:ring-blue-500/50"
              />
              <span className="flex-1 text-sm text-slate-300 truncate">
                {rec.description}
              </span>
              <span className="text-xs text-emerald-400 font-medium">
                +{rec.impact ?? 0}
              </span>
            </label>
          );
        })}
      </div>

      {/* Projected result */}
      <div className="p-4 rounded-lg bg-slate-800/50 border border-white/5">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-slate-400">Current Score</span>
          <span className="font-medium">{currentScore}</span>
        </div>
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-slate-400">
            Selected ({selectedCount})
          </span>
          <span className={`font-medium ${delta > 0 ? "text-emerald-400" : "text-slate-400"}`}>
            {delta > 0 ? `+${delta}` : delta}
          </span>
        </div>
        <div className="border-t border-white/10 pt-2 mt-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-slate-300">
              Projected Score
            </span>
            <span className={`text-2xl font-bold ${delta > 0 ? "text-emerald-400" : ""}`}>
              {projectedScore}
            </span>
          </div>
          {delta > 0 && (
            <p className="text-xs text-slate-500 mt-1 text-right">
              New classification: {getClassification(projectedScore)}
            </p>
          )}
        </div>
      </div>

      {/* Configuration tip - shown when user has selections */}
      {selectedCount > 0 && onOpenConfig && (
        <div className="mt-4 p-3 rounded-lg border border-blue-500/20 bg-blue-500/5" data-testid="config-tip">
          <div className="flex items-start gap-2">
            <Lightbulb className="h-4 w-4 text-blue-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-xs text-blue-300/80">
                Some scoring components can be adjusted via configuration. Use presets like "Skip E2E" if certain collectors are unavailable.
              </p>
              <button
                onClick={onOpenConfig}
                className="mt-2 text-xs text-blue-400 hover:text-blue-300 underline underline-offset-2"
                type="button"
              >
                Open Configuration →
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
