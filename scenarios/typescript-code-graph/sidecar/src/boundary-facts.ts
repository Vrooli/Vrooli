import * as path from "node:path";

import { Node, Project, SyntaxKind } from "ts-morph";

/** Version of the stable boundary-facts wire shape. */
export const BOUNDARY_FACTS_VERSION = 1 as const;

export type BoundaryTarget =
  | { kind: "scenario"; name: string }
  | { kind: "unknown"; name: null };

export interface BoundaryFact {
  version: typeof BOUNDARY_FACTS_VERSION;
  source: { path: string; line: number; column: number };
  boundaryKind: "embedded_http_request";
  target: BoundaryTarget;
  transport: "rest";
  addressing: "embedded_path" | "dynamic_embedded_path";
  evidence: { syntax: string; text: string };
}

export interface BoundaryFactsInput {
  /** Already-created parser project. No config discovery or subprocesses occur. */
  project: Project;
  /** Paths, absolute or relative to rootDir, whose facts are requested. */
  files: readonly string[];
  /** Root used to produce stable, portable source paths. */
  rootDir: string;
  /** Optional scenario identity override, primarily for focused tests. */
  scenarioName?: string;
}

const EMBEDDED_SCENARIO_PATH = /^\/embedded\/([a-z0-9][a-z0-9-]*)(?:\/|$)/i;

/**
 * Extract syntax-backed outbound scenario boundary facts from caller-selected
 * source files. Facts describe only explicit embedded fetches; they do not
 * infer targets from comments, prose, variable names, resolver calls, or
 * wrapper/import relationships. Resolver and alternate sink coverage remains
 * an explicit gap until provenance can be established reliably.
 */
export function extractBoundaryFacts(input: BoundaryFactsInput): BoundaryFact[] {
  const root = path.resolve(input.rootDir);
  const wanted = new Set(input.files.map((file) => path.resolve(root, file)));
  const sourceFiles = input.project
    .getSourceFiles()
    .filter((sourceFile) => wanted.has(path.resolve(sourceFile.getFilePath())))
    .sort((a, b) => portablePath(a.getFilePath(), root).localeCompare(portablePath(b.getFilePath(), root)));

  const facts: BoundaryFact[] = [];
  for (const sourceFile of sourceFiles) {
    const sourcePath = portablePath(sourceFile.getFilePath(), root);
    const sourceScenario = input.scenarioName ?? scenarioFromPath(sourcePath);
    for (const call of sourceFile.getDescendantsOfKind(SyntaxKind.CallExpression)) {
      const expression = call.getExpression();
      // Restrict this boundary to the global-style fetch call. A property
      // method named fetch may be an unrelated local API with different
      // semantics and is not evidence of a scenario HTTP request.
      if (Node.isIdentifier(expression) && expression.getText() === "fetch") {
        const argument = call.getArguments()[0];
        if (!argument || !Node.isExpression(argument) || isLocallyBound(expression)) continue;
        const prefix = embeddedPathPrefix(argument);
        if (prefix === null) continue;
        const match = EMBEDDED_SCENARIO_PATH.exec(prefix);
        if (match) {
          const targetName = match[1]!;
          if (sourceScenario && targetName.toLowerCase() === sourceScenario.toLowerCase()) continue;
          facts.push(makeFact(sourceFile, sourcePath, call, {
            boundaryKind: "embedded_http_request",
            target: { kind: "scenario", name: targetName },
            addressing: "embedded_path",
            syntax: Node.isStringLiteral(argument) || Node.isNoSubstitutionTemplateLiteral(argument)
              ? "fetch_literal_url"
              : "fetch_dynamic_url_with_literal_target",
            evidenceText: argument.getText(),
          }));
          continue;
        }
        if (prefix === "/embedded/" || prefix.startsWith("/embedded/")) {
          facts.push(makeFact(sourceFile, sourcePath, call, {
            boundaryKind: "embedded_http_request",
            target: { kind: "unknown", name: null },
            addressing: "dynamic_embedded_path",
            syntax: "fetch_dynamic_embedded_url",
            evidenceText: argument.getText(),
          }));
        }
        continue;
      }

    }
  }
  return facts.sort((a, b) =>
    a.source.path.localeCompare(b.source.path) ||
    a.source.line - b.source.line ||
    a.source.column - b.source.column ||
    a.boundaryKind.localeCompare(b.boundaryKind) ||
    (a.target.kind === "scenario" ? a.target.name : "").localeCompare(b.target.kind === "scenario" ? b.target.name : ""),
  );
}

/** Returns a syntactically known string prefix, or null when no prefix is provable. */
function embeddedPathPrefix(expression: import("ts-morph").Expression): string | null {
  if (Node.isStringLiteral(expression) || Node.isNoSubstitutionTemplateLiteral(expression)) {
    return expression.getLiteralText();
  }
  if (Node.isTemplateExpression(expression)) {
    let prefix = expression.getHead().getLiteralText();
    for (const span of expression.getTemplateSpans()) {
      const value = staticStringValue(span.getExpression());
      if (value === null) return prefix;
      prefix += value + span.getLiteral().getLiteralText();
    }
    return prefix;
  }
  if (Node.isBinaryExpression(expression) && expression.getOperatorToken().getText() === "+") {
    const left = embeddedPathPrefix(expression.getLeft());
    if (left === null) return null;
    const right = embeddedPathPrefix(expression.getRight());
    return right === null ? left : left + right;
  }
  if (Node.isCallExpression(expression) && expression.getArguments().length === 1) {
    const callee = expression.getExpression();
    const argument = expression.getArguments()[0];
    if (Node.isIdentifier(callee) && callee.getText() === "encodeURIComponent" &&
        !hasLexicalBinding(callee, "encodeURIComponent") && argument && Node.isExpression(argument)) {
      const value = staticStringValue(argument);
      if (value !== null) return encodeURIComponent(value);
    }
  }
  return null;
}

function staticStringValue(expression: import("ts-morph").Expression): string | null {
  if (Node.isStringLiteral(expression) || Node.isNoSubstitutionTemplateLiteral(expression)) return expression.getLiteralText();
  if (Node.isCallExpression(expression) && expression.getArguments().length === 1) {
    const callee = expression.getExpression();
    const argument = expression.getArguments()[0];
    if (Node.isIdentifier(callee) && callee.getText() === "encodeURIComponent" &&
        !hasLexicalBinding(callee, "encodeURIComponent") && argument && Node.isExpression(argument) &&
        (Node.isStringLiteral(argument) || Node.isNoSubstitutionTemplateLiteral(argument))) {
      return encodeURIComponent(argument.getLiteralText());
    }
  }
  return null;
}

function isLocallyBound(identifier: import("ts-morph").Identifier): boolean {
  return hasLexicalBinding(identifier, identifier.getText());
}

function hasLexicalBinding(identifier: import("ts-morph").Identifier, name: string): boolean {
  // Inspect syntax instead of asking the TypeScript checker for a symbol. A
  // repository-wide Project may contain thousands of unrelated tsconfigs and
  // asking for one symbol can construct a multi-gigabyte checker Program.
  const sourceFile = identifier.getSourceFile();
  let current: Node | undefined = identifier.getParent();
  while (current) {
    if (Node.isFunctionLikeDeclaration(current)) {
      if (current.getParameters().some((parameter) => bindingNameContains(parameter.getNameNode(), name)) ||
          functionDeclaresVar(current, name)) return true;
    }
    if (Node.isCatchClause(current) && current.getVariableDeclaration() &&
        bindingNameContains(current.getVariableDeclaration()!.getNameNode(), name)) {
      return true;
    }
    if (Node.isBlock(current) && blockDeclaresName(current, name)) return true;
    if ((Node.isForStatement(current) || Node.isForInStatement(current) || Node.isForOfStatement(current)) &&
        variableListDeclaresName(current.getInitializer(), name)) return true;
    if (Node.isCaseBlock(current) && caseBlockDeclaresName(current, name)) return true;
    if (Node.isSourceFile(current)) {
      if (sourceFileImportsName(current, name) || blockDeclaresName(current, name) || varDeclaredWithin(current, name)) return true;
      return false;
    }
    current = current.getParent();
  }
  return false;
}

function blockDeclaresName(block: import("ts-morph").Block | import("ts-morph").SourceFile, name: string): boolean {
  return block.getStatements().some((statement) => {
    if (Node.isVariableStatement(statement)) {
      return variableListDeclaresName(statement.getDeclarationList(), name);
    }
    if (Node.isFunctionDeclaration(statement) || Node.isClassDeclaration(statement)) {
      return statement.getName() === name;
    }
    return false;
  });
}

function variableListDeclaresName(node: Node | undefined, name: string): boolean {
  return !!node && Node.isVariableDeclarationList(node) && node.getDeclarations().some((declaration) =>
    bindingNameContains(declaration.getNameNode(), name),
  );
}

function functionDeclaresVar(fn: Node, name: string): boolean {
  return varDeclaredWithin(fn, name);
}

function varDeclaredWithin(scope: Node, name: string): boolean {
  return scope.getDescendantsOfKind(SyntaxKind.VariableDeclaration).some((declaration) => {
    if (!bindingNameContains(declaration.getNameNode(), name)) return false;
    const declarationList = declaration.getFirstAncestorByKind(SyntaxKind.VariableDeclarationList);
    if (declarationList?.getDeclarationKind() !== "var") return false;
    let parent: Node | undefined = declaration.getParent();
    while (parent && parent !== scope) {
      if (Node.isFunctionLikeDeclaration(parent) || Node.isModuleDeclaration(parent)) return false;
      parent = parent.getParent();
    }
    return parent === scope;
  });
}

function bindingNameContains(nameNode: Node, name: string): boolean {
  if (Node.isIdentifier(nameNode)) return nameNode.getText() === name;
  if (Node.isObjectBindingPattern(nameNode) || Node.isArrayBindingPattern(nameNode)) {
    return nameNode.getElements().some((element) => Node.isBindingElement(element) && bindingNameContains(element.getNameNode(), name));
  }
  return false;
}

function caseBlockDeclaresName(block: import("ts-morph").CaseBlock, name: string): boolean {
  return block.getClauses().some((clause) => clause.getStatements().some((statement) => {
    if (Node.isVariableStatement(statement)) return variableListDeclaresName(statement.getDeclarationList(), name);
    return (Node.isFunctionDeclaration(statement) || Node.isClassDeclaration(statement)) && statement.getName() === name;
  })) || block.getClauses().some((clause) => clause.getDescendantsOfKind(SyntaxKind.VariableDeclaration).some((declaration) => {
    if (!bindingNameContains(declaration.getNameNode(), name)) return false;
    const declarationList = declaration.getFirstAncestorByKind(SyntaxKind.VariableDeclarationList);
    if (declarationList?.getDeclarationKind() !== "var") return false;
    let parent: Node | undefined = declaration.getParent();
    while (parent && parent !== clause) {
      if (Node.isFunctionLikeDeclaration(parent)) return false;
      parent = parent.getParent();
    }
    return parent === clause;
  }));
}

function sourceFileImportsName(sourceFile: import("ts-morph").SourceFile, name: string): boolean {
  return sourceFile.getImportDeclarations().some((declaration) => {
    const clause = declaration.getImportClause();
    if (!clause) return false;
    if (clause.getDefaultImport()?.getText() === name) return true;
    const namedBindings = clause.getNamedBindings();
    if (Node.isNamespaceImport(namedBindings) && namedBindings.getName() === name) return true;
    return Node.isNamedImports(namedBindings) && namedBindings.getElements().some((element) =>
      (element.getAliasNode()?.getText() ?? element.getName()) === name,
    );
  });
}

function scenarioFromPath(sourcePath: string): string | undefined {
  return /^scenarios\/([^/]+)(?:\/|$)/.exec(sourcePath)?.[1];
}

function makeFact(
  sourceFile: ReturnType<Project["getSourceFileOrThrow"]>,
  sourcePath: string,
  call: import("ts-morph").CallExpression,
  values: Pick<BoundaryFact, "boundaryKind" | "target" | "addressing"> & { syntax: string; evidenceText: string },
): BoundaryFact {
  const location = sourceFile.getLineAndColumnAtPos(call.getStart());
  return {
    version: BOUNDARY_FACTS_VERSION,
    source: { path: sourcePath, line: location.line, column: location.column },
    boundaryKind: values.boundaryKind,
    target: values.target,
    transport: "rest",
    addressing: values.addressing,
    evidence: { syntax: values.syntax, text: values.evidenceText },
  };
}

function portablePath(filePath: string, rootDir: string): string {
  return path.relative(rootDir, path.resolve(filePath)).split(path.sep).join("/");
}
