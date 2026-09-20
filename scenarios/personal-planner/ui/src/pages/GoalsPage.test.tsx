import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { completeMilestone, createGoal, createMilestone, fetchGoals, fetchMilestones } from "../api/goals";
import { renderWithProviders } from "../test-utils";
import { GoalsPage } from "./GoalsPage";
import { fetchWorkItems } from "../api/work";

vi.mock("../api/goals", () => ({ createGoal: vi.fn(), fetchGoals: vi.fn(), updateGoalProgress: vi.fn(), fetchMilestones: vi.fn(), createMilestone: vi.fn(), completeMilestone: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn().mockResolvedValue([]) }));
afterEach(() => { cleanup(); vi.clearAllMocks(); });

describe("GoalsPage", () => {
  it("creates an explicit outcome without treating work time as progress", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchGoals).mockResolvedValue([]);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(createGoal).mockResolvedValue({ id: "goal-1", title: "Protect mornings", purpose: "Be deliberate", status: "active", progressMethod: "manual", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n } as never);
    renderWithProviders(<GoalsPage />);
    await user.type(await screen.findByLabelText("Goal title"), "Protect mornings");
    await user.type(screen.getByLabelText("Purpose"), "Be deliberate");
    await user.click(screen.getByRole("button", { name: "Progress" }));
    await user.click(screen.getByRole("option", { name: "Derive from milestones" }));
    await user.click(screen.getByRole("button", { name: "Create goal" }));
    expect(createGoal).toHaveBeenCalledWith({ title: "Protect mornings", purpose: "Be deliberate", progressMethod: "milestones", targetBasisPoints: 10000n });
  });

  it("records progress separately from goal creation", async () => {
    const goal = { id: "goal-1", title: "Protect mornings", purpose: "Be deliberate", status: "active", progressMethod: "manual", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n };
    vi.mocked(fetchGoals).mockResolvedValue([goal] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(fetchWorkItems).mockResolvedValue([{ id: "work-1", title: "Draft morning plan" }] as never);
    renderWithProviders(<GoalsPage />);
    expect(await screen.findByRole("slider")).toBeInTheDocument();
  });

  it("states when goal data is unavailable", async () => {
    vi.mocked(fetchGoals).mockRejectedValue(new Error("offline"));
    renderWithProviders(<GoalsPage />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Goals are unavailable");
  });

  it("records a milestone with explicit criteria and due date", async () => {
    const user = userEvent.setup();
    const goal = { id: "goal-1", title: "Protect mornings", purpose: "Be deliberate", status: "active", progressMethod: "manual", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n };
    vi.mocked(fetchGoals).mockResolvedValue([goal] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(createMilestone).mockResolvedValue({ id: "m-1", goalId: "goal-1", title: "No meetings", criteria: "No meetings before 10", dueDate: "2026-10-01", status: "open", revision: 1n } as never);
    renderWithProviders(<GoalsPage />);
    await user.type(await screen.findByLabelText("Milestone title"), "No meetings");
    await user.type(screen.getByLabelText("Completion criteria"), "No meetings before 10");
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-10-01" } });
    await user.click(screen.getByRole("button", { name: "Linked work" }));
    await user.click(screen.getByRole("option", { name: "Draft morning plan" }));
    await user.click(screen.getByRole("button", { name: "Add milestone" }));
    expect(createMilestone).toHaveBeenCalledWith({ goalId: "goal-1", title: "No meetings", criteria: "No meetings before 10", dueDate: "2026-10-01", linkedWorkItemId: "work-1", prerequisiteMilestoneIds: [] });
  });

  it("shows completion state and can confirm an open milestone", async () => {
    const user = userEvent.setup();
    const goal = { id: "goal-1", title: "Protect mornings", purpose: "Be deliberate", status: "active", progressMethod: "manual", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n };
    vi.mocked(fetchGoals).mockResolvedValue([goal] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([{ id: "m-1", goalId: "goal-1", title: "No meetings", criteria: "No meetings before 10", dueDate: "2026-10-01", status: "open", revision: 1n }] as never);
    vi.mocked(completeMilestone).mockResolvedValue({ id: "m-1", goalId: "goal-1", title: "No meetings", status: "complete", revision: 2n } as never);
    renderWithProviders(<GoalsPage />);
    await user.click(await screen.findByRole("button", { name: "Mark complete" }));
    expect(completeMilestone).toHaveBeenCalledWith(expect.objectContaining({ id: "m-1", revision: 1n }));
  });

  it("shows a dependent milestone as waiting until its prerequisite is complete", async () => {
    const user = userEvent.setup();
    const goal = { id: "goal-1", title: "Ship", purpose: "", status: "active", progressMethod: "milestones", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n };
    vi.mocked(fetchGoals).mockResolvedValue([goal] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([
      { id: "m-1", goalId: "goal-1", title: "Foundation", status: "open", revision: 1n, prerequisiteMilestoneIds: [] },
      { id: "m-2", goalId: "goal-1", title: "Launch", status: "open", revision: 1n, prerequisiteMilestoneIds: ["m-1"] },
    ] as never);
    renderWithProviders(<GoalsPage />);
    expect(await screen.findByRole("button", { name: "Waiting" })).toBeDisabled();
    await user.click(screen.getByLabelText("Foundation", { selector: "input" }));
  });
});
