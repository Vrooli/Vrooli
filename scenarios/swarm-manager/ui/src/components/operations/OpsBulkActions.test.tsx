import { describe, it, expect, beforeEach, vi } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  resetOperationsStoreService,
  setOperationsStoreService,
  useOperationsStore,
} from "../../stores/operations-store";
import type { IOperationsService } from "../../services/operations-service";
import type {
  ActivityRow,
  BulkStopRequest,
  BulkStopResponse,
  OperationsView,
} from "../../types/operations";
import { selectors } from "../../consts/selectors";
import { OpsBulkActions } from "./OpsBulkActions";

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: overrides.activityId ?? "act-1",
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

function view(activities: ActivityRow[] = []): OperationsView {
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

function makeService(
  opts: {
    bulkStop?: (req: BulkStopRequest) => Promise<BulkStopResponse>;
    activeRows?: ActivityRow[];
  } = {},
): { service: IOperationsService; bulkStopSpy: ReturnType<typeof vi.fn> } {
  const bulkStopSpy = vi.fn(
    opts.bulkStop ??
      (async () => ({ outcomes: [], total: 0, stopped: 0, failed: 0 })),
  );
  const service: IOperationsService = {
    fetchOperations: async () => view(opts.activeRows ?? []),
    bulkStop: bulkStopSpy,
  };
  return { service, bulkStopSpy };
}

beforeEach(() => {
  useOperationsStore.getState().reset();
  resetOperationsStoreService();
});

describe("OpsBulkActions", () => {
  it("returns null when nothing is selected and no rows are active", () => {
    const { service } = makeService();
    setOperationsStoreService(service);

    const { container } = render(<OpsBulkActions />);
    expect(container.firstChild).toBeNull();
  });

  it("renders the bar with 'Stop all running' when there are active rows", async () => {
    const { service } = makeService({
      activeRows: [row({ runId: "run-a", lane: "execute" })],
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();

    render(<OpsBulkActions />);
    expect(screen.getByTestId(selectors.operationsCenter.bulkActionBar)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.operationsCenter.bulkStopAll),
    ).toBeEnabled();
    // "Stop selected" is disabled when nothing is selected.
    expect(
      screen.getByTestId(selectors.operationsCenter.bulkStopSelected),
    ).toBeDisabled();
  });

  it("enables Stop selected and shows count when the operator selects rows", async () => {
    const { service } = makeService({
      activeRows: [row({ runId: "run-a" })],
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();

    useOperationsStore.getState().setSelection(["run-a", "run-b"]);
    render(<OpsBulkActions />);

    expect(
      screen.getByTestId(selectors.operationsCenter.bulkStopSelected),
    ).toBeEnabled();
    expect(screen.getByText(/Stop selected \(2\)/)).toBeInTheDocument();
    expect(screen.getByText(/2 selected/)).toBeInTheDocument();
  });

  it("calls bulkStopSelected after operator confirms", async () => {
    const { service, bulkStopSpy } = makeService({
      activeRows: [row({ runId: "run-a" })],
      bulkStop: async () => ({
        outcomes: [{ runId: "run-a", success: true }],
        total: 1,
        stopped: 1,
        failed: 0,
      }),
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();
    useOperationsStore.getState().setSelection(["run-a"]);

    const user = userEvent.setup();
    render(<OpsBulkActions />);

    await user.click(screen.getByTestId(selectors.operationsCenter.bulkStopSelected));
    // ConfirmDialog opens; click "Stop 1".
    const confirmButton = await screen.findByRole("button", { name: /^Stop 1$/ });
    await user.click(confirmButton);

    await waitFor(() => {
      expect(bulkStopSpy).toHaveBeenCalledTimes(1);
    });
    expect(bulkStopSpy.mock.calls[0]?.[0]).toEqual({ runIds: ["run-a"] });
  });

  it("requires typing STOP ALL to confirm the global stop", async () => {
    const { service, bulkStopSpy } = makeService({
      activeRows: [
        row({ runId: "run-a", lane: "execute" }),
        row({ runId: "run-b", lane: "investigate" }),
      ],
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();

    const user = userEvent.setup();
    render(<OpsBulkActions />);

    await user.click(screen.getByTestId(selectors.operationsCenter.bulkStopAll));
    const confirmButton = await screen.findByRole("button", { name: /^Stop 2$/ });
    expect(confirmButton).toBeDisabled();

    const input = screen.getByPlaceholderText("STOP ALL");
    await user.type(input, "STOP ALL");
    expect(confirmButton).toBeEnabled();

    await user.click(confirmButton);
    await waitFor(() => {
      expect(bulkStopSpy).toHaveBeenCalledTimes(1);
    });
    expect(bulkStopSpy.mock.calls[0]?.[0]).toEqual({ filter: {} });
  });

  it("renders an outcome panel after a stop call resolves", async () => {
    const { service } = makeService({
      activeRows: [row({ runId: "run-a" })],
      bulkStop: async () => ({
        outcomes: [
          { runId: "run-a", success: true },
          { runId: "run-b", success: false, error: "manager unreachable" },
        ],
        total: 2,
        stopped: 1,
        failed: 1,
      }),
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();
    useOperationsStore.getState().setSelection(["run-a", "run-b"]);

    const user = userEvent.setup();
    render(<OpsBulkActions />);

    await user.click(screen.getByTestId(selectors.operationsCenter.bulkStopSelected));
    const confirmButton = await screen.findByRole("button", { name: /^Stop 2$/ });
    await user.click(confirmButton);

    const outcome = await screen.findByTestId(
      selectors.operationsCenter.bulkStopOutcomeToast,
    );
    expect(outcome).toHaveTextContent(/Stopped 1 of 2; 1 failed/);
    expect(outcome).toHaveTextContent(/manager unreachable/);
  });

  it("clearing selection from the store hides the selection-specific actions", async () => {
    const { service } = makeService({
      activeRows: [row({ runId: "run-a" })],
    });
    setOperationsStoreService(service);
    await useOperationsStore.getState().refresh();
    useOperationsStore.getState().setSelection(["run-a"]);

    render(<OpsBulkActions />);
    expect(screen.queryByTestId(selectors.operationsCenter.bulkClearSelection)).toBeInTheDocument();

    act(() => {
      useOperationsStore.getState().clearSelection();
    });

    expect(
      screen.queryByTestId(selectors.operationsCenter.bulkClearSelection),
    ).toBeNull();
  });
});
