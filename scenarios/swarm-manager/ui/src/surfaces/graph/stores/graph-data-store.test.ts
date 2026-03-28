import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Edge, Node } from "@xyflow/react";
import type { GraphProjection, GraphRequestOptions } from "../../../services/graph-service";

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

const makeNode = (id: string, type?: string): Node => ({
  id,
  type,
  position: { x: 0, y: 0 },
  data: { label: id, entityType: type },
});

const makeEdge = (source: string, target: string): Edge => ({
  id: `${source}->${target}`,
  source,
  target,
});

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
    expect(state.settingsByLens.topology.groupingMode).toBe("none");
    expect(state.settingsByLens.flow.groupingMode).toBe("none");
  });

  it("migrates legacy grouped topology settings to the flat default", () => {
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
    expect(initialState.settingsByLens.topology.groupingMode).toBe("none");
    expect(initialState.settingsByLens.topology.entityFilters.capture).toBe(false);
    expect(initialState.settingsByLens.topology.showSecondaryEdges).toBe(false);
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

    const persisted = window.localStorage.getItem("swarm-manager.graph.settings.v3");
    expect(persisted).toContain("\"capture\":false");
  });

  it("tracks status visibility for the active lens", () => {
    useGraphDataStore.getState().setStatusVisibility("running", false);

    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.running).toBe(false);

    useGraphDataStore.getState().clearStatusFilter("running");
    expect(useGraphDataStore.getState().settingsByLens.topology.statusFilters.running).toBeUndefined();
  });

  it("resets only the current lens settings", () => {
    const store = useGraphDataStore.getState();
    store.setEntityFilter("capture", false);
    store.setGroupingMode("none");
    store.setStatusVisibility("running", false);
    store.setLens("flow");
    store.setEntityFilter("execution", false);

    useGraphDataStore.getState().setLens("topology");
    useGraphDataStore.getState().resetLensSettings();

    const state = useGraphDataStore.getState();
    expect(state.settingsByLens.topology.entityFilters.capture).toBe(true);
    expect(state.settingsByLens.topology.groupingMode).toBe("none");
    expect(state.settingsByLens.topology.statusFilters.running).toBeUndefined();
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
      nodes: Node[];
      edges: Edge[];
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
      nodes: Node[];
      edges: Edge[];
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
          ...makeNode("run/abc", "agent-run"),
          data: { label: "Run abc", entityType: "agent-run", pulsing: true },
        },
      ],
      graphsByLens: {
        topology: {
          nodes: [
            {
              ...makeNode("run/abc", "agent-run"),
              data: { label: "Run abc", entityType: "agent-run", pulsing: true },
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
    expect((node.data as Record<string, unknown>).pulsing).toBe(true);
  });
});
