import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const uiDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const scenarioDir = path.resolve(uiDir, "..");
const assetRoots = [
  path.join(scenarioDir, "library", "foundations"),
  path.join(scenarioDir, "library", "hooks"),
  path.join(scenarioDir, "library", "services"),
  path.join(scenarioDir, "library", "primitives"),
  path.join(scenarioDir, "library", "components"),
];
const eslintBin = path.join(uiDir, "node_modules", "eslint", "bin", "eslint.js");
const typescriptBin = path.join(uiDir, "node_modules", "typescript", "bin", "tsc");
// Keep the isolated project beneath uiDir so TypeScript's normal ancestor-based
// node_modules lookup remains valid and ESLint can lint disposable sources
// without treating them as outside the scenario boundary.
const scratchDir = mkdtempSync(path.join(uiDir, ".catalog-conformance-"));
const generatedTSConfig = path.join(scratchDir, "catalog-tsconfig.json");
const packageRoot = path.resolve(uiDir, "../../../packages/react-component-library");
const packageManifest = readJSON(path.join(packageRoot, "package.json"));
const libraryRoot = path.join(scenarioDir, "library");
// Composition between library assets is an edge inside the source graph. The
// package build resolves it from each version's generated lock and the Vite
// plugin resolves it from the authored tree — it refuses a `dist` root
// outright. The type-checker resolved the same specifier through the built
// declarations, so a generated export map that had drifted produced "cannot
// find module" plus a cascade of ordinary-looking type errors inside frozen
// released source. Resolving from the same authored source the other two
// tools use removes that third answer.
const { resolveCatalogExports } = await import(
  pathToFileURL(path.join(packageRoot, "scripts", "export-resolution.mjs")).href
);
// Export resolution rejects an inconsistent catalog — a manifest whose latest
// no longer names the highest release, a version with no public entry module.
// Publishing writes the version directory before it updates the manifest, so a
// run that lands inside that window sees a real inconsistency. Report it as a
// catalog problem with the command that diagnoses it, rather than as a stack
// trace from a module the caller never invoked.
let sourceAssets;
let sourceResolutions;
try {
  ({ assets: sourceAssets, resolutions: sourceResolutions } = await resolveCatalogExports({
    libraryRoot,
    manifestRoot: libraryRoot,
  }));
} catch (error) {
  throw new Error(
    `the catalog cannot be resolved, so component versions cannot be type-checked against their sources.\n` +
      `Run \`pnpm run exports:check\` for the full report.\n${error.message}`,
    { cause: error },
  );
}
const activeVersionsByAsset = new Map(
  sourceAssets.map((asset) => [asset.name, new Set(asset.activeVersions)]),
);

// A deprecated release is frozen: its source cannot be corrected and its
// declaration is generated from those same frozen bytes, so the two cannot
// drift apart. Reading its body would only report findings nobody may act on,
// so deprecated pins keep resolving through the published declaration while
// every live version resolves to source.
function isDeprecatedPin(subpath) {
  const segments = subpath.slice(2).split("/");
  const version = segments.at(-1);
  if (!/^\d+\.\d+\.\d+$/.test(version ?? "")) return false;
  const active = activeVersionsByAsset.get(segments[0]);
  return active ? !active.has(version) : false;
}

function readJSON(filePath) {
  return JSON.parse(readFileSync(filePath, "utf8"));
}

function componentName(manifest) {
  const libraryID = String(manifest.libraryId ?? "");
  const [, name] = libraryID.split(":");
  return name || "";
}

function versionPaths(manifestPath) {
  const manifest = readJSON(manifestPath);
  const name = componentName(manifest);
  if (!name) {
    throw new Error(`${manifestPath} is missing a libraryId component name`);
  }

  const versions = [manifest.latest, manifest.draft]
    .map((version) => String(version ?? "").trim())
    .filter(Boolean);
  if (versions.length === 0) {
    throw new Error(`${manifestPath} does not declare a latest or draft version`);
  }

  const root = path.dirname(manifestPath);
  return [...new Set(versions)].flatMap((version) => {
    const versionDir = path.join(root, "versions", version);
    if (!existsSync(versionDir)) {
      throw new Error(`${manifestPath} points at missing version directory ${versionDir}`);
    }
    const sourceFiles = readdirSync(versionDir, { withFileTypes: true })
      .filter((entry) => entry.isFile() && /\.tsx?$/.test(entry.name) && !/\.(?:test|spec)\.[^.]+$/.test(entry.name))
      .map((entry) => path.join(versionDir, entry.name))
      .sort((a, b) => a.localeCompare(b));
    if (sourceFiles.length === 0) {
      throw new Error(
        `${manifestPath} version ${version} has no top-level TypeScript source files`,
      );
    }
    return sourceFiles.map((filePath) => {
      const scenarioRelative = path.relative(scenarioDir, filePath);
      if (scenarioRelative.startsWith("..") || path.isAbsolute(scenarioRelative)) {
        throw new Error(`${filePath} escapes the scenario conformance boundary`);
      }
      validateVersionLocalImports(filePath);
      const relative = path.relative(uiDir, filePath);
      return {
        scenarioRelative: scenarioRelative.split(path.sep).join("/"),
        uiRelative: relative.split(path.sep).join("/"),
      };
    });
  });
}

function validateVersionLocalImports(filePath) {
  const source = readFileSync(filePath, "utf8");
  if (/runtimePhase\d+|shared\/runtime/.test(source)) {
    throw new Error(
      `${filePath} contains a legacy shared runtime reference; ` +
        "published assets must expose their real version-local source",
    );
  }
  const relativeImports = /\b(?:from\s*|import\s*\()\s*["'](\.{1,2}\/[^"']+)["']/g;
  for (const match of source.matchAll(relativeImports)) {
    const specifier = match[1];
    const resolved = path.resolve(path.dirname(filePath), specifier);
    const isPublishedAsset = path.relative(scenarioDir, filePath).startsWith(`library${path.sep}`);
    const reachesSharedRuntime = path
      .relative(scenarioDir, resolved)
      .split(path.sep)
      .includes("shared");
    if (isPublishedAsset && reachesSharedRuntime) {
      throw new Error(
        `${filePath} imports ${specifier} outside its version directory; ` +
          "released asset source must not depend on shared runtime shells",
      );
    }
  }
}

function catalogFiles() {
  return assetRoots
    .flatMap((assetRoot) =>
      readdirSync(assetRoot, { withFileTypes: true })
        .filter((entry) => entry.isDirectory() && entry.name !== "shared")
        .flatMap((entry) => versionPaths(path.join(assetRoot, entry.name, "component.json"))),
    )
    .sort((a, b) => a.uiRelative.localeCompare(b.uiRelative));
}

function writeGeneratedTSConfig(outputPath, files) {
  const catalogConfig = readJSON(path.join(uiDir, "tsconfig.catalog.json"));
  // Every subpath the authored tree can answer resolves to that source, so the
  // type-checker reads the same bytes the build compiles and the dev server
  // serves. A new asset therefore type-checks before the export map is
  // regenerated, rather than failing inside an unrelated consumer.
  const packagePaths = {};
  for (const [subpath, resolution] of Object.entries(sourceResolutions)) {
    if (isDeprecatedPin(subpath)) continue;
    packagePaths[`@vrooli/react-component-library/${subpath.slice(2)}`] = [
      path.join(libraryRoot, resolution.source),
    ];
  }
  // Released versions evicted from the working tree are still pinned by older
  // releases and have no source left to point at, so they keep resolving
  // through the published declaration. A subpath the export map advertises but
  // the package never emitted gets no mapping at all: an import that actually
  // reaches one then fails at the importing site, naming the specifier, rather
  // than silently resolving to a file that is not there.
  const unbuilt = [];
  for (const [key, value] of Object.entries(packageManifest.exports ?? {})) {
    if (!key.startsWith("./") || !value || typeof value.types !== "string") continue;
    const specifier = `@vrooli/react-component-library/${key.slice(2)}`;
    if (packagePaths[specifier]) continue;
    const artifact = path.resolve(packageRoot, value.types);
    if (!existsSync(artifact)) {
      unbuilt.push(specifier);
      continue;
    }
    packagePaths[specifier] = [artifact];
  }
  if (unbuilt.length > 0) {
    console.warn(
      `[catalog-conformance] ${unbuilt.length} export subpath(s) are advertised by the package ` +
        `manifest but have neither authored source nor a built declaration (first: ${unbuilt
          .slice(0, 3)
          .join(", ")}). External consumers importing them resolve nothing.`,
    );
  }
  writeFileSync(
    outputPath,
    `${JSON.stringify(
      {
        extends: path.join(uiDir, "tsconfig.catalog.json"),
        compilerOptions: {
          paths: { ...(catalogConfig.compilerOptions?.paths ?? {}), ...packagePaths },
        },
        files: [path.join(uiDir, "src/catalog-story-harness.d.ts"), ...files],
        include: [path.join(uiDir, "src")],
      },
      null,
      2,
    )}\n`,
  );
}

function typeScriptCheckFiles(files) {
  return files.map((file) => path.resolve(uiDir, file.uiRelative));
}

function eslintCheckFiles(files) {
  return files.map((file) => path.resolve(scenarioDir, file.scenarioRelative));
}

// Diagnostics are captured as well as shown when a report is requested, so the
// Go gate can attribute a failure to the assets whose files it names instead of
// recording one unattributed corpus finding. Output is still streamed to the
// terminal in both modes; the capture is additive.
const reportPath = process.env.RCL_CATALOG_REPORT ?? "";
const captured = [];

async function run(command, args, cwd = uiDir, environment = {}) {
  const label = path.basename(args[0] ?? command);
  const child = spawn(command, args, {
    cwd,
    env: { ...process.env, ...environment },
    stdio: reportPath ? ["inherit", "pipe", "pipe"] : "inherit",
  });
  if (reportPath) {
    for (const stream of [child.stdout, child.stderr]) {
      stream?.setEncoding("utf8");
      stream?.on("data", (chunk) => {
        captured.push(chunk);
        process.stdout.write(chunk);
      });
    }
  }
  const heartbeat = setInterval(() => {
    console.log(`[catalog-conformance] ${label} is still running`);
  }, 10_000);
  try {
    await new Promise((resolve, reject) => {
      child.once("error", reject);
      child.once("exit", (code, signal) => {
        if (code === 0) resolve();
        else reject(new Error(`${label} exited with ${code ?? signal ?? "unknown status"}`));
      });
    });
  } finally {
    clearInterval(heartbeat);
  }
}

const mode = process.argv[2] ?? "check";
const files = catalogFiles();
const typeScriptFiles = ["type-check", "lint", "check"].includes(mode)
  ? typeScriptCheckFiles(files)
  : files.map((file) => file.uiRelative);
const eslintFiles = mode === "lint" || mode === "check" ? eslintCheckFiles(files) : files.map((file) => file.scenarioRelative);

try {
  writeGeneratedTSConfig(generatedTSConfig, typeScriptFiles);
  if (mode === "type-check" || mode === "check") {
    await run(process.execPath, [typescriptBin, "--noEmit", "--project", generatedTSConfig]);
  }
  if (mode === "lint" || mode === "check") {
    await run(
      process.execPath,
      [eslintBin, "--config", "ui/eslint.catalog.config.js", "--no-ignore", ...eslintFiles],
      scenarioDir,
      { RCL_CATALOG_TSCONFIG: generatedTSConfig },
    );
  }
  if (!["type-check", "lint", "check"].includes(mode)) {
    throw new Error(`unknown catalog conformance mode ${mode}`);
  }
  if (mode === "check") {
    console.log("[REQ:CC-001] Catalog conformance passed for every declared component version.");
  }
} finally {
  if (reportPath) writeReport();
  rmSync(scratchDir, { force: true, recursive: true });
}

// Both tools name the offending file on its own terms: tsc as
// `path(line,col): error TSxxxx: message`, ESLint as a bare path line followed
// by indented `line:col severity message` rows. Neither emits JSON that covers
// the other, so the two shapes are normalized here — at the point where the
// absolute paths are still unambiguous — rather than re-derived in Go.
function writeReport() {
  const text = captured.join("");
  const diagnostics = [];
  const tsc = /^(\S.*?)\((\d+),(\d+)\):\s+(error|warning)\s+(TS\d+:\s.*)$/gm;
  for (const match of text.matchAll(tsc)) {
    diagnostics.push({ file: absolute(match[1]), line: Number(match[2]), severity: match[4], message: match[5], source: "tsc" });
  }
  let eslintFile = "";
  for (const rawLine of text.split("\n")) {
    const line = rawLine.replace(/\u001b\[[0-9;]*m/g, "").trimEnd();
    const header = /^(\/.*\.[cm]?[jt]sx?)$/.exec(line);
    if (header) {
      eslintFile = header[1];
      continue;
    }
    const row = /^\s+(\d+):(\d+)\s+(error|warning)\s+(.*?)(?:\s{2,}([\w@/-]+))?$/.exec(line);
    if (row && eslintFile) {
      diagnostics.push({ file: eslintFile, line: Number(row[1]), severity: row[3], message: `${row[4]}${row[5] ? ` (${row[5]})` : ""}`, source: "eslint" });
    }
  }
  writeFileSync(reportPath, `${JSON.stringify({ schemaVersion: 1, mode, diagnostics }, null, 2)}\n`);
}

function absolute(candidate) {
  return path.isAbsolute(candidate) ? candidate : path.resolve(uiDir, candidate);
}
