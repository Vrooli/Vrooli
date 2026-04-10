/**
 * RecentSection — Collapsible section showing recent events.
 *
 * Derives from execution-store and backlog-store timestamps.
 * Simple chronological list, max 20 items.
 */

import { useState, useMemo } from "react";
import { ChevronRight, History } from "lucide-react";
import { cn } from "../../lib/utils";
import { formatRelativeTime } from "../../lib";
import { useBacklogStore } from "../../stores/backlog-store";
import { useExecutionStore } from "../../stores/execution-store";
import type { BacklogItem, ExecutionRecord } from "../../types";

const MAX_RECENT = 20;

interface RecentEvent {
  id: string;
  label: string;
  timestamp: string;
}

function buildRecentEvents(backlogItems: BacklogItem[], executions: ExecutionRecord[]): RecentEvent[] {
  const events: RecentEvent[] = [];

  for (const exec of executions) {
    if (exec.status === "completed" || exec.status === "failed" || exec.status === "needs_review") {
      events.push({
        id: `exec-${exec.executionId}`,
        label: `${exec.status === "completed" ? "Completed" : exec.status === "failed" ? "Failed" : "Review needed"}: ${exec.backlogName || exec.executionId}`,
        timestamp: exec.updatedAt ?? exec.createdAt,
      });
    }
  }

  for (const item of backlogItems) {
    if (item.status === "completed" || item.status === "ready") {
      events.push({
        id: `backlog-${item.kind}/${item.name}`,
        label: `${item.status === "completed" ? "Completed" : "Ready"}: ${item.title || item.name}`,
        timestamp: item.updated,
      });
    }
  }

  events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  return events.slice(0, MAX_RECENT);
}

export function RecentSection() {
  const [expanded, setExpanded] = useState(false);
  const backlogItems = useBacklogStore((s) => s.items);
  const executions = useExecutionStore((s) => s.items);

  const events = useMemo(
    () => buildRecentEvents(backlogItems, executions),
    [backlogItems, executions],
  );

  if (events.length === 0) return null;

  return (
    <div data-testid="recent-section">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 transition-colors hover:bg-slate-800/50 hover:text-slate-300"
        data-testid="recent-section-toggle"
      >
        <ChevronRight
          className={cn("h-4 w-4 transition-transform", expanded && "rotate-90")}
        />
        <History className="h-3.5 w-3.5" />
        <span>Recent activity</span>
      </button>

      {expanded && (
        <div className="mt-1 space-y-0.5 pl-4">
          {events.map((event) => (
            <div
              key={event.id}
              className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm"
            >
              <span className="min-w-0 flex-1 truncate text-slate-300">
                {event.label}
              </span>
              <span className="shrink-0 text-xs text-slate-500">
                {formatRelativeTime(event.timestamp)}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
