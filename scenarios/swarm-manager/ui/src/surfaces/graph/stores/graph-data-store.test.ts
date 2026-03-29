import { beforeEach, describe, expect, it, vi } from "vitest";
import type { GraphProjection, GraphRequestOptions } from "../../../services/graph-service";
import type { GraphEdge, GraphNode } from "../types";
import { makeGraphEdge, makeGraphNode, makeRunNode } from "../test-helpers";

const { getGraphMock } = vi.hoisted(() => ({
  getGraphMock: vi.fn<(lens: string, options?: GraphRequestOptions) => Promise<GraphProjection>>(),
}));

vi.mock("../../../services", () => ({
  graphService: {
    getGraph: getGraphMock,
  },
}));

import {
  cloneGraphDataInitialState,
  createGraphDataInitialState,
  resetGraphRequestState,
  useGraphDataStore,
} from "./graph-data-store";

function resetStore() {
  resetGraphRequestState();
  useGraphDataStore.setState(cloneGraphDataInitialState());
  window.localStorage.clear();
  getGraphMock.mockReset();
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve;
    reject = nextReject;
  });
  return { promise, resolve, reject };
}

const makeNode = (id: string, type: Parameters<typeof makeGraphNode>[1] = "scenario"): GraphNode =>
  makeGraphNode(id, type, { label: id });

const makeEdge = (source: string, target: string): GraphEdge =>
  makeGraphEdge(`${source}->${target}`, source, target);

describe("graphDataStore", () => {
  beforeEach(resetStore);

  it("starts with empty graph data", () => {
    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual([]);
    expect(state.edges).toEqual([]);
    expect(state.meta).toBeNull();
    expect(state.graphsByLens.topology.meta).toBeNull();
  });

  it("defaults to the topology lens with initiative grouping", () => {
    const state = useGraphDataStore.getState();
    expect(state.lens).toBe("topology");
    expect(state.settingsByLens.topology.groupingMode).toBe("initiative");
    expect(state.settingsByLens.flow.groupingMode).toBe("none");
    expect(state.settingsByLens.operations.groupingMode).toBe("none");
  });

  it("migrates legacy grouped topology settings to the new topology default", () => {
    window.localStorage.setItem(
      "swarm-manager.graph.settings.v2",
      JSON.stringify({
        topology: {
          entityFilters: { capture: false },
          groupingMode: "initiative",
          showSecondaryEdges: false,
          autoFitOnChange: true,
        },
      }),
    );

    const initialState = createGraphDataInitialState();
    // Legacy migration resets groupingMode to the default for the lens.
    // Topology's default is now "initiative".
    expect(initialState.settingsByLens.topology.groupingMode).toBe("initiative");
    expect(initialState.settingsByLens.topology.entityFilters.capture).toBe(false);
    expect(initialState.settingsByLens.topology.showSecondaryEdges).toBe(false);
  });

  it("migrates v3 flat status filters to v4 grouped format", () => {
    window.localStorage.setItem(
      "swarm-manager.graph.settings.v3",
      JSON.stringify({
        topology: {
          entityFilters: {},
          statusFilters: { completed: false, failed: false },
          groupingMode: "none",
          showSecondaryEdges: true,
          autoFitOnChange: true,
        },
      }),
    );

    const initialState = createGraphDataInitialState();
    const filters = initialState.settingsByLens.topology.statusFilters;

    // "completed" exists in backlog, execution, and initiative
    expect(filters.backlog?.completed).toBe(false);
    expect(filters.execution?.completed).toBe(false);
    expect(filters.initiative?.completed).toBe(false);

    // "failed" exists in backlog, execution, capture, agent-activity, agent-run
    expect(filters.backlog?.failed).toBe(false);
    expect(filters.execution?.failed).toBe(false);
    expect(filters.capture?.failed).toBe(false);
    expect(filters["agent-activity"]?.failed).toBe(false);
    expect(filters["agent-run"]?.failed).toBe(false);

    // Scenario shouldn't have "completed" since it's not in its status list
    expect(filters.scenario?.completed).toBeUndefined();
  });

  it("loads v4 grouped status filters directly", () => {
    window.localStorage.setItem(
      "swarm-manager.graph.settings.v4",
      JSON.stringify({
        topology: {
          entityFilters: {},
          statusFilters: {
            backlog: { completed: false },
            execution: { running: false },
          },
          groupingMode: "none",
          showSecondaryEdges: true,
          autoFitOnChange: true,
        },
      }),
    );

    const initialState = createGraphDataInitialState();
    const filters = initialState.settingsByLens.topology.statusFilters;
    expect(filters.backlog?.completed).toBe(false);
    expect(filters.execution?.running).toBe(false);
    // Not cross-contaminated
    expect(filters.backlog?.running).toBeUndefined();
  });

  it("sets graph data atomically", () => {
    const nodes = [makeNode("scenario/test")];
    const edges = [makeEdge("a", "b")];

    useGraphDataStore.getState().setGraphData(nodes, edges, {
      lens: "topology",
      nodeCount: 1,
      edgeCount: 1,
      generatedAt: "2026-03-28T00:00:00Z",
      agentManagerAvailable: null,
    });

    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual(nodes);
    expect(state.edges).toEqual(edges);
    expect(state.meta?.nodeCount).toBe(1);
    expect(state.graphsByLens.topology.nodes).toEqual(nodes);
  });

  it("applies entity filters per lens and persists them", () => {
    useGraphDataStore.getState().setEntityFilter("capture", false);

    const { settingsByLens } = useGraphDataStore.getState();
    expect(settingsByLens.topology.entityFilters.capture).toBe(false);
    expect(settingsByLens.flow.entityFilters.capture).toBe(true);

    const persisted = window.localStorage.getItem("swarm-manager.graph.settings.v4");
    expect(persisted).toContain("\"capture\":false");
  });

  it("tracks status visibility per entity type for the active lens", () => {
    useGraphDataStore.getState().setStatusVisibility("execution", "running", false);

    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.execution?.running).toBe(false);

    useGraphDataStore.getState().clearStatusFilter("execution", "running");
    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.execution?.running).toBeUndefined();
  });

  it("keeps status filters independent across entity types", () => {
    useGraphDataStore.getState().setStatusVisibility("backlog", "completed", false);
    useGraphDataStore.getState().setStatusVisibility("execution", "completed", true);

    const filters = useGraphDataStore.getState().settingsByLens.topology.statusFilters;
    expect(filters.backlog?.completed).toBe(false);
    expect(filters.execution?.completed).toBe(true);
  });

  it("sets all statuses for an entity type via group visibility", () => {
    const statuses = ["running", "stopped", "error", "unknown"];
    useGraphDataStore.getState().setEntityStatusGroupVisibility("scenario", statuses, false);

    const group = useGraphDataStore.getState().settingsByLens.topology.statusFilters.scenario;
    expect(group).toBeDefined();
    for (const status of statuses) {
      expect(group![status]).toBe(false);
    }
  });

  it("cleans up empty entity group when last status filter is cleared", () => {
    useGraphDataStore.getState().setStatusVisibility("capture", "failed", false);
    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.capture).toBeDefined();

    useGraphDataStore.getState().clearStatusFilter("capture", "failed");
    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.capture).toBeUndefined();
  });

  it("resets only the current lens settings", () => {
    const store = useGraphDataStore.getState();
    store.setEntityFilter("capture", false);
    store.setGroupingMode("none");
    store.setStatusVisibility("execution", "running", false);
    store.setLens("flow");
    store.setEntityFilter("execution", false);

    useGraphDataStore.getState().setLens("topology");
    useGraphDataStore.getState().resetLensSettings();

    const state = useGraphDataStore.getState();
    expect(state.settingsByLens.topology.entityFilters.capture).toBe(true);
    expect(state.settingsByLens.topology.groupingMode).toBe("initiative");
    expect(state.settingsByLens.topology.statusFilters).toEqual({});
    expect(state.settingsByLens.flow.entityFilters.execution).toBe(false);
  });

  it("fetches graph data through the graph service", async () => {
    getGraphMock.mockResolvedValue({
      nodes: [makeNode("scenario/swarm-manager", "scenario")],
      edges: [],
      meta: {
        lens: "topology",
        nodeCount: 1,
        edgeCount: 0,
        generatedAt: "2026-03-28T00:00:00Z",
        agentManagerAvailable: null,
      },
    });

    await useGraphDataStore.getState().fetchGraph("topology");

    const state = useGraphDataStore.getState();
    const firstCall = getGraphMock.mock.calls[0];
    expect(firstCall).toBeDefined();
    if (!firstCall) {
      throw new Error("Expected graph service call");
    }
    expect(firstCall[0]).toBe("topology");
    expect(firstCall[1]?.signal).toBeInstanceOf(AbortSignal);
    expect(state.nodes).toHaveLength(1);
    expect(state.meta?.generatedAt).toBe("2026-03-28T00:00:00Z");
    expect(state.loading).toBe(false);
    expect(state.error).toBeNull();
  });

  it("reuses a fresh per-lens snapshot instead of refetching", async () => {
    getGraphMock
      .mockResolvedValueOnce({
        nodes: [makeNode("scenario/swarm-manager", "scenario")],
        edges: [],
        meta: {
          lens: "topology",
          nodeCount: 1,
          edgeCount: 0,
          generatedAt: "2026-03-28T00:00:00Z",
          agentManagerAvailable: null,
        },
      })
      .mockResolvedValueOnce({
        nodes: [makeNode("flow/item", "backlog")],
        edges: [],
        meta: {
          lens: "flow",
          nodeCount: 1,
          edgeCount: 0,
          generatedAt: "2026-03-28T00:01:00Z",
          agentManagerAvailable: null,
        },
      });

    await useGraphDataStore.getState().fetchGraph("topology");
    useGraphDataStore.getState().setLens("flow");
    await useGraphDataStore.getState().fetchGraph("flow");
    useGraphDataStore.getState().setLens("topology");

    await useGraphDataStore.getState().fetchGraph("topology");

    expect(getGraphMock).toHaveBeenCalledTimes(2);
    expect(useGraphDataStore.getState().nodes[0]?.id).toBe("scenario/swarm-manager");
  });

  it("dedupes concurrent fetches for the same lens", async () => {
    const pending = deferred<{
      nodes: GraphNode[];
      edges: GraphEdge[];
      meta: {
        lens: "topology";
        nodeCount: number;
        edgeCount: number;
        generatedAt: string;
        agentManagerAvailable: null;
      };
    }>();
    getGraphMock.mockReturnValue(pending.promise);

    const first = useGraphDataStore.getState().fetchGraph("topology");
    const second = useGraphDataStore.getState().fetchGraph("topology");

    expect(getGraphMock).toHaveBeenCalledTimes(1);

    pending.resolve({
      nodes: [makeNode("scenario/swarm-manager", "scenario")],
      edges: [],
      meta: {
        lens: "topology",
        nodeCount: 1,
        edgeCount: 0,
        generatedAt: "2026-03-28T00:00:00Z",
        agentManagerAvailable: null,
      },
    });

    await Promise.all([first, second]);
  });

  it("ignores aborted requests and keeps the active graph stable", async () => {
    const pending = deferred<{
      nodes: GraphNode[];
      edges: GraphEdge[];
      meta: {
        lens: "topology";
        nodeCount: number;
        edgeCount: number;
        generatedAt: string;
        agentManagerAvailable: null;
      };
    }>();
    getGraphMock
      .mockReturnValueOnce(pending.promise)
      .mockResolvedValueOnce({
        nodes: [makeNode("scenario/new", "scenario")],
        edges: [],
        meta: {
          lens: "topology",
          nodeCount: 1,
          edgeCount: 0,
          generatedAt: "2026-03-28T00:00:01Z",
          agentManagerAvailable: null,
        },
      });

    const first = useGraphDataStore.getState().fetchGraph("topology");
    const second = useGraphDataStore.getState().fetchGraph("topology", { force: true });

    pending.reject(new DOMException("aborted", "AbortError"));

    await Promise.allSettled([first, second]);

    const state = useGraphDataStore.getState();
    expect(state.nodes[0]?.id).toBe("scenario/new");
    expect(state.error).toBeNull();
  });

  it("preserves pulsing node state across graph replacements", () => {
    useGraphDataStore.setState({
      ...cloneGraphDataInitialState(),
      nodes: [
        {
          ...makeRunNode("run/abc", { label: "Run abc", pulsing: true }),
        },
      ],
      graphsByLens: {
        topology: {
          nodes: [
            {
              ...makeRunNode("run/abc", { label: "Run abc", pulsing: true }),
            },
          ],
          edges: [],
          meta: null,
          loading: false,
          error: null,
          fetchedAtMs: null,
        },
        flow: {
          nodes: [],
          edges: [],
          meta: null,
          loading: false,
          error: null,
          fetchedAtMs: null,
        },
        operations: {
          nodes: [],
          edges: [],
          meta: null,
          loading: false,
          error: null,
          fetchedAtMs: null,
        },
      },
    });

    useGraphDataStore.getState().setGraphData([makeNode("run/abc", "agent-run")], []);

    const [node] = useGraphDataStore.getState().nodes;
    expect(node).toBeDefined();
    if (!node) {
      throw new Error("Expected runtime node");
    }
    expect(node.data.pulsing).toBe(true);
  });

  it("sets error on fetch failure", async () => {
    getGraphMock.mockRejectedValueOnce(new Error("Network failure"));

    await useGraphDataStore.getState().fetchGraph("topology");

    const state = useGraphDataStore.getState();
    expect(state.loading).toBe(false);
    expect(state.error).toBe("Network failure");
    expect(state.nodes).toHaveLength(0);
  });

  it("clears error on successful fetch after failure", async () => {
    getGraphMock
      .mockRejectedValueOnce(new Error("First fail"))
      .mockResolvedValueOnce({
        nodes: [makeNode("scenario/test", "scenario")],
        edges: [],
        meta: {
          lens: "topology",
          nodeCount: 1,
          edgeCount: 0,
          generatedAt: "2026-03-28T00:00:00Z",
          agentManagerAvailable: null,
        },
      });

    await useGraphDataStore.getState().fetchGraph("topology");
    expect(useGraphDataStore.getState().error).toBe("First fail");

    // Force refetch
    await useGraphDataStore.getState().fetchGraph("topology", { force: true });
    expect(useGraphDataStore.getState().error).toBeNull();
    expect(useGraphDataStore.getState().nodes).toHaveLength(1);
  });

  it("isolates graph data between lenses", async () => {
    getGraphMock
      .mockResolvedValueOnce({
        nodes: [makeNode("scenario/topo", "scenario")],
        edges: [],
        meta: { lens: "topology", nodeCount: 1, edgeCount: 0, generatedAt: "t1", agentManagerAvailable: null },
      })
      .mockResolvedValueOnce({
        nodes: [makeNode("execution/flow", "execution"), makeNode("backlog-item/execute/b", "backlog")],
        edges: [],
        meta: { lens: "flow", nodeCount: 2, edgeCount: 0, generatedAt: "t2", agentManagerAvailable: null },
      });

    await useGraphDataStore.getState().fetchGraph("topology");
    expect(useGraphDataStore.getState().nodes).toHaveLength(1);

    useGraphDataStore.getState().setLens("flow");
    await useGraphDataStore.getState().fetchGraph("flow");
    expect(useGraphDataStore.getState().nodes).toHaveLength(2);

    // Switch back — topology snapshot should be preserved
    useGraphDataStore.getState().setLens("topology");
    expect(useGraphDataStore.getState().nodes).toHaveLength(1);
    expect(useGraphDataStore.getState().nodes[0]?.id).toBe("scenario/topo");
  });

  it("persists and restores settings including initiative status filters", () => {
    useGraphDataStore.getState().setStatusVisibility("initiative", "archived", false);

    const persisted = window.localStorage.getItem("swarm-manager.graph.settings.v4");
    expect(persisted).toBeTruthy();

    // Create new state from localStorage (simulating page reload)
    const freshState = createGraphDataInitialState();
    expect(freshState.settingsByLens.topology.statusFilters.initiative?.archived).toBe(false);
  });

  it("handles silent fetch without showing loading state", async () => {
    getGraphMock.mockResolvedValueOnce({
      nodes: [makeNode("scenario/test", "scenario")],
      edges: [],
      meta: { lens: "topology", nodeCount: 1, edgeCount: 0, generatedAt: "t1", agentManagerAvailable: null },
    });

    // Initial fetch
    await useGraphDataStore.getState().fetchGraph("topology");

    // Force silent refetch
    getGraphMock.mockResolvedValueOnce({
      nodes: [makeNode("scenario/test", "scenario"), makeNode("scenario/new", "scenario")],
      edges: [],
      meta: { lens: "topology", nodeCount: 2, edgeCount: 0, generatedAt: "t2", agentManagerAvailable: null },
    });

    const fetchPromise = useGraphDataStore.getState().fetchGraph("topology", { force: true, silent: true });
    // During silent fetch, loading should remain false
    expect(useGraphDataStore.getState().loading).toBe(false);

    await fetchPromise;
    expect(useGraphDataStore.getState().nodes).toHaveLength(2);
  });
});
