// Stage the production UI into lifecycle's requested output directory.
// The app is intentionally framework-free, so the build copies its static
// assets without requiring a shell or a bundler.
import { cp, mkdir, rm } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const outDirIndex = process.argv.indexOf("--outDir");
const requestedOutDir = outDirIndex >= 0 ? process.argv[outDirIndex + 1] : "";
const dist = requestedOutDir && !requestedOutDir.startsWith("-")
  ? resolve(requestedOutDir)
  : join(uiRoot, "dist");

const assets = ["index.html", "app.js", "styles.css"];

await rm(dist, { recursive: true, force: true });
await mkdir(dist, { recursive: true });
for (const asset of assets) {
  await cp(join(uiRoot, asset), join(dist, asset));
}
console.log(`Staged ${assets.length} asset(s) into ${dist}`);
