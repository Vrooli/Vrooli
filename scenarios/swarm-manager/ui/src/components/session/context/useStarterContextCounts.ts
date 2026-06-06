/**
 * Derives live counts for the starter-action cards on an empty session.
 *
 * Counts come from the *same stores and converters the picker uses*
 * ({@link buildContextOptionsByType}), so a card's badge equals the picker's
 * selectable set by construction. Only the stores the current kind's cards
 * actually need are fetched — opening the picker later reuses these caches.
 */
import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { initiativeModeService } from "../../../services";
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

  const wantsModes = neededTypes.has("operating_mode");
  const modesQuery = useQuery({
    queryKey: ["operating-modes", "catalog"],
    queryFn: () => initiativeModeService.catalog(),
    enabled: wantsModes,
  });
  // Read scalars before combining — TanStack's result union otherwise narrows
  // `data` to `never` when `isLoading` is used inline in the same expression.
  const modesData = modesQuery.data?.modes;
  const modes = useMemo(() => modesData ?? [], [modesData]);
  const modesLoading = modesQuery.isLoading;
  const modesCount = modes.length;

  useEffect(() => {
    if (neededTypes.has("backlog_item")) void fetchBacklog();
    if (neededTypes.has("execution")) void fetchExecutions();
    if (neededTypes.has("initiative")) void fetchInitiatives();
    // operating_mode is fetched by the React Query above (enabled-gated).
  }, [neededTypes, fetchBacklog, fetchExecutions, fetchInitiatives]);

  const optionsByType = useMemo(
    () =>
      buildContextOptionsByType({
        backlogItems,
        initiatives,
        executions,
        modes,
        // Types the starter cards never count — picker builds these from its own
        // full subscriptions; empty here keeps the hook's fetch surface minimal.
        captures: [],
        activities: [],
        scenarios: [],
        sessions: [],
        sessionKind,
        currentSessionId: undefined,
      }),
    [backlogItems, initiatives, executions, modes, sessionKind],
  );

  const loading = useMemo<Partial<Record<AgentSessionContextType, boolean>>>(() => {
    // Skeleton while the store is idle/loading AND still empty. Once a fetch
    // succeeds (even with zero items) we show the real "0", never a stuck spinner.
    const pending = (status: string, count: number) => (status === "idle" || status === "loading") && count === 0;
    const result: Partial<Record<AgentSessionContextType, boolean>> = {};
    if (neededTypes.has("backlog_item")) result.backlog_item = pending(backlogStatus, backlogItems.length);
    if (neededTypes.has("execution")) result.execution = pending(executionStatus, executions.length);
    if (neededTypes.has("initiative")) result.initiative = pending(initiativeStatus, initiatives.length);
    if (neededTypes.has("operating_mode")) {
      result.operating_mode = modesLoading && modesCount === 0;
    }
    return result;
  }, [
    neededTypes,
    backlogStatus,
    backlogItems.length,
    executionStatus,
    executions.length,
    initiativeStatus,
    initiatives.length,
    modesLoading,
    modesCount,
  ]);

  return { optionsByType, executions, loading };
}
