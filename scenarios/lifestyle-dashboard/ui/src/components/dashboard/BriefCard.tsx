/**
 * Brief Card Component
 *
 * Displays a morning or evening brief with domain sections and score.
 *
 * [REQ:LD-BRIEF-MORNING] - Morning brief display
 * [REQ:LD-BRIEF-EVENING] - Evening review display
 * [REQ:LD-BRIEF-CONSOLIDATE] - Cross-domain consolidation view
 */
import { Sun, Moon, TrendingUp, TrendingDown, Minus, Activity } from "lucide-react";
import type { Brief, BriefSection } from "../../lib/api";
import { Card, CardHeader, CardTitle } from "../ui";
import { formatRelativeTime } from "../../lib/format";
import { DATA_SELECTORS } from "../../consts/selectors";

interface BriefCardProps {
  brief: Brief | null;
  isLoading: boolean;
}

function BriefSectionItem({ section }: { section: BriefSection }) {
  const priorityColors = {
    1: "border-l-green-500",
    2: "border-l-yellow-500",
    3: "border-l-gray-500",
  };

  const priorityColor = priorityColors[section.priority as 1 | 2 | 3] ?? priorityColors[3];

  return (
    <div
      className={`border-l-2 ${priorityColor} pl-3 py-2`}
      data-testid={DATA_SELECTORS.BRIEF_SECTION}
    >
      <div className="flex items-center justify-between mb-1">
        <span className="font-medium text-white">{section.display_name}</span>
        <span className="text-xs text-gray-400">
          {section.event_count} event{section.event_count !== 1 ? "s" : ""}
        </span>
      </div>
      <ul className="text-sm text-gray-300 space-y-0.5">
        {section.items.map((item, idx) => (
          <li key={idx} className="flex items-start">
            <span className="mr-2 text-gray-500">&bull;</span>
            {item}
          </li>
        ))}
        {section.items.length === 0 && (
          <li className="text-gray-500 italic">No activity</li>
        )}
      </ul>
    </div>
  );
}

export default function BriefCard({ brief, isLoading }: BriefCardProps) {
  if (isLoading) {
    return (
      <Card data-testid={DATA_SELECTORS.BRIEF_CARD}>
        <div className="animate-pulse space-y-4">
          <div className="h-6 bg-white/10 rounded w-1/3" />
          <div className="h-4 bg-white/10 rounded w-2/3" />
          <div className="space-y-2">
            <div className="h-16 bg-white/10 rounded" />
            <div className="h-16 bg-white/10 rounded" />
          </div>
        </div>
      </Card>
    );
  }

  if (!brief) {
    return (
      <Card data-testid={DATA_SELECTORS.BRIEF_CARD}>
        <div className="text-center text-gray-400 py-8">
          <Activity className="w-8 h-8 mx-auto mb-2 opacity-50" />
          <p>Unable to load brief</p>
        </div>
      </Card>
    );
  }

  const isMorning = brief.type === "morning";
  const Icon = isMorning ? Sun : Moon;
  const title = isMorning ? "Morning Brief" : "Evening Review";

  const TrendIcon = brief.score_trend === "up"
    ? TrendingUp
    : brief.score_trend === "down"
    ? TrendingDown
    : Minus;

  const trendColor = brief.score_trend === "up"
    ? "text-green-400"
    : brief.score_trend === "down"
    ? "text-red-400"
    : "text-gray-400";

  return (
    <Card data-testid={DATA_SELECTORS.BRIEF_CARD}>
      <CardHeader className="mb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Icon className={`w-5 h-5 ${isMorning ? "text-yellow-400" : "text-indigo-400"}`} />
            <CardTitle>{title}</CardTitle>
          </div>
          <span className="text-xs text-gray-400">
            {formatRelativeTime(brief.generated_at)}
          </span>
        </div>
      </CardHeader>

      {/* Summary */}
      <p className="text-gray-300 mb-4" data-testid={DATA_SELECTORS.BRIEF_SUMMARY}>
        {brief.summary}
      </p>

      {/* Score (if available) */}
      {brief.score !== undefined && brief.score !== null && (
        <div
          className="flex items-center gap-3 mb-4 p-3 bg-white/5 rounded-lg"
          data-testid={DATA_SELECTORS.BRIEF_SCORE}
        >
          <div className="text-2xl font-bold text-white">{brief.score}</div>
          <div className="flex-1">
            <div className="text-sm text-gray-400">Lifestyle Score</div>
            <div className={`flex items-center gap-1 text-sm ${trendColor}`}>
              <TrendIcon className="w-4 h-4" />
              <span className="capitalize">{brief.score_trend ?? "stable"}</span>
            </div>
          </div>
        </div>
      )}

      {/* Domain Sections */}
      <div className="space-y-3" data-testid={DATA_SELECTORS.BRIEF_SECTIONS}>
        {brief.sections.length > 0 ? (
          brief.sections.map((section) => (
            <BriefSectionItem key={section.domain} section={section} />
          ))
        ) : (
          <div className="text-center text-gray-500 py-4">
            No domain activity to show
          </div>
        )}
      </div>

      {/* Footer with date */}
      <div className="mt-4 pt-3 border-t border-white/10 text-xs text-gray-500 text-center">
        Brief for {brief.date}
      </div>
    </Card>
  );
}
