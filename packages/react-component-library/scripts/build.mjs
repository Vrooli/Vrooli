import { cp, mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, dirname, relative as relativePath } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const packageRoot = dirname(fileURLToPath(import.meta.url)).replace(/\/scripts$/, "");
const libraryRoot = join(packageRoot, "..", "..", "scenarios", "react-component-library", "library");
const sourceRoot = join(packageRoot, ".build-source");
const checkOnly = process.argv.includes("--check-only");
// The catalog library remains the only authored source of truth. The package
// build uses a disposable transformed tree because released versions may
// intentionally import an older compatibility path while the package artifact
// must compile against the latest compatible foundation implementation.
await rm(sourceRoot, { recursive: true, force: true });
await cp(libraryRoot, sourceRoot, { recursive: true });
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
    source = source
      .replaceAll("../../../../foundations/ClassMerge/versions/1.0.0/ClassMerge", "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge");
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

await rm(join(packageRoot, "dist"), { recursive: true, force: true });
await mkdir(join(packageRoot, "dist"), { recursive: true });
const sync = spawnSync(process.execPath, [join(packageRoot, "scripts", "sync-exports.mjs")], { cwd: packageRoot, stdio: "inherit" });
if (sync.status !== 0) process.exit(sync.status ?? 1);
const tsc = join(packageRoot, "..", "..", "scenarios", "react-component-library", "ui", "node_modules", ".bin", "tsc");
if (!existsSync(tsc)) {
  console.error(`TypeScript compiler not found at ${tsc}`);
  process.exit(1);
}
const result = spawnSync(tsc, ["-p", "tsconfig.build.json"], { cwd: packageRoot, stdio: "inherit" });
if (result.status !== 0) {
  await rm(sourceRoot, { recursive: true, force: true });
  process.exit(result.status ?? 1);
}

for (const file of styleFiles) {
  const relative = file.slice(sourceRoot.length + 1);
  await mkdir(join(packageRoot, "dist", dirname(relative)), { recursive: true });
  await cp(file, join(packageRoot, "dist", relative));
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
await rewriteRelativeImports(join(packageRoot, "dist"));
await rm(sourceRoot, { recursive: true, force: true });
console.log("Built @vrooli/react-component-library with declarations.");
