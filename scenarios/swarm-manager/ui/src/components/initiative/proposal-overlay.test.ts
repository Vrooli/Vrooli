import { describe, it, expect } from "vitest";
import { buildOverlay } from "./proposal-overlay";
import type { Proposal } from "../../types";

function mut(id: string, rest: Record<string, unknown>): Proposal["mutations"] extends (infer U)[] | undefined ? U : never {
  return { id, ...rest } as never;
}

describe("buildOverlay", () => {
  it("returns empty for full_graph proposals", () => {
    const overlay = buildOverlay({
      form: "full_graph",
      graph: { nodes: [], edges: [] },
    });
    expect(overlay).toEqual({});
  });

  it("collects added/archived/moved nodes and status changes", () => {
    const proposal: Proposal = {
      form: "mutation_list",
      mutations: [
        mut("m1", {
          op: "add_item",
          item: { kind: "execute", name: "new", title: "New item", depends_on: ["execute/old"] },
        }),
        mut("m2", { op: "archive_item", target: "execute/stale" }),
        mut("m3", { op: "move_initiative", target: "execute/old", initiative: "other" }),
        mut("m4", { op: "change_status", target: "execute/keep", status: "ready" }),
        mut("m5", { op: "add_edge", from: "execute/a", to: "execute/b" }),
        mut("m6", { op: "remove_edge", from: "execute/a", to: "execute/c" }),
      ],
    };
    const overlay = buildOverlay(proposal);
    expect(overlay.addedNodeIds).toContain("execute/new");
    expect(overlay.archivedNodeIds).toContain("execute/stale");
    expect(overlay.movedOutNodeIds).toContain("execute/old");
    expect(overlay.statusChanges).toEqual({ "execute/keep": "ready" });
    expect(overlay.addedEdges).toEqual(
      expect.arrayContaining([
        { from: "execute/old", to: "execute/new" },
        { from: "execute/a", to: "execute/b" },
      ]),
    );
    expect(overlay.removedEdges).toEqual([{ from: "execute/a", to: "execute/c" }]);
  });

  it("respects the acceptedIds filter", () => {
    const proposal: Proposal = {
      form: "mutation_list",
      mutations: [
        mut("m1", { op: "archive_item", target: "execute/a" }),
        mut("m2", { op: "archive_item", target: "execute/b" }),
      ],
    };
    const overlay = buildOverlay(proposal, { acceptedIds: ["m1"] });
    expect(overlay.archivedNodeIds).toEqual(["execute/a"]);
  });

  it("split_item archives the source and adds the new items", () => {
    const proposal: Proposal = {
      form: "mutation_list",
      mutations: [
        mut("m1", {
          op: "split_item",
          target: "execute/big",
          into: [
            { kind: "execute", name: "left", title: "Left half" },
            { kind: "execute", name: "right", title: "Right half" },
          ],
        }),
      ],
    };
    const overlay = buildOverlay(proposal);
    expect(overlay.archivedNodeIds).toEqual(["execute/big"]);
    expect(overlay.addedNodeIds).toEqual(["execute/left", "execute/right"]);
    expect(overlay.addedNodes?.map((n) => n.id)).toEqual(["execute/left", "execute/right"]);
  });

  it("merge_items archives sources, adds the merged item, and projects its non-source deps as edges", () => {
    const proposal: Proposal = {
      form: "mutation_list",
      mutations: [
        mut("m1", {
          op: "merge_items",
          sources: ["execute/alpha", "execute/beta"],
          item: {
            kind: "execute",
            name: "merged",
            title: "Merged",
            // include a source ref to verify it's filtered out, plus a
            // genuine outbound dep that should project to an edge.
            depends_on: ["execute/alpha", "execute/gamma"],
          },
        }),
      ],
    };
    const overlay = buildOverlay(proposal);
    expect(overlay.archivedNodeIds).toEqual(
      expect.arrayContaining(["execute/alpha", "execute/beta"]),
    );
    expect(overlay.addedNodeIds).toEqual(["execute/merged"]);
    expect(overlay.addedNodes?.map((n) => n.id)).toEqual(["execute/merged"]);
    // Source-ref dep filtered; gamma dep projected.
    expect(overlay.addedEdges).toEqual([{ from: "execute/gamma", to: "execute/merged" }]);
  });

  it("ignores ops that don't affect graph topology", () => {
    const proposal: Proposal = {
      form: "mutation_list",
      mutations: [
        mut("m1", { op: "update_item", target: "execute/a", patch: { title: "renamed" } }),
        mut("m2", { op: "change_priority", target: "execute/a", priority: 3 }),
        mut("m3", { op: "interrupt_in_progress", target: "execute/a" }),
      ],
    };
    const overlay = buildOverlay(proposal);
    expect(overlay.addedNodeIds).toEqual([]);
    expect(overlay.archivedNodeIds).toEqual([]);
    expect(overlay.addedEdges).toEqual([]);
    expect(overlay.removedEdges).toEqual([]);
    expect(overlay.statusChanges).toEqual({});
  });
});
