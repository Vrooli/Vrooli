// react-scripts emits into build/; the lifecycle serves dist/. Restage with
// Node's fs so the step carries no shell quoting and runs the same everywhere.
import { cpSync, existsSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const buildDir = join(uiRoot, "build");
const distDir = join(uiRoot, "dist");

if (!existsSync(buildDir)) {
  console.error(`Missing build output at ${buildDir}`);
  process.exit(1);
}
rmSync(distDir, { recursive: true, force: true });
cpSync(buildDir, distDir, { recursive: true });
console.log("Staged build/ into dist/");
