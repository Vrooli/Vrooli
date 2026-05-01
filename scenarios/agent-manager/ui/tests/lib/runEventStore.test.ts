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
