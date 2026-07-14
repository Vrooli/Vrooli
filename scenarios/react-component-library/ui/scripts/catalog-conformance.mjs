import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const uiDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const scenarioDir = path.resolve(uiDir, "..");
const componentsDir = path.join(scenarioDir, "library", "components");
const eslintBin = path.join(uiDir, "node_modules", "eslint", "bin", "eslint.js");
const generatedTSConfig = path.join(uiDir, ".catalog-tsconfig.generated.json");

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
      .filter((entry) => entry.isFile() && /\.tsx?$/.test(entry.name))
      .map((entry) => path.join(versionDir, entry.name))
      .sort((a, b) => a.localeCompare(b));
    if (sourceFiles.length === 0) {
      throw new Error(`${manifestPath} version ${version} has no top-level TypeScript source files`);
    }
    return sourceFiles.map((filePath) => {
      const scenarioRelative = path.relative(scenarioDir, filePath);
      if (scenarioRelative.startsWith("..") || path.isAbsolute(scenarioRelative)) {
        throw new Error(`${filePath} escapes the scenario conformance boundary`);
      }
      const relative = path.relative(uiDir, filePath);
      return {
        scenarioRelative: scenarioRelative.split(path.sep).join("/"),
        uiRelative: relative.split(path.sep).join("/"),
      };
    });
  });
}

function catalogFiles() {
  return readdirSync(componentsDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .flatMap((entry) => versionPaths(path.join(componentsDir, entry.name, "component.json")))
    .sort((a, b) => a.uiRelative.localeCompare(b.uiRelative));
}

function writeGeneratedTSConfig(outputPath, files) {
  writeFileSync(
    outputPath,
    `${JSON.stringify({ extends: "./tsconfig.catalog.json", files, include: [] }, null, 2)}\n`,
  );
}

function run(command, args, cwd = uiDir) {
  execFileSync(command, args, { cwd, stdio: "inherit" });
}

const mode = process.argv[2] ?? "check";
const files = catalogFiles();
const typeScriptFiles = files.map((file) => file.uiRelative);
const eslintFiles = files.map((file) => file.scenarioRelative);

try {
  writeGeneratedTSConfig(generatedTSConfig, typeScriptFiles);
  if (mode === "type-check" || mode === "check") {
    run("pnpm", ["exec", "tsc", "--noEmit", "--project", ".catalog-tsconfig.generated.json"]);
  }
  if (mode === "lint" || mode === "check") {
    run(process.execPath, [eslintBin, "--config", "ui/eslint.catalog.config.js", "--no-ignore", ...eslintFiles], scenarioDir);
  }
  if (!["type-check", "lint", "check"].includes(mode)) {
    throw new Error(`unknown catalog conformance mode ${mode}`);
  }
  if (mode === "check") {
    console.log("[REQ:CC-001] Catalog conformance passed for every declared component version.");
  }
} finally {
  rmSync(generatedTSConfig, { force: true });
}
