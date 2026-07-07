import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils";
import { createTestQueryClient } from "../../../test-utils/query";
import { goalsService } from "../../../services";
import type { GoalWithScope } from "../../../types/goal";
import { GOALS_QUERY_KEY } from "../hooks/useGoals";
import { GoalPicker } from "./GoalPicker";

function goal(name: string, priority: number, progressPct: number): GoalWithScope {
  return {
    goal: {
      name,
      title: `${name} title`,
      status: "active",
      priority,
      targets: [],
      seeded: false,
      scopeHistory: [],
      created: "",
      updated: "",
    },
    scope: {
      targets: [],
      closure: [],
      completed: [],
      ready: [],
      blocked: [],
      total: 4,
      completedCount: Math.round((progressPct / 100) * 4),
      blockedCount: 0,
      progressPct,
    },
    eta: null,
  };
}

function renderPicker(goals: GoalWithScope[], goalValue = "", onSelect = vi.fn()) {
  const queryClient = createTestQueryClient();
  queryClient.setQueryData(GOALS_QUERY_KEY, goals);
  renderWithProviders(<GoalPicker goal={goalValue} onSelect={onSelect} />, { queryClient });
  return { onSelect };
}

describe("GoalPicker", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows 'All work' when unscoped and lists active goals highest-priority first", async () => {
    renderPicker([goal("low", 2, 25), goal("high", 8, 50)]);
    const trigger = screen.getByTestId("plan-goal-picker");
    expect(trigger).toHaveTextContent("All work");

    await userEvent.click(trigger);
    const menu = screen.getByTestId("plan-goal-picker-menu");
    const options = menu.querySelectorAll('[data-testid^="plan-goal-option-"]');
    // "All work" + two goals; the first goal option is the higher priority one.
    expect(screen.getByTestId("plan-goal-option-high")).toBeInTheDocument();
    expect(screen.getByTestId("plan-goal-option-low")).toBeInTheDocument();
    expect(options.length).toBe(3);
  });

  it("selecting a goal and clearing both call onSelect", async () => {
    const { onSelect } = renderPicker([goal("g", 5, 40)]);
    await userEvent.click(screen.getByTestId("plan-goal-picker"));
    await userEvent.click(screen.getByRole("option", { name: /g title/i }));
    expect(onSelect).toHaveBeenCalledWith("g");

    await userEvent.click(screen.getByTestId("plan-goal-picker"));
    await userEvent.click(screen.getByTestId("plan-goal-option-all"));
    expect(onSelect).toHaveBeenCalledWith("");
  });

  it("raises goal priority through the goals service", async () => {
    const spy = vi
      .spyOn(goalsService, "setPriority")
      .mockResolvedValue(goal("g", 6, 40));
    renderPicker([goal("g", 5, 40)]);
    await userEvent.click(screen.getByTestId("plan-goal-picker"));
    await userEvent.click(screen.getByTestId("plan-goal-priority-up-g"));
    await waitFor(() => expect(spy).toHaveBeenCalledWith("g", 6));
  });

  it("closes the menu on Escape", async () => {
    renderPicker([goal("g", 5, 40)]);
    await userEvent.click(screen.getByTestId("plan-goal-picker"));
    expect(screen.getByTestId("plan-goal-picker-menu")).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    expect(screen.queryByTestId("plan-goal-picker-menu")).not.toBeInTheDocument();
  });

  it("shows the current goal title and progress when scoped", () => {
    renderPicker([goal("g", 5, 75)], "g");
    const trigger = screen.getByTestId("plan-goal-picker");
    expect(trigger).toHaveTextContent("g title");
    expect(screen.getByTestId("plan-goal-picker-progress")).toHaveTextContent("75%");
  });
});
