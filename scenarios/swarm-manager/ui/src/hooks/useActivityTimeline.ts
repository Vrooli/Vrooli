/**
 * useActivityTimeline - Merges execution records and agent activities into a
 * unified chronological timeline for a backlog item.
 *
 * Activities are grouped under their parent execution via `executionId`.
 * Activities with no `executionId` appear as standalone top-level entries.
 * Top-level entries are sorted newest-first by timestamp.
 */

import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import type { AgentActivity, ExecutionRecord } from "../types";
import { executionService } from "../services/execution-service";
import { agentActivityService } from "../services/agent-activity-service";
import type { BacklogKind } from "../types";

export type TimelineEntryType = "execution" | "activity";

export interface TimelineEntry {
  /** Unique id (executionId or activityId) */
  id: string;
  /** Entry type */
  type: TimelineEntryType;
  /** ISO timestamp used for sorting (createdAt for executions, requestedAt for activities) */
  timestamp: string;
  /** Present when type === "execution" */
  execution?: ExecutionRecord;
  /** Present when type === "activity" (standalone entries only) */
  activity?: AgentActivity;
  /** Activities that belong to this execution (populated only on execution entries) */
  childActivities?: AgentActivity[];
}

interface UseActivityTimelineParams {
  backlogKind: string | undefined;
  backlogName: string | undefined;
  /** Only fetch activities when the drawer is open */
  enabled: boolean;
  /** Poll more frequently when an agent run is active */
  agentRunIsActive: boolean;
}

interface UseActivityTimelineResult {
  entries: TimelineEntry[];
  isLoading: boolean;
  error: Error | null;
}

const POLL_INTERVAL_MS = 10_000;

/**
 * Pure merge function — exported for testing.
 */
export function mergeTimeline(
  executions: ExecutionRecord[] | undefined,
  activities: AgentActivity[] | undefined,
): TimelineEntry[] {
  if (!executions?.length && !activities?.length) return [];

  // Index activities by executionId
  const activitiesByExecId = new Map<string, AgentActivity[]>();
  const orphanActivities: AgentActivity[] = [];

  for (const act of activities ?? []) {
    if (act.executionId) {
      const list = activitiesByExecId.get(act.executionId);
      if (list) {
        list.push(act);
      } else {
        activitiesByExecId.set(act.executionId, [act]);
      }
    } else {
      orphanActivities.push(act);
    }
  }

  // Sort child activity lists newest-first
  for (const list of activitiesByExecId.values()) {
    list.sort((a, b) => new Date(b.requestedAt).getTime() - new Date(a.requestedAt).getTime());
  }

  const entries: TimelineEntry[] = [];

  // Execution entries with child activities attached
  for (const exec of executions ?? []) {
    entries.push({
      id: exec.executionId,
      type: "execution",
      timestamp: exec.createdAt,
      execution: exec,
      childActivities: activitiesByExecId.get(exec.executionId),
    });
  }

  // Orphan activity entries
  for (const act of orphanActivities) {
    entries.push({
      id: act.activityId,
      type: "activity",
      timestamp: act.requestedAt,
      activity: act,
    });
  }

  // Sort top-level entries newest-first
  entries.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return entries;
}

export function useActivityTimeline({
  backlogKind,
  backlogName,
  enabled,
  agentRunIsActive,
}: UseActivityTimelineParams): UseActivityTimelineResult {
  const hasParams = !!backlogKind && !!backlogName;

  const {
    data: executions,
    isLoading: execLoading,
    error: execError,
  } = useQuery({
    queryKey: ["executions", backlogKind, backlogName],
    queryFn: () =>
      executionService.list({
        backlogKind: backlogKind as BacklogKind,
        backlogName: backlogName ?? "",
      }),
    enabled: hasParams && enabled,
    refetchInterval: enabled ? POLL_INTERVAL_MS : false,
  });

  const {
    data: activities,
    isLoading: actLoading,
    error: actError,
  } = useQuery({
    queryKey: ["agent-activities-timeline", backlogKind, backlogName],
    queryFn: () =>
      agentActivityService.list({
        ownerType: "backlog",
        ownerKind: backlogKind,
        ownerName: backlogName,
      }),
    enabled: hasParams && enabled,
    refetchInterval: enabled && agentRunIsActive ? POLL_INTERVAL_MS : false,
  });

  const entries = useMemo(
    () => mergeTimeline(executions, activities),
    [executions, activities],
  );

  return {
    entries,
    isLoading: (execLoading || actLoading) && enabled,
    error: (execError ?? actError),
  };
}
