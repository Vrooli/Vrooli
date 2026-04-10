// Events timeline showing recent health check results
// [REQ:UI-EVENTS-001] [REQ:FAIL-SAFE-001]
import { useQuery } from "@tanstack/react-query";
import { Clock, Filter } from "lucide-react";
import { useState, useMemo } from "react";
import { fetchTimeline, TimelineEvent } from "../../../lib/api";
import { ErrorDisplay, StatusIcon } from "../../../shared/components";
import { Card } from "../../../shared/ui/primitives";
import { selectors } from "../../../consts/selectors";
import { useCheckMetadata } from "../../../shared/contexts/CheckMetadataContext";
import { formatRelativeTime } from "../../../lib/utils";

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
      <Card className="p-4">
        <div className="flex items-center gap-2 mb-3">
          <Clock size={18} className="text-accent-primary" />
          <h3 className="font-medium">Recent Events</h3>
        </div>
        <p className="text-sm text-text-muted">Loading timeline...</p>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="p-4">
        <div className="flex items-center gap-2 mb-3">
          <Clock size={18} className="text-accent-primary" />
          <h3 className="font-medium">Recent Events</h3>
        </div>
        <ErrorDisplay
          error={error}
          onRetry={() => refetch()}
          compact
        />
      </Card>
    );
  }

  return (
    <Card data-testid={selectors.eventsTimeline}>
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border-default/50 p-4">
        <div className="flex min-w-0 items-center gap-2">
          <Clock size={18} className="text-accent-primary" />
          <h3 className="font-medium">Recent Events</h3>
          <span className="text-xs text-text-muted">({data?.count || 0} total)</span>
        </div>

        {/* Filter toggle */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setFilter(filter === "all" ? "issues" : "all")}
            className={`flex items-center gap-1.5 rounded px-2 py-1 text-xs transition-colors ${
              filter === "issues"
                ? "border border-accent-warning/40 bg-accent-warning/20 text-accent-warning"
                : "bg-surface-overlay/50 text-text-muted hover:bg-surface-overlay/70"
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
          <div className="p-4 text-center text-sm text-text-muted">
            {filter === "issues" ? "No issues found" : "No events yet"}
          </div>
        ) : (
          <div className="divide-y divide-border-default/30">
            {filteredEvents.map((event, idx) => (
              <EventRow key={`${event.checkId}-${event.timestamp}-${idx}`} event={event} />
            ))}
          </div>
        )}
      </div>

      {/* Show more */}
      {data?.events && data.events.length > showCount && (
        <div className="border-t border-border-default/50 p-2">
          <button
            onClick={() => setShowCount(prev => prev + 20)}
            className="w-full py-1.5 text-xs text-text-muted transition-colors hover:text-text-primary"
          >
            Show more ({data.events.length - showCount} remaining)
          </button>
        </div>
      )}
    </Card>
  );
}

function EventRow({ event }: { event: TimelineEvent }) {
  const { getTitle } = useCheckMetadata();
  const title = getTitle(event.checkId);
  const showCheckId = title !== event.checkId; // Only show checkId if we have a different title

  return (
    <div className="flex items-start gap-3 p-3 transition-colors hover:bg-surface-overlay/40">
      <div className="mt-0.5">
        <StatusIcon status={event.status} size={14} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <span className="block truncate text-sm font-medium text-text-primary" title={event.checkId}>
              {title}
            </span>
            {showCheckId && (
              <span className="break-all font-mono text-xs text-text-muted/80">{event.checkId}</span>
            )}
          </div>
          <span className="shrink-0 text-xs text-text-muted" title={new Date(event.timestamp).toLocaleString()}>
            {formatRelativeTime(event.timestamp)}
          </span>
        </div>
        <p className="mt-0.5 break-words text-xs text-text-muted">{event.message}</p>
      </div>
    </div>
  );
}
