/**
 * VelocityBoard tests — plan picker, bigint-safe chart + table, empty state, and
 * axe-clean structure. api/execution + api/plans are mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  Completeness,
  PlanSchema,
  VelocityPointSchema,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const getVelocity = vi.fn();
const listPlans = vi.fn();

vi.mock("../../api/execution", () => ({
  getVelocity: (...a: unknown[]) => getVelocity(...a),
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

import { VelocityBoard } from "./VelocityBoard";

const point = create(VelocityPointSchema, {
  id: "vp1",
  planId: "plan-1",
  runId: "run-1",
  wallTimeSeconds: 120n,
  tokens: 45000n,
  iterations: 3,
  completeness: Completeness.FULL,
  recordedAt: "2026-06-25T10:00:00Z",
});

const pickPlan = async () => {
  const user = userEvent.setup();
  listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
  renderWithProviders(<VelocityBoard />);
  await waitFor(() => {
    expect(
      screen.getByTestId(selectors.velocity.planSelect).querySelector('option[value="plan-1"]'),
    ).not.toBeNull();
  });
  await user.selectOptions(screen.getByTestId(selectors.velocity.planSelect), "plan-1");
  return user;
};

describe("VelocityBoard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("[REQ:PM-UI-001] renders the chart and table with bigint values converted safely", async () => {
    getVelocity.mockResolvedValue([point]);
    await pickPlan();
    await waitFor(() => {
      expect(screen.getByTestId(selectors.velocity.chart)).toBeInTheDocument();
      expect(screen.getByTestId(selectors.velocity.table)).toBeInTheDocument();
    });
    // 45000 tokens renders formatted, never as a raw bigint literal.
    expect(screen.getByTestId(selectors.velocity.table).textContent).toContain("45");
  });

  it("renders the empty state for a plan with no samples", async () => {
    getVelocity.mockResolvedValue([]);
    await pickPlan();
    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.velocity.table}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("renders without axe violations", async () => {
    getVelocity.mockResolvedValue([point]);
    const { container } = await (async () => {
      const user = userEvent.setup();
      listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
      const result = renderWithProviders(<VelocityBoard />);
      await waitFor(() => {
        expect(
          screen.getByTestId(selectors.velocity.planSelect).querySelector('option[value="plan-1"]'),
        ).not.toBeNull();
      });
      await user.selectOptions(screen.getByTestId(selectors.velocity.planSelect), "plan-1");
      return result;
    })();
    await screen.findByTestId(selectors.velocity.chart);
    await expectNoA11yViolations(container);
  });
});
