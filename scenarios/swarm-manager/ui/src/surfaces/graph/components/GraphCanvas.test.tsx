import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import React from "react";

const { mockUseGoalMembershipIndex } = vi.hoisted(() => ({
  mockUseGoalMembershipIndex: vi.fn(() => new Map()),
}));

const mockNavigate = vi.fn();
vi.mock("react-router-dom", () => ({
  useNavigate: () => mockNavigate,
  useSearchParams: () => [new URLSearchParams(), vi.fn()],
}));

vi.mock("../hooks/useGoalMembership", () => ({
  nodeIdToGoalRef: (nodeId: string) => {
    if (nodeId.startsWith("backlog-item/")) return nodeId.slice("backlog-item/".length);
    if (nodeId.startsWith("goal/")) return nodeId;
    return null;
  },
  useGoalMembershipIndex: mockUseGoalMembershipIndex,
}));

// Module-scoped spies so tests can assert on React Flow interactions.
const mockFitView = vi.fn();
const mockSetCenter = vi.fn();
const mockGetViewport = vi.fn(() => ({ x: 0, y: 0, zoom: 1 }));

vi.mock("@xyflow/react", async () => {
  const ReactModule = await import("react");
  interface MockGraphNode {
    id: string;
    data?: { goalBadges?: unknown[] };
    style?: Record<string, unknown>;
  }
  interface MockGraphEdge {
    id: string;
    type?: string;
    data?: { relationshipType?: unknown };
    style?: Record<string, unknown>;
  }
  interface MockReactFlowProps {
    nodes: MockGraphNode[];
    edges: MockGraphEdge[];
    children?: React.ReactNode;
    onlyRenderVisibleElements?: boolean;
    onInit?: (instance: {
      fitView: typeof mockFitView;
      setCenter: typeof mockSetCenter;
      getViewport: typeof mockGetViewport;
    }) => void;
    fitView?: boolean;
  }

  return {
    ReactFlow: ({ nodes, edges, children, onInit, fitView, onlyRenderVisibleElements }: MockReactFlowProps) => {
      ReactModule.useEffect(() => {
        onInit?.({ fitView: mockFitView, setCenter: mockSetCenter, getViewport: mockGetViewport });
      }, [onInit]);

      return (
        <div data-testid="mock-react-flow">
          <div data-testid="fit-view-flag">{String(fitView)}</div>
          <div data-testid="only-render-visible-elements">{String(Boolean(onlyRenderVisibleElements))}</div>
          <div data-testid="rendered-node-ids">{nodes.map((node) => node.id).join(",")}</div>
          <div data-testid="rendered-edge-ids">{edges.map((edge) => edge.id).join(",")}</div>
          <div data-testid="rendered-edge-types">{edges.map((edge) => edge.type ?? "").join(",")}</div>
          <div data-testid="rendered-edge-relationship-types">{edges.map((edge) => String(edge.data?.relationshipType ?? "")).join(",")}</div>
          <div data-testid="node-goal-badge-counts">{JSON.stringify(
            nodes.reduce<Record<string, number>>((acc, n) => {
              acc[n.id] = n.data?.goalBadges?.length ?? 0;
              return acc;
            }, {}),
          )}</div>
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
import { useGraphSettingsStore, cloneGraphSettingsInitialState } from "../stores/graph-settings-store";
import { cloneGraphUIInitialState, useGraphUIStore } from "../stores/graph-ui-store";
import {
  makeExecutionNode,
  makeGraphEdge,
  makeGoalNode,
  makeBacklogNode,
  makeCaptureNode,
  makeScenarioNode,
} from "../test-helpers";

function resetStores() {
  useGraphDataStore.setState(cloneGraphDataInitialState());
  useGraphSettingsStore.setState(cloneGraphSettingsInitialState());
  useGraphUIStore.setState(cloneGraphUIInitialState());
}

describe("GraphCanvas", () => {
  beforeEach(() => {
    resetStores();
    mockFitView.mockClear();
    mockSetCenter.mockClear();
    mockGetViewport.mockClear();
    mockUseGoalMembershipIndex.mockReset();
    mockUseGoalMembershipIndex.mockReturnValue(new Map());
    mockGetViewport.mockReturnValue({ x: 0, y: 0, zoom: 1 });
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    });
    vi.stubGlobal("cancelAnimationFrame", vi.fn());
  });

  it("keeps backlog items visible in flat topology view", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
        },
      },
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes: [
        makeGoalNode("goal/graph-adoption", {
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
          "goal/graph-adoption",
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
    expect(renderedNodeIds).toContain("goal/graph-adoption");
    expect(renderedNodeIds).toContain("scenario/swarm-manager");
  });

  it("hides secondary edges when the current lens disables them", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
          groupingMode: "none",
          showSecondaryEdges: false,
        },
      },
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
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

  it("does not reuse another lens intent when the active lens has no saved intent", async () => {
    useGraphUIStore.setState((state) => ({
      ...state,
      viewportIntentByLens: {
        ...state.viewportIntentByLens,
        topology: { nodeId: "scenario/swarm-manager", zoom: 0.9 },
      },
    }));

    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "focus",
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

    // No intent for `operations` lens → should fit-view, not set-center.
    expect(mockFitView).toHaveBeenCalled();
    expect(mockSetCenter).not.toHaveBeenCalled();
  });

  it("restores viewport intent when the node still exists", async () => {
    useGraphUIStore.setState((state) => ({
      ...state,
      viewportIntentByLens: {
        ...state.viewportIntentByLens,
        focus: { nodeId: "execution-record/exec-1", zoom: 1.4 },
      },
    }));

    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "focus",
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
      expect(mockSetCenter).toHaveBeenCalled();
    });

    const [, , options] = mockSetCenter.mock.calls[0] ?? [];
    expect(options).toMatchObject({ zoom: 1.4, duration: 0 });
  });

  it("falls back to fitView when intent's node is no longer present", async () => {
    useGraphUIStore.setState((state) => ({
      ...state,
      viewportIntentByLens: {
        ...state.viewportIntentByLens,
        operations: { nodeId: "execution-record/exec-does-not-exist", zoom: 1.4 },
      },
    }));

    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "focus",
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

    expect(mockSetCenter).not.toHaveBeenCalled();
    expect(mockFitView).toHaveBeenCalled();
  });

  it("dims non-highlighted nodes and edges when a node is selected", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
        },
      },
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
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

    const nodeOpacities: Record<string, number> = JSON.parse(screen.getByTestId("node-opacities").textContent ?? "{}") as Record<string, number>;
    // Highlighted nodes should be full opacity
    expect(nodeOpacities["backlog-item/execute/task-a"]).toBe(1);
    expect(nodeOpacities["backlog-item/execute/task-b"]).toBe(1);
    // Non-highlighted node should be dimmed
    expect(nodeOpacities["scenario/app"]).toBe(0.5);

    const edgeOpacities: Record<string, number | undefined> = JSON.parse(screen.getByTestId("edge-opacities").textContent ?? "{}") as Record<string, number | undefined>;
    // Edge between two highlighted nodes should NOT be dimmed
    expect(edgeOpacities["e1"]).toBeUndefined(); // no opacity override → full
  });

  it("dims edges where either endpoint is not highlighted", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
        },
      },
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
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

    const edgeOpacities: Record<string, number> = JSON.parse(screen.getByTestId("edge-opacities").textContent ?? "{}") as Record<string, number>;
    // Edge has one non-highlighted endpoint → should be dimmed
    expect(edgeOpacities["e1"]).toBe(0.15);
  });

  it("all nodes return to full opacity when highlight is cleared", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
      settingsByLens: {
        ...state.settingsByLens,
        topology: {
          ...state.settingsByLens.topology,
        },
      },
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
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

    // In normal mode, nodes have no opacity override (undefined = browser default = fully visible).
    // This is a performance optimization: we skip creating new style objects in normal mode.
    const nodeOpacities: Record<string, number | undefined> = JSON.parse(screen.getByTestId("node-opacities").textContent ?? "{}") as Record<string, number | undefined>;
    expect(nodeOpacities["backlog-item/execute/task-a"]).toBeUndefined();
    expect(nodeOpacities["scenario/app"]).toBeUndefined();
  });

  it("injects precomputed goal badges once at the canvas boundary", async () => {
    mockUseGoalMembershipIndex.mockReturnValue(new Map([
      ["execute/task-a", [{ name: "goal-a", title: "Goal A", priority: 3 }]],
    ]));

    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes: [
        makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
        makeScenarioNode("scenario/app", { label: "App", status: "running" }),
      ],
      edges: [],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("backlog-item/execute/task-a");
    });

    const badgeCounts = JSON.parse(
      screen.getByTestId("node-goal-badge-counts").textContent ?? "{}",
    ) as Record<string, number>;
    expect(badgeCounts["backlog-item/execute/task-a"]).toBe(1);
    expect(badgeCounts["scenario/app"]).toBe(0);
  });

  it("uses straight edge rendering for large graphs without losing relationship metadata", async () => {
    const nodes = [
      makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
      makeBacklogNode("backlog-item/execute/task-b", { label: "B", status: "queued" }),
    ];
    const edges = Array.from({ length: 300 }, (_, i) =>
      makeGraphEdge(`member_of:${i}`, "backlog-item/execute/task-a", "backlog-item/execute/task-b", "member_of"),
    );

    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes,
      edges,
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-edge-ids").textContent).toContain("member_of:0");
    });

    expect(screen.getByTestId("rendered-edge-types").textContent?.split(",")).toEqual(
      Array(300).fill("straight"),
    );
    expect(screen.getByTestId("rendered-edge-relationship-types").textContent?.split(",")).toEqual(
      Array(300).fill("member_of"),
    );
  });

  it("keeps small graphs on full rendering without large-graph overlay overhead", async () => {
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes: [
        makeBacklogNode("backlog-item/execute/task-a", { label: "A", status: "queued" }),
        makeScenarioNode("scenario/app", { label: "App", status: "running" }),
      ],
      edges: [
        makeGraphEdge("e1", "backlog-item/execute/task-a", "scenario/app", "targets"),
      ],
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("scenario/app");
    });

    expect(screen.getByTestId("only-render-visible-elements").textContent).toBe("false");
    expect(screen.getByTestId("background")).toBeInTheDocument();
    expect(screen.queryByTestId("graph-topology-summary")).toBeNull();
  });

  it("enables viewport rendering and topology scope for large graphs", async () => {
    const nodes = Array.from({ length: 300 }, (_, i) =>
      makeBacklogNode(`backlog-item/execute/task-${i}`, { label: `Task ${i}`, status: "queued" }),
    );
    const edges = nodes.slice(1).map((node, i) =>
      makeGraphEdge(`depends_on:${i}`, nodes[i]!.id, node.id, "depends_on"),
    );

    mockUseGoalMembershipIndex.mockReturnValue(new Map([
      ["execute/task-42", [{ name: "goal-a", title: "Goal A", priority: 3 }]],
    ]));
    useGraphSettingsStore.setState((state) => ({
      ...state,
      activeLens: "topology",
    }));
    useGraphDataStore.setState((state) => ({
      ...state,
      lens: "topology",
      nodes,
      edges,
    }));

    render(<GraphCanvas />);

    await waitFor(() => {
      expect(screen.getByTestId("rendered-node-ids").textContent).toContain("backlog-item/execute/task-42");
    });

    expect(screen.getByTestId("only-render-visible-elements").textContent).toBe("true");
    expect(screen.queryByTestId("background")).toBeNull();
    expect(screen.getByTestId("graph-topology-summary")).toHaveTextContent("300 nodes / 299 edges");

    const badgeCounts = JSON.parse(
      screen.getByTestId("node-goal-badge-counts").textContent ?? "{}",
    ) as Record<string, number>;
    expect(badgeCounts["backlog-item/execute/task-42"]).toBe(1);
  });
});
