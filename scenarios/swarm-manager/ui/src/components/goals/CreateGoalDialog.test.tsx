import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { goalsService } from "../../services";
import { useBacklogStore } from "../../stores";
import type { BacklogItem } from "../../types";
import type { GoalWithScope } from "../../types/goal";
import { CreateGoalDialog } from "./CreateGoalDialog";

function makeBacklogItem(kind: BacklogItem["kind"], name: string, title: string): BacklogItem {
  return {
    kind,
    name,
    title,
    description: "",
    status: "ready",
    priority: 1,
    tags: [],
    created: "2026-07-01T00:00:00Z",
    updated: "2026-07-01T00:00:00Z",
  } as unknown as BacklogItem;
}

function makeGoal(name: string): GoalWithScope {
  return {
    goal: {
      name,
      title: name,
      status: "active",
      priority: 0,
      targets: [],
      milestones: [],
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
      total: 0,
      completedCount: 0,
      blockedCount: 0,
      progressPct: 0,
    },
    eta: null,
  };
}

function seedStores() {
  useBacklogStore.getState().setItems([
    makeBacklogItem("execute", "workspace-goal", "Workspace goal"),
  ]);
}

describe("CreateGoalDialog", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    useBacklogStore.getState().reset();
  });

  it("requires a title", async () => {
    seedStores();
    renderWithProviders(<CreateGoalDialog isOpen onClose={vi.fn()} />);

    await userEvent.click(screen.getByTestId("create-goal-submit"));

    expect(screen.getByText("Title is required.")).toBeInTheDocument();
  });

  it("creates a goal with selected targets", async () => {
    seedStores();
    const create = vi.spyOn(goalsService, "create").mockResolvedValue(makeGoal("workspace"));
    const onCreated = vi.fn();

    renderWithProviders(
      <CreateGoalDialog isOpen onClose={vi.fn()} onCreated={onCreated} />,
    );

    await userEvent.type(screen.getByTestId("create-goal-title"), "Workspace");
    await userEvent.click(screen.getByText("Workspace goal"));
    await userEvent.click(screen.getByTestId("create-goal-submit"));

    await waitFor(() => expect(create).toHaveBeenCalledWith({
      title: "Workspace",
      description: undefined,
      targets: ["execute/workspace-goal"],
    }));
    expect(onCreated).toHaveBeenCalledWith(expect.objectContaining({
      goal: expect.objectContaining({ name: "workspace" }),
    }));
  });
});
