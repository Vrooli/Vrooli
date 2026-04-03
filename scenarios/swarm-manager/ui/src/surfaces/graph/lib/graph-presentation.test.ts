import { describe, expect, it } from "vitest";
import { buildGraphPresentation, filterGraphEdges, filterGraphNodes } from "./graph-presentation";
import { createDefaultLensSettings } from "../stores/graph-settings-store";
import type { GraphLensSettings } from "../stores/graph-settings-store";
import type { GraphEdge, GraphNode } from "../types";
import { makeGraphEdge, makeGraphNode } from "../test-helpers";

const makeNode = (
  id: string,
  entityType: Parameters<typeof makeGraphNode>[1],
  extra: Record<string, unknown> = {},
): GraphNode => makeGraphNode(id, entityType, { label: id, ...extra });

const makeEdge = (id: string, source: string, target: string, type: string): GraphEdge =>
  makeGraphEdge(id, source, target, type);

describe("filterGraphNodes — entity-scoped status filters", () => {
  it("hides a backlog node by status without affecting execution nodes with the same status", () => {
    const settings: GraphLensSettings = {
      ...createDefaultLensSettings("topology"),
      statusFilters: {
        backlog: { completed: false },
      },
    };

    const nodes = [
      makeNode("backlog-item/execute/task-a", "backlog", { status: "completed" }),
      makeNode("execution/exec-1", "execution", { status: "completed" }),
      makeNode("backlog-item/execute/task-b", "backlog", { status: "in_progress" }),
    ];

    const result = filterGraphNodes(nodes, settings);
    expect(result.map((n) => n.id)).toEqual([
      "execution/exec-1",
      "backlog-item/execute/task-b",
    ]);
  });

  it("shows all nodes when statusFilters is empty", () => {
    const settings = createDefaultLensSettings("topology");

    const nodes = [
      makeNode("backlog-item/execute/task-a", "backlog", { status: "completed" }),
      makeNode("execution/exec-1", "execution", { status: "failed" }),
    ];

    const result = filterGraphNodes(nodes, settings);
    expect(result).toHaveLength(2);
  });

  it("filters multiple statuses within a single entity type", () => {
    const settings: GraphLensSettings = {
      ...createDefaultLensSettings("topology"),
      statusFilters: {
        execution: { completed: false, canceled: false },
      },
    };

    const nodes = [
      makeNode("execution/e1", "execution", { status: "completed" }),
      makeNode("execution/e2", "execution", { status: "running" }),
      makeNode("execution/e3", "execution", { status: "canceled" }),
    ];

    const result = filterGraphNodes(nodes, settings);
    expect(result.map((n) => n.id)).toEqual(["execution/e2"]);
  });

  it("hides only nodes whose entity+status combo is filtered", () => {
    const settings: GraphLensSettings = {
      ...createDefaultLensSettings("topology"),
      statusFilters: {
        initiative: { active: false },
      },
    };

    const nodes = [
      makeNode("initiative/init-1", "initiative", { status: "active" }),
      makeNode("initiative/init-2", "initiative", { status: "completed" }),
    ];

    const result = filterGraphNodes(nodes, settings);
    // init-1 is hidden (status=active is filtered), init-2 has status=completed which is not filtered
    expect(result.map((n) => n.id)).toEqual(["initiative/init-2"]);
  });
});

describe("filterGraphEdges", () => {
  it("keeps edges between visible nodes", () => {
    const settings = createDefaultLensSettings("topology");
    const visibleNodes = [
      makeNode("a", "backlog"),
      makeNode("b", "backlog"),
    ];
    const edges = [makeEdge("e1", "a", "b", "depends_on")];

    const result = filterGraphEdges(edges, visibleNodes, settings);
    expect(result).toHaveLength(1);
  });

  it("drops edges where source or target is missing from visible set", () => {
    const settings = createDefaultLensSettings("topology");
    const visibleNodes = [makeNode("a", "backlog")];
    const edges = [
      makeEdge("e1", "a", "b", "depends_on"),
      makeEdge("e2", "c", "a", "depends_on"),
    ];

    const result = filterGraphEdges(edges, visibleNodes, settings);
    expect(result).toHaveLength(0);
  });

  it("hides secondary edges when showSecondaryEdges is false", () => {
    const settings: GraphLensSettings = {
      ...createDefaultLensSettings("topology"),
      showSecondaryEdges: false,
    };
    const nodes = [makeNode("a", "backlog"), makeNode("b", "backlog")];
    const edges = [
      makeEdge("e1", "a", "b", "depends_on"),     // NOT secondary (primary)
      makeEdge("e2", "a", "b", "classified_as"),   // IS secondary
    ];

    const result = filterGraphEdges(edges, nodes, settings);
    const ids = result.map((e) => e.id);
    expect(ids).toContain("e1");
    expect(ids).not.toContain("e2");
  });

  it("shows secondary edges when showSecondaryEdges is true", () => {
    const settings = createDefaultLensSettings("topology"); // default true
    const nodes = [makeNode("a", "backlog"), makeNode("b", "scenario")];
    const edges = [makeEdge("e1", "a", "b", "classified_as")];

    const result = filterGraphEdges(edges, nodes, settings);
    expect(result).toHaveLength(1);
  });

  it("returns empty for empty edge list", () => {
    const settings = createDefaultLensSettings("topology");
    const result = filterGraphEdges([], [makeNode("a", "backlog")], settings);
    expect(result).toHaveLength(0);
  });
});

describe("buildGraphPresentation", () => {
  it("returns empty result for empty input", () => {
    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [],
      edges: [],
      settings: createDefaultLensSettings("topology"),
      expandedTopologyClusters: new Set<string>(),
    });
    expect(result.processedNodes).toHaveLength(0);
    expect(result.processedEdges).toHaveLength(0);
    expect(result.visibleNodeCount).toBe(0);
    expect(result.visibleEdgeTypes).toHaveLength(0);
  });

  it("uses flat presentation on non-topology lens regardless of groupingMode", () => {
    const settings = createDefaultLensSettings("operations");
    settings.groupingMode = "initiative";

    const result = buildGraphPresentation({
      lens: "operations",
      nodes: [
        makeNode("backlog-item/execute/task-a", "backlog", { status: "in_progress" }),
        makeNode("execution/exec-1", "execution", { status: "running" }),
      ],
      edges: [makeEdge("e1", "execution/exec-1", "backlog-item/execute/task-a", "executes")],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    // Flat presentation: nodes pass through without clustering
    expect(result.processedNodes).toHaveLength(2);
    expect(result.processedNodes.every((n) => n.type !== "cluster")).toBe(true);
  });

  it("uses flat presentation on topology when groupingMode is none", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "none";

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/init-1", "initiative", { status: "active" }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog" }),
      ],
      edges: [makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of")],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    // Flat: initiative is a regular node, no clusters created
    expect(result.processedNodes).toHaveLength(2);
    expect(result.processedNodes.some((n) => n.type === "cluster")).toBe(false);
  });

  it("expands cluster when in expandedTopologyClusters set", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "initiative";

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/init-1", "initiative", { status: "active", rollup: { total: 1, completed: 0, in_progress: 0, failed: 0, pending: 1 } }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog", kind: "execute" }),
      ],
      edges: [makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of")],
      settings,
      expandedTopologyClusters: new Set(["initiative/init-1"]),
    });

    // Expanded: cluster node + child node both present
    const clusterNode = result.processedNodes.find((n) => n.id === "initiative/init-1");
    expect(clusterNode).toBeDefined();
    const childNode = result.processedNodes.find((n) => n.id === "backlog-item/execute/task-a");
    expect(childNode).toBeDefined();
    expect(childNode?.parentId).toBe("initiative/init-1");
  });

  it("hides child nodes when cluster is collapsed", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "initiative";

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/init-1", "initiative", { status: "active", rollup: { total: 2, completed: 0, in_progress: 0, failed: 0, pending: 2 } }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog", kind: "execute" }),
        makeNode("backlog-item/execute/task-b", "backlog", { status: "backlog", kind: "execute" }),
      ],
      edges: [
        makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of"),
        makeEdge("mo2", "backlog-item/execute/task-b", "initiative/init-1", "member_of"),
      ],
      settings,
      expandedTopologyClusters: new Set<string>(), // all collapsed
    });

    // Collapsed: only cluster node, no child nodes
    const nodeIds = result.processedNodes.map((n) => n.id);
    expect(nodeIds).toContain("initiative/init-1");
    expect(nodeIds).not.toContain("backlog-item/execute/task-a");
    expect(nodeIds).not.toContain("backlog-item/execute/task-b");
  });

  it("collects visible edge types from processed edges", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "none"; // flat mode to preserve edges

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("a", "backlog"),
        makeNode("b", "backlog"),
        makeNode("c", "scenario"),
      ],
      edges: [
        makeEdge("e1", "a", "b", "depends_on"),
        makeEdge("e2", "a", "c", "targets"),
      ],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    expect(result.visibleEdgeTypes).toContain("depends_on");
    expect(result.visibleEdgeTypes).toContain("targets");
  });

  it("applies entity filters before grouping", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "initiative";
    settings.entityFilters.backlog = false;

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/init-1", "initiative", { status: "active" }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog" }),
        makeNode("scenario/app", "scenario", { status: "running" }),
      ],
      edges: [
        makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of"),
      ],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    // Backlog filtered out → no cluster members → cluster is empty
    // Only scenario should remain as unclustered
    const nodeIds = result.processedNodes.map((n) => n.id);
    expect(nodeIds).not.toContain("backlog-item/execute/task-a");
    expect(nodeIds).toContain("scenario/app");
  });

  it("keeps backlog items visible in flat topology presentation", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "none"; // explicit flat mode

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/graph-adoption", "initiative", { status: "active" }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog", kind: "execute" }),
        makeNode("scenario/swarm-manager", "scenario", { status: "running" }),
      ],
      edges: [
        makeEdge(
          "member_of:a",
          "backlog-item/execute/task-a",
          "initiative/graph-adoption",
          "member_of",
        ),
      ],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    expect(result.processedNodes.map((node) => node.id)).toEqual([
      "initiative/graph-adoption",
      "backlog-item/execute/task-a",
      "scenario/swarm-manager",
    ]);
  });

  it("collapses backlog items into initiative clusters when compression is enabled", () => {
    const settings = createDefaultLensSettings("topology");
    settings.groupingMode = "initiative";

    const result = buildGraphPresentation({
      lens: "topology",
      nodes: [
        makeNode("initiative/graph-adoption", "initiative", { status: "active" }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog", kind: "execute" }),
        makeNode("scenario/swarm-manager", "scenario", { status: "running" }),
      ],
      edges: [
        makeEdge(
          "member_of:a",
          "backlog-item/execute/task-a",
          "initiative/graph-adoption",
          "member_of",
        ),
        makeEdge(
          "targets:a",
          "backlog-item/execute/task-a",
          "scenario/swarm-manager",
          "targets",
        ),
      ],
      settings,
      expandedTopologyClusters: new Set<string>(),
    });

    expect(result.processedNodes.map((node) => node.id)).toEqual([
      "initiative/graph-adoption",
      "scenario/swarm-manager",
    ]);
    expect(result.processedEdges).toMatchObject([
      {
        id: "agg:initiative/graph-adoption|scenario/swarm-manager|targets",
        source: "initiative/graph-adoption",
        target: "scenario/swarm-manager",
        type: "targets",
      },
    ]);
  });
});
