// Placeholder strings check: this scenario has no generated strings module yet,
// so the check only asserts the source tree is present. Kept as a script file
// rather than a `node -e` one-liner because the nested quoting in an inline
// program does not survive cmd.exe.
import { existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const src = join(uiRoot, "src");

if (!existsSync(src)) {
  console.error(`Missing source directory: ${src}`);
  process.exit(1);
}
