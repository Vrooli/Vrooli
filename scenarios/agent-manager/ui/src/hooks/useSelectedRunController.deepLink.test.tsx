import { create } from "@bufbuild/protobuf";
import { renderHook, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { RunSchema } from "@vrooli/proto-types/agent-manager/v1/domain/run_pb";
import { RunStatus } from "../types";
import type { UseRunEventStoreReturn } from "./useRunEventStore";
import { useSelectedRunController } from "./useSelectedRunController";

test("loads a deep-linked historical run that is absent from the recent page", async () => {
  const run = create(RunSchema, { id: "historical-run", taskId: "task-1", status: RunStatus.PENDING });
  const actions = {
    runSnapshotLoaded: vi.fn(),
    subscribeRun: vi.fn(),
    unsubscribeRun: vi.fn(),
    eventsGapFilled: vi.fn(),
  };
  const store = {
    state: { runsById: {}, lastSequenceByRunId: {} },
    actions,
    getRunEvents: vi.fn(() => []),
  } as unknown as UseRunEventStoreReturn;
  const onGetRun = vi.fn().mockResolvedValue(run);
  const onGetEvents = vi.fn().mockResolvedValue([]);

  const { result } = renderHook(() => useSelectedRunController({
    runs: [],
    tasks: [],
    routeRunId: "historical-run",
    isDeselectingRef: { current: false },
    onGetRun,
    onGetEvents,
    onGetDiff: vi.fn(),
    onGetTask: vi.fn().mockResolvedValue({ id: "task-1", title: "Historical task" }),
    runEventStore: store,
    wsSubscribe: vi.fn(),
    wsUnsubscribe: vi.fn(),
  }));

  await waitFor(() => expect(result.current.selectedRun?.id).toBe("historical-run"));
  expect(onGetRun).toHaveBeenCalledWith("historical-run");
  expect(onGetEvents).toHaveBeenCalledWith("historical-run");
});
