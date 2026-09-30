import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useLocation } from "react-router-dom";
import { selectors } from "../../../consts/selectors";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import type { NextActionFeedEntry } from "../../../services/next-action-service";
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

function LocationProbe() {
  const location = useLocation();
  return <span data-testid="location-probe">{location.pathname}{location.search}</span>;
}

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

function feedEntry(name: string): NextActionFeedEntry {
  return {
    entity_kind: "backlog_item",
    entity_ref: `fix/${name}`,
    entity_title: `${name} title`,
    tier: 1,
    action: { id: "decide", compact_label: "Decide", expanded_label: "Decide on proposals", enabled: true },
  } as NextActionFeedEntry;
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
    milestone: "",
    effort: "",
    gate: null,
    outcome: "",
    finishedAt: "",
    executionId: "",
    unblocks: 0,
    ...overrides,
  };
}

function gateCard(id: string, kind: "decide" | "review" | "proposal", count: number): PlanCardData {
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
    meta: { generatedAt: "2026-07-02T12:00:00Z", windowSeconds: 86400, maxWave: 2, cycles: [], eta: null },
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
    listCanonicalPlans: vi.fn().mockResolvedValue([]),
    importPlan: vi.fn().mockRejectedValue(new Error("not implemented in PlanBoard tests")),
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
      listCanonicalPlans: vi.fn().mockResolvedValue([]),
      importPlan: vi.fn().mockRejectedValue(new Error("not implemented in PlanBoard tests")),
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

    // Gate card is visually distinct (badge) and carries its count. The badge
    // shows the display label, never the raw wire enum.
    expect(screen.getByTestId(selectors.plan.cardGateBadge)).toHaveTextContent("Decide 3");
    // Later cards carry wave badges.
    expect(screen.getAllByTestId(selectors.plan.cardWaveBadge).length).toBeGreaterThan(0);
    // Done outcome glyph renders.
    expect(screen.getByTestId(selectors.plan.cardOutcomeGlyph)).toHaveTextContent("✓");
    // Lane utilization bars render (all four canonical lanes).
    expect(await screen.findAllByTestId(selectors.operationsCenter.laneBar)).toHaveLength(4);
  });

  it("renders the ETA strip when the board carries a band", async () => {
    setPlanStoreService(
      stubService(
        makeBoard({
          meta: {
            generatedAt: "2026-07-02T12:00:00Z",
            windowSeconds: 86400,
            maxWave: 2,
            cycles: [],
            eta: {
              p50Hours: 120,
              p80Hours: 240,
              p50Label: "~5 days",
              p80Label: "~10 days",
              basis: "live",
              basisLabel: "27 samples",
              confidence: "high",
              remainingItems: 6,
              laneCapacity: 3,
            },
          },
        }),
      ),
    );
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    const strip = screen.getByTestId("plan-eta-strip");
    // Closed state is a compact merged range; basis stays in the popover.
    expect(screen.getByTestId("plan-eta-label")).toHaveTextContent("~5–10 days");
    expect(strip).not.toHaveTextContent("27 samples");
    expect(strip).toHaveAttribute("title", "ETA ~5 days – ~10 days · 27 samples");
  });

  it("opens the ETA details popover", async () => {
    setPlanStoreService(
      stubService(
        makeBoard({
          meta: {
            generatedAt: "2026-07-02T12:00:00Z",
            windowSeconds: 86400,
            maxWave: 2,
            cycles: [],
            eta: {
              p50Hours: 120,
              p80Hours: 240,
              p50Label: "~5 days",
              p80Label: "~10 days",
              basis: "live",
              basisLabel: "27 samples",
              confidence: "high",
              remainingItems: 6,
              laneCapacity: 3,
            },
          },
        }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    await user.click(screen.getByTestId("plan-eta-strip"));

    expect(await screen.findByTestId("plan-eta-popover")).toHaveTextContent("Remaining");
    expect(screen.getByText("6 items")).toBeInTheDocument();
    expect(screen.getByTestId("plan-eta-stats-link")).toHaveTextContent("Open Stats dashboard");
  });

  it("carries the selected goal when opening Measures from ETA", async () => {
    setPlanStoreService(
      stubService(
        makeBoard({
          meta: {
            generatedAt: "2026-07-02T12:00:00Z",
            windowSeconds: 86400,
            maxWave: 2,
            cycles: [],
            eta: {
              p50Hours: 120,
              p80Hours: 240,
              p50Label: "~5 days",
              p80Label: "~10 days",
              basis: "live",
              basisLabel: "27 samples",
              confidence: "high",
              remainingItems: 6,
              laneCapacity: 3,
            },
          },
        }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(
      <>
        <PlanBoard />
        <LocationProbe />
      </>,
      { initialEntries: ["/plan?goal=goal-x"] },
    );

    await screen.findByTestId(selectors.plan.board);
    await user.click(screen.getByTestId("plan-eta-strip"));
    await user.click(await screen.findByTestId("plan-eta-stats-link"));

    expect(screen.getByTestId("location-probe")).toHaveTextContent("/stats");
  });

  it("offers attach-to-session from the ETA popover", async () => {
    setPlanStoreService(
      stubService(
        makeBoard({
          meta: {
            generatedAt: "2026-07-02T12:00:00Z",
            windowSeconds: 86400,
            maxWave: 2,
            cycles: [],
            eta: {
              p50Hours: 120,
              p80Hours: 240,
              p50Label: "~5 days",
              p80Label: "~10 days",
              basis: "live",
              basisLabel: "27 samples",
              confidence: "high",
              remainingItems: 6,
              laneCapacity: 3,
            },
          },
        }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    await user.click(screen.getByTestId("plan-eta-strip"));
    await screen.findByTestId("plan-eta-popover");

    expect(screen.getByTestId(selectors.agentSessions.entityAttachAction)).toBeInTheDocument();
  });

  it("offers attach-to-session from the dependency-cycle popover", async () => {
    setPlanStoreService(
      stubService(
        makeBoard({
          meta: {
            generatedAt: "2026-07-02T12:00:00Z",
            windowSeconds: 86400,
            maxWave: 2,
            cycles: ["backlog-item/fix/a -> backlog-item/fix/b"],
            eta: null,
          },
        }),
      ),
    );
    const user = userEvent.setup();
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    await user.click(screen.getByTestId("plan-cycle-warning"));
    await screen.findByTestId("plan-cycle-popover");

    expect(screen.getByTestId(selectors.agentSessions.entityAttachAction)).toBeInTheDocument();
  });

  it("keeps filters and refresh out of the plan context row", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);

    expect(screen.queryByTestId(selectors.plan.boardFilters)).toBeNull();
    expect(screen.queryByTestId(selectors.plan.boardRefresh)).toBeNull();
  });

  it("opens the shared filter drawer from plan store state", async () => {
    setPlanStoreService(stubService(makeBoard()));
    usePlanDataStore.setState({ filterDrawerOpen: true });
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);

    expect(screen.getByTestId(selectors.plan.filterDrawer)).toBeInTheDocument();
  });

  it("omits the ETA strip when there is nothing to estimate", async () => {
    setPlanStoreService(stubService(makeBoard())); // meta.eta = null
    renderWithProviders(<PlanBoard />);
    await screen.findByTestId(selectors.plan.board);
    expect(screen.queryByTestId("plan-eta-strip")).not.toBeInTheDocument();
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
    renderWithProviders(<PlanBoard />, { initialEntries: ["/plan?drawer=decisions"] });

    await screen.findByTestId(selectors.plan.board);
    expect(await screen.findByTestId(selectors.plan.decisionDrawer)).toBeInTheDocument();
  });

  it("does not offer bulk Run until the server action projection marks an item runnable", async () => {
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />);

    await screen.findByTestId(selectors.plan.board);
    expect(screen.queryByTestId(selectors.plan.nextRunAll)).not.toBeInTheDocument();
  });

  it("counts pending decisions from the same feed the drawer paginates", async () => {
    // The badge used to sum gate counts off the board while the drawer counted
    // feed entries, so the two advertised different sizes for one queue.
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["next-actions-feed"], { entries: [feedEntry("a"), feedEntry("b")] });
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />, { queryClient });

    await screen.findByTestId(selectors.plan.board);
    expect(screen.getByTestId(selectors.plan.nextAnswerAll)).toHaveTextContent("2");
  });

  it("hides the decisions inbox when the feed is empty", async () => {
    const queryClient = createTestQueryClient();
    queryClient.setQueryData(["next-actions-feed"], { entries: [] });
    setPlanStoreService(stubService(makeBoard()));
    renderWithProviders(<PlanBoard />, { queryClient });

    await screen.findByTestId(selectors.plan.board);
    expect(screen.queryByTestId(selectors.plan.nextAnswerAll)).not.toBeInTheDocument();
  });

  it("surfaces dependency-cycle diagnostics", async () => {
    setPlanStoreService(stubService(makeBoard({
      meta: { generatedAt: "", windowSeconds: 86400, maxWave: 2, cycles: ["fix/a -> fix/b -> fix/a"], eta: null },
    })));
    renderWithProviders(<PlanBoard />);

    const warning = await screen.findByTestId(selectors.plan.cycleWarning);
    // Visible label is terse ("1 cycle"); the full wording lives in title/aria.
    expect(warning).toHaveTextContent("1 cycle");
    expect(warning).not.toHaveTextContent("dependency");
    expect(warning).toHaveAttribute("title", "1 dependency cycle");
  });

  it("opens dependency-cycle details with graph actions", async () => {
    setPlanStoreService(stubService(makeBoard({
      next: { groups: [group("cycle", [itemCard("a"), itemCard("b")])], cardCount: 2 },
      meta: { generatedAt: "", windowSeconds: 86400, maxWave: 2, cycles: ["fix/a -> fix/b -> fix/a"], eta: null },
    })));
    const user = userEvent.setup();
    renderWithProviders(<PlanBoard />);

    await user.click(await screen.findByTestId(selectors.plan.cycleWarning));

    const popover = await screen.findByTestId("plan-cycle-popover");
    expect(popover).toHaveTextContent("a title");
    expect(popover).toHaveTextContent("b title");
    expect(popover).toHaveTextContent("Loops back to a title.");
    expect(popover).not.toHaveTextContent("fix/a");
    expect(screen.getByTestId("plan-cycle-resolve")).toHaveTextContent("Open backlog item");
  });
});
