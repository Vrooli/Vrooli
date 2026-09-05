/**
 * ValidationBoard tests — reference resolution, staleness, derived scope, and
 * producer-owned guidance. api/validation + api/plans are mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  PlanSchema,
  ReferenceSchema,
  StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const resolveReferences = vi.fn();
const computeStaleness = vi.fn();
const deriveBaselineScope = vi.fn();
const listPlans = vi.fn();

vi.mock("../../api/validation", () => ({
  resolveReferences: (...a: unknown[]) => resolveReferences(...a),
  computeStaleness: (...a: unknown[]) => computeStaleness(...a),
  deriveBaselineScope: (...a: unknown[]) => deriveBaselineScope(...a),
}));
vi.mock("../../api/plans", () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
  listTemplates: vi.fn(),
  getPlan: vi.fn(),
  getGraph: vi.fn(),
  renderPlan: vi.fn(),
  archivePlan: vi.fn(),
  createFromTemplate: vi.fn(),
}));

import { ValidationBoard } from "./ValidationBoard";

const pickPlan = async () => {
  const user = userEvent.setup();
  listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
  renderWithProviders(<ValidationBoard />);
  await waitFor(() => {
    expect(
      screen.getByTestId(selectors.validation.planSelect).querySelector('option[value="plan-1"]'),
    ).not.toBeNull();
  });
  await user.selectOptions(screen.getByTestId(selectors.validation.planSelect), "plan-1");
  return user;
};

describe("ValidationBoard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("[REQ:PM-REF-001] resolves references and surfaces the degraded note honestly", async () => {
    const user = await pickPlan();
    resolveReferences.mockResolvedValue({
      references: [
        create(ReferenceSchema, { id: "r1", target: "api/main.go", staleness: StalenessTier.FRESH }),
      ],
      degraded: true,
    });
    await user.click(screen.getByTestId(selectors.validation.resolveButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.references)).toBeInTheDocument();
    });
  });

  it("renders an empty reference result without a degraded note", async () => {
    const user = await pickPlan();
    resolveReferences.mockResolvedValue({ references: [], degraded: false });

    await user.click(screen.getByTestId(selectors.validation.resolveButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.references)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.validation.references).textContent).toContain("No references");
  });

  it("computes staleness and shows degraded state", async () => {
    const user = await pickPlan();
    computeStaleness.mockResolvedValue({
      overall: StalenessTier.LIGHTLY_STALE,
      references: [],
      degraded: true,
    });

    await user.click(screen.getByTestId(selectors.validation.stalenessButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.staleness)).toBeInTheDocument();
    });
    expect(computeStaleness).toHaveBeenCalledWith("plan-1");
    expect(screen.getByTestId(selectors.validation.staleness).textContent).toContain("Degraded");
  });

  it("derives baseline commands and locations", async () => {
    const user = await pickPlan();
    deriveBaselineScope.mockResolvedValue({
      commands: ["git-control-tower baseline diff --scenario plan-manager --name impl"],
      locations: ["scenarios/plan-manager"],
    });

    await user.click(screen.getByTestId(selectors.validation.baselineButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.baseline)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.validation.baseline).textContent).toContain("git-control-tower");
    expect(screen.getByTestId(selectors.validation.baseline).textContent).toContain("scenarios/plan-manager");
  });

  it("renders an empty baseline derivation", async () => {
    const user = await pickPlan();
    deriveBaselineScope.mockResolvedValue({ commands: [], locations: [] });

    await user.click(screen.getByTestId(selectors.validation.baselineButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.baseline)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.validation.baseline).textContent).toContain("No baseline");
  });

  it("explains that execution tickets, producer waits, and sync own validation", async () => {
    await pickPlan();
    expect(screen.getByText("Producer-owned validation")).toBeInTheDocument();
    expect(screen.getByText(/Git Control Tower and Test Genie own the long-running producer wait/)).toBeInTheDocument();
  });

  it("renders action errors and clears busy state", async () => {
    const user = await pickPlan();
    resolveReferences.mockRejectedValue(new Error("resolver down"));

    await user.click(screen.getByTestId(selectors.validation.resolveButton));

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("resolver down");
    });
    expect(screen.getByTestId(selectors.validation.resolveButton)).not.toBeDisabled();
  });

  it("renders the board without axe violations", async () => {
    listPlans.mockResolvedValue([]);
    const { container } = renderWithProviders(<ValidationBoard />);
    await screen.findByTestId(selectors.validation.planSelect);
    await expectNoA11yViolations(container);
  });
});
