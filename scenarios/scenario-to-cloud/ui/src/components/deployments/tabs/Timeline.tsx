import {
  Plus,
  Package,
  CheckCircle2,
  XCircle,
  Settings,
  Rocket,
  Search,
  Pause,
  RefreshCw,
  Heart,
  Info,
  Clock,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { useState } from "react";
import { getEventTypeInfo, formatDuration, getTimeSince } from "../../../hooks/useLiveState";
import { cn } from "../../../lib/utils";
import type { HistoryEvent } from "../../../lib/api";

interface TimelineProps {
  events: HistoryEvent[];
}

export function Timeline({ events }: TimelineProps) {
  const [expandedEvent, setExpandedEvent] = useState<number | null>(null);

  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-400">
        <Clock className="h-12 w-12 mb-4 opacity-50" />
        <p className="text-sm">No history events yet</p>
        <p className="text-xs text-slate-500 mt-1">
          Events will appear as you deploy and manage this deployment
        </p>
      </div>
    );
  }

  // Group events by date
  const groupedEvents = events.reduce((acc, event) => {
    const date = new Date(event.timestamp).toLocaleDateString();
    if (!acc[date]) {
      acc[date] = [];
    }
    acc[date].push(event);
    return acc;
  }, {} as Record<string, HistoryEvent[]>);

  return (
    <div className="space-y-6">
      {Object.entries(groupedEvents).map(([date, dateEvents]) => (
        <div key={date}>
          {/* Date header */}
          <div className="flex items-center gap-3 mb-4">
            <div className="h-px flex-1 bg-white/10" />
            <span className="text-xs font-medium text-slate-400 px-2">{date}</span>
            <div className="h-px flex-1 bg-white/10" />
          </div>

          {/* Events for this date */}
          <div className="relative">
            {/* Timeline line */}
            <div className="absolute left-4 top-0 bottom-0 w-px bg-white/10" />

            {/* Events */}
            <div className="space-y-4">
              {dateEvents.map((event, idx) => {
                const eventInfo = getEventTypeInfo(event.type);
                const Icon = getIconForType(eventInfo.icon);
                const isExpanded = expandedEvent === idx;
                const hasDetails = event.details || event.data || event.bundle_hash;

                return (
                  <div
                    key={idx}
                    className={cn(
                      "relative pl-10",
                      hasDetails && "cursor-pointer"
                    )}
                    onClick={() => hasDetails && setExpandedEvent(isExpanded ? null : idx)}
                  >
                    {/* Timeline dot */}
                    <div
                      className={cn(
                        "absolute left-2 top-1.5 w-5 h-5 rounded-full flex items-center justify-center",
                        `bg-${eventInfo.color}-500/20 border border-${eventInfo.color}-500/30`
                      )}
                    >
                      <Icon className={cn("h-3 w-3", `text-${eventInfo.color}-400`)} />
                    </div>

                    {/* Event card */}
                    <div
                      className={cn(
                        "p-3 rounded-lg border transition-colors",
                        event.success === false
                          ? "border-red-500/30 bg-red-500/5"
                          : "border-white/10 bg-slate-900/50",
                        hasDetails && "hover:bg-slate-800/50"
                      )}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span
                              className={cn(
                                "text-sm font-medium",
                                `text-${eventInfo.color}-400`
                              )}
                            >
                              {eventInfo.label}
                            </span>
                            {event.step_name && (
                              <span className="text-xs text-slate-500">
                                ({event.step_name})
                              </span>
                            )}
                            {hasDetails && (
                              isExpanded ? (
                                <ChevronDown className="h-4 w-4 text-slate-500" />
                              ) : (
                                <ChevronRight className="h-4 w-4 text-slate-500" />
                              )
                            )}
                          </div>

                          {event.message && (
                            <p className="text-sm text-slate-300 mt-1 truncate">
                              {event.message}
                            </p>
                          )}

                          <div className="flex items-center gap-3 mt-2 text-xs text-slate-500">
                            <span>{getTimeSince(event.timestamp)}</span>
                            <span>{new Date(event.timestamp).toLocaleTimeString()}</span>
                            {event.duration_ms && (
                              <span className="text-emerald-400">
                                {formatDuration(event.duration_ms)}
                              </span>
                            )}
                          </div>
                        </div>

                        {/* Status indicator */}
                        {event.success !== undefined && (
                          <div className="flex-shrink-0">
                            {event.success ? (
                              <CheckCircle2 className="h-5 w-5 text-emerald-400" />
                            ) : (
                              <XCircle className="h-5 w-5 text-red-400" />
                            )}
                          </div>
                        )}
                      </div>

                      {/* Expanded details */}
                      {isExpanded && (
                        <div className="mt-3 pt-3 border-t border-white/10 space-y-2">
                          {event.bundle_hash && (
                            <div className="text-xs">
                              <span className="text-slate-500">Bundle: </span>
                              <span className="text-purple-400 font-mono">
                                {event.bundle_hash.slice(0, 16)}...
                              </span>
                            </div>
                          )}

                          {event.details && (
                            <pre className="text-xs text-slate-400 bg-slate-950 p-2 rounded overflow-x-auto">
                              {event.details}
                            </pre>
                          )}

                          {event.data !== undefined && event.data !== null && (
                            <pre className="text-xs text-slate-400 bg-slate-950 p-2 rounded overflow-x-auto">
                              {typeof event.data === "string" ? event.data : JSON.stringify(event.data, null, 2)}
                            </pre>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// Helper to get the icon component for a type
function getIconForType(iconName: string) {
  const iconMap: Record<string, typeof Plus> = {
    plus: Plus,
    package: Package,
    "check-circle": CheckCircle2,
    "x-circle": XCircle,
    settings: Settings,
    rocket: Rocket,
    search: Search,
    pause: Pause,
    refresh: RefreshCw,
    heart: Heart,
    info: Info,
  };

  return iconMap[iconName] || Info;
}
