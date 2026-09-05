import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { test } from "vitest";
import { RunStatus } from "../../src/types.js";
import { useRunEventStore } from "../../src/hooks/useRunEventStore.js";
import { makeRun } from "../testutil/runs.js";
import { makeMessageEvent } from "../testutil/runEvents.js";

test("useRunEventStore exposes durable subscription, event, snapshot, and reconciliation actions", () => {
  const { result } = renderHook(() => useRunEventStore());
  act(() => result.current.actions.subscribeRun("run-1"));
  act(() => result.current.actions.runEventReceived(makeMessageEvent("event-1", 1n, "first")));
  assert.deepEqual(result.current.getRunEvents("run-1").map((event) => event.id), ["event-1"]);
  act(() => result.current.actions.runStatusReceived({ id: "run-1", status: RunStatus.COMPLETE }));
  assert.deepEqual(result.current.reconciliationIntents, [{ runId: "run-1", afterSequence: 1n, reason: "terminal" }]);
  act(() => result.current.actions.eventsGapFilled("run-1", [makeMessageEvent("event-2", 2n, "second")]));
  assert.equal(result.current.getRunEvents("run-1").length, 2);
  assert.equal(result.current.reconciliationIntents.length, 0);
  act(() => result.current.actions.runsSnapshotLoaded([makeRun({ id: "run-2", status: RunStatus.RUNNING })]));
  assert.equal(result.current.state.runsById["run-2"]?.status, RunStatus.RUNNING);
  act(() => result.current.actions.subscribeAll());
  act(() => result.current.actions.disconnected());
  act(() => result.current.actions.connected());
  assert.equal(result.current.state.reconnectGeneration, 1);
  act(() => result.current.actions.unsubscribeAll());
  assert.equal(result.current.state.allEventsSubscribed, false);
  act(() => result.current.actions.unsubscribeRun("run-1"));
  assert.equal(result.current.state.subscribedRunIds.size, 0);
});
