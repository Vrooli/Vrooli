import { existsSync } from "node:fs";
import { readdir } from "node:fs/promises";
import { dirname, extname, isAbsolute, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";

const AST_PATTERNS = [
  "import $A from $S",
  "import \"$S\"",
  "export { $$$A } from \"$S\"",
  "export type { $$$A } from \"$S\"",
  "export * from \"$S\"",
  "import(\"$S\")",
  "require(\"$S\")",
];

const JSX_PATTERNS = [
  "<$TAG $$$ATTRS>$$$CHILDREN</$TAG>",
  "<$TAG $$$ATTRS />",
];

const EXPORT_PATTERNS = [
  "export const $NAME = $$$VALUE",
  "export function $NAME($$$PARAMS) { $$$BODY }",
  "export type $NAME = $VALUE",
  "export interface $NAME { $$$BODY }",
];

const CALL_PATTERN = "$CALLEE($$$ARGS)";

const unquote = (value) => {
  const text = String(value ?? "").trim();
  if (text.length >= 2 && ((text.startsWith('"') && text.endsWith('"')) || (text.startsWith("'") && text.endsWith("'")))) {
    return text.slice(1, -1);
  }
  return /^[.@a-z0-9_/-]+$/i.test(text) ? text : "";
};

const astMatches = (root, language, pattern) => {
  const result = spawnSync("ast-grep", ["run", "--lang", language, "--pattern", pattern, "--json=stream", root], {
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (result.status !== 0 && result.stdout.trim() === "") {
    throw new Error(`ast-grep failed for ${language} pattern ${JSON.stringify(pattern)}:\n${result.stderr.trim()}`);
  }
  return result.stdout.split("\n").filter(Boolean).map((line) => JSON.parse(line));
};

export function scanModuleSpecifiers(root) {
  const absoluteRoot = resolve(root);
  const byFile = new Map();
  for (const language of ["typescript", "tsx"]) {
    for (const pattern of AST_PATTERNS) {
      for (const match of astMatches(absoluteRoot, language, pattern)) {
        const specifier = unquote(match.metaVariables?.single?.S?.text);
        if (!specifier) continue;
        const file = isAbsolute(match.file) ? resolve(match.file) : resolve(match.file);
        const values = byFile.get(file) ?? new Set();
        values.add(specifier);
        byFile.set(file, values);
      }
    }
  }
  return byFile;
}

const parseAttribute = (text) => {
  const match = String(text).trim().match(/^([A-Za-z_:][A-Za-z0-9_.:-]*)(?:\s*=\s*(.*))?$/s);
  if (!match) return null;
  return { name: match[1], value: match[2] ?? null };
};

// This is the shared structured source seam for TSX-shaped gates. Consumers
// receive facts from AST nodes instead of matching the authored source text.
// Keep the fact vocabulary generic so new gates do not grow component-specific
// parsers or regexes in the Go layer.
export function analyzeSourceFacts(root) {
  const absoluteRoot = resolve(root);
  const byFile = new Map();
  for (const pattern of JSX_PATTERNS) {
    for (const match of astMatches(absoluteRoot, "tsx", pattern)) {
      const file = resolve(match.file);
      const facts = byFile.get(file) ?? {
        jsxElements: new Set(),
        attributes: new Map(),
        literalValues: new Set(),
        elements: [],
        exports: new Set(),
        hookCalls: new Set(),
        calls: new Set(),
        inlineStyleElements: 0,
      };
      const tag = match.metaVariables?.single?.TAG?.text;
      if (tag) facts.jsxElements.add(tag);
      const elementAttributes = {};
      for (const rawAttribute of match.metaVariables?.multi?.ATTRS ?? []) {
        const attribute = parseAttribute(rawAttribute.text);
        if (!attribute) continue;
        const values = facts.attributes.get(attribute.name) ?? new Set();
        values.add(attribute.value);
        facts.attributes.set(attribute.name, values);
        (elementAttributes[attribute.name] ??= new Set()).add(attribute.value);
        if (attribute.value) facts.literalValues.add(attribute.value);
      }
      facts.elements.push({
        tag,
        attributes: Object.fromEntries(Object.entries(elementAttributes).map(([name, values]) => [name, [...values]])),
      });
      if ([...(elementAttributes.style ?? [])].some((value) => String(value ?? "").trim().startsWith("{{"))) {
        facts.inlineStyleElements += 1;
      }
      byFile.set(file, facts);
    }
  }
  for (const language of ["typescript", "tsx"]) {
    for (const pattern of EXPORT_PATTERNS) {
      for (const match of astMatches(absoluteRoot, language, pattern)) {
        const file = resolve(match.file);
        const facts = byFile.get(file) ?? {
          jsxElements: new Set(),
          attributes: new Map(),
          literalValues: new Set(),
          elements: [],
          exports: new Set(),
          hookCalls: new Set(),
          calls: new Set(),
          inlineStyleElements: 0,
        };
        const name = match.metaVariables?.single?.NAME?.text;
        if (name) facts.exports.add(name);
        byFile.set(file, facts);
      }
    }
    for (const match of astMatches(absoluteRoot, language, CALL_PATTERN)) {
      const callee = match.metaVariables?.single?.CALLEE?.text;
      if (!callee) continue;
      const file = resolve(match.file);
      const facts = byFile.get(file) ?? {
        jsxElements: new Set(),
        attributes: new Map(),
        literalValues: new Set(),
        elements: [],
        exports: new Set(),
        hookCalls: new Set(),
        calls: new Set(),
        inlineStyleElements: 0,
      };
      facts.calls.add(callee);
      if (/^use[A-Z]/.test(callee)) facts.hookCalls.add(callee);
      byFile.set(file, facts);
    }
  }
  return byFile;
}

export function serializeSourceFacts(root) {
  const factsByFile = analyzeSourceFacts(root);
  for (const [file, imports] of scanModuleSpecifiers(root)) {
    const facts = factsByFile.get(file) ?? {
      jsxElements: new Set(),
      attributes: new Map(),
      literalValues: new Set(),
      elements: [],
      exports: new Set(),
      hookCalls: new Set(),
      calls: new Set(),
      inlineStyleElements: 0,
    };
    facts.imports = imports;
    factsByFile.set(file, facts);
  }
  return [...factsByFile].sort(([left], [right]) => left.localeCompare(right)).map(([file, facts]) => ({
    file,
    elements: facts.elements,
    jsxElements: [...facts.jsxElements].sort(),
    attributes: Object.fromEntries([...facts.attributes]
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([name, values]) => [name, [...values].sort()])),
    literalValues: [...facts.literalValues].sort(),
    exports: [...facts.exports].sort(),
    hookCalls: [...facts.hookCalls].sort(),
    calls: [...(facts.calls ?? [])].sort(),
    inlineStyleElements: facts.inlineStyleElements ?? 0,
    imports: [...(facts.imports ?? [])].sort(),
  }));
}

const resolveRelativeModule = (fromFile, specifier) => {
  if (!specifier.startsWith(".")) return null;
  const base = resolve(dirname(fromFile), specifier);
  const extension = extname(base);
  const sourceBase = extension === ".js" || extension === ".jsx" ? base.slice(0, -extension.length) : base;
  const nonModuleAsset = [".css", ".json", ".svg"].includes(extension);
  const candidates = nonModuleAsset
    ? [base]
    : [base, `${sourceBase}.ts`, `${sourceBase}.tsx`, `${sourceBase}.js`, `${sourceBase}.jsx`, join(sourceBase, "index.ts"), join(sourceBase, "index.tsx")];
  return candidates.find((candidate) => existsSync(candidate)) ?? null;
};

export async function resolveVersionImports({ entryFile, versionRoot, specifiersByFile }) {
  const pending = [resolve(entryFile)];
  // Retention must see imports that are not reachable from the public entry
  // module. Stories and preview harnesses are package consumers too.
  const files = await readdir(resolve(versionRoot), { recursive: true, withFileTypes: true });
  for (const entry of files) {
    if (entry.isFile() && /\.(?:ts|tsx|js|jsx)$/.test(entry.name)) pending.push(resolve(entry.parentPath ?? versionRoot, entry.name));
  }
  const visited = new Set();
  const specifiers = new Set();
  while (pending.length > 0) {
    const file = pending.pop();
    if (visited.has(file)) continue;
    visited.add(file);
    for (const specifier of specifiersByFile.get(file) ?? []) {
      const sibling = resolveRelativeModule(file, specifier);
      if (sibling?.startsWith(`${resolve(versionRoot)}/`)) {
        pending.push(sibling);
      } else if (sibling) {
        specifiers.add(`file://${sibling}`);
      } else if (specifier.startsWith(".")) {
        throw new Error(`${file}: relative module ${JSON.stringify(specifier)} does not resolve inside ${resolve(versionRoot)}`);
      } else {
        specifiers.add(specifier);
      }
    }
  }
  return [...specifiers].sort();
}

if (process.argv[1] && process.argv[1].endsWith("resolve-imports.mjs") && process.argv[2] === "--facts") {
  console.log(JSON.stringify(serializeSourceFacts(process.argv[3])));
}
if (process.argv[1] && process.argv[1].endsWith("resolve-imports.mjs") && process.argv[2] === "--facts-root") {
  console.log(JSON.stringify(serializeSourceFacts(process.argv[3])));
}
