/**
 * ExecutionRunner tests — start gate, just-in-time context, transition,
 * decision/finding capture, complete + handoff, and axe-clean structure.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import {
  DecisionSchema,
  HandoffSchema,
  PhaseSchema,
  PhaseStatus,
  StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import {
  ExecutionSchema,
  PhaseContextSchema,
} from "@vrooli/proto-types/plan-manager/v1/execution/execution_pb";
import { PlanSchema } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const startExecution = vi.fn();
const getStatus = vi.fn();
const transitionPhase = vi.fn();
const recordDecision = vi.fn();
const recordFinding = vi.fn();
const completeExecution = vi.fn();
const listPlans = vi.fn();

vi.mock("../../api/execution", () => ({
  startExecution: (...a: unknown[]) => startExecution(...a),
  getStatus: (...a: unknown[]) => getStatus(...a),
  transitionPhase: (...a: unknown[]) => transitionPhase(...a),
  recordDecision: (...a: unknown[]) => recordDecision(...a),
  recordFinding: (...a: unknown[]) => recordFinding(...a),
  completeExecution: (...a: unknown[]) => completeExecution(...a),
  getNext: vi.fn(),
  getHandoff: vi.fn(),
  listCandidateFindings: vi.fn(),
  triageFinding: vi.fn(),
  getVelocity: vi.fn(),
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

import { ExecutionRunner } from "./ExecutionRunner";

const execution = create(ExecutionSchema, { id: "exec-1", planId: "plan-1", currentPhaseId: "p1" });
const context = create(PhaseContextSchema, {
  currentPhase: create(PhaseSchema, { id: "p1", order: 1, title: "Contracts", status: PhaseStatus.ACTIVE }),
  requiredReading: ["docs/PLAN.md"],
  reminders: ["keep it small"],
  staleness: StalenessTier.FRESH,
});

const startAndLand = async () => {
  const user = userEvent.setup();
  listPlans.mockResolvedValue([create(PlanSchema, { id: "plan-1", title: "Migrate auth" })]);
  startExecution.mockResolvedValue(execution);
  getStatus.mockResolvedValue({ execution, context });

  renderWithProviders(<ExecutionRunner />);
  await waitFor(() => {
    expect(
      screen.getByTestId(selectors.execution.planSelect).querySelector('option[value="plan-1"]'),
    ).not.toBeNull();
  });
  await user.selectOptions(screen.getByTestId(selectors.execution.planSelect), "plan-1");
  await user.click(screen.getByTestId(selectors.execution.startButton));
  await screen.findByTestId(selectors.execution.context);
  return user;
};

describe("ExecutionRunner", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("starts a run and renders just-in-time context", async () => {
    await startAndLand();
    expect(startExecution).toHaveBeenCalledWith("plan-1", "");
    expect(screen.getByTestId(selectors.execution.context)).toBeInTheDocument();
  });

  it("transitions the current phase", async () => {
    const user = await startAndLand();
    transitionPhase.mockResolvedValue({ execution });
    await user.click(screen.getByTestId(selectors.execution.transitionButton));
    await waitFor(() => {
      expect(transitionPhase).toHaveBeenCalledWith("exec-1", "p1", PhaseStatus.DONE);
    });
  });

  it("records a decision in-flow", async () => {
    const user = await startAndLand();
    recordDecision.mockResolvedValue(create(DecisionSchema, { id: "d1", summary: "chose Connect" }));
    await user.type(screen.getByTestId(selectors.execution.decisionSummary), "chose Connect");
    await user.click(screen.getByTestId(selectors.execution.recordDecisionButton));
    await waitFor(() => {
      expect(recordDecision).toHaveBeenCalledWith("exec-1", "p1", "chose Connect", "");
    });
  });

  it("completes the run and shows the handoff", async () => {
    const user = await startAndLand();
    completeExecution.mockResolvedValue({
      handoff: create(HandoffSchema, { id: "h1", executionId: "exec-1", staleness: StalenessTier.FRESH }),
      nudges: [],
    });
    await user.click(screen.getByTestId(selectors.execution.completeButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.execution.handoff)).toBeInTheDocument();
    });
  });

  it("renders the runner without axe violations", async () => {
    listPlans.mockResolvedValue([]);
    const { container } = renderWithProviders(<ExecutionRunner />);
    await screen.findByTestId(selectors.execution.startForm);
    await expectNoA11yViolations(container);
  });
});
