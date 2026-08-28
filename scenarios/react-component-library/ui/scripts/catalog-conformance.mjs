import { spawn } from "node:child_process";
import { existsSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

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
  const packagePaths = Object.fromEntries(
    Object.entries(packageManifest.exports ?? {})
      .filter(([key, value]) => key.startsWith("./") && value && typeof value.types === "string")
      .map(([key, value]) => {
        const artifact = path.resolve(packageRoot, value.types);
        const match = value.types.match(/^\.\/dist\/(foundations|hooks|services|primitives|components)\/([^/]+)\/versions\/([^/]+)\/([^/.]+)\.d\.ts$/);
        if (!match) return [`@vrooli/react-component-library/${key.slice(2)}`, [artifact]];
        return [
          `@vrooli/react-component-library/${key.slice(2)}`,
          [artifact],
        ];
      }),
  );
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

async function run(command, args, cwd = uiDir, environment = {}) {
  const label = path.basename(args[0] ?? command);
  const child = spawn(command, args, {
    cwd,
    env: { ...process.env, ...environment },
    stdio: "inherit",
  });
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
  rmSync(scratchDir, { force: true, recursive: true });
}
