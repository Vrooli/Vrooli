/**
 * ActivityTimeline - Unified chronological feed of executions and agent
 * activities for a backlog item, displayed inside the Activity Timeline Drawer.
 */

import { useState, useCallback, useEffect, useRef } from "react";
import { renderMarkdown } from "../../lib/render-markdown";
import {
  ArrowUpRight,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Loader2,
  Square,
} from "lucide-react";
import { Button } from "../ui/button";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { cn, formatRelativeTime, canFollowUpExecution, resolvePostRunExecution } from "../../lib";
import {
  EXECUTION_STATUS_COLORS,
  formatExecutionStatus,
  type AgentActivity,
  type AgentActivityPurpose,
  type ExecutionRecord,
} from "../../types";
import type { TimelineEntry } from "../../hooks/useActivityTimeline";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const ACTIVITY_PURPOSE_COLORS: Record<AgentActivityPurpose, string> = {
  initialize: "bg-slate-500",
  workshop: "bg-violet-500",
  finalize: "bg-emerald-500",
  research: "bg-blue-500",
  process: "bg-cyan-500",
  fixup: "bg-orange-500",
  followup: "bg-amber-500",
  spec_sync: "bg-indigo-500",
  classify: "bg-pink-500",
  clarify: "bg-teal-500",
  review: "bg-sky-500",
};

const ACTIVITY_STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-500",
  starting: "bg-violet-500",
  running: "bg-cyan-500",
  needs_review: "bg-yellow-500",
  complete: "bg-emerald-500",
  failed: "bg-red-500",
  cancelled: "bg-amber-500",
  unspecified: "bg-slate-500",
};

function formatDuration(seconds?: number): string {
  if (!seconds || seconds <= 0) return "Unknown";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
}

function formatPurpose(purpose: string): string {
  return purpose.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export interface ActivityTimelineProps {
  entries: TimelineEntry[];
  isLoading: boolean;
  error: Error | null;
  onViewExecution: (exec: ExecutionRecord) => void;
  onStopRun: (runId: string) => void;
  onFollowUp: (exec: ExecutionRecord) => void;
  latestAgentActivity: AgentActivityRecord | undefined;
  agentRunIsActive: boolean;
  /** Base URL for the agent-manager UI. When set, "Run" links appear on items that have a runId. */
  agentManagerUiUrl?: string;
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ActivityItem({ activity, agentManagerUiUrl }: { activity: AgentActivity; agentManagerUiUrl?: string }) {
  const [expanded, setExpanded] = useState(false);
  const statusColor = ACTIVITY_STATUS_COLORS[activity.status] ?? "bg-slate-500";
  const purposeColor = ACTIVITY_PURPOSE_COLORS[activity.purpose] ?? "bg-slate-500";

  const duration =
    activity.startedAt && activity.finishedAt
      ? (new Date(activity.finishedAt).getTime() - new Date(activity.startedAt).getTime()) / 1000
      : undefined;

  return (
    <div className="rounded bg-slate-800/30">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-1.5 px-2 py-1.5 text-left"
      >
        {expanded ? (
          <ChevronDown className="h-2.5 w-2.5 shrink-0 text-slate-500" />
        ) : (
          <ChevronRight className="h-2.5 w-2.5 shrink-0 text-slate-500" />
        )}
        <span className={cn("inline-block h-1.5 w-1.5 shrink-0 rounded-full", statusColor)} />
        <span className={cn("rounded px-1 py-0.5 text-[10px] font-medium text-white", purposeColor)}>
          {formatPurpose(activity.purpose)}
        </span>
        <span className="rounded bg-slate-700/60 px-1 py-0.5 text-[10px] text-slate-400">
          {activity.interactionType}
        </span>
        <span className="ml-auto text-[10px] text-slate-500">
          {formatRelativeTime(activity.requestedAt)}
        </span>
      </button>
      {expanded && (
        <div className="space-y-1 border-t border-slate-700/30 px-2 py-1.5 text-xs text-slate-400">
          {activity.failureReason && (
            <div className="prose-sm-slate rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200" dangerouslySetInnerHTML={{ __html: renderMarkdown(activity.failureReason) }} />
          )}
          {duration !== undefined && <p>Duration: {formatDuration(duration)}</p>}
          {activity.startedAt && <p>Started: {formatRelativeTime(activity.startedAt)}</p>}
          {activity.finishedAt && <p>Finished: {formatRelativeTime(activity.finishedAt)}</p>}
          {activity.requestedBy && <p>Requested by: {activity.requestedBy}</p>}
          <p className="font-mono text-[11px] text-slate-500">
            ID: {activity.activityId}
          </p>
          {activity.runId && (
            <p className="font-mono text-[11px] text-slate-500">
              Run: {activity.runId}
            </p>
          )}
          {activity.runId && agentManagerUiUrl && (
            <a
              href={`${agentManagerUiUrl}/runs/${activity.runId}`}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
            >
              <Button size="sm" variant="outline" className="h-6 px-1.5 text-[10px]">
                <ExternalLink className="mr-1 h-2.5 w-2.5" />
                Run
              </Button>
            </a>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export function ActivityTimeline({
  entries,
  isLoading,
  error,
  onViewExecution,
  onStopRun,
  onFollowUp,
  latestAgentActivity,
  agentRunIsActive,
  agentManagerUiUrl,
}: ActivityTimelineProps) {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const seenIdsRef = useRef<Set<string>>(new Set());

  // Auto-expand new execution entries as they arrive
  useEffect(() => {
    const newExecIds: string[] = [];
    for (const entry of entries) {
      if (entry.type === "execution" && !seenIdsRef.current.has(entry.id)) {
        newExecIds.push(entry.id);
        seenIdsRef.current.add(entry.id);
      }
    }
    if (newExecIds.length > 0) {
      setExpandedIds((prev) => {
        const next = new Set<string>(prev);
        for (const id of newExecIds) next.add(id);
        return next;
      });
    }
  }, [entries]);

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set<string>(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  // Loading state
  if (isLoading && entries.length === 0) {
    return (
      <div className="flex items-center justify-center gap-2 px-4 py-12 text-sm text-slate-400">
        <Loader2 className="h-4 w-4 animate-spin" />
        Loading activity history…
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="px-4 py-8 text-center text-sm text-red-300">
        Failed to load activity history: {error.message}
      </div>
    );
  }

  // Empty state
  if (entries.length === 0) {
    return (
      <div className="px-4 py-12 text-center text-sm text-slate-500">
        No activity history yet.
      </div>
    );
  }

  return (
    <div className="space-y-2 p-3">
      {entries.map((entry) => {
        if (entry.type === "execution" && entry.execution) {
          return (
            <ExecutionTimelineItem
              key={entry.id}
              entry={entry}
              isExpanded={expandedIds.has(entry.id)}
              onToggle={() => toggleExpand(entry.id)}
              onViewExecution={onViewExecution}
              onStopRun={onStopRun}
              onFollowUp={onFollowUp}
              latestAgentActivity={latestAgentActivity}
              agentRunIsActive={agentRunIsActive}
              agentManagerUiUrl={agentManagerUiUrl}
            />
          );
        }

        if (entry.type === "activity" && entry.activity) {
          return (
            <div key={entry.id} className="ml-0">
              <ActivityItem activity={entry.activity} agentManagerUiUrl={agentManagerUiUrl} />
            </div>
          );
        }

        return null;
      })}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Execution timeline item
// ---------------------------------------------------------------------------

function ExecutionTimelineItem({
  entry,
  isExpanded,
  onToggle,
  onViewExecution,
  onStopRun,
  onFollowUp,
  latestAgentActivity,
  agentRunIsActive,
  agentManagerUiUrl,
}: {
  entry: TimelineEntry;
  isExpanded: boolean;
  onToggle: () => void;
  onViewExecution: (exec: ExecutionRecord) => void;
  onStopRun: (runId: string) => void;
  onFollowUp: (exec: ExecutionRecord) => void;
  latestAgentActivity: AgentActivityRecord | undefined;
  agentRunIsActive: boolean;
  agentManagerUiUrl?: string;
}) {
  const exec = entry.execution;
  if (!exec) return null;
  const statusColor = EXECUTION_STATUS_COLORS[exec.status] ?? "bg-slate-500";
  const isActiveExecRun = !!(
    exec.runId &&
    latestAgentActivity &&
    exec.runId === latestAgentActivity.runId &&
    agentRunIsActive
  );
  const duration =
    exec.startedAt && exec.finishedAt
      ? (new Date(exec.finishedAt).getTime() - new Date(exec.startedAt).getTime()) / 1000
      : undefined;

  return (
    <div className="rounded-lg border border-white/5 bg-slate-800/40">
      {/* Header row */}
      <button
        type="button"
        onClick={onToggle}
        className="flex w-full items-center gap-2 px-3 py-2 text-left"
      >
        {isExpanded ? (
          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-slate-400" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 shrink-0 text-slate-400" />
        )}
        <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", statusColor)} />
        <span className="text-xs font-medium text-slate-200">
          {formatExecutionStatus(exec.status)}
        </span>
        {exec.operation && (
          <span className="rounded bg-slate-700/60 px-1 py-0.5 text-[10px] text-slate-400">
            {exec.operation}
          </span>
        )}
        {entry.childActivities && entry.childActivities.length > 0 && (
          <span className="rounded-full bg-slate-700 px-1.5 py-0.5 text-[10px] text-slate-400">
            {entry.childActivities.length} {entry.childActivities.length === 1 ? "activity" : "activities"}
          </span>
        )}
        <span className="ml-auto text-[10px] text-slate-500">
          {formatRelativeTime(exec.createdAt)}
        </span>
      </button>

      {/* Expanded details */}
      {isExpanded && (
        <div className="space-y-2 border-t border-slate-700/40 px-3 py-2">
          {/* Failure reason */}
          {exec.failureReason && (
            <div className="prose-sm-slate rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200" dangerouslySetInnerHTML={{ __html: renderMarkdown(exec.failureReason) }} />
          )}

          {/* Post-run status */}
          {(() => {
            const resolved = resolvePostRunExecution(exec);
            if (!resolved) return null;
            return <PostRunStatusBadge execution={resolved} />;
          })()}

          {/* Metadata */}
          <div className="space-y-1 text-xs text-slate-400">
            {duration !== undefined && <p>Duration: {formatDuration(duration)}</p>}
            {exec.startedAt && <p>Started: {formatRelativeTime(exec.startedAt)}</p>}
            {exec.finishedAt && <p>Finished: {formatRelativeTime(exec.finishedAt)}</p>}
            <p className="font-mono text-[11px] text-slate-500">ID: {exec.executionId}</p>
            {exec.runId && (
              <p className="font-mono text-[11px] text-slate-500">Run: {exec.runId}</p>
            )}
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => onViewExecution(exec)}
            >
              <ArrowUpRight className="mr-1 h-3 w-3" />
              View
            </Button>
            {exec.runId && agentManagerUiUrl && (
              <a
                href={`${agentManagerUiUrl}/runs/${exec.runId}`}
                target="_blank"
                rel="noopener noreferrer"
                onClick={(e) => e.stopPropagation()}
              >
                <Button variant="outline" size="sm" className="h-7 px-2 text-xs">
                  <ExternalLink className="mr-1 h-3 w-3" />
                  Run
                </Button>
              </a>
            )}
            {isActiveExecRun && (
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={() => onStopRun(latestAgentActivity?.runId ?? "")}
                disabled={latestAgentActivity?.isStopping}
              >
                <Square className="mr-1 h-3 w-3" />
                {latestAgentActivity?.isStopping ? "Stopping…" : "Stop"}
              </Button>
            )}
            {canFollowUpExecution(exec.status) && (
              <Button
                variant="outline"
                size="sm"
                className="h-7 px-2 text-xs"
                onClick={() => onFollowUp(exec)}
              >
                Follow Up
              </Button>
            )}
          </div>

          {/* Child activities */}
          {entry.childActivities && entry.childActivities.length > 0 && (
            <div className="space-y-1 border-l-2 border-slate-700/50 pl-2 pt-1">
              <p className="text-[10px] font-medium uppercase tracking-wider text-slate-500">
                Agent Activities
              </p>
              {entry.childActivities.map((act) => (
                <ActivityItem key={act.activityId} activity={act} agentManagerUiUrl={agentManagerUiUrl} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
