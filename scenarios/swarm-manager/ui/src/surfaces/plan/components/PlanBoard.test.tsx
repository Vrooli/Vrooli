import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from "vitest";
import { Route, Routes, useLocation } from "react-router-dom";
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
import { defaultApiClient } from "../../../lib/api-client";

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

// These new cases use real plan/operations services and stores. Only their
// network boundary is controlled; earlier service-injection tests stay intact.
describe("PlanBoard actual service deep-link reconciliation", () => {
  const clients: ReturnType<typeof createTestQueryClient>[] = [];
  const unexpected: string[] = [];
  let boardResponse: () => Promise<unknown>;
  let get: MockInstance<typeof defaultApiClient.get>;
  const originalScroll = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
  beforeEach(() => {
    resetStore(); useOperationsStore.getState().reset(); resetOperationsStoreService(); resetPlanStoreService();
    unexpected.length = 0; boardResponse = async () => makeBoard();
    // jsdom lacks this browser affordance; keep the real scroll effect and observe it.
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: vi.fn() });
    get = vi.spyOn(defaultApiClient, "get").mockImplementation(async path => {
      if (path === "/plan" || path.startsWith("/plan?")) return boardResponse();
      if (path === "/operations" || path.startsWith("/operations?")) return emptyOpsView();
      if (path === "/next-actions/feed") return { entries: [] };
      if (path === "/goals") return { items: [] };
      if (path === "/backlog/summary") return { pending_questions: { items: [] } };
      if (path === "/settings") return { settings: {} };
      if (path === "/proposal-sessions" || path.startsWith("/proposal-sessions?")) return { sessions: [] };
      unexpected.push(path); throw new Error(`Unexpected plan read ${path}`);
    });
    vi.spyOn(defaultApiClient, "post").mockImplementation(async path => {
      if (path === "/backlog/next-actions") return { results: [] };
      throw new Error(`Unexpected plan write ${path}`);
    });
    vi.spyOn(defaultApiClient, "patch").mockRejectedValue(new Error("Unexpected plan patch"));
    vi.spyOn(defaultApiClient, "delete").mockRejectedValue(new Error("Unexpected plan delete"));
  });
  afterEach(() => {
    clients.splice(0).forEach(c => c.clear()); resetStore(); useOperationsStore.getState().reset();
    expect(unexpected).toEqual([]); expect(defaultApiClient.patch).not.toHaveBeenCalled(); expect(defaultApiClient.delete).not.toHaveBeenCalled();
    expect(vi.mocked(defaultApiClient.post).mock.calls.every(([path]) => path === "/backlog/next-actions")).toBe(true);
    vi.restoreAllMocks(); resetPlanStoreService(); resetOperationsStoreService();
    if (originalScroll) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", originalScroll); else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
  });
  function mountLink(path: string) {
    const client = createTestQueryClient(); clients.push(client);
    return renderWithProviders(<><Routes><Route path="/plan" element={<PlanBoard />} /><Route path="*" element={<div data-testid="destination" />} /></Routes><LocationProbe /></>, { queryClient: client, initialEntries: [path] });
  }
  function location() { return screen.getByTestId("location-probe").textContent ?? ""; }
  it("consumes only select/focus, highlights the exact present card and retains unrelated URL state", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Frunnable&focus=old&retained=owner");
    const card = await screen.findByTestId("plan-card-backlog-item/fix/runnable");
    await waitFor(() => expect(card).toHaveClass("ring-2"));
    await waitFor(() => expect(location()).not.toContain("select=")); expect(location()).not.toContain("focus="); expect(location()).toContain("retained=owner"); expect(screen.queryByTestId("plan-select-miss")).toBeNull();
    expect(usePlanDataStore.getState().board?.next.groups[1]?.cards[0]?.itemName).toBe("runnable");
  });
  it("reveals and highlights a beyond-horizon target instead of reporting a false miss", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Fdeep");
    const card = await screen.findByTestId("plan-card-backlog-item/fix/deep"); await waitFor(() => expect(card).toHaveClass("ring-2"));
    expect(screen.queryByTestId("plan-select-miss")).toBeNull(); await waitFor(() => expect(location()).not.toContain("select="));
  });
  it("offers exact details for a missing target without replacing the board or filters", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Fmissing&q=owner");
    expect(await screen.findByTestId("plan-select-miss")).toHaveTextContent("missing");
    expect(screen.getByTestId("plan-select-miss-details")).toHaveAttribute("href", "/backlog/fix/missing");
    expect(location()).toContain("q=owner"); await waitFor(() => expect(location()).not.toContain("select="));
    expect(screen.getByTestId("plan-card-backlog-item/fix/runnable")).toBeVisible();
    fireEvent.click(screen.getByTestId("plan-select-miss-details")); expect(location()).toContain("/backlog/fix/missing");
  });
  it("opens the graph with the exact missed focus and selection without a mutation", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Fmissing"); await screen.findByTestId("plan-select-miss");
    fireEvent.click(screen.getByTestId("plan-select-miss-graph"));
    await waitFor(() => expect(new URL(location(), "http://fixture.invalid").pathname).toBe("/graph"));
    const url = new URL(location(), "http://fixture.invalid"); expect(url.pathname).toBe("/graph"); expect(url.searchParams.get("focus")).toBe("backlog-item/fix/missing"); expect(url.searchParams.get("select")).toBe("backlog-item/fix/missing");
  });
  it("dismisses a missing target once without changing the server board or consuming unrelated parameters", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Fmissing&retained=owner"); await screen.findByTestId("plan-select-miss");
    const before = usePlanDataStore.getState().board; fireEvent.click(screen.getByTestId("plan-select-miss-dismiss"));
    expect(screen.queryByTestId("plan-select-miss")).toBeNull(); expect(usePlanDataStore.getState().board).toBe(before); expect(location()).toContain("retained=owner");
    await act(async () => usePlanDataStore.getState().fetchBoard({ force: true })); expect(screen.queryByTestId("plan-select-miss")).toBeNull();
  });
  it("clears filters only when requested and promotes an exact newly returned target to a highlight", async () => {
    mountLink("/plan?select=backlog-item%2Ffix%2Fmissing&q=owner&lane=execute"); await screen.findByTestId("plan-select-miss");
    fireEvent.click(screen.getByTestId("plan-select-miss-clear")); await waitFor(() => expect(location()).not.toContain("q=")); expect(location()).not.toContain("lane=");
    boardResponse = async () => makeBoard({ next: { groups: [group("revealed", [itemCard("missing")])], cardCount: 1 } });
    await act(async () => usePlanDataStore.getState().fetchBoard({ force: true }));
    await waitFor(() => expect(screen.queryByTestId("plan-select-miss")).toBeNull()); expect(screen.getByTestId("plan-card-backlog-item/fix/missing")).toHaveClass("ring-2");
    expect(get.mock.calls.filter(([path]) => typeof path === "string" && path.startsWith("/plan" )).length).toBeGreaterThan(1);
  });
  it("keeps selection unconsumed while loading and reconciles only after the real service returns a board", async () => {
    let resolve!: (value: unknown) => void; boardResponse = () => new Promise(r => { resolve = r; });
    mountLink("/plan?select=backlog-item%2Ffix%2Frunnable&retained=owner");
    expect(screen.getByTestId(selectors.plan.boardLoading)).toBeVisible(); expect(location()).toContain("select="); expect(screen.queryByTestId("plan-select-miss")).toBeNull();
    await act(async () => resolve(makeBoard())); await waitFor(() => expect(screen.getByTestId("plan-card-backlog-item/fix/runnable")).toHaveClass("ring-2")); await waitFor(() => expect(location()).not.toContain("select="));
  });
  it("preserves the requested target through read refusal and consumes it after an explicit retry succeeds", async () => {
    boardResponse = async () => { throw new Error("Owner projection refused"); }; mountLink("/plan?select=backlog-item%2Ffix%2Frunnable");
    expect(await screen.findByTestId(selectors.plan.boardError)).toHaveTextContent("Owner projection refused"); expect(location()).toContain("select=");
    boardResponse = async () => makeBoard(); fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(screen.getByTestId("plan-card-backlog-item/fix/runnable")).toHaveClass("ring-2")); await waitFor(() => expect(location()).not.toContain("select="));
  });
  it("keeps the last authoritative board when a later refresh is refused", async () => {
    mountLink("/plan"); await screen.findByTestId(selectors.plan.board); const before = usePlanDataStore.getState().board;
    boardResponse = async () => { throw new Error("Refresh authority unavailable"); };
    await act(async () => usePlanDataStore.getState().fetchBoard({ force: true }));
    expect(usePlanDataStore.getState().board).toBe(before); expect(screen.getByTestId("plan-card-backlog-item/fix/runnable")).toBeVisible(); expect(usePlanDataStore.getState().error).toBe("Refresh authority unavailable");
  });
});
