// Conservative literal prop-to-host-attribute evidence. Unknown expressions,
// wrappers and spreads never become proof. This is source reachability, not a
// claim that a conditional branch is visible at runtime.
import { createRequire } from "node:module";
import { readFileSync, existsSync, statSync } from "node:fs";
import { dirname, resolve, join } from "node:path";
import { fileURLToPath } from "node:url";
import { resolveLibrarySpecifier } from "./resolve-specifier.mjs";
const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
const ts = createRequire(join(repo, "scenarios/react-component-library/ui/package.json"))("typescript");

export async function forwardedDOMBindings(files, { libraryRoot = join(repo, "scenarios/react-component-library/library") } = {}) {
  const modules = new Map();
  const resolutions = new Map();
  function module(file) {
    if (modules.has(file)) return modules.get(file);
    if (modules.size >= 1000 || !existsSync(file) || statSync(file).size > 1024 * 1024) return null;
    const ast = ts.createSourceFile(file, readFileSync(file, "utf8"), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    modules.set(file, ast);
    return ast;
  }
  async function imported(file, specifier) {
    const key = `${file}:${specifier}`;
    if (resolutions.has(key)) return resolutions.get(key);
    let result = null;
    if (specifier.startsWith(".")) {
      const base = resolve(dirname(file), specifier).replace(/\.jsx?$/, "");
      result = [base, `${base}.tsx`, `${base}.ts`, `${base}.jsx`, `${base}.js`, join(base, "index.tsx"), join(base, "index.ts")]
        .find(path => existsSync(path) && statSync(path).isFile()) ?? null;
    } else if (specifier.startsWith("@vrooli/react-component-library/")) {
      try { result = (await resolveLibrarySpecifier(specifier, { libraryRoot })).sourcePath; } catch { /* unavailable is not evidence */ }
    }
    resolutions.set(key, result);
    return result;
  }
  function unwrap(node) {
    while (node && (ts.isParenthesizedExpression(node) || ts.isAsExpression(node) || ts.isNonNullExpression(node))) node = node.expression;
    return node;
  }
  async function callable(file, name, seen = new Set()) {
    const key = `${file}:${name}`;
    if (seen.has(key) || seen.size >= 12) return null;
    seen = new Set([...seen, key]);
    const ast = module(file);
    if (!ast) return null;
    for (const statement of ast.statements) {
      if (ts.isImportDeclaration(statement) && statement.importClause) {
        const clause = statement.importClause;
        let exported = clause.name?.text === name ? "default" : null;
        if (clause.namedBindings && ts.isNamedImports(clause.namedBindings)) {
          const item = clause.namedBindings.elements.find(item => item.name.text === name && !item.isTypeOnly);
          if (item) exported = item.propertyName?.text ?? item.name.text;
        }
        if (exported) {
          const target = await imported(file, statement.moduleSpecifier.text);
          return target ? callable(target, exported, seen) : null;
        }
      }
      if (ts.isExportDeclaration(statement) && statement.moduleSpecifier && statement.exportClause && ts.isNamedExports(statement.exportClause)) {
        const item = statement.exportClause.elements.find(item => item.name.text === name);
        if (item) {
          const target = await imported(file, statement.moduleSpecifier.text);
          return target ? callable(target, item.propertyName?.text ?? item.name.text, seen) : null;
        }
      }
      if (ts.isFunctionDeclaration(statement) && (statement.name?.text === name || name === "default" && statement.modifiers?.some(m => m.kind === ts.SyntaxKind.DefaultKeyword))) return { file, fn: statement };
      if (ts.isVariableStatement(statement)) for (const declaration of statement.declarationList.declarations) {
        if (declaration.name.getText(ast) !== name) continue;
        let fn = unwrap(declaration.initializer);
        if (fn && ts.isCallExpression(fn)) {
          // React's transparent wrappers and the library's class seam preserve
          // ordinary props. ClassMerge is resolved through its source authority;
          // unrelated functions with the same name do not qualify.
          const wrapper = fn.expression.getText(ast);
          let transparent = false;
          for (const imp of ast.statements.filter(ts.isImportDeclaration)) {
            const bindings = imp.importClause?.namedBindings;
            if (bindings && ts.isNamedImports(bindings)) for (const item of bindings.elements) {
              if (item.name.text !== wrapper) continue;
              const original = item.propertyName?.text ?? item.name.text;
              if (imp.moduleSpecifier.text === "react" && ["memo", "forwardRef"].includes(original)) transparent = true;
              if (original === "withClassName" && imp.moduleSpecifier.text.startsWith("@vrooli/react-component-library/ClassMerge/")) transparent = true;
            }
          }
          if (!transparent) return null;
          fn = unwrap(fn.arguments[0]);
        }
        if (fn && (ts.isArrowFunction(fn) || ts.isFunctionExpression(fn))) return { file, fn };
      }
    }
    return null;
  }
  function value(node, env) {
    node = unwrap(node);
    if (!node) return null;
    if (ts.isStringLiteral(node)) return { text: node.text, forwarded: false };
    if (ts.isIdentifier(node)) return env.get(node.text) ?? null;
    if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.expression)) return env.get(`${node.expression.text}.${node.name.text}`) ?? null;
    return null;
  }
  function parameters(fn, props) {
    const env = new Map();
    const parameter = fn.parameters[0];
    if (!parameter) return env;
    if (ts.isIdentifier(parameter.name)) for (const [name, item] of props) env.set(`${parameter.name.text}.${name}`, item);
    if (ts.isObjectBindingPattern(parameter.name)) for (const binding of parameter.name.elements) {
      if (binding.dotDotDotToken || !ts.isIdentifier(binding.name)) continue;
      const name = binding.propertyName?.text ?? binding.name.text;
      // An explicitly unknown prop must not activate a default.
      const item = props.has(name) ? props.get(name) : value(binding.initializer, new Map());
      env.set(binding.name.text, item);
    }
    function invalidate(node) {
      if (ts.isBinaryExpression(node) && node.operatorToken.kind >= ts.SyntaxKind.FirstAssignment && node.operatorToken.kind <= ts.SyntaxKind.LastAssignment) {
        env.delete(node.left.getText());
      }
      if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name)) env.delete(node.name.text);
      if (node !== fn.body && ts.isFunctionLike(node)) return;
      ts.forEachChild(node, invalidate);
    }
    if (fn.body) invalidate(fn.body);
    return env;
  }
  const output = new Map();
  for (const root of files) {
    const results = new Map();
    let expansions = 0;
    async function walk(node, file, env, chain, depth, rootWalk = false) {
      if (!node || depth > 8 || expansions > 400) return;
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        const props = new Map();
        let unknownSpread = false;
        for (const attribute of node.attributes.properties) {
          if (ts.isJsxSpreadAttribute(attribute)) { props.clear(); unknownSpread = true; continue; }
          const expression = attribute.initializer && ts.isJsxExpression(attribute.initializer) ? attribute.initializer.expression : attribute.initializer;
          const item = value(expression, env);
          props.set(attribute.name.text, item && { ...item, forwarded: rootWalk || item.forwarded });
        }
        const tag = node.tagName.getText();
        const dynamicTag = env.get(tag);
        const host = /^[a-z][a-z0-9-]*$/.test(tag) || dynamicTag && /^[a-z][a-z0-9-]*$/.test(dynamicTag.text);
        if (host && !rootWalk) {
          for (const [attribute, item] of props) if (item?.forwarded && attribute.startsWith("data-")) {
            results.set(`${attribute}:${item.text}`, { attribute, value: item.text, via: [...chain, file] });
          }
        } else if (!host && [...props.values()].some(item => item?.forwarded)) {
          const target = await callable(file, tag);
          if (target) {
            expansions++;
            const nestedEnv = parameters(target.fn, props);
            // Unknown spreads can supply an 'as' prop; a default host tag is
            // then unproved. Explicit props after the spread remain usable.
            if (unknownSpread) for (const binding of target.fn.parameters[0]?.name.elements ?? []) {
              const propName = binding.propertyName?.text ?? binding.name.text;
              if (!props.has(propName)) nestedEnv.delete(binding.name.text);
            }
            await walk(target.fn.body, target.file, nestedEnv, [...chain, file], depth + 1);
          }
        }
      }
      const children = [];
      ts.forEachChild(node, child => { children.push(child); });
      for (const child of children) {
        if (!rootWalk && ts.isFunctionLike(child)) continue;
        await walk(child, file, env, chain, depth, rootWalk);
      }
    }
    await walk(module(root), root, new Map(), [], 0, true);
    output.set(root, [...results.values()]);
  }
  return output;
}
