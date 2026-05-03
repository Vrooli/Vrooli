import { describe, expect, it, beforeEach, vi } from "vitest";
import {
  setOperationsStoreService,
  resetOperationsStoreService,
  useOperationsStore,
  operationsStoreInitialFilters,
} from "./operations-store";
import type { IOperationsService } from "../services/operations-service";
import type {
  ActivityRow,
  BulkStopRequest,
  BulkStopResponse,
  OperationsView,
} from "../types/operations";

function fakeView(activities: ActivityRow[] = []): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: 0, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 0, maxDepth: 50 },
    activities,
    recentlyFinished: [],
    generatedAt: "2026-05-02T00:00:00Z",
    windowSeconds: 10800,
  };
}

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: overrides.activityId ?? "a-1",
    runId: overrides.runId,
    ownerType: overrides.ownerType ?? "backlog",
    ownerName: overrides.ownerName ?? "owner",
    purpose: overrides.purpose ?? "process",
    status: overrides.status ?? "running",
    requestedAt: overrides.requestedAt ?? "2026-05-02T00:00:00Z",
    lane: overrides.lane,
    ...overrides,
  };
}

const NOOP_BULK_STOP = async (): Promise<BulkStopResponse> => ({
  outcomes: [],
  total: 0,
  stopped: 0,
  failed: 0,
});

beforeEach(() => {
  useOperationsStore.getState().reset();
  resetOperationsStoreService();
});

describe("operations-store", () => {
  it("populates view, lastRefreshedAt, and clears error on a successful refresh", async () => {
    const view = fakeView();
    const calls: unknown[] = [];
    const svc: IOperationsService = {
      async fetchOperations(filters) {
        calls.push(filters);
        return view;
      },
      bulkStop: NOOP_BULK_STOP,
    };
    setOperationsStoreService(svc);

    await useOperationsStore.getState().refresh();
    const state = useOperationsStore.getState();
    expect(state.view).toBe(view);
    expect(state.error).toBeNull();
    expect(state.isLoading).toBe(false);
    expect(state.isRefreshing).toBe(false);
    expect(state.lastRefreshedAt).not.toBeNull();
    expect(calls).toHaveLength(1);
  });

  it("captures an error when fetchOperations throws", async () => {
    const svc: IOperationsService = {
      async fetchOperations() {
        throw new Error("boom");
      },
      bulkStop: NOOP_BULK_STOP,
    };
    setOperationsStoreService(svc);

    await useOperationsStore.getState().refresh();
    const state = useOperationsStore.getState();
    expect(state.view).toBeNull();
    expect(state.error?.message).toBe("boom");
    expect(state.isLoading).toBe(false);
  });

  it("merges filter updates without dropping unset keys", () => {
    useOperationsStore.getState().setFilters({ q: "auth" });
    expect(useOperationsStore.getState().filters.q).toBe("auth");
    useOperationsStore.getState().setFilters({ statuses: ["running"] });
    const state = useOperationsStore.getState();
    expect(state.filters.q).toBe("auth");
    expect(state.filters.statuses).toEqual(["running"]);
  });

  it("resets filters to the canonical default", () => {
    useOperationsStore.getState().setFilters({ q: "x", lanes: ["execute"] });
    useOperationsStore.getState().resetFilters();
    expect(useOperationsStore.getState().filters).toEqual(
      operationsStoreInitialFilters,
    );
  });

  it("toggles selection idempotently", () => {
    useOperationsStore.getState().toggleSelection("run-1");
    expect(useOperationsStore.getState().selection.has("run-1")).toBe(true);
    useOperationsStore.getState().toggleSelection("run-1");
    expect(useOperationsStore.getState().selection.has("run-1")).toBe(false);
  });

  it("clearSelection empties the set", () => {
    useOperationsStore.getState().setSelection(["a", "b"]);
    expect(useOperationsStore.getState().selection.size).toBe(2);
    useOperationsStore.getState().clearSelection();
    expect(useOperationsStore.getState().selection.size).toBe(0);
  });

  describe("bulkStopSelected", () => {
    it("posts the selected run IDs and clears selection on success", async () => {
      const bulkSpy = vi.fn(async (req: BulkStopRequest): Promise<BulkStopResponse> => {
        expect(req).toEqual({ runIds: ["run-a", "run-b"] });
        return {
          outcomes: [
            { runId: "run-a", success: true },
            { runId: "run-b", success: true },
          ],
          total: 2,
          stopped: 2,
          failed: 0,
        };
      });
      const svc: IOperationsService = {
        fetchOperations: async () => fakeView(),
        bulkStop: bulkSpy,
      };
      setOperationsStoreService(svc);

      useOperationsStore.getState().setSelection(["run-a", "run-b"]);
      const result = await useOperationsStore.getState().bulkStopSelected();

      expect(bulkSpy).toHaveBeenCalledTimes(1);
      expect(result.stopped).toBe(2);
      const state = useOperationsStore.getState();
      expect(state.selection.size).toBe(0);
      expect(state.isBulkStopping).toBe(false);
      expect(state.stoppingRunIds.size).toBe(0);
      expect(state.lastBulkStopResult?.stopped).toBe(2);
    });

    it("marks selected ids as stopping while the call is in flight", async () => {
      let release: (value: BulkStopResponse) => void = () => {};
      const inFlight = new Promise<BulkStopResponse>((resolve) => {
        release = resolve;
      });
      const svc: IOperationsService = {
        fetchOperations: async () => fakeView(),
        bulkStop: () => inFlight,
      };
      setOperationsStoreService(svc);

      useOperationsStore.getState().setSelection(["run-a"]);
      const promise = useOperationsStore.getState().bulkStopSelected();

      // Mid-flight: optimistic state is set.
      const midState = useOperationsStore.getState();
      expect(midState.isBulkStopping).toBe(true);
      expect(midState.stoppingRunIds.has("run-a")).toBe(true);

      release({
        outcomes: [{ runId: "run-a", success: true }],
        total: 1,
        stopped: 1,
        failed: 0,
      });
      await promise;

      const after = useOperationsStore.getState();
      expect(after.isBulkStopping).toBe(false);
      expect(after.stoppingRunIds.has("run-a")).toBe(false);
    });

    it("preserves selection and surfaces error on failure", async () => {
      const svc: IOperationsService = {
        fetchOperations: async () => fakeView(),
        bulkStop: async () => {
          throw new Error("network down");
        },
      };
      setOperationsStoreService(svc);

      useOperationsStore.getState().setSelection(["run-a"]);
      await expect(
        useOperationsStore.getState().bulkStopSelected(),
      ).rejects.toThrow("network down");

      const state = useOperationsStore.getState();
      expect(state.selection.has("run-a")).toBe(true);
      expect(state.isBulkStopping).toBe(false);
      expect(state.stoppingRunIds.size).toBe(0);
      expect(state.error?.message).toBe("network down");
    });

    it("returns an empty result and skips the network call when selection is empty", async () => {
      const bulkSpy = vi.fn();
      const svc: IOperationsService = {
        fetchOperations: async () => fakeView(),
        bulkStop: bulkSpy as unknown as IOperationsService["bulkStop"],
      };
      setOperationsStoreService(svc);

      const result = await useOperationsStore.getState().bulkStopSelected();
      expect(result.total).toBe(0);
      expect(bulkSpy).not.toHaveBeenCalled();
    });
  });

  describe("bulkStopAll", () => {
    it("sends a filter payload and triggers refresh on success", async () => {
      const bulkSpy = vi.fn(async (req: BulkStopRequest): Promise<BulkStopResponse> => {
        expect(req).toEqual({ filter: { lane: "execute" } });
        return {
          outcomes: [{ runId: "run-x", success: true }],
          total: 1,
          stopped: 1,
          failed: 0,
        };
      });
      const fetchSpy = vi.fn(async () => fakeView());
      const svc: IOperationsService = {
        fetchOperations: fetchSpy,
        bulkStop: bulkSpy,
      };
      setOperationsStoreService(svc);

      // First load so the optimistic snapshot can use the active rows.
      await useOperationsStore.getState().refresh();
      fetchSpy.mockClear();

      const result = await useOperationsStore.getState().bulkStopAll({ lane: "execute" });
      expect(result.stopped).toBe(1);
      expect(bulkSpy).toHaveBeenCalledTimes(1);
      // The post-call refresh runs as a fire-and-forget so a microtask
      // tick is enough to observe it.
      await Promise.resolve();
      expect(fetchSpy).toHaveBeenCalledTimes(1);
    });

    it("uses an empty filter when none is provided", async () => {
      const bulkSpy = vi.fn(async (req: BulkStopRequest): Promise<BulkStopResponse> => {
        expect(req).toEqual({ filter: {} });
        return { outcomes: [], total: 0, stopped: 0, failed: 0 };
      });
      const svc: IOperationsService = {
        fetchOperations: async () => fakeView(),
        bulkStop: bulkSpy,
      };
      setOperationsStoreService(svc);

      await useOperationsStore.getState().bulkStopAll();
      expect(bulkSpy).toHaveBeenCalledTimes(1);
    });

    it("optimistically marks visible matching run IDs as stopping", async () => {
      let release: (value: BulkStopResponse) => void = () => {};
      const inFlight = new Promise<BulkStopResponse>((resolve) => {
        release = resolve;
      });
      const view = fakeView([
        row({ activityId: "a", runId: "run-a", lane: "execute", status: "running" }),
        row({ activityId: "b", runId: "run-b", lane: "investigate", status: "running" }),
      ]);
      const svc: IOperationsService = {
        fetchOperations: async () => view,
        bulkStop: () => inFlight,
      };
      setOperationsStoreService(svc);
      await useOperationsStore.getState().refresh();

      const promise = useOperationsStore.getState().bulkStopAll({ lane: "execute" });
      const mid = useOperationsStore.getState();
      expect(mid.stoppingRunIds.has("run-a")).toBe(true);
      expect(mid.stoppingRunIds.has("run-b")).toBe(false);

      release({ outcomes: [], total: 0, stopped: 0, failed: 0 });
      await promise;
    });
  });
});
