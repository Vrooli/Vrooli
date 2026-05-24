// Extract a deterministic CodeGraph from a TS project using ts-morph.
//
// REQ-P0-003 load-bearing contract: every emitted declaration node carries
// `leading_comments` as an array of VERBATIM source-text strings taken from
// getLeadingCommentRanges(). No trimming, no JSDoc parsing.
//
// Determinism:
//   - source files: stable sort by file path
//   - declarations within a file: stable sort by (line, column)
//   - edges: stable sort by (from_node_id, to_node_id)
//
// Workspace detection: if a pnpm-workspace.yaml exists at scenario_path or
// any parent up to (and including) the dir containing tsconfig.json, throw
// WorkspaceUnsupportedError. The Go side maps this to ErrorKind
// "workspace_unsupported".

import * as fs from "node:fs";
import * as path from "node:path";
import {
  Node,
  Project,
  SourceFile,
  SyntaxKind,
  ts,
} from "ts-morph";

import type {
  CodeGraph,
  CodeGraphEdge,
  CodeGraphNode,
  CodeGraphWarning,
} from "./protocol.js";

// --- Wire constants (mirror the proto enum values) ----------------------------

// common.v1.NodeKind
const NK_UNSPECIFIED = 0;
const NK_FILE = 1;
// Reserved tag values for TS-specific kinds (200..209). These ride on
// attributes["kind"] per the typescript-code-graph proto; the top-level
// CodeGraphNode.kind for declarations is NK_MODULE-adjacent. We emit
// FILE for file nodes and use the TS_NODE_KIND_* numeric value on
// declarations so consumers can also read it from Node.kind directly.
const TS_NODE_KIND_MODULE = 200;
const TS_NODE_KIND_COMPONENT = 201;
const TS_NODE_KIND_HOOK = 202;
const TS_NODE_KIND_CLASS = 203;
const TS_NODE_KIND_INTERFACE = 204;
const TS_NODE_KIND_TYPE = 205;
const TS_NODE_KIND_FUNCTION = 206;
const TS_NODE_KIND_VAR = 207;
const TS_NODE_KIND_CONST = 208;
const TS_NODE_KIND_RE_EXPORT = 209;

// common.v1.EdgeKind
const EK_IMPORT = 1;
const EK_RE_EXPORT = 3;

// common.v1.CodeGraphWarningKind
const WK_PARSE_ERROR = 1;
const WK_TYPE_CHECK_FAILURE = 3;

// Short kind segments used in node ids: "<kind>:<path>[:<name>]"
const KIND_SHORT: Record<number, string> = {
  [NK_FILE]: "file",
  [TS_NODE_KIND_MODULE]: "ts_module",
  [TS_NODE_KIND_COMPONENT]: "ts_component",
  [TS_NODE_KIND_HOOK]: "ts_hook",
  [TS_NODE_KIND_CLASS]: "ts_class",
  [TS_NODE_KIND_INTERFACE]: "ts_interface",
  [TS_NODE_KIND_TYPE]: "ts_type",
  [TS_NODE_KIND_FUNCTION]: "ts_function",
  [TS_NODE_KIND_VAR]: "ts_var",
  [TS_NODE_KIND_CONST]: "ts_const",
  [TS_NODE_KIND_RE_EXPORT]: "ts_re_export",
};

// String name used in attributes["kind"] (matches TsNodeKind enum names).
const KIND_ATTR: Record<number, string> = {
  [TS_NODE_KIND_MODULE]: "TS_NODE_KIND_MODULE",
  [TS_NODE_KIND_COMPONENT]: "TS_NODE_KIND_COMPONENT",
  [TS_NODE_KIND_HOOK]: "TS_NODE_KIND_HOOK",
  [TS_NODE_KIND_CLASS]: "TS_NODE_KIND_CLASS",
  [TS_NODE_KIND_INTERFACE]: "TS_NODE_KIND_INTERFACE",
  [TS_NODE_KIND_TYPE]: "TS_NODE_KIND_TYPE",
  [TS_NODE_KIND_FUNCTION]: "TS_NODE_KIND_FUNCTION",
  [TS_NODE_KIND_VAR]: "TS_NODE_KIND_VAR",
  [TS_NODE_KIND_CONST]: "TS_NODE_KIND_CONST",
  [TS_NODE_KIND_RE_EXPORT]: "TS_NODE_KIND_RE_EXPORT",
};

// --- Typed errors -------------------------------------------------------------

export class NoTsConfigError extends Error {
  readonly kind = "no_tsconfig_found" as const;
}
export class MultipleTsConfigError extends Error {
  readonly kind = "multiple_tsconfig_files" as const;
}
export class WorkspaceUnsupportedError extends Error {
  readonly kind = "workspace_unsupported" as const;
}
export class PathUnreadableError extends Error {
  readonly kind = "path_unreadable" as const;
}
export class ParseFailureError extends Error {
  readonly kind = "parse_failure" as const;
}

// --- Public API ---------------------------------------------------------------

export interface ExtractInput {
  scenarioPath: string;
  /**
   * Test seam: skip the on-disk tsconfig / workspace detection and use the
   * provided pre-built ts-morph Project instead. Production callers do not
   * set this.
   */
  _project?: Project;
  /**
   * Test seam: when `_project` is set, this provides the rootDir to relativize
   * paths against. Defaults to scenarioPath.
   */
  _rootDirOverride?: string;
}

export interface ExtractOutput {
  graph: CodeGraph;
  warnings: CodeGraphWarning[];
}

export function extract(input: ExtractInput): ExtractOutput {
  const project = input._project ?? buildProject(input.scenarioPath);
  const rootDir = input._rootDirOverride ?? input.scenarioPath;

  const sourceFiles = [...project.getSourceFiles()].sort((a, b) =>
    a.getFilePath().localeCompare(b.getFilePath()),
  );

  const nodes: CodeGraphNode[] = [];
  const edges: CodeGraphEdge[] = [];
  const warnings: CodeGraphWarning[] = [];

  for (const sf of sourceFiles) {
    const relPath = relativize(rootDir, sf.getFilePath());

    // FILE node
    const fileId = `file:${relPath}`;
    nodes.push({
      id: fileId,
      kind: NK_FILE,
      name: path.basename(relPath),
      path: relPath,
      attributes: { language: "typescript" },
      leading_comments: [],
    });

    // MODULE node (one per file)
    const moduleId = `${KIND_SHORT[TS_NODE_KIND_MODULE]}:${relPath}`;
    nodes.push({
      id: moduleId,
      kind: TS_NODE_KIND_MODULE,
      name: path.basename(relPath),
      path: relPath,
      attributes: {
        language: "typescript",
        kind: KIND_ATTR[TS_NODE_KIND_MODULE]!,
      },
      leading_comments: [],
    });

    // Declarations
    const decls = collectDeclarations(sf);
    for (const d of decls) {
      const tsKind = classify(d);
      const shortKind = KIND_SHORT[tsKind] ?? "ts_function";
      const id = `${shortKind}:${relPath}:${d.name}`;
      nodes.push({
        id,
        kind: tsKind,
        name: d.name,
        path: relPath,
        attributes: {
          language: "typescript",
          kind: KIND_ATTR[tsKind] ?? KIND_ATTR[TS_NODE_KIND_FUNCTION]!,
          exported: d.exported ? "true" : "false",
        },
        leading_comments: d.leadingComments,
      });
    }

    // Imports + Re-exports
    for (const imp of sf.getImportDeclarations()) {
      const spec = imp.getModuleSpecifierValue();
      const target = resolveSpecifier(rootDir, sf, spec);
      if (!target) continue;
      edges.push({
        id: `import:${moduleId}->${target}`,
        kind: EK_IMPORT,
        from_node_id: moduleId,
        to_node_id: target,
        attributes: { specifier: spec },
      });
    }
    for (const exp of sf.getExportDeclarations()) {
      const spec = exp.getModuleSpecifierValue();
      if (!spec) continue; // `export { foo }` without `from` is intra-file
      const target = resolveSpecifier(rootDir, sf, spec);
      if (!target) continue;
      // Emit a TS_RE_EXPORT node + an EDGE_KIND_RE_EXPORT edge.
      const reExpId = `${KIND_SHORT[TS_NODE_KIND_RE_EXPORT]}:${relPath}:${spec}`;
      nodes.push({
        id: reExpId,
        kind: TS_NODE_KIND_RE_EXPORT,
        name: spec,
        path: relPath,
        attributes: {
          language: "typescript",
          kind: KIND_ATTR[TS_NODE_KIND_RE_EXPORT]!,
          specifier: spec,
        },
        leading_comments: leadingCommentsOf(exp),
      });
      edges.push({
        id: `re_export:${moduleId}->${target}`,
        kind: EK_RE_EXPORT,
        from_node_id: moduleId,
        to_node_id: target,
        attributes: { specifier: spec },
      });
    }

    // Diagnostics → warnings
    for (const diag of sf.getPreEmitDiagnostics()) {
      const cat = diag.getCategory();
      // Category: 0=Warning, 1=Error, 2=Suggestion, 3=Message
      // Map syntax errors to PARSE_ERROR, type-check errors to TYPE_CHECK_FAILURE.
      const code = diag.getCode();
      const isSyntax = code >= 1000 && code < 2000;
      warnings.push({
        kind: isSyntax ? WK_PARSE_ERROR : WK_TYPE_CHECK_FAILURE,
        file: relPath,
        message: flattenMessage(diag.getMessageText()),
      });
      void cat; // keep classification rule simple in v1
    }
  }

  // Stable sort
  nodes.sort((a, b) => a.id.localeCompare(b.id));
  edges.sort((a, b) => {
    const fc = a.from_node_id.localeCompare(b.from_node_id);
    if (fc !== 0) return fc;
    return a.to_node_id.localeCompare(b.to_node_id);
  });
  warnings.sort((a, b) => {
    const f = a.file.localeCompare(b.file);
    if (f !== 0) return f;
    return a.message.localeCompare(b.message);
  });

  return { graph: { nodes, edges }, warnings };
}

// --- Project construction -----------------------------------------------------

function buildProject(scenarioPath: string): Project {
  let stat: fs.Stats;
  try {
    stat = fs.statSync(scenarioPath);
  } catch (err) {
    throw new PathUnreadableError(`cannot stat scenario_path: ${(err as Error).message}`);
  }
  if (!stat.isDirectory()) {
    throw new PathUnreadableError(`scenario_path is not a directory: ${scenarioPath}`);
  }

  // Locate tsconfig.json
  const direct = path.join(scenarioPath, "tsconfig.json");
  let tsconfigPath: string | null = null;
  const matches: string[] = [];
  try {
    if (fs.existsSync(direct)) {
      tsconfigPath = direct;
      matches.push(direct);
    } else {
      // shallow scan for siblings (e.g. tsconfig.app.json AND tsconfig.json)
      const entries = fs.readdirSync(scenarioPath);
      for (const e of entries) {
        if (e === "tsconfig.json") matches.push(path.join(scenarioPath, e));
      }
    }
  } catch (err) {
    throw new PathUnreadableError(`cannot read scenario_path: ${(err as Error).message}`);
  }
  if (matches.length === 0) {
    throw new NoTsConfigError(`no tsconfig.json found at ${scenarioPath}`);
  }
  if (matches.length > 1) {
    throw new MultipleTsConfigError(
      `multiple tsconfig.json files: ${matches.join(", ")}`,
    );
  }
  tsconfigPath = matches[0]!;

  // Workspace check: pnpm-workspace.yaml at scenarioPath or any parent up to
  // and including the tsconfig's dir.
  const tsconfigDir = path.dirname(tsconfigPath);
  let cursor = scenarioPath;
  // Walk upward but bounded.
  for (let i = 0; i < 64; i++) {
    if (fs.existsSync(path.join(cursor, "pnpm-workspace.yaml"))) {
      throw new WorkspaceUnsupportedError(
        `pnpm-workspace.yaml found at ${cursor}; workspaces unsupported`,
      );
    }
    if (cursor === tsconfigDir) break;
    const parent = path.dirname(cursor);
    if (parent === cursor) break;
    cursor = parent;
  }

  try {
    return new Project({
      tsConfigFilePath: tsconfigPath,
      skipAddingFilesFromTsConfig: false,
    });
  } catch (err) {
    throw new ParseFailureError(
      `ts-morph project construction failed: ${(err as Error).message}`,
    );
  }
}

// --- Declaration collection + classification ----------------------------------

interface CollectedDecl {
  name: string;
  node: Node;
  exported: boolean;
  leadingComments: string[];
  // Heuristic flags
  returnsJsx: boolean;
}

function collectDeclarations(sf: SourceFile): CollectedDecl[] {
  const out: CollectedDecl[] = [];

  for (const fn of sf.getFunctions()) {
    const name = fn.getName();
    if (!name) continue;
    out.push({
      name,
      node: fn,
      exported: fn.isExported(),
      leadingComments: leadingCommentsOf(fn),
      returnsJsx: bodyReturnsJsx(fn),
    });
  }
  for (const cls of sf.getClasses()) {
    const name = cls.getName();
    if (!name) continue;
    out.push({
      name,
      node: cls,
      exported: cls.isExported(),
      leadingComments: leadingCommentsOf(cls),
      returnsJsx: false,
    });
  }
  for (const iface of sf.getInterfaces()) {
    out.push({
      name: iface.getName(),
      node: iface,
      exported: iface.isExported(),
      leadingComments: leadingCommentsOf(iface),
      returnsJsx: false,
    });
  }
  for (const ta of sf.getTypeAliases()) {
    out.push({
      name: ta.getName(),
      node: ta,
      exported: ta.isExported(),
      leadingComments: leadingCommentsOf(ta),
      returnsJsx: false,
    });
  }
  for (const en of sf.getEnums()) {
    out.push({
      name: en.getName(),
      node: en,
      exported: en.isExported(),
      leadingComments: leadingCommentsOf(en),
      returnsJsx: false,
    });
  }
  for (const vs of sf.getVariableStatements()) {
    const exported = vs.isExported();
    const lead = leadingCommentsOf(vs);
    for (const d of vs.getDeclarations()) {
      const init = d.getInitializer();
      const fnLike =
        init &&
        (init.getKind() === SyntaxKind.ArrowFunction ||
          init.getKind() === SyntaxKind.FunctionExpression);
      out.push({
        name: d.getName(),
        node: d,
        exported,
        leadingComments: lead,
        returnsJsx: fnLike ? expressionReturnsJsx(init) : false,
      });
    }
  }

  // Sort by (line, column)
  out.sort((a, b) => {
    const ap = a.node.getStart();
    const bp = b.node.getStart();
    return ap - bp;
  });

  return out;
}

function classify(d: CollectedDecl): number {
  const kind = d.node.getKind();
  if (kind === SyntaxKind.ClassDeclaration) return TS_NODE_KIND_CLASS;
  if (kind === SyntaxKind.InterfaceDeclaration) return TS_NODE_KIND_INTERFACE;
  if (kind === SyntaxKind.TypeAliasDeclaration) return TS_NODE_KIND_TYPE;
  if (kind === SyntaxKind.EnumDeclaration) return TS_NODE_KIND_TYPE;

  // Hook: /^use[A-Z]/
  if (/^use[A-Z]/.test(d.name)) return TS_NODE_KIND_HOOK;
  // Component: /^[A-Z]/ AND returns JSX
  if (/^[A-Z]/.test(d.name) && d.returnsJsx) return TS_NODE_KIND_COMPONENT;

  if (
    kind === SyntaxKind.FunctionDeclaration ||
    (kind === SyntaxKind.VariableDeclaration && isFunctionLikeVar(d.node))
  ) {
    return TS_NODE_KIND_FUNCTION;
  }

  // VariableDeclaration: const vs let/var via parent flags
  if (kind === SyntaxKind.VariableDeclaration) {
    const parent = d.node.getParent();
    const stmt = parent?.getParent();
    if (stmt && Node.isVariableStatement(stmt)) {
      const flags = stmt.getDeclarationKind();
      if (flags === "const") return TS_NODE_KIND_CONST;
      return TS_NODE_KIND_VAR;
    }
    return TS_NODE_KIND_VAR;
  }

  return TS_NODE_KIND_FUNCTION;
}

function isFunctionLikeVar(node: Node): boolean {
  if (!Node.isVariableDeclaration(node)) return false;
  const init = node.getInitializer();
  if (!init) return false;
  const k = init.getKind();
  return k === SyntaxKind.ArrowFunction || k === SyntaxKind.FunctionExpression;
}

function bodyReturnsJsx(fn: Node): boolean {
  return containsJsx(fn);
}

function expressionReturnsJsx(expr: Node): boolean {
  return containsJsx(expr);
}

function containsJsx(root: Node): boolean {
  let found = false;
  root.forEachDescendant((child, traversal) => {
    const k = child.getKind();
    if (
      k === SyntaxKind.JsxElement ||
      k === SyntaxKind.JsxSelfClosingElement ||
      k === SyntaxKind.JsxFragment
    ) {
      found = true;
      traversal.stop();
    }
  });
  return found;
}

// --- Leading comments — verbatim source slices --------------------------------

function leadingCommentsOf(node: Node): string[] {
  const sf = node.getSourceFile();
  const fullText = sf.getFullText();
  // ts-morph exposes the underlying ts API on getLeadingCommentRanges()
  const compilerNode = node.compilerNode;
  const ranges = ts.getLeadingCommentRanges(fullText, compilerNode.pos) ?? [];
  return ranges.map((r) => fullText.slice(r.pos, r.end));
}

// --- Helpers ------------------------------------------------------------------

function relativize(rootDir: string, absPath: string): string {
  const rel = path.relative(rootDir, absPath);
  // Normalize to forward slashes for stable cross-platform ids (Linux-only
  // host today, but the wire shape should be portable).
  return rel.split(path.sep).join("/");
}

function resolveSpecifier(
  rootDir: string,
  sf: SourceFile,
  spec: string,
): string | null {
  // Only resolve relative specifiers in v1 — bare specifiers (e.g. "react")
  // are external and we don't emit nodes for them.
  if (!spec.startsWith(".") && !spec.startsWith("/")) return null;
  const fromDir = path.dirname(sf.getFilePath());
  const candidates = [
    spec,
    spec + ".ts",
    spec + ".tsx",
    spec + ".d.ts",
    path.join(spec, "index.ts"),
    path.join(spec, "index.tsx"),
  ];
  for (const c of candidates) {
    const abs = path.resolve(fromDir, c);
    if (fs.existsSync(abs)) {
      const rel = relativize(rootDir, abs);
      return `ts_module:${rel}`;
    }
  }
  // Couldn't resolve on disk; still emit a pointer using the literal path
  // (relative to the importing file's dir) so consumers see the unresolved
  // edge. Caller may want to fold this into a warning.
  const guess = relativize(rootDir, path.resolve(fromDir, spec));
  return `ts_module:${guess}`;
}

interface DiagChainLike {
  messageText: string;
  next?: DiagChainLike[] | undefined;
}

function flattenMessage(m: unknown): string {
  if (typeof m === "string") return m;
  if (m && typeof m === "object" && "messageText" in m) {
    const chain = m as DiagChainLike;
    let out = chain.messageText;
    let next = chain.next;
    while (next && next.length > 0) {
      out += " " + next[0]!.messageText;
      next = next[0]!.next;
    }
    return out;
  }
  return String(m);
}
