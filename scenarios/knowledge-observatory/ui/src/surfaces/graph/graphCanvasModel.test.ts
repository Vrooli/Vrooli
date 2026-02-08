import { describe, expect, it } from "vitest";
import type { GraphResponse } from "../../shared/services/api";
import {
  buildCanvasRenderModel,
  fitViewportToRenderModel,
  mergeCanvasGraphs,
  toCanvasGraph,
} from "./graphCanvasModel";

function makeResponse(center: string): GraphResponse {
  return {
    center,
    took_ms: 18,
    nodes: [
      { id: "center", label: center, score: 1, metadata: { type: "center" } },
      { id: "n1", label: "Node 1", score: 0.91, metadata: { namespace: "alpha" } },
      { id: "n2", label: "Node 2", score: 0.88, metadata: { namespace: "alpha" } },
    ],
    edges: [
      { source: "center", target: "n1", weight: 0.91, relationship: "semantic_similarity" },
      { source: "n1", target: "n2", weight: 0.75, relationship: "semantic_similarity" },
    ],
  };
}

describe("graphCanvasModel", () => {
  it("normalizes API graph response for canvas rendering", () => {
    const graph = toCanvasGraph(makeResponse("semantic drift"));

    expect(graph.centerNodeID).toBe("center:semantic drift");
    expect(graph.nodes.some((node) => node.id === "center:semantic drift")).toBe(true);
    expect(graph.edges[0]?.source).toBe("center:semantic drift");
  });

  it("builds render model with neighbor highlighting and weight filter", () => {
    const graph = toCanvasGraph(makeResponse("semantic drift"));
    const model = buildCanvasRenderModel({
      graph,
      layoutMode: "radial",
      minWeight: 0.8,
      selectedNodeID: "n1",
      highlightNeighbors: true,
      maxNodes: 20,
    });

    expect(model.edges).toHaveLength(1);
    expect(model.nodes.some((node) => node.id === "n1" && node.isSelected)).toBe(true);
  });

  it("merges expanded graph data by anchoring expansion center to selected node", () => {
    const base = toCanvasGraph(makeResponse("semantic drift"));
    const incoming = toCanvasGraph(makeResponse("Node 1"));

    const merged = mergeCanvasGraphs({
      base,
      incoming,
      anchorNodeID: "n1",
    });

    expect(merged.nodes.some((node) => node.id === "n1")).toBe(true);
    expect(merged.edges.every((edge) => edge.source !== "center:node 1" && edge.target !== "center:node 1")).toBe(
      true
    );
  });

  it("computes a finite fit viewport for non-empty models", () => {
    const graph = toCanvasGraph(makeResponse("semantic drift"));
    const model = buildCanvasRenderModel({
      graph,
      layoutMode: "force",
      minWeight: 0,
      selectedNodeID: undefined,
      highlightNeighbors: false,
      maxNodes: 20,
    });

    const viewport = fitViewportToRenderModel(model);

    expect(Number.isFinite(viewport.scale)).toBe(true);
    expect(Number.isFinite(viewport.offsetX)).toBe(true);
    expect(Number.isFinite(viewport.offsetY)).toBe(true);
    expect(viewport.scale).toBeGreaterThan(0);
  });
});
