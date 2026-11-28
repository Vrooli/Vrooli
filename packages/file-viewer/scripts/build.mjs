// Build dist/: compile TypeScript, then copy the hand-written runtime scripts
// (plain JS loaded as text by the HTML preview) and their declarations, which
// tsc does not emit.
//
// dist/ is NOT deleted first. Scenario UIs install this package as a pnpm copy
// whose dist files are hard links to these files; deleting and recreating them
// would give every file a new inode and leave each consumer on the old build
// until its install reruns. Compiling over the existing files rewrites them in
// place, so every consumer sees the new build at once. Outputs of deleted
// sources are removed afterwards, by age.
import { spawnSync } from "node:child_process";
import { cpSync, mkdirSync, readdirSync, rmSync, statSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const src = join(root, "src");
const dist = join(root, "dist");

const startedAt = Date.now() - 1000;

const tsc = join(root, "node_modules", ".bin", "tsc");
const compiled = spawnSync(tsc, ["-p", "tsconfig.build.json"], { cwd: root, stdio: "inherit" });
if (compiled.status !== 0) process.exit(compiled.status ?? 1);

// Files tsc does not emit: .js sources and ambient .d.ts declarations.
function* assets(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) {
      if (name === "__tests__" || name === "test-utils") continue;
      yield* assets(path);
    } else if (/\.js$/.test(name) || /\.d\.ts$/.test(name)) {
      yield path;
    }
  }
}

let copied = 0;
for (const file of assets(src)) {
  const target = join(dist, relative(src, file));
  mkdirSync(dirname(target), { recursive: true });
  cpSync(file, target);
  copied++;
}
console.log(`file-viewer: compiled and copied ${String(copied)} runtime/declaration files to dist/`);

// Anything this build did not write is the output of a source that no longer exists.
function* files(dir) {
  for (const name of readdirSync(dir)) {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) yield* files(path);
    else yield path;
  }
}
let removed = 0;
for (const file of files(dist)) {
  if (statSync(file).mtimeMs < startedAt) { rmSync(file); removed++; }
}
if (removed) console.log(`file-viewer: removed ${String(removed)} stale files from dist/`);
