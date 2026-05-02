/**
 * ESLint rule: `strings/no-unused-keys`
 *
 * Walks `src/i18n/locales/en.json`, flattens it to dotted key paths, and
 * scans every `*.ts`/`*.tsx` file under `src/` for at least one usage of
 * each leaf key. Reports each orphan as a separate diagnostic.
 *
 * Why we walk the source tree manually instead of relying on ESLint's
 * per-file AST visit: ESLint visits one file at a time, so per-file
 * walks would miss usages that live elsewhere. We scan the whole tree
 * once per lint pass (memoized by `context.cwd`), then anchor every
 * diagnostic on `strings.generated.ts` so the rule reports exactly once
 * per run regardless of how many files ESLint lints.
 *
 * Why anchor on `strings.generated.ts` and not `en.json`: the template's
 * `eslint.config.js` has `files: ["**\/*.{ts,tsx}"]`, so ESLint never
 * visits `en.json`. `strings.generated.ts` IS already linted, exists
 * exactly once per scenario, and is the closest semantic mirror of the
 * catalog — it's the right anchor.
 *
 * Underscore-prefix sentinel skip: keys whose final segment starts with
 * `_` are sentinels (e.g., `_comment`) and never go through `t()`. They
 * are skipped here, in `scripts/gen-strings.mjs`, and in
 * `src/i18n/locales/locales.test.ts`. The convention is duplicated by
 * intent — each consumer documents its own skip.
 *
 * Plural CLDR variants (`refreshCount_one`, etc.) are NOT sentinels.
 * They are valid catalog leaves with their own usage signal: i18next
 * resolves them automatically via the base key, so a callsite of
 * `t(strings.health.refreshCount, { count })` covers all variants. We
 * treat any plural-suffixed key as "used iff its base key is used."
 */
import { readFileSync, readdirSync, statSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join, relative, extname } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const UI_ROOT = join(__dirname, "..");
const SRC_DIR = join(UI_ROOT, "src");
const CATALOG_PATH = join(UI_ROOT, "src/i18n/locales/en.json");

const ANCHOR_BASENAME = "strings.generated.ts";
const PLURAL_SUFFIX = /_(zero|one|two|few|many|other)$/;
const SOURCE_EXTS = new Set([".ts", ".tsx"]);
// Files we skip during the source-tree scan. The generated registry trivially
// "uses" every key (each leaf is the dotted path), so including it would
// always report zero orphans. The locale JSONs are themselves the catalog.
const SKIP_FILES = new Set([
  "strings.generated.ts",
]);
const SKIP_DIRS = new Set(["node_modules", "dist", "build", "coverage", ".vite"]);

const isSentinelSegment = (segment) => segment.startsWith("_");

const flattenKeys = (catalog, prefix = []) => {
  const out = [];
  for (const [key, value] of Object.entries(catalog)) {
    if (isSentinelSegment(key)) continue;
    const path = [...prefix, key];
    if (typeof value === "string") {
      out.push(path.join("."));
    } else if (value && typeof value === "object" && !Array.isArray(value)) {
      out.push(...flattenKeys(value, path));
    }
  }
  return out;
};

const collectSourceFiles = (root) => {
  const out = [];
  const walk = (dir) => {
    let entries;
    try {
      entries = readdirSync(dir);
    } catch {
      return;
    }
    for (const entry of entries) {
      const full = join(dir, entry);
      let stat;
      try {
        stat = statSync(full);
      } catch {
        continue;
      }
      if (stat.isDirectory()) {
        if (SKIP_DIRS.has(entry)) continue;
        walk(full);
        continue;
      }
      if (SKIP_FILES.has(entry)) continue;
      if (SOURCE_EXTS.has(extname(entry))) out.push(full);
    }
  };
  walk(root);
  return out;
};

// Memoize the (catalog, source scan) pair across multiple rule instances /
// multiple files in the same lint run. Keyed on UI_ROOT since the rule's
// path layout pins it; a different scenario root would import a different
// copy of this module (each scenario has its own eslint-rules/).
let cached = null;

const computeOrphans = () => {
  if (cached) return cached;
  if (!existsSync(CATALOG_PATH)) {
    cached = { orphans: [], catalogMissing: true };
    return cached;
  }
  const catalog = JSON.parse(readFileSync(CATALOG_PATH, "utf-8"));
  const allKeys = flattenKeys(catalog);
  // Strip plural suffix to a base key — usage is reported iff the base form
  // is referenced anywhere. i18next resolves variants from the base call.
  const basesNeeded = new Set(allKeys.map((k) => k.replace(PLURAL_SUFFIX, "")));

  const sourceFiles = collectSourceFiles(SRC_DIR);
  const haystack = sourceFiles
    .map((f) => readFileSync(f, "utf-8"))
    .join("\n");

  const orphans = [];
  for (const base of basesNeeded) {
    // Two usage signals: the dotted literal "feature.key" or any path-segment
    // that walks the registry to the same leaf (`strings.feature.key`).
    // The dotted-string check covers `t("feature.key")`, `getByText(strings.feature.key)`
    // (because the registry produces literal strings), and `en.feature.key`
    // (the real-locale test path). The strings.* fallback covers cases where
    // an author imported the registry under a different alias.
    const dotted = base;
    const accessor = `strings.${base}`;
    if (haystack.includes(`"${dotted}"`) || haystack.includes(`'${dotted}'`)) continue;
    if (haystack.includes(accessor)) continue;
    orphans.push(base);
  }
  cached = { orphans, catalogMissing: false };
  return cached;
};

export default {
  meta: {
    type: "problem",
    docs: {
      description:
        "Catalog keys in en.json must be referenced from at least one src/ file. Prevents accumulating dead translations.",
    },
    schema: [],
    messages: {
      orphan:
        "Catalog key '{{key}}' has no callsite in src/. Either reference it (via `t(strings.{{key}})`) or remove it from src/i18n/locales/en.json.",
      catalogMissing:
        "Could not read catalog at {{path}} — the no-unused-keys audit cannot run.",
    },
  },
  create(context) {
    const filename = context.filename;
    if (!filename.endsWith(ANCHOR_BASENAME)) {
      return {};
    }

    return {
      Program(node) {
        const result = computeOrphans();
        if (result.catalogMissing) {
          context.report({
            node,
            messageId: "catalogMissing",
            data: { path: relative(UI_ROOT, CATALOG_PATH) },
          });
          return;
        }
        for (const orphan of result.orphans) {
          context.report({
            node,
            messageId: "orphan",
            data: { key: orphan },
          });
        }
      },
    };
  },
};
