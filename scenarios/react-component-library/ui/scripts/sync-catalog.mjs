/**
 * Codegen: rebuild `src/i18n/locales/en.json` from the strings the UI actually renders.
 *
 * The catalog used to be authored by hand and drifted until every workbench
 * string rendered from its inline `defaultValue` while en.json held 520 keys
 * that no file referenced. This script makes usage the source of truth:
 *
 *   - every `t("key", { defaultValue })` contributes `key → defaultValue`
 *   - every `t(strings.a.b)` / `strings.a.b` accessor contributes the key, with
 *     its value carried over from the previous catalog
 *   - keys with no callsite are dropped from en.json
 *   - ja/ar keep their translations for keys that survive; untranslated keys
 *     fall back to English at runtime (see `i18n/index.ts`)
 *
 * Run modes:
 *   node scripts/sync-catalog.mjs           # write en.json (and prune ja/ar)
 *   node scripts/sync-catalog.mjs --check   # exit 1 if en.json is out of date
 *
 * It is deliberately AST-based rather than regex-based: `defaultValue` can
 * contain quotes, and comments mention `t("...")` shapes that are not calls.
 */
import { readFileSync, readdirSync, statSync, writeFileSync, existsSync } from "node:fs";
import { join, relative, extname } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const UI_ROOT = join(fileURLToPath(new URL("..", import.meta.url)));
const SRC = join(UI_ROOT, "src");
const EN_PATH = join(SRC, "i18n/locales/en.json");
const LOCALES = ["ja", "ar"];

const SKIP_DIRS = new Set(["node_modules", "dist", "coverage", ".vite"]);
const SKIP_FILES = new Set(["strings.generated.ts"]);

const isSentinel = (segment) => segment.startsWith("_");

const flatten = (catalog, prefix = "", out = {}) => {
  for (const [key, value] of Object.entries(catalog)) {
    if (isSentinel(key)) continue;
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") out[path] = value;
    else if (value && typeof value === "object" && !Array.isArray(value)) flatten(value, path, out);
    else throw new Error(`Unsupported catalog value at ${path}`);
  }
  return out;
};

/** Rebuild a nested object from dotted keys, sorted for stable diffs. */
const nest = (flat) => {
  const root = {};
  for (const key of Object.keys(flat).sort()) {
    const segments = key.split(".");
    let cursor = root;
    for (const segment of segments.slice(0, -1)) {
      cursor[segment] ??= {};
      cursor = cursor[segment];
    }
    cursor[segments.at(-1)] = flat[key];
  }
  return root;
};

const literal = (node) => {
  if (!node) return undefined;
  if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) return node.text;
  return undefined;
};

const propertyOf = (object, name) =>
  object.properties.find(
    (property) => ts.isPropertyAssignment(property) && ts.isIdentifier(property.name) && property.name.text === name,
  );

/**
 * `pluralDefault(count, singular, plural)` predates catalog plurals. Reading
 * both branches here lets the catalog capture the copy before the callsite
 * switches to i18next's `count` resolution.
 */
const pluralBranches = (node) => {
  if (!node || !ts.isCallExpression(node)) return undefined;
  if (!ts.isIdentifier(node.expression) || node.expression.text !== "pluralDefault") return undefined;
  const singular = literal(node.arguments[1]);
  const plural = literal(node.arguments[2]);
  return singular !== undefined && plural !== undefined ? { singular, plural } : undefined;
};

/**
 * Static label tables pair a catalog key with its English text:
 *   { key: "workbench.studio.token.bg", label: "Background" }
 *   { key: "workbench.nav.home", fallback: "Home" }
 *   { labelKey: "workbench.studio.add.text", label: "Text" }
 * Reading the pair keeps map-based callsites translatable without asking them
 * to duplicate the English text at the callsite.
 */
const pairFromObject = (object) => {
  const pairs = [];
  const keyProperty = propertyOf(object, "key") ?? propertyOf(object, "labelKey");
  const textProperty = propertyOf(object, "label") ?? propertyOf(object, "fallback");
  if (keyProperty && textProperty) {
    const key = literal(keyProperty.initializer);
    const text = literal(textProperty.initializer);
    if (key && text) pairs.push({ key, text });
  }
  return pairs;
};

const accessorPath = (node) => {
  if (!node || !ts.isPropertyAccessExpression(node)) return undefined;
  const parts = [];
  let cursor = node;
  while (ts.isPropertyAccessExpression(cursor)) {
    parts.unshift(cursor.name.text);
    cursor = cursor.expression;
  }
  return ts.isIdentifier(cursor) && cursor.text === "strings" ? parts.join(".") : undefined;
};

const collect = (file, source) => {
  const found = new Map();
  const pairs = new Map();
  const accessors = new Set();
  const problems = [];
  const scriptKind = file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const tree = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, scriptKind);

  const visit = (node) => {
    if (ts.isObjectLiteralExpression(node)) {
      for (const pair of pairFromObject(node)) pairs.set(pair.key, pair.text);
    }

    if (ts.isPropertyAccessExpression(node)) {
      const path = accessorPath(node);
      if (path) accessors.add(path);
    }

    if (ts.isCallExpression(node)) {
      const callee = node.expression;
      const named =
        (ts.isIdentifier(callee) && callee.text === "t") ||
        (ts.isPropertyAccessExpression(callee) &&
          callee.name.text === "t" &&
          ts.isIdentifier(callee.expression) &&
          (callee.expression.text === "i18n" || callee.expression.text === "translate"));

      if (named) {
        const first = node.arguments[0];
        const key = literal(first) ?? accessorPath(first);
        const options = node.arguments[1];
        let defaultValue;
        if (key && options && ts.isObjectLiteralExpression(options)) {
          for (const property of options.properties) {
            if (
              ts.isPropertyAssignment(property) &&
              ts.isIdentifier(property.name) &&
              property.name.text === "defaultValue"
            ) {
              defaultValue = literal(property.initializer);
              const plural = pluralBranches(property.initializer);
              if (plural && key) {
                found.set(`${key}_one`, plural.singular);
                found.set(`${key}_other`, plural.plural);
                found.set(key, plural.plural);
              }
              if (defaultValue === undefined && !plural) {
                const { line } = tree.getLineAndCharacterOfPosition(property.getStart(tree));
                problems.push({
                  file: relative(UI_ROOT, file),
                  line: line + 1,
                  reason: "defaultValue is not a string literal",
                });
              }
            }
          }
        }
        if (!key) {
          const { line } = tree.getLineAndCharacterOfPosition(node.getStart(tree));
          // Dynamic keys are a lint/typing hazard; keep them out of the catalog.
          if (ts.isTemplateExpression(first ?? node)) {
            problems.push({
              file: relative(UI_ROOT, file),
              line: line + 1,
              reason: "t() key is a template literal; use a static key map",
            });
          }
        } else if (defaultValue !== undefined) {
          const previous = found.get(key);
          if (previous !== undefined && previous !== defaultValue) {
            const { line } = tree.getLineAndCharacterOfPosition(node.getStart(tree));
            problems.push({
              file: relative(UI_ROOT, file),
              line: line + 1,
              reason: `key "${key}" has two different defaultValue texts`,
            });
          }
          found.set(key, defaultValue);
        } else {
          accessors.add(key);
        }
      }
    }

    ts.forEachChild(node, visit);
  };
  visit(tree);
  return { found, pairs, accessors, problems };
};

const collectSourceFiles = (dir) => {
  const out = [];
  for (const entry of readdirSync(dir)) {
    if (SKIP_DIRS.has(entry)) continue;
    const full = join(dir, entry);
    if (statSync(full).isDirectory()) out.push(...collectSourceFiles(full));
    else if ([".ts", ".tsx"].includes(extname(entry)) && !SKIP_FILES.has(entry)) out.push(full);
  }
  return out;
};

const readCatalog = (path) => (existsSync(path) ? JSON.parse(readFileSync(path, "utf8")) : {});

const main = () => {
  const check = process.argv.includes("--check");
  const existingEn = flatten(readCatalog(EN_PATH));
  const literalDefaults = new Map();
  const accessors = new Set();
  const problems = [];

  for (const file of collectSourceFiles(SRC)) {
    const result = collect(file, readFileSync(file, "utf8"));
    problems.push(...result.problems);
    for (const [key, value] of result.pairs) {
      if (literalDefaults.has(key) && literalDefaults.get(key) !== value) {
        problems.push({ file: relative(UI_ROOT, file), line: 0, reason: `key "${key}" label table disagrees with its callsite default` });
      }
      if (!literalDefaults.has(key)) literalDefaults.set(key, value);
    }
    for (const [key, value] of result.found) {
      const previous = literalDefaults.get(key);
      if (previous !== undefined && previous !== value) {
        problems.push({ file: relative(UI_ROOT, file), line: 0, reason: `key "${key}" disagrees with an earlier file` });
      }
      literalDefaults.set(key, value);
    }
    for (const key of result.accessors) accessors.add(key);
  }

  // `strings.errors` (a namespace object) is not a leaf key; drop any path
  // that is also a prefix of a longer accessor.
  const accessorList = [...accessors].filter(
    (key) => ![...accessors].some((other) => other !== key && other.startsWith(`${key}.`)),
  );
  const next = {};
  const missing = [];
  for (const [key, value] of literalDefaults) next[key] = value;
  for (const key of accessorList) {
    if (next[key] !== undefined) continue;
    if (existingEn[key] !== undefined) next[key] = existingEn[key];
    else missing.push(key);
  }
  for (const key of missing) {
    problems.push({ file: "src", line: 0, reason: `key "${key}" has no value: no defaultValue and no catalog entry` });
  }

  // Preserve CLDR variants of any key that survives. i18next resolves
  // `key_one` / `key_other` from `t(key, { count })`; the callsite only
  // names the base key, so the variants must travel with it.
  for (const key of Object.keys(existingEn)) {
    if (!/_(zero|one|two|few|many|other)$/.test(key)) continue;
    const base = key.replace(/_(zero|one|two|few|many|other)$/, "");
    if (next[base] !== undefined && next[key] === undefined) next[key] = existingEn[key];
  }

  if (problems.length > 0) {
    console.error(`sync-catalog: ${problems.length} problem(s)`);
    for (const problem of problems.slice(0, 40)) {
      console.error(`  ${problem.file}:${problem.line} ${problem.reason}`);
    }
    process.exit(1);
  }

  const previousEn = readCatalog(EN_PATH);
  const nextEn = nest(next);
  if (previousEn._comment) nextEn._comment = previousEn._comment;
  const serialized = `${JSON.stringify(nextEn, null, 2)}\n`;
  const current = existsSync(EN_PATH) ? readFileSync(EN_PATH, "utf8") : "";

  if (check) {
    if (current !== serialized) {
      console.error("sync-catalog: en.json is out of date (run `pnpm strings:catalog`)");
      process.exit(1);
    }
    console.log(`sync-catalog: ${Object.keys(next).length} keys, up to date`);
    return;
  }

  writeFileSync(EN_PATH, serialized);

  for (const locale of LOCALES) {
    const path = join(SRC, `i18n/locales/${locale}.json`);
    const catalog = readCatalog(path);
    const translated = flatten(catalog);
    const kept = {};
    for (const key of Object.keys(next)) {
      if (translated[key] !== undefined) kept[key] = translated[key];
    }
    writeFileSync(path, `${JSON.stringify(nest(kept), null, 2)}\n`);
  }

  console.log(
    `sync-catalog: ${Object.keys(next).length} keys written; dropped ${Object.keys(existingEn).length - Object.keys(next).length} orphans`,
  );
};

main();
