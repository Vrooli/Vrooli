import { useQuery } from "@tanstack/react-query";
import { Activity, Database, Heart, RefreshCw, Clock, TrendingUp, AlertCircle, CheckCircle2 } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth, fetchDomains, fetchSummary, fetchTimeline, fetchEvents, type Domain, type Event, type Summary, type TimelineEntry } from "./lib/api";
import { useState } from "react";

// Format relative time
function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

// Status badge component
function StatusBadge({ status }: { status: string }) {
  const styles = {
    healthy: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
    active: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
    degraded: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    inactive: "bg-slate-500/20 text-slate-400 border-slate-500/30",
    unhealthy: "bg-red-500/20 text-red-400 border-red-500/30",
  }[status] || "bg-slate-500/20 text-slate-400 border-slate-500/30";

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${styles}`}>
      {status}
    </span>
  );
}

// Domain card component
function DomainCard({ domain }: { domain: Domain }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 hover:bg-white/[0.07] transition-colors">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-violet-500/30 to-fuchsia-500/30 flex items-center justify-center">
            <Heart className="w-5 h-5 text-violet-400" />
          </div>
          <div>
            <h3 className="font-medium text-slate-100">{domain.display_name}</h3>
            <p className="text-xs text-slate-500">{domain.name}</p>
          </div>
        </div>
        <StatusBadge status={domain.status} />
      </div>
      {domain.description && (
        <p className="mt-3 text-sm text-slate-400 line-clamp-2">{domain.description}</p>
      )}
      {domain.capabilities && domain.capabilities.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1">
          {domain.capabilities.slice(0, 3).map((cap) => (
            <span key={cap} className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-400">
              {cap}
            </span>
          ))}
          {domain.capabilities.length > 3 && (
            <span className="text-xs px-2 py-0.5 rounded bg-slate-800 text-slate-400">
              +{domain.capabilities.length - 3}
            </span>
          )}
        </div>
      )}
      {domain.last_health_at && (
        <p className="mt-3 text-xs text-slate-500">
          Last check: {formatRelativeTime(domain.last_health_at)}
        </p>
      )}
    </div>
  );
}

// Event row component
function EventRow({ event }: { event: Event }) {
  return (
    <div className="flex items-center gap-4 py-3 border-b border-white/5 last:border-0">
      <div className="w-2 h-2 rounded-full bg-violet-500 flex-shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-slate-200 truncate">{event.event_type}</span>
          {event.is_intervention && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-400">intervention</span>
          )}
        </div>
        <p className="text-xs text-slate-500">{event.domain}</p>
      </div>
      <span className="text-xs text-slate-500 flex-shrink-0">{formatRelativeTime(event.timestamp)}</span>
    </div>
  );
}

// Timeline chart (simple bar visualization)
function TimelineChart({ data }: { data: TimelineEntry[] }) {
  // Group by day
  const byDay = data.reduce((acc, entry) => {
    if (!acc[entry.day]) acc[entry.day] = 0;
    acc[entry.day] += entry.count;
    return acc;
  }, {} as Record<string, number>);

  const days = Object.keys(byDay).sort();
  const maxCount = Math.max(...Object.values(byDay), 1);

  if (days.length === 0) {
    return (
      <div className="flex items-center justify-center h-32 text-slate-500 text-sm">
        No data to display yet
      </div>
    );
  }

  return (
    <div className="flex items-end gap-1 h-32">
      {days.map((day) => {
        const count = byDay[day];
        const height = (count / maxCount) * 100;
        return (
          <div key={day} className="flex-1 flex flex-col items-center gap-1">
            <div
              className="w-full bg-gradient-to-t from-violet-600 to-violet-400 rounded-t"
              style={{ height: `${Math.max(height, 4)}%` }}
              title={`${day}: ${count} events`}
            />
            <span className="text-[10px] text-slate-500 -rotate-45 origin-top-left whitespace-nowrap">
              {new Date(day).toLocaleDateString("en-US", { weekday: "short" })}
            </span>
          </div>
        );
      })}
    </div>
  );
}

// Stat card component
function StatCard({ label, value, icon: Icon, trend }: { label: string; value: string | number; icon: React.ElementType; trend?: string }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4">
      <div className="flex items-center justify-between">
        <Icon className="w-5 h-5 text-slate-400" />
        {trend && <span className="text-xs text-emerald-400">{trend}</span>}
      </div>
      <p className="mt-3 text-2xl font-semibold text-slate-100">{value}</p>
      <p className="text-sm text-slate-500">{label}</p>
    </div>
  );
}

export default function App() {
  const [refreshKey, setRefreshKey] = useState(0);

  const healthQuery = useQuery({
    queryKey: ["health", refreshKey],
    queryFn: fetchHealth,
    refetchInterval: 30000,
  });

  const domainsQuery = useQuery({
    queryKey: ["domains", refreshKey],
    queryFn: fetchDomains,
    refetchInterval: 60000,
  });

  const summaryQuery = useQuery({
    queryKey: ["summary", refreshKey],
    queryFn: fetchSummary,
    refetchInterval: 30000,
  });

  const timelineQuery = useQuery({
    queryKey: ["timeline", refreshKey],
    queryFn: () => fetchTimeline(7),
    refetchInterval: 60000,
  });

  const eventsQuery = useQuery({
    queryKey: ["events", refreshKey],
    queryFn: () => fetchEvents({ limit: 10 }),
    refetchInterval: 30000,
  });

  const handleRefresh = () => {
    setRefreshKey((k) => k + 1);
  };

  const isLoading = healthQuery.isLoading || domainsQuery.isLoading || summaryQuery.isLoading;
  const hasError = healthQuery.error || domainsQuery.error || summaryQuery.error;

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50">
      {/* Header */}
      <header className="border-b border-white/10 bg-slate-950/80 backdrop-blur sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center">
                <Activity className="w-5 h-5 text-white" />
              </div>
              <div>
                <h1 className="text-xl font-semibold">Lifestyle Dashboard</h1>
                <p className="text-xs text-slate-500">Personal health intelligence</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              {healthQuery.data && (
                <div className="flex items-center gap-2">
                  {healthQuery.data.status === "healthy" ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  ) : (
                    <AlertCircle className="w-4 h-4 text-amber-400" />
                  )}
                  <StatusBadge status={healthQuery.data.status} />
                </div>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={handleRefresh}
                disabled={isLoading}
                className="border-white/10 hover:bg-white/5"
              >
                <RefreshCw className={`w-4 h-4 mr-2 ${isLoading ? "animate-spin" : ""}`} />
                Refresh
              </Button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
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
            value={timelineQuery.data?.timeline?.reduce((sum, e) => sum + e.count, 0) ?? "-"}
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
            {summaryQuery.data?.events_by_domain && summaryQuery.data.events_by_domain.length > 0 ? (
              <div className="space-y-3">
                {summaryQuery.data.events_by_domain.map((item) => {
                  const maxCount = Math.max(...summaryQuery.data!.events_by_domain.map((d) => d.count), 1);
                  const width = (item.count / maxCount) * 100;
                  return (
                    <div key={item.domain}>
                      <div className="flex justify-between text-sm mb-1">
                        <span className="text-slate-300 truncate">{item.domain}</span>
                        <span className="text-slate-500">{item.count}</span>
                      </div>
                      <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
                        <div
                          className="h-full bg-gradient-to-r from-violet-500 to-fuchsia-500 rounded-full"
                          style={{ width: `${width}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-slate-500 text-sm">No events recorded yet</p>
            )}
          </div>
        </div>

        {/* Two-column layout for domains and events */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Registered domains */}
          <div className="rounded-xl border border-white/10 bg-white/5 p-6">
            <h2 className="text-lg font-medium mb-4">Registered Domains</h2>
            {domainsQuery.isLoading ? (
              <div className="text-slate-500">Loading...</div>
            ) : domainsQuery.data?.domains && domainsQuery.data.domains.length > 0 ? (
              <div className="space-y-4">
                {domainsQuery.data.domains.map((domain) => (
                  <DomainCard key={domain.name} domain={domain} />
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
            <h2 className="text-lg font-medium mb-4">Recent Events</h2>
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

        {/* System info footer */}
        {healthQuery.data && (
          <div className="rounded-xl border border-white/10 bg-white/5 p-4">
            <div className="flex flex-wrap items-center gap-4 text-sm text-slate-500">
              <span>Service: {healthQuery.data.service}</span>
              <span>•</span>
              <span>Version: {healthQuery.data.version || "1.0.0"}</span>
              <span>•</span>
              <span>Uptime: {Math.floor(healthQuery.data.uptime_seconds || 0)}s</span>
              {healthQuery.data.dependencies?.database && (
                <>
                  <span>•</span>
                  <span>
                    Database: {healthQuery.data.dependencies.database.connected ? "connected" : "disconnected"}
                    {healthQuery.data.dependencies.database.latency_ms && (
                      <> ({healthQuery.data.dependencies.database.latency_ms.toFixed(1)}ms)</>
                    )}
                  </span>
                </>
              )}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
