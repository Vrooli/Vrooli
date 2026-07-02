import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";
import type { IOperationsService } from "../../../services/operations-service";
import {
  resetOperationsStoreService,
  setOperationsStoreService,
  useOperationsStore,
} from "../../../stores/operations-store";
import type { ActivityRow, OperationsView } from "../../../types/operations";
import { NowColumn } from "./NowColumn";

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: "act-1",
    ownerType: "backlog",
    ownerName: "owner",
    purpose: "process",
    status: "running",
    requestedAt: "2026-07-02T00:00:00Z",
    ...overrides,
  };
}

function view(activities: ActivityRow[]): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 0, capacity: 6, queue: 0 },
      { lane: "execute", active: activities.length, capacity: 3, queue: 0 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 2, maxDepth: 50 },
    activities,
    recentlyFinished: [],
    generatedAt: "2026-07-02T00:00:00Z",
    windowSeconds: 10800,
  };
}

function service(activities: ActivityRow[]): IOperationsService {
  return {
    fetchOperations: vi.fn().mockResolvedValue(view(activities)),
    bulkStop: vi.fn().mockResolvedValue({ outcomes: [], total: 0, stopped: 0, failed: 0 }),
  };
}

async function loadView(activities: ActivityRow[]) {
  setOperationsStoreService(service(activities));
  await act(async () => {
    await useOperationsStore.getState().refresh({ force: true });
  });
}

describe("NowColumn", () => {
  beforeEach(() => {
    useOperationsStore.getState().reset();
  });

  afterEach(() => {
    resetOperationsStoreService();
    useOperationsStore.getState().reset();
  });

  it("groups activities by initiative with a standalone bucket", async () => {
    await loadView([
      row({ activityId: "a1", runId: "r1", initiativeName: "apollo", ownerTitle: "Apollo item", lane: "execute" }),
      row({ activityId: "a2", runId: "r2", ownerTitle: "Loose item", lane: "review" }),
    ]);
    renderWithProviders(<NowColumn />);

    expect(screen.getByTestId("plan-now-group-apollo")).toHaveTextContent("Apollo item");
    expect(screen.getByTestId("plan-now-group-standalone")).toHaveTextContent("Loose item");
    expect(screen.getAllByTestId(selectors.operationsCenter.laneBar)).toHaveLength(4);
  });

  it("switches to by-phase lane buckets", async () => {
    await loadView([
      row({ activityId: "a1", runId: "r1", ownerTitle: "Exec item", lane: "execute" }),
    ]);
    useOperationsStore.getState().setViewMode("by-phase");
    renderWithProviders(<NowColumn />);

    expect(screen.getByTestId("plan-now-lane-group-execute")).toHaveTextContent("Exec item");
    expect(screen.getByTestId("plan-now-lane-group-review")).toHaveTextContent("idle");
  });

  it("select mode shows row checkboxes and toggles selection", async () => {
    await loadView([
      row({ activityId: "a1", runId: "r1", ownerTitle: "Stoppable", lane: "execute" }),
    ]);
    const user = userEvent.setup();
    renderWithProviders(<NowColumn />);

    await user.click(screen.getByTestId(selectors.plan.nowSelectToggle));
    expect(useOperationsStore.getState().selectionMode).toBe(true);

    const checkbox = screen.getByRole("checkbox");
    await user.click(checkbox);
    expect(useOperationsStore.getState().selection.has("r1")).toBe(true);
  });

  it("shows the empty state with spawn CTA when idle", async () => {
    setOperationsStoreService({
      fetchOperations: vi.fn().mockResolvedValue({
        ...view([]),
        queue: { depth: 0, maxDepth: 50 },
      }),
      bulkStop: vi.fn(),
    });
    await act(async () => {
      await useOperationsStore.getState().refresh({ force: true });
    });
    renderWithProviders(<NowColumn />);

    expect(screen.getByTestId(selectors.plan.nowEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.nowSpawnCta)).toBeInTheDocument();
  });

  it("refresh button forces a refetch", async () => {
    const svc = service([row({ activityId: "a1", runId: "r1", lane: "execute" })]);
    setOperationsStoreService(svc);
    await act(async () => {
      await useOperationsStore.getState().refresh({ force: true });
    });
    const user = userEvent.setup();
    renderWithProviders(<NowColumn />);

    await user.click(screen.getByTestId(selectors.plan.nowRefresh));
    expect(svc.fetchOperations).toHaveBeenCalledTimes(2);
  });
});
