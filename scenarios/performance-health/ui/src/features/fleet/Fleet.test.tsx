import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { FleetView } from "./FleetView";

vi.mock("../../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/perf")>();
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
          {
            scenario: "beta",
            tier: "0",
            hasBudget: true,
            goBuildMs: 200n,
            uiBuildMs: 1000n,
            regressed: false,
            degradedReason: "",
          },
        ],
        tierDistribution: [
          { tier: "1", scenarioCount: 1 },
          { tier: "0", scenarioCount: 1 },
        ],
        errors: [],
        scenarioCount: 2,
        noBudgetCount: 1,
        regressedCount: 1,
      }),
    },
  };
});

const renderFleet = () =>
  renderWithProviders(
    <ScenarioProvider>
      <FleetView />
    </ScenarioProvider>,
  );

describe("FleetView", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the offender views and headline counters from ScanFleet", async () => {
    renderFleet();
    expect(screen.getByTestId(selectors.pages.fleet)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.summary)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.fleet.summaryScenarios)).toHaveTextContent("2");
    expect(screen.getByTestId(selectors.fleet.summaryNoBudget)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.fleet.summaryRegressed)).toHaveTextContent("1");
    // The regressed scenario shows up in the regressed offender section.
    expect(screen.getByTestId(selectors.fleet.tiers)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.fleet.slowest)).toBeInTheDocument();
  });
});
