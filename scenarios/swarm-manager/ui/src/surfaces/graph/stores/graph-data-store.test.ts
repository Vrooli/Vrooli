import { describe, it, expect, beforeEach } from "vitest";
import { useGraphDataStore, graphDataInitialState } from "./graph-data-store";
import type { Node, Edge } from "@xyflow/react";

function resetStore() {
  useGraphDataStore.setState({ ...graphDataInitialState, entityFilters: { ...graphDataInitialState.entityFilters } });
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

  it("starts with empty nodes and edges", () => {
    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual([]);
    expect(state.edges).toEqual([]);
  });

  it("sets nodes", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    useGraphDataStore.getState().setNodes(nodes);
    expect(useGraphDataStore.getState().nodes).toEqual(nodes);
  });

  it("sets edges", () => {
    const edges = [makeEdge("a", "b")];
    useGraphDataStore.getState().setEdges(edges);
    expect(useGraphDataStore.getState().edges).toEqual(edges);
  });

  it("sets graph data (nodes + edges atomically)", () => {
    const nodes = [makeNode("a")];
    const edges = [makeEdge("a", "b")];
    useGraphDataStore.getState().setGraphData(nodes, edges);
    const state = useGraphDataStore.getState();
    expect(state.nodes).toEqual(nodes);
    expect(state.edges).toEqual(edges);
  });

  it("defaults to topology lens", () => {
    expect(useGraphDataStore.getState().lens).toBe("topology");
  });

  it("switches lens", () => {
    useGraphDataStore.getState().setLens("flow");
    expect(useGraphDataStore.getState().lens).toBe("flow");
    useGraphDataStore.getState().setLens("operations");
    expect(useGraphDataStore.getState().lens).toBe("operations");
  });

  it("has all entity filters enabled by default", () => {
    const filters = useGraphDataStore.getState().entityFilters;
    expect(filters.backlog).toBe(true);
    expect(filters.scenario).toBe(true);
    expect(filters.execution).toBe(true);
    expect(filters.capture).toBe(true);
    expect(filters["agent-run"]).toBe(true);
    expect(filters.initiative).toBe(true);
  });

  it("toggles entity filter", () => {
    useGraphDataStore.getState().toggleEntityFilter("capture");
    expect(useGraphDataStore.getState().entityFilters.capture).toBe(false);
    useGraphDataStore.getState().toggleEntityFilter("capture");
    expect(useGraphDataStore.getState().entityFilters.capture).toBe(true);
  });

  it("sets entity filter explicitly", () => {
    useGraphDataStore.getState().setEntityFilter("backlog", false);
    expect(useGraphDataStore.getState().entityFilters.backlog).toBe(false);
  });

  it("resets filters to defaults", () => {
    useGraphDataStore.getState().toggleEntityFilter("backlog");
    useGraphDataStore.getState().toggleEntityFilter("scenario");
    useGraphDataStore.getState().resetFilters();
    const filters = useGraphDataStore.getState().entityFilters;
    expect(filters.backlog).toBe(true);
    expect(filters.scenario).toBe(true);
  });
});
