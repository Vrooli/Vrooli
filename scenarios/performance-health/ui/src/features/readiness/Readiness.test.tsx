import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { ReadinessPanel } from "./ReadinessPanel";

vi.mock("../../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/perf")>();
  return {
    ...actual,
    perfClient: {
      scanFleet: vi.fn().mockResolvedValue({
        entries: [{ scenario: "performance-health", tier: "1" }],
        tierDistribution: [],
        errors: [],
        scenarioCount: 1,
        noBudgetCount: 0,
        regressedCount: 0,
      }),
      validateReadiness: vi.fn().mockResolvedValue({
        scenario: "performance-health",
        tier: actual.CaptureTier.CAPTURE_TIER_0,
        uiFramework: "react-vite",
        surfaces: ["ui"],
        degradedReason: "",
        autofixableCount: 2,
        assessment: {
          findings: [
            {
              code: "PERF_MISSING_PROFILE_BUILD",
              severity: "warning",
              title: "Missing profile build script",
              message: "build:profile script is absent",
              location: "ui/package.json",
              remediation: "Add build:profile",
              autofixAvailable: true,
            },
          ],
        },
      }),
      applyReadinessFix: vi.fn().mockResolvedValue({
        scenario: "performance-health",
        applied: true,
        candidates: [],
        messages: ["wrote build:profile script"],
      }),
      previewReadinessFix: vi.fn().mockResolvedValue({
        scenario: "performance-health",
        applied: false,
        candidates: [],
        messages: [],
      }),
    },
  };
});

const renderReadiness = () =>
  renderWithProviders(
    <ScenarioProvider>
      <ReadinessPanel />
    </ScenarioProvider>,
  );

describe("ReadinessPanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders readiness gaps and the autofixable count", async () => {
    renderReadiness();
    expect(screen.getByTestId(selectors.pages.readiness)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.readiness.autofixableCount)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.readiness.gapRow({ code: "PERF_MISSING_PROFILE_BUILD" })),
    ).toBeInTheDocument();
  });

  it("applies an autofix via ApplyReadinessFix", async () => {
    const { perfClient } = await import("../../api/perf");
    renderReadiness();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.readiness.applyButton)).toBeEnabled(),
    );
    screen.getByTestId(selectors.readiness.applyButton).click();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.readiness.fixResult)).toBeInTheDocument(),
    );
    expect(perfClient.applyReadinessFix).toHaveBeenCalled();
  });
});
