import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders, makeHealthResponse } from "../test-utils";

vi.mock("../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/inventory")>();
  return { ...actual, fetchRuns: vi.fn() };
});
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn() };
});

import { DashboardPage } from "./DashboardPage";

describe("DashboardPage accessibility", () => {
  beforeEach(async () => {
    const { fetchRuns } = await import("../api/inventory");
    const { fetchHealth } = await import("../api/health");
    vi.mocked(fetchRuns).mockResolvedValue([]);
    vi.mocked(fetchHealth).mockResolvedValue(makeHealthResponse());
  });
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<DashboardPage />);
    await waitFor(() =>
      expect(screen.getByTestId("dashboard-recent-empty")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
