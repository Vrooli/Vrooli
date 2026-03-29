import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import { ClusterNode } from "./ClusterNode";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { makeClusterNodeData } from "../test-helpers";

function renderClusterNode(data: Parameters<typeof makeClusterNodeData>[0]) {
  const props: ComponentProps<typeof ClusterNode> = {
    id: "test-cluster",
    data: makeClusterNodeData(data),
    type: "cluster",
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
    width: 180,
    height: 72,
  };

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

  it("toggles cluster expansion when chevron is clicked", async () => {
    const toggleSpy = vi.fn();
    useGraphUIStore.setState({ toggleTopologyCluster: toggleSpy });

    renderClusterNode({
      label: "Clickable Init",
      collapsed: true,
      rollup: { total: 3, completed: 1, in_progress: 1, failed: 0, pending: 1 },
    });

    const toggle = screen.getByTestId("cluster-toggle");
    await userEvent.click(toggle);
    expect(toggleSpy).toHaveBeenCalledWith("test-cluster");
  });

  it("shows expand label when collapsed and collapse label when expanded", () => {
    const { unmount } = render(
      <ReactFlowProvider>
        <svg>
          <foreignObject>
            <ClusterNode
              {...{
                id: "c1",
                data: makeClusterNodeData({ label: "A", collapsed: true, rollup: null }),
                type: "cluster",
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
                width: 180,
                height: 72,
              }}
            />
          </foreignObject>
        </svg>
      </ReactFlowProvider>,
    );
    expect(screen.getByLabelText("Expand cluster")).toBeInTheDocument();
    unmount();

    render(
      <ReactFlowProvider>
        <svg>
          <foreignObject>
            <ClusterNode
              {...{
                id: "c2",
                data: makeClusterNodeData({ label: "B", collapsed: false, rollup: null }),
                type: "cluster",
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
                width: 180,
                height: 72,
              }}
            />
          </foreignObject>
        </svg>
      </ReactFlowProvider>,
    );
    expect(screen.getByLabelText("Collapse cluster")).toBeInTheDocument();
  });
});
