import { describe, expect, it } from "vitest";
import type { Edge, Node } from "@xyflow/react";
import { buildGraphPresentation } from "./graph-presentation";
import { createDefaultLensSettings } from "../stores/graph-data-store";

const makeNode = (id: string, entityType: string, extra: Record<string, unknown> = {}): Node => ({
  id,
  type: entityType,
  position: { x: 0, y: 0 },
  data: { label: id, entityType, ...extra },
});

const makeEdge = (id: string, source: string, target: string, type: string): Edge => ({
  id,
  source,
  target,
  type,
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
