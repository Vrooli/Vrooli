import { readFile, readdir, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { authoredRoot } from "./catalog-source.mjs";

const ledgerPath = join(authoredRoot, "released-version-hashes.json");

export async function generateReleaseHashes({ check = false } = {}) {
  if (!existsSync(ledgerPath)) return { stale: false, removed: 0 };
  const document = JSON.parse(await readFile(ledgerPath, "utf8"));
  const entries = Array.isArray(document.entries) ? document.entries : [];
  const retained = entries.filter((entry) =>
    /\.(?:ts|tsx)$/.test(String(entry.path ?? "")) && existsSync(join(authoredRoot, entry.path))
  );
  // Existing release bytes are immutable during ordinary builds. The one-time
  // canonical-shape migration may explicitly accept the already-applied
  // authored bytes, but an unqualified build must leave the historical hash
  // oracle untouched so a later in-place edit remains observable.
  const acceptMigration = process.env.RCL_ACCEPT_RELEASE_MIGRATION === "1";
  const refreshed = [];
  for (const entry of retained) {
    refreshed.push({
      ...entry,
      sha256: acceptMigration
        ? createHash("sha256").update(await readFile(join(authoredRoot, entry.path))).digest("hex")
        : entry.sha256,
    });
  }
  const known = new Set(refreshed.map((entry) => entry.path));
  const additions = [];
  for (const kind of ["foundations", "hooks", "services", "primitives", "components"]) {
    const kindRoot = join(authoredRoot, kind);
    if (!existsSync(kindRoot)) continue;
    for (const asset of await readdir(kindRoot, { withFileTypes: true })) {
      if (!asset.isDirectory()) continue;
      const versionsRoot = join(kindRoot, asset.name, "versions");
      if (!existsSync(versionsRoot)) continue;
      for (const version of await readdir(versionsRoot, { withFileTypes: true })) {
        if (!version.isDirectory() || version.name.endsWith(".retired")) continue;
        for (const file of await readdir(join(versionsRoot, version.name))) {
          if (!/\.(?:ts|tsx)$/.test(file)) continue;
          const path = `${kind}/${asset.name}/versions/${version.name}/${file}`;
          if (known.has(path)) continue;
          const bytes = await readFile(join(versionsRoot, version.name, file));
          additions.push({ path, sha256: createHash("sha256").update(bytes).digest("hex") });
        }
      }
    }
  }
  const nextEntries = [...refreshed, ...additions].sort((left, right) => left.path.localeCompare(right.path));
  const content = `${JSON.stringify({ ...document, entries: nextEntries }, null, 2)}\n`;
  const stale = JSON.stringify(entries) !== JSON.stringify(nextEntries);
  if (stale && !check) await writeFile(ledgerPath, content);
  return { stale, removed: entries.length - retained.length, added: additions.length };
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(new URL(import.meta.url).pathname)) {
  const result = await generateReleaseHashes({ check: process.argv.includes("--check") });
  if (result.stale && process.argv.includes("--check")) {
    console.error(JSON.stringify({ staleReleaseHashes: true, removed: result.removed }));
    process.exitCode = 1;
  } else {
    console.log(`${process.argv.includes("--check") ? "Checked" : "Generated"} authored release hashes (${result.removed} derived entries removed, ${result.added} missing entries added).`);
  }
}
