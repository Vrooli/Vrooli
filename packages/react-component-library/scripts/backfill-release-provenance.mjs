import { existsSync } from "node:fs";
import { readFile, readdir, writeFile } from "node:fs/promises";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";
import { authoredRoot } from "./catalog-source.mjs";

const releasePattern = /^\d+\.\d+\.\d+$/;

export async function backfillReleaseProvenance({ libraryRoot = authoredRoot } = {}) {
  const root = resolve(libraryRoot);
  const hashLedger = JSON.parse(await readFile(join(root, "released-version-hashes.json"), "utf8"));
  const hashedPrefixes = new Set(hashLedger.entries.map((entry) => entry.path.split("/").slice(0, 4).join("/")));
  const path = join(root, "release-provenance.json");
  const existing = existsSync(path) ? JSON.parse(await readFile(path, "utf8")) : { schemaVersion: 1, entries: [] };
  const byRelease = new Map(existing.entries.map((entry) => [`${entry.libraryId}@${entry.version}`, entry]));

  for (const kindEntry of await readdir(root, { withFileTypes: true })) {
    if (!kindEntry.isDirectory()) continue;
    const kindRoot = join(root, kindEntry.name);
    for (const assetEntry of await readdir(kindRoot, { withFileTypes: true })) {
      if (!assetEntry.isDirectory()) continue;
      const manifestPath = join(kindRoot, assetEntry.name, "component.json");
      if (!existsSync(manifestPath)) continue;
      const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
      const versionsRoot = join(kindRoot, assetEntry.name, "versions");
      if (!existsSync(versionsRoot)) continue;
      for (const versionEntry of await readdir(versionsRoot, { withFileTypes: true })) {
        if (!versionEntry.isDirectory() || !releasePattern.test(versionEntry.name)) continue;
        const prefix = `${kindEntry.name}/${assetEntry.name}/versions/${versionEntry.name}`;
        if (!hashedPrefixes.has(prefix)) {
          const versionRoot = join(versionsRoot, versionEntry.name);
          for (const file of await readdir(versionRoot, { withFileTypes: true })) {
            if (!file.isFile()) continue;
            const relativePath = `${prefix}/${file.name}`;
            const raw = await readFile(join(versionRoot, file.name));
            hashLedger.entries.push({ path: relativePath, sha256: createHash("sha256").update(raw).digest("hex") });
          }
          hashedPrefixes.add(prefix);
        }
        const key = `${manifest.libraryId}@${versionEntry.name}`;
        if (!byRelease.has(key)) {
          byRelease.set(key, {
            libraryId: String(manifest.libraryId),
            version: versionEntry.name,
            publishedAt: "2026-08-27T00:00:00Z",
            backfilled: true,
          });
        }
      }
    }
  }
  const entries = [...byRelease.values()].sort((left, right) => left.libraryId.localeCompare(right.libraryId)
    || left.version.localeCompare(right.version, undefined, { numeric: true }));
  hashLedger.entries.sort((left, right) => left.path.localeCompare(right.path));
  await writeFile(join(root, "released-version-hashes.json"), `${JSON.stringify(hashLedger, null, 2)}\n`);
  await writeFile(path, `${JSON.stringify({ schemaVersion: 1, entries }, null, 2)}\n`);
  return { entries: entries.length };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = await backfillReleaseProvenance();
  console.log(`Backfilled ${result.entries} release provenance records.`);
}
