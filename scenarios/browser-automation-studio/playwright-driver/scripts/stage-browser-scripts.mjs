// Stage the injected browser scripts alongside the bundled server. esbuild emits
// dist/server.js; these files ship as-is because they are evaluated inside the
// page, not bundled. Node's own fs does the copy — no shell glob, so the step
// runs the same on every host.
import { cpSync, existsSync, mkdirSync, readdirSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const source = join(packageRoot, "src", "recording", "capture", "browser-scripts");
const target = join(packageRoot, "dist", "browser-scripts");

if (!existsSync(source)) {
  console.error(`Missing browser-scripts source: ${source}`);
  process.exit(1);
}

mkdirSync(target, { recursive: true });
const scripts = readdirSync(source).filter((name) => name.endsWith(".js"));
for (const name of scripts) {
  cpSync(join(source, name), join(target, name));
}
console.log(`Staged ${scripts.length} browser script(s) into dist/browser-scripts/`);
