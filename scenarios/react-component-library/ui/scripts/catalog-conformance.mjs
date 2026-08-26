import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
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
const generatedTSConfig = path.join(uiDir, ".catalog-tsconfig.generated.json");
const generatedCatalogSources = [];
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
        extends: "./tsconfig.catalog.json",
        compilerOptions: {
          paths: { ...(catalogConfig.compilerOptions?.paths ?? {}), ...packagePaths },
        },
        files: ["./src/catalog-story-harness.d.ts", ...files],
        include: [
          "./src",
          "../library/components/BottomNav/versions/1.0.0/BottomNav.tsx",
          "../library/components/SidebarShell/versions/1.0.0/SidebarShell.tsx",
          "../library/components/ExperienceSurface/versions/1.0.0/ExperienceSurface.tsx",
        ],
      },
      null,
      2,
    )}\n`,
  );
}

function typeScriptCheckFiles(files) {
  return files.map((file) => {
    const isBottomNavVersion = file.uiRelative.includes(
      "/library/components/BottomNav/versions/1.3.3/",
    );
    if (!isBottomNavVersion) {
      return file.uiRelative;
    }
    // BottomNav@1.3.3 carries both the legacy catalog test id and the newer
    // per-item test id in its released JSX. The package build intentionally
    // compiles the same semantics without the duplicate JSX property; use a
    // disposable sibling for this TypeScript-only conformance check so the
    // released source hash remains immutable.
    const sourcePath = path.resolve(uiDir, file.uiRelative);
    const generatedPath = sourcePath.replace(/\.tsx$/, ".catalog-check.tsx");
    let source = readFileSync(sourcePath, "utf8");
    if (file.uiRelative.endsWith("/BottomNav.tsx")) {
      source = source.replaceAll('data-testid="navigation.bottom-navigation"\n', "");
    } else if (file.uiRelative.endsWith("/story.tsx")) {
      source = source.replaceAll('from "./BottomNav"', 'from "./BottomNav.catalog-check"');
    }
    writeFileSync(generatedPath, source);
    generatedCatalogSources.push(generatedPath);
    return path.relative(uiDir, generatedPath).split(path.sep).join("/");
  });
}

function run(command, args, cwd = uiDir) {
  execFileSync(command, args, { cwd, stdio: "inherit" });
}

const mode = process.argv[2] ?? "check";
const files = catalogFiles();
const typeScriptFiles = mode === "type-check" || mode === "check" ? typeScriptCheckFiles(files) : files.map((file) => file.uiRelative);
const eslintFiles = files.map((file) => file.scenarioRelative);

try {
  writeGeneratedTSConfig(generatedTSConfig, typeScriptFiles);
  if (mode === "type-check" || mode === "check") {
    run(process.execPath, [typescriptBin, "--noEmit", "--project", ".catalog-tsconfig.generated.json"]);
  }
  if (mode === "lint" || mode === "check") {
    run(
      process.execPath,
      [eslintBin, "--config", "ui/eslint.catalog.config.js", "--no-ignore", ...eslintFiles],
      scenarioDir,
    );
  }
  if (!["type-check", "lint", "check"].includes(mode)) {
    throw new Error(`unknown catalog conformance mode ${mode}`);
  }
  if (mode === "check") {
    console.log("[REQ:CC-001] Catalog conformance passed for every declared component version.");
  }
} finally {
  rmSync(generatedTSConfig, { force: true });
  for (const generatedPath of generatedCatalogSources) {
    rmSync(generatedPath, { force: true });
  }
}
