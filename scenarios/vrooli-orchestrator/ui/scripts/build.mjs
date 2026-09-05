// Static build: stage the served assets into dist/. Node's own fs is the whole
// toolchain here — no shell, so the build runs the same on every host.
import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const source = join(uiRoot, "src");
const dist = join(uiRoot, "dist");

if (!existsSync(source)) {
  console.error(`Missing source directory: ${source}`);
  process.exit(1);
}

rmSync(dist, { recursive: true, force: true });
mkdirSync(dist, { recursive: true });
cpSync(source, dist, { recursive: true });
console.log("Staged src/ into dist/");
