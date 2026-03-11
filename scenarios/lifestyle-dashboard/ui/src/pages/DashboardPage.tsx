/**
 * Dashboard Page
 *
 * Main overview page showing timeline, domain summaries, and recent events.
 * This is the primary view for the unified lifestyle intelligence dashboard.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Unified dashboard UI with timeline view
 * [REQ:LD-UI-TRENDS] - Trend charts with 7d/30d/90d selectable periods
 * [REQ:LD-UI-TIMELINE] - Timeline view across all domains
 * [REQ:LD-UI-SCORE] - Daily Lifestyle Score display
 */
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Database, Heart, Clock, TrendingUp, Activity } from "lucide-react";
import { Link } from "react-router-dom";

import {
  DomainCard,
  EventRow,
  TimelineChart,
  StatCard,
  DomainBreakdown,
  LifestyleScoreCard,
  BriefPreview,
} from "../components/dashboard";
import type { TrendPeriod } from "../components/dashboard/TimelineChart";
import { Card, CardHeader, CardTitle } from "../components/ui";
import { ErrorAlert } from "../components/ErrorAlert";

import {
  fetchDomains,
  fetchSummary,
  fetchTimeline,
  fetchEvents,
  fetchScore,
  fetchCurrentBrief,
} from "../lib/api";

import { formatRelativeTime } from "../lib/format";

export default function DashboardPage() {
  // [REQ:LD-UI-TRENDS] - State for timeline period selection (7d/30d/90d)
  const [timelinePeriod, setTimelinePeriod] = useState<TrendPeriod>(7);

  const domainsQuery = useQuery({
    queryKey: ["domains"],
    queryFn: fetchDomains,
    refetchInterval: 60000,
  });

  const summaryQuery = useQuery({
    queryKey: ["summary"],
    queryFn: fetchSummary,
    refetchInterval: 30000,
  });

  // [REQ:LD-UI-TRENDS] - Timeline query uses selected period
  const timelineQuery = useQuery({
    queryKey: ["timeline", timelinePeriod],
    queryFn: () => fetchTimeline(timelinePeriod),
    refetchInterval: 60000,
  });

  const eventsQuery = useQuery({
    queryKey: ["events"],
    queryFn: () => fetchEvents({ limit: 10 }),
    refetchInterval: 30000,
  });

  // [REQ:LD-UI-SCORE] - Fetch lifestyle score for dashboard display
  const scoreQuery = useQuery({
    queryKey: ["score"],
    queryFn: () => fetchScore(7),
    refetchInterval: 60000,
  });

  // [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] - Fetch current brief for preview
  const briefQuery = useQuery({
    queryKey: ["brief", "current"],
    queryFn: fetchCurrentBrief,
    refetchInterval: 60000,
  });

  // Get the first error from any query, with priority to most important
  const error = domainsQuery.error || summaryQuery.error || timelineQuery.error || eventsQuery.error || scoreQuery.error || briefQuery.error;

  const handleRetry = () => {
    domainsQuery.refetch();
    summaryQuery.refetch();
    timelineQuery.refetch();
    eventsQuery.refetch();
    scoreQuery.refetch();
    briefQuery.refetch();
  };

  return (
    <div className="space-y-6">
      {/* Error state with structured error handling */}
      <ErrorAlert
        error={error as Error | null}
        onRetry={handleRetry}
      />

      {/* Top row: Lifestyle Score + Stats [REQ:LD-UI-SCORE] */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Lifestyle Score card */}
        <LifestyleScoreCard
          score={scoreQuery.data?.current ?? null}
          isLoading={scoreQuery.isLoading}
        />

        {/* Stats grid (2 columns on mobile, 2x2 on desktop within this column) */}
        <div className="lg:col-span-2 grid grid-cols-2 gap-4">
        <StatCard
          label="Total Events"
          value={summaryQuery.data?.total_events ?? "-"}
          icon={Database}
        />
        <StatCard
          label="Active Domains"
          value={summaryQuery.data?.active_domains ?? "-"}
          icon={Heart}
        />
        <StatCard
          label="Last Activity"
          value={summaryQuery.data?.last_event_at ? formatRelativeTime(summaryQuery.data.last_event_at) : "-"}
          icon={Clock}
        />
        <StatCard
          label="7-Day Trend"
          value={timelineQuery.data?.timeline?.reduce((sum: number, e: { count: number }) => sum + e.count, 0) ?? "-"}
          icon={TrendingUp}
        />
        </div>
      </div>

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Timeline chart with period selector [REQ:LD-UI-TRENDS] */}
        <Card padding="lg" className="lg:col-span-2">
          <CardTitle className="mb-4">Activity Timeline</CardTitle>
          {timelineQuery.isLoading ? (
            <div className="h-32 flex items-center justify-center text-slate-500">Loading...</div>
          ) : (
            <TimelineChart
              data={timelineQuery.data?.timeline ?? []}
              period={timelinePeriod}
              onPeriodChange={setTimelinePeriod}
              showPeriodSelector
            />
          )}
        </Card>

        {/* Daily brief preview [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] */}
        <BriefPreview
          brief={briefQuery.data?.brief ?? null}
          isLoading={briefQuery.isLoading}
          error={briefQuery.error as Error | null}
        />
      </div>

      {/* Three-column layout for domains breakdown, registered domains, and events */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Domain breakdown */}
        <Card padding="lg">
          <CardTitle className="mb-4">Events by Domain</CardTitle>
          <DomainBreakdown data={summaryQuery.data?.events_by_domain ?? []} />
        </Card>

        {/* Registered domains */}
        <Card padding="lg">
          <CardHeader>
            <CardTitle>Registered Domains</CardTitle>
            <Link
              to="/domains"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </CardHeader>
          {domainsQuery.isLoading ? (
            <div className="text-slate-500">Loading...</div>
          ) : domainsQuery.data?.domains && domainsQuery.data.domains.length > 0 ? (
            <div className="space-y-4">
              {domainsQuery.data.domains.slice(0, 3).map((domain) => (
                <Link key={domain.name} to={`/domains/${domain.name}`}>
                  <DomainCard domain={domain} />
                </Link>
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <Heart className="w-12 h-12 mx-auto text-slate-700 mb-3" />
              <p className="text-slate-400">No domains registered yet</p>
              <p className="text-sm text-slate-500 mt-1">
                Domain scenarios will appear here when they connect
              </p>
            </div>
          )}
        </Card>

        {/* Recent events */}
        <Card padding="lg">
          <CardHeader>
            <CardTitle>Recent Events</CardTitle>
            <Link
              to="/events"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </CardHeader>
          {eventsQuery.isLoading ? (
            <div className="text-slate-500">Loading...</div>
          ) : eventsQuery.data?.events && eventsQuery.data.events.length > 0 ? (
            <div className="divide-y divide-white/5">
              {eventsQuery.data.events.slice(0, 6).map((event) => (
                <EventRow key={event.id} event={event} />
              ))}
            </div>
          ) : (
            <div className="text-center py-8">
              <Activity className="w-12 h-12 mx-auto text-slate-700 mb-3" />
              <p className="text-slate-400">No events recorded yet</p>
              <p className="text-sm text-slate-500 mt-1">
                Events from domain scenarios will appear here
              </p>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
