import { readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { authoredRoot } from "./catalog-source.mjs";

const roots = ["foundations", "hooks", "services", "primitives", "components"];

const storyContract = (kind) => `${JSON.stringify({
  schemaVersion: 5,
  kind: kind === "hooks" ? "hook" : "component",
  args: { fields: [] },
  environment: { fixtures: [] },
  stories: [{ id: "default", name: "Default", args: {}, expect: [] }],
}, null, 2)}\n`;

export async function generateStoryContracts({ root = authoredRoot, check = false } = {}) {
  const pending = [];
  for (const kind of roots) {
    const kindRoot = join(root, kind);
    if (!existsSync(kindRoot)) continue;
    for (const asset of await readdir(kindRoot, { withFileTypes: true })) {
      if (!asset.isDirectory()) continue;
      const versionsRoot = join(kindRoot, asset.name, "versions");
      if (!existsSync(versionsRoot)) continue;
      for (const version of await readdir(versionsRoot, { withFileTypes: true })) {
        if (!version.isDirectory() || version.name.endsWith(".retired")) continue;
        const path = join(versionsRoot, version.name, "story.json");
        if (!existsSync(path)) pending.push({ path, content: storyContract(kind) });
      }
    }
  }
  if (!check) for (const item of pending) await writeFile(item.path, item.content);
  return { written: check ? 0 : pending.length, missing: pending.map(({ path }) => path) };
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(new URL(import.meta.url).pathname)) {
  const result = await generateStoryContracts({ check: process.argv.includes("--check") });
  if (process.argv.includes("--check") && result.missing.length > 0) {
    console.error(JSON.stringify({ missingStoryContracts: result.missing }, null, 2));
    process.exitCode = 1;
  } else {
    console.log(`${process.argv.includes("--check") ? "Checked" : "Generated"} ${result.written || "all"} missing story contracts.`);
  }
}
