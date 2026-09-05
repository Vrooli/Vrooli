import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";

const { listCatalogAssets, listAdoptions } = vi.hoisted(() => ({
  listCatalogAssets: vi.fn(),
  listAdoptions: vi.fn(),
}));

vi.mock("../api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/components")>();
  return { ...actual, listCatalogAssets };
});
vi.mock("../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/adoptions")>();
  return { ...actual, adoptionsClient: { listAdoptions } };
});

import { DashboardPage } from "./DashboardPage";

describe("DashboardPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("summarizes assets, attention, scenario adoption health, and next moves", async () => {
    listCatalogAssets.mockResolvedValue({
      components: [
        {
          id: "asset-1",
          displayName: "Banner",
          version: "1.2.0",
          metrics: { directAdoptionCount: 2 },
        },
      ],
    });
    listAdoptions.mockResolvedValue({
      adoptions: [{ id: "adoption-1", scenario: "demo", libraryVersionStatus: 1, localStatus: 1 }],
    });
    renderWithProviders(<DashboardPage />);

    expect(await screen.findByText("Banner")).toBeInTheDocument();
    expect(screen.getByText("dashboard.recentlyEvolved")).toBeInTheDocument();
    expect(screen.getByText("dashboard.adoptionHealth")).toBeInTheDocument();
    expect(screen.getByText("dashboard.nextMoves")).toBeInTheDocument();
    expect(screen.getByText("demo")).toBeInTheDocument();
  });
});
