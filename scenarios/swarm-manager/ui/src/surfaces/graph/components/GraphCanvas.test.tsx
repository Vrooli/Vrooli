import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));

vi.mock("@xyflow/react", async () => {
  const ReactModule = await import("react");
  interface MockGraphNode {
    id: string;
    style?: Record<string, unknown>;
  }
  interface MockGraphEdge {
    id: string;
    style?: Record<string, unknown>;
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
          <div data-testid="node-opacities">{JSON.stringify(
            nodes.reduce<Record<string, number | undefined>>((acc, n) => {
              acc[n.id] = n.style?.opacity as number | undefined;
              return acc;
            }, {}),
          )}</div>
          <div data-testid="edge-opacities">{JSON.stringify(
            edges.reduce<Record<string, number | undefined>>((acc, e) => {
              acc[e.id] = e.style?.opacity as number | undefined;
              return acc;
            }, {}),
          )}</div>
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
    MarkerType: { ArrowClosed: "arrowclosed" },
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

  it("dims non-highlighted nodes and edges when a node is selected", async () => {
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
        makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
        makeBacklogNode("backlog-item/execute/task-b", { label: "B", status: "queued" }),
        makeScenarioNode("scenario/app", { label: "App", status: "running" }),
      ],
      edges: [
        makeGraphEdge("e1", "backlog-item/execute/task-a", "backlog-item/execute/task-b", "depends_on"),
      ],
    }));

    // Simulate selecting task-a → highlights task-a and task-b (neighbors), dims scenario/app
    useGraphUIStore.setState((state) => ({
      ...state,
      highlightState: {
        highlighted: new Set(["backlog-item/execute/task-a", "backlog-item/execute/task-b"]),
        mode: "dim" as const,
      },
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("scenario/app");
    });

    const nodeOpacities = JSON.parse(screen.getByTestId("node-opacities").textContent ?? "{}");
    // Highlighted nodes should be full opacity
    expect(nodeOpacities["backlog-item/execute/task-a"]).toBe(1);
    expect(nodeOpacities["backlog-item/execute/task-b"]).toBe(1);
    // Non-highlighted node should be dimmed
    expect(nodeOpacities["scenario/app"]).toBe(0.5);

    const edgeOpacities = JSON.parse(screen.getByTestId("edge-opacities").textContent ?? "{}");
    // Edge between two highlighted nodes should NOT be dimmed
    expect(edgeOpacities["e1"]).toBeUndefined(); // no opacity override → full
  });

  it("dims edges where either endpoint is not highlighted", async () => {
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
        makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
        makeScenarioNode("scenario/app", { label: "App", status: "running" }),
      ],
      edges: [
        makeGraphEdge("e1", "backlog-item/execute/task-a", "scenario/app", "targets"),
      ],
    }));

    // Only task-a is highlighted, scenario/app is NOT
    useGraphUIStore.setState((state) => ({
      ...state,
      highlightState: {
        highlighted: new Set(["backlog-item/execute/task-a"]),
        mode: "dim" as const,
      },
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-edge-ids").textContent).toContain("e1");
    });

    const edgeOpacities = JSON.parse(screen.getByTestId("edge-opacities").textContent ?? "{}");
    // Edge has one non-highlighted endpoint → should be dimmed
    expect(edgeOpacities["e1"]).toBe(0.15);
  });

  it("all nodes return to full opacity when highlight is cleared", async () => {
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
        makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
        makeScenarioNode("scenario/app", { label: "App", status: "running" }),
      ],
      edges: [],
    }));

    // Normal mode — no highlighting
    useGraphUIStore.setState((state) => ({
      ...state,
      highlightState: {
        highlighted: new Set<string>(),
        mode: "normal" as const,
      },
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("scenario/app");
    });

    const nodeOpacities = JSON.parse(screen.getByTestId("node-opacities").textContent ?? "{}");
    expect(nodeOpacities["backlog-item/execute/task-a"]).toBe(1);
    expect(nodeOpacities["scenario/app"]).toBe(1);
  });
});
