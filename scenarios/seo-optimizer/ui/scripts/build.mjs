// Static build: stage the served assets into dist/. Node's own fs is the whole
// toolchain here — no shell, so the build runs the same on every host.
import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const requestedOutDir = process.argv.find((arg) => arg.startsWith("--outDir="))?.slice("--outDir=".length)
  ?? (() => {
    const index = process.argv.indexOf("--outDir");
    return index >= 0 ? process.argv[index + 1] : null;
  })();
const dist = requestedOutDir ? resolve(uiRoot, requestedOutDir) : join(uiRoot, "dist");
const assets = ["index.html", "styles.css", "script.js", "bridge-init.js"];

rmSync(dist, { recursive: true, force: true });
mkdirSync(dist, { recursive: true });

let staged = 0;
for (const asset of assets) {
  const source = join(uiRoot, asset);
  if (!existsSync(source)) continue;
  cpSync(source, join(dist, asset));
  staged++;
}

if (staged === 0) {
  console.error(`No assets found to stage. Looked for: ${assets.join(", ")}`);
  process.exit(1);
}
console.log(`Staged ${staged} asset(s) into dist/`);
