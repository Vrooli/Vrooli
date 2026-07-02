import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";
import type { IPlanService } from "../../../services/plan-service";
import type { IOperationsService } from "../../../services/operations-service";
import {
  resetOperationsStoreService,
  setOperationsStoreService,
  useOperationsStore,
} from "../../../stores/operations-store";
import type { OperationsView } from "../../../types/operations";
import type { PlanBoardData, PlanCardData, PlanCardGroupData } from "../types";
import {
  createPlanDataInitialState,
  resetPlanRequestState,
  resetPlanStoreService,
  setPlanStoreService,
  usePlanDataStore,
} from "../stores/plan-data-store";
import { PlanBoard } from "./PlanBoard";

function emptyOpsView(): OperationsView {
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
    generatedAt: "2026-07-02T00:00:00Z",
    windowSeconds: 10800,
  };
}

function stubOpsService(): IOperationsService {
  return {
    fetchOperations: vi.fn().mockResolvedValue(emptyOpsView()),
    bulkStop: vi.fn().mockResolvedValue({ outcomes: [], total: 0, stopped: 0, failed: 0 }),
  };
}

function itemCard(id: string, overrides?: Partial<PlanCardData>): PlanCardData {
  return {
    id: `backlog-item/fix/${id}`,
    cardType: "item",
    action: "run",
    itemKind: "fix",
    itemName: id,
    title: `${id} title`,
    status: "ready",
    priority: 3,
    wave: 0,
    initiative: "",
    effort: "",
    gate: null,
    outcome: "",
    finishedAt: "",
    executionId: "",
    unblocks: 0,
    ...overrides,
  };
}

function gateCard(id: string, kind: "decide" | "review" | "classify", count: number): PlanCardData {
  return itemCard(id, {
    cardType: "gate",
    action: kind,
    gate: {
      id: `${kind}:backlog/fix/${id}`,
      kind,
      ownerType: "backlog",
      ownerKind: "fix",
      ownerName: id,
      ownerTitle: `${id} title`,
      count,
      blocks: [],
      decidableSince: "",
      suggested: "",
    },
  });
}

function group(id: string, cards: PlanCardData[], overrides?: Partial<PlanCardGroupData>): PlanCardGroupData {
  return { id, label: id, blockerKind: "none", gateId: "", blockerKeys: [], cards, ...overrides };
}

function makeBoard(overrides?: Partial<PlanBoardData>): PlanBoardData {
  return {
    now: {
      activeCount: 2,
      queueDepth: 1,
      maxQueueDepth: 10,
      lanes: [{ lane: "execute", active: 2, capacity: 3 }],
    },
    next: {
      groups: [
        group("gates", [gateCard("questions", "decide", 3)], { label: "Decisions & reviews" }),
        group("ready", [itemCard("runnable")], { label: "Ready to run" }),
      ],
      cardCount: 2,
    },
    later: {
      groups: [
        group("items:fix/runnable", [itemCard("blocked", { wave: 1, action: "workshop" })], {
          label: "after runnable title",
          blockerKind: "items",
          blockerKeys: ["fix/runnable"],
        }),
        group("deep-group", [itemCard("deep", { wave: 7 })], {
          label: "after something deep",
          blockerKind: "items",
        }),
      ],
      cardCount: 2,
    },
    done: {
      groups: [
        group("recent", [
          itemCard("shipped", {
            cardType: "outcome",
            action: "none",
            outcome: "ok",
            finishedAt: "2026-07-02T10:00:00Z",
          }),
        ], { label: "Recent outcomes" }),
      ],
      cardCount: 1,
    },
    meta: { generatedAt: "2026-07-02T12:00:00Z", windowSeconds: 86400, maxWave: 2, cycles: [] },
    ...overrides,
  };
}

function resetStore() {
  resetPlanRequestState();
  usePlanDataStore.setState({ ...createPlanDataInitialState() });
}

function stubService(board: PlanBoardData | Error): IPlanService {
  return {
    getBoard: board instanceof Error
      ? vi.fn().mockRejectedValue(board)
      : vi.fn().mockResolvedValue(board),
  };
}

describe("PlanBoard", () => {
  beforeEach(() => {
    resetStore();
    useOperationsStore.getState().reset();
    setOperationsStoreService(stubOpsService());
  });

  afterEach(() => {
    resetPlanStoreService();
    resetOperationsStoreService();
    useOperationsStore.getState().reset();
    resetStore();
  });

  it("shows a loading state before the first snapshot", async () => {
    let resolveBoard: ((b: PlanBoardData) => void) | undefined;
    setPlanStoreService({
      getBoard: vi.fn().mockImplementation(
        () => new Promise<PlanBoardData>((resolve) => {
          resolveBoard = resolve;
        }),
      ),
    });
    renderWithProviders(<PlanBoard />);

    expect(screen.getByTestId(selectors.plan.boardLoading)).toBeInTheDocument();
    resolveBoard?.(makeBoard());
    expect(await screen.findByTestId(selectors.plan.board)).toBeInTheDocument();
  });

  it("renders all four columns with cards and counts", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />);

    expect(await screen.findByTestId(selectors.plan.board)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.columnNow)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.columnNext)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.columnLater)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.columnDone)).toBeInTheDocument();

    // Gate card is visually distinct (badge) and carries its count.
    expect(screen.getByTestId(selectors.plan.cardGateBadge)).toHaveTextContent("decide 3");
    // Later cards carry wave badges.
    expect(screen.getAllByTestId(selectors.plan.cardWaveBadge).length).toBeGreaterThan(0);
    // Done outcome glyph renders.
    expect(screen.getByTestId(selectors.plan.cardOutcomeGlyph)).toHaveTextContent("✓");
    // Lane utilization bars render (all four canonical lanes).
    expect(await screen.findAllByTestId(selectors.operationsCenter.laneBar)).toHaveLength(4);
  });

  it("rolls deep cards into the beyond-horizon disclosure", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />);

    const horizon = await screen.findByTestId(selectors.plan.beyondHorizon);
    expect(horizon).toHaveTextContent("beyond horizon (1)");
  });

  it("collapses a group on toggle", async () => {
    setPlanStoreService(stubService(makeBoard()));
    const user = userEvent.setup();
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    const readyGroup = screen.getByTestId("plan-group-ready");
    expect(readyGroup).toHaveTextContent("runnable title");

    const toggle = readyGroup.querySelector('[data-testid="plan-group-toggle"]');
    expect(toggle).not.toBeNull();
    await user.click(toggle as HTMLElement);
    expect(readyGroup).not.toHaveTextContent("runnable title");
  });

  it("shows per-column empty states", async () => {
    setPlanStoreService(stubService(makeBoard({
      now: { activeCount: 0, queueDepth: 0, maxQueueDepth: 10, lanes: [] },
      next: { groups: [], cardCount: 0 },
      later: { groups: [], cardCount: 0 },
      done: { groups: [], cardCount: 0 },
    })));
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    expect(screen.getByTestId(selectors.plan.nowEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.nowSpawnCta)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.nextEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.laterEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.doneEmpty)).toBeInTheDocument();
  });

  it("shows the error state with retry when the first fetch fails", async () => {
    setPlanStoreService(stubService(new Error("projection unavailable")));
    renderWithProviders(<PlanBoard />);

    const error = await screen.findByTestId(selectors.plan.boardError);
    expect(error).toHaveTextContent("projection unavailable");
  });

  it("opens the decision drawer from the ?drawer=decisions deep link", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />, { initialEntries: ["/graph/plan?drawer=decisions"] });

    await screen.findByTestId(selectors.plan.board);
    expect(await screen.findByTestId(selectors.plan.decisionDrawer)).toBeInTheDocument();
  });

  it("shows Next-column bulk actions for ready items and pending questions", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    expect(screen.getByTestId(selectors.plan.nextRunAll)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.plan.nextAnswerAll)).toHaveTextContent("3");
  });

  it("surfaces dependency-cycle diagnostics", async () => {
    setPlanStoreService(stubService(makeBoard({
      meta: { generatedAt: "", windowSeconds: 86400, maxWave: 2, cycles: ["fix/a -> fix/b -> fix/a"] },
    })));
    renderWithProviders(<PlanBoard />);

    const warning = await screen.findByTestId(selectors.plan.cycleWarning);
    expect(warning).toHaveTextContent("1 dependency cycle");
  });
});
