import { readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/scripts$/, "");
const libraryRoot = join(packageRoot, "..", "..", "scenarios", "react-component-library", "library");
const packageJSONPath = join(packageRoot, "package.json");

async function filesUnder(root) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      if (["node_modules", ".git", "dist", ".vite"].includes(entry.name) || entry.name.startsWith(".vrooli-artifact-stage-")) continue;
      files.push(...await filesUnder(path));
    }
    else files.push(path);
  }
  return files;
}

async function findBrokenVersionImports(exportsMap) {
  const repoRoot = join(packageRoot, "..", "..");
  const importRE = /(?:from\s+|import\s*\(\s*)["'](@vrooli\/react-component-library\/([^/'"\s]+)\/([^/'"\s]+))["']/g;
  const broken = [];
  for (const root of [join(repoRoot, "scenarios"), join(repoRoot, "templates")]) {
    if (!existsSync(root)) continue;
    for (const file of await filesUnder(root)) {
      if (!/\.(?:ts|tsx|js|jsx|mjs|cjs)$/.test(file)) continue;
      const source = await readFile(file, "utf8");
      if (file.includes(`${join("catalog", "calibration")}`)) continue;
      for (const match of source.matchAll(importRE)) {
        const key = `./${match[2]}/${match[3]}`;
        if (!exportsMap[key]) broken.push({ file: relative(repoRoot, file), specifier: match[1], key });
      }
    }
  }
  return broken;
}

// A kebab-named asset directory carries its entry file in PascalCase
// (markdown-renderer/MarkdownRenderer.tsx). Treat that stem as the directory's
// entry point so the asset still earns an export; without this the asset is
// dropped from the export map silently while adopters import it by subpath.
function pascalCase(name) {
  return name.split("-").map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1)).join("");
}

function componentPart(file) {
  const parts = relative(libraryRoot, file).split("/");
  if (parts.length < 5 || parts[2] !== "versions") return null;
  const [kind, name, , version, filename] = parts;
  if (kind === "preview-harnesses") return null;
  if (!filename || !/\.(?:ts|tsx)$/.test(filename) || filename === "story.tsx") return null;
  const stem = filename.replace(/\.(?:ts|tsx)$/, "");
  const isEntryStem = stem === name || stem === pascalCase(name);
  if (!isEntryStem && !(kind === "hooks" || kind === "services" || kind === "foundations" || kind === "primitives")) return null;
  return { kind, name, version, stem };
}

function distPath(kind, name, version, stem) {
  return `./dist/${kind}/${name}/versions/${version}/${stem}`;
}

const sourceFiles = (await filesUnder(libraryRoot)).map(componentPart).filter(Boolean);
const versions = new Map();
for (const part of sourceFiles) {
  if (!versions.has(`${part.kind}/${part.name}/${part.version}`)) versions.set(`${part.kind}/${part.name}/${part.version}`, part);
}

const exportsMap = {};
const latestByAsset = new Map();
for (const part of versions.values()) {
  const key = `${part.kind}/${part.name}`;
  const current = latestByAsset.get(key);
  if (!current || part.version.localeCompare(current.version, undefined, { numeric: true }) > 0) latestByAsset.set(key, part);
  exportsMap[`./${part.name}/${part.version}`] = {
    types: `${distPath(part.kind, part.name, part.version, part.stem)}.d.ts`,
    import: `${distPath(part.kind, part.name, part.version, part.stem)}.js`,
  };
}
for (const part of latestByAsset.values()) {
  exportsMap[`./${part.name}`] = {
    types: `${distPath(part.kind, part.name, part.version, part.stem)}.d.ts`,
    import: `${distPath(part.kind, part.name, part.version, part.stem)}.js`,
  };
}

const packageJSON = JSON.parse(await readFile(packageJSONPath, "utf8"));
packageJSON.exports = Object.fromEntries(Object.entries(exportsMap).sort(([a], [b]) => a.localeCompare(b)));
const next = `${JSON.stringify(packageJSON, null, 2)}\n`;
const brokenVersionImports = await findBrokenVersionImports(exportsMap);
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
console.log(JSON.stringify({ assets: latestByAsset.size, versionedExports: versions.size, exports: Object.keys(exportsMap).length, brokenVersionImports: brokenVersionImports.length, checked: process.argv.includes("--check") }));
