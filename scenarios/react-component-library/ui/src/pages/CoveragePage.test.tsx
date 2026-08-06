import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { getCatalogCoverage, listCatalogNextWork } = vi.hoisted(() => ({ getCatalogCoverage: vi.fn(), listCatalogNextWork: vi.fn() }));
vi.mock("../api/catalog", () => ({ getCatalogCoverage, listCatalogNextWork }));

import { CoveragePage } from "./CoveragePage";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("CoveragePage", () => {
  afterEach(() => cleanup());
  beforeEach(() => {
    getCatalogCoverage.mockReset();
    listCatalogNextWork.mockReset();
  });

  it("exposes maturity, ranked next work, and searchable coverage rows", async () => {
    getCatalogCoverage.mockResolvedValue({
      maturity: { total: 3, atOrAboveTarget: 2, byRung: { verified: 1, production_ready: 2 } },
      rows: [{ assetId: "controls.button", name: "Button", domain: "controls", target: "production-ready", achieved: "verified", blocksDownstream: 2 }],
    });
    listCatalogNextWork.mockResolvedValue({ rows: [{ assetId: "controls.button", name: "Button", target: "production-ready", achieved: "verified", blocksDownstream: 2 }] });

    renderWithProviders(<CoveragePage />);

    expect(await screen.findByRole("heading", { name: "Catalog coverage" })).toBeInTheDocument();
    expect(screen.getByText("At or above target")).toBeInTheDocument();
    expect(screen.getByText("Ranked next work")).toBeInTheDocument();
    expect(screen.getByTestId("coverage-table")).toBeInTheDocument();
    await waitFor(() => expect(screen.getAllByText("Button").length).toBeGreaterThan(0));
  });

  it("shows a recoverable empty state when the coverage service is unavailable", async () => {
    getCatalogCoverage.mockRejectedValue(new Error("offline"));
    listCatalogNextWork.mockResolvedValue({ rows: [] });
    renderWithProviders(<CoveragePage />);
    expect(await screen.findByText("Coverage unavailable")).toBeInTheDocument();
  });

  it("exercises coverage table search and sorting", async () => {
    const user = userEvent.setup();
    getCatalogCoverage.mockResolvedValue({
      maturity: { total: 2, atOrAboveTarget: 1, byRung: { missing: 1, custom: 1 } },
      rows: [
        { assetId: "zeta", name: "Zeta", domain: "z", target: "verified", achieved: "missing", blocksDownstream: 0 },
        { assetId: "alpha", name: "", domain: "a", target: "production-ready", achieved: "production-ready", blocksDownstream: 3 },
      ],
    });
    listCatalogNextWork.mockResolvedValue({ rows: [] });
    renderWithProviders(<CoveragePage />);

    expect(await screen.findByTestId("coverage-table")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Sort by Domain" }));
    await user.click(screen.getByRole("button", { name: "Sort by Domain" }));
    await user.type(screen.getByPlaceholderText("Search assets, domains, or maturity"), "zeta");
    expect(screen.getByText("Zeta")).toBeInTheDocument();
    expect(screen.queryByText("production-ready")).not.toBeInTheDocument();
  });

  it("shows independent loading and empty-table states", async () => {
    getCatalogCoverage.mockResolvedValue({ maturity: { total: 0, atOrAboveTarget: 0, byRung: {} }, rows: [] });
    listCatalogNextWork.mockReturnValue(new Promise(() => undefined));
    renderWithProviders(<CoveragePage />);
    expect(await screen.findByText("Calculating next work…")).toBeInTheDocument();

    cleanup();
    listCatalogNextWork.mockResolvedValue({ rows: [] });
    renderWithProviders(<CoveragePage />);
    expect(await screen.findByText("No catalog rows")).toBeInTheDocument();
  });
});
