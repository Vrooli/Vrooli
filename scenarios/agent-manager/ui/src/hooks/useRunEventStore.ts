import { useCallback, useMemo, useReducer } from "react";
import type { Run, RunEvent, Task } from "../types";
import {
  createInitialRunEventStoreState,
  runEventStoreReducer,
  selectReconciliationIntents,
  selectRunEvents,
  type RunSnapshot,
} from "../lib/runEventStore";

export function useRunEventStore() {
  const [state, dispatch] = useReducer(runEventStoreReducer, undefined, createInitialRunEventStoreState);

  const actions = useMemo(
    () => ({
      runStatusReceived: (run: RunSnapshot) => dispatch({ type: "runStatusReceived", run }),
      runEventReceived: (event: RunEvent) => dispatch({ type: "runEventReceived", event }),
      eventsGapFilled: (runId: string, events: RunEvent[]) => dispatch({ type: "eventsGapFilled", runId, events }),
      runSnapshotLoaded: (run: Run) => dispatch({ type: "runSnapshotLoaded", run }),
      runsSnapshotLoaded: (runs: Run[]) => dispatch({ type: "runsSnapshotLoaded", runs }),
      taskStatusReceived: (task: Partial<Task> & { id: string }) => dispatch({ type: "taskStatusReceived", task }),
      subscribeRun: (runId: string) => dispatch({ type: "subscribeRun", runId }),
      unsubscribeRun: (runId: string) => dispatch({ type: "unsubscribeRun", runId }),
      subscribeAll: () => dispatch({ type: "subscribeAll" }),
      unsubscribeAll: () => dispatch({ type: "unsubscribeAll" }),
      connected: () => dispatch({ type: "connected" }),
      disconnected: () => dispatch({ type: "disconnected" }),
      clearReconciliationIntent: (runId: string) => dispatch({ type: "clearReconciliationIntent", runId }),
    }),
    []
  );

  const getRunEvents = useCallback((runId: string) => selectRunEvents(state, runId), [state]);
  const reconciliationIntents = useMemo(() => selectReconciliationIntents(state), [state]);

  return {
    state,
    actions,
    getRunEvents,
    reconciliationIntents,
  };
}

export type UseRunEventStoreReturn = ReturnType<typeof useRunEventStore>;
