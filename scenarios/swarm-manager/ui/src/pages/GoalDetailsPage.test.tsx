import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { goalsService } from "../services/goals-service";
import { nextActionService } from "../services/next-action-service";
import { ApiError } from "../lib/api-client";
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
      targets: ["execute/ship-workspace"],
      milestones: [
        { name: "foundation", title: "Foundation", description: "Lay the groundwork.", items: ["fix/blocker"], acceptanceCriteria: [], dependsOn: [] },
        { name: "delivery", title: "Delivery", items: ["execute/ship-workspace"], acceptanceCriteria: [], dependsOn: ["foundation"] },
      ],
      seeded: false,
      scopeHistory: [
        { at: "2026-07-01T00:00:00Z", targetCount: 1, closureSize: 3, completed: 1 },
      ],
      created: "2026-07-01T00:00:00Z",
      updated: "2026-07-02T00:00:00Z",
    },
    scope: {
      targets: ["execute/ship-workspace"],
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
    vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);

    renderPage();

    await waitFor(() => expect(screen.getByText("Workspace goal")).toBeInTheDocument());
    expect(screen.getByTestId("goal-overview")).toHaveTextContent("50%");
    expect(screen.getByTestId("goal-overview")).toHaveTextContent("P4");
    expect(screen.getByTestId("goal-targets")).toHaveTextContent("ship-workspace");
    expect(screen.getByTestId("goal-scope")).toHaveTextContent("50%");
    expect(screen.getByTestId("goal-ready")).toHaveTextContent("ship-workspace");
    expect(screen.getByTestId("goal-blocked")).toHaveTextContent("blocker");
    await userEvent.click(screen.getByRole("tab", { name: "Milestones" }));
    expect(screen.getByTestId("goal-milestones")).toHaveTextContent("Foundation");
    expect(screen.getByTestId("goal-milestones")).toHaveTextContent("Delivery");
    expect(screen.getByTestId("goal-milestone-foundation")).toHaveTextContent("1 assigned");
    expect(screen.getByTestId("goal-milestone-foundation-toggle")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Overview" }));
    expect(screen.getByText("1d-2d")).toBeInTheDocument();
    // Scope Creep is collapsed by default; expanding reveals the history table.
    expect(screen.getByTestId("goal-history")).not.toHaveTextContent("3");
    await userEvent.click(screen.getByTestId("goal-history-toggle"));
    expect(screen.getByTestId("goal-history")).toHaveTextContent("3");

    expect(screen.getAllByTestId("goal-ref-execute/ship-workspace")[0]).toHaveAttribute(
      "href",
      "/backlog/execute/ship-workspace",
    );
  });

  it("guards archive and delete with confirmation dialogs", async () => {
    vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
    vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);
    vi.spyOn(goalsService, "archive").mockResolvedValue();
    vi.spyOn(goalsService, "remove").mockResolvedValue();

    renderPage();

    // Archive and delete live in the header overflow menu.
    await waitFor(() => expect(screen.getByTestId("detail-header-actions")).toBeInTheDocument());
    await userEvent.click(screen.getByTestId("detail-header-actions"));
    await userEvent.click(screen.getByTestId("goal-archive"));
    expect(screen.getByTestId("goal-archive-confirm")).toBeInTheDocument();
    await userEvent.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByTestId("goal-archive-confirm")).toBeNull());

    await userEvent.click(screen.getByTestId("detail-header-actions"));
    await userEvent.click(screen.getByTestId("goal-delete"));
    const dialog = screen.getByTestId("goal-delete-confirm");
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByText("workspace")).toBeInTheDocument();
  });

  // The header's primary action is projected by the server, so the page has to
  // handle every action id it can emit. `review` and `define_criteria` had no
  // branch and fell through to a navigate() carrying a query param nothing
  // reads — pressing the button did nothing at all, with no error to see.
  describe("primary action dispatch", () => {
    function feedEntry(id: string, target: string) {
      return {
        entries: [{
          entity_kind: "goal" as const,
          entity_ref: "workspace",
          entity_title: "Workspace goal",
          action: { id, compact_label: "Start review", expanded_label: "Start milestone review", enabled: true, target },
          tier: 1,
        }],
      };
    }

    it("starts the milestone review when the projected action is review", async () => {
      vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
      vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);
      const startReview = vi.spyOn(goalsService, "startMilestoneReview")
        .mockResolvedValue({ execution_id: "exec-1", definition_digest: "digest" });
      vi.spyOn(nextActionService, "getFeed")
        .mockResolvedValue(feedEntry("review", "milestone_review:delivery") as never);

      renderPage();

      const button = await screen.findByRole("button", { name: /Start review/ });
      await userEvent.click(button);

      await waitFor(() => expect(startReview).toHaveBeenCalledWith("workspace", "delivery"));
      // Starting a review only queues the run, so the confirmation must not
      // claim the review is finished.
      expect(await screen.findByText("Review started for delivery")).toBeInTheDocument();
    });

    it("reports the failure instead of going quiet when the review cannot start", async () => {
      vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
      vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);
      vi.spyOn(goalsService, "startMilestoneReview")
        .mockRejectedValue(new ApiError("http", "milestone has no acceptance criteria", { status: 400 }));
      vi.spyOn(nextActionService, "getFeed")
        .mockResolvedValue(feedEntry("review", "milestone_review:delivery") as never);

      renderPage();

      await userEvent.click(await screen.findByRole("button", { name: /Start review/ }));

      const alert = await screen.findByRole("alert");
      expect(alert).toHaveTextContent("Couldn't start the milestone review");
      // The server's own 4xx explanation is the useful part and must survive.
      expect(alert).toHaveTextContent("milestone has no acceptance criteria");
    });

    it("opens the milestone editor when the projected action is define_criteria", async () => {
      vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
      vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);
      vi.spyOn(nextActionService, "getFeed")
        .mockResolvedValue(feedEntry("define_criteria", "milestone_criteria:foundation") as never);

      renderPage();

      await userEvent.click(await screen.findByRole("button", { name: /Start review/ }));

      // The operator lands on the milestone that needs criteria, not the goal.
      const drawer = await screen.findByRole("dialog");
      expect(within(drawer).getByDisplayValue("Foundation")).toBeInTheDocument();
    });

    it("says so when the action names a milestone this goal no longer has", async () => {
      vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
      vi.spyOn(goalsService, "getFiles").mockResolvedValue([]);
      vi.spyOn(nextActionService, "getFeed")
        .mockResolvedValue(feedEntry("define_criteria", "milestone_criteria:renamed-away") as never);

      renderPage();

      await userEvent.click(await screen.findByRole("button", { name: /Start review/ }));

      expect(await screen.findByText("That action can't be completed from this page")).toBeInTheDocument();
    });
  });

  it("uses the editable shared file workspace for goal files", async () => {
    vi.spyOn(goalsService, "get").mockResolvedValue(makeGoal());
    vi.spyOn(goalsService, "getFiles").mockResolvedValue([
      { name: "notes.md", path: "notes.md", type: "file", size: 12, children: [] },
    ]);

    renderPage();
    await waitFor(() => expect(screen.getByText("Workspace goal")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("tab", { name: "Files" }));

    expect(await screen.findByText("No file selected")).toBeInTheDocument();
    expect(screen.getByTestId("backlog-details-file-tree")).toHaveTextContent("notes.md");
  });
});
