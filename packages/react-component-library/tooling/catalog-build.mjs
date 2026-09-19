import { spawnSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const scripts = dirname(fileURLToPath(import.meta.url));
const run = (script, args = []) => {
  const result = spawnSync(process.execPath, [join(scripts, script), ...args], { cwd: join(scripts, ".."), stdio: "inherit" });
  if (result.status !== 0) process.exit(result.status ?? 1);
};
const check = process.argv.includes("--check");
// The public catalog projection has one entry point. Its internal projections
// are deliberately not exposed as independent repair choices.
run("sync-exports.mjs", check ? ["--check"] : []);
run("generate-manifests.mjs", check ? ["--check"] : []);
run("generate-release-hashes.mjs", check ? ["--check"] : []);
run("generate-story-contracts.mjs", check ? ["--check"] : []);
run("generate-locks.mjs", check ? ["--check"] : []);
console.log(JSON.stringify({ generator: "catalog-build/v1", check, status: "ok" }));
