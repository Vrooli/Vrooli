import { cp, mkdir, readFile, readdir, rename, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, dirname, relative as relativePath } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { authoredRoot, projectCatalogSource } from "./catalog-source.mjs";
import { resolveCatalogExports } from "./export-resolution.mjs";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/scripts$/, "");
const sourceRoot = join(packageRoot, ".build-source");
const checkOnly = process.argv.includes("--check-only");
// The catalog library remains the only authored source of truth. The package
// build uses a disposable transformed tree because released versions may
// intentionally import an older compatibility path while the package artifact
// must compile against the latest compatible foundation implementation.
await projectCatalogSource(sourceRoot);
// Historical released entries can still point at the 1.0.1 locale path while
// the authored source migrates to the provider-backed API. Keep that
// compatibility surface in the disposable build tree only; it must never be
// written back into the catalog's released version directory.
const legacyLocaleCompat = `import { createContext, createElement, useContext, type ReactNode } from "react";

export type StringDefaults = Readonly<Record<string, string>>;
export interface DefinedStrings { namespace: string; defaults: StringDefaults; }
export interface LibraryStringsProviderProps {
  children: ReactNode;
  strings?: DefinedStrings | readonly DefinedStrings[];
  translate?: (key: string, fallback: string) => string;
}
interface StringsContextValue { translate: (key: string, fallback: string) => string; }
const StringsContext = createContext<StringsContextValue | null>(null);
export function defineStrings(namespace: string, defaults: StringDefaults): DefinedStrings {
  return { namespace, defaults };
}
function flattenStrings(strings: DefinedStrings | readonly DefinedStrings[] | undefined): StringDefaults {
  const entries = Array.isArray(strings) ? strings : strings ? [strings] : [];
  return Object.fromEntries(entries.flatMap((entry) => Object.entries(entry.defaults)));
}
export function LibraryStringsProvider({ children, strings, translate }: LibraryStringsProviderProps) {
  const defaults = flattenStrings(strings);
  const value: StringsContextValue = {
    translate: (key, fallback) => translate?.(key, defaults[key] ?? fallback) ?? defaults[key] ?? fallback,
  };
  return createElement(StringsContext.Provider, { value }, children);
}
export function useStrings(): (key: string, fallback: string) => string;
export function useStrings(key: string, fallback: string): string;
export function useStrings(key?: string, fallback?: string): ((key: string, fallback: string) => string) | string {
  const context = useContext(StringsContext);
  const resolver = (nextKey: string, nextFallback: string) => context?.translate(nextKey, nextFallback) ?? nextFallback;
  return key === undefined ? resolver : resolver(key, fallback ?? "");
}
export function useLocale() {
  return typeof document !== "undefined" ? document.documentElement.lang || "en" : "en";
}
export function translate(key: string, fallback: string): string { return fallback; }
`;
await writeFile(join(sourceRoot, "hooks/useLocale/versions/1.0.1/useLocale.ts"), legacyLocaleCompat);
const sourceFiles = [];
const styleFiles = [];
const pascalCase = (value) => value.split("-").map((part) => part.slice(0, 1).toUpperCase() + part.slice(1)).join("");
const collectSourceFiles = async (root) => {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) await collectSourceFiles(path);
    else if (/\.(?:ts|tsx)$/.test(entry.name) && entry.name !== "story.tsx" && !/\.(?:test|spec)\.(?:ts|tsx)$/.test(entry.name)) sourceFiles.push(path);
    else if (entry.name.endsWith(".css")) styleFiles.push(path);
  }
};
await collectSourceFiles(sourceRoot);

const packageTargets = new Map();
for (const file of sourceFiles) {
  const relative = file.slice(sourceRoot.length + 1).replaceAll("\\", "/");
  const match = relative.match(/^(?:components|primitives|hooks|foundations|services)\/([^/]+)\/versions\/([^/]+)\/([^/]+)\.(?:ts|tsx)$/);
  // Only the version entry module is a package subpath. Helpers such as
  // styles.ts may live beside it, but must not shadow the public entry when
  // transforming package imports in the disposable build tree.
  if (match && (match[3] === match[1] || match[3] === pascalCase(match[1]))) {
    packageTargets.set(`${match[1]}/${match[2]}`, file);
  }
}
const { resolutions: exportResolutions } = await resolveCatalogExports({
  libraryRoot: sourceRoot,
  manifestRoot: authoredRoot,
});
// The package compiles its authored graph before alias entrypoints exist in
// dist. Resolve manifest-backed major aliases inside the disposable source
// tree so library-owned dependencies can use the same house style without a
// bootstrap cycle.
for (const [subpath, resolution] of Object.entries(exportResolutions)) {
  const alias = subpath.slice(2);
  if (alias.split("/").length !== 2) continue;
  const target = join(sourceRoot, resolution.source);
  if (existsSync(target)) packageTargets.set(alias, target);
}

const transformBuildSource = async (root) => {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      await transformBuildSource(path);
      continue;
    }
    if (!/\.(?:ts|tsx)$/.test(entry.name)) continue;
    let source = await readFile(path, "utf8");
    source = source.replace(
      /((?:from\s+|import\s*\(|require\s*\()(['"]))@vrooli\/react-component-library\/([^/'"\s]+)\/([^/'"\s]+)(\2)/g,
      (_match, prefix, quote, name, version, suffix) => {
        const target = packageTargets.get(`${name}/${version}`);
        if (!target) return _match;
        let relativeTarget = relativePath(dirname(path), target)
          .replaceAll("\\", "/")
          .replace(/\.(?:tsx?|jsx?)$/, "");
        if (!relativeTarget.startsWith(".")) relativeTarget = `./${relativeTarget}`;
        return `${prefix}${relativeTarget}${suffix}`;
      },
    );
    const legacyStringResolver = ["resolve", "Strings"].join("");
    source = source.replaceAll(legacyStringResolver, "useStrings");
    source = source.replaceAll("{ useStrings, useStrings }", "{ useStrings }");
    // react-dom is supplied by the consuming React runtime and typed through
    // tsconfig.build.json. Older Portal releases carried this suppression
    // before the package build had that type path, so remove it only from the
    // disposable projection once it is no longer needed.
    source = source.replaceAll("// @ts-expect-error react-dom is supplied by the consuming React runtime.\n", "");
    source = source.replaceAll(
      "ref={overlay.surfaceRef}",
      "ref={(node) => { overlay.surfaceRef.current = node; }}",
    );
    source = source
      .replaceAll("../../../../foundations/ClassMerge/versions/1.0.0/ClassMerge", "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge");
    // Keep historical authored edges buildable when a source version has
    // already been retired from the catalog. These substitutions are scoped
    // to the disposable artifact and do not alter the catalog's versioned
    // source or its adoption ledger.
    source = source
      .replaceAll(
        "../../../../primitives/ProgressiveImage/versions/1.0.0/ProgressiveImage",
        "../../../../primitives/ProgressiveImage/versions/1.1.4/ProgressiveImage",
      )
      .replaceAll(
        "../../../BoundedMeter/versions/1.0.1/BoundedMeter",
        "../../../BoundedMeter/versions/1.0.2/BoundedMeter",
      )
      .replaceAll(
        "../../../SearchResults/versions/1.0.0/SearchResults",
        "../../../SearchResults/versions/1.0.4/SearchResults",
      )
      .replaceAll(
        "../../../DrawerShell/versions/1.0.0/DrawerShell",
        "../../../DrawerShell/versions/1.0.0/useFocusTrap",
      );
    if (path.endsWith("/components/MonetizationAccount/versions/1.0.0/MonetizationAccount.tsx")) {
      source = source
        .replace(
          'export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";\n',
          'export type EntitlementStatus = "active" | "trialing" | "past_due" | "canceled" | "inactive";\n\n' +
            'export interface SubscriptionStatusCardProps { plan: string; status: EntitlementStatus; credits: number; multiplier?: number; label?: string; className?: string }\n' +
            'export interface AuthSectionProps { signedIn: boolean; onSignIn: () => void; onSignOut: () => void; className?: string }\n' +
            'export interface UpgradePromptProps { feature: string; requiredPlan: string; href?: string; className?: string }\n' +
            'export interface PendingSyncBadgeProps { pending: number; className?: string }\n' +
            'export interface EntitlementErrorCardProps { errorType: string; children?: ReactNode; className?: string }\n',
        )
        .replace(
          '{ plan: string; status: EntitlementStatus; credits: number; multiplier?: number; label?: string }',
          "SubscriptionStatusCardProps",
        )
        .replace(
          '{ signedIn: boolean; onSignIn: () => void; onSignOut: () => void }',
          "AuthSectionProps",
        )
        .replace(
          '{ feature: string; requiredPlan: string; href?: string }',
          "UpgradePromptProps",
        )
        .replace('{ pending: number }', "PendingSyncBadgeProps")
        .replace('{ errorType: string; children?: ReactNode }', "EntitlementErrorCardProps");
    }
    if (
      path.endsWith("/components/BottomNav/versions/1.3.1/BottomNav.tsx") ||
      path.endsWith("/components/BottomNav/versions/1.3.3/BottomNav.tsx") ||
      path.endsWith("/components/Tabs/versions/1.0.1/Tabs.tsx")
    ) {
      source = source.replaceAll('data-testid="navigation.bottom-navigation"\n', "");
      source = source.replaceAll('data-testid="navigation.tabs"\n', "");
    }
    await writeFile(path, source);
  }
};
await transformBuildSource(sourceRoot);

if (checkOnly) {
  const tsc = join(packageRoot, "..", "..", "scenarios", "react-component-library", "ui", "node_modules", ".bin", "tsc");
  if (!existsSync(tsc)) {
    await rm(sourceRoot, { recursive: true, force: true });
    console.error(`TypeScript compiler not found at ${tsc}`);
    process.exit(1);
  }
  const result = spawnSync(tsc, ["-p", "tsconfig.build.json", "--noEmit"], { cwd: packageRoot, stdio: "inherit" });
  await rm(sourceRoot, { recursive: true, force: true });
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
  await rm(sourceRoot, { recursive: true, force: true });
  process.exit(code ?? 1);
};
const sync = spawnSync(process.execPath, [join(packageRoot, "scripts", "sync-exports.mjs")], { cwd: packageRoot, stdio: "inherit" });
if (sync.status !== 0) process.exit(sync.status ?? 1);
const tsc = join(packageRoot, "..", "..", "scenarios", "react-component-library", "ui", "node_modules", ".bin", "tsc");
if (!existsSync(tsc)) {
  console.error(`TypeScript compiler not found at ${tsc}`);
  process.exit(1);
}
const result = spawnSync(tsc, ["-p", "tsconfig.build.json", "--outDir", stagingRoot], { cwd: packageRoot, stdio: "inherit" });
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

await rm(sourceRoot, { recursive: true, force: true });
console.log("Built @vrooli/react-component-library with declarations.");
