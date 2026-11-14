import { Project } from "ts-morph";
import { describe, expect, it } from "vitest";

import { extract } from "../src/extract.js";

// A minified bundle is one very long line with tens of thousands of
// identifiers — the shape of vendored files such as Draco or Emscripten
// output that JavaScript-only projects keep under public/. Location
// attributes must cost O(log n) per node, not a rescan of the file, or a
// single extract pins a CPU core for hours.
function minifiedBundle(statements: number): string {
  const parts = ["var a0=0;"];
  for (let i = 1; i < statements; i++) parts.push(`var a${i}=a${i - 1}+1;`);
  return parts.join("");
}

describe("source locations", () => {
  it("extracts a minified single-line bundle in bounded time", () => {
    const project = new Project({ useInMemoryFileSystem: true, compilerOptions: { allowJs: true } });
    const statements = 24_000;
    project.createSourceFile("/proj/src/bundle.js", minifiedBundle(statements));

    const started = performance.now();
    const graph = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" }).graph;
    const elapsedMs = performance.now() - started;

    const last = graph.nodes.find((n) => n.id.startsWith("ts_reference:") && n.name === `a${statements - 2}`);
    expect(last?.attributes.start_line).toBe("1");
    expect(Number(last?.attributes.start_column)).toBeGreaterThan(200_000);
    // Linear location lookups finish this in a few seconds even on a
    // loaded host; per-node rescans take minutes.
    expect(elapsedMs).toBeLessThan(15_000);
  }, 120_000);

  it("extracts a file with more usage facts than one call can take as arguments", () => {
    const project = new Project({ useInMemoryFileSystem: true, compilerOptions: { allowJs: true } });
    const references = 150_000;
    project.createSourceFile("/proj/public/bundle.min.js", `var a=0;${"a;".repeat(references)}`);

    const nodes = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" }).graph.nodes;

    expect(nodes.filter((n) => n.id.startsWith("ts_reference:")).length).toBe(references);
  }, 120_000);

  it("reports the same line and column as ts-morph for every line-break style", () => {
    const text = [
      "const lf = 1;\n",
      "const crlf = lf;\r\n",
      "const cr = crlf;\r",
      "const ls = cr;\u2028const ps = ls;\u2029",
      "\tconst tabbed = ps; const wide = '\u{1F600}' + tabbed;\n",
      "\n\nfunction last() { return wide; }",
    ].join("");
    const project = new Project({ useInMemoryFileSystem: true });
    const sf = project.createSourceFile("/proj/src/mixed.ts", text);

    const expected = new Map<string, Set<string>>();
    sf.forEachDescendant((node) => {
      if (node.getKindName() !== "Identifier") return;
      const start = sf.getLineAndColumnAtPos(node.getStart());
      const end = sf.getLineAndColumnAtPos(node.getEnd());
      const ranges = expected.get(node.getText()) ?? new Set<string>();
      ranges.add(`${start.line}:${start.column}-${end.line}:${end.column}`);
      expected.set(node.getText(), ranges);
    });

    const nodes = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" }).graph.nodes;
    const references = nodes.filter((n) => n.id.startsWith("ts_reference:"));
    expect(references.map((n) => n.name).sort()).toEqual(["cr", "crlf", "lf", "ls", "ps", "tabbed", "wide"]);
    for (const n of references) {
      const a = n.attributes;
      const range = `${a.start_line}:${a.start_column}-${a.end_line}:${a.end_column}`;
      expect(expected.get(n.name)?.has(range), `${n.name} emitted ${range}`).toBe(true);
      expect(n.id).toBe(`ts_reference:src/mixed.ts:${a.start_line}:${a.start_column}:${n.name}`);
    }
  });
});
