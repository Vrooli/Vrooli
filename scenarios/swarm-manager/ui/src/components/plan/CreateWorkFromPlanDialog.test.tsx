import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { planService } from "../../services/plan-service";
import { CreateWorkFromPlanDialog } from "./CreateWorkFromPlanDialog";

describe("CreateWorkFromPlanDialog", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("imports an existing plan into an milestone container", async () => {
    vi.spyOn(planService, "listCanonicalPlans").mockResolvedValue([{
      id: "plan-1",
      slug: "alpha-plan",
      title: "Alpha Plan",
      status: "READY",
      updatedAt: "2026-07-08T12:00:00Z",
      phaseCount: 2,
    }]);
    const importPlan = vi.spyOn(planService, "importPlan").mockResolvedValue({
      slug: "alpha-plan",
      planId: "plan-1",
      container: "milestone",
      items: [{ kind: "execute", name: "alpha-plan-phase-1", title: "Build", action: "created" }],
      milestone: { name: "alpha", title: "Alpha", mode: "holistic-loop", action: "created" },
      count: 1,
      created: 1,
      linked: 0,
      updated: 0,
    });
    const onImported = vi.fn();

    renderWithProviders(
      <CreateWorkFromPlanDialog isOpen onClose={vi.fn()} onImported={onImported} />,
    );

    await screen.findByText("Alpha Plan");
    await userEvent.click(screen.getByTestId("create-work-plan-option-alpha-plan"));
    await userEvent.click(screen.getByText("Milestone"));
    await userEvent.type(screen.getByPlaceholderText("milestone-name"), "alpha");
    await userEvent.click(screen.getByTestId("create-work-from-plan-submit"));

    await waitFor(() => expect(importPlan).toHaveBeenCalledWith({
      planId: "plan-1",
      sourcePath: undefined,
      markdown: undefined,
      title: undefined,
      slug: undefined,
      container: {
        type: "milestone",
        name: "alpha",
        title: undefined,
        description: undefined,
        mode: "holistic-loop",
      },
    }));
    expect(onImported).toHaveBeenCalledWith(expect.objectContaining({ slug: "alpha-plan" }));
    expect(await screen.findByTestId("create-work-result-links")).toHaveTextContent("alpha-plan");
  });

  it("adopts pasted markdown into backlog items", async () => {
    vi.spyOn(planService, "listCanonicalPlans").mockResolvedValue([]);
    const importPlan = vi.spyOn(planService, "importPlan").mockResolvedValue({
      slug: "scratch-adopted-plan",
      planId: "plan-scratch",
      container: "items",
      items: [{ kind: "execute", name: "scratch-adopted-plan-phase-1", title: "Build", action: "created" }],
      count: 1,
      created: 1,
      linked: 0,
      updated: 0,
    });

    renderWithProviders(
      <CreateWorkFromPlanDialog isOpen onClose={vi.fn()} />,
    );

    await userEvent.click(screen.getByText("Adopt markdown"));
    await userEvent.type(screen.getByTestId("create-work-markdown"), "# Scratch adopted plan\n\n## Phase 1\nBuild it.");
    await userEvent.type(screen.getByPlaceholderText("Plan title"), "Scratch adopted plan");
    await userEvent.type(screen.getByPlaceholderText("stable-slug"), "scratch-adopted-plan");
    await userEvent.click(screen.getByTestId("create-work-from-plan-submit"));

    await waitFor(() => expect(importPlan).toHaveBeenCalledWith({
      planId: undefined,
      sourcePath: undefined,
      markdown: "# Scratch adopted plan\n\n## Phase 1\nBuild it.",
      title: "Scratch adopted plan",
      slug: "scratch-adopted-plan",
      container: {
        type: "items",
        name: undefined,
        title: undefined,
        description: undefined,
        mode: undefined,
      },
    }));
    expect(await screen.findByTestId("create-work-result-links")).toHaveTextContent("scratch-adopted-plan");
  });
});
