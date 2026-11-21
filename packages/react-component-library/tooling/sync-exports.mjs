import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join, dirname } from "node:path";
import { authoredRoot } from "./catalog-source.mjs";
import { resolveCatalogExports } from "./export-resolution.mjs";
import { findBrokenVersionImports } from "./version-import-scan.mjs";
import { fileURLToPath } from "node:url";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/tooling$/, "");
const packageJSONPath = join(packageRoot, "package.json");
const outputIndex = process.argv.indexOf("--output-package");
const outputPackagePath = outputIndex >= 0 ? process.argv[outputIndex + 1] : null;
if (outputIndex >= 0 && (!outputPackagePath || outputPackagePath.startsWith("--"))) {
  throw new Error("--output-package requires a destination path");
}
// Export only what the package compiler can actually emit. Cold versions may
// remain durable in the ledger, but they have no authored source in the
// package tree and therefore must not be advertised as package subpaths.
const libraryRoot = authoredRoot;

const { assets, resolutions } = await resolveCatalogExports({ libraryRoot, manifestRoot: authoredRoot });
const availableSubpaths = new Set(Object.keys(resolutions));

const packageJSON = JSON.parse(await readFile(packageJSONPath, "utf8"));
packageJSON.exports = {
  ".": {
    types: "./dist/exports/index.d.ts",
    import: "./dist/exports/index.js",
  },
};
// Publish every materialized exact release as well as the stable bare and
// major aliases. Immutable historical assets can retain exact cross-asset
// imports, and consumers must be able to resolve those imports through the
// package exports contract instead of depending on filesystem layout.
const publicSubpaths = [...availableSubpaths];
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
const brokenVersionImports = await findBrokenVersionImports(availableSubpaths, join(packageRoot, "..", ".."));
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
  const destination = outputPackagePath || packageJSONPath;
  await mkdir(dirname(destination), { recursive: true });
  await writeFile(destination, next);
}
const versionedExports = Object.keys(resolutions).filter((key) => /\/\d+\.\d+\.\d+$/.test(key)).length;
console.log(JSON.stringify({ assets: assets.length, versionedExports, exports: publicSubpaths.length + 1, sideMapEntries: availableSubpaths.size, brokenVersionImports: brokenVersionImports.length, checked: process.argv.includes("--check"), outputPackage: outputPackagePath || packageJSONPath }));
