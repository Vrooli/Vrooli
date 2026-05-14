import { AlertCircle, CheckCircle2, Clock3, MessageSquare, RefreshCw, Terminal, Wrench } from "lucide-react";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib/format-utils";
import type { AgentSessionRunEvent } from "../../types";

interface SessionEventTimelineProps {
  events: AgentSessionRunEvent[];
  isLoading: boolean;
  error: string | null;
  onRefresh: () => void;
  variant?: "panel" | "plain";
}

export function SessionEventTimeline({
  events,
  isLoading,
  error,
  onRefresh,
  variant = "panel",
}: SessionEventTimelineProps) {
  return (
    <section className={cn("space-y-3", variant === "panel" && "rounded-lg border border-white/10 bg-slate-950/30 p-3")}>
      <div className="flex items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-medium text-slate-100">Events</h3>
          <p className="text-xs text-slate-400">{events.length > 0 ? `${events.length} run events` : "No run events yet."}</p>
        </div>
        <Button variant="ghost" size="sm" onClick={onRefresh} disabled={isLoading}>
          <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", isLoading && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {error && (
        <div className="flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-100" role="alert">
          <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span className="min-w-0 break-words">{error}</span>
        </div>
      )}

      {events.length === 0 ? (
        <div className="rounded-md border border-dashed border-white/10 px-3 py-6 text-center text-sm text-slate-400">
          {isLoading ? "Loading run events..." : "Events will appear after the run starts."}
        </div>
      ) : (
        <ol className="space-y-2">
          {events.map((event) => (
            <li key={event.id || event.sequence.toString()} className="rounded-md border border-white/10 bg-slate-900/40 p-2">
              <div className="flex items-start gap-2">
                <EventIcon event={event} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs">
                    <span className="font-medium text-slate-200">{eventLabel(event)}</span>
                    {event.createdAt && <span className="text-slate-500">{formatRelativeTime(event.createdAt)}</span>}
                    <span className="text-slate-600">#{event.sequence.toString()}</span>
                  </div>
                  <EventSummary event={event} />
                </div>
              </div>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}

function EventIcon({ event }: { event: AgentSessionRunEvent }) {
  const className = "mt-0.5 h-4 w-4 shrink-0";
  if (event.eventType === "tool_call" || event.eventType === "tool_result") {
    return <Wrench className={cn(className, "text-cyan-300")} />;
  }
  if (event.eventType === "error") {
    return <AlertCircle className={cn(className, "text-red-300")} />;
  }
  if (event.eventType === "message") {
    return <MessageSquare className={cn(className, "text-blue-300")} />;
  }
  if (event.eventType === "status") {
    return <CheckCircle2 className={cn(className, "text-emerald-300")} />;
  }
  if (event.eventType === "log") {
    return <Terminal className={cn(className, "text-slate-300")} />;
  }
  return <Clock3 className={cn(className, "text-slate-400")} />;
}

function EventSummary({ event }: { event: AgentSessionRunEvent }) {
  const text = eventSummary(event);
  const detail = eventDetail(event);
  return (
    <div className="mt-1 space-y-1">
      {text && <p className="min-w-0 whitespace-pre-wrap break-words text-xs text-slate-300">{text}</p>}
      {detail && (
        <details className="text-xs text-slate-400">
          <summary className="cursor-pointer select-none text-slate-500">Payload</summary>
          <pre className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded bg-slate-950/70 p-2">{detail}</pre>
        </details>
      )}
    </div>
  );
}

function eventLabel(event: AgentSessionRunEvent): string {
  if (event.eventType === "tool_call") return `Tool call${event.toolName ? `: ${event.toolName}` : ""}`;
  if (event.eventType === "tool_result") return `Tool result${event.toolName ? `: ${event.toolName}` : ""}`;
  if (event.eventType === "message") return `Message${event.role ? `: ${event.role}` : ""}`;
  if (event.eventType === "status") return "Status";
  if (event.eventType === "error") return "Error";
  if (event.eventType === "compaction") return "Compaction";
  return event.eventType.replace(/_/g, " ");
}

function eventSummary(event: AgentSessionRunEvent): string {
  if (event.content) return event.content;
  if (event.error) return event.error;
  if (event.progressMessage) return event.progressMessage;
  if (event.status || event.previousStatus) return [event.previousStatus, event.status].filter(Boolean).join(" -> ");
  if (event.summary) return event.summary;
  if (event.output) return event.output;
  return "";
}

function eventDetail(event: AgentSessionRunEvent): string {
  return event.input || event.rawJson || "";
}
