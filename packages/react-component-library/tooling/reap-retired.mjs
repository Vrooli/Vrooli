import { readdir, readFile, rm, stat } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(join(fileURLToPath(new URL("..", import.meta.url))));
const repoRoot = resolve(join(packageRoot, "..", ".."));
const retiredRoot = join(repoRoot, "scenarios", "react-component-library", "library", ".retired");
const retentionDays = Number(process.argv.find((arg) => arg.startsWith("--retention-days="))?.split("=")[1] ?? 30);
if (!Number.isInteger(retentionDays) || retentionDays < 1) throw new Error("--retention-days must be a positive integer");
const apply = process.argv.includes("--apply");

async function filesUnder(root) {
  const result = [];
  if (!existsSync(root)) return result;
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) {
      if (["node_modules", "dist", ".retired", ".git"].includes(entry.name)) continue;
      result.push(...await filesUnder(path));
    }
    else result.push(path);
  }
  return result;
}

// Reachability is evaluated against the live library and published package
// projections. The quarantine itself and unrelated scenario corpora are not
// roots, so they must not make every retained tree appear reachable.
const sourceFiles = [
  ...await filesUnder(join(repoRoot, "scenarios", "react-component-library", "library")),
  ...await filesUnder(join(repoRoot, "packages", "react-component-library")),
];
const isReferenced = async (marker, name) => {
  for (const path of sourceFiles.filter((candidate) => /\.(?:ts|tsx|js|jsx|json)$/.test(candidate))) {
    const source = await readFile(path, "utf8");
    if (source.includes(marker) || source.includes(name)) return true;
  }
  return false;
};
const cutoff = Date.now() - retentionDays * 24 * 60 * 60 * 1000;
const candidates = [];
for (const entry of await readdir(retiredRoot, { withFileTypes: true }).catch(() => [])) {
  if (!entry.isDirectory()) continue;
  const path = join(retiredRoot, entry.name);
  const info = await stat(path);
  if (info.mtimeMs > cutoff) continue;
  const marker = relative(repoRoot, path).replaceAll("\\", "/");
  if (await isReferenced(marker, entry.name)) {
    throw new Error(`refusing to reap reachable quarantine tree ${marker}`);
  }
  candidates.push(marker);
  if (apply) await rm(path, { recursive: true, force: true });
}
console.log(JSON.stringify({ reaper: "retired/v1", retentionDays, apply, candidates, deleted: apply ? candidates.length : 0 }));
