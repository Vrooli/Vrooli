import { existsSync } from "node:fs";
import { readFile, readdir } from "node:fs/promises";
import { join, relative, sep } from "node:path";

async function filesUnder(root, depth = 0) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      // A project's top-level docs tree holds retained evidence and history,
      // not live package consumers. Keep scanning nested source paths such as
      // ui/src/docs so a directory name alone cannot hide active code.
      if (depth === 1 && entry.name === "docs") continue;
      // `.retired` holds source the catalog has already removed from the live
      // set. A pin inside it cannot reach a consumer or released version.
      if (["node_modules", ".git", "dist", ".vite", ".retired"].includes(entry.name) || entry.name.startsWith(".vrooli-artifact-stage-")) continue;
      files.push(...await filesUnder(path, depth + 1));
    } else {
      files.push(path);
    }
  }
  return files;
}

export async function findBrokenVersionImports(availableSubpaths, repoRoot) {
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
      if (file.startsWith(join(repoRoot, "scenarios", "react-component-library", "library") + sep)) continue;
      const source = await readFile(file, "utf8");
      if (file.includes(join("catalog", "calibration")) || file.includes(join("tools", "testdata"))) continue;
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
