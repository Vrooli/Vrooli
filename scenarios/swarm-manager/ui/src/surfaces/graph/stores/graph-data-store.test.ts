import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Edge, Node } from "@xyflow/react";

const { getGraphMock } = vi.hoisted(() => ({
  getGraphMock: vi.fn(),
}));

vi.mock("../../../services", () => ({
  graphService: {
    getGraph: getGraphMock,
  },
}));

import {
  cloneGraphDataInitialState,
  useGraphDataStore,
} from "./graph-data-store";

function resetStore() {
  useGraphDataStore.setState(cloneGraphDataInitialState());
  window.localStorage.clear();
  getGraphMock.mockReset();
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
  });

  it("defaults to the topology lens with initiative grouping", () => {
    const state = useGraphDataStore.getState();
    expect(state.lens).toBe("topology");
    expect(state.settingsByLens.topology.groupingMode).toBe("initiative");
    expect(state.settingsByLens.flow.groupingMode).toBe("none");
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
  });

  it("applies entity filters per lens and persists them", () => {
    useGraphDataStore.getState().setEntityFilter("capture", false);

    const { settingsByLens } = useGraphDataStore.getState();
    expect(settingsByLens.topology.entityFilters.capture).toBe(false);
    expect(settingsByLens.flow.entityFilters.capture).toBe(true);

    const persisted = window.localStorage.getItem("swarm-manager.graph.settings.v2");
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
    expect(state.settingsByLens.topology.groupingMode).toBe("initiative");
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
    expect(getGraphMock).toHaveBeenCalledWith("topology");
    expect(state.nodes).toHaveLength(1);
    expect(state.meta?.generatedAt).toBe("2026-03-28T00:00:00Z");
    expect(state.loading).toBe(false);
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
    });

    useGraphDataStore.getState().setGraphData([makeNode("run/abc", "agent-run")], []);

    const [node] = useGraphDataStore.getState().nodes;
    expect((node.data as Record<string, unknown>).pulsing).toBe(true);
  });
});
