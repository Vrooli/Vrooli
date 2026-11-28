import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { useRunEventStore } from "./useRunEventStore";
import { useRunReconciliation } from "./useRunReconciliation";
import { makeRun } from "../test-utils/fixtures/runs";
import { makeMessageEvent } from "../test-utils/fixtures/runEvents";
import { RunStatus, type Run } from "../types";

beforeEach(() => {
  vi.useFakeTimers();
  vi.spyOn(console, "error").mockImplementation(() => {});
});
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); });

function setup(getRun = vi.fn().mockResolvedValue(makeRun()), getRunEvents = vi.fn().mockResolvedValue([])) {
  const view = renderHook(() => {
    const store = useRunEventStore();
    useRunReconciliation({ intents: store.reconciliationIntents, actions: store.actions, getRun, getRunEvents });
    return store;
  });
  const subscribe = async () => {
    await act(async () => {
      view.result.current.actions.runSnapshotLoaded(makeRun({ status: RunStatus.RUNNING, updatedAt: timestampFromMs(1000) }));
      view.result.current.actions.subscribeRun("run-1");
      view.result.current.actions.connected();
    });
  };
  return { ...view, subscribe, getRun, getRunEvents };
}

test("a single reconnect read failure recovers terminal state without another socket event", async () => {
  const terminal = makeRun({ status: RunStatus.COMPLETE, updatedAt: timestampFromMs(2000) });
  const getRun = vi.fn().mockRejectedValueOnce(new Error("temporary")).mockResolvedValue(terminal);
  const event = makeMessageEvent("answer", 9n, "42");
  const view = setup(getRun, vi.fn().mockResolvedValue([event]));
  await view.subscribe();
  expect(view.result.current.state.connected).toBe(true);
  expect(view.result.current.state.runsById["run-1"].status).toBe(RunStatus.RUNNING);
  expect(view.result.current.reconciliationIntents).toHaveLength(1);
  await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
  expect(view.result.current.state.runsById["run-1"].status).toBe(RunStatus.COMPLETE);
  expect(view.result.current.getRunEvents("run-1")).toEqual([event]);
  expect(view.result.current.reconciliationIntents).toHaveLength(0);
  expect(view.result.current.state.reconnectGeneration).toBe(1);
  view.unmount();
});

test("repeated failures back off to 30 seconds and unsubscribe cancels retries", async () => {
  const view = setup(vi.fn().mockRejectedValue(new Error("offline")));
  await view.subscribe();
  let calls = 1;
  for (const delay of [1000, 2000, 4000, 8000, 16000, 30000, 30000]) {
    // Unrelated store renders must not restart or accelerate this intent.
    act(() => view.result.current.actions.runsSnapshotLoaded([makeRun({ id: "other" })]));
    await act(async () => { await vi.advanceTimersByTimeAsync(delay - 1); });
    expect(view.getRun).toHaveBeenCalledTimes(calls);
    await act(async () => { await vi.advanceTimersByTimeAsync(1); });
    expect(view.getRun).toHaveBeenCalledTimes(++calls);
  }
  act(() => view.result.current.actions.unsubscribeRun("run-1"));
  await act(async () => { await vi.advanceTimersByTimeAsync(120_000); });
  expect(view.getRun).toHaveBeenCalledTimes(calls);
  expect(view.result.current.reconciliationIntents).toHaveLength(0);
  view.unmount();
});

test("unsubscribe and unmount abort in-flight paired HTTP reads", async () => {
  const view = setup(vi.fn().mockImplementation(() => new Promise(() => {})), vi.fn().mockImplementation(() => new Promise(() => {})));
  await view.subscribe();
  const first = view.getRun.mock.calls[0][1].signal as AbortSignal;
  expect(view.getRunEvents.mock.calls[0][1].signal).toBe(first);
  act(() => view.result.current.actions.unsubscribeRun("run-1"));
  expect(first.aborted).toBe(true);
  await view.subscribe();
  const second = view.getRun.mock.calls[1][1].signal as AbortSignal;
  view.unmount();
  expect(second.aborted).toBe(true);
  await act(async () => { await vi.advanceTimersByTimeAsync(120_000); });
  expect(view.getRun).toHaveBeenCalledTimes(2);
});

test("a newer terminal intent excludes a delayed older response", async () => {
  let finishOld!: (run: Run) => void;
  const terminal = makeRun({ status: RunStatus.COMPLETE, updatedAt: timestampFromMs(2000) });
  const getRun = vi.fn().mockImplementationOnce(() => new Promise<Run>((resolve) => { finishOld = resolve; })).mockResolvedValue(terminal);
  const view = setup(getRun);
  await view.subscribe();
  const oldSignal = getRun.mock.calls[0][1].signal as AbortSignal;
  await act(async () => { view.result.current.actions.runStatusReceived(terminal); });
  expect(oldSignal.aborted).toBe(true);
  expect(view.result.current.reconciliationIntents).toHaveLength(0);
  await act(async () => { finishOld(makeRun({ status: RunStatus.RUNNING, updatedAt: timestampFromMs(1000) })); });
  expect(view.result.current.state.runsById["run-1"].status).toBe(RunStatus.COMPLETE);
  expect(getRun).toHaveBeenCalledTimes(2);
  view.unmount();
});
