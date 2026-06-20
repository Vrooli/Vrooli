import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  FleetScanErrorSchema,
  FleetScenarioEntrySchema,
  ProfileDistributionSchema,
  RuleConformanceSchema,
  ScanFleetResponseSchema,
} from "@vrooli/proto-types/structure-health/v1/fleet/fleet_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/fleet")>();
  return { ...actual, fleetClient: { scanFleet: vi.fn() } };
});

import { FleetView } from "./FleetView";
import { fleetClient } from "../../api/fleet";

const mockScan = vi.mocked(fleetClient.scanFleet);

const fullResponse = () =>
  create(ScanFleetResponseSchema, {
    scenarioCount: 3,
    passingCount: 1,
    missingFreshnessCount: 1,
    autofixableTotal: 4,
    entries: [
      create(FleetScenarioEntrySchema, {
        scenario: "swarm-manager",
        passed: false,
        profileId: "react-vite-go",
        profileRecognized: true,
        errorCount: 2,
        warningCount: 1,
        autofixableCount: 3,
        missingFreshnessCheck: true,
        surfaces: ["api", "cli", "ui"],
        degradedReason: "",
      }),
      create(FleetScenarioEntrySchema, {
        scenario: "measures-health",
        passed: true,
        profileId: "unknown",
        profileRecognized: false,
        errorCount: 0,
        warningCount: 0,
        autofixableCount: 1,
        missingFreshnessCheck: false,
        surfaces: ["api"],
        degradedReason: "partial scan",
      }),
    ],
    ruleConformance: [
      create(RuleConformanceSchema, {
        code: "FRESHNESS_CHECK_MISSING",
        offendingScenarios: 2,
        totalFindings: 2,
        autofixable: 1,
        worstSeverity: "error",
      }),
      create(RuleConformanceSchema, {
        code: "PROFILE_ENV_VALIDATION",
        offendingScenarios: 1,
        totalFindings: 1,
        autofixable: 0,
        worstSeverity: "warning",
      }),
    ],
    profileDistribution: [
      create(ProfileDistributionSchema, {
        profileId: "react-vite-go",
        scenarioCount: 2,
        recognized: true,
      }),
      create(ProfileDistributionSchema, {
        profileId: "unknown",
        scenarioCount: 1,
        recognized: false,
      }),
    ],
    errors: [
      create(FleetScanErrorSchema, { scenario: "broken-scenario", reason: "no service.json" }),
    ],
  });

describe("FleetView", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the loading state while the scan is in flight", () => {
    mockScan.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<FleetView />);
    expect(screen.getByTestId(selectors.fleet.loading)).toBeInTheDocument();
  });

  it("renders the error state when the scan fails", async () => {
    mockScan.mockRejectedValue(new Error("boom"));
    renderWithProviders(<FleetView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument(),
    );
  });

  it("renders the empty state when no scenarios were graded", async () => {
    mockScan.mockResolvedValue(create(ScanFleetResponseSchema, {}));
    renderWithProviders(<FleetView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument(),
    );
  });

  it("renders summary stats, profiles, rules, offenders and scan errors", async () => {
    mockScan.mockResolvedValue(fullResponse());
    renderWithProviders(<FleetView />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.summary)).toBeInTheDocument(),
    );

    // Summary counters
    expect(screen.getByTestId(selectors.fleet.summaryScenarios)).toHaveTextContent("3");
    expect(screen.getByTestId(selectors.fleet.summaryPassing)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.fleet.summaryMissingFreshness)).toHaveTextContent("1");
    expect(screen.getByTestId(selectors.fleet.summaryAutofixable)).toHaveTextContent("4");

    // Profile distribution
    expect(
      screen.getByTestId(selectors.fleet.profileRow({ profileId: "react-vite-go" })),
    ).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.fleet.profileRow({ profileId: "unknown" })),
    ).toBeInTheDocument();

    // Rule conformance — server order preserved (most-offending first)
    const ruleSection = screen.getByTestId(selectors.fleet.rules);
    const ruleRows = ruleSection.querySelectorAll("tbody tr");
    expect(ruleRows).toHaveLength(2);
    expect(ruleRows[0]).toHaveAttribute(
      "data-testid",
      selectors.fleet.ruleRow({ code: "FRESHNESS_CHECK_MISSING" }),
    );

    // Scenario offenders
    const failRow = screen.getByTestId(selectors.fleet.scenarioRow({ scenario: "swarm-manager" }));
    expect(failRow).toHaveAttribute("data-passed", "false");
    expect(failRow).toHaveTextContent("missing freshness");
    const passRow = screen.getByTestId(
      selectors.fleet.scenarioRow({ scenario: "measures-health" }),
    );
    expect(passRow).toHaveAttribute("data-passed", "true");
    expect(passRow).toHaveTextContent("partial scan");

    // Scan errors
    expect(screen.getByTestId(selectors.fleet.errors)).toHaveTextContent("broken-scenario");
  });

  it("shows the rule-empty placeholder when there are entries but no rule findings", async () => {
    mockScan.mockResolvedValue(
      create(ScanFleetResponseSchema, {
        scenarioCount: 1,
        passingCount: 1,
        entries: [
          create(FleetScenarioEntrySchema, {
            scenario: "clean-scenario",
            passed: true,
            profileId: "react-vite-go",
            profileRecognized: true,
          }),
        ],
      }),
    );
    renderWithProviders(<FleetView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.rulesEmpty)).toBeInTheDocument(),
    );
  });

  it("re-runs the scan when the refresh button is clicked", async () => {
    mockScan.mockResolvedValue(fullResponse());
    renderWithProviders(<FleetView />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.summary)).toBeInTheDocument(),
    );
    expect(mockScan).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByTestId(selectors.fleet.refreshButton));
    await waitFor(() => expect(mockScan).toHaveBeenCalledTimes(2));
  });
});
