import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { completeMilestone, createGoal, createMilestone, fetchGoals, fetchMilestones, updateGoalTargetDate } from "../api/goals";
import { renderWithProviders } from "../test-utils";
import { GoalsPage } from "./GoalsPage";
import { fetchWorkItems } from "../api/work";
import { fetchGoalDrifts, fetchGoalVariances } from "../api/review";

vi.mock("../api/goals", () => ({ createGoal: vi.fn(), fetchGoals: vi.fn(), updateGoalProgress: vi.fn(), updateGoalTargetDate: vi.fn(), fetchMilestones: vi.fn(), createMilestone: vi.fn(), completeMilestone: vi.fn() }));
vi.mock("../api/work", () => ({ fetchWorkItems: vi.fn().mockResolvedValue([]) }));
vi.mock("../api/review", () => ({ fetchGoalVariances: vi.fn().mockResolvedValue([]), fetchGoalDrifts: vi.fn().mockResolvedValue([]) }));
afterEach(() => { cleanup(); vi.clearAllMocks(); vi.unstubAllGlobals(); });

describe("GoalsPage", () => {
  it("uses the compact mobile goal-card composition", async () => {
    vi.stubGlobal("matchMedia", () => ({ matches: true, addEventListener: vi.fn(), removeEventListener: vi.fn() }));
    vi.mocked(fetchGoals).mockResolvedValue([{ id: "goal-mobile", title: "Protect mornings", purpose: "", status: "active", progressMethod: "manual", progressBasisPoints: 4200n, targetBasisPoints: 10000n, revision: 1n }] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    renderWithProviders(<GoalsPage />);
    expect((await screen.findAllByText("42%")).length).toBeGreaterThan(0);
    expect(document.querySelector(".goal-card-mobile")).toBeTruthy();
  });

  it("renders stored early, on-time, and over goal variance badges", async () => {
    vi.mocked(fetchGoals).mockResolvedValue([
      { id: "early", title: "Early", purpose: "", status: "complete", progressMethod: "manual", progressBasisPoints: 10000n, targetBasisPoints: 10000n, revision: 1n },
      { id: "ontime", title: "On time", purpose: "", status: "complete", progressMethod: "manual", progressBasisPoints: 10000n, targetBasisPoints: 10000n, revision: 1n },
      { id: "over", title: "Over", purpose: "", status: "complete", progressMethod: "manual", progressBasisPoints: 10000n, targetBasisPoints: 10000n, revision: 1n },
    ] as never);
    vi.mocked(fetchGoalVariances).mockResolvedValue([
      { goalId: "early", targetDate: "2026-09-20", completedDate: "2026-09-18", deltaDays: -2, label: "2 days early" },
      { goalId: "ontime", targetDate: "2026-09-20", completedDate: "2026-09-20", deltaDays: 0, label: "on time" },
      { goalId: "over", targetDate: "2026-09-20", completedDate: "2026-09-23", deltaDays: 3, label: "3 days over" },
    ]);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    renderWithProviders(<GoalsPage />);
    await userEvent.setup().click(await screen.findByRole("tab", { name: "Completed" }));
    expect(await screen.findByLabelText("Completed 2 days early")).toBeInTheDocument();
    expect(screen.getByLabelText("Completed on time")).toBeInTheDocument();
    expect(screen.getByLabelText("Completed 3 days over")).toBeInTheDocument();
  });

  it("nudges an active goal when stored pace is behind", async () => {
    vi.mocked(fetchGoals).mockResolvedValue([{ id: "goal-1", title: "Ship", purpose: "", status: "active", progressMethod: "manual", progressBasisPoints: 1000n, targetBasisPoints: 10000n, revision: 1n }] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(fetchGoalDrifts).mockResolvedValue([{ goalId: "goal-1", targetDate: "2026-10-01", expectedBasis: 5000, actualBasis: 1000, driftBasis: -4000, label: "Behind pace" }]);
    renderWithProviders(<GoalsPage />);
    expect(await screen.findByLabelText("Goal Behind pace")).toBeInTheDocument();
  });

  it("lets an active goal owner save a target date", async () => {
    vi.mocked(fetchGoals).mockResolvedValue([{ id: "goal-1", title: "Ship", purpose: "", status: "active", progressMethod: "manual", progressBasisPoints: 1000n, targetBasisPoints: 10000n, revision: 1n }] as never);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(fetchGoalDrifts).mockResolvedValue([{ goalId: "goal-1", targetDate: "", expectedBasis: 0, actualBasis: 1000, driftBasis: 1000, label: "" }]);
    renderWithProviders(<GoalsPage />);
    const input = await screen.findByLabelText("Target date for Ship");
    await userEvent.setup().clear(input);
    await userEvent.setup().type(input, "2026-10-01");
    await userEvent.setup().click(screen.getByRole("button", { name: "Save target" }));
    expect(updateGoalTargetDate).toHaveBeenCalledWith({ id: "goal-1", targetDate: "2026-10-01" });
  });

  it("filters the list through the explicit Active, All, and Completed views", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchGoals).mockResolvedValue([]);
    renderWithProviders(<GoalsPage />);
    await screen.findByRole("button", { name: "+ New goal" });
    await user.click(screen.getByRole("tab", { name: "All" }));
    await user.click(screen.getByRole("tab", { name: "Completed" }));
    await user.click(screen.getByRole("tab", { name: "Active" }));
    expect(screen.getByRole("tab", { name: "Active" })).toHaveAttribute("aria-selected", "true");
  });
  it("creates an explicit outcome without treating work time as progress", async () => {
    const user = userEvent.setup();
    vi.mocked(fetchGoals).mockResolvedValue([]);
    vi.mocked(fetchMilestones).mockResolvedValue([]);
    vi.mocked(createGoal).mockResolvedValue({ id: "goal-1", title: "Protect mornings", purpose: "Be deliberate", status: "active", progressMethod: "manual", progressBasisPoints: 0n, targetBasisPoints: 10000n, revision: 1n } as never);
    renderWithProviders(<GoalsPage />);
    await user.click(await screen.findByRole("button", { name: "+ New goal" }));
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
    const slider = await screen.findByRole("slider");
    expect(slider).toBeInTheDocument();
    fireEvent.change(slider, { target: { value: "5000" } });
    fireEvent.mouseUp(slider);
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
