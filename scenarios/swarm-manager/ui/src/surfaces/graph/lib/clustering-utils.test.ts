import { describe, it, expect } from "vitest";
import type { Node, Edge } from "@xyflow/react";
import {
  buildClusterHierarchy,
  aggregateEdgesForCollapsed,
  applyNodeCap,
  UNASSIGNED_CLUSTER_ID,
} from "./clustering-utils";

const makeNode = (id: string, entityType: string, extra: Record<string, unknown> = {}): Node => ({
  id,
  position: { x: 0, y: 0 },
  data: { entityType, label: id, ...extra },
});

const makeEdge = (id: string, source: string, target: string, type: string): Edge => ({
  id,
  source,
  target,
  type,
});

function expectDefined<T>(value: T | undefined, message: string): T {
  expect(value).toBeDefined();
  if (value === undefined) {
    throw new Error(message);
  }
  return value;
}

describe("buildClusterHierarchy", () => {
  it("groups backlog items by initiative via member_of edges", () => {
    const nodes = [
      makeNode("initiative/init-1", "initiative", { title: "Init 1" }),
      makeNode("backlog-item/execute/task-a", "backlog"),
      makeNode("backlog-item/execute/task-b", "backlog"),
      makeNode("scenario/my-app", "scenario"),
    ];
    const edges = [
      makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of"),
      makeEdge("mo2", "backlog-item/execute/task-b", "initiative/init-1", "member_of"),
    ];

    const { clusters, unclustered } = buildClusterHierarchy(nodes, edges);

    expect(clusters).toHaveLength(1);
    const firstCluster = expectDefined(clusters[0], "Expected initiative cluster");
    expect(firstCluster.id).toBe("initiative/init-1");
    expect(firstCluster.members).toHaveLength(2);
    expect(unclustered).toHaveLength(1);
    const firstUnclustered = expectDefined(unclustered[0], "Expected unclustered scenario");
    expect(firstUnclustered.id).toBe("scenario/my-app");
  });

  it("creates unassigned group for backlog items without initiative", () => {
    const nodes = [
      makeNode("backlog-item/execute/task-a", "backlog"),
      makeNode("backlog-item/fix/bug-1", "backlog"),
    ];

    const { clusters } = buildClusterHierarchy(nodes, []);

    expect(clusters).toHaveLength(1);
    const unassignedCluster = expectDefined(clusters[0], "Expected unassigned cluster");
    expect(unassignedCluster.id).toBe(UNASSIGNED_CLUSTER_ID);
    expect(unassignedCluster.members).toHaveLength(2);
  });

  it("handles empty initiatives", () => {
    const nodes = [
      makeNode("initiative/empty", "initiative"),
      makeNode("backlog-item/execute/task-a", "backlog"),
    ];

    const { clusters } = buildClusterHierarchy(nodes, []);

    // Only unassigned cluster (empty initiative has no members -> no cluster)
    expect(clusters).toHaveLength(1);
    const firstCluster = expectDefined(clusters[0], "Expected unassigned cluster");
    expect(firstCluster.id).toBe(UNASSIGNED_CLUSTER_ID);
  });

  it("extracts rollup data from initiative nodes", () => {
    const nodes = [
      makeNode("initiative/init-1", "initiative", {
        title: "Init 1",
        rollup: { total: 5, completed: 2, in_progress: 1, failed: 1, pending: 1 },
      }),
      makeNode("backlog-item/execute/task-a", "backlog"),
    ];
    const edges = [
      makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of"),
    ];

    const { clusters } = buildClusterHierarchy(nodes, edges);

    const firstCluster = expectDefined(clusters[0], "Expected initiative cluster");
    expect(firstCluster.rollup).toEqual({
      total: 5, completed: 2, in_progress: 1, failed: 1, pending: 1,
    });
  });
});

describe("aggregateEdgesForCollapsed", () => {
  it("returns edges unchanged when no clusters are collapsed", () => {
    const edges = [makeEdge("e1", "a", "b", "depends_on")];
    const result = aggregateEdgesForCollapsed(edges, new Set(), []);
    expect(result).toEqual(edges);
  });

  it("merges edges from collapsed cluster members into single aggregated edge", () => {
    const clusters = [{
      id: "initiative/init-1",
      label: "Init 1",
      members: ["task-a", "task-b"],
      rollup: null,
    }];
    const edges = [
      makeEdge("e1", "task-a", "scenario/app", "targets"),
      makeEdge("e2", "task-b", "scenario/app", "targets"),
    ];

    const result = aggregateEdgesForCollapsed(edges, new Set(["initiative/init-1"]), clusters);

    // Should produce one aggregated edge
    const aggregated = result.filter((e) => e.id.startsWith("agg:"));
    expect(aggregated).toHaveLength(1);
    const aggregatedEdge = expectDefined(aggregated[0], "Expected aggregated edge");
    expect((aggregatedEdge.data as Record<string, unknown>).aggregatedCount).toBe(2);
    expect(aggregatedEdge.source).toBe("initiative/init-1");
    expect(aggregatedEdge.target).toBe("scenario/app");
  });

  it("removes intra-cluster edges when collapsed", () => {
    const clusters = [{
      id: "initiative/init-1",
      label: "Init 1",
      members: ["task-a", "task-b"],
      rollup: null,
    }];
    const edges = [
      makeEdge("e1", "task-a", "task-b", "depends_on"),
    ];

    const result = aggregateEdgesForCollapsed(edges, new Set(["initiative/init-1"]), clusters);
    expect(result).toHaveLength(0);
  });

  it("preserves edges between non-clustered nodes", () => {
    const clusters = [{
      id: "initiative/init-1",
      label: "Init 1",
      members: ["task-a"],
      rollup: null,
    }];
    const edges = [
      makeEdge("e1", "scenario/a", "scenario/b", "targets"),
    ];

    const result = aggregateEdgesForCollapsed(edges, new Set(["initiative/init-1"]), clusters);
    expect(result).toHaveLength(1);
    const firstEdge = expectDefined(result[0], "Expected preserved edge");
    expect(firstEdge.id).toBe("e1");
  });
});

describe("applyNodeCap", () => {
  it("returns all nodes when under limit", () => {
    const nodes = [makeNode("a", "backlog"), makeNode("b", "backlog")];
    const { visible, cappedCount } = applyNodeCap(nodes, 50);
    expect(visible).toHaveLength(2);
    expect(cappedCount).toBe(0);
  });

  it("caps at limit and adds pseudo-node", () => {
    const nodes = Array.from({ length: 10 }, (_, i) =>
      makeNode(`node-${i}`, "backlog", { priority: i }),
    );
    const { visible, cappedCount } = applyNodeCap(nodes, 5);
    // 5 items + 1 pseudo-node
    expect(visible).toHaveLength(6);
    expect(cappedCount).toBe(5);

    const pseudoNode = visible.find((n) => n.id === "__more-items__");
    const definedPseudoNode = expectDefined(pseudoNode, "Expected pseudo node");
    expect((definedPseudoNode.data as Record<string, unknown>).label).toBe("More items (5)");
  });

  it("sorts by priority descending (keeps highest priority)", () => {
    const nodes = [
      makeNode("low", "backlog", { priority: 1 }),
      makeNode("high", "backlog", { priority: 10 }),
      makeNode("mid", "backlog", { priority: 5 }),
    ];
    const { visible } = applyNodeCap(nodes, 2);
    // Should keep high (10) and mid (5), cap low (1)
    const ids = visible.filter((n) => n.id !== "__more-items__").map((n) => n.id);
    expect(ids).toEqual(["high", "mid"]);
  });
});
