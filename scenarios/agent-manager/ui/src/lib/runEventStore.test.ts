import { test } from "vitest";
import assert from "node:assert/strict";
import {
  createInitialRunEventStoreState,
  runEventStoreReducer,
  selectReconciliationIntents,
  selectRunEvents,
} from "./runEventStore.js";
import { makeMessageEvent } from "../test-utils/fixtures/runEvents.js";
import { parseWebSocketMessage } from "./webSocketProtocol.js";
import type { RunEvent } from "../types";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { create } from "@bufbuild/protobuf";
import { RunSchema, RunResultSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";

test("delayed snapshots and notifications cannot rewind a newer run revision", () => {
  const earlier = { ...timestampFromMs(1000), nanos: 10 };
  const later = { ...timestampFromMs(1000), nanos: 11 };
  for (const type of ["runSnapshotLoaded", "runsSnapshotLoaded", "runStatusReceived"] as const) {
    let state = runEventStoreReducer(createInitialRunEventStoreState(), {
      type: "runStatusReceived",
      run: { id: "run-1", status: 5, updatedAt: later, title: "completed" },
    });
    const stale = { id: "run-1", status: 3, updatedAt: earlier, title: "running" };
    state = runEventStoreReducer(state, type === "runsSnapshotLoaded"
      ? { type, runs: [stale] }
      : { type, run: stale });
    assert.equal(state.runsById["run-1"].status, 5, type);
    assert.equal(state.runsById["run-1"].title, "completed", type);
  }
});

test("a newer continued run can advance a terminal snapshot", () => {
  let state = runEventStoreReducer(createInitialRunEventStoreState(), {
    type: "runSnapshotLoaded", run: { id: "run-1", status: 5, updatedAt: timestampFromMs(1000) },
  });
  state = runEventStoreReducer(state, {
    type: "runSnapshotLoaded", run: { id: "run-1", status: 3, updatedAt: timestampFromMs(2000) },
  });
  assert.equal(state.runsById["run-1"].status, 3);
  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "clearReconciliationIntent", runId: "run-1" });
  const delayed = runEventStoreReducer(state, {
    type: "runStatusReceived", run: { id: "run-1", status: 5, updatedAt: timestampFromMs(1000) },
  });
  assert.equal(delayed, state);
  assert.deepEqual(selectReconciliationIntents(delayed), []);
});

test("an earlier gap response cannot clear a later terminal reconciliation", () => {
  let state = runEventStoreReducer(createInitialRunEventStoreState(), { type: "subscribeRun", runId: "run-1" });
  const first = selectReconciliationIntents(state)[0];
  state = runEventStoreReducer(state, { type: "runStatusReceived", run: { id: "run-1", status: 5 } });
  const terminal = selectReconciliationIntents(state)[0];
  state = runEventStoreReducer(state, {
    type: "eventsGapFilled", runId: "run-1", events: [makeMessageEvent("event-1", 1n, "earlier")], intent: first,
  });
  assert.deepEqual(selectReconciliationIntents(state), [terminal]);
  assert.equal(selectRunEvents(state, "run-1").length, 1);
  state = runEventStoreReducer(state, { type: "clearReconciliationIntent", runId: "run-1", intent: first });
  assert.deepEqual(selectReconciliationIntents(state), [terminal]);
  state = runEventStoreReducer(state, { type: "eventsGapFilled", runId: "run-1", events: [], intent: terminal });
  assert.deepEqual(selectReconciliationIntents(state), []);
});

test("equal revisions merge partial notification metadata with hydrated detail", () => {
  const updatedAt = timestampFromMs(1000);
  let state = runEventStoreReducer(createInitialRunEventStoreState(), {
    type: "runStatusReceived", run: { id: "run-1", status: 5, updatedAt, taskId: "task-1" },
  });
  state = runEventStoreReducer(state, {
    type: "runSnapshotLoaded", run: { id: "run-1", status: 5, updatedAt, taskId: "task-1", title: "answer" },
  });
  state = runEventStoreReducer(state, {
    type: "runStatusReceived", run: { id: "run-1", status: 5, updatedAt },
  });
  assert.equal(state.runsById["run-1"].taskId, "task-1");
  assert.equal(state.runsById["run-1"].title, "answer");
});

test("newer authoritative snapshots clear absent optional results on continuation", () => {
  const terminal = create(RunSchema, {
    id: "continued", status: 5, updatedAt: timestampFromMs(1000), result: create(RunResultSchema, { finalMessage: "42" }),
  });
  const continued = create(RunSchema, { id: "continued", status: 3, updatedAt: timestampFromMs(2000) });
  for (const type of ["runSnapshotLoaded", "runsSnapshotLoaded"] as const) {
    let state = runEventStoreReducer(createInitialRunEventStoreState(), { type: "runSnapshotLoaded", run: terminal });
    state = runEventStoreReducer(state, type === "runsSnapshotLoaded"
      ? { type, runs: [continued] } : { type, run: continued });
    assert.equal(state.runsById.continued.status, 3);
    assert.equal(state.runsById.continued.result, undefined, type);
  }
});

test("runEventReceived appends events in sequence order", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-2", 2n, "event-2") });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-1", 1n, "event-1") });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1", "event-2"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 2n);
});

test("runEventReceived ignores duplicates by event id and then sequence", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-1", 1n, "event-1") });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-1", 2n, "event-1") });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-2", 1n, "event-2") });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 1n);
});

test("native tool and diagnostic replay merges once across websocket and reconnect gap fill", () => {
  let state = createInitialRunEventStoreState();
  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  const wireEvents = [
    { id: "source-offset-0-tool", sequence: "0", event_type: "RUN_EVENT_TYPE_TOOL_RESULT", tool_result: { tool_call_id: "mixed", outcome_known: false, raw_output_json: '[{"type":"image"}]' } },
    { id: "source-offset-0-warning", sequence: "1", event_type: "RUN_EVENT_TYPE_LOG", log: { level: "warning", message: "Unsupported Codex tool output content retained" } },
    { id: "source-offset-100-tool", sequence: "2", event_type: "RUN_EVENT_TYPE_TOOL_RESULT", tool_result: { tool_call_id: "later", outcome_known: false, output: "valid later" } },
  ];
  const events = wireEvents.map((event) => {
    const message = parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT", run_id: "run-1", run_event: { ...event, run_id: "run-1" } });
    assert.equal(message?.type, "run_event");
    return message!.payload as RunEvent;
  });
  // The failed warning/cursor write can replay already committed evidence.
  // Production App routes these normalized messages to this same collector.
  for (const event of [events[0], events[0], events[1], events[0], events[1], events[2]]) {
    state = runEventStoreReducer(state, { type: "runEventReceived", event });
  }
  state = runEventStoreReducer(state, { type: "connected" });
  assert.deepEqual(selectReconciliationIntents(state), [{ runId: "run-1", afterSequence: 2n, reason: "reconnect" }]);
  state = runEventStoreReducer(state, { type: "eventsGapFilled", runId: "run-1", events });
  const collected = selectRunEvents(state, "run-1");
  assert.deepEqual(collected.map((event) => event.id), wireEvents.map((event) => event.id));
  assert.deepEqual(collected.map((event) => event.sequence), [0n, 1n, 2n]);
  assert.equal((collected[0].data.value as { outcomeKnown: boolean }).outcomeKnown, false);
  assert.equal(collected[1].data.case, "log");
  assert.equal((collected[2].data.value as { toolCallId: string }).toolCallId, "later");
  assert.equal(state.lastSequenceByRunId["run-1"], 2n);
  assert.deepEqual(selectReconciliationIntents(state), []);
});

test("eventsGapFilled merges missing REST events and clears reconciliation intent", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-1", 1n, "event-1") });
  state = runEventStoreReducer(state, {
    type: "eventsGapFilled",
    runId: "run-1",
    events: [
      makeMessageEvent("event-2", 2n, "event-2"),
      makeMessageEvent("event-1", 1n, "event-1"),
      makeMessageEvent("event-3", 3n, "event-3"),
    ],
  });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1", "event-2", "event-3"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 3n);
  assert.deepEqual(selectReconciliationIntents(state), []);
});

test("terminal run status records targeted reconciliation after the latest sequence", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-4", 4n, "event-4") });
  state = runEventStoreReducer(state, {
    type: "runStatusReceived",
    run: { id: "run-1", status: 5 },
  });

  assert.deepEqual(selectReconciliationIntents(state), [
    { runId: "run-1", afterSequence: 4n, reason: "terminal" },
  ]);
});

test("unsubscribed live events are ignored while global statuses update metadata only", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-1", 1n, "event-1") });
  state = runEventStoreReducer(state, {
    type: "runStatusReceived",
    run: { id: "run-1", status: 5 },
  });

  assert.deepEqual(selectRunEvents(state, "run-1"), []);
  assert.equal(state.runsById["run-1"]?.status, 5);
  assert.deepEqual(selectReconciliationIntents(state), []);
});

test("reconnect preserves subscriptions and requests after-sequence gap fill", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeMessageEvent("event-6", 6n, "event-6") });
  state = runEventStoreReducer(state, { type: "connected" });

  assert.equal(state.subscribedRunIds.has("run-1"), true);
  assert.equal(state.reconnectGeneration, 1);
  assert.deepEqual(selectReconciliationIntents(state), [
    { runId: "run-1", afterSequence: 6n, reason: "reconnect" },
  ]);
});

test("run event store merges snapshots and task updates while leaving idempotent transitions unchanged", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "runSnapshotLoaded", run: { id: "run-1", status: 1, title: "initial" } });
  state = runEventStoreReducer(state, { type: "runStatusReceived", run: { id: "run-1", status: 2 } });
  state = runEventStoreReducer(state, {
    type: "runsSnapshotLoaded",
    runs: [{ id: "run-2", status: 3 }, { id: "run-3", status: 4 }],
  });
  state = runEventStoreReducer(state, { type: "taskStatusReceived", task: { id: "task-1", status: 1, title: "first" } });
  state = runEventStoreReducer(state, { type: "taskStatusReceived", task: { id: "task-1", status: 2 } });

  assert.deepEqual(state.runsById["run-1"], { id: "run-1", status: 2, title: "initial" });
  assert.equal(state.runsById["run-2"]?.status, 3);
  assert.deepEqual(state.taskStatusesById["task-1"], { id: "task-1", status: 2, title: "first" });

  const duplicateSubscription = runEventStoreReducer(state, { type: "subscribeRun", runId: "missing" });
  assert.equal(runEventStoreReducer(duplicateSubscription, { type: "subscribeRun", runId: "missing" }), duplicateSubscription);
  assert.equal(runEventStoreReducer(state, { type: "unsubscribeRun", runId: "missing" }), state);
  assert.equal(runEventStoreReducer(state, { type: "disconnected" }), state);
  assert.equal(runEventStoreReducer(state, { type: "clearReconciliationIntent", runId: "missing" }), state);
});

test("run event store supports global subscriptions, eventless messages, and reconnect cleanup", () => {
  let state = createInitialRunEventStoreState();
  state = runEventStoreReducer(state, { type: "subscribeAll" });
  assert.equal(state.allEventsSubscribed, true);
  assert.equal(runEventStoreReducer(state, { type: "subscribeAll" }), state);

  const first = makeMessageEvent("z-event", 2n, "later");
  const eventless = makeMessageEvent("a-event", 1n, "earlier");
  delete (eventless as { sequence?: bigint }).sequence;
  state = runEventStoreReducer(state, { type: "runEventReceived", event: first });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: eventless });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: { ...eventless, id: "" } });
  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["z-event", "", "a-event"]);

  state = runEventStoreReducer(state, { type: "connected" });
  assert.equal(state.connected, true);
  state = runEventStoreReducer(state, { type: "disconnected" });
  assert.equal(state.connected, false);
  state = runEventStoreReducer(state, { type: "unsubscribeAll" });
  assert.equal(state.allEventsSubscribed, false);
  assert.equal(runEventStoreReducer(state, { type: "unsubscribeAll" }), state);
});
