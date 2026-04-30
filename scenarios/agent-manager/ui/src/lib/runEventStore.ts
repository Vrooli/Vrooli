import type { Run, RunEvent, Task } from "../types";

export type RunSnapshot = Partial<Run> & { id: string };

export type RunEventReconciliationReason = "first_load" | "reconnect" | "terminal";

export interface RunEventReconciliationIntent {
  runId: string;
  afterSequence?: bigint;
  reason: RunEventReconciliationReason;
}

export interface RunEventStoreState {
  runsById: Record<string, RunSnapshot>;
  eventsByRunId: Record<string, RunEvent[]>;
  lastSequenceByRunId: Record<string, bigint>;
  subscribedRunIds: Set<string>;
  allEventsSubscribed: boolean;
  reconnectGeneration: number;
  taskStatusesById: Record<string, Partial<Task> & { id: string }>;
  reconciliationIntentsByRunId: Record<string, RunEventReconciliationIntent>;
  connected: boolean;
}

export type RunEventStoreAction =
  | { type: "runStatusReceived"; run: RunSnapshot }
  | { type: "runEventReceived"; event: RunEvent }
  | { type: "eventsGapFilled"; runId: string; events: RunEvent[] }
  | { type: "runSnapshotLoaded"; run: RunSnapshot }
  | { type: "runsSnapshotLoaded"; runs: RunSnapshot[] }
  | { type: "taskStatusReceived"; task: Partial<Task> & { id: string } }
  | { type: "subscribeRun"; runId: string }
  | { type: "unsubscribeRun"; runId: string }
  | { type: "subscribeAll" }
  | { type: "unsubscribeAll" }
  | { type: "connected" }
  | { type: "disconnected" }
  | { type: "clearReconciliationIntent"; runId: string };

const TERMINAL_RUN_STATUSES = new Set<unknown>([4, 5, 6, 7]);

export function createInitialRunEventStoreState(): RunEventStoreState {
  return {
    runsById: {},
    eventsByRunId: {},
    lastSequenceByRunId: {},
    subscribedRunIds: new Set(),
    allEventsSubscribed: false,
    reconnectGeneration: 0,
    taskStatusesById: {},
    reconciliationIntentsByRunId: {},
    connected: false,
  };
}

export function selectRunEvents(state: RunEventStoreState, runId: string): RunEvent[] {
  return state.eventsByRunId[runId] ?? [];
}

export function selectReconciliationIntents(state: RunEventStoreState): RunEventReconciliationIntent[] {
  return Object.values(state.reconciliationIntentsByRunId);
}

function eventSequence(event: RunEvent): bigint | undefined {
  return typeof event.sequence === "bigint" ? event.sequence : undefined;
}

function sortEvents(events: RunEvent[]): RunEvent[] {
  return [...events].sort((left, right) => {
    const leftSeq = eventSequence(left);
    const rightSeq = eventSequence(right);
    if (leftSeq !== undefined && rightSeq !== undefined && leftSeq !== rightSeq) {
      return leftSeq < rightSeq ? -1 : 1;
    }
    if (leftSeq !== undefined && rightSeq === undefined) return -1;
    if (leftSeq === undefined && rightSeq !== undefined) return 1;
    return left.id.localeCompare(right.id);
  });
}

function mergeEvents(existing: RunEvent[], incoming: RunEvent[]): RunEvent[] {
  const byId = new Set(existing.map((event) => event.id).filter(Boolean));
  const bySequence = new Set(existing.map(eventSequence).filter((sequence) => sequence !== undefined));
  const merged = [...existing];

  for (const event of incoming) {
    if (event.id && byId.has(event.id)) {
      continue;
    }
    const sequence = eventSequence(event);
    if (sequence !== undefined && bySequence.has(sequence)) {
      continue;
    }
    merged.push(event);
    if (event.id) {
      byId.add(event.id);
    }
    if (sequence !== undefined) {
      bySequence.add(sequence);
    }
  }

  return sortEvents(merged);
}

function lastSequence(events: RunEvent[]): bigint | undefined {
  let last: bigint | undefined;
  for (const event of events) {
    const sequence = eventSequence(event);
    if (sequence !== undefined && (last === undefined || sequence > last)) {
      last = sequence;
    }
  }
  return last;
}

function withEvents(state: RunEventStoreState, runId: string, events: RunEvent[]): RunEventStoreState {
  const nextLastSequence = lastSequence(events);
  const nextLastSequenceByRunId = { ...state.lastSequenceByRunId };
  if (nextLastSequence !== undefined) {
    nextLastSequenceByRunId[runId] = nextLastSequence;
  }

  return {
    ...state,
    eventsByRunId: {
      ...state.eventsByRunId,
      [runId]: events,
    },
    lastSequenceByRunId: nextLastSequenceByRunId,
  };
}

function withReconciliationIntent(
  state: RunEventStoreState,
  runId: string,
  reason: RunEventReconciliationReason
): RunEventStoreState {
  return {
    ...state,
    reconciliationIntentsByRunId: {
      ...state.reconciliationIntentsByRunId,
      [runId]: {
        runId,
        afterSequence: state.lastSequenceByRunId[runId],
        reason,
      },
    },
  };
}

function clearReconciliationIntent(state: RunEventStoreState, runId: string): RunEventStoreState {
  if (!state.reconciliationIntentsByRunId[runId]) {
    return state;
  }
  const next = { ...state.reconciliationIntentsByRunId };
  delete next[runId];
  return {
    ...state,
    reconciliationIntentsByRunId: next,
  };
}

function shouldTrackRunEvents(state: RunEventStoreState, runId: string): boolean {
  return state.allEventsSubscribed || state.subscribedRunIds.has(runId);
}

export function runEventStoreReducer(
  state: RunEventStoreState,
  action: RunEventStoreAction
): RunEventStoreState {
  switch (action.type) {
    case "runSnapshotLoaded":
      return {
        ...state,
        runsById: {
          ...state.runsById,
          [action.run.id]: action.run,
        },
      };
    case "runsSnapshotLoaded":
      return {
        ...state,
        runsById: {
          ...state.runsById,
          ...Object.fromEntries(action.runs.map((run) => [run.id, run])),
        },
      };
    case "runStatusReceived": {
      const existing = state.runsById[action.run.id];
      const next = {
        ...state,
        runsById: {
          ...state.runsById,
          [action.run.id]: existing ? { ...existing, ...action.run } : action.run,
        },
      };
      return TERMINAL_RUN_STATUSES.has(action.run.status) && shouldTrackRunEvents(next, action.run.id)
        ? withReconciliationIntent(next, action.run.id, "terminal")
        : next;
    }
    case "runEventReceived": {
      const runId = action.event.runId;
      if (!runId) return state;
      if (!shouldTrackRunEvents(state, runId)) return state;
      const merged = mergeEvents(state.eventsByRunId[runId] ?? [], [action.event]);
      return withEvents(state, runId, merged);
    }
    case "eventsGapFilled": {
      const merged = mergeEvents(state.eventsByRunId[action.runId] ?? [], action.events);
      return clearReconciliationIntent(withEvents(state, action.runId, merged), action.runId);
    }
    case "taskStatusReceived":
      return {
        ...state,
        taskStatusesById: {
          ...state.taskStatusesById,
          [action.task.id]: {
            ...state.taskStatusesById[action.task.id],
            ...action.task,
          },
        },
      };
    case "subscribeRun": {
      if (state.subscribedRunIds.has(action.runId)) {
        return state;
      }
      const subscribedRunIds = new Set(state.subscribedRunIds);
      subscribedRunIds.add(action.runId);
      return withReconciliationIntent({ ...state, subscribedRunIds }, action.runId, "first_load");
    }
    case "unsubscribeRun": {
      if (!state.subscribedRunIds.has(action.runId)) {
        return state;
      }
      const subscribedRunIds = new Set(state.subscribedRunIds);
      subscribedRunIds.delete(action.runId);
      return clearReconciliationIntent({ ...state, subscribedRunIds }, action.runId);
    }
    case "subscribeAll":
      return state.allEventsSubscribed ? state : { ...state, allEventsSubscribed: true };
    case "unsubscribeAll":
      return state.allEventsSubscribed ? { ...state, allEventsSubscribed: false } : state;
    case "connected": {
      let next: RunEventStoreState = {
        ...state,
        connected: true,
        reconnectGeneration: state.reconnectGeneration + 1,
      };
      for (const runId of state.subscribedRunIds) {
        next = withReconciliationIntent(next, runId, "reconnect");
      }
      return next;
    }
    case "disconnected":
      return state.connected ? { ...state, connected: false } : state;
    case "clearReconciliationIntent":
      return clearReconciliationIntent(state, action.runId);
    default:
      return state;
  }
}
