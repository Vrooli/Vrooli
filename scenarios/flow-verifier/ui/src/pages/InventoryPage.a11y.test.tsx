import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return {
    ...actual,
    fetchFlows: vi.fn(),
    fetchRuns: vi.fn(),
    verifyFlow: vi.fn(),
  };
});

import { InventoryPage } from "./InventoryPage";

describe("InventoryPage accessibility", () => {
  beforeEach(async () => {
    const { fetchFlows, fetchRuns } = await import("../api/inventory");
    vi.mocked(fetchFlows).mockResolvedValue([
      { flowId: "alpha.flow", contractPath: "a/flow.json", language: "ts", schemaVersion: 1 },
    ]);
    vi.mocked(fetchRuns).mockResolvedValue([]);
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<InventoryPage />);
    await waitFor(() =>
      expect(screen.getByTestId("inventory-table")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
