import { describe, expect, it } from "vitest";
import { buildGraphPresentation, filterGraphNodes } from "./graph-presentation";
import { createDefaultLensSettings } from "../stores/graph-data-store";
import type { GraphLensSettings } from "../stores/graph-data-store";
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

describe("buildGraphPresentation", () => {
  it("keeps backlog items visible in the default topology presentation", () => {
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
      settings: createDefaultLensSettings("topology"),
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
