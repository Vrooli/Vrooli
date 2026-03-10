/**
 * LifestyleScoreCard displays the daily composite lifestyle score.
 * Shows score (0-100), trend indicator, and per-domain breakdown.
 *
 * [REQ:LD-UI-SCORE] Daily Lifestyle Score display on dashboard
 */
import { TrendingUp, TrendingDown, Minus, Activity } from "lucide-react";
import { Card } from "../ui";
import type { LifestyleScore, DomainScore } from "../../lib/api";
import { DATA_SELECTORS } from "../../consts/selectors";

interface LifestyleScoreCardProps {
  score: LifestyleScore | null;
  isLoading?: boolean;
}

/**
 * Returns the color class for a score value.
 */
function getScoreColor(score: number): string {
  if (score >= 80) return "text-emerald-400";
  if (score >= 60) return "text-green-400";
  if (score >= 40) return "text-yellow-400";
  if (score >= 20) return "text-orange-400";
  return "text-slate-400";
}

/**
 * Returns the trend icon and color.
 */
function getTrendIndicator(trend: string) {
  switch (trend) {
    case "up":
      return { Icon: TrendingUp, color: "text-emerald-400" };
    case "down":
      return { Icon: TrendingDown, color: "text-red-400" };
    default:
      return { Icon: Minus, color: "text-slate-400" };
  }
}

/**
 * Renders a single domain's contribution to the score.
 */
function DomainScoreRow({ domain }: { domain: DomainScore }) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-sm text-slate-400 truncate max-w-[120px]">
        {domain.display_name}
      </span>
      <div className="flex items-center gap-2">
        <span className="text-xs text-slate-500">
          {domain.event_count} events
        </span>
        <span className={`text-sm font-medium ${getScoreColor(domain.score)}`}>
          {domain.score}
        </span>
      </div>
    </div>
  );
}

export function LifestyleScoreCard({ score, isLoading }: LifestyleScoreCardProps) {
  if (isLoading) {
    return (
      <Card padding="lg" data-testid={DATA_SELECTORS.LIFESTYLE_SCORE}>
        <div className="animate-pulse">
          <div className="h-4 bg-slate-700 rounded w-24 mb-4" />
          <div className="h-16 bg-slate-700 rounded w-20 mb-2" />
          <div className="h-4 bg-slate-700 rounded w-32" />
        </div>
      </Card>
    );
  }

  if (!score) {
    return (
      <Card padding="lg" data-testid={DATA_SELECTORS.LIFESTYLE_SCORE}>
        <div className="text-center py-4">
          <Activity className="w-8 h-8 mx-auto text-slate-600 mb-2" />
          <p className="text-slate-400 text-sm">Score unavailable</p>
        </div>
      </Card>
    );
  }

  const { Icon: TrendIcon, color: trendColor } = getTrendIndicator(score.trend);
  const hasActiveDomains = (score.domain_scores?.length ?? 0) > 0;

  return (
    <Card padding="lg" data-testid={DATA_SELECTORS.LIFESTYLE_SCORE}>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-slate-300">Lifestyle Score</h3>
        <span className={`text-xs px-2 py-0.5 rounded ${
          score.data_quality === "good"
            ? "bg-emerald-500/20 text-emerald-400"
            : score.data_quality === "limited"
            ? "bg-yellow-500/20 text-yellow-400"
            : "bg-slate-500/20 text-slate-400"
        }`}>
          {score.data_quality === "good"
            ? "Good data"
            : score.data_quality === "limited"
            ? "Limited data"
            : "Needs data"}
        </span>
      </div>

      {/* Main score */}
      <div className="flex items-end gap-3 mb-3">
        <span
          className={`text-5xl font-bold ${getScoreColor(score.score)}`}
          data-testid={DATA_SELECTORS.LIFESTYLE_SCORE_VALUE}
        >
          {score.score}
        </span>
        <div className="flex items-center gap-1 pb-2">
          <TrendIcon className={`w-4 h-4 ${trendColor}`} />
          {score.change_from_yesterday !== 0 && (
            <span className={`text-sm ${trendColor}`}>
              {score.change_from_yesterday > 0 ? "+" : ""}{score.change_from_yesterday}
            </span>
          )}
        </div>
      </div>

      {/* Message */}
      <p className="text-sm text-slate-400 mb-4">
        {score.message}
      </p>

      {/* Domain breakdown */}
      {hasActiveDomains && (
        <div className="border-t border-white/10 pt-3">
          <p className="text-xs text-slate-500 mb-2">Today by domain</p>
          <div className="space-y-0.5">
            {score.domain_scores?.map((domain) => (
              <DomainScoreRow key={domain.domain} domain={domain} />
            ))}
          </div>
        </div>
      )}
    </Card>
  );
}
