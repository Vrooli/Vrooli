import { beforeEach, describe, expect, it, vi } from "vitest";
import type { GraphProjection, GraphProjectionMeta, GraphRequestOptions } from "../../../services/graph-service";
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

  it("defaults to the plan lens", () => {
    const state = useGraphDataStore.getState();
    expect(state.lens).toBe("plan");
  });

  it("sets graph data atomically", () => {
    const nodes = [makeNode("scenario/test")];
    const edges = [makeEdge("a", "b")];

    useGraphDataStore.getState().setLens("topology");
    useGraphDataStore.getState().setGraphData(nodes, edges, {
      lens: "topology",
      nodeCount: 1,
      edgeCount: 1,
      generatedAt: "2026-03-28T00:00:00Z",
      agentManagerAvailable: null, hint: null,
    });

    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual(nodes);
    expect(state.edges).toEqual(edges);
    expect(state.meta?.nodeCount).toBe(1);
    expect(state.graphsByLens.topology.nodes).toEqual(nodes);
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
        agentManagerAvailable: null, hint: null,
      },
    });

    useGraphDataStore.getState().setLens("topology");
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
    useGraphDataStore.getState().setLens("topology");
    getGraphMock.mockResolvedValueOnce({
      nodes: [makeNode("scenario/swarm-manager", "scenario")],
      edges: [],
      meta: {
        lens: "topology",
        nodeCount: 1,
        edgeCount: 0,
        generatedAt: "2026-03-28T00:00:00Z",
        agentManagerAvailable: null, hint: null,
      },
    });

    await useGraphDataStore.getState().fetchGraph("topology");
    useGraphDataStore.getState().setLens("topology");
    await useGraphDataStore.getState().fetchGraph("topology");

    expect(getGraphMock).toHaveBeenCalledTimes(1);
    expect(useGraphDataStore.getState().nodes[0]?.id).toBe("scenario/swarm-manager");
  });

  it("dedupes concurrent fetches for the same lens", async () => {
    const pending = deferred<{
      nodes: GraphNode[];
      edges: GraphEdge[];
      meta: GraphProjectionMeta;
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
        agentManagerAvailable: null, hint: null,
      },
    });

    await Promise.all([first, second]);
  });

  it("ignores aborted requests and keeps the active graph stable", async () => {
    const pending = deferred<{
      nodes: GraphNode[];
      edges: GraphEdge[];
      meta: GraphProjectionMeta;
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
          agentManagerAvailable: null, hint: null,
        },
      });

    useGraphDataStore.getState().setLens("topology");
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
      lens: "topology",
      nodes: [
        {
          ...makeRunNode("run/abc", { label: "Run abc", pulsing: true }),
        },
      ],
      graphsByLens: {
        plan: {
          nodes: [],
          edges: [],
          meta: null,
          loading: false,
          error: null,
          fetchedAtMs: null,
        },
        focus: {
          nodes: [],
          edges: [],
          meta: null,
          loading: false,
          error: null,
          fetchedAtMs: null,
        },
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
    useGraphDataStore.getState().setLens("topology");
    getGraphMock.mockRejectedValueOnce(new Error("Network failure"));

    await useGraphDataStore.getState().fetchGraph("topology");

    const state = useGraphDataStore.getState();
    expect(state.loading).toBe(false);
    expect(state.error).toBe("Network failure");
    expect(state.nodes).toHaveLength(0);
  });

  it("clears error on successful fetch after failure", async () => {
    useGraphDataStore.getState().setLens("topology");
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
          agentManagerAvailable: null, hint: null,
        },
      });

    await useGraphDataStore.getState().fetchGraph("topology");
    expect(useGraphDataStore.getState().error).toBe("First fail");

    // Force refetch
    await useGraphDataStore.getState().fetchGraph("topology", { force: true });
    expect(useGraphDataStore.getState().error).toBeNull();
    expect(useGraphDataStore.getState().nodes).toHaveLength(1);
  });


  it("includes initiatives and scenarios in focus lens when they link to attention-worthy items", async () => {
    // Regression: initiative and scenario nodes have no attention state of
    // their own, so filtering by computeNodeAttention used to drop all of
    // them from the focus lens. They should be pulled in whenever they
    // connect to an attention-worthy item via member_of / targets edges.
    const backlogNode = makeGraphNode("backlog-item/fix/failing-task", "backlog", {
      label: "failing-task",
      kind: "fix",
      name: "failing-task",
      title: "Failing Task",
      status: "failed", // failed status → needsAttention
      priority: 1,
    });
    const initiativeNode = makeGraphNode("initiative/my-init", "initiative", {
      label: "My Initiative",
      name: "my-init",
      title: "My Initiative",
      status: "active",
      rollup: { total: 1, completed: 0, in_progress: 0, failed: 1, pending: 0 },
    });
    const orphanInitiative = makeGraphNode("initiative/other-init", "initiative", {
      label: "Other Initiative",
      name: "other-init",
      title: "Other Initiative",
      status: "active",
      rollup: { total: 0, completed: 0, in_progress: 0, failed: 0, pending: 0 },
    });
    const scenarioNode = makeGraphNode("scenario/target-app", "scenario", {
      label: "target-app",
      name: "target-app",
      status: "running",
    });
    const orphanScenario = makeGraphNode("scenario/unrelated", "scenario", {
      label: "unrelated",
      name: "unrelated",
      status: "running",
    });
    const memberEdge = makeGraphEdge(
      "member_of:fix/failing-task->my-init",
      "backlog-item/fix/failing-task",
      "initiative/my-init",
      "member_of",
    );
    const targetsEdge = makeGraphEdge(
      "targets:fix/failing-task->target-app",
      "backlog-item/fix/failing-task",
      "scenario/target-app",
      "targets",
    );

    getGraphMock.mockResolvedValueOnce({
      nodes: [backlogNode, initiativeNode, orphanInitiative, scenarioNode, orphanScenario],
      edges: [memberEdge, targetsEdge],
      meta: {
        lens: "topology",
        nodeCount: 5,
        edgeCount: 2,
        generatedAt: "t1",
        agentManagerAvailable: null,
        hint: null,
      },
    });

    await useGraphDataStore.getState().fetchGraph("focus");

    const focusSnapshot = useGraphDataStore.getState().graphsByLens.focus;
    const ids = focusSnapshot.nodes.map((n) => n.id);
    expect(ids).toContain("backlog-item/fix/failing-task");
    expect(ids).toContain("initiative/my-init");
    expect(ids).toContain("scenario/target-app");
    // Orphans — no attention-worthy neighbor — should stay out.
    expect(ids).not.toContain("initiative/other-init");
    expect(ids).not.toContain("scenario/unrelated");
    // Context edges should be preserved now that both endpoints are visible.
    const edgeIds = focusSnapshot.edges.map((e) => e.id);
    expect(edgeIds).toContain("member_of:fix/failing-task->my-init");
    expect(edgeIds).toContain("targets:fix/failing-task->target-app");
  });

  it("handles silent fetch without showing loading state", async () => {
    useGraphDataStore.getState().setLens("topology");
    getGraphMock.mockResolvedValueOnce({
      nodes: [makeNode("scenario/test", "scenario")],
      edges: [],
      meta: { lens: "topology", nodeCount: 1, edgeCount: 0, generatedAt: "t1", agentManagerAvailable: null, hint: null },
    });

    // Initial fetch
    await useGraphDataStore.getState().fetchGraph("topology");

    // Force silent refetch
    getGraphMock.mockResolvedValueOnce({
      nodes: [makeNode("scenario/test", "scenario"), makeNode("scenario/new", "scenario")],
      edges: [],
      meta: { lens: "topology", nodeCount: 2, edgeCount: 0, generatedAt: "t2", agentManagerAvailable: null, hint: null },
    });

    const fetchPromise = useGraphDataStore.getState().fetchGraph("topology", { force: true, silent: true });
    // During silent fetch, loading should remain false
    expect(useGraphDataStore.getState().loading).toBe(false);

    await fetchPromise;
    expect(useGraphDataStore.getState().nodes).toHaveLength(2);
  });
});
