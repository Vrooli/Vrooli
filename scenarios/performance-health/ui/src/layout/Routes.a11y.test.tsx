/**
 * Per-route accessibility regression. Renders every route through the real
 * router (so axe sees the actual landmark composition: header + nav + main +
 * bottom nav + the page content) and asserts zero axe violations in English.
 *
 * Feature screens fetch data, so the perf client is stubbed with quiet,
 * populated-enough responses; we wait for each page's root testid before
 * scanning so axe runs against settled content, not a spinner.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

vi.mock("../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/perf")>();
  return {
    ...actual,
    perfClient: {
      scanFleet: vi.fn().mockResolvedValue({
        entries: [
          {
            scenario: "alpha",
            tier: "1",
            hasBudget: false,
            goBuildMs: 600n,
            uiBuildMs: 4000n,
            regressed: true,
            degradedReason: "",
          },
        ],
        tierDistribution: [{ tier: "1", scenarioCount: 1 }],
        errors: [],
        scenarioCount: 1,
        noBudgetCount: 1,
        regressedCount: 1,
      }),
      validateReadiness: vi.fn().mockResolvedValue({
        scenario: "alpha",
        tier: actual.CaptureTier.CAPTURE_TIER_1,
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

const ROUTES: [string, string][] = [
  ["/", selectors.pages.dashboard],
  ["/audit", selectors.pages.audit],
  ["/trends", selectors.pages.trends],
  ["/fleet", selectors.pages.fleet],
  ["/trace", selectors.pages.trace],
  ["/readiness", selectors.pages.readiness],
  ["/budgets", selectors.pages.budgets],
  ["/settings", selectors.pages.settings],
];

describe("per-route accessibility (English)", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setLocale("en");
  });
  afterEach(() => cleanup());

  it.each(ROUTES)("has no axe violations at %s", async (path, rootSelector) => {
    const { container } = renderWithProviders(<TestAppRouter initialEntries={[path]} />, {
      withoutRouter: true,
    });
    // Wait for the page root, then for every query to settle (no spinner left)
    // so axe scans against fully-resolved content and no state update lands
    // outside act() after the assertion.
    await screen.findByTestId(rootSelector);
    await waitFor(() =>
      expect(screen.queryByTestId(selectors.state.loading)).not.toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
