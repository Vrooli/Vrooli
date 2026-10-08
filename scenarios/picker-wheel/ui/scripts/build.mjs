// Static build: stage the served assets into dist/. Node's own fs is the whole
// toolchain here — no shell, so the build runs the same on every host.
import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const outDirIndex = process.argv.indexOf("--outDir");
const requestedOutDir = outDirIndex >= 0 ? process.argv[outDirIndex + 1] : "";
const dist = requestedOutDir && !requestedOutDir.startsWith("-")
  ? resolve(requestedOutDir)
  : join(uiRoot, "dist");
const assets = ["index.html", "styles.css", "script.js"];

rmSync(dist, { recursive: true, force: true });
mkdirSync(dist, { recursive: true });

const missing = assets.filter((asset) => !existsSync(join(uiRoot, asset)));
if (missing.length > 0) {
  console.error(`Missing asset(s): ${missing.join(", ")}`);
  process.exit(1);
}
for (const asset of assets) {
  cpSync(join(uiRoot, asset), join(dist, asset));
}
console.log(`Staged ${assets.length} asset(s) into ${dist}`);
