import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { makeFleetEntry, makeScanFleetResponse } from "../storage/mocks/factories";

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

import { DashboardView } from "./DashboardView";

const populated = makeScanFleetResponse({
  scenarioCount: 3,
  isolationUnreadyCount: 1,
  noBackupCount: 1,
  findingCount: 4,
  scannedAt: new Date(Date.now() - 60_000).toISOString(),
  engineDistribution: [{ engine: "sqlite", scenarioCount: 2 } as never],
  entries: [
    makeFleetEntry({ scenario: "ready", isolationReady: true }),
    makeFleetEntry({
      scenario: "risky",
      isolationReady: false,
      isolationReason: "Routed seams unwired",
    }),
  ],
});

beforeEach(() => {
  getInventory.mockResolvedValue(populated);
  scanFleet.mockResolvedValue(populated);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("DashboardView", () => {
  it("renders the hero stat band from the inventory snapshot", async () => {
    renderWithProviders(<DashboardView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.band)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.dashboard.statScenarios)).toHaveTextContent("3");
    expect(screen.getByTestId(selectors.dashboard.statIsolationUnready)).toHaveTextContent("1");
  });

  it("previews the isolation scorecard with the unready scenario", async () => {
    renderWithProviders(<DashboardView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.scorecard)).toBeInTheDocument();
    });
    expect(screen.getByText(/risky/)).toBeInTheDocument();
  });

  it("renders the engine distribution and snapshot freshness", async () => {
    renderWithProviders(<DashboardView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.engines)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.dashboard.freshness)).toBeInTheDocument();
  });

  it("renders the empty CTA when the snapshot has no scenarios", async () => {
    getInventory.mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 }));
    renderWithProviders(<DashboardView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.empty)).toBeInTheDocument();
    });
  });

  it("triggers a live scan from the empty CTA", async () => {
    const user = userEvent.setup();
    getInventory.mockResolvedValue(makeScanFleetResponse({ scenarioCount: 0 }));
    renderWithProviders(<DashboardView />);
    await waitFor(() => expect(screen.getByTestId(selectors.dashboard.empty)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.state.emptyAction));
    await waitFor(() => expect(scanFleet).toHaveBeenCalled());
  });

  it("renders the error state when the inventory fetch fails", async () => {
    getInventory.mockRejectedValue(new Error("boom"));
    renderWithProviders(<DashboardView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.dashboard.error)).toBeInTheDocument();
    });
  });
});
