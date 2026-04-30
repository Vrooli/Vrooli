import test from "node:test";
import assert from "node:assert/strict";
import type { RunEvent } from "../../src/types.js";
import {
  createInitialRunEventStoreState,
  runEventStoreReducer,
  selectReconciliationIntents,
  selectRunEvents,
} from "../../src/lib/runEventStore.js";

function makeEvent(id: string, sequence: bigint, runId = "run-1"): RunEvent {
  return {
    id,
    runId,
    sequence,
    eventType: 2,
    data: { case: "message", value: { role: "assistant", content: id } },
  } as RunEvent;
}

test("runEventReceived appends events in sequence order", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-2", 2n) });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-1", 1n) });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1", "event-2"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 2n);
});

test("runEventReceived ignores duplicates by event id and then sequence", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-1", 1n) });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-1", 2n) });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-2", 1n) });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 1n);
});

test("eventsGapFilled merges missing REST events and clears reconciliation intent", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-1", 1n) });
  state = runEventStoreReducer(state, {
    type: "eventsGapFilled",
    runId: "run-1",
    events: [makeEvent("event-2", 2n), makeEvent("event-1", 1n), makeEvent("event-3", 3n)],
  });

  assert.deepEqual(selectRunEvents(state, "run-1").map((event) => event.id), ["event-1", "event-2", "event-3"]);
  assert.equal(state.lastSequenceByRunId["run-1"], 3n);
  assert.deepEqual(selectReconciliationIntents(state), []);
});

test("terminal run status records targeted reconciliation after the latest sequence", () => {
  let state = createInitialRunEventStoreState();

  state = runEventStoreReducer(state, { type: "subscribeRun", runId: "run-1" });
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-4", 4n) });
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

  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-1", 1n) });
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
  state = runEventStoreReducer(state, { type: "runEventReceived", event: makeEvent("event-6", 6n) });
  state = runEventStoreReducer(state, { type: "connected" });

  assert.equal(state.subscribedRunIds.has("run-1"), true);
  assert.equal(state.reconnectGeneration, 1);
  assert.deepEqual(selectReconciliationIntents(state), [
    { runId: "run-1", afterSequence: 6n, reason: "reconnect" },
  ]);
});
