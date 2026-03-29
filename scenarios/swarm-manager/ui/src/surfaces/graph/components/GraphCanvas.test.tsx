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
    fitView?: boolean;
    defaultViewport?: { x: number; y: number; zoom: number };
  }

  return {
    ReactFlow: ({ nodes, edges, children, onInit, fitView, defaultViewport }: MockReactFlowProps) => {
      ReactModule.useEffect(() => {
        onInit?.({ fitView: vi.fn() });
      }, [onInit]);

      return (
        <div data-testid="mock-react-flow">
          <div data-testid="fit-view-flag">{String(fitView)}</div>
          <div data-testid="default-viewport">{JSON.stringify(defaultViewport ?? null)}</div>
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
import {
  makeExecutionNode,
  makeGraphEdge,
  makeInitiativeNode,
  makeBacklogNode,
  makeCaptureNode,
  makeScenarioNode,
} from "../test-helpers";

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
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
          groupingMode: "initiative",
        },
      },
      nodes: [
        makeInitiativeNode("initiative/graph-adoption", {
          label: "Graph Adoption",
          status: "active",
          rollup: { total: 2, completed: 0, in_progress: 1, failed: 0, pending: 1 },
        }),
        makeBacklogNode("backlog-item/execute/task-a", {
          label: "Task A",
          status: "queued",
        }),
        makeBacklogNode("backlog-item/execute/task-b", {
          label: "Task B",
          status: "queued",
        }),
      ],
      edges: [
        makeGraphEdge(
          "member_of:a",
          "backlog-item/execute/task-a",
          "initiative/graph-adoption",
          "member_of",
        ),
        makeGraphEdge(
          "member_of:b",
          "backlog-item/execute/task-b",
          "initiative/graph-adoption",
          "member_of",
        ),
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

  it("keeps backlog items visible in flat topology view", async () => {
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
          groupingMode: "none",
        },
      },
      nodes: [
        makeInitiativeNode("initiative/graph-adoption", {
          label: "Graph Adoption",
          status: "active",
        }),
        makeBacklogNode("backlog-item/execute/task-a", {
          label: "Task A",
          status: "backlog",
        }),
        makeScenarioNode("scenario/swarm-manager", {
          label: "Swarm Manager",
          status: "running",
        }),
      ],
      edges: [
        makeGraphEdge(
          "member_of:a",
          "backlog-item/execute/task-a",
          "initiative/graph-adoption",
          "member_of",
        ),
      ],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids")).toBeInTheDocument();
    });

    const renderedNodeIds = screen.getByTestId("rendered-node-ids").textContent ?? "";
    expect(renderedNodeIds).toContain("backlog-item/execute/task-a");
    expect(renderedNodeIds).toContain("initiative/graph-adoption");
    expect(renderedNodeIds).toContain("scenario/swarm-manager");
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
        makeCaptureNode("capture/c-1", {
          label: "Capture",
          text: "Capture",
          status: "classified",
        }),
        makeBacklogNode("backlog-item/execute/task-a", {
          label: "Task A",
          status: "queued",
        }),
      ],
      edges: [
        makeGraphEdge(
          "classified_as:capture->task",
          "capture/c-1",
          "backlog-item/execute/task-a",
          "classified_as",
        ),
      ],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids")).toBeInTheDocument();
    });

    expect(screen.getByTestId("rendered-edge-ids").textContent).toBe("");
  });

  it("does not reuse another lens viewport when the active lens has no saved camera", async () => {
    useGraphUIStore.setState((state) => ({
      ...state,
      viewportByLens: {
        ...state.viewportByLens,
        topology: { x: 120, y: 80, zoom: 0.9 },
      },
    }));

    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "operations",
      nodes: [
        makeExecutionNode("execution-record/exec-1", {
          label: "Execution 1",
          status: "running",
        }),
      ],
      edges: [],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("execution-record/exec-1");
    });

    expect(screen.getByTestId("fit-view-flag").textContent).toBe("true");
    expect(screen.getByTestId("default-viewport").textContent).toBe(
      JSON.stringify({ x: 0, y: 0, zoom: 1 }),
    );
  });
});
