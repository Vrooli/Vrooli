import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { AuditWorkbench } from "./AuditWorkbench";

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
        tier: actual.CaptureTier.CAPTURE_TIER_1,
        uiFramework: "react-vite",
        surfaces: ["ui"],
        degradedReason: "",
        autofixableCount: 0,
        assessment: { findings: [] },
      }),
      runAudit: vi.fn().mockResolvedValue({
        scenario: "performance-health",
        outcome: actual.AuditOutcome.CAPTURED,
        tier: actual.CaptureTier.CAPTURE_TIER_1,
        traceArtifact: "/tmp/performance.json",
        webVitalsArtifact: "/tmp/performance.web-vitals.json",
        reason: "",
      }),
    },
  };
});

const renderAudit = () =>
  renderWithProviders(
    <ScenarioProvider>
      <AuditWorkbench />
    </ScenarioProvider>,
  );

describe("AuditWorkbench", () => {
  beforeEach(() => vi.clearAllMocks());

  it("renders the workbench and the decided capture tier from ValidateReadiness", async () => {
    renderAudit();
    expect(screen.getByTestId(selectors.pages.audit)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.audit.tierBadge)).toBeInTheDocument(),
    );
  });

  it("runs an audit and surfaces the produced trace artifact link", async () => {
    const { perfClient } = await import("../../api/perf");
    renderAudit();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.audit.tierBadge)).toBeInTheDocument(),
    );
    screen.getByTestId(selectors.audit.runButton).click();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.audit.analyzeTraceLink)).toBeInTheDocument(),
    );
    expect(perfClient.runAudit).toHaveBeenCalled();
  });
});
