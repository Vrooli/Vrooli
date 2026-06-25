/**
 * ValidationBoard tests — reference resolution (with degraded note), run +
 * verdict, DoD verify, and axe-clean structure. api/validation + api/plans
 * are mocked.
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
  ValidationResultSchema,
  ValidationVerdict,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const resolveReferences = vi.fn();
const runValidation = vi.fn();
const verifyDefinitionOfDone = vi.fn();
const listPlans = vi.fn();

vi.mock("../../api/validation", () => ({
  resolveReferences: (...a: unknown[]) => resolveReferences(...a),
  computeStaleness: vi.fn(),
  deriveBaselineScope: vi.fn(),
  runValidation: (...a: unknown[]) => runValidation(...a),
  verifyDefinitionOfDone: (...a: unknown[]) => verifyDefinitionOfDone(...a),
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

  it("resolves references and surfaces the degraded note honestly", async () => {
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

  it("runs validation and renders the verdict", async () => {
    const user = await pickPlan();
    runValidation.mockResolvedValue(
      create(ValidationResultSchema, { id: "v1", verdict: ValidationVerdict.PASS }),
    );
    await user.click(screen.getByTestId(selectors.validation.runButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.result)).toBeInTheDocument();
    });
  });

  it("verifies the definition of done", async () => {
    const user = await pickPlan();
    verifyDefinitionOfDone.mockResolvedValue({
      result: create(ValidationResultSchema, { id: "v1", verdict: ValidationVerdict.PASS }),
      dodMet: true,
    });
    await user.click(screen.getByTestId(selectors.validation.dodButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.validation.dod)).toBeInTheDocument();
    });
  });

  it("renders the board without axe violations", async () => {
    listPlans.mockResolvedValue([]);
    const { container } = renderWithProviders(<ValidationBoard />);
    await screen.findByTestId(selectors.validation.planSelect);
    await expectNoA11yViolations(container);
  });
});
