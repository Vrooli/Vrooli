#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const libraryRoot = path.join(repoRoot, "scenarios/react-component-library/library");
const sourceExtensions = ["", ".ts", ".tsx", ".js", ".jsx"];

function walk(directory) {
  const result = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) result.push(...walk(absolute));
    else if (entry.isFile() && /\.(?:ts|tsx|js|jsx)$/.test(entry.name)) result.push(absolute);
  }
  return result;
}

function resolveSource(specifier, importer) {
  const base = path.resolve(path.dirname(importer), specifier);
  for (const extension of sourceExtensions) {
    const candidate = `${base}${extension}`;
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) return candidate;
  }
  for (const extension of [".ts", ".tsx", ".js", ".jsx"]) {
    const candidate = path.join(base, `index${extension}`);
    if (fs.existsSync(candidate)) return candidate;
  }
  return null;
}

function packageSpecifier(target) {
  const relativeTarget = path.relative(libraryRoot, target).split(path.sep).join("/");
  const match = relativeTarget.match(/^(?:components|primitives|hooks|foundations|services)\/([^/]+)\/versions\/([^/]+)\//);
  if (!match) return null;
  return `@vrooli/react-component-library/${match[1]}/${match[2]}`;
}

function migrateFile(file, checkOnly) {
  const original = fs.readFileSync(file, "utf8");
  const pattern = /((?:from\s*|import\s*\(|require\s*\()(["']))(\.\.?\/[^"'\n]*\/versions\/[^"'\n]+)(\2)/g;
  const migrated = original.replace(pattern, (whole, prefix, quote, specifier, suffix) => {
    const target = resolveSource(specifier, file);
    const replacement = target ? packageSpecifier(target) : null;
    return replacement ? `${prefix}${replacement}${suffix}` : whole;
  });
  if (migrated !== original && !checkOnly) fs.writeFileSync(file, migrated);
  return migrated !== original;
}

const checkOnly = process.argv.includes("--check");
const changed = walk(libraryRoot).filter((file) => migrateFile(file, checkOnly));
if (checkOnly && changed.length) {
  console.error(`cross-version imports remain in ${changed.length} source file(s)`);
  process.exitCode = 1;
}
console.log(JSON.stringify({ filesChanged: changed.length, checked: checkOnly }));
