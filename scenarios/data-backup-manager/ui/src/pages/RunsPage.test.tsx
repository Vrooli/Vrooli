import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listRuns: vi.fn(),
}));
vi.mock("../api/plans", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/plans")>()),
  listPlans: vi.fn(),
}));

import * as runsApi from "../api/runs";
import * as plansApi from "../api/plans";
import { RunStatus, TargetOutcomeStatus, TriggerSource } from "../api/runs";
import { RunsPage } from "./RunsPage";

const start = timestampFromDate(new Date(Date.now() - 5 * 60 * 1000));
const finish = timestampFromDate(new Date(Date.now() - 4 * 60 * 1000));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(plansApi.listPlans).mockResolvedValue([{ id: "p1", name: "nightly" }] as never);
  vi.mocked(runsApi.listRuns).mockResolvedValue([
    {
      id: "r1",
      planId: "p1",
      trigger: TriggerSource.MANUAL,
      status: RunStatus.PARTIAL_FAILED,
      failureCode: "credential_missing",
      failureCategory: "credential",
      nextAction: "restore repository credential",
      preflightIncidents: [{ code: "credential_missing", scope: "destination", message: "credential unavailable", nextAction: "restore credential" }],
      startedAt: start,
      finishedAt: finish,
      outcomes: [
        { targetId: "t1", destinationId: "d1", status: TargetOutcomeStatus.SUCCEEDED, snapshotId: "snap-1", bytes: 1024n, error: "" },
        { targetId: "t2", destinationId: "d1", status: TargetOutcomeStatus.BLOCKED, snapshotId: "", bytes: 0n, error: "cap reached" },
      ],
    },
  ] as never);
});

afterEach(() => cleanup());

describe("RunsPage", () => {
  it("shows a partial failure as partial (amber), not a flat failure", async () => {
    renderWithProviders(<RunsPage />);
    const row = await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    expect(within(row).getByText(strings.status.run.partialFailed)).toBeInTheDocument();
    expect(within(row).queryByText(strings.status.run.failed)).not.toBeInTheDocument();
  });

  it("expands to per-target outcomes, keeping cap-blocked distinct from failed", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RunsPage />);
    const row = await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    await user.click(within(row).getByRole("button"));

    const blocked = within(row).getByTestId(selectors.runs.outcomeRow({ targetId: "t2" }));
    expect(within(blocked).getByText(strings.status.outcome.blocked)).toBeInTheDocument();
    const ok = within(row).getByTestId(selectors.runs.outcomeRow({ targetId: "t1" }));
    expect(within(ok).getByText(strings.status.outcome.succeeded)).toBeInTheDocument();
  });

  it("shows stable run failure evidence and the safe next action", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RunsPage />);
    const row = await screen.findByTestId(selectors.runs.row({ id: "r1" }));
    await user.click(within(row).getByRole("button"));
    expect(within(row).getByTestId("run-failure-code")).toHaveTextContent("credential_missing (credential)");
    expect(within(row).getByTestId("run-next-action")).toHaveTextContent(
      `${strings.runs.nextAction}: restore repository credential`,
    );
    expect(within(row).getByText(/credential_missing: credential unavailable/)).toBeInTheDocument();
  });
});
