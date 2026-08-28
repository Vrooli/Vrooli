import { readFile, readdir, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { authoredRoot } from "./catalog-source.mjs";
import { entryForVersion } from "./export-resolution.mjs";
import { resolveVersionImports, scanModuleSpecifiers } from "./resolve-imports.mjs";

const prefix = "@vrooli/react-component-library/";
const semver = /^\d+\.\d+\.\d+$/;
const major = /^\d+$/;
const compareVersions = (left, right) => left.localeCompare(right, undefined, { numeric: true });
const rankByRoot = new Map([
  ["foundations", 1],
  ["hooks", 2],
  ["services", 2],
  ["adapters", 2],
  ["primitives", 3],
  ["components", 4],
  ["patterns", 5],
  ["navigation", 5],
  ["page-templates", 6],
]);

async function assetIndex(libraryRoot) {
  const byName = new Map();
  for (const rootEntry of await readdir(libraryRoot, { withFileTypes: true })) {
    if (!rootEntry.isDirectory() || !rankByRoot.has(rootEntry.name)) continue;
    const kindRoot = join(libraryRoot, rootEntry.name);
    for (const assetEntry of await readdir(kindRoot, { withFileTypes: true })) {
      if (!assetEntry.isDirectory()) continue;
      const manifestPath = join(kindRoot, assetEntry.name, "component.json");
      if (!existsSync(manifestPath)) continue;
      const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
      const versionsRoot = join(kindRoot, assetEntry.name, "versions");
      const versions = existsSync(versionsRoot)
        ? (await readdir(versionsRoot, { withFileTypes: true })).filter((entry) => entry.isDirectory() && semver.test(entry.name)).map((entry) => entry.name).sort(compareVersions)
        : [];
      byName.set(assetEntry.name, {
        kind: rootEntry.name,
        rank: rankByRoot.get(rootEntry.name),
        manifest,
        manifestPath,
        versions,
      });
    }
  }
  return byName;
}

function resolveTarget(asset, requested) {
  const deprecated = new Set(asset.manifest.deprecatedVersions ?? []);
  const evicted = new Set(asset.manifest.evictedVersions ?? []);
  const active = asset.versions.filter((version) => !deprecated.has(version) && !evicted.has(version));
  if (semver.test(requested)) {
    if (!asset.versions.includes(requested)) throw new Error(`exact dependency version ${requested} is not materialized`);
    return requested;
  }
  if (major.test(requested)) {
    const match = active.filter((version) => version.startsWith(`${requested}.`)).at(-1);
    if (!match) throw new Error(`dependency has no active release on major ${requested}`);
    return match;
  }
  const latest = String(asset.manifest.latest ?? "").trim();
  if (!latest || !active.includes(latest)) throw new Error("dependency manifest latest is not an active materialized release");
  return latest;
}

function lockDependency(specifier, assets, libraryRoot) {
  let name;
  let requested;
  if (specifier.startsWith(prefix)) {
    [name, requested = ""] = specifier.slice(prefix.length).split("/");
  } else if (specifier.startsWith("file://")) {
    const target = relative(libraryRoot, specifier.slice("file://".length)).replaceAll("\\", "/");
    const match = target.match(/^(?:components|primitives|hooks|foundations|services|adapters|patterns|navigation|page-templates)\/([^/]+)\/versions\/([^/]+)\//);
    if (!match) return null;
    [, name, requested] = match;
  } else {
    return null;
  }
  const asset = assets.get(name);
  if (!asset) throw new Error(`unknown intra-library dependency ${JSON.stringify(specifier)}`);
  let version;
  try {
    version = resolveTarget(asset, requested);
  } catch (error) {
    throw new Error(`${specifier}: ${error.message}`);
  }
  return { libraryId: String(asset.manifest.libraryId), version, rank: asset.rank };
}

export async function generateLocks({ libraryRoot = authoredRoot, resolvedAt = new Date().toISOString() } = {}) {
  const root = resolve(libraryRoot);
  const assets = await assetIndex(root);
  const specifiersByFile = scanModuleSpecifiers(root);
  const pendingWrites = [];

  for (const [name, asset] of [...assets.entries()].sort(([left], [right]) => left.localeCompare(right))) {
    for (const version of asset.versions) {
      const entry = await entryForVersion(root, asset.kind, name, version);
      if (!entry) continue;
      const versionRoot = join(root, asset.kind, name, "versions", version);
      const imports = resolveVersionImports({ entryFile: join(root, entry.source), versionRoot, specifiersByFile });
      const dependencies = imports.map((specifier) => lockDependency(specifier, assets, root)).filter(Boolean);
      const unique = [...new Map(dependencies.map((dependency) => [`${dependency.libraryId}@${dependency.version}`, dependency])).values()]
        .sort((left, right) => left.libraryId.localeCompare(right.libraryId) || compareVersions(left.version, right.version));
      const lockPath = join(versionRoot, "dependencies.json");
      let lockResolvedAt = resolvedAt;
      if (existsSync(lockPath)) {
        const existing = JSON.parse(await readFile(lockPath, "utf8"));
        if (existing.libraryId === asset.manifest.libraryId && existing.version === version && JSON.stringify(existing.dependencies) === JSON.stringify(unique)) {
          lockResolvedAt = existing.resolvedAt;
        }
      }
      const lock = { schemaVersion: 1, libraryId: String(asset.manifest.libraryId), version, resolvedAt: lockResolvedAt, dependencies: unique };
      pendingWrites.push({ path: lockPath, content: `${JSON.stringify(lock, null, 2)}\n` });
    }
  }
  for (const pending of pendingWrites) await writeFile(pending.path, pending.content);
  return { written: pendingWrites.length };
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const result = await generateLocks();
  console.log(`Generated ${result.written} dependency locks from the authored TypeScript graph.`);
}
