import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IPlanService } from "../../../services/plan-service";
import type { PlanBoardData } from "../types";
import {
  createPlanDataInitialState,
  resetPlanRequestState,
  resetPlanStoreService,
  setPlanStoreService,
  usePlanDataStore,
} from "./plan-data-store";

function makeBoard(overrides?: Partial<PlanBoardData>): PlanBoardData {
  return {
    now: { activeCount: 1, queueDepth: 0, maxQueueDepth: 10, lanes: [] },
    next: { groups: [], cardCount: 0 },
    later: { groups: [], cardCount: 0 },
    done: { groups: [], cardCount: 0 },
    meta: { generatedAt: "2026-07-02T00:00:00Z", windowSeconds: 86400, maxWave: 0, cycles: [], eta: null },
    ...overrides,
  };
}

function stubService(impl: Partial<IPlanService>): IPlanService {
  return {
    getBoard: vi.fn().mockResolvedValue(makeBoard()),
    listCanonicalPlans: vi.fn().mockResolvedValue([]),
    importPlan: vi.fn().mockRejectedValue(new Error("not implemented in plan-data-store tests")),
    ...impl,
  };
}

function boardOnlyService(getBoard: IPlanService["getBoard"]): IPlanService {
  return stubService({ getBoard });
}

function resetStore() {
  resetPlanRequestState();
  usePlanDataStore.setState({ ...createPlanDataInitialState() });
}

describe("plan-data-store", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    resetPlanStoreService();
    resetStore();
  });

  it("fetches and stores the board", async () => {
    const service = stubService({});
    setPlanStoreService(service);

    await usePlanDataStore.getState().fetchBoard();

    const state = usePlanDataStore.getState();
    expect(state.board?.now.activeCount).toBe(1);
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
    expect(state.fetchedAtMs).not.toBeNull();
  });

  it("records errors without clearing a previous board", async () => {
    const service = stubService({});
    setPlanStoreService(service);
    await usePlanDataStore.getState().fetchBoard();

    setPlanStoreService(stubService({
      getBoard: vi.fn().mockRejectedValue(new Error("boom")),
    }));
    await usePlanDataStore.getState().fetchBoard({ force: true });

    const state = usePlanDataStore.getState();
    expect(state.error).toBe("boom");
    expect(state.board).not.toBeNull();
  });

  it("skips refetch while fresh unless forced", async () => {
    const getBoard = vi.fn().mockResolvedValue(makeBoard());
    setPlanStoreService(boardOnlyService(getBoard));

    await usePlanDataStore.getState().fetchBoard();
    await usePlanDataStore.getState().fetchBoard();
    expect(getBoard).toHaveBeenCalledTimes(1);

    await usePlanDataStore.getState().fetchBoard({ force: true });
    expect(getBoard).toHaveBeenCalledTimes(2);
  });

  it("passes the configured window to the service and refetches on change", async () => {
    const getBoard = vi.fn().mockResolvedValue(makeBoard());
    setPlanStoreService(boardOnlyService(getBoard));

    usePlanDataStore.getState().setWindowSeconds(3600);
    await vi.waitFor(() => {
      expect(getBoard).toHaveBeenCalled();
    });
    expect(getBoard).toHaveBeenCalledWith(expect.objectContaining({ windowSeconds: 3600 }));
  });

  it("ignores stale responses when a newer request superseded them", async () => {
    let resolveFirst: ((board: PlanBoardData) => void) | undefined;
    const first = new Promise<PlanBoardData>((resolve) => {
      resolveFirst = resolve;
    });
    const getBoard = vi
      .fn()
      .mockImplementationOnce(() => first)
      .mockResolvedValueOnce(makeBoard({
        now: { activeCount: 9, queueDepth: 0, maxQueueDepth: 10, lanes: [] },
      }));
    setPlanStoreService(boardOnlyService(getBoard));

    const p1 = usePlanDataStore.getState().fetchBoard();
    const p2 = usePlanDataStore.getState().fetchBoard({ force: true });
    resolveFirst?.(makeBoard({ now: { activeCount: 1, queueDepth: 0, maxQueueDepth: 10, lanes: [] } }));
    await Promise.all([p1, p2]);

    expect(usePlanDataStore.getState().board?.now.activeCount).toBe(9);
  });
});
