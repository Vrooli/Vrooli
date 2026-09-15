import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

vi.mock("../api/plans", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/plans")>()),
  listPlans: vi.fn(),
  createPlan: vi.fn(),
  updatePlan: vi.fn(),
  deletePlan: vi.fn(),
}));
vi.mock("../api/targets", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/targets")>()),
  listTargets: vi.fn(),
}));
vi.mock("../api/destinations", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/destinations")>()),
  listDestinations: vi.fn(),
}));
vi.mock("../api/runs", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../api/runs")>()),
  listRuns: vi.fn(),
  triggerRun: vi.fn(),
}));

import * as plansApi from "../api/plans";
import * as targetsApi from "../api/targets";
import * as destinationsApi from "../api/destinations";
import * as runsApi from "../api/runs";
import { SourceKind } from "../api/targets";
import { PlansPage } from "./PlansPage";

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(plansApi.listPlans).mockResolvedValue([
    { id: "p1", name: "nightly", targetIds: ["t1"], destinationIds: ["d1"], schedule: "0 2 * * *", retention: { keepLatest: 7 }, enabled: true },
  ] as never);
  vi.mocked(targetsApi.listTargets).mockResolvedValue([
    { id: "t1", owner: "prompt-manager", name: "store", sourceKind: SourceKind.FILESYSTEM, locator: "store" },
  ] as never);
  vi.mocked(destinationsApi.listDestinations).mockResolvedValue([{ id: "d1", name: "local" }] as never);
  vi.mocked(runsApi.listRuns).mockResolvedValue([] as never);
  vi.mocked(plansApi.createPlan).mockResolvedValue({ id: "p2" } as never);
  vi.mocked(runsApi.triggerRun).mockResolvedValue({ id: "r1" } as never);
});

afterEach(() => cleanup());

describe("PlansPage", () => {
  it("triggers an on-demand run from a plan row", async () => {
    const user = userEvent.setup();
    renderWithProviders(<PlansPage />);
    const row = await screen.findByTestId(selectors.plans.row({ id: "p1" }));
    await user.click(within(row).getByTestId(selectors.plans.runNowButton));
    expect(runsApi.triggerRun).toHaveBeenCalledWith("p1");
  });

  it("requires a target and a destination before creating a plan", async () => {
    const user = userEvent.setup();
    renderWithProviders(<PlansPage />);
    await user.click(screen.getByTestId(selectors.plans.createButton));
    await user.type(screen.getByTestId(selectors.plans.formName), "weekly");
    await user.click(screen.getByTestId(selectors.plans.formSubmit));
    expect(screen.getByText(strings.plans.needTarget)).toBeInTheDocument();
    expect(plansApi.createPlan).not.toHaveBeenCalled();
  });

  it("creates a plan once a target and destination are selected", async () => {
    const user = userEvent.setup();
    renderWithProviders(<PlansPage />);
    await user.click(screen.getByTestId(selectors.plans.createButton));
    await user.type(screen.getByTestId(selectors.plans.formName), "weekly");
    const targetPicker = await screen.findByTestId(selectors.plans.targetPicker);
    await user.click(within(targetPicker).getByRole("checkbox"));
    const destPicker = screen.getByTestId(selectors.plans.destinationPicker);
    await user.click(within(destPicker).getByRole("checkbox"));
    await user.click(screen.getByTestId(selectors.plans.formSubmit));
    expect(plansApi.createPlan).toHaveBeenCalledWith(
      expect.objectContaining({ name: "weekly", targetIds: ["t1"], destinationIds: ["d1"] }),
    );
  });
});
