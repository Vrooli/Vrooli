import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  DomainCoverageSchema,
  MeasureSummarySchema,
  SummarySchema,
  ValidateScenarioResponseSchema,
} from "@vrooli/proto-types/measures-health/v1/validation/validation_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/fleet", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/fleet")>();
  return { ...actual, fleetClient: { listFleetCoverage: vi.fn(), validateScenario: vi.fn() } };
});

import { ScenarioCoverageCard } from "./ScenarioCoverageCard";
import { fleetClient, DomainStatus, Tier } from "../../api/fleet";

const mockValidate = vi.mocked(fleetClient.validateScenario);

const response = () =>
  create(ValidateScenarioResponseSchema, {
    scenario: "swarm-manager",
    passed: false,
    summary: create(SummarySchema, { errors: 1, warnings: 0, infos: 1 }),
    domains: [
      create(DomainCoverageSchema, {
        domain: "backlog",
        status: DomainStatus.COVERED,
        measureCount: 1,
        tier: Tier.FULL,
        measures: [
          create(MeasureSummarySchema, {
            name: "backlog.completed",
            intent: "How many backlog items completed in a window.",
            tier: Tier.FULL,
            effect: "read",
            questionCount: 3,
          }),
        ],
      }),
      create(DomainCoverageSchema, {
        domain: "captures",
        status: DomainStatus.UNCOVERED,
        measureCount: 0,
        tier: Tier.UNSPECIFIED,
      }),
      create(DomainCoverageSchema, {
        domain: "queue",
        status: DomainStatus.WAIVED,
        measureCount: 0,
        waiverReason: "ephemeral; no historical value",
      }),
    ],
  });

describe("ScenarioCoverageCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows the select hint and runs no query when no scenario is selected", () => {
    renderWithProviders(<ScenarioCoverageCard />);
    expect(screen.getByTestId(selectors.fleet.detail.hint)).toBeInTheDocument();
    expect(mockValidate).not.toHaveBeenCalled();
  });

  it("renders domain rows ordered by urgency with the verdict badge", async () => {
    mockValidate.mockResolvedValue(response());
    renderWithProviders(<ScenarioCoverageCard scenario="swarm-manager" />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.detail.domains)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.fleet.detail.status)).toHaveAttribute("data-passed", "false");

    // UNCOVERED must sort to the top.
    const list = screen.getByTestId(selectors.fleet.detail.domains);
    const rows = within(list).getAllByText(/backlog|captures|queue/);
    expect(rows[0]).toHaveTextContent("captures");
  });

  it("nests covered measures under their domain", async () => {
    mockValidate.mockResolvedValue(response());
    renderWithProviders(<ScenarioCoverageCard scenario="swarm-manager" />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.domainRow({ domain: "backlog" }))).toBeInTheDocument());
    const backlog = screen.getByTestId(selectors.fleet.domainRow({ domain: "backlog" }));
    // The measure name is response data, not UI copy — assert via textContent.
    expect(backlog).toHaveTextContent("backlog.completed");
  });

  it("surfaces the waiver reason on a waived domain", async () => {
    mockValidate.mockResolvedValue(response());
    renderWithProviders(<ScenarioCoverageCard scenario="swarm-manager" />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.domainRow({ domain: "queue" }))).toBeInTheDocument());
    const queue = screen.getByTestId(selectors.fleet.domainRow({ domain: "queue" }));
    // The waiver reason is response data, not UI copy — assert via textContent.
    expect(queue).toHaveTextContent("ephemeral; no historical value");
  });

  it("shows the error state when validation fails", async () => {
    mockValidate.mockRejectedValue(new Error("boom"));
    renderWithProviders(<ScenarioCoverageCard scenario="swarm-manager" />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.detail.error)).toBeInTheDocument());
  });
});
