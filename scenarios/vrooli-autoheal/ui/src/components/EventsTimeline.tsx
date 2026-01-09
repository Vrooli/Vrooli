// Events timeline showing recent health check results
// [REQ:UI-EVENTS-001] [REQ:FAIL-SAFE-001]
import { useQuery } from "@tanstack/react-query";
import { Clock, Filter } from "lucide-react";
import { useState, useMemo } from "react";
import { fetchTimeline, TimelineEvent } from "../lib/api";
import { StatusIcon } from "./StatusIcon";
import { ErrorDisplay } from "./ErrorDisplay";
import { selectors } from "../consts/selectors";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";

function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}h ago`;
  const diffDays = Math.floor(diffHour / 24);
  return `${diffDays}d ago`;
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

type FilterOption = "all" | "issues";

export function EventsTimeline() {
  const [filter, setFilter] = useState<FilterOption>("all");
  const [showCount, setShowCount] = useState(20);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["timeline"],
    queryFn: fetchTimeline,
    refetchInterval: 30000, // Refresh every 30 seconds
    retry: 2, // Retry failed requests twice
    retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10000),
  });

  const filteredEvents = useMemo(() => {
    if (!data?.events) return [];
    const events = filter === "issues"
      ? data.events.filter(e => e.status !== "ok")
      : data.events;
    return events.slice(0, showCount);
  }, [data?.events, filter, showCount]);

  const issueCount = useMemo(() => {
    if (!data?.events) return 0;
    return data.events.filter(e => e.status !== "ok").length;
  }, [data?.events]);

  if (isLoading) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4">
        <div className="flex items-center gap-2 mb-3">
          <Clock size={18} className="text-blue-400" />
          <h3 className="font-medium">Recent Events</h3>
        </div>
        <p className="text-sm text-slate-500">Loading timeline...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4">
        <div className="flex items-center gap-2 mb-3">
          <Clock size={18} className="text-blue-400" />
          <h3 className="font-medium">Recent Events</h3>
        </div>
        <ErrorDisplay
          error={error}
          onRetry={() => refetch()}
          compact
        />
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/5" data-testid={selectors.eventsTimeline}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-white/10">
        <div className="flex items-center gap-2">
          <Clock size={18} className="text-blue-400" />
          <h3 className="font-medium">Recent Events</h3>
          <span className="text-xs text-slate-500">({data?.count || 0} total)</span>
        </div>

        {/* Filter toggle */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setFilter(filter === "all" ? "issues" : "all")}
            className={`flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors ${
              filter === "issues"
                ? "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                : "bg-white/5 text-slate-400 hover:bg-white/10"
            }`}
          >
            <Filter size={12} />
            {filter === "issues" ? `Issues (${issueCount})` : "All"}
          </button>
        </div>
      </div>

      {/* Events list */}
      <div className="max-h-80 overflow-y-auto">
        {filteredEvents.length === 0 ? (
          <div className="p-4 text-center text-slate-500 text-sm">
            {filter === "issues" ? "No issues found" : "No events yet"}
          </div>
        ) : (
          <div className="divide-y divide-white/5">
            {filteredEvents.map((event, idx) => (
              <EventRow key={`${event.checkId}-${event.timestamp}-${idx}`} event={event} />
            ))}
          </div>
        )}
      </div>

      {/* Show more */}
      {data?.events && data.events.length > showCount && (
        <div className="p-2 border-t border-white/10">
          <button
            onClick={() => setShowCount(prev => prev + 20)}
            className="w-full py-1.5 text-xs text-slate-400 hover:text-slate-200 transition-colors"
          >
            Show more ({data.events.length - showCount} remaining)
          </button>
        </div>
      )}
    </div>
  );
}

function EventRow({ event }: { event: TimelineEvent }) {
  const { getTitle } = useCheckMetadata();
  const title = getTitle(event.checkId);
  const showCheckId = title !== event.checkId; // Only show checkId if we have a different title

  return (
    <div className="flex items-start gap-3 p-3 hover:bg-white/[0.02] transition-colors">
      <div className="mt-0.5">
        <StatusIcon status={event.status} size={14} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <div className="min-w-0">
            <span className="text-sm font-medium text-slate-200 truncate block" title={event.checkId}>
              {title}
            </span>
            {showCheckId && (
              <span className="text-xs text-slate-600 font-mono">{event.checkId}</span>
            )}
          </div>
          <span className="text-xs text-slate-500 flex-shrink-0" title={new Date(event.timestamp).toLocaleString()}>
            {formatRelativeTime(event.timestamp)}
          </span>
        </div>
        <p className="text-xs text-slate-400 truncate mt-0.5">{event.message}</p>
      </div>
    </div>
  );
}
