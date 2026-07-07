import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { GOALS_QUERY_KEY } from "../../plan/hooks/useGoals";
import { goalsService } from "../../../services";
import { renderWithProviders, createTestQueryClient } from "../../../test-utils";
import type { GoalWithScope } from "../../../types/goal";
import { useGraphDataStore, graphDataInitialState } from "../stores/graph-data-store";
import { useGraphUIStore, graphUIInitialState } from "../stores/graph-ui-store";
import { makeBacklogNode, makeScenarioNode } from "../test-helpers";
import type { CappedGraphNodeData, GraphNode } from "../types";
import { NodeInspectorPanel } from "./NodeInspectorPanel";

function goal(name: string, targets: string[], closure: string[] = targets, priority = 1): GoalWithScope {
  return {
    goal: {
      name,
      title: name,
      description: "",
      status: "active",
      priority,
      targets,
      seeded: false,
      scopeHistory: [],
      created: "",
      updated: "",
    },
    scope: {
      targets,
      closure,
      completed: [],
      ready: [],
      blocked: [],
      total: closure.length,
      completedCount: 0,
      blockedCount: 0,
      progressPct: 0,
    },
    eta: null,
  };
}

function renderInspector(node: GraphNode, goals: GoalWithScope[] = []) {
  const queryClient = createTestQueryClient();
  queryClient.setQueryData(GOALS_QUERY_KEY, goals);
  useGraphDataStore.setState({
    ...graphDataInitialState,
    lens: "topology",
    nodes: [node],
  });
  useGraphUIStore.setState({
    ...graphUIInitialState,
    selectedNodeId: node.id,
  });
  renderWithProviders(<NodeInspectorPanel />, {
    queryClient,
    initialEntries: [`/graph?select=${encodeURIComponent(node.id)}`],
  });
  return queryClient;
}

describe("NodeInspectorPanel goal actions", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    useGraphDataStore.setState({ ...graphDataInitialState });
    useGraphUIStore.setState({ ...graphUIInitialState });
  });

  it("shows goal membership and can add a backlog node to an existing goal", async () => {
    const existingGoal = goal("goal-a", ["execute/other"], ["execute/thing", "execute/other"], 7);
    const node = makeBacklogNode("backlog-item/execute/thing", { title: "Thing" });
    const addTargets = vi.spyOn(goalsService, "addTargets").mockResolvedValue(existingGoal);

    renderInspector(node, [existingGoal]);

    expect(screen.getByTestId("inspector-goal-membership")).toHaveTextContent("goal-a");
    await userEvent.click(screen.getByTestId("inspector-set-goal"));
    await userEvent.click(screen.getByTestId("set-as-goal-add-goal-a"));

    await waitFor(() => {
      expect(addTargets).toHaveBeenCalledWith("goal-a", ["execute/thing"]);
    });
  });

  it("can create a new goal for an initiative node", async () => {
    const createdGoal = goal("new-goal", ["initiative/payments"]);
    const create = vi.spyOn(goalsService, "create").mockResolvedValue(createdGoal);
    const node = {
      id: "initiative/payments",
      type: "initiative",
      position: { x: 0, y: 0 },
      data: {
        label: "Payments",
        entityType: "initiative",
        rawType: "Initiative",
        name: "payments",
        title: "Payments",
        status: "active",
        rollup: { total: 0, completed: 0, in_progress: 0, failed: 0, pending: 0 },
      },
    } satisfies GraphNode;

    renderInspector(node);

    await userEvent.click(screen.getByTestId("inspector-set-goal"));
    await userEvent.click(screen.getByTestId("set-as-goal-create"));

    await waitFor(() => {
      expect(create).toHaveBeenCalledWith({ title: "Payments", targets: ["initiative/payments"] });
    });
  });

  it("does not offer goal assignment for unsupported graph nodes", () => {
    const node = makeScenarioNode("scenario/swarm-manager");

    renderInspector(node);

    expect(screen.queryByTestId("inspector-set-goal")).not.toBeInTheDocument();
    expect(screen.getByTestId("inspector-goal-unsupported")).toHaveTextContent(
      "Goal targets are available for backlog items and initiatives.",
    );
  });

  it("does not turn synthetic capped backlog nodes into invalid goal refs", () => {
    const cappedNode: GraphNode = {
      id: "capped/backlog",
      type: "backlog",
      position: { x: 0, y: 0 },
      data: {
        label: "More backlog items",
        entityType: "backlog",
        rawType: "Synthetic",
        status: "capped",
        isCapNode: true,
      } satisfies CappedGraphNodeData,
    };

    renderInspector(cappedNode);

    expect(screen.queryByTestId("inspector-set-goal")).not.toBeInTheDocument();
    expect(screen.getByTestId("inspector-goal-unsupported")).toBeInTheDocument();
  });
});
