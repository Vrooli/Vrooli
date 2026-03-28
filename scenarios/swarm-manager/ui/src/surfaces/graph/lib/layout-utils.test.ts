import { describe, it, expect } from "vitest";
import type { Node, Edge } from "@xyflow/react";
import { applyDagreLayout, getDagreConfig } from "./layout-utils";

const makeNode = (id: string): Node => ({
  id,
  position: { x: 0, y: 0 },
  data: {},
});

const makeEdge = (source: string, target: string): Edge => ({
  id: `${source}->${target}`,
  source,
  target,
});

describe("getDagreConfig", () => {
  it("returns TB direction for hierarchical", () => {
    const config = getDagreConfig("hierarchical");
    expect(config.rankdir).toBe("TB");
    expect(config.ranker).toBe("network-simplex");
  });

  it("returns LR direction for compact", () => {
    const config = getDagreConfig("compact");
    expect(config.rankdir).toBe("LR");
    expect(config.ranker).toBe("tight-tree");
  });

  it("returns TB direction with larger spacing for grouped", () => {
    const config = getDagreConfig("grouped");
    expect(config.rankdir).toBe("TB");
    expect(config.nodesep).toBeGreaterThan(getDagreConfig("hierarchical").nodesep);
  });
});

describe("applyDagreLayout", () => {
  it("returns empty array for empty input", () => {
    expect(applyDagreLayout([], [], "hierarchical")).toEqual([]);
  });

  it("positions a single node", () => {
    const result = applyDagreLayout([makeNode("a")], [], "hierarchical");
    expect(result).toHaveLength(1);
    expect(typeof result[0]!.position.x).toBe("number");
    expect(typeof result[0]!.position.y).toBe("number");
  });

  it("positions multiple connected nodes at distinct locations", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c")];
    const result = applyDagreLayout(nodes, edges, "hierarchical");

    expect(result).toHaveLength(3);

    // All positions should be distinct.
    const positions = result.map((n) => `${n.position.x},${n.position.y}`);
    const unique = new Set(positions);
    expect(unique.size).toBe(3);
  });

  it("produces different layouts for different modes", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    const edges = [makeEdge("a", "b")];

    const hierarchical = applyDagreLayout(nodes, edges, "hierarchical");
    const compact = applyDagreLayout(nodes, edges, "compact");

    // Hierarchical is TB, compact is LR — positions should differ.
    const hierPos = hierarchical.map((n) => [n.position.x, n.position.y]);
    const compPos = compact.map((n) => [n.position.x, n.position.y]);
    expect(hierPos).not.toEqual(compPos);
  });

  it("handles disconnected components", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b")]; // c is disconnected
    const result = applyDagreLayout(nodes, edges, "hierarchical");
    expect(result).toHaveLength(3);
    // All should have valid positions.
    for (const node of result) {
      expect(Number.isFinite(node.position.x)).toBe(true);
      expect(Number.isFinite(node.position.y)).toBe(true);
    }
  });

  it("preserves node data and id", () => {
    const node = { ...makeNode("test"), data: { label: "Test Node" } };
    const result = applyDagreLayout([node], [], "hierarchical");
    expect(result[0]!.id).toBe("test");
    expect(result[0]!.data.label).toBe("Test Node");
  });
});
