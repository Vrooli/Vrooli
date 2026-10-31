import { readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { authoredRoot } from "./catalog-source.mjs";

const roots = ["foundations", "hooks", "services", "primitives", "components"];

async function projection(path) {
  const current = JSON.parse(await readFile(path, "utf8"));
  const output = { ...current };
  // Library-prefixed identities are valid catalog projections for assets that
  // do not yet have a domain id. Preserve the identity edge when normalizing a
  // legacy manifest so declaration coverage and indexing can still join it.
  if (!output.supplemental && !output.catalogId && output.libraryId) output.catalogId = output.libraryId;
  const stable = Object.fromEntries(Object.keys(output).sort().map((field) => [field, output[field]]));
  return `${JSON.stringify(stable, null, 2)}\n`;
}

export async function generateManifests({ root = authoredRoot, check = false } = {}) {
  const pending = [];
  for (const kind of roots) {
    const kindRoot = join(root, kind);
    if (!existsSync(kindRoot)) continue;
    for (const asset of await readdir(kindRoot, { withFileTypes: true })) {
      if (!asset.isDirectory()) continue;
      const path = join(kindRoot, asset.name, "component.json");
      if (!existsSync(path)) continue;
      pending.push({ path, content: await projection(path) });
    }
  }
  const stale = [];
  for (const item of pending) {
    if (!existsSync(item.path) || await readFile(item.path, "utf8") !== item.content) stale.push(item.path);
  }
  if (!check) for (const item of pending) await writeFile(item.path, item.content);
  return { written: check ? 0 : pending.length, stale: stale.map((path) => relative(resolve(root, "..", ".."), path)) };
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(new URL(import.meta.url).pathname)) {
  const result = await generateManifests({ check: process.argv.includes("--check") });
  if (process.argv.includes("--check") && result.stale.length > 0) {
    console.error(JSON.stringify({ staleManifests: result.stale }, null, 2));
    process.exitCode = 1;
  } else {
    console.log(`${process.argv.includes("--check") ? "Checked" : "Generated"} ${result.written || result.stale.length || "all"} component manifest projections.`);
  }
}
