/**
 * FleetView accessibility regression. The view owns its own query state, so the
 * axe wait + mock live with the feature rather than the app-level a11y suite.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  FleetScenarioEntrySchema,
  ProfileDistributionSchema,
  RuleConformanceSchema,
  ScanFleetResponseSchema,
} from "@vrooli/proto-types/structure-health/v1/fleet/fleet_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/fleet")>();
  return { ...actual, fleetClient: { scanFleet: vi.fn() } };
});

import { FleetView } from "./FleetView";
import { fleetClient } from "../../api/fleet";

describe("FleetView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the populated dashboard without axe violations", async () => {
    vi.mocked(fleetClient.scanFleet).mockResolvedValue(
      create(ScanFleetResponseSchema, {
        scenarioCount: 1,
        passingCount: 0,
        missingFreshnessCount: 1,
        autofixableTotal: 1,
        entries: [
          create(FleetScenarioEntrySchema, {
            scenario: "swarm-manager",
            passed: false,
            profileId: "react-vite-go",
            profileRecognized: true,
            errorCount: 1,
            warningCount: 0,
            autofixableCount: 1,
            missingFreshnessCheck: true,
          }),
        ],
        ruleConformance: [
          create(RuleConformanceSchema, {
            code: "FRESHNESS_CHECK_MISSING",
            offendingScenarios: 1,
            totalFindings: 1,
            autofixable: 1,
            worstSeverity: "error",
          }),
        ],
        profileDistribution: [
          create(ProfileDistributionSchema, {
            profileId: "react-vite-go",
            scenarioCount: 1,
            recognized: true,
          }),
        ],
      }),
    );

    const { container } = renderWithProviders(<FleetView />);
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.summary)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
