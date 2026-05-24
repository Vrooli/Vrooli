import { Project } from "ts-morph";
import { describe, expect, it } from "vitest";

import { extract } from "../src/extract.js";

function inMemoryProject(): Project {
  return new Project({
    useInMemoryFileSystem: true,
    compilerOptions: { jsx: 2 /* React */, target: 99, allowJs: false },
  });
}

describe("extract", () => {
  it("emits file + module nodes and classifies hook, component, class, interface, type", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/widget.tsx",
      `
/** @vrooliWidget kind=card */
export function MyWidget() {
  return <div>hi</div>;
}

/** custom hook */
export function useCounter() { return 0; }

export class Box {}
export interface BoxShape { w: number }
export type BoxId = string;
export const PI = 3.14;
export let count = 0;
      `,
    );
    project.createSourceFile(
      "/proj/src/index.ts",
      `export * from "./widget";`,
    );

    const out = extract({
      scenarioPath: "/proj",
      _project: project,
      _rootDirOverride: "/proj",
    });

    const ids = out.graph.nodes.map((n) => n.id).sort();
    // FILE + MODULE for each source file
    expect(ids).toContain("file:src/widget.tsx");
    expect(ids).toContain("ts_module:src/widget.tsx");
    expect(ids).toContain("file:src/index.ts");
    expect(ids).toContain("ts_module:src/index.ts");

    // Classifications
    expect(ids).toContain("ts_component:src/widget.tsx:MyWidget");
    expect(ids).toContain("ts_hook:src/widget.tsx:useCounter");
    expect(ids).toContain("ts_class:src/widget.tsx:Box");
    expect(ids).toContain("ts_interface:src/widget.tsx:BoxShape");
    expect(ids).toContain("ts_type:src/widget.tsx:BoxId");
    expect(ids).toContain("ts_const:src/widget.tsx:PI");
    expect(ids).toContain("ts_var:src/widget.tsx:count");

    // Re-export edge + node
    expect(ids).toContain("ts_re_export:src/index.ts:./widget");
    const reExportEdges = out.graph.edges.filter((e) => e.kind === 3);
    expect(reExportEdges.length).toBeGreaterThan(0);
  });

  it("preserves verbatim leading comments on declarations (REQ-P0-003)", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/widget.tsx",
      `/** @vrooliWidget kind=card */
export function MyWidget() {
  return <div>hi</div>;
}`,
    );
    const out = extract({
      scenarioPath: "/proj",
      _project: project,
      _rootDirOverride: "/proj",
    });
    const widget = out.graph.nodes.find((n) => n.id === "ts_component:src/widget.tsx:MyWidget");
    expect(widget).toBeDefined();
    expect(widget!.leading_comments).toEqual(["/** @vrooliWidget kind=card */"]);
  });

  it("preserves multiple leading comment blocks verbatim", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/x.ts",
      `// line comment
/* block */
/** jsdoc */
export const X = 1;`,
    );
    const out = extract({
      scenarioPath: "/proj",
      _project: project,
      _rootDirOverride: "/proj",
    });
    const x = out.graph.nodes.find((n) => n.id === "ts_const:src/x.ts:X");
    expect(x).toBeDefined();
    expect(x!.leading_comments).toEqual([
      "// line comment",
      "/* block */",
      "/** jsdoc */",
    ]);
  });

  it("produces stable output across runs", () => {
    const make = () => {
      const p = inMemoryProject();
      p.createSourceFile("/proj/src/b.ts", `export const B = 2;`);
      p.createSourceFile("/proj/src/a.ts", `export const A = 1;`);
      return extract({ scenarioPath: "/proj", _project: p, _rootDirOverride: "/proj" });
    };
    const r1 = make();
    const r2 = make();
    expect(JSON.stringify(r1)).toBe(JSON.stringify(r2));
  });

  it("emits IMPORT edge for relative imports between in-memory files", () => {
    const project = inMemoryProject();
    project.createSourceFile("/proj/src/util.ts", `export const U = 1;`);
    project.createSourceFile(
      "/proj/src/main.ts",
      `import { U } from "./util"; export const M = U;`,
    );
    const out = extract({ scenarioPath: "/proj", _project: project, _rootDirOverride: "/proj" });
    // In-memory FS won't satisfy fs.existsSync, so the import resolves to a
    // best-guess ts_module pointer — verify the edge exists with kind=IMPORT.
    const importEdges = out.graph.edges.filter((e) => e.kind === 1);
    expect(importEdges.length).toBeGreaterThan(0);
    expect(importEdges[0]!.from_node_id).toBe("ts_module:src/main.ts");
  });
});
