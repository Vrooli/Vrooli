/**
 * Derives live counts for the starter-action cards on an empty session.
 *
 * Counts come from the *same stores and converters the picker uses*
 * ({@link buildContextOptionsByType}), so a card's badge equals the picker's
 * selectable set by construction. Only the stores the current kind's cards
 * actually need are fetched — opening the picker later reuses these caches.
 */
import { useEffect, useMemo } from "react";
import { useBacklogStore, useExecutionStore, useInitiativeStore } from "../../../stores";
import type { AgentSessionContextType, AgentSessionKind, ExecutionRecord } from "../../../types";
import { buildContextOptionsByType } from "./session-context-options";
import type { SessionContextOption } from "./session-context-refs";
import { countableTypesForKind } from "../session-starter-suggestions";

export interface StarterContextCounts {
  /** Per-type option lists, built identically to the picker (countable types only). */
  optionsByType: Record<AgentSessionContextType, SessionContextOption[]>;
  /** Raw executions, for failed/stale narrowing that needs entity timestamps. */
  executions: ExecutionRecord[];
  /** True while a countable type's backing fetch is still resolving (show skeleton, not "0"). */
  loading: Partial<Record<AgentSessionContextType, boolean>>;
}

export function useStarterContextCounts(sessionKind: AgentSessionKind): StarterContextCounts {
  const neededTypes = useMemo(() => new Set(countableTypesForKind(sessionKind)), [sessionKind]);

  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const backlogItems = useBacklogStore((s) => s.items);
  const backlogStatus = useBacklogStore((s) => s.status);
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const executions = useExecutionStore((s) => s.items);
  const executionStatus = useExecutionStore((s) => s.status);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  const initiatives = useInitiativeStore((s) => s.items);
  const initiativeStatus = useInitiativeStore((s) => s.status);

  useEffect(() => {
    if (neededTypes.has("backlog_item")) void fetchBacklog();
    if (neededTypes.has("execution")) void fetchExecutions();
    if (neededTypes.has("initiative")) void fetchInitiatives();
  }, [neededTypes, fetchBacklog, fetchExecutions, fetchInitiatives]);

  const optionsByType = useMemo(
    () =>
      buildContextOptionsByType({
        backlogItems,
        initiatives,
        executions,
        // Types the starter cards never count — picker builds these from its own
        // full subscriptions; empty here keeps the hook's fetch surface minimal.
        captures: [],
        activities: [],
        scenarios: [],
        sessions: [],
        sessionKind,
        currentSessionId: undefined,
      }),
    [backlogItems, initiatives, executions, sessionKind],
  );

  const loading = useMemo<Partial<Record<AgentSessionContextType, boolean>>>(() => {
    // Skeleton while the store is idle/loading AND still empty. Once a fetch
    // succeeds (even with zero items) we show the real "0", never a stuck spinner.
    const pending = (status: string, count: number) => (status === "idle" || status === "loading") && count === 0;
    const result: Partial<Record<AgentSessionContextType, boolean>> = {};
    if (neededTypes.has("backlog_item")) result.backlog_item = pending(backlogStatus, backlogItems.length);
    if (neededTypes.has("execution")) result.execution = pending(executionStatus, executions.length);
    if (neededTypes.has("initiative")) result.initiative = pending(initiativeStatus, initiatives.length);
    return result;
  }, [
    neededTypes,
    backlogStatus,
    backlogItems.length,
    executionStatus,
    executions.length,
    initiativeStatus,
    initiatives.length,
  ]);

  return { optionsByType, executions, loading };
}
