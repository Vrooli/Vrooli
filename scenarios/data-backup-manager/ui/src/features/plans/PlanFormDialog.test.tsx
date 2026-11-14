/**
 * PlanFormDialog coverage-guard test. When the API rejects a create with
 * FAILED_PRECONDITION (non-sensitive recommended targets still unregistered),
 * the dialog surfaces a distinct warning and an explicit "proceed" control that
 * resubmits with allowIncompleteCoverage — incomplete coverage is never silent.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { SourceKind } from "../../api/targets";

vi.mock("../../api/plans", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/plans")>()),
  createPlan: vi.fn(),
}));
vi.mock("../../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/targets")>()),
  listTargets: vi.fn(),
}));
vi.mock("../../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../api/destinations")>()),
  listDestinations: vi.fn(),
}));

import * as plansApi from "../../api/plans";
import * as targetsApi from "../../api/targets";
import * as destinationsApi from "../../api/destinations";
import { PlanFormDialog } from "./PlanFormDialog";

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(targetsApi.listTargets).mockResolvedValue([
    { id: "t1", owner: "vrooli", name: "store", sourceKind: SourceKind.FILESYSTEM, locator: "store" },
  ] as never);
  vi.mocked(destinationsApi.listDestinations).mockResolvedValue([{ id: "d1", name: "local" }] as never);
});

afterEach(() => cleanup());

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByTestId(selectors.plans.formName), "nightly");
  const targetPicker = screen.getByTestId(selectors.plans.targetPicker);
  await user.click(targetPicker.querySelector("input[type=checkbox]") as HTMLElement);
  const destPicker = screen.getByTestId(selectors.plans.destinationPicker);
  await user.click(destPicker.querySelector("input[type=checkbox]") as HTMLElement);
  await user.click(screen.getByTestId(selectors.plans.formSubmit));
}

describe("PlanFormDialog incomplete coverage", () => {
  it("shows the proceed control on FAILED_PRECONDITION and resubmits with allowIncompleteCoverage", async () => {
    const user = userEvent.setup();
    vi.mocked(plansApi.createPlan)
      .mockRejectedValueOnce(new ConnectError("incomplete coverage", Code.FailedPrecondition))
      .mockResolvedValueOnce({ id: "p1", name: "nightly" } as never);

    renderWithProviders(<PlanFormDialog open onClose={() => {}} />);

    await fillAndSubmit(user);

    // First attempt rejected → warning + proceed control appears.
    const proceed = await screen.findByTestId(selectors.plans.proceedIncompleteCoverage);
    expect(screen.getByTestId(selectors.plans.coverageWarning)).toBeInTheDocument();

    await user.click(proceed);

    await waitFor(() => expect(plansApi.createPlan).toHaveBeenCalledTimes(2));
    const secondCall = vi.mocked(plansApi.createPlan).mock.calls[1]?.[0];
    expect(secondCall?.allowIncompleteCoverage).toBe(true);
  });
});
