import { describe, expect, it, beforeEach } from "vitest";
import {
  setOperationsStoreService,
  resetOperationsStoreService,
  useOperationsStore,
  operationsStoreInitialFilters,
} from "./operations-store";
import type { IOperationsService } from "../services/operations-service";
import type { OperationsView } from "../types/operations";

function fakeView(): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: 0, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 0, maxDepth: 50 },
    activities: [],
    recentlyFinished: [],
    generatedAt: "2026-05-02T00:00:00Z",
    windowSeconds: 10800,
  };
}

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
});
