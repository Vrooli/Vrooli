import { existsSync } from "node:fs";
import { readFile, readdir } from "node:fs/promises";
import { join, resolve } from "node:path";

const packagePrefix = "@vrooli/react-component-library/";
const releasePattern = /^\d+\.\d+\.\d+$/;
const majorPattern = /^\d+$/;
const compareVersions = (left, right) => left.localeCompare(right, undefined, { numeric: true });

function assertSourceRoot(libraryRoot) {
  const normalized = resolve(libraryRoot).replaceAll("\\", "/");
  if (normalized.split("/").includes("dist")) {
    throw new Error(`source resolver refuses build output root ${normalized}`);
  }
}

async function findAsset(libraryRoot, name) {
  assertSourceRoot(libraryRoot);
  for (const kindEntry of await readdir(libraryRoot, { withFileTypes: true })) {
    if (!kindEntry.isDirectory()) continue;
    const assetRoot = join(libraryRoot, kindEntry.name, name);
    const manifestPath = join(assetRoot, "component.json");
    if (!existsSync(manifestPath)) continue;
    return {
      kind: kindEntry.name,
      assetRoot,
      manifest: JSON.parse(await readFile(manifestPath, "utf8")),
    };
  }
  throw new Error(`unknown intra-library asset ${JSON.stringify(name)}`);
}

export async function resolveLibrarySpecifier(specifier, { libraryRoot } = {}) {
  const root = resolve(libraryRoot);
  assertSourceRoot(root);
  if (!specifier.startsWith(packagePrefix)) return null;
  const segments = specifier.slice(packagePrefix.length).split("/").filter(Boolean);
  const name = segments[0];
  let requested = segments[1] ?? "";
  if (!name || segments.length > 3) throw new Error(`invalid library specifier ${JSON.stringify(specifier)}`);
  if (segments.length === 3) {
    if (!majorPattern.test(segments[1]) || !releasePattern.test(segments[2]) || !segments[2].startsWith(`${segments[1]}.`)) {
      throw new Error(`invalid major-scoped library specifier ${JSON.stringify(specifier)}`);
    }
    requested = segments[2];
  }

  const asset = await findAsset(root, name);
  const versionsRoot = join(asset.assetRoot, "versions");
  const versions = (await readdir(versionsRoot, { withFileTypes: true }))
    .filter((entry) => entry.isDirectory() && releasePattern.test(entry.name))
    .map((entry) => entry.name)
    .sort(compareVersions);
  const unavailable = new Set([...(asset.manifest.deprecatedVersions ?? []), ...(asset.manifest.evictedVersions ?? [])]);
  const active = versions.filter((version) => !unavailable.has(version));
  let version;
  if (releasePattern.test(requested)) {
    if (!versions.includes(requested)) throw new Error(`${specifier}: exact release is not materialized`);
    version = requested;
  } else if (majorPattern.test(requested)) {
    version = active.filter((candidate) => candidate.startsWith(`${requested}.`)).at(-1);
  } else if (requested === "") {
    version = active.at(-1);
  } else {
    throw new Error(`invalid library version selector ${JSON.stringify(requested)}`);
  }
  if (!version) throw new Error(`${specifier}: no active released version is available`);

  const entries = await readdir(join(versionsRoot, version), { withFileTypes: true });
  const entry = entries.find((candidate) => candidate.isFile() && /\.(?:ts|tsx)$/.test(candidate.name)
    && candidate.name.replace(/\.(?:ts|tsx)$/, "") === name);
  if (!entry) throw new Error(`${specifier}: ${version} has no public entry module`);
  return {
    name,
    kind: asset.kind,
    version,
    exactSpecifier: `${packagePrefix}${name}/${version}`,
    sourcePath: join(versionsRoot, version, entry.name),
    libraryId: String(asset.manifest.libraryId),
  };
}

export function sourceLibraryResolver({ libraryRoot }) {
  const root = resolve(libraryRoot);
  assertSourceRoot(root);
  return {
    name: "react-component-library-source-resolver",
    enforce: "pre",
    async resolveId(id) {
      if (!id.startsWith(packagePrefix)) return null;
      return (await resolveLibrarySpecifier(id, { libraryRoot: root })).sourcePath;
    },
  };
}
