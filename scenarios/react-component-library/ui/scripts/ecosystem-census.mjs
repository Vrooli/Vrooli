#!/usr/bin/env node

/**
 * Emit the repository facts used by the component-library retirement and
 * ecosystem gates.  This intentionally uses only Node's standard library so
 * the census is runnable from a clean checkout before the UI workspace is
 * installed.
 */
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptDir, "../../../..");
const libraryRoot = path.join(repoRoot, "scenarios/react-component-library/library");
const packageManifestPath = path.join(repoRoot, "packages/react-component-library/package.json");

const textExtensions = new Set([".js", ".jsx", ".mjs", ".ts", ".tsx", ".json"]);
const sourceExtensions = new Set([".js", ".jsx", ".ts", ".tsx"]);

function walk(directory, seen = new Set()) {
  if (!fs.existsSync(directory)) return [];
  let realDirectory;
  try {
    realDirectory = fs.realpathSync(directory);
  } catch {
    return [];
  }
  if (seen.has(realDirectory)) return [];
  seen.add(realDirectory);
  const entries = fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name));
  const files = [];
  for (const entry of entries) {
    if (
      entry.isSymbolicLink() ||
      entry.name === "node_modules" ||
      entry.name === ".git" ||
      entry.name.startsWith(".vrooli-artifact-stage-") ||
      entry.name === ".vite" ||
      entry.name === "dist"
    ) continue;
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      for (const child of walk(absolute, seen)) files.push(child);
    }
    else if (entry.isFile()) files.push(absolute);
  }
  return files;
}

function relative(absolute) {
  return path.relative(repoRoot, absolute).split(path.sep).join("/");
}

function readJSON(file) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch {
    return null;
  }
}

function increment(map, key, amount = 1) {
  map[key] = (map[key] ?? 0) + amount;
}

function collectKeys(value, output = []) {
  if (!value || typeof value !== "object") return output;
  if (Array.isArray(value)) {
    for (const item of value) collectKeys(item, output);
    return output;
  }
  for (const [key, child] of Object.entries(value)) {
    output.push(key);
    collectKeys(child, output);
  }
  return output;
}

function storyStats(files) {
  const result = {
    contracts: 0,
    stories: 0,
    contractsWithOneStory: 0,
    storiesWithOneSpecimen: 0,
    storiesWithZeroExpectations: 0,
    storiesWithExpectations: 0,
    storiesWithFrame: 0,
    storiesWithAxes: 0,
    structuredTagKeys: {},
    storyFiles: [],
  };
  for (const file of files.filter((item) => path.basename(item) === "story.json")) {
    const contract = readJSON(file);
    if (!contract || !Array.isArray(contract.stories)) continue;
    result.contracts += 1;
    result.stories += contract.stories.length;
    if (contract.stories.length === 1) result.contractsWithOneStory += 1;
    for (const story of contract.stories) {
      const specimens = story?.specimens ?? story?.specimen ?? story?.args?.specimens;
      if (story?.composition?.specimen || story?.specimen || (Array.isArray(specimens) && specimens.length === 1)) result.storiesWithOneSpecimen += 1;
      const expectations = story?.expect ?? story?.expectations ?? story?.assertions;
      if (!Array.isArray(expectations) || expectations.length === 0) result.storiesWithZeroExpectations += 1;
      else result.storiesWithExpectations += 1;
      if (story?.frame || story?.viewport || contract?.frame || contract?.composition?.frame) result.storiesWithFrame += 1;
      if (story?.axes || story?.axis || story?.variants) result.storiesWithAxes += 1;
      for (const key of collectKeys(story)) {
        if (/^(\$node|\$icon|\$handler|\$slot|\$data|\$fixture|\$action|\$state|\$ref)$/.test(key)) increment(result.structuredTagKeys, key);
      }
    }
    result.storyFiles.push(relative(file));
  }
  result.storyFiles.sort();
  return result;
}

function versionInventory() {
  const components = [];
  const versions = [];
  for (const manifestPath of walk(libraryRoot).filter((file) => path.basename(file) === "component.json")) {
    const manifest = readJSON(manifestPath);
    if (!manifest) continue;
    const componentDirectory = path.dirname(manifestPath);
    const componentRelative = path.relative(libraryRoot, componentDirectory).split(path.sep).join("/");
    const versionRoot = path.join(componentDirectory, "versions");
    const component = {
      libraryId: manifest.libraryId ?? "",
      catalogId: manifest.catalogId ?? "",
      displayName: manifest.displayName ?? path.basename(componentDirectory),
      kind: componentRelative.split("/")[0] ?? "",
      name: path.basename(componentDirectory),
      latest: manifest.latest_version ?? manifest.latest ?? "",
      draft: manifest.draft ?? "",
      manifest: relative(manifestPath),
      versions: [],
    };
    if (fs.existsSync(versionRoot)) {
      for (const entry of fs.readdirSync(versionRoot, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
        if (!entry.isDirectory()) continue;
        const version = entry.name;
        component.versions.push(version);
        versions.push({ libraryId: component.libraryId, kind: component.kind, name: component.name, version, path: relative(path.join(versionRoot, version)) });
      }
    }
    component.versions.sort();
    components.push(component);
  }
  components.sort((a, b) => a.libraryId.localeCompare(b.libraryId));
  versions.sort((a, b) => `${a.libraryId}@${a.version}`.localeCompare(`${b.libraryId}@${b.version}`));
  return { components, versions };
}

function pinnedImports(files) {
  const imports = [];
  const pattern = /@vrooli\/react-component-library\/([^'"`\\s)]+)/g;
  for (const file of files) {
    if (!sourceExtensions.has(path.extname(file))) continue;
    const source = fs.readFileSync(file, "utf8");
    for (const match of source.matchAll(pattern)) {
      const specifier = match[1].replace(/[;,]+$/, "");
      const parts = specifier.split("/");
      const versionIndex = parts.findIndex((part) => /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(part));
      if (versionIndex < 0) continue;
      imports.push({ file: relative(file), specifier, asset: parts.slice(0, versionIndex).join("/"), version: parts[versionIndex] });
    }
  }
  imports.sort((a, b) => `${a.file}:${a.specifier}`.localeCompare(`${b.file}:${b.specifier}`));
  return imports;
}

function crossVersionImports(files) {
  const imports = [];
  const pattern = /(?:from\s*|import\s*\(|require\s*\()(['"])(\.\.?\/[^'"\n]*\/versions\/[^'"\n]+)\1/g;
  for (const file of files.filter((item) => sourceExtensions.has(path.extname(item)))) {
    const source = fs.readFileSync(file, "utf8");
    for (const match of source.matchAll(pattern)) imports.push({ file: relative(file), specifier: match[2] });
  }
  imports.sort((a, b) => `${a.file}:${a.specifier}`.localeCompare(`${b.file}:${b.specifier}`));
  return imports;
}

function adopterTests(files) {
  const tests = files.filter((file) => /scenarios\//.test(relative(file)) && !relative(file).startsWith("scenarios/react-component-library/") && /\.(test|spec)\.[jt]sx?$/.test(file));
  const orphan = [];
  const hashes = new Map();
  for (const file of tests) {
    const source = fs.readFileSync(file, "utf8");
    if (!source.includes("@vrooli/react-component-library")) continue;
    const normalized = source.replace(/\/\/.*$/gm, "").replace(/\s+/g, " ").trim();
    const hash = crypto.createHash("sha256").update(normalized).digest("hex");
    const item = { file: relative(file), hash };
    orphan.push(item);
    if (!hashes.has(hash)) hashes.set(hash, []);
    hashes.get(hash).push(item.file);
  }
  const duplicateGroups = [...hashes.entries()].filter(([, paths]) => paths.length > 1).map(([hash, paths]) => ({ hash, files: paths.sort() })).sort((a, b) => a.hash.localeCompare(b.hash));
  return { tests: orphan.sort((a, b) => a.file.localeCompare(b.file)), duplicateGroups };
}

function textStats(files) {
  let translateSites = 0;
  let translateSources = 0;
  let positionalKeys = 0;
  let userStrings = 0;
  const keys = new Set();
  const selectorNames = [];
  let selectorEmitters = 0;
  let inlineStyleOverrides = 0;
  let forwardedDOMProps = 0;
  for (const file of files.filter((item) => sourceExtensions.has(path.extname(item)))) {
    const source = fs.readFileSync(file, "utf8");
    if (/\btranslate\s*\(/.test(source)) translateSources += 1;
    for (const match of source.matchAll(/\btranslate\s*\(\s*["']([^"']+)["']/g)) {
      translateSites += 1;
      keys.add(match[1]);
      if (/(?:^|\.)\d+(?:\.|$)/.test(match[1])) positionalKeys += 1;
    }
    userStrings += [...source.matchAll(/(?:aria-label|placeholder|title|label)\s*=\s*["'][^"']+["']/g)].length;
    const testids = [...source.matchAll(/data-testid\s*=\s*["']([^"']+)["']/g)].map((match) => match[1]);
    if (testids.length) selectorEmitters += 1;
    selectorNames.push(...testids.map((name) => ({ file: relative(file), name })));
    inlineStyleOverrides += [...source.matchAll(/style\s*=|style\s*:/g)].length;
    forwardedDOMProps += [...source.matchAll(/\.\.\.(?:rest|props|inputProps|buttonProps|domProps)\b/g)].length;
  }
  return { translateSources, translateSites, distinctTranslationKeys: keys.size, positionalKeys, userStrings, selectorEmitters, selectorNames: selectorNames.sort((a, b) => `${a.file}:${a.name}`.localeCompare(`${b.file}:${b.name}`)), inlineStyleOverrides, forwardedDOMProps };
}

function packageStats() {
  const manifest = readJSON(packageManifestPath) ?? {};
  const exports = Object.keys(manifest.exports ?? {}).sort();
  const versionedExports = exports.filter((item) => /\/\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\/|$)/.test(item));
  return { exports: exports.length, versionedExports: versionedExports.length, exportNames: exports };
}

export function inspectTidiness(root = repoRoot) {
  const library = path.join(root, "scenarios/react-component-library/library");
  const tools = path.join(root, "scenarios/react-component-library/tools");
  const generatedRoots = ["captures", "docs/evidence", ".vite"].map((relativePath) =>
    path.join(root, "scenarios/react-component-library", relativePath),
  );
  const datedArtifactFiles = generatedRoots
    .flatMap((directory) => walk(directory))
    .filter((file) => /(?:^|\/)(?:20\d{2}-\d{2}-\d{2}|.*\d{4}-\d{2}-\d{2}.*)/.test(file))
    .map((file) => path.relative(root, file).split(path.sep).join("/"))
    .sort();
  const hashNamedTestFiles = walk(library)
    .filter((file) => /\.[0-9a-f]{8}\.test\.tsx$/.test(file))
    .map((file) => path.relative(root, file).split(path.sep).join("/"))
    .sort();
  const unreferencedToolFiles = walk(tools)
    .filter((file) => {
      const relativePath = path.relative(tools, file).split(path.sep).join("/");
      return (
        relativePath !== "capture-assets.sh" &&
        !relativePath.startsWith("preview-runtime-") &&
        !relativePath.startsWith("testdata/")
      );
    })
    .map((file) => path.relative(root, file).split(path.sep).join("/"))
    .sort();
  return { datedArtifactFiles, hashNamedTestFiles, unreferencedToolFiles };
}

function main() {
  const args = process.argv.slice(2);
  const writeIndex = args.indexOf("--write");
  const outputPath = writeIndex >= 0 ? path.resolve(repoRoot, args[writeIndex + 1]) : null;
  const allFiles = walk(repoRoot).filter((file) => textExtensions.has(path.extname(file)));
  const libraryFiles = walk(libraryRoot);
  const inventory = versionInventory();
  const pinned = pinnedImports(allFiles);
  const crossVersion = crossVersionImports(libraryFiles);
  const stories = storyStats(libraryFiles);
  const adopters = adopterTests(allFiles);
  const text = textStats(libraryFiles);
  const templateForks = allFiles.filter((file) => relative(file).startsWith("templates/scenarios/react-vite/ui/src/") && fs.readFileSync(file, "utf8").includes("@vrooliComponentSource")).map(relative).sort();
  const schemaTagKeys = {};
  for (const file of libraryFiles.filter((item) => path.basename(item) === "story.json")) {
    for (const key of collectKeys(readJSON(file))) {
      if (key.startsWith("$")) increment(schemaTagKeys, key);
    }
  }
  const result = {
    schemaVersion: 1,
    root: ".",
    inventory: {
      components: inventory.components,
      componentCount: inventory.components.length,
      versionCount: inventory.versions.length,
      versions: inventory.versions,
      latestCount: inventory.components.filter((item) => item.latest).length,
      draftCount: inventory.components.filter((item) => item.draft).length,
    },
    liveness: {
      pinnedImports: pinned,
      pinnedImportCount: pinned.length,
      pinnedAssets: [...new Set(pinned.map((item) => `${item.asset}@${item.version}`))].sort(),
      crossVersionImports: crossVersion,
      crossVersionImportCount: crossVersion.length,
    },
    package: packageStats(),
    stories,
    adopters: {
      adopterTestCount: adopters.tests.length,
      tests: adopters.tests,
      duplicateBodyGroups: adopters.duplicateGroups,
      duplicateBodyGroupCount: adopters.duplicateGroups.length,
    },
    sourceQuality: text,
    templates: { forkCount: templateForks.length, forks: templateForks },
    schema: { structuredTagKeys: schemaTagKeys },
    editor: {
      controller: "scenarios/react-component-library/ui/src/features/components/ComponentEditorController.tsx",
      hasHoverProvider: fs.readFileSync(path.join(repoRoot, "scenarios/react-component-library/ui/src/features/components/ComponentEditorController.tsx"), "utf8").includes("registerHoverProvider"),
      hasDefinitionProvider: fs.readFileSync(path.join(repoRoot, "scenarios/react-component-library/ui/src/features/components/ComponentEditorController.tsx"), "utf8").includes("registerDefinitionProvider"),
    },
    tidiness: inspectTidiness(),
  };
  const serialized = `${JSON.stringify(result, null, 2)}\n`;
  if (outputPath) {
    fs.mkdirSync(path.dirname(outputPath), { recursive: true });
    fs.writeFileSync(outputPath, serialized);
  }
  process.stdout.write(serialized);
  if (
    result.tidiness.datedArtifactFiles.length ||
    result.tidiness.hashNamedTestFiles.length ||
    result.tidiness.unreferencedToolFiles.length
  ) process.exitCode = 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
