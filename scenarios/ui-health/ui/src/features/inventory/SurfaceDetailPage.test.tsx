import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import type { InventoryScan } from "../../api/inventory";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  const scanScenario = vi.fn();
  return { ...actual, scanScenario };
});

import { SurfaceDetailPage } from "./SurfaceDetailPage";
import { scanScenario } from "../../api/inventory";

const baseScan: InventoryScan = {
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
  provenance: [
    {
      provenance: "adopted-modified",
      library: "react-component-library",
      libraryVersion: "1.2.3",
      componentName: "Card",
      adoptionId: "adopt-1",
    },
  ],
  widgets: [],
  scannedAt: "2026-05-20T10:00:00.000Z",
};

beforeEach(() => {
  vi.mocked(scanScenario).mockReset();
});

describe("SurfaceDetailPage", () => {
  it("renders surface metadata + provenance for an existing surface", async () => {
    vi.mocked(scanScenario).mockResolvedValueOnce(baseScan);
    renderWithProviders(
      <Routes>
        <Route path="/inventory/:surfaceId" element={<SurfaceDetailPage />} />
      </Routes>,
      { routerEntries: ["/inventory/ui-health__DashboardPage"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.inventory.detail.meta)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.inventory.detail.provenance)).toBeInTheDocument();
    const meta = screen.getByTestId(selectors.inventory.detail.meta);
    expect(meta.textContent).toContain("ui/src/pages/DashboardPage.tsx");
  });

  it("renders not-found when the slot isn't in the scan", async () => {
    vi.mocked(scanScenario).mockResolvedValueOnce({ ...baseScan, surfaces: [] });
    renderWithProviders(
      <Routes>
        <Route path="/inventory/:surfaceId" element={<SurfaceDetailPage />} />
      </Routes>,
      { routerEntries: ["/inventory/ui-health__GhostSlot"] },
    );
    expect(
      await screen.findByTestId(selectors.inventory.detail.notFound),
    ).toBeInTheDocument();
  });

  it("renders not-found for a malformed surfaceId", () => {
    renderWithProviders(
      <Routes>
        <Route path="/inventory/:surfaceId" element={<SurfaceDetailPage />} />
      </Routes>,
      { routerEntries: ["/inventory/no-separator"] },
    );
    expect(screen.getByTestId(selectors.inventory.detail.notFound)).toBeInTheDocument();
    expect(scanScenario).not.toHaveBeenCalled();
  });
});
