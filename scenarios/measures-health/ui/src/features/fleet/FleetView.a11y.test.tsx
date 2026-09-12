/**
 * FleetView accessibility regression tests. The table + drill-down own their
 * own query states, so the axe waits and mocks live with the feature rather
 * than leaking into the app-level a11y suite.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  DomainCoverageSchema,
  FleetEntrySchema,
  ListFleetCoverageResponseSchema,
  MeasureSummarySchema,
  ScenarioCoverageReportSchema,
  SummarySchema,
} from "@vrooli/proto-types/measures-health/v1/validation/validation_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

vi.mock("../../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/fleet")>();
  return { ...actual, fleetClient: { listFleetCoverage: vi.fn(), validateScenario: vi.fn() } };
});

import { FleetView } from "./FleetView";
import { fleetClient, DomainStatus, Tier } from "../../api/fleet";

describe("FleetView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the fleet table and drill-down without axe violations", async () => {
    vi.mocked(fleetClient.listFleetCoverage).mockResolvedValue(
      create(ListFleetCoverageResponseSchema, {
        entries: [
          create(FleetEntrySchema, {
            scenario: "swarm-manager",
            passed: false,
            expected: 2,
            covered: 1,
            waived: 0,
            uncovered: 1,
            worstTier: Tier.PARTIAL,
            measureCount: 1,
          }),
        ],
      }),
    );
    vi.mocked(fleetClient.validateScenario).mockResolvedValue(
      create(ScenarioCoverageReportSchema, {
        scenario: "swarm-manager",
        passed: false,
        summary: create(SummarySchema, { errors: 1, warnings: 0, infos: 0 }),
        domains: [
          create(DomainCoverageSchema, {
            domain: "backlog",
            status: DomainStatus.COVERED,
            measureCount: 1,
            tier: Tier.FULL,
            measures: [
              create(MeasureSummarySchema, {
                name: "backlog.completed",
                intent: "How many backlog items completed.",
                tier: Tier.FULL,
                effect: "read",
                questionCount: 2,
              }),
            ],
          }),
          create(DomainCoverageSchema, {
            domain: "captures",
            status: DomainStatus.UNCOVERED,
            measureCount: 0,
            tier: Tier.UNSPECIFIED,
          }),
        ],
      }),
    );

    const { container } = renderWithProviders(<FleetView />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.table)).toBeInTheDocument());
    await expectNoA11yViolations(container);

    // Select a scenario and re-scan the loaded drill-down.
    await userEvent.click(screen.getByTestId(selectors.fleet.row({ scenario: "swarm-manager" })));
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.detail.domains)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
