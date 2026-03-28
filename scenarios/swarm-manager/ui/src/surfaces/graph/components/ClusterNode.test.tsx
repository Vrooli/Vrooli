import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { ClusterNode } from "./ClusterNode";
import type { NodeProps } from "@xyflow/react";

function renderClusterNode(data: Record<string, unknown>) {
  const props = {
    id: "test-cluster",
    data,
    type: "cluster",
    selected: false,
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
    width: 180,
    height: 72,
  } as unknown as NodeProps;

  return render(
    <ReactFlowProvider>
      <svg>
        <foreignObject>
          <ClusterNode {...props} />
        </foreignObject>
      </svg>
    </ReactFlowProvider>,
  );
}

describe("ClusterNode", () => {
  it("renders initiative title when collapsed", () => {
    renderClusterNode({
      label: "My Initiative",
      collapsed: true,
      rollup: { total: 5, completed: 2, in_progress: 1, failed: 1, pending: 1 },
    });
    expect(screen.getByText("My Initiative")).toBeInTheDocument();
  });

  it("renders rollup badge when collapsed with rollup data", () => {
    renderClusterNode({
      label: "Init",
      collapsed: true,
      rollup: { total: 10, completed: 3, in_progress: 2, failed: 1, pending: 4 },
    });
    expect(screen.getByTestId("rollup-badge")).toBeInTheDocument();
    expect(screen.getByText("10 total")).toBeInTheDocument();
  });

  it("does not render rollup badge when expanded", () => {
    renderClusterNode({
      label: "Init",
      collapsed: false,
      rollup: { total: 5, completed: 2, in_progress: 1, failed: 1, pending: 1 },
    });
    expect(screen.queryByTestId("rollup-badge")).not.toBeInTheDocument();
  });

  it("shows unassigned styling when isUnassigned is true", () => {
    renderClusterNode({
      label: "Unassigned",
      collapsed: true,
      rollup: null,
      isUnassigned: true,
    });
    expect(screen.getByText("unassigned")).toBeInTheDocument();
  });
});
