import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return {
    ...actual,
    fetchFlows: vi.fn(),
    fetchRuns: vi.fn(),
    verifyFlow: vi.fn(),
  };
});

vi.mock("../api/scenarios", () => ({
  fetchScenarios: vi.fn().mockResolvedValue({ vrooliRoot: "/repo", scenarios: [] }),
  fetchScenarioDetail: vi.fn(),
}));

import { InventoryPage } from "./InventoryPage";

const flowAlpha = {
  flowId: "alpha.flow",
  contractPath: "a/flow.json",
  language: "ts" as const,
  schemaVersion: 1,
  kind: "temporal",
};

const flowBeta = {
  flowId: "beta.flow",
  contractPath: "b/flow.json",
  language: "go" as const,
  schemaVersion: 1,
  kind: "temporal",
};

describe("InventoryPage", () => {
  beforeEach(async () => {
    const { fetchFlows, fetchRuns, verifyFlow } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockReset();
    vi.mocked(fetchRuns).mockReset();
    vi.mocked(verifyFlow).mockReset();
  });
  afterEach(() => cleanup());

  it("renders the empty state when no flows are returned", async () => {
    const { fetchFlows, fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([]);
    vi.mocked(fetchRuns).mockResolvedValue([]);
    renderWithProviders(<InventoryPage />);
    await waitFor(() =>
      expect(screen.getByTestId("inventory-empty")).toBeInTheDocument(),
    );
  });

  it("renders the inventory table when flows are returned", async () => {
    const { fetchFlows, fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([flowAlpha, flowBeta]);
    vi.mocked(fetchRuns).mockResolvedValue([]);
    renderWithProviders(<InventoryPage />);
    await waitFor(() =>
      expect(screen.getByTestId("inventory-table")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("inventory-row-alpha.flow")).toBeInTheDocument();
    expect(screen.getByTestId("inventory-row-beta.flow")).toBeInTheDocument();
  });

  it("filters by search input", async () => {
    const { fetchFlows, fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([flowAlpha, flowBeta]);
    vi.mocked(fetchRuns).mockResolvedValue([]);
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await waitFor(() =>
      expect(screen.getByTestId("inventory-table")).toBeInTheDocument(),
    );
    await user.type(screen.getByTestId("inventory-search"), "beta");
    await waitFor(() =>
      expect(screen.queryByTestId("inventory-row-alpha.flow")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("inventory-row-beta.flow")).toBeInTheDocument();
  });

  it("filters by language", async () => {
    const { fetchFlows, fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([flowAlpha, flowBeta]);
    vi.mocked(fetchRuns).mockResolvedValue([]);
    const user = userEvent.setup();
    renderWithProviders(<InventoryPage />);
    await waitFor(() =>
      expect(screen.getByTestId("inventory-table")).toBeInTheDocument(),
    );
    await user.selectOptions(screen.getByTestId("inventory-language"), "go");
    await waitFor(() =>
      expect(screen.queryByTestId("inventory-row-alpha.flow")).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("inventory-row-beta.flow")).toBeInTheDocument();
  });
});
