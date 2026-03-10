/**
 * Dashboard Page
 *
 * Main overview page showing timeline, domain summaries, and recent events.
 * This is the primary view for the unified lifestyle intelligence dashboard.
 *
 * [REQ:LD-DASHBOARD-TIMELINE] - Unified dashboard UI with timeline view
 */
import { useQuery } from "@tanstack/react-query";
import { Database, Heart, Clock, TrendingUp, AlertCircle, Activity } from "lucide-react";
import { Link } from "react-router-dom";

import {
  DomainCard,
  EventRow,
  TimelineChart,
  StatCard,
  DomainBreakdown,
} from "../components/dashboard";

import {
  fetchDomains,
  fetchSummary,
  fetchTimeline,
  fetchEvents,
} from "../lib/api";

import { formatRelativeTime } from "../lib/format";

export default function DashboardPage() {
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

  const timelineQuery = useQuery({
    queryKey: ["timeline"],
    queryFn: () => fetchTimeline(7),
    refetchInterval: 60000,
  });

  const eventsQuery = useQuery({
    queryKey: ["events"],
    queryFn: () => fetchEvents({ limit: 10 }),
    refetchInterval: 30000,
  });

  const hasError = domainsQuery.error || summaryQuery.error;

  return (
    <div className="space-y-6">
      {/* Error state */}
      {hasError && (
        <div className="rounded-xl border border-red-500/30 bg-red-500/10 p-4">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-5 h-5 text-red-400" />
            <p className="text-red-400">
              Unable to connect to the API. Make sure the scenario is running.
            </p>
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Run: <code className="px-2 py-0.5 rounded bg-slate-800 text-slate-300">vrooli scenario start lifestyle-dashboard</code>
          </p>
        </div>
      )}

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
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

      {/* Main content grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Timeline chart */}
        <div className="lg:col-span-2 rounded-xl border border-white/10 bg-white/5 p-6">
          <h2 className="text-lg font-medium mb-4">Activity Timeline</h2>
          {timelineQuery.isLoading ? (
            <div className="h-32 flex items-center justify-center text-slate-500">Loading...</div>
          ) : (
            <TimelineChart data={timelineQuery.data?.timeline ?? []} />
          )}
        </div>

        {/* Domain breakdown */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <h2 className="text-lg font-medium mb-4">Events by Domain</h2>
          <DomainBreakdown data={summaryQuery.data?.events_by_domain ?? []} />
        </div>
      </div>

      {/* Two-column layout for domains and events */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Registered domains */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium">Registered Domains</h2>
            <Link
              to="/domains"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </div>
          {domainsQuery.isLoading ? (
            <div className="text-slate-500">Loading...</div>
          ) : domainsQuery.data?.domains && domainsQuery.data.domains.length > 0 ? (
            <div className="space-y-4">
              {domainsQuery.data.domains.slice(0, 4).map((domain) => (
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
        </div>

        {/* Recent events */}
        <div className="rounded-xl border border-white/10 bg-white/5 p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium">Recent Events</h2>
            <Link
              to="/events"
              className="text-sm text-blue-400 hover:text-blue-300 transition-colors"
            >
              View all
            </Link>
          </div>
          {eventsQuery.isLoading ? (
            <div className="text-slate-500">Loading...</div>
          ) : eventsQuery.data?.events && eventsQuery.data.events.length > 0 ? (
            <div className="divide-y divide-white/5">
              {eventsQuery.data.events.map((event) => (
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
        </div>
      </div>
    </div>
  );
}
