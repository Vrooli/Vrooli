import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  FleetEntrySchema,
  ListFleetCoverageResponseSchema,
} from "@vrooli/proto-types/measures-health/v1/validation/validation_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/fleet")>();
  return { ...actual, fleetClient: { listFleetCoverage: vi.fn(), validateScenario: vi.fn() } };
});

import { FleetTable } from "./FleetTable";
import { fleetClient, Tier } from "../../api/fleet";

const mockList = vi.mocked(fleetClient.listFleetCoverage);

const response = () =>
  create(ListFleetCoverageResponseSchema, {
    entries: [
      create(FleetEntrySchema, {
        scenario: "swarm-manager",
        passed: false,
        expected: 5,
        covered: 3,
        waived: 1,
        uncovered: 1,
        worstTier: Tier.PARTIAL,
        measureCount: 4,
      }),
      create(FleetEntrySchema, {
        scenario: "measures-health",
        passed: true,
        expected: 2,
        covered: 2,
        waived: 0,
        uncovered: 0,
        worstTier: Tier.FULL,
        measureCount: 2,
      }),
    ],
  });

describe("FleetTable", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders one row per scenario with its verdict badge", async () => {
    mockList.mockResolvedValue(response());
    renderWithProviders(<FleetTable onSelect={() => {}} />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.rowVerdict({ scenario: "swarm-manager" }))).toBeInTheDocument(),
    );

    const failing = screen.getByTestId(selectors.fleet.rowVerdict({ scenario: "swarm-manager" }));
    expect(failing).toHaveAttribute("data-passed", "false");
    const passing = screen.getByTestId(selectors.fleet.rowVerdict({ scenario: "measures-health" }));
    expect(passing).toHaveAttribute("data-passed", "true");
  });

  it("invokes onSelect when a row is clicked", async () => {
    mockList.mockResolvedValue(response());
    const onSelect = vi.fn();
    renderWithProviders(<FleetTable onSelect={onSelect} />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.row({ scenario: "swarm-manager" }))).toBeInTheDocument());
    await userEvent.click(screen.getByTestId(selectors.fleet.row({ scenario: "swarm-manager" })));
    expect(onSelect).toHaveBeenCalledWith("swarm-manager");
  });

  it("marks the selected row", async () => {
    mockList.mockResolvedValue(response());
    renderWithProviders(<FleetTable selected="measures-health" onSelect={() => {}} />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.fleet.row({ scenario: "measures-health" }))).toHaveAttribute(
        "data-selected",
        "true",
      ),
    );
  });

  it("shows the empty state when no scenarios are discovered", async () => {
    mockList.mockResolvedValue(create(ListFleetCoverageResponseSchema, { entries: [] }));
    renderWithProviders(<FleetTable onSelect={() => {}} />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
  });

  it("shows the error state when the rollup fails", async () => {
    mockList.mockRejectedValue(new Error("boom"));
    renderWithProviders(<FleetTable onSelect={() => {}} />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.error)).toBeInTheDocument());
  });
});
