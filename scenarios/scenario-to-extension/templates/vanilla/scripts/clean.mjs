// Remove build outputs. Node's own fs is the whole toolchain — no shell, so the
// same script runs on every host.
import { rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const targets = ["dist"];

for (const target of targets) {
  rmSync(join(packageRoot, target), { recursive: true, force: true });
}
console.log(`Removed: ${targets.join(", ")}`);
