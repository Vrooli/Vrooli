import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { goalsService } from "../services/goals-service";
import type { GoalWithScope } from "../types/goal";
import { GoalDetailsPage } from "./GoalDetailsPage";

function makeGoal(): GoalWithScope {
  return {
    goal: {
      name: "workspace",
      title: "Workspace goal",
      description: "Finish the workspace.",
      status: "active",
      priority: 4,
      targets: ["execute/ship-workspace", "initiative/navigation"],
      seeded: false,
      scopeHistory: [
        { at: "2026-07-01T00:00:00Z", targetCount: 1, closureSize: 3, completed: 1 },
      ],
      created: "2026-07-01T00:00:00Z",
      updated: "2026-07-02T00:00:00Z",
    },
    scope: {
      targets: ["execute/ship-workspace", "initiative/navigation"],
      closure: ["execute/ship-workspace", "fix/blocker"],
      completed: ["fix/blocker"],
      ready: ["execute/ship-workspace"],
      blocked: ["fix/blocker"],
      total: 2,
      completedCount: 1,
      blockedCount: 1,
      progressPct: 50,
    },
    eta: {
      p50Hours: 24,
      p80Hours: 48,
      p50Label: "1d",
      p80Label: "2d",
      basis: "velocity",
      basisLabel: "current velocity",
      confidence: "medium",
      remainingItems: 1,
      laneCapacity: 2,
    },
  };
}

function renderPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/goals/:name" element={<GoalDetailsPage />} />
      <Route path="/plan" element={<div>Plan route</div>} />
    </Routes>,
    { initialEntries: ["/goals/workspace"] },
  );
}

describe("GoalDetailsPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("renders goal sections and deep links target chips", async () => {
    vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());

    renderPage();

    await waitFor(() => expect(screen.getByText("Workspace goal")).toBeInTheDocument());
    expect(screen.getByTestId("goal-targets")).toHaveTextContent("ship-workspace");
    expect(screen.getByTestId("goal-targets")).toHaveTextContent("navigation");
    expect(screen.getByTestId("goal-scope")).toHaveTextContent("50%");
    expect(screen.getByTestId("goal-ready")).toHaveTextContent("ship-workspace");
    expect(screen.getByTestId("goal-blocked")).toHaveTextContent("blocker");
    expect(screen.getByTestId("goal-history")).toHaveTextContent("3");
    expect(screen.getByText("1d-2d")).toBeInTheDocument();
    expect(screen.getAllByTestId("goal-ref-execute/ship-workspace")[0]).toHaveAttribute(
      "href",
      "/backlog/execute/ship-workspace",
    );
  });

  it("guards archive and delete with confirmation dialogs", async () => {
    vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
    vi.spyOn(goalsService, "archive").mockResolvedValue();
    vi.spyOn(goalsService, "remove").mockResolvedValue();

    renderPage();

    await waitFor(() => expect(screen.getByTestId("goal-archive")).toBeInTheDocument());
    await userEvent.click(screen.getByTestId("goal-archive"));
    expect(screen.getByTestId("goal-archive-confirm")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("goal-delete"));
    const dialog = screen.getByTestId("goal-delete-confirm");
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByText("workspace")).toBeInTheDocument();
  });
});
