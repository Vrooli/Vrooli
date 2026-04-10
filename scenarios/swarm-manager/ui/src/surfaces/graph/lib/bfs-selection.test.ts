import { describe, it, expect } from "vitest";
import type { Node, Edge } from "@xyflow/react";
import { bfsNeighborhood } from "./bfs-selection";

const makeNode = (id: string, type?: string): Node => ({
  id,
  type,
  position: { x: 0, y: 0 },
  data: {},
});

const makeEdge = (source: string, target: string): Edge => ({
  id: `${source}->${target}`,
  source,
  target,
});

describe("bfsNeighborhood", () => {
  it("returns only the start node when no edges exist", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    const result = bfsNeighborhood("a", nodes, []);
    expect(result).toEqual(new Set(["a"]));
  });

  it("returns empty set for unknown start node", () => {
    const result = bfsNeighborhood("unknown", [makeNode("a")], []);
    expect(result).toEqual(new Set());
  });

  it("finds direct neighbors at depth 1", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c"), makeNode("d")];
    const edges = [makeEdge("a", "b"), makeEdge("a", "c"), makeEdge("c", "d")];
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 1 });
    expect(result).toEqual(new Set(["a", "b", "c"]));
  });

  it("respects maxDepth > 1", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c"), makeNode("d")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c"), makeEdge("c", "d")];
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 2 });
    expect(result).toEqual(new Set(["a", "b", "c"]));
  });

  it("traverses full chain at sufficient depth", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c")];
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 3 });
    expect(result).toEqual(new Set(["a", "b", "c"]));
  });

  it("handles cycles gracefully", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c"), makeEdge("c", "a")];
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 10 });
    expect(result).toEqual(new Set(["a", "b", "c"]));
  });

  it("respects type constraints (allowedTypes)", () => {
    const nodes = [
      makeNode("a", "backlog"),
      makeNode("b", "scenario"),
      makeNode("c", "backlog"),
      makeNode("d", "execution"),
    ];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c"), makeEdge("a", "d")];

    const result = bfsNeighborhood("a", nodes, edges, {
      maxDepth: 2,
      allowedTypes: new Set(["backlog", "scenario"]),
    });

    expect(result.has("a")).toBe(true);
    expect(result.has("b")).toBe(true);
    expect(result.has("c")).toBe(true);
    expect(result.has("d")).toBe(false); // execution type excluded
  });

  it("handles disconnected graph (unreachable nodes)", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b")];
    // c is disconnected
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 10 });
    expect(result).toEqual(new Set(["a", "b"]));
  });

  it("handles single-node graph", () => {
    const result = bfsNeighborhood("a", [makeNode("a")], []);
    expect(result).toEqual(new Set(["a"]));
  });

  it("handles empty graph", () => {
    const result = bfsNeighborhood("a", [], []);
    expect(result).toEqual(new Set());
  });

  it("traverses edges bidirectionally", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    const edges = [makeEdge("b", "a")]; // edge points TO a
    const result = bfsNeighborhood("a", nodes, edges, { maxDepth: 1 });
    expect(result).toEqual(new Set(["a", "b"]));
  });
});
