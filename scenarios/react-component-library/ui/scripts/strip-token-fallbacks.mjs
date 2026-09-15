#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const root = path.resolve(new URL("../..", import.meta.url).pathname, "../..");
const libraryRoot = path.join(root, "scenarios", "react-component-library", "library");
const configPath = path.join(root, "scenarios", "react-component-library", "catalog", "config.json");

const config = JSON.parse(fs.readFileSync(configPath, "utf8"));
const exemptions = new Set((config["x-token-fallback-exemptions"] ?? []).map((entry) => entry.token));

function activeVersionDirs() {
  const dirs = [];
  for (const kind of fs.readdirSync(libraryRoot)) {
    const kindDir = path.join(libraryRoot, kind);
    if (!fs.statSync(kindDir).isDirectory()) continue;
    for (const asset of fs.readdirSync(kindDir)) {
      const assetDir = path.join(kindDir, asset);
      const manifestPath = path.join(assetDir, "component.json");
      if (!fs.existsSync(manifestPath)) continue;
      const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
      for (const version of new Set([manifest.latest, manifest.draft].filter(Boolean))) {
        const versionDir = path.join(assetDir, "versions", version);
        if (fs.existsSync(versionDir)) dirs.push(versionDir);
      }
    }
  }
  return dirs;
}

function rewrite(source) {
  let output = "";
  let cursor = 0;
  let changes = 0;
  while (cursor < source.length) {
    const start = source.indexOf("var(", cursor);
    if (start < 0) {
      output += source.slice(cursor);
      break;
    }
    output += source.slice(cursor, start);
    let depth = 0;
    let comma = -1;
    let end = -1;
    for (let index = start + 4; index < source.length; index += 1) {
      const char = source[index];
      if (char === "(") depth += 1;
      else if (char === ")") {
        if (depth === 0) { end = index; break; }
        depth -= 1;
      } else if (char === "," && depth === 0 && comma < 0) comma = index;
    }
    if (end < 0) { output += source.slice(start); break; }
    const property = source.slice(start + 4, comma < 0 ? end : comma).trim();
    if (comma > 0 && !exemptions.has(property)) {
      output += `var(${property})`;
      changes += 1;
    } else {
      output += source.slice(start, end + 1);
    }
    cursor = end + 1;
  }
  return { output, changes };
}

let filesChanged = 0;
let fallbackChanges = 0;
for (const dir of activeVersionDirs()) {
  for (const file of fs.readdirSync(dir)) {
    if (!/\.(ts|tsx|css)$/.test(file)) continue;
    const filePath = path.join(dir, file);
    const source = fs.readFileSync(filePath, "utf8");
    const result = rewrite(source);
    if (result.changes === 0) continue;
    fs.writeFileSync(filePath, result.output);
    filesChanged += 1;
    fallbackChanges += result.changes;
  }
}
console.log(JSON.stringify({ filesChanged, fallbackChanges, exemptions: [...exemptions].sort() }));
