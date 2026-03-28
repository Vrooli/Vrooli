import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

vi.mock("@xyflow/react", async () => {
  const ReactModule = await import("react");
  interface MockGraphNode {
    id: string;
  }
  interface MockGraphEdge {
    id: string;
  }
  interface MockReactFlowProps {
    nodes: MockGraphNode[];
    edges: MockGraphEdge[];
    children?: React.ReactNode;
    onInit?: (instance: { fitView: ReturnType<typeof vi.fn> }) => void;
  }

  return {
    ReactFlow: ({ nodes, edges, children, onInit }: MockReactFlowProps) => {
      ReactModule.useEffect(() => {
        onInit?.({ fitView: vi.fn() });
      }, [onInit]);

      return (
        <div data-testid="mock-react-flow">
          <div data-testid="rendered-node-ids">{nodes.map((node) => node.id).join(",")}</div>
          <div data-testid="rendered-edge-ids">{edges.map((edge) => edge.id).join(",")}</div>
          {children}
        </div>
      );
    },
    Background: () => <div data-testid="background" />,
    BackgroundVariant: { Dots: "dots" },
    MiniMap: () => <div data-testid="minimap" />,
    Handle: () => null,
    Position: { Top: "top", Bottom: "bottom" },
    useNodesState: (initial: unknown[]) => {
      const [nodes, setNodes] = ReactModule.useState(initial);
      return [nodes, setNodes, vi.fn()];
    },
    useEdgesState: (initial: unknown[]) => {
      const [edges, setEdges] = ReactModule.useState(initial);
      return [edges, setEdges, vi.fn()];
    },
  };
});

import { GraphCanvas } from "./GraphCanvas";
import { cloneGraphDataInitialState, useGraphDataStore } from "../stores/graph-data-store";
import { cloneGraphUIInitialState, useGraphUIStore } from "../stores/graph-ui-store";

function resetStores() {
  useGraphDataStore.setState(cloneGraphDataInitialState());
  useGraphUIStore.setState(cloneGraphUIInitialState());
}

describe("GraphCanvas", () => {
  beforeEach(() => {
    resetStores();
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });

  it("renders initiative clusters collapsed on the first paint", async () => {
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes: [
        {
          id: "initiative/graph-adoption",
          type: "initiative",
          position: { x: 0, y: 0 },
          data: {
            label: "Graph Adoption",
            entityType: "initiative",
            status: "active",
            rollup: { total: 2, completed: 0, in_progress: 1, failed: 0, pending: 1 },
          },
        },
        {
          id: "backlog-item/execute/task-a",
          type: "backlog",
          position: { x: 0, y: 0 },
          data: { label: "Task A", entityType: "backlog", kind: "execute", status: "queued" },
        },
        {
          id: "backlog-item/execute/task-b",
          type: "backlog",
          position: { x: 0, y: 0 },
          data: { label: "Task B", entityType: "backlog", kind: "execute", status: "queued" },
        },
      ],
      edges: [
        {
          id: "member_of:a",
          source: "backlog-item/execute/task-a",
          target: "initiative/graph-adoption",
          type: "member_of",
        },
        {
          id: "member_of:b",
          source: "backlog-item/execute/task-b",
          target: "initiative/graph-adoption",
          type: "member_of",
        },
      ],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("initiative/graph-adoption");
    });

    const renderedNodeIds = screen.getByTestId("rendered-node-ids").textContent ?? "";
    expect(renderedNodeIds).not.toContain("backlog-item/execute/task-a");
    expect(renderedNodeIds).not.toContain("backlog-item/execute/task-b");
  });

  it("hides secondary edges when the current lens disables them", async () => {
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
          groupingMode: "none",
          showSecondaryEdges: false,
        },
      },
      nodes: [
        {
          id: "capture/c-1",
          type: "capture",
          position: { x: 0, y: 0 },
          data: { label: "Capture", entityType: "capture", status: "classified" },
        },
        {
          id: "backlog-item/execute/task-a",
          type: "backlog",
          position: { x: 0, y: 0 },
          data: { label: "Task A", entityType: "backlog", kind: "execute", status: "queued" },
        },
      ],
      edges: [
        {
          id: "classified_as:capture->task",
          source: "capture/c-1",
          target: "backlog-item/execute/task-a",
          type: "classified_as",
        },
      ],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids")).toBeInTheDocument();
    });

    expect(screen.getByTestId("rendered-edge-ids").textContent).toBe("");
  });
});
