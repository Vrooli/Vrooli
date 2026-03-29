import { describe, it, expect } from "vitest";
import type { Node, Edge } from "@xyflow/react";
import { applyDagreLayout, applyGroupedLayout, getDagreConfig } from "./layout-utils";
import type { GraphEntityType } from "../types";

const makeNode = (id: string): Node => ({
  id,
  position: { x: 0, y: 0 },
  data: {},
});

const makeTypedNode = (id: string, entityType: GraphEntityType): Node => ({
  id,
  position: { x: 0, y: 0 },
  data: { entityType },
});

const makeEdge = (source: string, target: string): Edge => ({
  id: `${source}->${target}`,
  source,
  target,
});

describe("getDagreConfig", () => {
  it("returns TB direction for hierarchical", () => {
    const config = getDagreConfig("hierarchical", "TB");
    expect(config.rankdir).toBe("TB");
    expect(config.ranker).toBe("network-simplex");
  });

  it("uses the provided direction for compact layouts", () => {
    const config = getDagreConfig("compact", "LR");
    expect(config.rankdir).toBe("LR");
    expect(config.ranker).toBe("tight-tree");
  });
});

describe("applyDagreLayout", () => {
  it("returns empty array for empty input", () => {
    expect(applyDagreLayout([], [], "hierarchical", "TB")).toEqual([]);
  });

  it("positions a single node", () => {
    const result = applyDagreLayout([makeNode("a")], [], "hierarchical", "TB");
    expect(result).toHaveLength(1);
    const firstNode = result[0];
    expect(firstNode).toBeDefined();
    expect(typeof firstNode?.position.x).toBe("number");
    expect(typeof firstNode?.position.y).toBe("number");
  });

  it("positions multiple connected nodes at distinct locations", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c")];
    const result = applyDagreLayout(nodes, edges, "hierarchical", "TB");

    expect(result).toHaveLength(3);

    // All positions should be distinct.
    const positions = result.map((n) => `${n.position.x},${n.position.y}`);
    const unique = new Set(positions);
    expect(unique.size).toBe(3);
  });

  it("produces different layouts for different modes", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    const edges = [makeEdge("a", "b")];

    const hierarchical = applyDagreLayout(nodes, edges, "hierarchical", "TB");
    const compact = applyDagreLayout(nodes, edges, "compact", "LR");

    // Hierarchical is TB, compact is LR — positions should differ.
    const hierPos = hierarchical.map((n) => [n.position.x, n.position.y]);
    const compPos = compact.map((n) => [n.position.x, n.position.y]);
    expect(hierPos).not.toEqual(compPos);
  });

  it("handles disconnected components", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b")]; // c is disconnected
    const result = applyDagreLayout(nodes, edges, "hierarchical", "TB");
    expect(result).toHaveLength(3);
    // All should have valid positions.
    for (const node of result) {
      expect(Number.isFinite(node.position.x)).toBe(true);
      expect(Number.isFinite(node.position.y)).toBe(true);
    }
  });

  it("preserves node data and id", () => {
    const node = { ...makeNode("test"), data: { label: "Test Node" } };
    const result = applyDagreLayout([node], [], "hierarchical", "TB");
    const firstNode = result[0];
    expect(firstNode).toBeDefined();
    expect(firstNode?.id).toBe("test");
    expect(firstNode?.data.label).toBe("Test Node");
  });

  it("respects TB direction — y increases along dependency chain", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c")];
    const result = applyDagreLayout(nodes, edges, "hierarchical", "TB");

    const posOf = (id: string) => result.find((n) => n.id === id)!.position;
    // In TB, a→b→c should have increasing y
    expect(posOf("a").y).toBeLessThan(posOf("b").y);
    expect(posOf("b").y).toBeLessThan(posOf("c").y);
  });

  it("respects LR direction — x increases along dependency chain", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c")];
    const edges = [makeEdge("a", "b"), makeEdge("b", "c")];
    const result = applyDagreLayout(nodes, edges, "hierarchical", "LR");

    const posOf = (id: string) => result.find((n) => n.id === id)!.position;
    expect(posOf("a").x).toBeLessThan(posOf("b").x);
    expect(posOf("b").x).toBeLessThan(posOf("c").x);
  });

  it("produces different spacing for hierarchical vs compact mode", () => {
    const nodes = [makeNode("a"), makeNode("b")];
    const edges = [makeEdge("a", "b")];

    const hier = applyDagreLayout(nodes, edges, "hierarchical", "TB");
    const comp = applyDagreLayout(nodes, edges, "compact", "TB");

    const hierGap = Math.abs(hier[0]!.position.y - hier[1]!.position.y);
    const compGap = Math.abs(comp[0]!.position.y - comp[1]!.position.y);
    // Hierarchical has ranksep=80, compact has ranksep=60 → hierarchical gap is larger
    expect(hierGap).toBeGreaterThan(compGap);
  });

  it("lays out many disconnected nodes without errors", () => {
    const nodes = Array.from({ length: 50 }, (_, i) => makeNode(`n${i}`));
    const result = applyDagreLayout(nodes, [], "hierarchical", "TB");
    expect(result).toHaveLength(50);
    // All should have finite positions
    for (const node of result) {
      expect(Number.isFinite(node.position.x)).toBe(true);
      expect(Number.isFinite(node.position.y)).toBe(true);
    }
  });

  it("separates branching graph into multiple columns", () => {
    // Diamond: a → b, a → c, b → d, c → d
    const nodes = [makeNode("a"), makeNode("b"), makeNode("c"), makeNode("d")];
    const edges = [
      makeEdge("a", "b"),
      makeEdge("a", "c"),
      makeEdge("b", "d"),
      makeEdge("c", "d"),
    ];
    const result = applyDagreLayout(nodes, edges, "hierarchical", "TB");

    const posOf = (id: string) => result.find((n) => n.id === id)!.position;
    // b and c should be at the same rank (similar y) but different x
    expect(Math.abs(posOf("b").y - posOf("c").y)).toBeLessThan(10);
    expect(posOf("b").x).not.toEqual(posOf("c").x);
  });

  it("places isolated nodes in a grid below connected nodes", () => {
    const nodes = [makeNode("a"), makeNode("b"), makeNode("iso1"), makeNode("iso2"), makeNode("iso3")];
    const edges = [makeEdge("a", "b")]; // iso1, iso2, iso3 are disconnected
    const result = applyDagreLayout(nodes, edges, "hierarchical", "TB");

    const posOf = (id: string) => result.find((n) => n.id === id)!.position;
    const connectedMaxY = Math.max(posOf("a").y, posOf("b").y);

    // All isolated nodes should be below the connected subgraph
    expect(posOf("iso1").y).toBeGreaterThan(connectedMaxY);
    expect(posOf("iso2").y).toBeGreaterThan(connectedMaxY);
    expect(posOf("iso3").y).toBeGreaterThan(connectedMaxY);
  });

  it("arranges all-isolated nodes in a grid (not a single line)", () => {
    // 9 nodes with no edges → should be 3×3 grid, not 1×9 line
    const nodes = Array.from({ length: 9 }, (_, i) => makeNode(`n${i}`));
    const result = applyDagreLayout(nodes, [], "hierarchical", "TB");

    const xs = new Set(result.map((n) => n.position.x));
    const ys = new Set(result.map((n) => n.position.y));
    // With 9 nodes, ceil(sqrt(9))=3 columns, so 3 distinct x and 3 distinct y values
    expect(xs.size).toBe(3);
    expect(ys.size).toBe(3);
  });

  it("delegates to grouped layout when mode is grouped", () => {
    const nodes = [
      makeTypedNode("a", "backlog"),
      makeTypedNode("b", "scenario"),
    ];
    const edges = [makeEdge("a", "b")];
    const result = applyDagreLayout(nodes, edges, "grouped", "TB");

    // Grouped layout ignores edges and groups by entity type,
    // so backlog and scenario should be in different lanes (different y).
    const posA = result.find((n) => n.id === "a")!.position;
    const posB = result.find((n) => n.id === "b")!.position;
    expect(posA.y).not.toBe(posB.y);
  });
});

describe("applyGroupedLayout", () => {
  it("returns empty array for empty input", () => {
    expect(applyGroupedLayout([], "TB")).toEqual([]);
  });

  it("groups nodes by entity type into separate lanes (TB)", () => {
    const nodes = [
      makeTypedNode("b1", "backlog"),
      makeTypedNode("b2", "backlog"),
      makeTypedNode("s1", "scenario"),
      makeTypedNode("i1", "initiative"),
    ];
    const result = applyGroupedLayout(nodes, "TB");

    const posOf = (id: string) => result.find((n) => n.id === id)!.position;

    // Initiative comes before backlog in lane order, backlog before scenario.
    expect(posOf("i1").y).toBeLessThan(posOf("b1").y);
    expect(posOf("b1").y).toBeLessThan(posOf("s1").y);

    // Nodes in the same lane (b1, b2) should share the same y range.
    expect(Math.abs(posOf("b1").y - posOf("b2").y)).toBeLessThan(200);
  });

  it("swaps axes for LR direction", () => {
    const nodes = [
      makeTypedNode("i1", "initiative"),
      makeTypedNode("b1", "backlog"),
    ];
    const resultTB = applyGroupedLayout(nodes, "TB");
    const resultLR = applyGroupedLayout(nodes, "LR");

    const tbI = resultTB.find((n) => n.id === "i1")!.position;
    const lrI = resultLR.find((n) => n.id === "i1")!.position;

    // In TB, initiative is at top (small y). In LR, it should be at left (small x).
    // LR swaps x and y, so LR.x should equal TB.y and LR.y should equal TB.x.
    expect(lrI.x).toBe(tbI.y);
    expect(lrI.y).toBe(tbI.x);
  });

  it("skips empty lanes without adding extra gaps", () => {
    // Only backlog and execution — no initiative, scenario, etc.
    const nodes = [
      makeTypedNode("b1", "backlog"),
      makeTypedNode("e1", "execution"),
    ];
    const result = applyGroupedLayout(nodes, "TB");

    const posB = result.find((n) => n.id === "b1")!.position;
    const posE = result.find((n) => n.id === "e1")!.position;

    // They should be in consecutive lanes, not separated by empty lane gaps.
    // With cellY=132 and laneGap=120, one lane gap should be ~252 max.
    expect(posE.y - posB.y).toBeLessThan(300);
    expect(posE.y).toBeGreaterThan(posB.y);
  });

  it("arranges multiple nodes within a lane in a grid", () => {
    const nodes = Array.from({ length: 4 }, (_, i) =>
      makeTypedNode(`b${i}`, "backlog"),
    );
    const result = applyGroupedLayout(nodes, "TB");

    // 4 nodes → ceil(sqrt(4))=2 columns → should have 2 distinct x values.
    const xs = new Set(result.map((n) => n.position.x));
    expect(xs.size).toBe(2);
  });

  it("preserves node data and id", () => {
    const node: Node = {
      id: "test-1",
      position: { x: 0, y: 0 },
      data: { entityType: "backlog", label: "Test" },
    };
    const result = applyGroupedLayout([node], "TB");
    expect(result[0]!.id).toBe("test-1");
    expect((result[0]!.data as { label: string }).label).toBe("Test");
  });

  it("places nodes without recognized entityType after typed lanes", () => {
    const nodes = [
      makeTypedNode("b1", "backlog"),
      makeNode("unknown1"), // no entityType
    ];
    const result = applyGroupedLayout(nodes, "TB");

    const posB = result.find((n) => n.id === "b1")!.position;
    const posU = result.find((n) => n.id === "unknown1")!.position;
    // Unknown node should appear after the backlog lane.
    expect(posU.y).toBeGreaterThan(posB.y);
  });
});
