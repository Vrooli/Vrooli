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
  isPackageNode,
  isSymbolNode,
  isFileNode,
} from "./graphAdapter";

const PKG_A = "package:example.com/m/pkga";
const PKG_B = "package:example.com/m/pkgb";
const PKG_C = "package:example.com/m/pkgc";

/** A small graph: pkga ⇄ pkgb (a 2-cycle), pkgc standalone; one file w/ symbols. */
function makeGraph(): CodeGraph {
  return create(CodeGraphSchema, {
    nodes: [
      { id: PKG_A, kind: NodeKind.PACKAGE, name: "pkga", path: "example.com/m/pkga" },
      { id: PKG_B, kind: NodeKind.PACKAGE, name: "pkgb", path: "example.com/m/pkgb" },
      { id: PKG_C, kind: NodeKind.PACKAGE, name: "pkgc", path: "example.com/m/pkgc" },
      {
        id: "file:pkga/a.go",
        kind: NodeKind.FILE,
        name: "a.go",
        path: "pkga/a.go",
        attributes: { package_id: PKG_A },
      },
      {
        // Symbol node: folded onto PACKAGE with attributes.kind set.
        id: "go_func:pkga:CallB",
        kind: NodeKind.PACKAGE,
        name: "CallB",
        path: "pkga/a.go",
        attributes: { kind: "go_func", file_id: "file:pkga/a.go", exported: "true" },
      },
      {
        id: "go_type:pkga:TypeA",
        kind: NodeKind.PACKAGE,
        name: "TypeA",
        path: "pkga/a.go",
        attributes: { kind: "go_type", file_id: "file:pkga/a.go", exported: "false" },
      },
    ],
    edges: [
      { id: "e1", kind: EdgeKind.IMPORT, fromNodeId: PKG_A, toNodeId: PKG_B },
      { id: "e2", kind: EdgeKind.IMPORT, fromNodeId: PKG_B, toNodeId: PKG_A },
    ],
  });
}

describe("node classifiers", () => {
  it("distinguishes package, file, and folded symbol nodes", () => {
    const g = makeGraph();
    const byId = new Map(g.nodes.map((n) => [n.id, n]));
    expect(isPackageNode(byId.get(PKG_A)!)).toBe(true);
    expect(isFileNode(byId.get("file:pkga/a.go")!)).toBe(true);
    expect(isSymbolNode(byId.get("go_func:pkga:CallB")!)).toBe(true);
    // A folded symbol is NOT counted as a package even though its proto kind is PACKAGE.
    expect(isPackageNode(byId.get("go_func:pkga:CallB")!)).toBe(false);
  });
});

describe("summarizeGraph", () => {
  it("counts files, packages, symbols, and import edges", () => {
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
      [PKG_A, [PKG_B]],
      [PKG_B, [PKG_A]],
      [PKG_C, []],
    ]);
    const cycles = detectCycleNodes([PKG_A, PKG_B, PKG_C], adjacency);
    expect(cycles.has(PKG_A)).toBe(true);
    expect(cycles.has(PKG_B)).toBe(true);
    expect(cycles.has(PKG_C)).toBe(false);
  });

  it("flags a self-edge as a cycle", () => {
    const adjacency = new Map<string, string[]>([[PKG_A, [PKG_A]]]);
    expect(detectCycleNodes([PKG_A], adjacency).has(PKG_A)).toBe(true);
  });
});

describe("buildGraphLayout", () => {
  it("emits one layout node per package and marks cycle membership", () => {
    const layout = buildGraphLayout(makeGraph());
    expect(layout.nodes.map((n) => n.id).sort()).toEqual([PKG_A, PKG_B, PKG_C].sort());
    expect(layout.cycleCount).toBe(2);
    const a = layout.nodes.find((n) => n.id === PKG_A);
    const c = layout.nodes.find((n) => n.id === PKG_C);
    expect(a?.inCycle).toBe(true);
    expect(c?.inCycle).toBe(false);
  });

  it("includes import edges between included packages and flags cycle edges", () => {
    const layout = buildGraphLayout(makeGraph());
    expect(layout.edges).toHaveLength(2);
    expect(layout.edges.every((e) => e.inCycle)).toBe(true);
  });

  it("applies the package filter, dropping edges to filtered-out packages", () => {
    const layout = buildGraphLayout(makeGraph(), new Set(["example.com/m/pkgc"]));
    expect(layout.nodes.map((n) => n.id)).toEqual([PKG_C]);
    expect(layout.edges).toHaveLength(0);
    // packages list is always the full pre-filter set, for the filter bar.
    expect(layout.packages).toContain("example.com/m/pkga");
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
  it("groups symbols under their file by file_id and sorts by name", () => {
    const files = buildFileIndex(makeGraph());
    expect(files).toHaveLength(1);
    const file = files[0]!;
    expect(file.path).toBe("pkga/a.go");
    expect(file.symbols.map((s) => s.name)).toEqual(["CallB", "TypeA"]);
    expect(file.symbols.find((s) => s.name === "CallB")?.exported).toBe(true);
    expect(file.symbols.find((s) => s.name === "TypeA")?.kind).toBe("go_type");
  });
});
