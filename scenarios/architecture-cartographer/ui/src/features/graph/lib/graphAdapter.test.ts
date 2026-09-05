import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  FileNodeSchema,
  GraphSnapshotSchema,
  ImportEdgeSchema,
  Language,
  PackageNodeSchema,
} from "@vrooli/proto-types/architecture-cartographer/v1/graph/graph_pb";
import {
  ConflictSchema,
  Severity,
} from "@vrooli/proto-types/architecture-cartographer/v1/shared/shared_pb";

import { buildGraphLayout } from "./graphAdapter";

const makePackage = (id: string, repoPath: string) =>
  create(PackageNodeSchema, {
    id,
    importPath: id,
    repoPath,
    language: Language.GO,
  });

const makeFile = (id: string, path: string, packageId: string) =>
  create(FileNodeSchema, {
    id,
    path,
    packageId,
    language: Language.GO,
    lines: 0,
    isTest: false,
  });

const makeEdge = (from: string, toPackageId: string) =>
  create(ImportEdgeSchema, {
    from,
    toPackageId,
    symbolIds: [],
    testOnly: false,
  });

const makeSnapshot = () =>
  create(GraphSnapshotSchema, {
    id: "snap-1",
    scenario: "demo",
    contentHash: "hash",
    languages: [Language.GO],
    files: [
      makeFile("file:graph/a.go", "graph/a.go", "pkg:graph"),
      makeFile("file:graph/b.go", "graph/b.go", "pkg:graph"),
      makeFile("file:conflicts/c.go", "conflicts/c.go", "pkg:conflicts"),
    ],
    packages: [
      makePackage("pkg:graph", "graph"),
      makePackage("pkg:conflicts", "conflicts"),
    ],
    symbols: [],
    imports: [makeEdge("file:graph/a.go", "pkg:conflicts")],
  });

describe("buildGraphLayout", () => {
  it("returns empty layout when snapshot is undefined", () => {
    const layout = buildGraphLayout(undefined, []);
    expect(layout.nodes).toEqual([]);
    expect(layout.edges).toEqual([]);
    expect(layout.domains).toEqual([]);
  });

  it("assigns layers by BFS through import edges", () => {
    const layout = buildGraphLayout(makeSnapshot(), []);
    const a = layout.nodes.find((n) => n.id === "file:graph/a.go");
    const b = layout.nodes.find((n) => n.id === "file:graph/b.go");
    const c = layout.nodes.find((n) => n.id === "file:conflicts/c.go");
    expect(a?.layer).toBe(0);
    expect(b?.layer).toBe(0);
    // c is imported by a (via pkg:conflicts) so it lands on layer 1.
    expect(c?.layer).toBe(1);
  });

  it("derives the domain from the package repo path's first segment", () => {
    const layout = buildGraphLayout(makeSnapshot(), []);
    expect(layout.domains).toEqual(["conflicts", "graph"]);
    const a = layout.nodes.find((n) => n.id === "file:graph/a.go");
    expect(a?.domain).toBe("graph");
  });

  it("overlays the highest-severity conflict on matching file paths", () => {
    const conflict = create(ConflictSchema, {
      id: "c-1",
      scenario: "demo",
      detector: "cycle",
      type: "cycle",
      severity: Severity.BLOCKER,
      locations: ["graph/a.go"],
      domains: [],
    });
    const layout = buildGraphLayout(makeSnapshot(), [conflict]);
    const a = layout.nodes.find((n) => n.id === "file:graph/a.go");
    const b = layout.nodes.find((n) => n.id === "file:graph/b.go");
    expect(a?.conflictSeverity).toBe("critical");
    expect(b?.conflictSeverity).toBeUndefined();
  });

  it("filters out nodes whose domain is excluded; drops dangling edges", () => {
    const layout = buildGraphLayout(makeSnapshot(), [], new Set(["graph"]));
    const ids = layout.nodes.map((n) => n.id);
    expect(ids).toContain("file:graph/a.go");
    expect(ids).not.toContain("file:conflicts/c.go");
    // Edge a -> c drops because c is filtered out.
    expect(layout.edges).toEqual([]);
  });

  it("produces a deterministic node order across runs", () => {
    const first = buildGraphLayout(makeSnapshot(), []).nodes.map((n) => n.id);
    const second = buildGraphLayout(makeSnapshot(), []).nodes.map((n) => n.id);
    expect(first).toEqual(second);
  });
});
