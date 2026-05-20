import { describe, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "../../test-utils/a11y";
import { selectors } from "../../consts/selectors";
import type { InventoryScan } from "../../api/inventory";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  const scenario: InventoryScan = {
    scenario: "ui-health",
    surfaces: [
      {
        scenario: "ui-health",
        slot: "DashboardPage",
        kind: "page",
        displayName: "DashboardPage",
        description: "main dashboard surface",
        filePath: "ui/src/pages/DashboardPage.tsx",
      },
    ],
    provenance: [],
    widgets: [],
    scannedAt: "2026-05-20T10:00:00.000Z",
  };
  const scanScenario = vi.fn(() => Promise.resolve(scenario));
  return { ...actual, scanScenario };
});

import { InventoryPage } from "./InventoryPage";
import { SurfaceDetailPage } from "./SurfaceDetailPage";

beforeEach(() => {
  window.localStorage.clear();
});

describe("Inventory feature accessibility", () => {
  it("InventoryPage has no axe violations (empty state)", async () => {
    const { container } = renderWithProviders(<InventoryPage />);
    await expectNoA11yViolations(container);
  });

  it("InventoryPage has no axe violations after a scan", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<InventoryPage />);
    await user.type(screen.getByTestId(selectors.inventory.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.inventory.submit));
    await waitFor(() =>
      screen.getByTestId(selectors.inventory.surfacesTable),
    );
    await expectNoA11yViolations(container);
  });

  it("SurfaceDetailPage has no axe violations", async () => {
    const { container, findByTestId } = renderWithProviders(
      <Routes>
        <Route path="/inventory/:surfaceId" element={<SurfaceDetailPage />} />
      </Routes>,
      { routerEntries: ["/inventory/ui-health__DashboardPage"] },
    );
    await findByTestId(selectors.inventory.detail.meta);
    await expectNoA11yViolations(container);
  });
});
