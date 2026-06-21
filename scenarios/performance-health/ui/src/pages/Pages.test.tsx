import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "../app/routes";

/**
 * Per-route smoke tests for the thin page wrappers. Each page wrapper just
 * mounts its feature; rendering the route through the real router proves the
 * wrapper + route config are wired. The perf client is stubbed so the
 * feature screens render without a live backend.
 */
vi.mock("../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/perf")>();
  const empty = {
    entries: [],
    tierDistribution: [],
    errors: [],
    scenarioCount: 0,
    noBudgetCount: 0,
    regressedCount: 0,
  };
  return {
    ...actual,
    perfClient: {
      scanFleet: vi.fn().mockResolvedValue(empty),
      validateReadiness: vi.fn().mockResolvedValue({
        scenario: "performance-health",
        tier: actual.CaptureTier.CAPTURE_TIER_0,
        uiFramework: "react-vite",
        surfaces: ["ui"],
        degradedReason: "",
        autofixableCount: 0,
        assessment: { findings: [] },
      }),
      getTrend: vi.fn().mockResolvedValue({ samples: [] }),
      getStartupTrend: vi.fn().mockResolvedValue({ measurements: [] }),
      getBudget: vi.fn().mockResolvedValue({ budget: undefined, declared: false }),
    },
  };
});

const renderRoute = (path: string) =>
  renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });

beforeEach(() => vi.clearAllMocks());
afterEach(() => cleanup());

describe("page route wrappers", () => {
  it.each([
    ["/audit", selectors.pages.audit],
    ["/trends", selectors.pages.trends],
    ["/fleet", selectors.pages.fleet],
    ["/trace", selectors.pages.trace],
    ["/readiness", selectors.pages.readiness],
    ["/budgets", selectors.pages.budgets],
    ["/settings", selectors.pages.settings],
  ])("renders %s", async (path, selector) => {
    renderRoute(path);
    await waitFor(() => expect(screen.getByTestId(selector)).toBeInTheDocument());
  });
});
