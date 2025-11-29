import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  RefreshCw,
  TrendingUp,
  Zap,
  BarChart3,
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
}

// [REQ:SCS-UI-003] Scenario detail view with full breakdown
export function ScenarioDetail({ scenario, onBack }: ScenarioDetailProps) {
  const queryClient = useQueryClient();

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
          </div>
        </div>

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

            <div className="space-y-4">
              <ScoreBar
                label="Quality"
                value={breakdown.quality}
                max={50}
              />
              <ScoreBar
                label="Coverage"
                value={breakdown.coverage}
                max={15}
              />
              <ScoreBar
                label="Quantity"
                value={breakdown.quantity}
                max={10}
              />
              <ScoreBar
                label="UI"
                value={breakdown.ui}
                max={25}
              />
              {breakdown.penalties < 0 && (
                <div className="pt-2 border-t border-white/10">
                  <div className="flex justify-between text-sm">
                    <span className="text-red-400">Penalties</span>
                    <span className="text-red-400">{breakdown.penalties}</span>
                  </div>
                </div>
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
                      <p className="text-sm text-slate-200">{rec.action}</p>
                      <p className="text-xs text-slate-500 mt-0.5">
                        Area: {rec.area}
                      </p>
                    </div>
                  </div>
                  <span className="text-emerald-400 font-medium text-sm">
                    +{rec.impact} pts
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

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
