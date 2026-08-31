import { readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative, dirname } from "node:path";
import { authoredRoot } from "./catalog-source.mjs";
import { resolveCatalogExports } from "./export-resolution.mjs";
import { fileURLToPath } from "node:url";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/tooling$/, "");
const packageJSONPath = join(packageRoot, "package.json");
// Export only what the package compiler can actually emit. Cold versions may
// remain durable in the ledger, but they have no authored source in the
// package tree and therefore must not be advertised as package subpaths.
const libraryRoot = authoredRoot;

async function filesUnder(root) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      // `.retired` holds source the catalog has already removed from the live
      // set. A pin inside it can no longer reach a consumer, so scanning it
      // only produces broken-import reports nobody can act on — the file is
      // not editable as a released version and not reachable as a live one.
      if (["node_modules", ".git", "dist", ".vite", ".retired"].includes(entry.name) || entry.name.startsWith(".vrooli-artifact-stage-")) continue;
      files.push(...await filesUnder(path));
    }
    else files.push(path);
  }
  return files;
}

async function findBrokenVersionImports(availableSubpaths) {
  const repoRoot = join(packageRoot, "..", "..");
  // Capture the whole subpath, however many segments it has. The previous
  // two-segment pattern could not see a `Name/<major>/<version>` import at all,
  // so the very specifier this guard exists to catch was invisible to it.
  const importRE = /(?:from\s+|import\s*\(\s*)["'](@vrooli\/react-component-library\/([^'"\s]+))["']/g;
  const broken = [];
  for (const root of [join(repoRoot, "scenarios"), join(repoRoot, "templates")]) {
    if (!existsSync(root)) continue;
    for (const file of await filesUnder(root)) {
      if (!/\.(?:ts|tsx|js|jsx|mjs|cjs)$/.test(file)) continue;
      // The library's immutable historical sources are compiled according to
      // the package build policy, not consumed as application sources. Their
      // old exact pins may intentionally point at retired releases; consumer
      // and template imports are the reachability contract checked here.
      if (file.startsWith(join(repoRoot, "scenarios", "react-component-library", "library") + "/")) continue;
      const source = await readFile(file, "utf8");
      if (file.includes(`${join("catalog", "calibration")}`) || file.includes(`${join("tools", "testdata")}`)) continue;
      // Test files name specifiers as fixtures — including deliberately absent
      // ones, to prove the resolver rejects them. A genuinely broken import in
      // a test fails when the test runs, which is a faster signal than this
      // scan, so they are out of scope here rather than a permanent finding.
      if (/\.(?:test|spec)\.[cm]?[jt]sx?$/.test(file)) continue;
      for (const match of source.matchAll(importRE)) {
        const key = `./${match[2]}`;
        if (!availableSubpaths.has(key)) broken.push({ file: relative(repoRoot, file), specifier: match[1], key });
      }
    }
  }
  return broken;
}

const { assets, resolutions } = await resolveCatalogExports({ libraryRoot, manifestRoot: authoredRoot });
const availableSubpaths = new Set(Object.keys(resolutions));

const packageJSON = JSON.parse(await readFile(packageJSONPath, "utf8"));
packageJSON.exports = {
  ".": {
    types: "./dist/exports/index.d.ts",
    import: "./dist/exports/index.js",
  },
};
// Node's public package contract carries the stable bare and major aliases.
// Exact releases remain available in the generated resolution side map and
// package artifact for the workbench and reproducible historical builds, but
// do not inflate the manifest every consumer parses.
const publicSubpaths = [...availableSubpaths].filter((subpath) => {
  const segments = subpath.slice(2).split("/");
  return segments.length <= 2 && (segments.length === 1 || /^\d+$/.test(segments[1] ?? ""));
});
for (const subpath of publicSubpaths.sort()) {
  const alias = subpath.slice(2);
  packageJSON.exports[subpath] = {
    types: `./dist/exports/${alias}.d.ts`,
    import: `./dist/exports/${alias}.js`,
  };
}
const next = `${JSON.stringify(packageJSON, null, 2)
  .replace(/^(\s*)"(\.[^\"]+)": \{\n\s+"types": "([^\"]+)",\n\s+"import": "([^\"]+)"\n\s+\}/gm,
    (_match, indent, key, types, importPath) => `${indent}"${key}": {"types": "${types}", "import": "${importPath}"}`)}\n`;
const brokenVersionImports = await findBrokenVersionImports(availableSubpaths);
if (brokenVersionImports.length > 0) {
  console.error(JSON.stringify({ brokenVersionImports }, null, 2));
  process.exitCode = 1;
}
if (process.argv.includes("--check")) {
  const current = await readFile(packageJSONPath, "utf8");
  if (current !== next) {
    console.error("react-component-library exports are stale; run pnpm sync-exports");
    process.exitCode = 1;
  }
} else {
  await writeFile(packageJSONPath, next);
}
const versionedExports = Object.keys(resolutions).filter((key) => /\/\d+\.\d+\.\d+$/.test(key)).length;
console.log(JSON.stringify({ assets: assets.length, versionedExports, exports: publicSubpaths.length + 1, sideMapEntries: availableSubpaths.size, brokenVersionImports: brokenVersionImports.length, checked: process.argv.includes("--check") }));
