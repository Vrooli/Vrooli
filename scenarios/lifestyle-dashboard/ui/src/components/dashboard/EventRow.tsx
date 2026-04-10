/**
 * EventRow displays a single event in the recent events list.
 * Shows event type, domain, timestamp, and intervention status.
 *
 * [REQ:LD-EVENT-SCHEMA] - UI representation of event schema
 */
import { formatRelativeTime } from "../../lib/format";
import type { Event } from "../../lib/api";

interface EventRowProps {
  event: Event;
  onClick?: () => void;
  showDomain?: boolean;
}

export function EventRow({ event, onClick, showDomain = true }: EventRowProps) {
  return (
    <div
      className="flex items-center gap-4 py-3 border-b border-white/5 last:border-0 hover:bg-white/5 cursor-pointer px-2 -mx-2 rounded"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => e.key === "Enter" && onClick?.()}
    >
      <div className="w-2 h-2 rounded-full bg-violet-500 flex-shrink-0" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-slate-200 truncate">{event.event_type}</span>
          {event.is_intervention && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-amber-500/20 text-amber-400">intervention</span>
          )}
        </div>
        {showDomain && <p className="text-xs text-slate-500">{event.domain}</p>}
      </div>
      <span className="text-xs text-slate-500 flex-shrink-0">{formatRelativeTime(event.timestamp)}</span>
    </div>
  );
}
