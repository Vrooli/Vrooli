import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ComponentProps } from "react";
import { goalsService } from "../../../../services";
import type { GoalWithScope } from "../../../../types/goal";
import { GoalsTab } from "./GoalsTab";

function makeGoal(name: string, priority: number, progressPct: number): GoalWithScope {
  return {
    goal: {
      name,
      title: name.replace(/-/g, " "),
      description: "",
      status: "active",
      priority,
      targets: [`fix/${name}`],
      seeded: false,
      scopeHistory: [],
      created: "2026-07-01T00:00:00Z",
      updated: "2026-07-02T00:00:00Z",
    },
    scope: {
      targets: [`fix/${name}`],
      closure: [`fix/${name}`],
      completed: progressPct === 100 ? [`fix/${name}`] : [],
      ready: [`fix/${name}`],
      blocked: [],
      total: 4,
      completedCount: Math.round((progressPct / 100) * 4),
      blockedCount: 0,
      progressPct,
    },
    eta: {
      p50Hours: 24,
      p80Hours: 48,
      p50Label: "1d",
      p80Label: "2d",
      basis: "velocity",
      basisLabel: "velocity",
      confidence: "medium",
      remainingItems: 2,
      laneCapacity: 1,
    },
  };
}

function renderTab(overrides: Partial<ComponentProps<typeof GoalsTab>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const props = {
    searchQuery: "",
    sort: { field: "priority" as const, direction: "desc" as const },
    onItemClick: vi.fn(),
    onClearSearch: vi.fn(),
    ...overrides,
  };
  render(
    <QueryClientProvider client={queryClient}>
      <GoalsTab {...props} />
    </QueryClientProvider>,
  );
  return props;
}

describe("GoalsTab", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders active goals with progress and ETA", async () => {
    vi.spyOn(goalsService, "list").mockResolvedValue([
      makeGoal("second-goal", 2, 25),
      makeGoal("top-goal", 8, 50),
    ]);

    renderTab();

    await waitFor(() => expect(screen.getByTestId("goal-row-top-goal")).toBeInTheDocument());
    const row = screen.getByTestId("goal-row-top-goal");
    expect(row).toHaveTextContent("top goal");
    expect(row).toHaveTextContent("50% · 2/4");
    expect(row).toHaveTextContent("ETA 1d-2d");
  });

  it("shows the empty state", async () => {
    vi.spyOn(goalsService, "list").mockResolvedValue([]);

    renderTab();

    await waitFor(() => expect(screen.getByText("No goals yet.")).toBeInTheDocument());
  });

  it("changes priority through the mutation", async () => {
    vi.spyOn(goalsService, "list").mockResolvedValue([makeGoal("top-goal", 8, 50)]);
    const setPriority = vi.spyOn(goalsService, "setPriority").mockResolvedValue(makeGoal("top-goal", 9, 50));

    renderTab();

    await waitFor(() => expect(screen.getByTestId("goal-priority-up-top-goal")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("goal-priority-up-top-goal"));

    await waitFor(() => expect(setPriority).toHaveBeenCalledWith("top-goal", 9));
  });

  it("navigates to the goal node on row click", async () => {
    vi.spyOn(goalsService, "list").mockResolvedValue([makeGoal("top-goal", 8, 50)]);
    const onItemClick = vi.fn();

    renderTab({ onItemClick });

    await waitFor(() => expect(screen.getByTestId("goal-row-top-goal")).toBeInTheDocument());
    fireEvent.click(screen.getByText("top goal"));

    expect(onItemClick).toHaveBeenCalledWith("goal/top-goal");
  });
});
