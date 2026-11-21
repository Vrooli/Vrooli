import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import type { InventoryScan, SurfaceRecord } from "../../api/inventory";

const makeSurface = (overrides: Partial<SurfaceRecord> = {}): SurfaceRecord => ({
  scenario: "ui-health",
  slot: "DashboardPage",
  kind: "page",
  displayName: "DashboardPage",
  description: "the dashboard",
  filePath: "ui/src/pages/DashboardPage.tsx",
  ...overrides,
});

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  const scanScenario = vi.fn((scenario: string): Promise<InventoryScan> => {
    if (scenario === "broken") return Promise.reject(new Error("boom"));
    if (scenario === "empty") {
      return Promise.resolve({
        scenario,
        surfaces: [],
        provenance: [],
        widgets: [],
        scannedAt: new Date().toISOString(),
      });
    }
    return Promise.resolve({
      scenario,
      surfaces: [
        makeSurface(),
        makeSurface({ slot: "Header", kind: "component", displayName: "Header" }),
      ],
      provenance: [],
      widgets: [],
      scannedAt: new Date().toISOString(),
    });
  });
  return { ...actual, scanScenario };
});

import { InventoryPage } from "./InventoryPage";
import { scanScenario } from "../../api/inventory";

beforeEach(() => {
  vi.mocked(scanScenario).mockClear();
});

describe("InventoryPage", () => {
  it("renders empty state before any scan", () => {
    renderWithProviders(<InventoryPage />);
    expect(screen.getByTestId(selectors.inventory.form)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.inventory.empty)).toBeInTheDocument();
    expect(scanScenario).not.toHaveBeenCalled();
  });

  it("rejects an invalid scenario name without calling the API", async () => {
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "Bad Name!");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    expect(scanScenario).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("scans a valid scenario and renders the surfaces table", async () => {
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    await waitFor(() => expect(scanScenario).toHaveBeenCalledWith("ui-health"));
    const table = await screen.findByTestId(selectors.inventory.surfacesTable);
    expect(within(table).getAllByRole("row").length - 1).toBe(2); // minus header
  });

  it("narrows the table by kind filter", async () => {
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    await screen.findByTestId(selectors.inventory.surfacesTable);
    await user.click(screen.getByTestId(selectors.inventory.kindFilter({ kind: "page" })));
    await waitFor(() => {
      const table = screen.getByTestId(selectors.inventory.surfacesTable);
      expect(within(table).getAllByRole("row").length - 1).toBe(1);
    });
  });

  it("shows the no-surfaces state for an empty scan", async () => {
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "empty");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    await waitFor(() => expect(scanScenario).toHaveBeenCalledWith("empty"));
    expect(await screen.findByTestId(selectors.inventory.noSurfaces)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.inventory.surfacesTable)).not.toBeInTheDocument();
  });

  it("renders the error state when the scan throws", async () => {
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "broken");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    expect(await screen.findByTestId(selectors.inventory.error)).toBeInTheDocument();
  });
});
