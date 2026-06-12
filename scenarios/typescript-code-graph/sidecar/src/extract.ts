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
// Workspace detection: if a pnpm-workspace.yaml exists at the project root or
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
// Reserved tag values for TS-specific kinds (200..299). These ride on
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
const TS_NODE_KIND_IMPORT_BINDING = 210;
const TS_NODE_KIND_REFERENCE = 211;
const TS_NODE_KIND_CALL = 212;
const TS_NODE_KIND_JSX_USAGE = 213;
const TS_NODE_KIND_EXPORT = 214;

// common.v1.EdgeKind
const EK_IMPORT = 1;
const EK_RE_EXPORT = 3;

// common.v1.CodeGraphWarningKind
const WK_PARSE_ERROR = 1;
const WK_UNRESOLVED_IMPORT = 2;
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
  [TS_NODE_KIND_IMPORT_BINDING]: "ts_import_binding",
  [TS_NODE_KIND_REFERENCE]: "ts_reference",
  [TS_NODE_KIND_CALL]: "ts_call",
  [TS_NODE_KIND_JSX_USAGE]: "ts_jsx_usage",
  [TS_NODE_KIND_EXPORT]: "ts_export",
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
  [TS_NODE_KIND_IMPORT_BINDING]: "TS_NODE_KIND_IMPORT_BINDING",
  [TS_NODE_KIND_REFERENCE]: "TS_NODE_KIND_REFERENCE",
  [TS_NODE_KIND_CALL]: "TS_NODE_KIND_CALL",
  [TS_NODE_KIND_JSX_USAGE]: "TS_NODE_KIND_JSX_USAGE",
  [TS_NODE_KIND_EXPORT]: "TS_NODE_KIND_EXPORT",
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
  projectPath: string;
  /**
   * Test seam: skip the on-disk tsconfig / workspace detection and use the
   * provided pre-built ts-morph Project instead. Production callers do not
   * set this.
   */
  _project?: Project;
  /**
   * Test seam: when `_project` is set, this provides the rootDir to relativize
   * paths against. Defaults to the resolved project root.
   */
  _rootDirOverride?: string;
}

export interface ExtractOutput {
  graph: CodeGraph;
  warnings: CodeGraphWarning[];
}

export function extract(input: ExtractInput): ExtractOutput {
  const resolved = input._project
    ? { project: input._project, rootDir: input._rootDirOverride ?? input.projectPath }
    : buildProject(input.projectPath);
  const project = resolved.project;
  const rootDir = input._rootDirOverride ?? resolved.rootDir;

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
    const declarationNodeIds = new Map<Node, string>();
    for (const d of decls) {
      const tsKind = classify(d);
      const shortKind = KIND_SHORT[tsKind] ?? "ts_function";
      const id = `${shortKind}:${relPath}:${d.name}`;
      declarationNodeIds.set(d.node, id);
      nodes.push({
        id,
        kind: tsKind,
        name: d.name,
        path: relPath,
        attributes: {
          language: "typescript",
          kind: KIND_ATTR[tsKind] ?? KIND_ATTR[TS_NODE_KIND_FUNCTION]!,
          exported: d.exported ? "true" : "false",
          ...sourceRangeAttrs(d.node),
        },
        leading_comments: d.leadingComments,
      });
      if (d.exported) {
        nodes.push(exportFactNode(relPath, d.name, "declaration", d.node, id));
      }
    }

    // Imports + Re-exports
    for (const imp of sf.getImportDeclarations()) {
      const spec = imp.getModuleSpecifierValue();
      for (const binding of importBindingFacts(relPath, imp)) {
        nodes.push(binding);
      }
      const res = resolveSpecifier(rootDir, sf, spec);
      if (!res.target) continue;
      // The guessed edge is still emitted so consumers see the
      // dependency; the warning is additive so they can tell a real
      // edge from a dangling one (REQ-P0 D3).
      if (res.unresolved) {
        warnings.push({ kind: WK_UNRESOLVED_IMPORT, file: relPath, message: spec });
      }
      edges.push({
        id: `import:${moduleId}->${res.target}`,
        kind: EK_IMPORT,
        from_node_id: moduleId,
        to_node_id: res.target,
        attributes: { specifier: spec },
      });
    }
    for (const exp of sf.getExportDeclarations()) {
      const spec = exp.getModuleSpecifierValue();
      if (!spec) continue; // `export { foo }` without `from` is intra-file
      const res = resolveSpecifier(rootDir, sf, spec);
      if (!res.target) continue;
      if (res.unresolved) {
        warnings.push({ kind: WK_UNRESOLVED_IMPORT, file: relPath, message: spec });
      }
      const target = res.target;
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
          ...sourceRangeAttrs(exp),
        },
        leading_comments: leadingCommentsOf(exp),
      });
      nodes.push(exportFactNode(relPath, spec, "re_export", exp, reExpId));
      edges.push({
        id: `re_export:${moduleId}->${target}`,
        kind: EK_RE_EXPORT,
        from_node_id: moduleId,
        to_node_id: target,
        attributes: { specifier: spec },
      });
    }

    const symbolIndex = buildSymbolIndex(decls, declarationNodeIds);
    nodes.push(...usageFactNodes(relPath, sf, symbolIndex));

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
  nodes.sort((a, b) => compareStable(a.id, b.id));
  edges.sort((a, b) => {
    const fc = compareStable(a.from_node_id, b.from_node_id);
    if (fc !== 0) return fc;
    return compareStable(a.to_node_id, b.to_node_id);
  });
  warnings.sort((a, b) => {
    const f = compareStable(a.file, b.file);
    if (f !== 0) return f;
    return compareStable(a.message, b.message);
  });

  return { graph: { nodes, edges }, warnings };
}

// --- Project construction -----------------------------------------------------

interface ResolvedProject {
  project: Project;
  rootDir: string;
}

function buildProject(projectPath: string): ResolvedProject {
  let stat: fs.Stats;
  try {
    stat = fs.statSync(projectPath);
  } catch (err) {
    throw new PathUnreadableError(`cannot stat project_path: ${(err as Error).message}`);
  }

  let projectRoot = projectPath;
  let explicitTsconfigPath: string | null = null;
  if (stat.isFile()) {
    if (path.basename(projectPath) !== "tsconfig.json") {
      throw new PathUnreadableError(`project_path file is not tsconfig.json: ${projectPath}`);
    }
    explicitTsconfigPath = projectPath;
    projectRoot = path.dirname(projectPath);
  } else if (!stat.isDirectory()) {
    throw new PathUnreadableError(`project_path is not a directory or tsconfig.json: ${projectPath}`);
  }

  // Locate tsconfig.json
  const direct = path.join(projectRoot, "tsconfig.json");
  let tsconfigPath: string | null = null;
  const matches: string[] = [];
  try {
    if (explicitTsconfigPath) {
      tsconfigPath = explicitTsconfigPath;
      matches.push(explicitTsconfigPath);
    } else if (fs.existsSync(direct)) {
      tsconfigPath = direct;
      matches.push(direct);
    } else {
      // shallow scan for siblings (e.g. tsconfig.app.json AND tsconfig.json)
      const entries = fs.readdirSync(projectRoot);
      for (const e of entries) {
        if (e === "tsconfig.json") matches.push(path.join(projectRoot, e));
      }
    }
  } catch (err) {
    throw new PathUnreadableError(`cannot read project_path: ${(err as Error).message}`);
  }
  if (matches.length === 0) {
    throw new NoTsConfigError(`no tsconfig.json found at ${projectPath}`);
  }
  if (matches.length > 1) {
    throw new MultipleTsConfigError(
      `multiple tsconfig.json files: ${matches.join(", ")}`,
    );
  }
  tsconfigPath = matches[0]!;

  // Workspace check: pnpm-workspace.yaml at projectRoot or any parent up to
  // and including the tsconfig's dir.
  const tsconfigDir = path.dirname(tsconfigPath);
  let cursor = projectRoot;
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
    const project = new Project({
      tsConfigFilePath: tsconfigPath,
      skipAddingFilesFromTsConfig: false,
    });
    return { project, rootDir: tsconfigDir };
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

// --- Generic usage facts ------------------------------------------------------

interface SymbolInfo {
  name: string;
  nodeId: string;
}

function buildSymbolIndex(
  decls: CollectedDecl[],
  declarationNodeIds: Map<Node, string>,
): Map<ts.Symbol, SymbolInfo> {
  const out = new Map<ts.Symbol, SymbolInfo>();
  for (const d of decls) {
    const sym = symbolOfDeclaration(d.node);
    const id = declarationNodeIds.get(d.node);
    if (sym && id) out.set(sym, { name: d.name, nodeId: id });
  }
  return out;
}

function symbolOfDeclaration(node: Node): ts.Symbol | undefined {
  if (
    Node.isFunctionDeclaration(node) ||
    Node.isClassDeclaration(node) ||
    Node.isInterfaceDeclaration(node) ||
    Node.isTypeAliasDeclaration(node) ||
    Node.isEnumDeclaration(node)
  ) {
    return node.getNameNode()?.getSymbol()?.compilerSymbol;
  }
  if (Node.isVariableDeclaration(node)) {
    return node.getNameNode().getSymbol()?.compilerSymbol;
  }
  return undefined;
}

function importBindingFacts(relPath: string, imp: import("ts-morph").ImportDeclaration): CodeGraphNode[] {
  const sourceModule = imp.getModuleSpecifierValue();
  const typeOnly = imp.isTypeOnly() ? "true" : "false";
  const nodes: CodeGraphNode[] = [];
  const defaultImport = imp.getDefaultImport();
  if (defaultImport) {
    nodes.push(importBindingNode(relPath, imp, sourceModule, "default", "default", defaultImport.getText(), typeOnly));
  }
  const namespaceImport = imp.getNamespaceImport();
  if (namespaceImport) {
    nodes.push(importBindingNode(relPath, imp, sourceModule, "namespace", "*", namespaceImport.getText(), typeOnly));
  }
  for (const named of imp.getNamedImports()) {
    const imported = named.getName();
    const alias = named.getAliasNode()?.getText() ?? imported;
    nodes.push(importBindingNode(
      relPath,
      named,
      sourceModule,
      "named",
      imported,
      alias,
      named.isTypeOnly() || imp.isTypeOnly() ? "true" : "false",
    ));
  }
  return nodes;
}

function importBindingNode(
  relPath: string,
  node: Node,
  sourceModule: string,
  importKind: string,
  importedName: string,
  localName: string,
  typeOnly: string,
): CodeGraphNode {
  return {
    id: `${KIND_SHORT[TS_NODE_KIND_IMPORT_BINDING]}:${relPath}:${stableLocation(node)}:${localName}`,
    kind: TS_NODE_KIND_IMPORT_BINDING,
    name: localName,
    path: relPath,
    attributes: {
      language: "typescript",
      kind: KIND_ATTR[TS_NODE_KIND_IMPORT_BINDING]!,
      source_module: sourceModule,
      import_kind: importKind,
      imported_name: importedName,
      local_name: localName,
      type_only: typeOnly,
      ...sourceRangeAttrs(node),
    },
    leading_comments: [],
  };
}

function exportFactNode(
  relPath: string,
  name: string,
  exportKind: string,
  node: Node,
  declarationNodeId: string,
): CodeGraphNode {
  return {
    id: `${KIND_SHORT[TS_NODE_KIND_EXPORT]}:${relPath}:${stableLocation(node)}:${name}`,
    kind: TS_NODE_KIND_EXPORT,
    name,
    path: relPath,
    attributes: {
      language: "typescript",
      kind: KIND_ATTR[TS_NODE_KIND_EXPORT]!,
      export_kind: exportKind,
      declaration_node_id: declarationNodeId,
      ...sourceRangeAttrs(node),
    },
    leading_comments: leadingCommentsOf(node),
  };
}

function usageFactNodes(
  relPath: string,
  sf: SourceFile,
  symbolIndex: Map<ts.Symbol, SymbolInfo>,
): CodeGraphNode[] {
  const nodes: CodeGraphNode[] = [];
  sf.forEachDescendant((child) => {
    if (Node.isImportDeclaration(child) || Node.isExportDeclaration(child)) return;
    if (Node.isCallExpression(child)) {
      nodes.push(callFactNode(relPath, child));
      return;
    }
    if (Node.isJsxSelfClosingElement(child)) {
      nodes.push(jsxFactNode(relPath, child, child.getTagNameNode().getText(), child.getAttributes().map((a) => a.getText())));
      return;
    }
    if (Node.isJsxOpeningElement(child)) {
      nodes.push(jsxFactNode(relPath, child, child.getTagNameNode().getText(), child.getAttributes().map((a) => a.getText())));
      return;
    }
    if (!Node.isIdentifier(child)) return;
    if (
      isDeclarationName(child) ||
      isImportOrExportIdentifier(child) ||
      isPropertyAccessName(child) ||
      isJsxTagIdentifier(child) ||
      isObjectLiteralPropertyName(child)
    ) return;
    nodes.push(referenceFactNode(relPath, child, symbolIndex));
  });
  return nodes;
}

function callFactNode(relPath: string, call: import("ts-morph").CallExpression): CodeGraphNode {
  const expr = call.getExpression();
  const signature = call.getType().getText(call);
  return {
    id: `${KIND_SHORT[TS_NODE_KIND_CALL]}:${relPath}:${stableLocation(call)}:${sanitizeId(expr.getText())}`,
    kind: TS_NODE_KIND_CALL,
    name: expr.getText(),
    path: relPath,
    attributes: {
      language: "typescript",
      kind: KIND_ATTR[TS_NODE_KIND_CALL]!,
      callee: expr.getText(),
      enclosing_declaration: enclosingDeclarationName(call),
      argument_count: String(call.getArguments().length),
      argument_summary: call.getArguments().map((a) => a.getType().getText(a)).join(","),
      return_type: signature,
      ...sourceRangeAttrs(call),
    },
    leading_comments: [],
  };
}

function jsxFactNode(
  relPath: string,
  node: Node,
  componentName: string,
  props: string[],
): CodeGraphNode {
  return {
    id: `${KIND_SHORT[TS_NODE_KIND_JSX_USAGE]}:${relPath}:${stableLocation(node)}:${sanitizeId(componentName)}`,
    kind: TS_NODE_KIND_JSX_USAGE,
    name: componentName,
    path: relPath,
    attributes: {
      language: "typescript",
      kind: KIND_ATTR[TS_NODE_KIND_JSX_USAGE]!,
      component_name: componentName,
      props_summary: props.join(","),
      enclosing_declaration: enclosingDeclarationName(node),
      ...sourceRangeAttrs(node),
    },
    leading_comments: [],
  };
}

function referenceFactNode(
  relPath: string,
  ident: import("ts-morph").Identifier,
  symbolIndex: Map<ts.Symbol, SymbolInfo>,
): CodeGraphNode {
  const symbol = ident.getSymbol()?.compilerSymbol;
  const resolved = symbol ? symbolIndex.get(symbol) : undefined;
  return {
    id: `${KIND_SHORT[TS_NODE_KIND_REFERENCE]}:${relPath}:${stableLocation(ident)}:${ident.getText()}`,
    kind: TS_NODE_KIND_REFERENCE,
    name: ident.getText(),
    path: relPath,
    attributes: {
      language: "typescript",
      kind: KIND_ATTR[TS_NODE_KIND_REFERENCE]!,
      referenced_name: ident.getText(),
      enclosing_declaration: enclosingDeclarationName(ident),
      resolved_node_id: resolved?.nodeId ?? "",
      resolved_name: resolved?.name ?? "",
      ...sourceRangeAttrs(ident),
    },
    leading_comments: [],
  };
}

function enclosingDeclarationName(node: Node): string {
  const executableOwner = node.getFirstAncestor((a) =>
    Node.isFunctionDeclaration(a) ||
    Node.isClassDeclaration(a) ||
    Node.isMethodDeclaration(a),
  );
  if (executableOwner) {
    return executableOwner.getName() ?? "";
  }
  const variableOwner = node.getFirstAncestor((a) => Node.isVariableDeclaration(a) && isFunctionLikeVar(a));
  if (!variableOwner || !Node.isVariableDeclaration(variableOwner)) return "";
  return variableOwner.getName();
}

function isDeclarationName(ident: import("ts-morph").Identifier): boolean {
  const parent = ident.getParent();
  return (
    (Node.isFunctionDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isClassDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isInterfaceDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isTypeAliasDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isVariableDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isEnumDeclaration(parent) && parent.getNameNode() === ident) ||
    (Node.isMethodDeclaration(parent) && parent.getNameNode() === ident)
  );
}

function isImportOrExportIdentifier(ident: import("ts-morph").Identifier): boolean {
  return ident.getFirstAncestor((a) =>
    Node.isImportDeclaration(a) || Node.isImportSpecifier(a) || Node.isExportDeclaration(a) || Node.isExportSpecifier(a),
  ) !== undefined;
}

function isPropertyAccessName(ident: import("ts-morph").Identifier): boolean {
  const parent = ident.getParent();
  return Node.isPropertyAccessExpression(parent) && parent.getNameNode() === ident;
}

function isJsxTagIdentifier(ident: import("ts-morph").Identifier): boolean {
  const parent = ident.getParent();
  const kind = parent.getKind();
  return (
    kind === SyntaxKind.JsxOpeningElement ||
    kind === SyntaxKind.JsxClosingElement ||
    kind === SyntaxKind.JsxSelfClosingElement
  );
}

function isObjectLiteralPropertyName(ident: import("ts-morph").Identifier): boolean {
  const parent = ident.getParent();
  return Node.isPropertyAssignment(parent) && parent.getNameNode() === ident;
}

function sourceRangeAttrs(node: Node): Record<string, string> {
  const sf = node.getSourceFile();
  const start = sf.getLineAndColumnAtPos(node.getStart());
  const end = sf.getLineAndColumnAtPos(node.getEnd());
  return {
    start_line: String(start.line),
    start_column: String(start.column),
    end_line: String(end.line),
    end_column: String(end.column),
  };
}

function stableLocation(node: Node): string {
  const start = node.getSourceFile().getLineAndColumnAtPos(node.getStart());
  return `${start.line}:${start.column}`;
}

function sanitizeId(value: string): string {
  return value.replace(/[^A-Za-z0-9_$.-]+/g, "_");
}

function compareStable(a: string, b: string): number {
  if (a < b) return -1;
  if (a > b) return 1;
  return 0;
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

// ResolveResult tells the caller both the edge target and whether the
// specifier actually resolved on disk. A null target means the
// specifier is external (bare) and should be skipped entirely — no edge,
// no warning. A non-null target with `unresolved: true` means we emit a
// best-effort (dangling) edge AND a WK_UNRESOLVED_IMPORT warning.
interface ResolveResult {
  target: string | null;
  unresolved: boolean;
}

function resolveSpecifier(
  rootDir: string,
  sf: SourceFile,
  spec: string,
): ResolveResult {
  // Only resolve relative specifiers in v1 — bare specifiers (e.g. "react")
  // are external and we don't emit nodes for them.
  if (!spec.startsWith(".") && !spec.startsWith("/")) {
    return { target: null, unresolved: false };
  }
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
      return { target: `ts_module:${rel}`, unresolved: false };
    }
  }
  // Couldn't resolve on disk; still emit a pointer using the literal path
  // (relative to the importing file's dir) so consumers see the unresolved
  // edge, and flag it as unresolved so the caller emits a warning.
  const guess = relativize(rootDir, path.resolve(fromDir, spec));
  return { target: `ts_module:${guess}`, unresolved: true };
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
