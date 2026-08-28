import { cp, mkdir, readFile, readdir, rename, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, dirname, relative as relativePath } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { authoredRoot } from "./catalog-source.mjs";
import { resolveCatalogExports } from "./export-resolution.mjs";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/scripts$/, "");
const sourceRoot = authoredRoot;
const checkOnly = process.argv.includes("--check-only");
const styleFiles = [];
const collectSourceFiles = async (root) => {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) await collectSourceFiles(path);
    else if (entry.name.endsWith(".css")) styleFiles.push(path);
  }
};
await collectSourceFiles(sourceRoot);

const { resolutions: exportResolutions } = await resolveCatalogExports({
  libraryRoot: sourceRoot,
  manifestRoot: authoredRoot,
});
const generatedConfigPath = join(packageRoot, ".build-tsconfig.json");
const baseConfig = JSON.parse(await readFile(join(packageRoot, "tsconfig.build.json"), "utf8"));
const packageNodeModules = join(packageRoot, "node_modules");
const packageDependencyPaths = {
  react: [join(packageNodeModules, "@types", "react", "index.d.ts")],
  "react/*": [join(packageNodeModules, "@types", "react", "*")],
  "react/jsx-runtime": [join(packageNodeModules, "@types", "react", "jsx-runtime.d.ts")],
  "react-dom": [join(packageNodeModules, "@types", "react-dom", "index.d.ts")],
  "lucide-react": [join(packageNodeModules, "lucide-react")],
  clsx: [join(packageNodeModules, "clsx")],
  "tailwind-merge": [join(packageNodeModules, "tailwind-merge")],
  shiki: [join(packageNodeModules, "shiki")],
  "react-markdown": [join(packageNodeModules, "react-markdown")],
  "remark-gfm": [join(packageNodeModules, "remark-gfm")],
  mermaid: [join(packageNodeModules, "mermaid")],
  "@vrooli/audio-capture-browser": [join(packageRoot, "..", "audio-capture-browser", "dist", "index.d.ts")],
};
const selfPaths = Object.fromEntries(Object.entries(exportResolutions).map(([subpath, resolution]) => [
  `@vrooli/react-component-library/${subpath.slice(2)}`,
  [relativePath(packageRoot, join(sourceRoot, resolution.source)).replaceAll("\\", "/")],
]));
baseConfig.compilerOptions.rootDir = sourceRoot;
baseConfig.compilerOptions.baseUrl = packageRoot;
baseConfig.compilerOptions.paths = { ...packageDependencyPaths, ...selfPaths };
baseConfig.include = [join(sourceRoot, "**/*.ts"), join(sourceRoot, "**/*.tsx")];
baseConfig.exclude = [
  join(sourceRoot, "**/story.tsx"),
  join(sourceRoot, "**/story.json"),
  join(sourceRoot, "**/*.test.ts"),
  join(sourceRoot, "**/*.test.tsx"),
  join(sourceRoot, "**/*.spec.ts"),
  join(sourceRoot, "**/*.spec.tsx"),
  join(sourceRoot, "story-contracts.spec.ts"),
  join(sourceRoot, "preview-harnesses/**"),
];
await writeFile(generatedConfigPath, `${JSON.stringify(baseConfig, null, 2)}\n`);

const tsc = join(packageRoot, "node_modules", ".bin", "tsc");

if (checkOnly) {
  if (!tsc || !existsSync(tsc)) {
    console.error("TypeScript compiler not found in the package toolchain or configured build environment");
    process.exit(1);
  }
  const result = spawnSync(tsc, ["-p", generatedConfigPath, "--noEmit"], { cwd: packageRoot, stdio: "inherit" });
  await rm(generatedConfigPath, { force: true });
  if (result.status !== 0) process.exit(result.status ?? 1);
  console.log("Checked @vrooli/react-component-library sources.");
  process.exit(0);
}

// The artifact is built into a staging directory and published with a rename.
// Compiling straight into `dist` meant deleting it first, so for the whole
// length of a tsc run anything reading the package — a dev server, a test
// suite, the running scenario that triggers this build — resolved modules that
// did not exist. Staging narrows that window to a single rename, and a failed
// build now leaves the previous artifact in place instead of no artifact.
const distRoot = join(packageRoot, "dist");
const stagingRoot = join(packageRoot, "dist.staging");
const retiredRoot = join(packageRoot, "dist.previous");
await rm(stagingRoot, { recursive: true, force: true });
await rm(retiredRoot, { recursive: true, force: true });
await mkdir(stagingRoot, { recursive: true });

const abandonStaging = async (code) => {
  await rm(stagingRoot, { recursive: true, force: true });
  await rm(generatedConfigPath, { force: true });
  process.exit(code ?? 1);
};
const sync = spawnSync(process.execPath, [join(packageRoot, "scripts", "sync-exports.mjs")], { cwd: packageRoot, stdio: "inherit" });
if (sync.status !== 0) process.exit(sync.status ?? 1);
if (!tsc || !existsSync(tsc)) {
  console.error("TypeScript compiler not found in the package toolchain or configured build environment");
  process.exit(1);
}
const result = spawnSync(tsc, ["-p", generatedConfigPath, "--outDir", stagingRoot], { cwd: packageRoot, stdio: "inherit" });
if (result.status !== 0) await abandonStaging(result.status);

for (const file of styleFiles) {
  const relative = file.slice(sourceRoot.length + 1);
  await mkdir(join(stagingRoot, dirname(relative)), { recursive: true });
  await cp(file, join(stagingRoot, relative));
}

// TypeScript's bundler resolution intentionally leaves relative imports
// extensionless. The published package is native ESM, however, so Node's
// resolver requires the emitted .js suffix for every relative module edge.
// Keep authored library sources ergonomic and normalize only the disposable
// package artifact after compilation.
const rewriteRelativeImports = async (root) => {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      await rewriteRelativeImports(path);
      continue;
    }
    if (!path.endsWith(".js")) continue;
    const source = await readFile(path, "utf8");
    const rewritten = source.replace(
      /((?:from\s+|import\s*\()(["']))(\.[^"']+)(\2)/g,
      (_match, prefix, _quote, modulePath, suffix) =>
        /\.[a-z0-9]+$/i.test(modulePath) ? `${prefix}${modulePath}${suffix}` : `${prefix}${modulePath}.js${suffix}`,
    );
    if (rewritten !== source) await writeFile(path, rewritten);
  }
};
await rewriteRelativeImports(stagingRoot);

// Generate the exact, major-scoped, and bare aliases from the same
// manifest-aware resolution used by sync-exports. Deprecated versions and
// drafts therefore cannot leak back into the package through filename order.
const aliasTargets = new Map();
for (const [subpath, resolution] of Object.entries(exportResolutions)) {
  const alias = subpath.slice(2);
  const relativeSource = resolution.source.replace(/\.(?:tsx?|jsx?)$/, ".js");
  aliasTargets.set(subpath, {
    target: join(stagingRoot, "exports", `${alias}.js`),
    relativeSource,
    version: resolution.version,
  });
}
const writeAlias = async (target, relativeSource) => {
  await mkdir(dirname(target), { recursive: true });
  const fromAlias = relativePath(dirname(target), join(stagingRoot, relativeSource)).replaceAll("\\", "/");
  const specifier = fromAlias.startsWith(".") ? fromAlias : `./${fromAlias}`;
  // `export *` never forwards a default binding, so an asset authored with a
  // default export would resolve through the alias with its primary component
  // missing. Forward it explicitly for the assets that have one.
  const emitted = join(stagingRoot, relativeSource);
  const hasDefault = existsSync(emitted)
    && /(^|\n)export default[\s({]|\bas default\b/.test(await readFile(emitted, "utf8"));
  const defaultLine = hasDefault ? `export { default } from "${specifier}";\n` : "";
  const defaultTypeLine = hasDefault ? `export { default } from "${specifier.replace(/\.js$/, "")}";\n` : "";
  await writeFile(target, `export * from "${specifier}";\n${defaultLine}`);
  await writeFile(target.replace(/\.js$/, ".d.ts"), `export * from "${specifier.replace(/\.js$/, "")}";\n${defaultTypeLine}`);
};
for (const { target, relativeSource } of aliasTargets.values()) await writeAlias(target, relativeSource);
await mkdir(join(stagingRoot, "exports"), { recursive: true });
const bareAliases = [...aliasTargets.keys()]
  .filter((subpath) => subpath.slice(2).split("/").length === 1)
  .map((subpath) => subpath.slice(2))
  .sort();
const indexEntries = bareAliases.map((name) => `export * from "./${name}.js";`).join("\n");
await writeFile(join(stagingRoot, "exports", "index.js"), `${indexEntries}\n`);
await writeFile(join(stagingRoot, "exports", "index.d.ts"), `${indexEntries.replaceAll(".js", "")}\n`);
const resolutionReport = Object.fromEntries(
  [...aliasTargets.entries()].sort(([left], [right]) => left.localeCompare(right)).map(([subpath, value]) => [
    subpath,
    {
      version: value.version,
      source: `./${value.relativeSource}`,
    },
  ]),
);
await writeFile(join(stagingRoot, "exports", "resolution.json"), `${JSON.stringify(resolutionReport, null, 2)}\n`);

// Bundler-injected environment never survives the trip to a consumer. Vite
// substitutes the literal token `import.meta.env`; an alias through a local
// defeats that substitution, and the consumer's minifier re-emits a member
// read that is undefined in every browser. Vitest hides this because
// vite-node materializes import.meta.env, so the artifact is the only place
// the contract can be checked honestly. `import.meta.url` stays legal.
//
// A released version that already shipped such a read cannot be edited in
// place, so it would need a dated exception here keyed by its emitted path.
// There are none today; keep it that way by fixing the source, not the guard.
const metaReadExceptions = new Map();
const collectMetaReads = async (root, base = root) => {
  const offenders = [];
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      offenders.push(...await collectMetaReads(path, base));
      continue;
    }
    if (!path.endsWith(".js")) continue;
    const relative = relativePath(base, path).replaceAll("\\", "/");
    if (metaReadExceptions.has(relative)) continue;
    // Comments are allowed to name the hazard; only executable reads fail.
    const source = await readFile(path, "utf8");
    const executable = source
      .replace(/\/\*[\s\S]*?\*\//g, (block) => block.replace(/[^\n]/g, " "))
      .replace(/\/\/[^\n]*/g, "");
    executable.split("\n").forEach((line, index) => {
      if (/import\.meta(?!\.url\b)/.test(line)) {
        offenders.push(`${relative}:${index + 1}: ${line.trim()}`);
      }
    });
  }
  return offenders;
};
const metaReads = await collectMetaReads(stagingRoot);
const staleExceptions = [...metaReadExceptions.keys()].filter((relative) => !existsSync(join(stagingRoot, relative)));
if (metaReads.length > 0 || staleExceptions.length > 0) {
  console.error("react-component-library build failed: bundler-injected import.meta contract violated");
  for (const offender of metaReads) console.error(`  reads import.meta: ${offender}`);
  for (const relative of staleExceptions) console.error(`  stale exception for a version that is no longer emitted: ${relative}`);
  console.error("Derive the behavior from a runtime-observable signal instead; see .ast-grep/rules/no-bundler-env-in-shared-library.yml");
  await rm(stagingRoot, { recursive: true, force: true });
  process.exit(1);
}

// Publish. Two renames rather than a delete-and-repopulate, so a reader is
// only ever between artifacts for the duration of one syscall.
if (existsSync(distRoot)) await rename(distRoot, retiredRoot);
await rename(stagingRoot, distRoot);
await rm(retiredRoot, { recursive: true, force: true });
await rm(generatedConfigPath, { force: true });

console.log("Built @vrooli/react-component-library with declarations.");
