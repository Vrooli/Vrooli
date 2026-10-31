/**
 * PlansList tests — list rendering, empty state, and create-from-template.
 * The api/plans boundary is mocked so no real Connect call fires.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import {
  PlanSchema,
  PlanStatus,
  PhaseSchema,
  PhaseStatus,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { PlanTemplateSchema } from "@vrooli/proto-types/plan-manager/v1/plans/plans_pb";

const listPlans = vi.fn();
const listTemplates = vi.fn();
const createFromTemplate = vi.fn();

vi.mock("../../api/plans", () => ({
  listPlans: (...args: unknown[]) => listPlans(...args),
  listTemplates: (...args: unknown[]) => listTemplates(...args),
  createFromTemplate: (...args: unknown[]) => createFromTemplate(...args),
  // Unused by PlansList but imported by the shared hook module.
  getPlan: vi.fn(),
  getGraph: vi.fn(),
  renderPlan: vi.fn(),
  archivePlan: vi.fn(),
}));

import { PlansList } from "./PlansList";

const samplePlan = create(PlanSchema, {
  id: "plan-1",
  title: "Migrate auth",
  status: PlanStatus.ACTIVE,
  updatedAt: "2026-06-25T10:00:00Z",
  phases: [
    create(PhaseSchema, { id: "p1", status: PhaseStatus.DONE }),
    create(PhaseSchema, { id: "p2", status: PhaseStatus.TODO }),
  ],
});

describe("PlansList", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a loading state then the plan rows", async () => {
    listPlans.mockResolvedValue([samplePlan]);
    listTemplates.mockResolvedValue([]);

    renderWithProviders(<PlansList />);

    expect(
      await screen.findByTestId(selectors.plans.row({ id: "plan-1" })),
    ).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plans.list)).toBeInTheDocument();
  });

  it("renders the empty state when there are no plans", async () => {
    listPlans.mockResolvedValue([]);
    listTemplates.mockResolvedValue([]);

    renderWithProviders(<PlansList />);

    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.plans.list}-${selectors.asyncSuffix.empty}`),
      ).toBeInTheDocument();
    });
  });

  it("renders an error state when the list query rejects", async () => {
    listPlans.mockRejectedValue(new Error("boom"));
    listTemplates.mockResolvedValue([]);

    renderWithProviders(<PlansList />);

    await waitFor(() => {
      expect(
        screen.getByTestId(`${selectors.plans.list}-${selectors.asyncSuffix.error}`),
      ).toBeInTheDocument();
    });
  });

  it("creates a plan from the selected template", async () => {
    const user = userEvent.setup();
    listPlans.mockResolvedValue([]);
    listTemplates.mockResolvedValue([
      create(PlanTemplateSchema, { id: "tpl-1", name: "Standard" }),
    ]);
    createFromTemplate.mockResolvedValue(samplePlan);

    renderWithProviders(<PlansList />);

    await screen.findByTestId(selectors.plans.createForm);
    // The template <option> only appears once listTemplates resolves.
    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.plans.templateSelect).querySelector('option[value="tpl-1"]'),
      ).not.toBeNull();
    });
    await user.selectOptions(screen.getByTestId(selectors.plans.templateSelect), "tpl-1");
    await user.type(screen.getByTestId(selectors.plans.titleInput), "New plan");
    await user.click(screen.getByTestId(selectors.plans.createButton));

    await waitFor(() => {
      expect(createFromTemplate).toHaveBeenCalledWith("tpl-1", "New plan");
    });
  });

  it("references the create button copy through the strings registry", async () => {
    listPlans.mockResolvedValue([]);
    listTemplates.mockResolvedValue([]);
    renderWithProviders(<PlansList />);
    expect(await screen.findByText(strings.pages.plans.create)).toBeInTheDocument();
  });
});
