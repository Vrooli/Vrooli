import { readFile, readdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative } from "node:path";

const pascalCase = (name) =>
  name.split("-").map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1)).join("");

const compareVersions = (left, right) => left.localeCompare(right, undefined, { numeric: true });
const isRelease = (version) => /^\d+\.\d+\.\d+$/.test(version);

async function entryForVersion(libraryRoot, kind, name, version) {
  const root = join(libraryRoot, kind, name, "versions", version);
  if (!existsSync(root)) return null;
  const candidates = await readdir(root, { withFileTypes: true });
  const stems = new Set([name, pascalCase(name)]);
  const entry = candidates
    .filter((candidate) => candidate.isFile() && /\.(?:ts|tsx)$/.test(candidate.name))
    .find((candidate) => stems.has(candidate.name.replace(/\.(?:ts|tsx)$/, "")));
  if (!entry) return null;
  return {
    kind,
    name,
    version,
    stem: entry.name.replace(/\.(?:ts|tsx)$/, ""),
    source: `${kind}/${name}/versions/${version}/${entry.name}`,
  };
}

export async function resolveCatalogExports({ libraryRoot, manifestRoot = libraryRoot }) {
  const resolutions = {};
  const assets = [];
  const failures = [];

  for (const kindEntry of await readdir(manifestRoot, { withFileTypes: true })) {
    if (!kindEntry.isDirectory()) continue;
    const kind = kindEntry.name;
    const kindRoot = join(manifestRoot, kind);
    for (const assetEntry of await readdir(kindRoot, { withFileTypes: true })) {
      if (!assetEntry.isDirectory()) continue;
      const name = assetEntry.name;
      const manifestPath = join(kindRoot, name, "component.json");
      if (!existsSync(manifestPath)) continue;
      const manifest = JSON.parse(await readFile(manifestPath, "utf8"));
      const versionsRoot = join(libraryRoot, kind, name, "versions");
      if (!existsSync(versionsRoot)) {
        failures.push(`${relative(manifestRoot, manifestPath)}: versions directory does not exist`);
        continue;
      }
      const diskVersions = (await readdir(versionsRoot, { withFileTypes: true }))
        .filter((entry) => entry.isDirectory() && isRelease(entry.name))
        .map((entry) => entry.name)
        .sort(compareVersions);
      const deprecated = new Set(Array.isArray(manifest.deprecatedVersions) ? manifest.deprecatedVersions : []);
      const latest = typeof manifest.latest === "string" ? manifest.latest.trim() : "";
      if (!latest || !diskVersions.includes(latest)) {
        failures.push(`${relative(manifestRoot, manifestPath)}: latest ${JSON.stringify(latest)} does not name a released version on disk`);
        continue;
      }
      if (deprecated.has(latest)) {
        failures.push(`${relative(manifestRoot, manifestPath)}: latest ${latest} is deprecated`);
        continue;
      }
      const highest = diskVersions.at(-1);
      if (highest !== latest && !(typeof manifest.latestRationale === "string" && manifest.latestRationale.trim())) {
        failures.push(`${relative(manifestRoot, manifestPath)}: latest ${latest} differs from highest release ${highest} without latestRationale`);
        continue;
      }

      const active = [];
      for (const version of diskVersions) {
        const entry = await entryForVersion(libraryRoot, kind, name, version);
        if (!entry) {
          failures.push(`${relative(manifestRoot, manifestPath)}: ${version} has no public entry module`);
          continue;
        }
        resolutions[`./${name}/${version}`] = entry;
        // `<name>/<major>/<version>` is the pinned form consumers write when
        // they want the exact release and the major line it belongs to visible
        // in the import itself. It resolves to the same entry as
        // `<name>/<version>` — including for deprecated versions, so the two
        // spellings never disagree about what is reachable.
        resolutions[`./${name}/${version.split(".")[0]}/${version}`] = entry;
        if (!deprecated.has(version)) active.push(entry);
      }
      const latestEntry = active.find((entry) => entry.version === latest);
      if (!latestEntry) {
        failures.push(`${relative(manifestRoot, manifestPath)}: latest ${latest} has no resolvable public entry module`);
        continue;
      }
      const latestByMajor = new Map();
      for (const entry of active) {
        const major = entry.version.split(".")[0];
        const current = latestByMajor.get(major);
        if (!current || compareVersions(entry.version, current.version) > 0) latestByMajor.set(major, entry);
      }
      for (const [major, entry] of latestByMajor) resolutions[`./${name}/${major}`] = entry;
      resolutions[`./${name}`] = latestEntry;
      assets.push({ kind, name, latest, activeVersions: active.map((entry) => entry.version) });
    }
  }

  if (failures.length > 0) {
    throw new Error(`react-component-library export resolution failed:\n${failures.map((failure) => `  - ${failure}`).join("\n")}`);
  }
  return { assets, resolutions };
}
