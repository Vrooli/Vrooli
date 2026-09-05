import { test } from "vitest";
import assert from "node:assert/strict";
import {
  createInitialRunEventStoreState,
  runEventStoreReducer,
  selectReconciliationIntents,
  selectRunEvents,
} from "../../src/lib/runEventStore.js";
import { makeMessageEvent } from "../testutil/runEvents.js";

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
