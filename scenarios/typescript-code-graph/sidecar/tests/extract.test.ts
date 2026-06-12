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
      projectPath: "/proj",
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
      projectPath: "/proj",
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
      projectPath: "/proj",
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
      return extract({ projectPath: "/proj", _project: p, _rootDirOverride: "/proj" });
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
    const out = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" });
    // In-memory FS won't satisfy fs.existsSync, so the import resolves to a
    // best-guess ts_module pointer — verify the edge exists with kind=IMPORT.
    const importEdges = out.graph.edges.filter((e) => e.kind === 1);
    expect(importEdges.length).toBeGreaterThan(0);
    expect(importEdges[0]!.from_node_id).toBe("ts_module:src/main.ts");
  });

  it("emits a WK_UNRESOLVED_IMPORT warning for a dangling relative import (D3)", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/main.ts",
      `import { Gone } from "./does-not-exist"; export const M = Gone;`,
    );
    const out = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" });

    // The dangling edge is still emitted (consumers see the dependency)...
    const importEdges = out.graph.edges.filter((e) => e.kind === 1);
    expect(importEdges.length).toBeGreaterThan(0);

    // ...AND an additive UNRESOLVED_IMPORT (kind=2) warning is raised so
    // consumers can distinguish a real edge from a dangling one.
    const unresolved = out.warnings.filter((w) => w.kind === 2);
    expect(unresolved.length).toBe(1);
    expect(unresolved[0]!.file).toBe("src/main.ts");
    expect(unresolved[0]!.message).toBe("./does-not-exist");
  });

  it("does not warn for external (bare) specifiers", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/main.ts",
      `import { useState } from "react"; export const M = useState;`,
    );
    const out = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" });
    // Bare specifiers are external: no edge, no warning.
    expect(out.warnings.filter((w) => w.kind === 2).length).toBe(0);
  });

  it("emits generic import, reference, call, JSX, and export facts", () => {
    const project = inMemoryProject();
    project.createSourceFile(
      "/proj/src/client.ts",
      `
export function sendThing(input: { id: string }) {
  return input.id;
}
export type Response = { ok: boolean };
      `,
    );
    project.createSourceFile(
      "/proj/src/widget.tsx",
      `
import React, { useMemo as memo, type ReactNode } from "react";
import * as Client from "./client";
import { sendThing } from "./client";

export function Widget({ child }: { child: ReactNode }) {
  const result = sendThing({ id: "1" });
  const other = Client.sendThing({ id: result });
  const value = memo(() => other, [other]);
  return <Panel title={value}>{child}</Panel>;
}

function Panel(props: { title: string; children: ReactNode }) {
  return <section>{props.children}</section>;
}
      `,
    );

    const out = extract({ projectPath: "/proj", _project: project, _rootDirOverride: "/proj" });
    const nodes = out.graph.nodes;
    const byKind = (kind: string) => nodes.filter((n) => n.attributes.kind === kind);

    const importBindings = byKind("TS_NODE_KIND_IMPORT_BINDING");
    expect(importBindings.map((n) => n.name)).toEqual(
      expect.arrayContaining(["React", "memo", "ReactNode", "Client", "sendThing"]),
    );
    expect(importBindings.find((n) => n.name === "ReactNode")?.attributes.type_only).toBe("true");
    expect(importBindings.find((n) => n.name === "Client")?.attributes.import_kind).toBe("namespace");

    const calls = byKind("TS_NODE_KIND_CALL");
    expect(calls.map((n) => n.attributes.callee)).toEqual(
      expect.arrayContaining(["sendThing", "Client.sendThing", "memo"]),
    );
    expect(calls.find((n) => n.attributes.callee === "sendThing")?.attributes.enclosing_declaration).toBe("Widget");

    const jsx = byKind("TS_NODE_KIND_JSX_USAGE");
    expect(jsx.map((n) => n.attributes.component_name)).toEqual(
      expect.arrayContaining(["Panel", "section"]),
    );
    expect(jsx.find((n) => n.attributes.component_name === "Panel")?.attributes.enclosing_declaration).toBe("Widget");

    const references = byKind("TS_NODE_KIND_REFERENCE");
    expect(references.map((n) => n.name)).toEqual(expect.arrayContaining(["sendThing", "result", "other", "child"]));

    const exports = byKind("TS_NODE_KIND_EXPORT");
    expect(exports.map((n) => n.name)).toEqual(expect.arrayContaining(["Widget", "sendThing", "Response"]));
    for (const fact of [...importBindings, ...calls, ...jsx, ...references, ...exports]) {
      expect(fact.attributes.start_line).toMatch(/^\d+$/);
      expect(fact.attributes.end_column).toMatch(/^\d+$/);
    }
  });
});
