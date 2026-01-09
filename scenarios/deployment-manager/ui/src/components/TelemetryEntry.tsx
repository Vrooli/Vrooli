import { AlertTriangle, RefreshCw } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import type { TelemetryEvent, TelemetrySummary } from "../lib/api";

interface TelemetryEntryProps {
  entry: TelemetrySummary;
  onRefresh: () => void;
}

function TelemetryEventItem({ evt, className }: { evt: TelemetryEvent; className?: string }) {
  return (
    <div className={className}>
      <div className="flex items-center justify-between gap-2">
        <span className="text-[12px] font-semibold uppercase tracking-wide">
          {evt.event || "unknown"}
        </span>
        <span className="text-[11px] text-slate-400">{evt.timestamp || "unknown time"}</span>
      </div>
      {evt.details && (
        <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-black/40 p-2 text-[10px] text-slate-100">
          {JSON.stringify(evt.details, null, 2)}
        </pre>
      )}
    </div>
  );
}

function FailureItem({ evt }: { evt: TelemetryEvent }) {
  return (
    <div className="flex items-start gap-2">
      <span className="mt-[2px] block h-1.5 w-1.5 rounded-full bg-red-300" />
      <div>
        <div className="font-semibold">{evt.event || "unknown"}</div>
        <div className="text-slate-200">{evt.timestamp}</div>
        {evt.details && (
          <pre className="mt-1 whitespace-pre-wrap break-words rounded bg-black/30 p-2 text-[10px] text-slate-100">
            {JSON.stringify(evt.details, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
}

/**
 * Displays a single telemetry summary entry with failure breakdowns,
 * recent events, and refresh capability.
 */
export function TelemetryEntry({ entry, onRefresh }: TelemetryEntryProps) {
  const failureEntries = Object.entries(entry.failure_counts || {}).filter(
    ([, val]) => (val || 0) > 0
  );
  const failureTotal = failureEntries.reduce((sum, [, val]) => sum + (val || 0), 0);
  const recentEvents = entry.recent_events || [];

  return (
    <div className="rounded border border-slate-700 bg-black/20 p-3 space-y-1">
      {/* Header */}
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2 text-sm font-semibold">
            {entry.scenario}
            {failureTotal > 0 ? (
              <Badge variant="destructive" className="gap-1">
                <AlertTriangle className="h-3 w-3" />
                {failureTotal} failure{failureTotal === 1 ? "" : "s"}
              </Badge>
            ) : (
              <Badge variant="secondary">Clean</Badge>
            )}
          </div>
          <p className="text-xs text-slate-400">
            {entry.total_events} events • last {entry.last_event || "event"} @{" "}
            {entry.last_timestamp || "unknown"}
          </p>
          <p className="text-[11px] text-slate-500 break-all">{entry.path}</p>
        </div>
        <Button size="sm" variant="ghost" onClick={onRefresh} className="gap-1">
          <RefreshCw className="h-4 w-4" /> Refresh
        </Button>
      </div>

      {/* Recent Failures */}
      {entry.recent_failures && entry.recent_failures.length > 0 && (
        <div className="rounded border border-red-500/30 bg-red-500/5 p-2 text-[11px] text-red-100 space-y-1">
          <p className="font-semibold">Recent failures</p>
          {entry.recent_failures.map((evt, idx) => (
            <FailureItem key={idx} evt={evt} />
          ))}
        </div>
      )}

      {/* Failure Breakdown */}
      {failureEntries.length > 0 && (
        <div className="rounded border border-red-500/20 bg-red-500/5 p-2 text-[11px] text-red-100 space-y-1">
          <p className="font-semibold">Failure breakdown</p>
          <div className="grid gap-2 sm:grid-cols-2">
            {failureEntries.map(([event, count]) => (
              <div
                key={event}
                className="rounded border border-red-500/20 bg-black/20 p-2 flex items-center justify-between"
              >
                <div className="flex items-center gap-2">
                  <Badge variant="destructive" className="uppercase tracking-wide">
                    {event}
                  </Badge>
                  <span className="text-sm font-semibold">{count}</span>
                </div>
                <span className="text-[11px] text-slate-300">Bundled runtime telemetry</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recent Events */}
      {recentEvents.length > 0 && (
        <div className="rounded border border-white/10 bg-white/5 p-2 text-[11px] text-slate-200 space-y-1">
          <p className="font-semibold">Recent events</p>
          <div className="space-y-1">
            {recentEvents.map((evt, idx) => (
              <TelemetryEventItem
                key={`${entry.path}-${idx}`}
                evt={evt}
                className="rounded border border-slate-700 bg-black/20 p-2"
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
