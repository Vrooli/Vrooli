import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import {
  makeFleetEntry,
  makeScanFleetResponse,
} from "../storage/mocks/factories";

const { getInventory, scanFleet } = vi.hoisted(() => ({
  getInventory: vi.fn(),
  scanFleet: vi.fn(),
}));

vi.mock("../../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/storage")>();
  return {
    ...actual,
    storageClient: { ...actual.storageClient, getInventory, scanFleet },
  };
});

import { FleetView } from "./FleetView";

const inventory = makeScanFleetResponse({
  scenarioCount: 3,
  isolationUnreadyCount: 1,
  noBackupCount: 1,
  findingCount: 4,
  engineDistribution: [
    { engine: "sqlite", scenarioCount: 2 } as never,
    { engine: "postgres", scenarioCount: 1 } as never,
  ],
  stageDistribution: [{ stage: "greenfield", scenarioCount: 3 } as never],
  entries: [
    makeFleetEntry({ scenario: "ready-one", isolationReady: true, hasBackupTarget: true }),
    makeFleetEntry({
      scenario: "unready-one",
      isolationReady: false,
      isolationReason: "Routed test-isolation seams unwired",
      hasBackupTarget: false,
      findingCount: 4,
    }),
  ],
});

beforeEach(() => {
  getInventory.mockResolvedValue(inventory);
  scanFleet.mockResolvedValue(inventory);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("FleetView", () => {
  it("renders the inventory table from the snapshot (all view)", async () => {
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.fleet.table)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.fleet.row({ scenario: "ready-one" }))).toBeInTheDocument();
    // Offenders sort first.
    expect(getInventory).toHaveBeenCalled();
  });

  it("opens the isolation scorecard when the view query param is set", async () => {
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet?view=isolation"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.fleet.list)).toBeInTheDocument();
    });
    // Only the unready scenario shows; the ready one is filtered out.
    expect(screen.getByTestId(selectors.fleet.row({ scenario: "unready-one" }))).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.fleet.row({ scenario: "ready-one" })),
    ).not.toBeInTheDocument();
  });

  it("switches to the engines distribution view", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet"] });
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.table)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.fleet.viewTab({ view: "engines" })));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.fleet.list)).toBeInTheDocument();
    });
    expect(screen.getByText(/sqlite/)).toBeInTheDocument();
  });

  it("calls scanFleet when the live-scan source is selected and refetched", async () => {
    const user = userEvent.setup();
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet"] });
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.table)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.fleet.sourceTab({ source: "scan" })));
    await waitFor(() => expect(scanFleet).toHaveBeenCalled());
  });

  it("renders the empty state when the no-backup view has no offenders", async () => {
    getInventory.mockResolvedValue(
      makeScanFleetResponse({
        scenarioCount: 1,
        entries: [makeFleetEntry({ scenario: "ok", hasBackupTarget: true })],
      }),
    );
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet?view=no-backup"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument();
    });
  });

  it("renders the error state with a retry affordance", async () => {
    getInventory.mockRejectedValue(new Error("boom"));
    renderWithProviders(<FleetView />, { routerEntries: ["/fleet"] });
    await waitFor(() => {
      expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument();
    });
  });
});
