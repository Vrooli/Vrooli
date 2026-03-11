/**
 * Briefs Page
 *
 * Displays daily briefs - morning and evening summaries across all domains.
 * Also includes weekly digest showing "What Changed?" comparisons.
 *
 * [REQ:LD-BRIEF-MORNING] - Morning brief with yesterday summary
 * [REQ:LD-BRIEF-EVENING] - Evening review with today's events
 * [REQ:LD-BRIEF-CONSOLIDATE] - Cross-domain consolidation
 * [REQ:LD-DIGEST-WEEKLY] - Weekly digest with baseline comparison
 */
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Sun, Moon, Calendar, RefreshCw, TrendingUp, TrendingDown, Minus, ChevronLeft, BarChart2 } from "lucide-react";
import { Link } from "react-router-dom";

import { BriefCard } from "../components/dashboard";
import { ErrorAlert } from "../components/ErrorAlert";
import { Card } from "../components/ui";
import { DATA_SELECTORS } from "../consts/selectors";

import {
  fetchCurrentBrief,
  fetchMorningBrief,
  fetchEveningBrief,
  fetchCurrentDigest,
  type WeeklyDigest,
} from "../lib/api";
import { formatDate } from "../lib/format";

type BriefTab = "current" | "morning" | "evening" | "weekly";

/**
 * WeeklyDigestCard displays the weekly "What Changed?" summary.
 * Shows score trends, domain activity changes, highlights, and focus areas.
 */
function WeeklyDigestCard({ digest, isLoading }: { digest: WeeklyDigest | null; isLoading: boolean }) {
  if (isLoading) {
    return (
      <Card padding="lg" className="animate-pulse">
        <div className="h-6 bg-white/10 rounded w-1/3 mb-4" />
        <div className="h-4 bg-white/10 rounded w-2/3 mb-2" />
        <div className="h-4 bg-white/10 rounded w-1/2" />
      </Card>
    );
  }

  if (!digest) {
    return (
      <Card padding="lg" className="text-center text-gray-400 py-12">
        <BarChart2 className="w-12 h-12 mx-auto mb-4 text-slate-700" />
        <p className="font-medium">No weekly digest available</p>
        <p className="text-sm text-gray-500 mt-1">
          Weekly digests are generated every Sunday
        </p>
      </Card>
    );
  }

  const TrendIcon = digest.score_trend.direction === "up" ? TrendingUp
    : digest.score_trend.direction === "down" ? TrendingDown
    : Minus;

  const trendColor = digest.score_trend.direction === "up" ? "text-green-400"
    : digest.score_trend.direction === "down" ? "text-red-400"
    : "text-gray-400";

  return (
    <Card padding="lg" data-testid="weekly-digest-card">
      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <BarChart2 className="w-5 h-5 text-purple-400" />
            <h2 className="text-lg font-medium">Weekly Digest</h2>
          </div>
          <p className="text-sm text-gray-400">
            {formatDate(digest.week_start_date)} - {formatDate(digest.week_end_date)}
          </p>
        </div>
      </div>

      {/* Summary */}
      <p className="text-gray-300 mb-6">{digest.summary}</p>

      {/* Score Trend */}
      <div className="rounded-lg bg-white/5 p-4 mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-gray-400">Lifestyle Score</span>
          <div className={`flex items-center gap-1 ${trendColor}`}>
            <TrendIcon className="w-4 h-4" />
            <span className="text-sm font-medium">
              {digest.score_trend.percent_change > 0 ? "+" : ""}
              {digest.score_trend.percent_change.toFixed(1)}%
            </span>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <div>
            <p className="text-2xl font-bold">{digest.score_trend.current_week_avg.toFixed(0)}</p>
            <p className="text-xs text-gray-500">This week</p>
          </div>
          <div className="text-gray-600">→</div>
          <div>
            <p className="text-xl text-gray-400">{digest.score_trend.baseline_avg.toFixed(0)}</p>
            <p className="text-xs text-gray-500">4-week avg</p>
          </div>
        </div>
        <p className="text-sm text-gray-400 mt-2">{digest.score_trend.message}</p>
      </div>

      {/* Domain Changes */}
      {digest.domain_changes.length > 0 && (
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-400 mb-3">Domain Activity</h3>
          <div className="space-y-2">
            {digest.domain_changes.map((change) => {
              const changeIcon = change.direction === "up" ? TrendingUp
                : change.direction === "down" ? TrendingDown
                : Minus;
              const changeColor = change.direction === "up" ? "text-green-400"
                : change.direction === "down" ? "text-red-400"
                : "text-gray-400";
              const ChangeIcon = changeIcon;

              return (
                <div
                  key={change.domain}
                  className={`flex items-center justify-between py-2 px-3 rounded-lg ${change.notable ? "bg-white/5" : ""}`}
                >
                  <div>
                    <span className="font-medium">{change.display_name}</span>
                    <span className="text-sm text-gray-500 ml-2">
                      {change.current_week_events} events
                    </span>
                  </div>
                  <div className={`flex items-center gap-1 ${changeColor}`}>
                    <ChangeIcon className="w-3 h-3" />
                    <span className="text-sm">
                      {change.percent_change > 0 ? "+" : ""}
                      {change.percent_change.toFixed(0)}%
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Highlights */}
      {digest.highlights.length > 0 && (
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-400 mb-3">Highlights</h3>
          <ul className="space-y-2">
            {digest.highlights.map((highlight, i) => (
              <li key={i} className="flex items-start gap-2 text-sm">
                <span className="text-green-400">•</span>
                <span className="text-gray-300">{highlight}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Next Week Focus */}
      {digest.next_week_focus.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-gray-400 mb-3">Focus for Next Week</h3>
          <ul className="space-y-2">
            {digest.next_week_focus.map((focus, i) => (
              <li key={i} className="flex items-start gap-2 text-sm">
                <span className="text-blue-400">→</span>
                <span className="text-gray-300">{focus}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </Card>
  );
}

export default function BriefsPage() {
  const [activeTab, setActiveTab] = useState<BriefTab>("current");
  const [selectedDate, setSelectedDate] = useState(() => {
    return new Date().toISOString().split("T")[0];
  });

  const currentQuery = useQuery({
    queryKey: ["brief", "current"],
    queryFn: fetchCurrentBrief,
    refetchInterval: 60000,
    enabled: activeTab === "current",
  });

  const morningQuery = useQuery({
    queryKey: ["brief", "morning", selectedDate],
    queryFn: () => fetchMorningBrief(selectedDate),
    enabled: activeTab === "morning",
  });

  const eveningQuery = useQuery({
    queryKey: ["brief", "evening", selectedDate],
    queryFn: () => fetchEveningBrief(selectedDate),
    enabled: activeTab === "evening",
  });

  // [REQ:LD-DIGEST-WEEKLY] Weekly digest query
  const digestQuery = useQuery({
    queryKey: ["digest", "current"],
    queryFn: fetchCurrentDigest,
    refetchInterval: 300000, // 5 minutes
    enabled: activeTab === "weekly",
  });

  const activeQuery = activeTab === "current"
    ? currentQuery
    : activeTab === "morning"
    ? morningQuery
    : activeTab === "evening"
    ? eveningQuery
    : digestQuery;

  const handleRefresh = () => {
    activeQuery.refetch();
  };

  const tabs: { id: BriefTab; label: string; icon: typeof Sun }[] = [
    { id: "current", label: "Current", icon: Calendar },
    { id: "morning", label: "Morning", icon: Sun },
    { id: "evening", label: "Evening", icon: Moon },
    { id: "weekly", label: "Weekly", icon: BarChart2 },
  ];

  return (
    <div className="space-y-6" data-testid={DATA_SELECTORS.BRIEFS_PAGE}>
      {/* Header with back button for navigation consistency */}
      <div className="flex items-center gap-4">
        <Link to="/" className="text-slate-400 hover:text-white transition-colors">
          <ChevronLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-white">Briefs & Digests</h1>
          <p className="text-gray-400 mt-1">
            Daily summaries and weekly insights across all domains
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={activeQuery.isFetching}
          className="flex items-center gap-2 px-3 py-2 text-sm bg-white/10 hover:bg-white/15 rounded-lg transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-4 h-4 ${activeQuery.isFetching ? "animate-spin" : ""}`} />
          Refresh
        </button>
      </div>

      {/* Error state */}
      <ErrorAlert
        error={activeQuery.error as Error | null}
        onRetry={handleRefresh}
      />

      {/* Tab navigation */}
      <div className="flex gap-2 flex-wrap">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            data-testid={
              id === "morning" ? DATA_SELECTORS.BRIEF_MORNING_TAB
              : id === "evening" ? DATA_SELECTORS.BRIEF_EVENING_TAB
              : id === "weekly" ? "brief-weekly-tab"
              : undefined
            }
            className={`flex items-center gap-2 px-4 py-2 rounded-lg transition-colors ${
              activeTab === id
                ? id === "weekly" ? "bg-purple-600 text-white" : "bg-blue-600 text-white"
                : "bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white"
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Date picker for morning/evening tabs */}
      {(activeTab === "morning" || activeTab === "evening") && (
        <Card className="flex items-center gap-3 p-3">
          <Calendar className="w-4 h-4 text-gray-400" />
          <label className="text-sm text-gray-400">Select date:</label>
          <input
            type="date"
            value={selectedDate}
            onChange={(e) => {
              const dateValue = e.target.value;
              if (dateValue) {
                setSelectedDate(dateValue);
              }
            }}
            max={new Date().toISOString().split("T")[0]}
            className="bg-white/5 border border-white/10 rounded px-3 py-1 text-white text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </Card>
      )}

      {/* Content based on active tab */}
      {activeTab === "weekly" ? (
        <WeeklyDigestCard
          digest={digestQuery.data?.digest ?? null}
          isLoading={digestQuery.isLoading}
        />
      ) : (
        <BriefCard
          brief={activeQuery.data && "brief" in activeQuery.data ? activeQuery.data.brief : null}
          isLoading={activeQuery.isLoading}
        />
      )}

      {/* Config info for daily briefs */}
      {activeTab !== "weekly" && currentQuery.data?.config && (
        <Card className="text-xs text-gray-500 flex items-center justify-center gap-4 p-3">
          <span className="flex items-center gap-1">
            <Sun className="w-3 h-3" />
            Morning: {currentQuery.data.config.morning_hour}:00
          </span>
          <span className="flex items-center gap-1">
            <Moon className="w-3 h-3" />
            Evening: {currentQuery.data.config.evening_hour}:00
          </span>
        </Card>
      )}
    </div>
  );
}
