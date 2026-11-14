/**
 * Codegen: derive `src/consts/strings.generated.ts` from `src/i18n/locales/en.json`.
 *
 * NOTE: deliberately no `#!/usr/bin/env node` shebang. Vite's esbuild-based
 * config loader bundles this file when resolving the strings codegen plugin,
 * and esbuild rejects shebangs in bundled inputs. We always invoke via
 * `node scripts/gen-strings.mjs`, so the shebang would be dead weight anyway.
 *
 * Run modes:
 *   node scripts/gen-strings.mjs           # write the file (no-op if up to date)
 *   node scripts/gen-strings.mjs --check   # exit 1 if the file is out of date
 *
 * The Vite plugin in `vite-plugin-strings-codegen.mjs` invokes this on every
 * dev start, on HMR of en.json, and on every build start, so the generated
 * file stays in sync without any manual step. The CLI mode + `--check` flag
 * exists so:
 *   1. Contributors editing en.json without `vite dev` running can regenerate
 *      on demand (`pnpm strings:gen`).
 *   2. CI can fail PRs that forgot to commit the regen (`pnpm strings:check`).
 *
 * Why a generated file (not runtime traversal)? Walking en.json at module
 * load time forces the bundler to ship the catalog twice (once as i18next's
 * resource, once as input to the registry). At ~500 strings that's tens of KB
 * in every user's initial download. Codegen eliminates the second copy.
 */
import { existsSync, mkdtempSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, "..");

export const SOURCE_PATH = join(ROOT, "src/i18n/locales/en.json");
export const TARGET_PATH = join(ROOT, "src/consts/strings.generated.ts");

const HEADER = `// AUTO-GENERATED — do not edit by hand.
//
// Source : src/i18n/locales/en.json
// Codegen: scripts/gen-strings.mjs (invoked automatically by
//          vite-plugin-strings-codegen on dev start, HMR of en.json, and
//          build start; also available as \`pnpm strings:gen\` and
//          \`pnpm strings:check\`).
//
// See src/consts/strings.ts for the registry's purpose, why it exists, and
// how to use it. This file mirrors the shape of en.json with each leaf
// replaced by its dotted key path — that's the value the i18next \`t()\`
// function takes as its first argument.
`;

/**
 * Catalog keys whose final segment starts with `_` are sentinels (e.g.,
 * `_comment` documenting the file). Skip them everywhere — they don't
 * belong in the typed registry, the parity test, or the unused-key audit.
 * See locales.test.ts and eslint-rules/no-unused-keys.js for the matching
 * skip logic; the convention is duplicated by intent so each consumer is
 * self-explanatory, not via a shared import.
 */
export const isSentinelKey = (key) => key.startsWith("_");

const buildKeys = (catalog, prefix = "") => {
  const result = {};
  for (const [key, value] of Object.entries(catalog)) {
    if (isSentinelKey(key)) continue;
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "string") {
      result[key] = path;
    } else if (value && typeof value === "object" && !Array.isArray(value)) {
      result[key] = buildKeys(value, path);
    } else {
      throw new Error(
        `Unsupported value type at '${path}': ${value === null ? "null" : Array.isArray(value) ? "array" : typeof value}. ` +
          "Catalog leaves must be strings; nested catalogs must be plain objects.",
      );
    }
  }
  return result;
};

const SAFE_KEY = /^[A-Za-z_$][\w$]*$/;

const renderTree = (tree, depth = 1) => {
  const pad = "  ".repeat(depth);
  const close = "  ".repeat(depth - 1);
  const entries = Object.entries(tree).map(([key, value]) => {
    const safeKey = SAFE_KEY.test(key) ? key : JSON.stringify(key);
    if (typeof value === "string") {
      return `${pad}${safeKey}: ${JSON.stringify(value)}`;
    }
    return `${pad}${safeKey}: ${renderTree(value, depth + 1)}`;
  });
  return `{\n${entries.join(",\n")},\n${close}}`;
};

export const generateContents = () => {
  if (!existsSync(SOURCE_PATH)) {
    throw new Error(`Catalog source not found: ${SOURCE_PATH}`);
  }
  const catalog = JSON.parse(readFileSync(SOURCE_PATH, "utf-8"));
  const tree = buildKeys(catalog);
  return `${HEADER}
export const strings = ${renderTree(tree)} as const;

export type Strings = typeof strings;
`;
};

export const writeIfChanged = () => {
  const next = generateContents();
  const current = existsSync(TARGET_PATH) ? readFileSync(TARGET_PATH, "utf-8") : "";
  if (current === next) return false;
  // Unit and performance providers can invoke Vite/codegen concurrently.
  // Publish a complete generated module in one rename so a reader never sees
  // the target between truncation and the final write.
  const outputDir = mkdtempSync(join(dirname(TARGET_PATH), ".strings-generated-"));
  const tempPath = join(outputDir, "strings.generated.ts");
  try {
    writeFileSync(tempPath, next);
    renameSync(tempPath, TARGET_PATH);
  } finally {
    rmSync(outputDir, { recursive: true, force: true });
  }
  return true;
};

export const isOutOfDate = () => {
  const next = generateContents();
  const current = existsSync(TARGET_PATH) ? readFileSync(TARGET_PATH, "utf-8") : "";
  return current !== next;
};

const isMain = process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1];
if (isMain) {
  const check = process.argv.includes("--check");
  const rel = relative(ROOT, TARGET_PATH);
  if (check) {
    if (isOutOfDate()) {
      console.error(
        `✖ ${rel} is out of date with en.json.\n  Run \`pnpm strings:gen\` and commit the result.`,
      );
      process.exit(1);
    }
    console.log(`✓ ${rel} is in sync with en.json`);
  } else {
    const wrote = writeIfChanged();
    console.log(wrote ? `✓ wrote ${rel}` : `✓ ${rel} already up to date`);
  }
}
