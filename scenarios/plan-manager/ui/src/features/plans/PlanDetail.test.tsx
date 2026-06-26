/**
 * PlanDetail tests — detail rendering, not-found empty state, markdown toggle,
 * and axe-clean structure. The api/plans boundary is mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { i18n, setLocale } from "../../i18n";
import { strings } from "../../consts/strings";
import {
  PlanSchema,
  PlanEdgeSchema,
  PlanStatus,
  PhaseSchema,
  PhaseStatus,
  ReferenceSchema,
  RegressionAnchorSchema,
  StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const getPlan = vi.fn();
const getGraph = vi.fn();
const renderPlan = vi.fn();
const archivePlan = vi.fn();

vi.mock("../../api/plans", () => ({
  getPlan: (...a: unknown[]) => getPlan(...a),
  getGraph: (...a: unknown[]) => getGraph(...a),
  renderPlan: (...a: unknown[]) => renderPlan(...a),
  archivePlan: (...a: unknown[]) => archivePlan(...a),
  listPlans: vi.fn(),
  listTemplates: vi.fn(),
  createFromTemplate: vi.fn(),
}));

import { PlanDetail } from "./PlanDetail";

const fullPlan = create(PlanSchema, {
  id: "plan-1",
  slug: "migrate-auth",
  title: "Migrate auth",
  status: PlanStatus.ACTIVE,
  updatedAt: "2026-06-25T10:00:00Z",
  purpose: "Move auth to Connect.",
  scope: "auth scenario only",
  definitionOfDone: "all green",
  references: [
    create(ReferenceSchema, { id: "r1", target: "api/main.go", staleness: StalenessTier.FRESH }),
  ],
  regressionAnchor: create(RegressionAnchorSchema, {
    strategy: "scenario_baseline",
    scenario: "auth",
    commands: ["vrooli scenario test auth"],
  }),
  phases: [
    create(PhaseSchema, {
      id: "p1",
      order: 1,
      title: "Contracts",
      intent: "define proto",
      status: PhaseStatus.DONE,
      requiredReading: ["docs/PLAN.md"],
    }),
  ],
});

describe("PlanDetail", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the not-found empty state when the plan is absent", async () => {
    getPlan.mockResolvedValue(undefined);
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="missing" />);

    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.pages.planDetail}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("renders the plan body with phases and references", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-1" />);

    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plans.phase({ id: "p1" }))).toBeInTheDocument();
  });

  it("renders empty optional plan sections without placeholders leaking", async () => {
    getPlan.mockResolvedValue(create(PlanSchema, { id: "plan-empty", title: "Empty plan" }));
    getGraph.mockResolvedValue([]);

    renderWithProviders(<PlanDetail planId="plan-empty" />);

    expect(await screen.findByTestId(selectors.pages.planDetail)).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.noPhases))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.noReferences))).toBeInTheDocument();
    expect(screen.getByText(i18n.t(strings.pages.plans.detail.anchorNone))).toBeInTheDocument();
    expect(await screen.findByText(i18n.t(strings.pages.plans.detail.edgeNone))).toBeInTheDocument();
  });

  it("renders supersession graph directions", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([
      create(PlanEdgeSchema, { fromPlanId: "plan-1", toPlanId: "old-plan", kind: "supersedes" }),
      create(PlanEdgeSchema, { fromPlanId: "new-plan", toPlanId: "plan-1", kind: "supersedes" }),
    ]);

    renderWithProviders(<PlanDetail planId="plan-1" />);

    const graph = await screen.findByTestId(selectors.plans.detailGraph);
    expect(graph.textContent).toContain("old-plan");
    expect(graph.textContent).toContain("new-plan");
  });

  it("lazily fetches and shows rendered markdown when toggled", async () => {
    const user = userEvent.setup();
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);
    renderPlan.mockResolvedValue("# Migrate auth");

    renderWithProviders(<PlanDetail planId="plan-1" />);

    await screen.findByTestId(selectors.pages.planDetail);
    expect(renderPlan).not.toHaveBeenCalled();

    await user.click(screen.getByTestId(selectors.plans.detailMarkdownToggle));
    await waitFor(() => {
      expect(renderPlan).toHaveBeenCalledWith("plan-1");
      expect(screen.getByTestId(selectors.plans.detailMarkdown)).toBeInTheDocument();
    });
  });

  it("renders the detail without axe violations", async () => {
    getPlan.mockResolvedValue(fullPlan);
    getGraph.mockResolvedValue([]);

    const { container } = renderWithProviders(<PlanDetail planId="plan-1" />);
    await screen.findByTestId(selectors.pages.planDetail);
    await expectNoA11yViolations(container);
  });
});
