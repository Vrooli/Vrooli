import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  CodeGraphSchema,
  NodeKind,
  EdgeKind,
} from "@vrooli/proto-types/common/v1/code_graph_pb";
import type { CodeGraph } from "@vrooli/proto-types/common/v1/code_graph_pb";

import {
  buildGraphLayout,
  buildFileIndex,
  summarizeGraph,
  detectCycleNodes,
  isModuleNode,
  isSymbolNode,
  isFileNode,
} from "./graphAdapter";

const MOD_A = "ts_module:src/a.ts";
const MOD_B = "ts_module:src/b.ts";
const MOD_C = "ts_module:src/c.ts";

/**
 * A small TS graph: module a ⇄ module b (a 2-cycle via import + re-export), c
 * standalone; one file with two symbols (one carrying JSDoc leading comments).
 * Module nodes carry attributes.kind = TS_NODE_KIND_MODULE (the numeric
 * NodeKind field holds the 200-range extension value the adapter ignores).
 */
function makeGraph(): CodeGraph {
  return create(CodeGraphSchema, {
    nodes: [
      { id: MOD_A, kind: NodeKind.MODULE, name: "a.ts", path: "src/a.ts", attributes: { kind: "TS_NODE_KIND_MODULE", language: "typescript" } },
      { id: MOD_B, kind: NodeKind.MODULE, name: "b.ts", path: "src/b.ts", attributes: { kind: "TS_NODE_KIND_MODULE", language: "typescript" } },
      { id: MOD_C, kind: NodeKind.MODULE, name: "c.ts", path: "src/c.ts", attributes: { kind: "TS_NODE_KIND_MODULE", language: "typescript" } },
      {
        id: "file:src/a.ts",
        kind: NodeKind.FILE,
        name: "a.ts",
        path: "src/a.ts",
        attributes: { language: "typescript" },
      },
      {
        // Symbol node: linked to its file by matching `path`.
        id: "ts_function:src/a.ts:callB",
        kind: NodeKind.UNSPECIFIED,
        name: "callB",
        path: "src/a.ts",
        attributes: { kind: "TS_NODE_KIND_FUNCTION", exported: "true" },
        leadingComments: ["/** Calls module b. */"],
      },
      {
        id: "ts_type:src/a.ts:TypeA",
        kind: NodeKind.UNSPECIFIED,
        name: "TypeA",
        path: "src/a.ts",
        attributes: { kind: "TS_NODE_KIND_TYPE", exported: "false" },
      },
    ],
    edges: [
      { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: MOD_A, toNodeId: MOD_B },
      { id: "e2", kind: EdgeKind.RE_EXPORT, fromNodeId: MOD_B, toNodeId: MOD_A },
    ],
  });
}

describe("node classifiers", () => {
  it("distinguishes module, file, and symbol nodes", () => {
    const g = makeGraph();
    const byId = new Map(g.nodes.map((n) => [n.id, n]));
    expect(isModuleNode(byId.get(MOD_A)!)).toBe(true);
    expect(isFileNode(byId.get("file:src/a.ts")!)).toBe(true);
    expect(isSymbolNode(byId.get("ts_function:src/a.ts:callB")!)).toBe(true);
    // A symbol is NOT a module even though both ride the common envelope.
    expect(isModuleNode(byId.get("ts_function:src/a.ts:callB")!)).toBe(false);
  });
});

describe("summarizeGraph", () => {
  it("counts files, modules, symbols, and dependency edges", () => {
    const summary = summarizeGraph(makeGraph());
    expect(summary).toEqual({ files: 1, packages: 3, symbols: 2, imports: 2 });
  });

  it("returns zeros for an undefined graph", () => {
    expect(summarizeGraph(undefined)).toEqual({ files: 0, packages: 0, symbols: 0, imports: 0 });
  });
});

describe("detectCycleNodes", () => {
  it("flags both members of a 2-cycle and excludes acyclic nodes", () => {
    const adjacency = new Map<string, string[]>([
      [MOD_A, [MOD_B]],
      [MOD_B, [MOD_A]],
      [MOD_C, []],
    ]);
    const cycles = detectCycleNodes([MOD_A, MOD_B, MOD_C], adjacency);
    expect(cycles.has(MOD_A)).toBe(true);
    expect(cycles.has(MOD_B)).toBe(true);
    expect(cycles.has(MOD_C)).toBe(false);
  });

  it("flags a self-edge as a cycle", () => {
    const adjacency = new Map<string, string[]>([[MOD_A, [MOD_A]]]);
    expect(detectCycleNodes([MOD_A], adjacency).has(MOD_A)).toBe(true);
  });
});

describe("buildGraphLayout", () => {
  it("emits one layout node per module and marks cycle membership", () => {
    const layout = buildGraphLayout(makeGraph());
    expect(layout.nodes.map((n) => n.id).sort()).toEqual([MOD_A, MOD_B, MOD_C].sort());
    expect(layout.cycleCount).toBe(2);
    const a = layout.nodes.find((n) => n.id === MOD_A);
    const c = layout.nodes.find((n) => n.id === MOD_C);
    expect(a?.inCycle).toBe(true);
    expect(c?.inCycle).toBe(false);
  });

  it("includes dependency edges (import + re-export) between included modules and flags cycle edges", () => {
    const layout = buildGraphLayout(makeGraph());
    expect(layout.edges).toHaveLength(2);
    expect(layout.edges.every((e) => e.inCycle)).toBe(true);
  });

  it("applies the module filter, dropping edges to filtered-out modules", () => {
    const layout = buildGraphLayout(makeGraph(), new Set(["src/c.ts"]));
    expect(layout.nodes.map((n) => n.id)).toEqual([MOD_C]);
    expect(layout.edges).toHaveLength(0);
    // packages list is always the full pre-filter set, for the filter bar.
    expect(layout.packages).toContain("src/a.ts");
  });

  it("returns an empty layout for an undefined graph", () => {
    expect(buildGraphLayout(undefined)).toEqual({
      nodes: [],
      edges: [],
      packages: [],
      cycleCount: 0,
    });
  });
});

describe("buildFileIndex", () => {
  it("groups symbols under their file by path and sorts by name", () => {
    const files = buildFileIndex(makeGraph());
    expect(files).toHaveLength(1);
    const file = files[0]!;
    expect(file.path).toBe("src/a.ts");
    expect(file.symbols.map((s) => s.name)).toEqual(["callB", "TypeA"]);
    expect(file.symbols.find((s) => s.name === "callB")?.exported).toBe(true);
    expect(file.symbols.find((s) => s.name === "TypeA")?.kind).toBe("TS_NODE_KIND_TYPE");
  });

  it("carries each symbol's leading comments (JSDoc) verbatim", () => {
    const files = buildFileIndex(makeGraph());
    const callB = files[0]!.symbols.find((s) => s.name === "callB");
    expect(callB?.leadingComments).toEqual(["/** Calls module b. */"]);
    // A symbol with no leading comments gets an empty list, never undefined.
    const typeA = files[0]!.symbols.find((s) => s.name === "TypeA");
    expect(typeA?.leadingComments).toEqual([]);
  });
});
