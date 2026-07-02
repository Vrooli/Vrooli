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

  it("keeps secondary edges in focus lens even when showSecondaryEdges is false", () => {
    // Regression: focus lens already aggressively filters to attention-worthy
    // items + structural context (initiatives via member_of, scenarios via
    // targets). Applying the secondary-edge filter on top hides the context
    // edges that were the reason we pulled those nodes in, leaving scenarios
    // floating without visible connections to the backlog items.
    const settings: GraphLensSettings = {
      ...createDefaultLensSettings("focus"),
      showSecondaryEdges: false,
    };
    const nodes = [makeNode("a", "backlog"), makeNode("b", "scenario")];
    const edges = [makeEdge("e1", "a", "b", "targets")];

    const result = filterGraphEdges(edges, nodes, settings, "focus");
    expect(result).toHaveLength(1);
    expect(result[0]?.id).toBe("e1");
  });
});
describe("buildGraphPresentation", () => {
  it("returns empty result for empty input", () => {
    const result = buildGraphPresentation({
      lens: "focus",
      nodes: [],
      edges: [],
      settings: createDefaultLensSettings("focus"),
    });
    expect(result.processedNodes).toHaveLength(0);
    expect(result.processedEdges).toHaveLength(0);
    expect(result.visibleNodeCount).toBe(0);
    expect(result.visibleEdgeTypes).toHaveLength(0);
  });

  it("passes nodes through flat without any clustering", () => {
    const settings = createDefaultLensSettings("focus");

    const result = buildGraphPresentation({
      lens: "focus",
      nodes: [
        makeNode("backlog-item/execute/task-a", "backlog", { status: "in_progress" }),
        makeNode("execution/exec-1", "execution", { status: "running" }),
      ],
      edges: [makeEdge("e1", "execution/exec-1", "backlog-item/execute/task-a", "executes")],
      settings,
    });

    expect(result.processedNodes).toHaveLength(2);
    expect(result.processedNodes.every((n) => n.type !== "cluster")).toBe(true);
    expect(result.visibleNodeCount).toBe(2);
  });

  it("collects visible edge types from processed edges", () => {
    const settings = createDefaultLensSettings("focus");
    settings.showSecondaryEdges = true;

    const result = buildGraphPresentation({
      lens: "focus",
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
    });

    expect(result.visibleEdgeTypes).toContain("depends_on");
    expect(result.visibleEdgeTypes).toContain("targets");
  });

  it("applies entity filters", () => {
    const settings = createDefaultLensSettings("focus");
    settings.entityFilters.backlog = false;

    const result = buildGraphPresentation({
      lens: "focus",
      nodes: [
        makeNode("initiative/init-1", "initiative", { status: "active" }),
        makeNode("backlog-item/execute/task-a", "backlog", { status: "backlog" }),
        makeNode("scenario/app", "scenario", { status: "running" }),
      ],
      edges: [
        makeEdge("mo1", "backlog-item/execute/task-a", "initiative/init-1", "member_of"),
      ],
      settings,
    });

    const nodeIds = result.processedNodes.map((n) => n.id);
    expect(nodeIds).not.toContain("backlog-item/execute/task-a");
    expect(nodeIds).toContain("scenario/app");
  });
});
