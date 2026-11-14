#!/usr/bin/env node

import { readdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scenarioRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..");


const HEADER = /@libraryId\s+([^\s*]+)[\s\S]*?@version\s+([^\s*]+)/;
const SOURCE_EXTENSIONS = new Set([".ts", ".tsx"]);

async function directories(root) {
  const entries = await readdir(root, { withFileTypes: true });
  return entries
    .filter((entry) => entry.isDirectory() && entry.name !== ".retired")
    .map((entry) => path.join(root, entry.name));
}

async function sourceFiles(versionDir) {
  const entries = await readdir(versionDir, { withFileTypes: true });
  return entries
    .filter((entry) => entry.isFile() && SOURCE_EXTENSIONS.has(path.extname(entry.name)))
    .map((entry) => path.join(versionDir, entry.name));
}

async function versionDirectories(libraryRoot) {
  const result = [];
  for (const kind of await directories(libraryRoot)) {
    for (const asset of await directories(kind)) {
      const versions = path.join(asset, "versions");
      try {
        const manifest = JSON.parse(await readFile(path.join(asset, "component.json"), "utf8"));
        // Only the active governed draft is writable. Released bytes are never
        // migration targets, even when they still use the legacy API.
        if (typeof manifest.draft === "string" && /^\d+\.\d+\.\d+-draft\.\d+$/.test(manifest.draft)) {
          result.push(path.join(versions, manifest.draft));
        }
      } catch (error) {
        if (error.code !== "ENOENT") throw error;
      }
    }
  }
  return result;
}

export async function migrateStylesheetKeys({ libraryRoot = path.join(scenarioRoot, "library"), apply = false, log = console.log } = {}) {
let blockedFiles = 0;
let changedFiles = 0;
let changedCalls = 0;
let skippedFiles = 0;

for (const versionDir of await versionDirectories(libraryRoot)) {
  for (const file of await sourceFiles(versionDir)) {
    const original = await readFile(file, "utf8");
    const header = original.match(HEADER);
    let libraryId = header?.[1];
    const version = header?.[2] ?? path.basename(versionDir);
    if (!libraryId) {
      try {
        const manifest = JSON.parse(await readFile(path.join(path.dirname(path.dirname(versionDir)), "component.json"), "utf8"));
        libraryId = manifest.libraryId;
      } catch (error) {
        if (error.code !== "ENOENT") throw error;
      }
    }
    if (!libraryId || (!original.includes("<StyleSheet") && !original.includes("useLibraryStyleSheet("))) {
      if (original.includes("<StyleSheet") || original.includes("useLibraryStyleSheet(")) skippedFiles += 1;
      continue;
    }
    const injections = (original.match(/(?:useLibraryStyleSheet\s*\(|<StyleSheet\b)/g) ?? []).length;
    if (injections > 1) {
      blockedFiles += 1;
      log(`manual-review ${path.relative(libraryRoot, file)}: combine sheets before assigning one owner/version key`);
      continue;
    }
    let updated = original;
    updated = updated.replace(
      /(useLibraryStyleSheet\s*\(\s*"[^"]+"\s*,\s*"\d+\.\d+\.\d+"\s*,\s*)"\d+\.\d+\.\d+"\s*,/g,
      "$1",
    );
    updated = updated.replace(
      /(<StyleSheet\b[^>]*?)\bname\s*=\s*("[^"]*"|'[^']*')/g,
      (_match, prefix) => `${prefix}libraryId="${libraryId}" version="${version}"`,
    );
    updated = updated.replace(
      /(useLibraryStyleSheet\s*\(\s*)("[^"]*"|'[^']*')(\s*,\s*)(?!\s*["'])/g,
      (_match, prefix, _legacyKey, comma) => `${prefix}"${libraryId}", "${version}"${comma}`,
    );
    if (updated === original) continue;
    changedFiles += 1;
    changedCalls += (original.match(/<StyleSheet\b[^>]*\bname\s*=/g) ?? []).length;
    changedCalls += (original.match(/useLibraryStyleSheet\s*\(\s*("[^"]*"|'[^']*')\s*,/g) ?? []).length;
    const relative = path.relative(scenarioRoot, file);
    log(`${apply ? "rewrite" : "would-rewrite"} ${relative}`);
    if (apply) await writeFile(file, updated);
  }
}

return { apply, changedFiles, changedCalls, skippedFiles, blockedFiles };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = await migrateStylesheetKeys({ apply: process.argv.includes("--apply") });
  console.log(JSON.stringify(result, null, 2));
  if (result.blockedFiles || (!result.apply && result.changedFiles)) process.exitCode = 2;
}
