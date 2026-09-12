import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import { GraphNode } from "./GraphNode";
import { useGraphDataStore } from "../stores/graph-data-store";
import { makeBacklogNode, makeScenarioNode } from "../test-helpers";
import type { GraphNodeData } from "../types";
import type { BacklogStatus } from "../../../types";

function renderGraphNode(
  data: GraphNodeData,
  overrides: Partial<ComponentProps<typeof GraphNode>> = {},
) {
  const props: ComponentProps<typeof GraphNode> = {
    id: "test-node",
    data,
    type: data.entityType,
    selected: false,
    draggable: false,
    isConnectable: true,
    zIndex: 0,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    dragging: false,
    dragHandle: undefined,
    sourcePosition: undefined,
    targetPosition: undefined,
    parentId: undefined,
    deletable: false,
    selectable: true,
    width: 100,
    height: 100,
    ...overrides,
  };

  return render(
    <ReactFlowProvider>
      <svg>
        <foreignObject>
          <GraphNode {...props} />
        </foreignObject>
      </svg>
    </ReactFlowProvider>,
  );
}

describe("GraphNode — actionable badge", () => {
  it.each(["backlog", "researching", "ready", "queued", "in_progress", "failed"] as BacklogStatus[])(
    "shows actionable-badge for actionable status %s in topology lens",
    (status) => {
      useGraphDataStore.setState({ lens: "topology" });
      const node = makeBacklogNode("backlog/execute/test", { status });
      renderGraphNode(node.data);
      expect(screen.getByTestId("actionable-badge")).toBeInTheDocument();
    },
  );

  it.each(["completed"] as BacklogStatus[])(
    "does NOT show actionable-badge for non-actionable status %s",
    (status) => {
      useGraphDataStore.setState({ lens: "topology" });
      const node = makeBacklogNode("backlog/execute/test", { status });
      renderGraphNode(node.data);
      expect(screen.queryByTestId("actionable-badge")).not.toBeInTheDocument();
    },
  );

  it("does NOT show actionable-badge in operations lens", () => {
    useGraphDataStore.setState({ lens: "focus" });
    const node = makeBacklogNode("backlog/execute/test", { status: "backlog" as BacklogStatus });
    renderGraphNode(node.data);
    expect(screen.queryByTestId("actionable-badge")).not.toBeInTheDocument();
  });

  it("does NOT show actionable-badge for non-backlog entity types", () => {
    useGraphDataStore.setState({ lens: "topology" });
    const node = makeScenarioNode("scenario/test", { status: "running" });
    renderGraphNode(node.data);
    expect(screen.queryByTestId("actionable-badge")).not.toBeInTheDocument();
  });

  it("shows both status-badge and actionable-badge when both conditions are met", () => {
    useGraphDataStore.setState({ lens: "topology" });
    const node = makeBacklogNode("backlog/execute/test", {
      status: "in_progress" as BacklogStatus,
      activeExecutionStatus: "running",
    });
    renderGraphNode(node.data);
    expect(screen.getByTestId("status-badge")).toBeInTheDocument();
    expect(screen.getByTestId("actionable-badge")).toBeInTheDocument();
  });
});

import { makeGoalNode } from "../test-helpers";

describe("GraphNode — goal active round chip", () => {
  it("renders the active-round chip with phase label when an active round is present", () => {
    const node = makeGoalNode("goal/foo", {
      activeRound: { mode: "holistic-loop", phase: "investigate", round: 3, status: "agent_running" },
      operatingMode: "holistic-loop",
    });
    renderGraphNode(node.data);
    expect(screen.getByTestId("graph-node-active-round-chip")).toHaveTextContent("Investigate");
  });

  it("does not render the chip on goals without an active round", () => {
    const node = makeGoalNode("goal/foo", {});
    renderGraphNode(node.data);
    expect(screen.queryByTestId("graph-node-active-round-chip")).toBeNull();
  });

  it("renders the chip in a non-pulsing variant when status is reserved", () => {
    const node = makeGoalNode("goal/foo", {
      activeRound: { mode: "holistic-loop", phase: "plan", round: 1, status: "reserved" },
    });
    renderGraphNode(node.data);
    const chip = screen.getByTestId("graph-node-active-round-chip");
    // Reserved chips use the amber palette; agent_running uses cyan.
    expect(chip.className).toMatch(/amber/);
  });
});

describe("GraphNode — goal membership badge", () => {
  it("renders the goal badge when the node belongs to an active goal", () => {
    const node = makeBacklogNode("backlog-item/execute/test");
    renderGraphNode({
      ...node.data,
      goalBadges: [{ name: "goal-a", title: "goal-a", priority: 1 }],
    }, { id: node.id });
    expect(screen.getByTestId("graph-node-goal-badge")).toHaveAttribute("title", "In goal: goal-a");
  });

  it("does not render the goal badge for unsupported graph node ids", () => {
    const node = makeScenarioNode("scenario/test", { status: "running" });
    renderGraphNode(node.data, { id: node.id });
    expect(screen.queryByTestId("graph-node-goal-badge")).not.toBeInTheDocument();
  });
});
