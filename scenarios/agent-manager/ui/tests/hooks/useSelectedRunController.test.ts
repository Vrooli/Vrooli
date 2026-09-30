import assert from "node:assert/strict";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useRef } from "react";
import { test, vi } from "vitest";
import { useRunEventStore } from "../../src/hooks/useRunEventStore.js";
import { useSelectedRunController } from "../../src/hooks/useSelectedRunController.js";
import { ApprovalState, RunStatus, type RunDiff } from "../../src/types.js";
import { makeMessageEvent } from "../testutil/runEvents.js";
import { makeRun } from "../testutil/runs.js";
import { makeTask } from "../testutil/tasks.js";

test("useSelectedRunController gap-fills selected run events through the run event store", async () => {
  const run = makeRun({
    id: "run-1",
    taskId: "task-1",
    status: RunStatus.RUNNING,
    approvalState: ApprovalState.NONE,
  });
  const task = makeTask({
    id: "task-1",
    title: "Selected task",
  });
  const restEvents = [
    makeMessageEvent("event-2", 2n, "second"),
    makeMessageEvent("event-1", 1n, "first"),
  ];
  const onGetRun = vi.fn(async () => run);
  const onGetEvents = vi.fn(async () => restEvents);
  const onGetDiff = vi.fn(async () => ({ files: [] }) as unknown as RunDiff);
  const onGetTask = vi.fn(async () => task);
  const wsSubscribe = vi.fn();
  const wsUnsubscribe = vi.fn();
  const runs = [run];
  const tasks = [task];

  const { result, unmount } = renderHook(() => {
    const isDeselectingRef = useRef(false);
    const runEventStore = useRunEventStore();
    const controller = useSelectedRunController({
      runs,
      tasks,
      isDeselectingRef,
      onGetRun,
      onGetEvents,
      onGetDiff,
      onGetTask,
      runEventStore,
      wsSubscribe,
      wsUnsubscribe,
    });

    return { controller, runEventStore };
  });

  await act(async () => {
    await result.current.controller.loadRunDetails(run);
  });

  await waitFor(() => {
    assert.deepEqual(
      result.current.controller.events.map((event) => event.id),
      ["event-1", "event-2"],
    );
  });

  assert.equal(result.current.controller.selectedRun?.id, "run-1");
  assert.equal(result.current.controller.getTaskTitle("task-1"), "Selected task");
  assert.deepEqual(onGetEvents.mock.calls[0], ["run-1"]);
  assert.equal(result.current.runEventStore.state.lastSequenceByRunId["run-1"], 2n);
  assert.equal(onGetDiff.mock.calls.length, 0);
  assert.equal(onGetTask.mock.calls.length, 0);
  assert.equal(wsSubscribe.mock.calls[0]?.[0], "run-1");

  unmount();
  assert.equal(wsUnsubscribe.mock.calls[0]?.[0], "run-1");
});
